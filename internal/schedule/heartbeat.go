package schedule

import (
	"context"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/infrastructure/scheduler"
	"time"
)

// newHeartbeat demonstrates a scheduled task using the shared logger.
func newHeartbeat() scheduler.Task {
	return scheduler.Task{
		Name:    "heartbeat",
		Spec:    "@every 1m",
		Timeout: 5 * time.Second,
		Run: func(ctx context.Context) error {
			core.Log.DebugContext(ctx, "scheduler heartbeat")
			return nil
		},
	}
}
