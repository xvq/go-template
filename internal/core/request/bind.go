// Package request provides request binding, validation and parameter helpers.
package request

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
)

func bind[T any](handler func(fiber.Ctx, *T) error, decode func(fiber.Ctx, *T) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		var value T
		if err := decode(c, &value); err != nil {
			return fiber.NewError(400, "请求格式不正确")
		}
		if err := validator.Struct(&value); err != nil {
			return err
		}
		return handler(c, &value)
	}
}

// Body selects the decoder using Content-Type (JSON or form, for example).
func Body[T any](h func(fiber.Ctx, *T) error) fiber.Handler {
	return bind(h, func(c fiber.Ctx, out *T) error { return c.Bind().Body(out) })
}

func JSON[T any](h func(fiber.Ctx, *T) error) fiber.Handler {
	return bind(h, func(c fiber.Ctx, out *T) error { return c.Bind().JSON(out) })
}

func Form[T any](h func(fiber.Ctx, *T) error) fiber.Handler {
	return bind(h, func(c fiber.Ctx, out *T) error { return c.Bind().Form(out) })
}

func Query[T any](h func(fiber.Ctx, *T) error) fiber.Handler {
	return bind(h, func(c fiber.Ctx, out *T) error { return c.Bind().Query(out) })
}
