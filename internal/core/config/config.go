package config

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App        App        `yaml:"app"`
	Server     Server     `yaml:"server"`
	Log        Log        `yaml:"log"`
	Database   Database   `yaml:"database"`
	Redis      Redis      `yaml:"redis"`
	Auth       Auth       `yaml:"auth"`
	HTTPClient HTTPClient `yaml:"http_client"`
	Schedule   Schedule   `yaml:"schedule"`
	Queue      Queue      `yaml:"queue"`
}
type App struct {
	Name     string `yaml:"name"`
	Timezone string `yaml:"timezone"`
}

// Location resolves the shared business timezone without changing time.Local.
func (a App) Location() (*time.Location, error) {
	if a.Timezone == "" || a.Timezone == "Local" {
		return nil, fmt.Errorf("app.timezone 必须为明确的 IANA 时区或 UTC")
	}
	loc, err := time.LoadLocation(a.Timezone)
	if err != nil {
		return nil, fmt.Errorf("app.timezone 无效: %w", err)
	}
	return loc, nil
}

type Server struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	BodyLimit       int           `yaml:"body_limit"`
	CORSOrigins     []string      `yaml:"cors_origins"`
	TrustedProxies  []string      `yaml:"trusted_proxies"`
}
type Log struct {
	Level      string `yaml:"level"`
	Output     string `yaml:"output"`
	Dir        string `yaml:"dir"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	Compress   bool   `yaml:"compress"`
}
type Database struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
	IOTimeout       time.Duration `yaml:"io_timeout"`
	SlowQuery       time.Duration `yaml:"slow_query"`
}
type Redis struct {
	Address  string        `yaml:"address"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	Timeout  time.Duration `yaml:"timeout"`
}
type Auth struct {
	Admin Session `yaml:"admin"`
}
type Session struct {
	TokenTTL              time.Duration `yaml:"token_ttl"`
	AllowMultipleSessions bool          `yaml:"allow_multiple_sessions"`
	LoginLimit            int           `yaml:"login_limit"`
	LoginWindow           time.Duration `yaml:"login_window"`
}
type HTTPClient struct {
	Timeout             time.Duration `yaml:"timeout"`
	ConnectTimeout      time.Duration `yaml:"connect_timeout"`
	MaxIdleConns        int           `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"`
	Proxy               string        `yaml:"proxy"`
}
type Schedule struct {
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}
type QueueRedis struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}
type Queue struct {
	Enabled         bool           `yaml:"enabled"`
	Concurrency     int            `yaml:"concurrency"`
	DefaultQueue    string         `yaml:"default_queue"`
	MaxRetry        int            `yaml:"max_retry"`
	TaskTimeout     time.Duration  `yaml:"task_timeout"`
	ShutdownTimeout time.Duration  `yaml:"shutdown_timeout"`
	Queues          map[string]int `yaml:"queues"`
	Redis           QueueRedis     `yaml:"redis"`
}

