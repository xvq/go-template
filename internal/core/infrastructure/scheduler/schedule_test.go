package scheduler

import (
	"context"
	"github.com/xvq/go-template/internal/core/config"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestInvalidCronFailsAndCanceledScheduleStops(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Defaults()
	task := Task{Name: "broken", Spec: "not-cron", Timeout: time.Second, Run: func(context.Context) error { return nil }}
	started := 0
	onStarted := func() { started++ }
	if Run(context.Background(), cfg, log, []Task{task}, onStarted) == nil {
		t.Fatal("invalid cron accepted")
	}
	if started != 0 {
		t.Fatal("startup notified before task validation succeeded")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, cfg, log, nil, onStarted); err != nil {
		t.Fatal(err)
	}
	if started != 1 {
		t.Fatalf("startup notifications=%d, want 1", started)
	}
	cfg.App.Timezone = "invalid/timezone"
	if err := Run(ctx, cfg, log, nil, onStarted); err == nil {
		t.Fatal("scheduler ignored invalid application timezone")
	}
	if started != 1 {
		t.Fatal("invalid timezone triggered startup notification")
	}
}
