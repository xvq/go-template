# GoTemplate

基于 Fiber v3、GORM/MySQL 和 Redis 的 Go 纯 API 项目模板，用于快速开始新项目。内置管理员管理、登录日志、系统设置、Redis 会话、定时任务和可选 Asynq 队列，不包含前端、普通用户业务或 RBAC。

## 快速开始

需要 Go 1.26+、MySQL 8.0+、Redis 7+。支持 Windows 开发与 Linux 部署，不依赖 CGO。

### 1. 创建项目

使用 gonew 将模板复制为自己的 Go 模块：

```sh
go install golang.org/x/tools/cmd/gonew@latest
gonew github.com/xvq/go-template@main github.com/yourname/myapp
cd myapp
go mod download
go run ./cmd config:init
```

编辑生成的 `config.yaml`，设置应用名称、MySQL 和 Redis 连接。配置示例见 [config.example.yaml](config.example.yaml)；实际配置文件已被 Git 忽略。

已有 MySQL/Redis 可直接使用，也可以启动项目提供的本地开发服务：

```sh
docker compose up -d --wait
```

Compose 创建数据库 `go_template`，账号 `app`，密码 `local-app-password`；将密码填入配置。这里的账号密码仅用于本地开发。

### 2. 初始化并运行

```sh
go run ./cmd migrate up
go run ./cmd admin:create --username admin --name 管理员
go run ./cmd server
```

创建管理员时交互输入密码；脚本可使用 `--password-stdin`。项目没有默认管理员。

服务默认地址为 `http://127.0.0.1:8080`。调用 `POST /api/admin/login`，以 JSON 提交 `username`、`password` 获取 Token，后续请求携带 `Authorization: Bearer <token>`。完整接口见 [API 文档](docs/api.md)。

## 目录结构

```text
cmd/main.go                 程序入口
internal/
  app/
    admin/                  管理端应用
      routes.go             应用路由与业务对象组装
      handler/              接收请求、调用服务、组织响应
      request/              请求结构与校验规则
      service/              应用业务规则、查询和事务
      resource/             输出字段与模型转换
      middleware/           管理员认证、登录限流
    web/                    网站端路由扩展入口，暂无业务
  core/
    core.go                 Config/DB/Redis/Session/Log/HTTP/Queue 全局入口
    bootstrap/              资源初始化、依赖声明与服务生命周期
    config/                 配置读取与校验
    infrastructure/         底层客户端、CLI、会话、校验器、队列和调度引擎
    route/                  应用路由注册入口
    middleware/             全局中间件与请求生命周期
    request/                公共绑定校验、ID、分页和 Token 提取
    response/               统一响应、错误处理与流式输出
    resource/               通用 Item/Collection/Page 转换
    testutil/               测试辅助
  model/                    数据模型，每个表一个文件
  service/                  可复用能力，当前包含 settings
  util/                     小型公共工具：query、str
  command/                  业务命令及注册入口
  job/                      队列任务及注册入口
  schedule/                 定时任务及注册入口
migrations/                 版本化 SQL，编译时嵌入程序
docs/                       开发、API 和部署文档
integration/                真实 MySQL/Redis 集成测试
scripts/build.go            构建脚本
```

应用使用 `request → handler → service → resource/response` 的分工。公共资源由 bootstrap 初始化，通过 `core` 使用；业务直接使用 GORM，不额外封装通用 Repository。新增功能的位置、注册入口和已有工具见 [开发文档](docs/development.md)。

## 常用命令

开发时使用 `go run ./cmd <命令>`；构建后使用 `app <命令>`。通过 `-c <路径>` 指定配置文件。

| 命令 | 用途 |
| --- | --- |
| `config:init` | 生成配置文件 |
| `server` | 运行 HTTP API |
| `migrate up` / `migrate version` | 应用迁移 / 查看迁移状态 |
| `admin:create --username admin` | 创建管理员 |
| `schedule` | 运行定时任务，内置可移除的心跳示例 |
| `queue:work` | 运行队列消费者 |
| `queue:example --delay 10s` | 投递示例任务 |
| `settings:clear-cache` | 清理系统设置缓存 |
| `version` | 查看构建信息 |
| `--help` | 查看命令及参数 |

队列默认关闭，使用前设置 `queue.enabled: true`。HTTP、调度和队列消费者分别运行；HTTP 服务不会自动启动消费者。

## 检查与构建

```sh
go test ./...
go vet ./...
go run ./scripts/build.go
```

本机构建输出为 `bin/app`，Windows 为 `bin/app.exe`。默认测试不连接真实 MySQL/Redis；集成测试运行方式见开发文档。

- [开发文档](docs/development.md)：扩展方式、公共能力与项目约定。
- [API 文档](docs/api.md)：认证、接口、参数与响应。
- [部署文档](docs/deployment.md)：构建、配置、进程与维护。
