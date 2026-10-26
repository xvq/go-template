// Package testutil provides fixtures for serial tests using process globals.
package testutil

import (
	"log/slog"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/core"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"gorm.io/gorm"
)

// UseCore installs isolated test resources and restores all globals on cleanup.
// Tests using this helper must not call t.Parallel; stop background work before
// cleanup. The caller owns and closes the supplied clients.
func UseCore(t *testing.T, c config.Config, db *gorm.DB, redis *redis.Client, log *slog.Logger) {
	t.Helper()
	previousConfig, previousDB, previousRedis := core.Config, core.DB, core.Redis
	previousLog, previousHTTP, previousQueue := core.Log, core.HTTP, core.Queue
	previousSession := core.Session
	core.Config, core.DB, core.Redis, core.Log = c, db, redis, log
	core.HTTP, core.Queue = nil, nil
	core.Session = nil
	if redis != nil {
		core.Session = session.New(redis)
	}
	t.Cleanup(func() {
		core.Config, core.DB, core.Redis = previousConfig, previousDB, previousRedis
		core.Log, core.HTTP, core.Queue = previousLog, previousHTTP, previousQueue
		core.Session = previousSession
	})
}
