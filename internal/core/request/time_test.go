package request_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
	"github.com/xvq/go-template/internal/core/request"
	"github.com/xvq/go-template/internal/core/response"
)

func TestTimeBindingBeforeValidation(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatal(err)
	}
	type period struct {
		Start *time.Time `json:"start" query:"start" form:"start" label:"开始时间" validate:"required"`
		End   *time.Time `json:"end" query:"end" form:"end" label:"结束时间" validate:"required,gtfield=Start"`
	}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler()})
	calls := 0
	h := func(c fiber.Ctx, req *period) error {
		calls++
		return c.JSON(map[string]*time.Time{"start": req.Start, "end": req.End})
	}
	app.Post("/json", request.JSON(h))
	app.Post("/form", request.Form(h))
	app.Get("/query", request.Query(h))
	app.Post("/body", request.Body(h))
	for _, source := range []string{"json", "form", "query", "body-json", "body-form"} {
		for _, tc := range []struct {
			name, start, end, want string
			status                 int
		}{
			{"local rejected", "2026-10-28 00:00:00", "2026-10-29T00:00:00Z", "", 400},
			{"offset", "2026-10-28T00:00:00+08:00", "2026-10-29T00:00:00+08:00", "2026-10-27T16:00:00Z", 200},
			{"utc", "2026-10-27T16:00:00Z", "2026-10-28T16:00:00Z", "2026-10-27T16:00:00Z", 200},
			{"negative offset", "2026-10-28T00:00:00-04:00", "2026-10-29T00:00:00-04:00", "2026-10-28T04:00:00Z", 200},
			{"fraction", "2026-10-28T00:00:00.123+08:00", "2026-10-29T00:00:00+08:00", "2026-10-27T16:00:00.123Z", 200},
			{"invalid date", "2026-02-30 00:00:00", "2026-10-29 00:00:00", "", 400},
			{"date only", "2026-10-28", "2026-10-29 00:00:00", "", 400},
			{"missing offset", "2026-10-28T00:00:00", "2026-10-29 00:00:00", "", 400},
			{"fraction without offset", "2026-10-28 00:00:00.123", "2026-10-29 00:00:00", "", 400},
			{"space with offset", "2026-10-28 00:00:00+08:00", "2026-10-29 00:00:00", "", 400},
			{"empty", "", "2026-10-29 00:00:00", "", 400},
			{"reversed", "2026-10-29T00:00:00+08:00", "2026-10-28T00:00:00+08:00", "", 422},
			{"equal instants", "2026-10-27T16:00:00Z", "2026-10-28T00:00:00+08:00", "", 422},
		} {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				method, path, contentType := "POST", "/"+source, "application/json"
				values := url.Values{"start": {tc.start}, "end": {tc.end}}
				body, err := json.Marshal(map[string]string{"start": tc.start, "end": tc.end})
				if err != nil {
					t.Fatal(err)
				}
				switch source {
				case "query":
					method, path, body = "GET", "/query?"+values.Encode(), nil
				case "form", "body-form":
					contentType, body = "application/x-www-form-urlencoded", []byte(values.Encode())
				}
				if strings.HasPrefix(source, "body-") {
					path = "/body"
				}
				req := httptest.NewRequest(method, path, strings.NewReader(string(body)))
				req.Header.Set("Content-Type", contentType)
				before := calls
				res, err := app.Test(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()
				if res.StatusCode != tc.status {
					text, _ := io.ReadAll(res.Body)
					t.Fatalf("status=%d, want %d: %s", res.StatusCode, tc.status, text)
				}
				if tc.status != 200 {
					if calls != before {
						t.Fatal("invalid time reached handler")
					}
					return
				}
				var got map[string]time.Time
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil || got["start"].UTC().Format(time.RFC3339Nano) != tc.want {
					t.Fatalf("time=%v, want %s, err=%v", got, tc.want, err)
				}
				original, err := time.Parse(time.RFC3339, tc.start)
				if err != nil {
					t.Fatal(err)
				}
				_, originalOffset := original.Zone()
				_, boundOffset := got["start"].Zone()
				if boundOffset != originalOffset {
					t.Fatal("binding changed the supplied timezone offset")
				}
			})
		}
	}
}

func TestOptionalAndNestedTimeBinding(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatal(err)
	}
	type optional struct {
		At *time.Time `json:"at" query:"at" label:"时间"`
	}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler()})
	h := func(c fiber.Ctx, req *optional) error { return c.JSON(req.At) }
	app.Post("/json", request.JSON(h))
	app.Get("/query", request.Query(h))
	for _, body := range []string{`{}`, `{"at":null}`} {
		req := httptest.NewRequest("POST", "/json", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		text, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || string(text) != "null" {
			t.Fatalf("optional time: %d %s", res.StatusCode, text)
		}
	}
	res, err := app.Test(httptest.NewRequest("GET", "/query", nil))
	if err != nil {
		t.Fatal(err)
	}
	text, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(text) != "null" {
		t.Fatalf("omitted query time: %d %s", res.StatusCode, text)
	}
	for _, body := range []string{`{"at":123}`, `{"at":false}`, `{"at":{}}`, `{"at":[]}`} {
		req := httptest.NewRequest("POST", "/json", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatalf("non-string time accepted: %s status=%d", body, res.StatusCode)
		}
	}
	var nested struct {
		Items []struct {
			At time.Time `json:"at" validate:"required"`
		} `json:"items" validate:"dive"`
	}
	if err := json.Unmarshal([]byte(`{"items":[{"at":"2026-10-28T00:00:00+08:00"}]}`), &nested); err != nil {
		t.Fatal(err)
	}
	if err := validator.Struct(&nested); err != nil {
		t.Fatal(err)
	}
	if got := nested.Items[0].At.UTC().Format(time.RFC3339); got != "2026-10-27T16:00:00Z" {
		t.Fatal("nested time represents the wrong instant", got)
	}
}
