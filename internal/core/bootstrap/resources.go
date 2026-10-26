package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/database"
	"github.com/xvq/go-template/internal/core/infrastructure/httpclient"
	"github.com/xvq/go-template/internal/core/infrastructure/logger"
	"github.com/xvq/go-template/internal/core/infrastructure/redisclient"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/core/infrastructure/taskqueue"
	"gorm.io/gorm"
)

// Options selects resources to initialize. Config and Log are always prepared.
// Name labels the logger; omitted resource flags default to false.
type Options struct {
	Name     string
	Database bool
	Redis    bool
	HTTP     bool
	Queue    bool
}

// Resources owns cleanup only; business code uses core.
type Resources struct {
	log     *slog.Logger
	closers []func() error
	closed  bool
}

var lifecycle struct {
	sync.Mutex
	active *Resources
}

// Open publishes globals only after initialization succeeds. Run once per
// command before starting handlers/workers, and Close after they have stopped.
func Open(ctx context.Context, c config.Config, options Options) (*Resources, error) {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	if lifecycle.active != nil {
		return nil, fmt.Errorf("公共资源已初始化，请先停止任务并关闭上一组资源")
	}
	var parts []string
	if options.Database {
		parts = append(parts, "database")
	}
	if options.Redis {
		parts = append(parts, "redis")
	}
	if options.HTTP {
		parts = append(parts, "http_client")
	}
	if options.Queue {
		parts = append(parts, "queue")
	}
	if err := c.Validate(parts...); err != nil {
		return nil, err
	}
	log, closeLog, err := logger.New(c.Log, options.Name)
	if err != nil {
		return nil, err
	}
	r := &Resources{log: log, closers: []func() error{closeLog}}
	var db *gorm.DB
	var redisConn *redis.Client
	var sessions *session.Store
	var http *resty.Client
	disabled := c
	disabled.Queue.Enabled = false
	queue := taskqueue.NewClient(disabled)
	fail := func(err error) (*Resources, error) { r.close(); return nil, err }
	if options.Database {
		db, err = database.Open(c.Database, log)
		if err != nil {
			return fail(err)
		}
		pool, e := db.DB()
		if e != nil {
			return fail(e)
		}
		r.closers = append(r.closers, pool.Close)
	}
	if options.Redis {
		redisConn, err = redisclient.Open(ctx, c.Redis)
		if err != nil {
			return fail(err)
		}
		r.closers = append(r.closers, redisConn.Close)
		sessions = session.New(redisConn, c.Redis.Prefix)
	}
	if options.HTTP {
		http = httpclient.New(c.HTTPClient)
		r.closers = append(r.closers, func() error { http.GetClient().CloseIdleConnections(); return nil })
	}
	if options.Queue {
		queue = taskqueue.NewClient(c)
		r.closers = append(r.closers, queue.Close)
	}
	core.Config, core.Log = c, log
	core.DB, core.Redis, core.HTTP, core.Queue = db, redisConn, http, queue
	core.Session = sessions
	lifecycle.active = r
	return r, nil
}

// Close is idempotent. All work using the globals must have stopped first.
func (r *Resources) Close() {
	lifecycle.Lock()
	defer lifecycle.Unlock()
	if r.closed {
		return
	}
	r.close()
	if lifecycle.active == r {
		core.Config = config.Config{}
		core.DB, core.Redis, core.Log, core.HTTP, core.Queue = nil, nil, nil, nil, nil
		core.Session = nil
		lifecycle.active = nil
	}
}

func (r *Resources) close() {
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i](); err != nil {
			r.log.Warn("resource close failed", "error", err)
		}
	}
	r.closers = nil
	r.closed = true
}
