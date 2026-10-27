package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin/handler"
	"github.com/xvq/go-template/internal/app/admin/middleware"
	"github.com/xvq/go-template/internal/app/admin/service"
	"github.com/xvq/go-template/internal/core/request"
)

func Register(app *fiber.App) error {
	admins, err := service.NewAdmin()
	if err != nil {
		return err
	}
	h := handler.NewAdmin(admins)
	s := &handler.Settings{}
	api := app.Group("/api/admin")
	api.Post("/login", middleware.LoginLimit(), request.JSON(h.Login))
	api.Use(middleware.Auth())
	api.Post("/logout", h.Logout)
	api.Get("/me", h.Me)
	api.Put("/me/password", request.JSON(h.ChangePassword))
	api.Get("/admins", h.List)
	api.Post("/admins", request.JSON(h.Create))
	api.Get("/admins/:id", h.Get)
	api.Put("/admins/:id", request.JSON(h.Update))
	api.Delete("/admins/:id", h.Delete)
	api.Put("/admins/:id/password", request.JSON(h.ResetPassword))
	api.Get("/login-logs", h.LoginLogs)
	api.Get("/settings", s.List)
	api.Patch("/settings", request.JSON(s.Update))
	return nil
}
