package validator

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/user/go-template/internal/common"
)

var V *validator.Validate

func Init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		V = v
		V.RegisterTagNameFunc(func(fld reflect.StructField) string {
			return fld.Tag.Get("json")
		})
	}
}

func Bind[T any](h func(*gin.Context, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Error(c, 40001, err.Error())
			return
		}
		h(c, &req)
	}
}
