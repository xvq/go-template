package response

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamLifetime(t *testing.T) {
	app := fiber.New()
	var upstream context.Context
	app.Get("/stream", func(c fiber.Ctx) error {
		return Stream(c, context.Background(), time.Second, "text/event-stream", func(ctx context.Context) (io.ReadCloser, error) {
			upstream = ctx
			return io.NopCloser(strings.NewReader("data: hello\n\n")), nil
		})
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/stream", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != "data: hello\n\n" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	select {
	case <-upstream.Done():
	case <-time.After(time.Second):
		t.Fatal("stream context not released")
	}
}
