package handler

import (
	"github.com/gofiber/fiber/v3"
	adminrequest "github.com/xvq/go-template/internal/app/admin/request"
	"github.com/xvq/go-template/internal/app/admin/resource"
	"github.com/xvq/go-template/internal/core/response"
	"github.com/xvq/go-template/internal/service/settings"
)

type Settings struct{}

func (h *Settings) List(c fiber.Ctx) error {
	out, err := settings.List(c.Context())
	if err != nil {
		return err
	}
	return response.OK(c, resource.SettingResource.Collection(out))
}
func (h *Settings) Update(c fiber.Ctx, req *adminrequest.UpdateSettings) error {
	if err := settings.Update(c.Context(), req.Values); err != nil {
		return err
	}
	return response.OK(c, nil)
}
