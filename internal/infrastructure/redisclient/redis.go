package redisclient

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/xvq/go-template/internal/config"
)

func Open(ctx context.Context, c config.Redis) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{Addr: c.Address, Username: c.Username, Password: c.Password, DB: c.DB, DialTimeout: c.Timeout, ReadTimeout: c.Timeout, WriteTimeout: c.Timeout, ContextTimeoutEnabled: true})
	ping, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()
	if err := r.Ping(ping).Err(); err != nil {
		r.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return r, nil
}
