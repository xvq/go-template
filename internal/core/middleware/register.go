package middleware

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/response"
)

// Register installs global middleware in execution order, before any routes.
func Register(app *fiber.App, ctx context.Context, onError fiber.ErrorHandler) {
	app.Use(RequestLifecycle(ctx, onError))
	app.Use(recover.New(recover.Config{PanicHandler: response.PanicHandler}))
	if origins := core.Config.Server.CORSOrigins; len(origins) > 0 {
		app.Use(cors.New(cors.Config{
			AllowOrigins:  origins,
			AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:  []string{"Authorization", "Content-Type"},
			ExposeHeaders: []string{"X-Request-ID"},
		}))
	}
}
