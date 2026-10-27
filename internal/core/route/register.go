// Package route explicitly registers the application's HTTP route groups.
package route

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin"
	"github.com/xvq/go-template/internal/app/web"
)

// Register mounts application routes after the server's global middleware.
func Register(app *fiber.App) error {
	if err := admin.Register(app); err != nil {
		return err
	}
	web.Register(app)
	return nil
}
