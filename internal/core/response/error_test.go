package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/testutil"
)

func TestErrorDiagnosticsAndPanicRecovery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		panic   bool
		status  int
		wantLog bool
	}{
		{"internal", errors.New("dial tcp: connection refused password=hidden known-secret"), false, 500, true},
		{"http500", fiber.NewError(500, "private backend detail"), false, 500, true},
		{"panic", errors.New("unexpected state password=hidden"), true, 500, true},
		{"panic-business", apperror.ErrCredentials, true, 500, true},
		{"business", apperror.ErrCredentials, false, 401, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			log := slog.New(slog.NewJSONHandler(&logs, nil))
			c := config.Defaults()
			c.Database.Password = "known-secret"
			testutil.UseCore(t, c, nil, nil, log)
			app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler()})
			app.Use(recover.New(recover.Config{PanicHandler: PanicHandler}))
			app.Get("/fail", func(c fiber.Ctx) error {
				c.Set("X-Request-ID", "test-request")
				if tc.panic {
					panic(tc.err)
				}
				return tc.err
			})
			app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("ok") })
			resp, err := app.Test(httptest.NewRequest("GET", "/fail", nil))
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.status {
				t.Fatalf("status=%d body=%s", resp.StatusCode, body)
			}
			if tc.wantLog {
				if !strings.Contains(string(body), "服务器内部错误") || strings.Contains(string(body), "stack") {
					t.Fatalf("unsafe response: %s", body)
				}
				var entry map[string]any
				if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
					t.Fatal(err)
				}
				if entry["error"] == "" || entry["request_id"] != "test-request" {
					t.Fatalf("missing diagnostics: %v", entry)
				}
				if strings.Contains(logs.String(), "hidden") || strings.Contains(logs.String(), "known-secret") {
					t.Fatal("credential leaked")
				}
				_, hasStack := entry["stack"]
				if hasStack != tc.panic {
					t.Fatalf("unexpected stack presence: %v", entry)
				}
				if tc.panic && !strings.Contains(entry["stack"].(string), "error_test.go:") {
					t.Fatal("panic source missing")
				}
				if tc.name == "internal" && !strings.Contains(entry["error"].(string), "connection refused") {
					t.Fatal("diagnostic reason lost")
				}
			} else if logs.Len() != 0 {
				t.Fatal("business error logged as server failure")
			}
			resp, err = app.Test(httptest.NewRequest("GET", "/ok", nil))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatal("server did not recover")
			}
		})
	}
}
