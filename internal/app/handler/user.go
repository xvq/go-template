package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user/go-template/internal/app/dto"
	"github.com/user/go-template/internal/app/model"
	"github.com/user/go-template/internal/common"
	"github.com/user/go-template/internal/support"
)

func GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.Error(c, 40001, "参数错误")
		return
	}

	var user model.User
	if result := support.DB.First(&user, id); result.Error != nil {
		common.Error(c, 40401, "用户不存在")
		return
	}

	common.Success(c, dto.ToUserResponse(&user))
}

func ListUsers(c *gin.Context) {
	var users []model.User
	support.DB.Find(&users)

	common.Success(c, dto.ToUserResponses(users))
}

func CreateUser(c *gin.Context, req *dto.CreateUserRequest) {
	user := model.User{Name: req.Name, Email: req.Email, Password: req.Password}
	if err := support.DB.Create(&user).Error; err != nil {
		common.Error(c, 50001, "创建失败")
		return
	}

	common.Success(c, nil)
}

func UpdateUser(c *gin.Context, req *dto.UpdateUserRequest) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.Error(c, 40001, "参数错误")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if err := support.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		common.Error(c, 50001, "更新失败")
		return
	}

	common.Success(c, nil)
}

func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.Error(c, 40001, "参数错误")
		return
	}

	if err := support.DB.Delete(&model.User{}, id).Error; err != nil {
		common.Error(c, 50001, "删除失败")
		return
	}

	common.Success(c, nil)
}
