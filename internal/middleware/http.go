package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
)

// RequestLifecycle supplies a real Go context. Fiber's pooled Ctx is never
// passed to a service or retained by a background goroutine.
func RequestLifecycle(parent context.Context, timeout time.Duration, log *slog.Logger, onError fiber.ErrorHandler) fiber.Handler {
	return func(c fiber.Ctx) error {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		id := hex.EncodeToString(b)
		c.Set("X-Request-ID", id)
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()
		c.SetContext(ctx)
		start := time.Now()
		err := c.Next()
		if err != nil {
			err = onError(c, err)
		}
		log.Info("request", "request_id", id, "method", c.Method(), "path", c.Path(), "status", c.Response().StatusCode(), "elapsed", time.Since(start), "ip", c.IP())
		return err
	}
}

var loginLimitScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return n
`)

func LoginLimit(r redis.UniversalClient, prefix string, limit int, window time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		n, err := loginLimitScript.Run(c.Context(), r, []string{prefix + ":login-limit:" + c.IP()}, window.Milliseconds()).Int()
		if err != nil {
			return err
		}
		if n > limit {
			return apperror.ErrRateLimited
		}
		return c.Next()
	}
}
