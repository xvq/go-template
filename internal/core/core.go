// Package core exposes process-wide infrastructure initialized by bootstrap.Open.
// Treat these values as read-only after startup. Business code must not replace
// them or close shared clients; the startup owner controls their lifecycle.
package core

import (
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/core/config"
	"github.com/xvq/go-template/internal/core/infrastructure/session"
	"github.com/xvq/go-template/internal/core/infrastructure/taskqueue"
	"gorm.io/gorm"
)

var (
	Config config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	// Session shares Redis and is available only when Redis is initialized.
	Session *session.Store
	Log     *slog.Logger
	HTTP    *resty.Client
	Queue   *taskqueue.Client
)
