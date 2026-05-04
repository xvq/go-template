package bootstrap

import (
	"log/slog"

	"github.com/robfig/cron/v3"
	_ "github.com/user/go-template/internal/app/worker"
	"github.com/user/go-template/internal/config"
	"github.com/user/go-template/internal/support"
)

func StartWorker(cfg *config.Config) {
	support.NewLogger(cfg.Log, "worker")
	support.NewDB(cfg)
	support.NewRedis(cfg)

	c := cron.New()
	for _, w := range support.GetWorkers() {
		if w.Cron == "" {
			slog.Info("worker starting (standalone)", "name", w.Name)
			go w.Run()
		} else {
			slog.Info("worker registered", "name", w.Name, "cron", w.Cron)
			c.AddFunc(w.Cron, w.Run)
		}
	}

	if len(c.Entries()) == 0 {
		slog.Warn("no cron workers registered")
	}

	slog.Info("worker starting")
	c.Run()
}
