package core

import (
	"log/slog"

	"github.com/xvq/go-template/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(name string) {
	cfg := config.AppConfig.Log
	w := &lumberjack.Logger{
		Filename:   cfg.Dir + name + "/" + name + ".log",
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: cfg.Level,
	})

	slog.SetDefault(slog.New(handler))
}
