package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/xvq/go-template/internal/core/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

func DSN(c config.Database) string {
	d := driver.NewConfig()
	d.User = c.Username
	d.Passwd = c.Password
	d.Net = "tcp"
	d.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	d.DBName = c.Database
	d.ParseTime = true
	d.Loc = time.UTC
	d.Timeout = c.ConnectTimeout
	d.ReadTimeout = c.IOTimeout
	d.WriteTimeout = c.IOTimeout
	d.Params = map[string]string{"charset": "utf8mb4", "time_zone": "'+00:00'"}
	return d.FormatDSN()
}
func Open(c config.Database, log *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(DSN(c)), &gorm.Config{TranslateError: true, Logger: sqlLogger{log: log, slow: c.SlowQuery, level: glog.Warn}, NowFunc: func() time.Time { return time.Now().UTC().Truncate(time.Second) }})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(c.MaxOpenConns)
	pool.SetMaxIdleConns(c.MaxIdleConns)
	pool.SetConnMaxLifetime(c.ConnMaxLifetime)
	return db, nil
}

// Never logs SQL/parameters: password hashes and settings may appear in them.
type sqlLogger struct {
	log   *slog.Logger
	slow  time.Duration
	level glog.LogLevel
}

func (l sqlLogger) LogMode(level glog.LogLevel) glog.Interface { l.level = level; return l }
func (l sqlLogger) Info(ctx context.Context, s string, a ...interface{}) {
	if l.level >= glog.Info {
		l.log.DebugContext(ctx, fmt.Sprintf(s, a...))
	}
}
func (l sqlLogger) Warn(ctx context.Context, s string, a ...interface{}) {
	if l.level >= glog.Warn {
		l.log.WarnContext(ctx, fmt.Sprintf(s, a...))
	}
}
func (l sqlLogger) Error(ctx context.Context, s string, a ...interface{}) {
	if l.level >= glog.Error {
		l.log.ErrorContext(ctx, fmt.Sprintf(s, a...))
	}
}
func (l sqlLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == glog.Silent {
		return
	}
	elapsed := time.Since(begin)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.log.ErrorContext(ctx, "database operation failed", "elapsed", elapsed, "error_type", fmt.Sprintf("%T", err))
	} else if elapsed > l.slow {
		l.log.WarnContext(ctx, "slow database operation", "elapsed", elapsed)
	}
}
