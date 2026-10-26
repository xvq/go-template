package scheduler

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/xvq/go-template/internal/core/config"
	"log/slog"
	"time"
)

type cronLog struct{ log *slog.Logger }

func (l cronLog) Info(msg string, kv ...interface{}) { l.log.Debug(msg, kv...) }
func (l cronLog) Error(err error, msg string, kv ...interface{}) {
	l.log.Error(msg, append(kv, "error", err)...)
}
func Run(ctx context.Context, c config.Schedule, log *slog.Logger, tasks []Task) error {
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return err
	}
	cl := cronLog{log}
	scheduler := cron.New(cron.WithLocation(loc), cron.WithChain(cron.Recover(cl), cron.SkipIfStillRunning(cl)))
	seen := map[string]bool{}
	for _, task := range tasks {
		if task.Name == "" || seen[task.Name] || task.Run == nil || task.Timeout <= 0 {
			return fmt.Errorf("invalid/duplicate scheduled task: %s", task.Name)
		}
		seen[task.Name] = true
		t := task
		if _, err = scheduler.AddFunc(t.Spec, func() {
			run, cancel := context.WithTimeout(ctx, t.Timeout)
			defer cancel()
			start := time.Now()
			if err := t.Run(run); err != nil {
				log.Error("scheduled task failed", "name", t.Name, "error", err)
			} else {
				log.Info("scheduled task completed", "name", t.Name, "elapsed", time.Since(start))
			}
		}); err != nil {
			return fmt.Errorf("register schedule %s: %w", t.Name, err)
		}
	}
	scheduler.Start()
	log.Info("scheduler started", "tasks", len(tasks))
	<-ctx.Done()
	stopped := scheduler.Stop()
	timer := time.NewTimer(c.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-stopped.Done():
		return nil
	case <-timer.C:
		return fmt.Errorf("scheduler shutdown timed out")
	}
}
