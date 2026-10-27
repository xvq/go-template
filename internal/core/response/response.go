package response

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/infrastructure/logger"
	"github.com/xvq/go-template/internal/core/infrastructure/validator"
)

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	Errors    any    `json:"errors,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func OK(c fiber.Ctx, data any) error {
	return c.JSON(Envelope{Code: "ok", Message: "ok", Data: data, RequestID: c.GetRespHeader("X-Request-ID")})
}
func Created(c fiber.Ctx, data any) error { c.Status(201); return OK(c, data) }
func ErrorHandler() fiber.ErrorHandler {
	secrets := []string{core.Config.Database.Password, core.Config.Redis.Password, core.Config.Queue.Redis.Password}
	return func(c fiber.Ctx, err error) error {
		status, code, message := 500, "internal_error", "服务器内部错误"
		var fields any
		var ve validator.Errors
		var fe *fiber.Error
		switch {
		case errors.As(err, &ve):
			status, code, message, fields = 422, "validation_failed", "参数校验失败", ve
		case errors.Is(err, apperror.ErrInvalidPagination):
			status, code, message = 400, "invalid_pagination", apperror.ErrInvalidPagination.Error()
		case errors.Is(err, apperror.ErrUnauthorized):
			status, code, message = 401, "unauthorized", apperror.ErrUnauthorized.Error()
		case errors.Is(err, apperror.ErrCredentials):
			status, code, message = 401, "unauthorized", apperror.ErrCredentials.Error()
		case errors.Is(err, apperror.ErrNotFound):
			status, code, message = 404, "not_found", apperror.ErrNotFound.Error()
		case errors.Is(err, apperror.ErrUsernameExists):
			status, code, message = 409, "username_exists", apperror.ErrUsernameExists.Error()
		case errors.Is(err, apperror.ErrLastAdmin):
			status, code, message = 409, "last_admin", apperror.ErrLastAdmin.Error()
		case errors.Is(err, apperror.ErrInvalidSetting):
			status, code, message = 422, "invalid_setting", apperror.ErrInvalidSetting.Error()
		case errors.Is(err, apperror.ErrRateLimited):
			status, code, message = 429, "rate_limited", apperror.ErrRateLimited.Error()
		case errors.Is(err, context.DeadlineExceeded):
			status, code, message = 504, "timeout", "处理超时"
		case errors.Is(err, context.Canceled):
			status, code, message = 503, "canceled", "请求已取消"
		case errors.As(err, &fe):
			status, code, message = fe.Code, "http_error", fe.Message
			if status >= 500 {
				code, message = "internal_error", "服务器内部错误"
			}
		}
		if status >= 500 {
			attrs := []any{"request_id", c.GetRespHeader("X-Request-ID"), "method", c.Method(), "path", c.Path(), "error_type", fmt.Sprintf("%T", err), "error", logger.SafeMessage(err.Error(), secrets...)}
			var panicErr *panicError
			if errors.As(err, &panicErr) {
				attrs = append(attrs, "stack", panicErr.stack)
			}
			core.Log.ErrorContext(c.Context(), "request failed", attrs...)
		}
		return c.Status(status).JSON(Envelope{Code: code, Message: message, Errors: fields, RequestID: c.GetRespHeader("X-Request-ID")})
	}
}
