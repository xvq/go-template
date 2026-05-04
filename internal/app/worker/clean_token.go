package worker

import (
	"log/slog"

	"github.com/xvq/go-template/internal/support"
)

func init() {
	support.RegisterWorker(&support.Worker{
		Name: "clean_token",
		Cron: "@every 30m",
		Run:  run,
	})
}

func run() {
	slog.Info("clean_token worker running")
}
