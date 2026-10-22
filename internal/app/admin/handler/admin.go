package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin/middleware"
	"github.com/xvq/go-template/internal/app/admin/resource"
	"github.com/xvq/go-template/internal/app/admin/service"
	"github.com/xvq/go-template/internal/app/admin/validator"
	base "github.com/xvq/go-template/internal/resource"
	"github.com/xvq/go-template/internal/response"
)

type Admin struct{ service *service.AdminService }

func NewAdmin(s *service.AdminService) *Admin { return &Admin{service: s} }
func (h *Admin) Login(c fiber.Ctx) error {
	var req validator.Login
	if err := validator.Body(c, &req); err != nil {
		return err
	}
	out, err := h.service.Login(c.Context(), req.Username, req.Password, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return err
	}
	return response.OK(c, fiber.Map{
		"token":      out.Token,
		"token_type": "Bearer",
		"expires_at": out.ExpiresAt,
		"admin":      resource.AdminItem(out.Admin),
	})
}
func (h *Admin) Logout(c fiber.Ctx) error {
	if err := h.service.Logout(c.Context(), middleware.Token(c)); err != nil {
		return err
	}
	return response.OK(c, nil)
}
func (h *Admin) Me(c fiber.Ctx) error {
	return response.OK(c, resource.AdminItem(*middleware.Current(c)))
}
func (h *Admin) ChangePassword(c fiber.Ctx) error {
	var req validator.ChangePassword
	if err := validator.Body(c, &req); err != nil {
		return err
	}
	if err := h.service.ChangePassword(c.Context(), middleware.Current(c).ID, req.CurrentPassword, req.Password); err != nil {
		return err
	}
	return response.OK(c, nil)
}
func (h *Admin) List(c fiber.Ctx) error {
	page, size, err := response.Pagination(c)
	if err != nil {
		return err
	}
	result, err := h.service.List(c.Context(), page, size)
	if err != nil {
		return err
	}
	return response.OK(c, base.Paginate(result, resource.AdminItem))
}
func (h *Admin) Get(c fiber.Ctx) error {
	id, err := response.ID(c)
	if err != nil {
		return err
	}
	out, err := h.service.Get(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, resource.AdminItem(out))
}
func (h *Admin) Create(c fiber.Ctx) error {
	var req validator.CreateAdmin
	if err := validator.Body(c, &req); err != nil {
		return err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	out, err := h.service.Create(c.Context(), req.Username, req.DisplayName, req.Password, enabled)
	if err != nil {
		return err
	}
	return response.Created(c, resource.AdminItem(out))
}
func (h *Admin) Update(c fiber.Ctx) error {
	id, err := response.ID(c)
	if err != nil {
		return err
	}
	var req validator.UpdateAdmin
	if err = validator.Body(c, &req); err != nil {
		return err
	}
	out, err := h.service.Update(c.Context(), id, service.AdminUpdate{DisplayName: req.DisplayName, Enabled: req.Enabled})
	if err != nil {
		return err
	}
	return response.OK(c, resource.AdminItem(out))
}
func (h *Admin) Delete(c fiber.Ctx) error {
	id, err := response.ID(c)
	if err != nil {
		return err
	}
	if err = h.service.Delete(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, nil)
}
func (h *Admin) ResetPassword(c fiber.Ctx) error {
	id, err := response.ID(c)
	if err != nil {
		return err
	}
	var req validator.Password
	if err = validator.Body(c, &req); err != nil {
		return err
	}
	if err = h.service.ResetPassword(c.Context(), id, req.Password); err != nil {
		return err
	}
	return response.OK(c, nil)
}
func (h *Admin) LoginLogs(c fiber.Ctx) error {
	page, size, err := response.Pagination(c)
	if err != nil {
		return err
	}
	result, err := h.service.LoginLogs(c.Context(), page, size)
	if err != nil {
		return err
	}
	return response.OK(c, base.Paginate(result, resource.LoginLogItem))
}
