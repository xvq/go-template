package bootstrap

import (
	"log/slog"

	"github.com/robfig/cron/v3"
	_ "github.com/xvq/go-template/internal/app/worker"
	"github.com/xvq/go-template/internal/core"
)

func StartWorker() {
	core.InitLogger("worker")
	core.InitDB()
	core.InitRedis()

	c := cron.New()
	for _, w := range core.GetWorkers() {
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
