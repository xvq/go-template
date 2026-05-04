package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/user/go-template/internal/config"
	"github.com/user/go-template/internal/router"
	"github.com/user/go-template/internal/support"
	"github.com/user/go-template/internal/validator"
)

func StartServer(cfg *config.Config) {
	support.NewLogger(cfg.Log, "server")
	support.NewDB(cfg)
	support.NewRedis(cfg)
	validator.Init()

	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	router.Setup(r)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
