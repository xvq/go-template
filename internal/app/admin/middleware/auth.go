package middleware

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/core/request"
)

type identityKey struct{}

func Auth() fiber.Handler {
	return func(c fiber.Ctx) error {
		token := request.Token(c)
		if token == "" {
			return apperror.ErrUnauthorized
		}
		identity, err := core.Session.Validate(c.Context(), token)
		if errors.Is(err, session.ErrInvalid) {
			return apperror.ErrUnauthorized
		}
		if err != nil {
			return err
		}
		if identity.Realm != "admin" {
			return apperror.ErrUnauthorized
		}
		c.Locals(identityKey{}, identity.SubjectID)
		return c.Next()
	}
}

func CurrentID(c fiber.Ctx) uint64 {
	id, _ := c.Locals(identityKey{}).(uint64)
	return id
}
