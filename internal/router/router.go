package router

import (
	"github.com/gin-gonic/gin"
	"github.com/user/go-template/internal/app/handler"
	"github.com/user/go-template/internal/app/middleware"
	"github.com/user/go-template/internal/validator"
)

func Setup(r *gin.Engine) {
	r.Use(gin.Recovery(), middleware.Logger())

	api := r.Group("/api")
	{
		api.GET("/users", handler.ListUsers)
		api.GET("/users/:id", handler.GetUser)
		api.POST("/users", 	validator.Bind(handler.CreateUser))
		api.PUT("/users/:id", 	validator.Bind(handler.UpdateUser))
		api.DELETE("/users/:id", handler.DeleteUser)
	}
}
