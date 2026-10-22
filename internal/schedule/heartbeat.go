package schedule

import (
	"context"
	"github.com/xvq/go-template/internal/infrastructure/scheduler"
	"log/slog"
	"time"
)

// newHeartbeat demonstrates a scheduled task with an injected dependency.
func newHeartbeat(log *slog.Logger) scheduler.Task {
	return scheduler.Task{
		Name:    "heartbeat",
		Spec:    "@every 1m",
		Timeout: 5 * time.Second,
		Run: func(ctx context.Context) error {
			log.DebugContext(ctx, "scheduler heartbeat")
			return nil
		},
	}
}
