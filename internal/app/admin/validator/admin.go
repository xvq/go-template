package validator

import (
	"encoding/json"
	"fmt"
	v "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofiber/fiber/v3"
	"regexp"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)
var passwordRule = v.By(func(value interface{}) error {
	p, ok := value.(string)
	if !ok || len(p) < 12 || len(p) > 72 {
		return fmt.Errorf("密码长度必须为 12～72 字节")
	}
	return nil
})

type Validatable interface{ Validate() error }

// Binding and validation are explicit calls from each handler.
func Body(c fiber.Ctx, out Validatable) error {
	if err := c.Bind().Body(out); err != nil {
		return fiber.NewError(400, "请求格式不正确")
	}
	return out.Validate()
}

type Login struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

func (r *Login) Validate() error {
	return v.ValidateStruct(r,
		v.Field(&r.Username, v.Required.Error("请输入账号"), v.Match(usernamePattern).Error("账号格式不正确")),
		v.Field(&r.Password, v.Required.Error("请输入密码"), v.By(func(value interface{}) error {
			if len(value.(string)) > 72 {
				return fmt.Errorf("密码过长")
			}
			return nil
		})),
	)
}

type CreateAdmin struct {
	Username    string `json:"username" form:"username"`
	DisplayName string `json:"display_name" form:"display_name"`
	Password    string `json:"password" form:"password"`
	Enabled     *bool  `json:"enabled" form:"enabled"`
}

func (r *CreateAdmin) Validate() error {
	return v.ValidateStruct(r, v.Field(&r.Username, v.Required.Error("请输入账号"), v.Match(usernamePattern).Error("账号需为 3～64 位字母、数字、下划线、点或连字符")), v.Field(&r.DisplayName, v.RuneLength(0, 100).Error("名称最多 100 字符")), v.Field(&r.Password, passwordRule))
}

type UpdateAdmin struct {
	DisplayName *string `json:"display_name" form:"display_name"`
	Enabled     *bool   `json:"enabled" form:"enabled"`
}

func (r *UpdateAdmin) Validate() error {
	if r.DisplayName == nil && r.Enabled == nil {
		return v.Errors{"body": fmt.Errorf("至少提供一个修改字段")}
	}
	return v.ValidateStruct(r, v.Field(&r.DisplayName, v.RuneLength(0, 100).Error("名称最多 100 字符")))
}

type Password struct {
	Password string `json:"password" form:"password"`
}

func (r *Password) Validate() error { return v.ValidateStruct(r, v.Field(&r.Password, passwordRule)) }

type ChangePassword struct {
	CurrentPassword string `json:"current_password" form:"current_password"`
	Password        string `json:"password" form:"password"`
}

func (r *ChangePassword) Validate() error {
	return v.ValidateStruct(r, v.Field(&r.CurrentPassword, v.Required.Error("请输入当前密码")), v.Field(&r.Password, passwordRule))
}

type UpdateSettings struct {
	Values map[string]json.RawMessage `json:"values"`
}

func (r *UpdateSettings) Validate() error {
	return v.ValidateStruct(r, v.Field(&r.Values, v.Required.Error("请输入设置项")))
}
