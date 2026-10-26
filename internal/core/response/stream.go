package response

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"io"
	"sync"
	"time"
)

type streamBody struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
	err    error
}

func (s *streamBody) Close() error {
	s.once.Do(func() { s.cancel(); s.err = s.ReadCloser.Close() })
	return s.err
}

// Stream creates a separate bounded context owned by the response body.
// Pass the application context, not the request-scoped context canceled when
// the handler returns. open must pass ctx into the upstream HTTP request.
// The Fiber writer closes the body on completion/disconnect; Close cancels
// upstream work. Headers/status must be decided before streaming starts.
func Stream(c fiber.Ctx, application context.Context, timeout time.Duration, contentType string, open func(context.Context) (io.ReadCloser, error)) error {
	ctx, cancel := context.WithTimeout(application, timeout)
	body, err := open(ctx)
	if err != nil {
		cancel()
		return err
	}
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "no-cache")
	c.Set("X-Accel-Buffering", "no")
	c.Response().ImmediateHeaderFlush = true
	wrapped := &streamBody{ReadCloser: body, cancel: cancel}
	if err = c.SendStream(wrapped); err != nil {
		wrapped.Close()
	}
	return err
}
