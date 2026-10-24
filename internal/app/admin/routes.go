package admin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin/handler"
	"github.com/xvq/go-template/internal/app/admin/middleware"
	"github.com/xvq/go-template/internal/app/admin/service"
	commonmiddleware "github.com/xvq/go-template/internal/middleware"
)

func Register(app *fiber.App) error {
	admins, err := service.NewAdmin()
	if err != nil {
		return err
	}
	h := handler.NewAdmin(admins)
	s := &handler.Settings{}
	api := app.Group("/api/admin")
	api.Post("/login", commonmiddleware.LoginLimit(), h.Login)
	api.Use(middleware.Auth(admins))
	api.Post("/logout", h.Logout)
	api.Get("/me", h.Me)
	api.Put("/me/password", h.ChangePassword)
	api.Get("/admins", h.List)
	api.Post("/admins", h.Create)
	api.Get("/admins/:id", h.Get)
	api.Patch("/admins/:id", h.Update)
	api.Delete("/admins/:id", h.Delete)
	api.Put("/admins/:id/password", h.ResetPassword)
	api.Get("/login-logs", h.LoginLogs)
	api.Get("/settings", s.List)
	api.Patch("/settings", s.Update)
	return nil
}
