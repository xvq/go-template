package request_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	adminrequest "github.com/xvq/go-template/internal/app/admin/request"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
	"github.com/xvq/go-template/internal/core/request"
	"github.com/xvq/go-template/internal/core/response"
)

func TestBindingAndValidationBeforeHandler(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler()})
	calls := 0
	type login struct {
		Username string `json:"username" form:"username" label:"账号" validate:"required,alphanum"`
		Password string `json:"password" form:"password" label:"密码" validate:"required,min=6,max=30"`
	}
	h := func(c fiber.Ctx, r *login) error {
		calls++
		return c.JSON(r.Username)
	}
	app.Post("/body", request.Body(h))
	app.Post("/json", request.JSON(h))
	app.Post("/form", request.Form(h))
	app.Patch("/update", request.JSON(func(c fiber.Ctx, r *adminrequest.UpdateAdmin) error {
		calls++
		return c.JSON(*r.Enabled)
	}))
	type search struct {
		Name string `query:"name" json:"name" label:"名称" validate:"required"`
	}
	app.Get("/query", request.Query(func(c fiber.Ctx, r *search) error { calls++; return c.JSON(r.Name) }))
	for _, tc := range []struct {
		path, contentType, body string
		status                  int
	}{
		{"/body", "application/json", `{"username":"admin","password":"secret"}`, 200},
		{"/body", "application/x-www-form-urlencoded", "username=admin&password=secret", 200},
		{"/json", "application/json", `{"username":"admin","password":"secret"}`, 200},
		{"/form", "application/x-www-form-urlencoded", "username=admin&password=secret", 200},
		{"/json", "application/x-www-form-urlencoded", "username=admin&password=secret", 400},
		{"/query?name=admin", "", "", 200},
		{"/body", "application/json", `{`, 400},
		{"/body", "application/json", `{"username":"!","password":"secret"}`, 422},
		{"/body", "application/json", `{"password":"secret"}`, 422},
		{"/query", "", "", 422},
		{"/update", "application/json", `{"display_name":"管理员","enabled":false}`, 200},
		{"/update", "application/json", `{"display_name":"管理员"}`, 422},
		{"/update", "application/json", `{"display_name":"管理员","enabled":null}`, 422},
		{"/update", "application/json", `{"display_name":"","enabled":true}`, 422},
	} {
		before := calls
		method := "POST"
		if tc.path == "/update" {
			method = "PATCH"
		}
		if strings.HasPrefix(tc.path, "/query") {
			method = "GET"
		}
		req := httptest.NewRequest(method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", tc.contentType)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if tc.status != 200 {
			err = json.NewDecoder(res.Body).Decode(&body)
		}
		res.Body.Close()
		if err != nil || res.StatusCode != tc.status {
			t.Fatalf("%s: status=%d err=%v", tc.path, res.StatusCode, err)
		}
		if tc.status == 200 && calls != before+1 || tc.status != 200 && calls != before {
			t.Fatal("invalid handler invocation")
		}
		if tc.status == 422 && tc.path == "/body" {
			fields := body["errors"].(map[string]any)
			message, ok := fields["username"].(string)
			if !ok || !strings.Contains(message, "账号") || strings.Contains(message, "secret") {
				t.Fatalf("invalid translated fields: %v", fields)
			}
		}
	}
}

func TestExistingRequestRules(t *testing.T) {
	if err := validator.Init(); err != nil {
		t.Fatal(err)
	}
	disabled := false
	for _, tc := range []struct {
		name  string
		value any
		valid bool
	}{
		{"create", &adminrequest.CreateAdmin{Username: "admin", Password: strings.Repeat("a", 12)}, true},
		{"short password", &adminrequest.Password{Password: strings.Repeat("a", 5)}, false},
		{"30 unicode characters", &adminrequest.Password{Password: strings.Repeat("中", 30)}, true},
		{"31 unicode characters", &adminrequest.Password{Password: strings.Repeat("中", 31)}, false},
		{"6 characters", &adminrequest.Password{Password: "abc123"}, true},
		{"symbols and spaces", &adminrequest.Password{Password: " 中 🔐! "}, true},
		{"30 emoji", &adminrequest.Password{Password: strings.Repeat("🔐", 30)}, true},
		{"invalid username symbols", &adminrequest.Login{Username: "admin_123", Password: "secret"}, false},
		{"uppercase username", &adminrequest.Login{Username: "Admin123", Password: "secret"}, true},
		{"empty update", &adminrequest.UpdateAdmin{}, false},
		{"empty name", &adminrequest.UpdateAdmin{DisplayName: "", Enabled: &disabled}, false},
		{"missing enabled", &adminrequest.UpdateAdmin{DisplayName: "管理员"}, false},
		{"disable", &adminrequest.UpdateAdmin{DisplayName: "管理员", Enabled: &disabled}, true},
		{"empty settings", &adminrequest.UpdateSettings{Values: map[string]json.RawMessage{}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Struct(tc.value)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if err != nil {
				var fields validator.Errors
				if !errors.As(err, &fields) {
					t.Fatalf("unexpected error: %T", err)
				}
			}
		})
	}
}

func TestID(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler()})
	app.Get("/items/:id", func(c fiber.Ctx) error {
		id, err := request.ID(c)
		if err != nil {
			return err
		}
		return c.JSON(id)
	})
	for _, tc := range []struct {
		id     string
		status int
	}{{"1", 200}, {"0", 400}, {"-1", 400}, {"abc", 400}, {"18446744073709551616", 400}} {
		res, err := app.Test(httptest.NewRequest("GET", "/items/"+tc.id, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.status {
			t.Fatalf("%s: status=%d", tc.id, res.StatusCode)
		}
	}
}