func Defaults() Config {
	return Config{
		App:        App{Name: "go-template", Timezone: "Asia/Hong_Kong"},
		Server:     Server{Host: "127.0.0.1", Port: 8080, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, RequestTimeout: 30 * time.Second, ShutdownTimeout: 15 * time.Second, BodyLimit: 4 << 20},
		Log:        Log{Level: "info", Output: "stdout", Dir: "logs", MaxSizeMB: 100, MaxBackups: 7, MaxAgeDays: 14, Compress: true},
		Database:   Database{Host: "127.0.0.1", Port: 3306, MaxOpenConns: 30, MaxIdleConns: 10, ConnMaxLifetime: 30 * time.Minute, ConnectTimeout: 5 * time.Second, IOTimeout: 30 * time.Second, SlowQuery: 500 * time.Millisecond},
		Redis:      Redis{Address: "127.0.0.1:6379", Timeout: 3 * time.Second},
		Auth:       Auth{Admin: Session{TokenTTL: 24 * time.Hour, AllowMultipleSessions: true, LoginLimit: 10, LoginWindow: time.Minute}},
		HTTPClient: HTTPClient{Timeout: 30 * time.Second, ConnectTimeout: 5 * time.Second, MaxIdleConns: 100, MaxIdleConnsPerHost: 10, IdleConnTimeout: 90 * time.Second},
		Schedule:   Schedule{ShutdownTimeout: 15 * time.Second},
		Queue:      Queue{Concurrency: 10, DefaultQueue: "default", MaxRetry: 3, TaskTimeout: 5 * time.Minute, ShutdownTimeout: 15 * time.Second, Queues: map[string]int{"default": 1}},
	}
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	return Parse(b)
}
func Parse(b []byte) (Config, error) {
	c := Defaults()
	d := yaml.NewDecoder(bytes.NewReader(b))
	d.KnownFields(true)
	if err := d.Decode(&c); err != nil {
		return c, fmt.Errorf("解析配置: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("配置只能包含一个 YAML 文档")
	}
	return c, nil
}

// Validate only the components required by the selected command.
func (c Config) Validate(parts ...string) error {
	if _, err := c.App.Location(); err != nil {
		return err
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
		return fmt.Errorf("log.level 无效")
	}
	if c.Log.Output != "stdout" && c.Log.Output != "file" && c.Log.Output != "both" {
		return fmt.Errorf("log.output 必须为 stdout/file/both")
	}
	if c.Log.MaxSizeMB <= 0 || c.Log.MaxBackups < 0 || c.Log.MaxAgeDays < 0 || strings.TrimSpace(c.Log.Dir) == "" {
		return fmt.Errorf("日志配置无效")
	}
	for _, p := range parts {
		switch p {
		case "database":
			d := c.Database
			if d.Host == "" || d.Port < 1 || d.Port > 65535 || d.Username == "" || d.Database == "" || d.MaxOpenConns < 1 || d.MaxIdleConns < 0 || d.MaxIdleConns > d.MaxOpenConns || d.ConnectTimeout <= 0 || d.IOTimeout <= 0 || d.ConnMaxLifetime <= 0 || d.SlowQuery <= 0 {
				return fmt.Errorf("database 配置缺失或无效")
			}
		case "redis":
			if c.Redis.Address == "" || c.Redis.DB < 0 || c.Redis.Timeout <= 0 {
				return fmt.Errorf("redis 配置无效")
			}
		case "server":
			s := c.Server
			if s.Port < 1 || s.Port > 65535 || s.BodyLimit < 1 || s.ReadTimeout <= 0 || s.WriteTimeout <= 0 || s.IdleTimeout <= 0 || s.RequestTimeout <= 0 || s.ShutdownTimeout <= 0 {
				return fmt.Errorf("server 配置无效")
			}
			a := c.Auth.Admin
			if a.TokenTTL < time.Second || a.LoginLimit < 1 || a.LoginWindow < time.Second {
				return fmt.Errorf("auth 配置无效")
			}
		case "http_client":
			h := c.HTTPClient
			if h.Timeout <= 0 || h.ConnectTimeout <= 0 || h.IdleConnTimeout <= 0 || h.MaxIdleConns < 1 || h.MaxIdleConnsPerHost < 1 {
				return fmt.Errorf("http_client 配置无效")
			}
			if h.Proxy != "" {
				u, e := url.Parse(h.Proxy)
				if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5") {
					return fmt.Errorf("http_client.proxy 无效")
				}
			}
		case "schedule":
			if c.Schedule.ShutdownTimeout <= 0 {
				return fmt.Errorf("schedule.shutdown_timeout 必须大于零")
			}
		case "queue":
			q := c.Queue
			if !q.Enabled {
				return fmt.Errorf("队列未启用，请设置 queue.enabled: true")
			}
			if q.Concurrency < 1 || q.MaxRetry < 0 || q.TaskTimeout <= 0 || q.ShutdownTimeout <= 0 || q.DefaultQueue == "" || q.Queues[q.DefaultQueue] <= 0 || q.Redis.DB < 0 {
				return fmt.Errorf("queue 配置无效")
			}
			for name, w := range q.Queues {
				if strings.TrimSpace(name) == "" || w < 1 {
					return fmt.Errorf("queue.queues 无效")
				}
			}
			if q.Redis.Address == "" && c.Redis.Address == "" {
				return fmt.Errorf("队列 Redis 地址不能为空")
			}
			if c.Redis.Timeout <= 0 || (q.Redis.Address == "" && c.Redis.DB < 0) {
				return fmt.Errorf("队列 Redis 超时或数据库编号无效")
			}
		default:
			return fmt.Errorf("未知配置组件: %s", p)
		}
	}
	return nil
}
