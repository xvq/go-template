package taskqueue

import (
	"context"
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/hibiken/asynq"
	"github.com/xvq/go-template/internal/core/config"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestDisabledQueueAndEnqueue(t *testing.T) {
	c := config.Defaults()
	client := NewClient(c)
	if _, err := client.Enqueue(context.Background(), "example", nil); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
	m := miniredis.RunT(t)
	c.Queue.Enabled = true
	c.Redis.Address = m.Addr()
	client = NewClient(c)
	defer client.Close()
	info, err := client.Enqueue(context.Background(), "example", map[string]int{"id": 1}, asynq.ProcessIn(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if info.State != asynq.TaskStateScheduled || info.MaxRetry != c.Queue.MaxRetry || info.Timeout != c.Queue.TaskTimeout {
		t.Fatalf("bad task %+v", info)
	}
	inspector := asynq.NewInspector(RedisOptions(c))
	defer inspector.Close()
	tasks, err := inspector.ListScheduledTasks(c.Queue.DefaultQueue)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks=%v err=%v", tasks, err)
	}
}

func TestWorkerConsumesTask(t *testing.T) {
	m := miniredis.RunT(t)
	c := config.Defaults()
	c.Queue.Enabled = true
	c.Redis.Address = m.Addr()
	c.Queue.Concurrency = 1
	server := NewServer(c, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := asynq.NewServeMux()
	done := make(chan struct{}, 1)
	mux.HandleFunc("test:ping", func(ctx context.Context, task *asynq.Task) error {
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})
	if err := server.Start(mux); err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown()
	client := NewClient(c)
	defer client.Close()
	if _, err := client.Enqueue(context.Background(), "test:ping", map[string]int{"id": 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("task was not consumed")
	}
}
