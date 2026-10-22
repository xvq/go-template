package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin/resource"
	"github.com/xvq/go-template/internal/app/admin/validator"
	base "github.com/xvq/go-template/internal/resource"
	"github.com/xvq/go-template/internal/response"
	service "github.com/xvq/go-template/internal/service/settings"
)

type Settings struct{ service *service.SettingsService }

func NewSettings(s *service.SettingsService) *Settings { return &Settings{service: s} }
func (h *Settings) List(c fiber.Ctx) error {
	out, err := h.service.List(c.Context())
	if err != nil {
		return err
	}
	return response.OK(c, base.Collection(out, resource.SettingItem))
}
func (h *Settings) Update(c fiber.Ctx) error {
	var req validator.UpdateSettings
	if err := validator.Body(c, &req); err != nil {
		return err
	}
	if err := h.service.Update(c.Context(), req.Values); err != nil {
		return err
	}
	return response.OK(c, nil)
}
