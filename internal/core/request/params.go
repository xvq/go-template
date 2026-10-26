package request

import (
	"github.com/gofiber/fiber/v3"
	"github.com/xvq/go-template/internal/apperror"
	"strconv"
	"strings"
)

func Pagination(c fiber.Ctx) (int, int, error) {
	page, size := 1, 20
	for key, target := range map[string]*int{"page": &page, "page_size": &size} {
		if val := c.Query(key); val != "" {
			n, err := strconv.Atoi(val)
			if err != nil {
				return 0, 0, apperror.ErrInvalidPagination
			}
			*target = n
		}
	}
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return 0, 0, apperror.ErrInvalidPagination
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

// Token extracts a Bearer token without authenticating it.
// Missing or malformed Authorization headers return an empty string.
func Token(c fiber.Ctx) string {
	parts := strings.Fields(c.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
