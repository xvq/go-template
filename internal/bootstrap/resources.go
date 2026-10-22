package bootstrap

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/config"
	"github.com/xvq/go-template/internal/infrastructure/database"
	"github.com/xvq/go-template/internal/infrastructure/httpclient"
	"github.com/xvq/go-template/internal/infrastructure/logger"
	"github.com/xvq/go-template/internal/infrastructure/redisclient"
	"github.com/xvq/go-template/internal/infrastructure/taskqueue"
	"gorm.io/gorm"
	"log/slog"
)

type Resources struct {
	Config  config.Config
	Log     *slog.Logger
	DB      *gorm.DB
	Redis   *redis.Client
	HTTP    *resty.Client
	Queue   *taskqueue.Client
	closers []func() error
}

func Open(ctx context.Context, c config.Config, name string, parts ...string) (*Resources, error) {
	if err := c.Validate(parts...); err != nil {
		return nil, err
	}
	log, closeLog, err := logger.New(c.Log, name)
	if err != nil {
		return nil, err
	}
	r := &Resources{Config: c, Log: log, closers: []func() error{closeLog}}
	fail := func(err error) (*Resources, error) { r.Close(); return nil, err }
	seen := map[string]bool{}
	for _, p := range parts {
		if seen[p] {
			continue
		}
		seen[p] = true
		switch p {
		case "database":
			r.DB, err = database.Open(c.Database, log)
			if err != nil {
				return fail(err)
			}
			pool, e := r.DB.DB()
			if e != nil {
				return fail(e)
			}
			r.closers = append(r.closers, pool.Close)
		case "redis":
			r.Redis, err = redisclient.Open(ctx, c.Redis)
			if err != nil {
				return fail(err)
			}
			r.closers = append(r.closers, r.Redis.Close)
		case "http_client":
			r.HTTP = httpclient.New(c.HTTPClient)
			r.closers = append(r.closers, func() error { r.HTTP.GetClient().CloseIdleConnections(); return nil })
		case "queue":
			r.Queue = taskqueue.NewClient(c)
			r.closers = append(r.closers, r.Queue.Close)
		}
	}
	if r.Queue == nil {
		disabled := c
		disabled.Queue.Enabled = false
		r.Queue = taskqueue.NewClient(disabled)
	}
	return r, nil
}
func (r *Resources) Close() {
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i](); err != nil {
			r.Log.Warn("resource close failed", "error", err)
		}
	}
	r.closers = nil
}
