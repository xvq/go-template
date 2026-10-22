package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/app/admin/service"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/model"
	"strings"
)

type identityKey struct{}

func Token(c fiber.Ctx) string {
	parts := strings.Fields(c.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
func Auth(s *service.AdminService) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := Token(c)
		if token == "" {
			return apperror.ErrUnauthorized
		}
		a, err := s.Authenticate(c.Context(), token)
		if err != nil {
			return err
		}
		c.Locals(identityKey{}, a)
		return c.Next()
	}
}
func Current(c fiber.Ctx) *model.Admin { a, _ := c.Locals(identityKey{}).(*model.Admin); return a }
