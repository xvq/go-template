package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xvq/go-template/internal/app/service"
	"github.com/xvq/go-template/internal/common"
)

func Auth(model string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			common.Error(c, 40101, "未登录")
			c.Abort()
			return
		}

		td := service.Validate(token)
		if td == nil || td.Model != model {
			common.Error(c, 40101, "未登录")
			c.Abort()
			return
		}

		c.Set("uid", td.UID)
		c.Set("model", td.Model)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}
