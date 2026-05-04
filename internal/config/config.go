package config

import (
	"log/slog"

	"github.com/xvq/go-template/internal/common"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Default     string              `yaml:"default"`
	Connections map[string]DBConfig `yaml:"connections"`
}

type DBConfig struct {
	Driver       string `yaml:"driver"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	File         string `yaml:"file"`   // sqlite
	Charset      string `yaml:"charset"`
	Collation    string `yaml:"collation"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

func (c *Config) DefaultDB() *DBConfig {
	conn, ok := c.Database.Connections[c.Database.Default]
	if !ok {
		return nil
	}
	return &conn
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LogConfig struct {
	Level      slog.Level `yaml:"level"`
	Dir        string     `yaml:"dir"`
	MaxSize    int        `yaml:"max_size"`
	MaxBackups int        `yaml:"max_backups"`
	MaxAge     int        `yaml:"max_age"`
	Compress   bool       `yaml:"compress"`
}

var AppConfig *Config

func Load(path string) *Config {
	cfg := &Config{
		Server: ServerConfig{Port: 8080, Mode: "debug"},
		Database: DatabaseConfig{
			Default: "mysql",
			Connections: map[string]DBConfig{
				"mysql": {Driver: "mysql", Port: 3306, MaxIdleConns: 10, MaxOpenConns: 100},
			},
		},
		Redis: RedisConfig{Port: 6379},
		Log: LogConfig{
			Level: slog.LevelError, Dir: "logs/",
			MaxSize: 100, MaxBackups: 7, MaxAge: 7, Compress: true,
		},
	}

	data, err := common.ReadFile(path)
	if err != nil {
		panic("读取配置文件失败: " + err.Error())
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		panic("解析配置文件失败: " + err.Error())
	}

	return cfg
}
