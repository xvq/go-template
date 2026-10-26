package request

type Login struct {
	Username string `json:"username" label:"账号" validate:"required,alphanum,min=3,max=64"`
	Password string `json:"password" label:"密码" validate:"required,min=6,max=30"`
}

type CreateAdmin struct {
	Username    string `json:"username" label:"账号" validate:"required,alphanum,min=3,max=64"`
	DisplayName string `json:"display_name" label:"名称" validate:"max=100"`
	Password    string `json:"password" label:"密码" validate:"required,min=6,max=30"`
	Enabled     *bool  `json:"enabled"`
}

type UpdateAdmin struct {
	DisplayName string `json:"display_name" label:"名称" validate:"required,max=100"`
	Enabled     *bool  `json:"enabled" label:"启用状态" validate:"required"`
}

type Password struct {
	Password string `json:"password" label:"密码" validate:"required,min=6,max=30"`
}

type ChangePassword struct {
	CurrentPassword string `json:"current_password" label:"当前密码" validate:"required,min=6,max=30"`
	Password        string `json:"password" label:"密码" validate:"required,min=6,max=30"`
}
