package support

import (
	"fmt"

	"github.com/xvq/go-template/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func NewDB(cfg *config.Config) {
	conn := cfg.DefaultDB()
	if conn == nil || dbHost(conn) == "" {
		return
	}

	var err error
	DB, err = openDB(conn)
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(conn.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conn.MaxOpenConns)
}

func openDB(cfg *config.DBConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{}

	switch cfg.Driver {
	case "mysql":
		return gormMysql(cfg, gormCfg)
	case "postgres":
		return gormPostgres(cfg, gormCfg)
	case "sqlite":
		return gormSqlite(cfg, gormCfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}
}

func gormMysql(cfg *config.DBConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database, charset)
	if cfg.Collation != "" {
		dsn += "&collation=" + cfg.Collation
	}
	return gorm.Open(mysql.Open(dsn), gormCfg)
}

func gormPostgres(cfg *config.DBConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Host, cfg.Username, cfg.Password, cfg.Database, cfg.Port)
	return gorm.Open(postgres.Open(dsn), gormCfg)
}

func gormSqlite(cfg *config.DBConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(cfg.File), gormCfg)
}

func dbHost(cfg *config.DBConfig) string {
	if cfg.Driver == "sqlite" {
		return cfg.File
	}
	return cfg.Host
}
