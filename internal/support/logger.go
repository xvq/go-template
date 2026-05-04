package support

import (
	"log/slog"

	"github.com/user/go-template/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLogger(cfg config.LogConfig, name string) {
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
