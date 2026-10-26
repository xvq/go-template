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
	cfg := config.Defaults().Schedule
	task := Task{Name: "broken", Spec: "not-cron", Timeout: time.Second, Run: func(context.Context) error { return nil }}
	if Run(context.Background(), cfg, log, []Task{task}) == nil {
		t.Fatal("invalid cron accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, cfg, log, nil); err != nil {
		t.Fatal(err)
	}
}
