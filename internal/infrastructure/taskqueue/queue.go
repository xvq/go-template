package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hibiken/asynq"
	"github.com/xvq/go-template/internal/config"
	"log/slog"
)

var ErrDisabled = errors.New("队列未启用")

type Client struct {
	client *asynq.Client
	cfg    config.Queue
}

func RedisOptions(c config.Config) asynq.RedisClientOpt {
	q := c.Queue.Redis
	if q.Address == "" {
		return asynq.RedisClientOpt{Addr: c.Redis.Address, Username: c.Redis.Username, Password: c.Redis.Password, DB: c.Redis.DB, DialTimeout: c.Redis.Timeout, ReadTimeout: c.Redis.Timeout, WriteTimeout: c.Redis.Timeout}
	}
	return asynq.RedisClientOpt{Addr: q.Address, Username: q.Username, Password: q.Password, DB: q.DB, DialTimeout: c.Redis.Timeout, ReadTimeout: c.Redis.Timeout, WriteTimeout: c.Redis.Timeout}
}
func NewClient(c config.Config) *Client {
	result := &Client{cfg: c.Queue}
	if c.Queue.Enabled {
		result.client = asynq.NewClient(RedisOptions(c))
	}
	return result
}
func (c *Client) Enqueue(ctx context.Context, taskType string, payload any, options ...asynq.Option) (*asynq.TaskInfo, error) {
	if c.client == nil {
		return nil, ErrDisabled
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	defaults := []asynq.Option{asynq.Queue(c.cfg.DefaultQueue), asynq.MaxRetry(c.cfg.MaxRetry), asynq.Timeout(c.cfg.TaskTimeout)}
	return c.client.EnqueueContext(ctx, asynq.NewTask(taskType, body), append(defaults, options...)...)
}
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

type slogAdapter struct{ log *slog.Logger }

func (l slogAdapter) Debug(args ...interface{}) { l.log.Debug("queue", "detail", args) }
func (l slogAdapter) Info(args ...interface{})  { l.log.Info("queue", "detail", args) }
func (l slogAdapter) Warn(args ...interface{})  { l.log.Warn("queue", "detail", args) }
func (l slogAdapter) Error(args ...interface{}) { l.log.Error("queue", "detail", args) }
func (l slogAdapter) Fatal(args ...interface{}) { l.log.Error("queue fatal", "detail", args) }
func NewServer(c config.Config, log *slog.Logger) *asynq.Server {
	return asynq.NewServer(RedisOptions(c), asynq.Config{Concurrency: c.Queue.Concurrency, Queues: c.Queue.Queues, ShutdownTimeout: c.Queue.ShutdownTimeout, Logger: slogAdapter{log}, ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, t *asynq.Task, err error) {
		log.ErrorContext(ctx, "task failed", "type", t.Type(), "error", err)
	})})
}
