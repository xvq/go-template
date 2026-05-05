package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/xvq/go-template/internal/common"

	zhLocale "github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

var V *validator.Validate
var trans ut.Translator

func Init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		V = v
		V.RegisterTagNameFunc(func(fld reflect.StructField) string {
			tag := fld.Tag.Get("json")
			if tag == "-" {
				return ""
			}
			name := strings.Split(tag, ",")[0]
			if name == "" {
				return fld.Name
			}
			return name
		})
		z := zhLocale.New()
		uni := ut.New(z, z)
		trans, _ = uni.GetTranslator("zh")
		zhTranslations.RegisterDefaultTranslations(V, trans)
	}
}

func translate(err error) string {
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return "参数错误"
	}
	if trans == nil {
		return errs.Error()
	}
	for _, e := range errs {
		return e.Translate(trans)
	}
	return errs.Error()
}

func BindQuery[T any](h func(*gin.Context, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T
		if err := c.ShouldBindQuery(&req); err != nil {
			common.Error(c, 40001, translate(err))
			return
		}
		h(c, &req)
	}
}

func BindForm[T any](h func(*gin.Context, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T
		if err := c.ShouldBindWith(&req, binding.Form); err != nil {
			common.Error(c, 40001, translate(err))
			return
		}
		h(c, &req)
	}
}

func BindJSON[T any](h func(*gin.Context, *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req T
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Error(c, 40001, translate(err))
			return
		}
		h(c, &req)
	}
}
