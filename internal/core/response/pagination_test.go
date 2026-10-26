package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/request"
	"github.com/xvq/go-template/internal/core/testutil"
)

func TestPaginationErrorsReachHTTPAsBadRequests(t *testing.T) {
	var logs bytes.Buffer
	testutil.UseCore(t, config.Defaults(), nil, nil, slog.New(slog.NewJSONHandler(&logs, nil)))
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler()})
	app.Get("/list", func(c fiber.Ctx) error {
		_, _, err := request.Pagination(c)
		return err
	})
	app.Get("/wrapped", func(c fiber.Ctx) error {
		return fmt.Errorf("private implementation detail: %w", apperror.ErrInvalidPagination)
	})
	for _, path := range []string{
		"/list?page=abc", "/list?page=999999999999999999999999999",
		"/list?page=0", "/list?page=-1", "/list?page=1000001",
		"/list?page_size=abc", "/list?page_size=0", "/list?page_size=-1", "/list?page_size=101",
		"/wrapped",
	} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var body Envelope
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 400 || body.Code != "invalid_pagination" || body.Message != apperror.ErrInvalidPagination.Error() {
			t.Fatalf("%s: status=%d body=%+v", path, resp.StatusCode, body)
		}
	}
	if logs.Len() != 0 {
		t.Fatal("pagination errors were logged as server failures")
	}
}

func TestPaginationDefaultsAndValidBoundaries(t *testing.T) {
	app := fiber.New()
	app.Get("/params", func(c fiber.Ctx) error {
		page, size, err := request.Pagination(c)
		if err != nil {
			return err
		}
		return c.JSON([]int{page, size})
	})
	for _, tc := range []struct {
		path string
		want [2]int
	}{
		{"/params", [2]int{1, 20}},
		{"/params?page=2&page_size=10", [2]int{2, 10}},
		{"/params?page=1&page_size=1", [2]int{1, 1}},
		{"/params?page=1000000&page_size=100", [2]int{1000000, 100}},
	} {
		resp, err := app.Test(httptest.NewRequest("GET", tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var got [2]int
		err = json.NewDecoder(resp.Body).Decode(&got)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || got != tc.want {
			t.Fatalf("%s: got %v, err=%v", tc.path, got, err)
		}
	}
}
