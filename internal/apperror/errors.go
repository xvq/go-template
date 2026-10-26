// Package apperror defines shared application errors without HTTP dependencies.
package apperror

import "errors"

var (
	ErrNotFound          = errors.New("记录不存在")
	ErrConflict          = errors.New("账号已存在")
	ErrUnauthorized      = errors.New("登录已失效，请重新登录")
	ErrCredentials       = errors.New("账号或密码不正确")
	ErrLastAdmin         = errors.New("不能删除或禁用最后一个启用的管理员")
	ErrInvalidSetting    = errors.New("设置项不存在或值不合法")
	ErrInvalidPagination = errors.New("分页参数无效：page 范围 1～1000000，page_size 范围 1～100")
	ErrRateLimited       = errors.New("登录尝试过于频繁，请稍后再试")
)
