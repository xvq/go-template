package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
)

var loginLimitScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return n
`)

func LoginLimit() fiber.Handler {
	limit, window := core.Config.Auth.Admin.LoginLimit, core.Config.Auth.Admin.LoginWindow
	return func(c fiber.Ctx) error {
		n, err := loginLimitScript.Run(c.Context(), core.Redis, []string{"login-limit:" + c.IP()}, window.Milliseconds()).Int()
		if err != nil {
			return err
		}
		if n > limit {
			return apperror.ErrRateLimited
		}
		return c.Next()
	}
}
