# GoTemplate

Go 项目模板，单二进制多命令架构。

## 目录结构

```
├── cmd/main.go                  # 程序入口
├── internal/
│   ├── app/                     # 业务代码（频繁变动）
│   │   ├── handler/             # HTTP 处理层
│   │   ├── model/               # 数据库实体
│   │   ├── dto/                 # 请求/响应结构体
│   │   ├── middleware/          # 中间件
│   │   └── worker/              # 定时任务
│   ├── bootstrap/               # 应用启动（不常动）
│   ├── command/                 # CLI 命令注册
│   ├── common/                  # 工具函数、统一响应
│   ├── config/                  # 配置结构体
│   ├── router/                  # 路由注册
│   ├── support/                 # 基础支撑（DB、Redis、Log）
│   └── validator/               # 参数校验 + 自动绑定
├── migrations/                  # SQL 迁移文件（编译进二进制）
├── embed.go                     # 静态资源打包
├── config.example.yaml          # 配置模板（提交 git）
├── config.yaml                  # 运行时配置（gitignore）
├── go.mod
└── README.md
```

## 快速开始

```bash
# 复制配置
cp config.example.yaml config.yaml

# 安装依赖
go mod tidy

# 创建数据库
mysql -u root -e "CREATE DATABASE my_service DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci"

# 数据库迁移
go run cmd/main.go -c config.yaml migrate up

# 启动服务
go run cmd/main.go -c config.yaml server
```

## 用法

```bash
./app                            # 帮助
./app -c config.yaml server      # 启动 HTTP 服务
./app -c config.yaml worker      # 启动定时任务
./app -c config.yaml migrate up  # 数据库迁移
./app -c config.yaml migrate down
./app -c config.yaml migrate steps 2
./app -c config.yaml migrate version
```

## 技术栈

- **HTTP**: gin
- **ORM**: gorm（mysql / postgres / sqlite）
- **配置**: yaml.v3
- **校验**: go-playground/validator
- **定时任务**: robfig/cron
- **数据库迁移**: golang-migrate（嵌入二进制）
- **缓存**: go-redis
- **日志**: slog + lumberjack（按大小轮转，JSON 格式）

## 使用 gonew 创建新项目

```bash
gonew github.com/user/go-template example.com/my-service
```
