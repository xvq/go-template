package bootstrap

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"io"
	"log/slog"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/testutil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestHTTPAuthValidationAndRateLimit(t *testing.T) {
	conn, m, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	c := config.Defaults()
	c.Auth.Admin.LoginLimit = 2
	testutil.UseCore(t, c, db, r, slog.New(slog.NewTextHandler(io.Discard, nil)))
	app, err := NewServer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/admin/admins", "/api/admin/settings", "/api/admin/login-logs", "/api/admin/me"} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Fatalf("%s status=%d", path, resp.StatusCode)
		}
	}
	// Only the valid-length password reaches authentication and audit storage.
	m.ExpectQuery("SELECT .*admins").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	m.ExpectBegin()
	m.ExpectExec("INSERT INTO .*admin_login_logs").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectCommit()
	for i := 0; i < 3; i++ {
		requestBody := `{"username":"admin","password":"wrong"}`
		if i == 1 {
			requestBody = `{"username":"admin","password":"wrong-password"}`
		}
		req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		want := 422
		if i == 1 {
			want = 401
		}
		if i == 2 {
			want = 429
		}
		if resp.StatusCode != want {
			t.Fatalf("attempt %d: status=%d body=%v", i, resp.StatusCode, body)
		}
		if resp.Header.Get("X-Request-ID") == "" {
			t.Fatal("missing request id")
		}
	}
	if r.Exists(context.Background(), "login-limit:1.2.3.4").Val() != 0 {
		t.Fatal("trusted spoofed forwarded IP")
	}
	if err = m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownImmediatelyAfterStart(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	started := false
	go func() { done <- serve(ctx, app, ln, time.Second, func() {}, func() { started = true }) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		if !started {
			t.Fatal("missing startup notification")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown raced with listener startup")
	}
}
