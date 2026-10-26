package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/response"
	"github.com/xvq/go-template/internal/core/testutil"
)

func TestAuthUsesRedisAndOnlyStoresID(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr(), MaxRetries: -1})
	defer client.Close()
	// A valid authenticated request must work with no database installed.
	testutil.UseCore(t, config.Defaults(), nil, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	token, _, err := core.Session.Issue(ctx, "admin", 7, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	web, _, err := core.Session.Issue(ctx, "web", 7, time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler()})
	calls := 0
	app.Get("/protected", Auth(), func(c fiber.Ctx) error {
		calls++
		if CurrentID(c) != 7 {
			t.Fatalf("unexpected identity: %d", CurrentID(c))
		}
		return c.SendStatus(204)
	})
	check := func(value string, status int) {
		t.Helper()
		request := httptest.NewRequest("GET", "/protected", nil)
		if value != "" {
			request.Header.Set("Authorization", "Bearer "+value)
		}
		result, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		result.Body.Close()
		if result.StatusCode != status {
			t.Fatalf("status=%d want=%d", result.StatusCode, status)
		}
	}
	check(token, 204)
	check("", 401)
	check("invalid", 401)
	check(web, 401)
	if err := core.Session.RevokeAll(ctx, "admin", 7); err != nil {
		t.Fatal(err)
	}
	check(token, 401)
	if calls != 1 {
		t.Fatalf("unauthorized requests reached handler: %d", calls)
	}
	redisServer.SetError("ERR unavailable")
	check(web, 500)
}
