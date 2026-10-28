# 部署指南

## 构建与配置

在项目根目录运行：

```sh
go test ./...
go vet ./...
go run ./scripts/build.go
```

本机构建输出为 `bin/app`（Windows 为 `bin/app.exe`）。Linux amd64 交叉编译可明确指定输出：

```sh
go run ./scripts/build.go -os linux -arch amd64 -o bin/app-linux-amd64
```

构建脚本写入 Git 版本、提交号和 UTC 构建时间，通过 `app version` 查看。发布时使用干净工作区和完整标签历史。

部署二进制及独立的 `config.yaml`；迁移 SQL 已嵌入程序，不需要复制 migrations 目录。使用 `app config:init` 生成配置，选项见 [config.example.yaml](../config.example.yaml)。配置仅来自 YAML，不自动读取 `.env` 或环境变量；默认路径相对于工作目录，部署建议使用 `-c` 指定绝对路径。

| 配置区 | 部署时关注 |
| --- | --- |
| `app` | 应用名称、业务与调度时区（默认 Asia/Shanghai） |
| `server` | 监听地址、超时、请求体上限、可信代理和 CORS |
| `database` | MySQL 连接、连接池和超时 |
| `redis` | 会话、登录限流与设置缓存连接 |
| `auth.admin` | 会话有效期、多会话策略和登录限流 |
| `log` | 日志级别、stdout/file/both、文件轮转 |
| `http_client` | 对外 HTTP 请求超时、连接池和代理；客户端默认不初始化，需在代码资源依赖中启用 |
| `schedule` | 调度退出等待时间 |
| `queue` | 启用开关、队列权重、并发、重试和超时 |

配置文件仅授予运行账号必要的读取权限；写文件日志时确保日志目录可写。根目录 Compose 及其示例密码用于本地开发，不能直接作为生产配置。

## 初始化与运行

以下命令中的 `app` 为部署后的二进制：

```sh
./app -c /etc/my-app/config.yaml migrate up
./app -c /etc/my-app/config.yaml admin:create --username admin --name 管理员
./app -c /etc/my-app/config.yaml server
```

通过 systemd、容器或进程管理器分别管理需要运行的进程：

| 命令 | 部署方式 |
| --- | --- |
| `server` | HTTP API，可多实例 |
| `schedule` | 有定时任务时运行，只部署一个实例 |
| `queue:work` | 启用队列时独立运行，可多实例 |

`queue.enabled` 默认 false；`queue.concurrency` 是每个消费者进程的并发数。`queue.redis.address` 为空时复用公共 Redis 的全部连接配置，包括 DB；需要独立队列 DB 时填写地址及对应连接配置。

SIGINT/SIGTERM 触发优雅退出，各进程按配置等待请求或任务结束；进程管理器的停止等待时间应留出相应余量。Windows 本地使用 Ctrl+C。

## 网络与 Redis

服务默认监听 `127.0.0.1:8080`，容器或外部访问需调整 `server.host`。TLS 通常由反向代理终止；`server.trusted_proxies` 只填写实际代理 IP/CIDR，并由代理规范转发头，确保登录限流使用正确的客户端 IP。

跨域默认关闭，需要时在 `server.cors_origins` 声明允许来源。

项目使用固定用途的 Redis 键名，不同项目必须使用独立 DB 或实例。逻辑 DB 不隔离容量、淘汰策略和故障。Redis 承载会话及可选队列，应配置持久化、容量监控和适当的恢复策略；保存队列的实例使用 `noeviction`。需要资源隔离时，将队列配置到独立实例。

当前连接配置面向单机 Redis，不支持 Cluster，也未内置 Redis Sentinel 或 MySQL/Redis 专用 TLS 配置；目标服务需要这些能力时，应先扩展连接配置和测试。

Asynqmon/Asynq CLI 可单独部署，连接相同的队列 Redis。监控工具不继承本项目管理员认证，应置于受控网络或认证代理后。

## 维护

发布前运行检查及 [集成测试](development.md#测试与验证)，备份数据库、审查迁移，再显式执行 `migrate up`。应用启动不会自动迁移。

| 命令 | 用途 |
| --- | --- |
| `migrate version` | 查看迁移版本和 dirty 状态 |
| `migrate down --yes` | 回滚一步 |
| `migrate down --all --yes` | 回滚全部 |
| `migrate force VERSION --yes` | 手工修复后校正版本标记，不执行 SQL |

MySQL DDL 失败可能留下部分表结构，先检查并修复实际结构，再决定是否 force；迁移回滚不能替代数据备份。

设置缓存没有 TTL。设置写入提交后会清理缓存；若清理失败，接口返回错误但数据库提交仍保留。修复 Redis 后执行：

```sh
./app -c /etc/my-app/config.yaml settings:clear-cache
```

该命令只需要 Redis，不需要 MySQL。直接修改设置表也应清理缓存。管理员密码、状态等通过应用接口修改，以执行会话撤销；直接改数据库不会自动撤销会话。

使用 `/health/live` 检查存活，`/health/ready` 检查 MySQL/Redis；另外监控日志、Redis 容量和队列积压。登录日志的保留和清理周期由具体项目决定。
