package response

import (
	"context"
	"errors"
	"fmt"
	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/apperror"
	"github.com/xvq/go-template/internal/infrastructure/logger"
	"log/slog"
	"strconv"
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
func Pagination(c fiber.Ctx) (int, int, error) {
	page, size := 1, 20
	for key, target := range map[string]*int{"page": &page, "page_size": &size} {
		if val := c.Query(key); val != "" {
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return 0, 0, fiber.NewError(400, "分页参数无效")
			}
			*target = n
		}
	}
	if size > 100 || page > 1000000 {
		return 0, 0, fiber.NewError(400, "page_size 最大 100，page 最大 1000000")
	}
	return page, size, nil
}
func ID(c fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, fiber.NewError(400, "ID 无效")
	}
	return id, nil
}
func ErrorHandler(log *slog.Logger, secrets ...string) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, code, message := 500, "internal_error", "服务器内部错误"
		var fields any
		var ve v.Errors
		var fe *fiber.Error
		switch {
		case errors.As(err, &ve):
			status, code, message, fields = 422, "validation_failed", "参数校验失败", ve
		case errors.Is(err, apperror.ErrUnauthorized):
			status, code, message = 401, "unauthorized", apperror.ErrUnauthorized.Error()
		case errors.Is(err, apperror.ErrCredentials):
			status, code, message = 401, "unauthorized", apperror.ErrCredentials.Error()
		case errors.Is(err, apperror.ErrNotFound):
			status, code, message = 404, "not_found", apperror.ErrNotFound.Error()
		case errors.Is(err, apperror.ErrConflict):
			status, code, message = 409, "conflict", apperror.ErrConflict.Error()
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
			log.ErrorContext(c.Context(), "request failed", attrs...)
		}
		return c.Status(status).JSON(Envelope{Code: code, Message: message, Errors: fields, RequestID: c.GetRespHeader("X-Request-ID")})
	}
}
