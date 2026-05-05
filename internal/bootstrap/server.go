package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/router"
	"github.com/xvq/go-template/internal/validator"
)

func StartServer() {
	core.InitLogger("server")
	core.InitDB()
	core.InitRedis()
	validator.Init()

	gin.SetMode(config.AppConfig.Server.Mode)
	r := gin.New()
	router.Setup(r)

	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	slog.Info("server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
