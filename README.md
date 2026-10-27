# GoTemplate

Fiber + GORM + MySQL 的纯 API 项目模板。内置管理员、登录日志、系统设置、Redis 会话、定时任务和可选 Asynq 队列。一个二进制运行不同命令；不包含普通用户业务、前端或 RBAC。

## 环境

- Go **1.26+**（精确依赖版本锁定在 go.mod / go.sum）。
- MySQL **8.0+**，开发/CI 示例使用 8.4。
- Redis **7+**，开发/CI 示例使用 7.4；第一版不支持 Redis Cluster。
- 可在 Windows 开发，Linux 部署；不依赖 CGO。

## 快速开始

### 1. 从模板创建项目

安装 Go 官方工具仓库提供的 [gonew](https://pkg.go.dev/golang.org/x/tools/cmd/gonew)（首次使用时执行）：

```sh
go install golang.org/x/tools/cmd/gonew@latest
```

在新项目的父目录执行，将 `github.com/yourname/myapp` 换成自己的模块路径：

```sh
gonew github.com/xvq/go-template@main github.com/yourname/myapp
cd myapp
```

### 2. 初始化配置

```sh
go mod download
go run ./cmd config:init
```

编辑生成的 `config.yaml`：

- 设置 `app.name`，例如 `myapp`。
- 填写 MySQL 和 Redis 连接信息。

实际配置文件已被 Git 忽略。默认配置路径相对于**当前工作目录**；部署建议使用绝对路径。

本地开发可以使用已有 MySQL/Redis，也可以通过附带的 Compose 启动（需要 Docker）：

```sh
docker compose up -d --wait
```

这会创建 `go_template` 数据库和账号 `app`，配置文件中的数据库密码填 `local-app-password`。Compose 密码只用于本地开发；已有 MySQL/Redis 可直接使用，不必安装 Docker。

### 3. 初始化数据库并启动

```sh
go run ./cmd migrate up
go run ./cmd admin:create --username admin --name 管理员
go run ./cmd server
```

创建管理员时交互输入密码，不回显，也不通过命令行参数传密码。脚本可使用 `--password-stdin`。密码长度为 **6～30 个 Unicode 字符**，不限制内容、不自动裁剪空格。没有默认管理员。

```sh
curl http://127.0.0.1:8080/health/live
curl -X POST http://127.0.0.1:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"your-password"}'
```

后续管理请求携带 `Authorization: Bearer <token>`。

## 命令

```text
app [-c config.yaml] server
app [-c config.yaml] schedule
app [-c config.yaml] queue:work
app [-c config.yaml] migrate up
app [-c config.yaml] migrate down --yes       # 只回滚一步
app [-c config.yaml] migrate down --all --yes # 回滚全部
app [-c config.yaml] migrate version
app [-c config.yaml] migrate force VERSION --yes
app [-c config.yaml] admin:create --username admin
app [-c config.yaml] queue:example --delay 10s
app [-c config.yaml] settings:clear-cache
app [-c config.yaml] config:init
app hello
app version
app --help
```

`help`、`hello` 不加载配置或访问数据库；迁移、创建管理员不依赖 Redis。`force` 只修改迁移版本标记，必须先人工检查并修复实际表结构。MySQL DDL 不是整体事务，失败迁移可能留下部分表结构。

## 应用组织

```text
cmd/main.go              程序入口
internal/
  app/admin/             管理端 routes、handler、request、service、resource、middleware
  app/web/               网站端路由扩展入口，不预设用户业务
  core/                  框架层与全局资源入口
    core.go              全局 Config、DB、Redis、Session、Log、HTTP、Queue
    bootstrap/           公共资源初始化、服务生命周期；dependencies.go 集中声明三个框架进程的资源依赖
    config/              YAML 配置与按命令校验
    infrastructure/      DB、Redis、日志、HTTP 客户端、会话、队列
      cli/               根命令、服务启动、迁移与配置生成
      scheduler/         定时任务类型与调度运行机制
    middleware/          register.go 注册公共中间件；请求上下文、超时与访问日志
    route/               register.go 集中注册 admin、web 等应用路由
    request/             路由绑定校验包装、ID、分页与 Token 提取
    response/            响应信封、错误转换、流式输出
    resource/            单个可选模型、集合、分页的通用输出转换
    testutil/            全局资源测试辅助
  apperror/              公共错误定义
  service/settings/      系统设置能力；其他公共能力各建子目录
  util/
    str/                 无业务依赖的字符串工具
    query/               GORM 分页查询辅助与 Page[T] 查询结果
  model/                 GORM 数据模型与关联
  command/               业务命令及显式注册
  schedule/              具体定时任务及显式注册
  job/                   具体队列任务及 register.go 显式注册
migrations/              编译进程序的 SQL
integration/             独立测试数据库上的真实联调
docs/                    API、扩展方法、部署说明
```

各应用默认运行在同一个服务，管理 API 前缀为 `/api/admin`，网站端预留 `/api/web`。基础设施由 bootstrap 按命令通过 `Options{Name, Database, Redis, HTTP, Queue}` 明确选择并统一初始化，业务直接使用 `core.Config/DB/Redis/Session/Log/HTTP/Queue`，不逐层注入。运行期间不替换全局资源，退出由启动代码统一关闭。GORM 查询能力直接保留，不强制 Repository。

管理端路由通过 `request.JSON` 完成 JSON 参数绑定和 go-playground/validator 校验（公共包另提供 Body/Form/Query 用于其他请求来源），再调用 handler；handler 组织 `service → resource`，最后由 response 输出。应用业务放在 `app/<应用>/service`；邮件、支付、存储等可复用能力放在 `internal/service/<能力>`，便于按项目挑选。每种 resource 定义输出结构、私有 transform 方法和公共入口（如 AdminResource、AdminLoginLogResource），统一通过 .Item/.Collection/.Page 转换，也支持嵌套资源。登录等组合响应由 handler 直接拼装。

## 管理员与认证

- 管理员 CRUD、启用/禁用、重置密码、本人修改密码、登录、退出、当前身份。
- 所有管理员权限相同，没有角色/菜单/权限体系。
- 用户名创建后不可修改；更新管理员提交完整表单，显示名称不能为空，启用状态必填。
- 不允许删除/禁用最后一个启用的管理员，避免误操作锁死管理端。
- 密码使用 Argon2id；登录成功/失败记录账号、时间、IP、User-Agent 和原因，不记录密码或 Token。
- `core.Session` 随公共 Redis 初始化，复用同一连接池，无需在业务服务中创建。
- 每个会话使用独立 Redis 键和 TTL，Redis 只存 Token 哈希标识。
- `auth.admin.allow_multiple_sessions: false` 表示新登录撤销旧会话；通过 Lua 原子执行，并发下也只保留一个。
- 多会话切换为单会话后，在下一次成功登录时清理旧会话。配置修改需重启。
- 认证中间件只验证 Redis 会话和 admin 身份域，在 Locals 保存当前 ID，不查询 MySQL、不加载管理员模型。需要模型的接口自行查询。修改/重置密码、切换启用状态和删除账号后删除该账号的全部 Redis 会话，撤销失败返回错误，不静默返回成功。
- 会话有效期固定，不自动续期；退出只撤销当前会话。本人改密和重置密码使用事务行锁，旧密码验证与更新在同一把锁内完成。

## 配置、日志与设置

数据库日期时间统一保存 UTC，精确到秒（`DATETIME`）；API 输出如 `2026-10-23T03:11:03Z`。

配置以 `config.yaml` 为唯一运行配置来源，不自动读取 `.env` 或环境变量。支持未知字段检查、默认值和按命令验证。日志支持 `debug/info/warn/error`，输出可选 `stdout/file/both`，文件按大小轮转。日志不输出 Authorization、密码、SQL 参数或请求正文。

业务设置存入 MySQL `settings` 表，以 JSON 保留类型。Redis 使用永久 Hash `settings` 缓存整份设置，每个字段保存 JSON 编码的值；`settings:version` 保存永久版本号，防止旧查询覆盖更新后的缓存。设置通过无状态包函数 `settings.String/Bool/Int/Strings/Set/Update/List/ClearCache` 调用，无需实例或注入。定义集中在 `internal/service/settings/definitions.go`，现有 `site.name` 和 `site.maintenance` 仍为示例，后者不自动阻止请求。

缓存未命中时合并代码默认值和数据库记录；读取和保存都检查已声明的 key、类型及设置规则。读取缓存失败时回源 MySQL，没有初始化 Redis 时直接读写数据库。修改提交数据库后递增版本并删除整个 Hash，失效最多重试 3 次、总计最多 3 秒；失败返回错误，数据库提交不会回滚。列表公开返回 key、type、value、default，JSON 类型保持不变。

手动清理：`go run ./cmd settings:clear-cache`，只需要 Redis，不需要 MySQL。该命令和修改设置共用“递增版本 + 删除缓存”的失效逻辑。修改默认值或校验定义后，也可用它主动刷新缓存。

## 队列和定时任务

Asynq 内置但 `queue.enabled` 默认 `false`。关闭时不初始化队列客户端、连接或消费者；尝试入队会明确返回错误。

启用后：

```sh
go run ./cmd queue:work
go run ./cmd queue:example --delay 10s
```

`server` 不启动消费者，`queue:work` 是独立进程。队列 Redis 地址为空时复用 `redis` 的连接配置；填入地址后使用 `queue.redis`。不同项目使用独立 Redis DB 或实例区分会话、缓存和队列数据。 Redis 的逻辑 DB 不隔离内存淘汰策略；持久化和 noeviction 配置见部署文档。

提供立即/延迟任务、队列权重、重试次数、超时和并发配置。任务的重复执行保护、业务补偿、数据库与入队一致性由项目自己实现。超时通过 context 协作取消，不会强杀任意 Go 函数。

Asynqmon 与 Asynq CLI 按需单独安装，连接同一个队列 Redis，不嵌入本项目。

`schedule` 按单实例运行，支持异常恢复、跳过同一任务的重叠执行、执行超时和退出等待。内置独立的 `heartbeat.go` 示例，每分钟执行一次，仅记录日志；不需要时删除文件和注册项即可。每个任务独立文件，在 `internal/schedule/register.go` 显式注册；任务可调用服务或投递队列任务，见开发文档。

## 验证与构建

```sh
go test ./...
go vet ./...
go run ./scripts/build.go
```

脚本默认输出到 `bin/app`，Windows 输出为 `bin/app.exe`。通过参数进行 Linux 交叉编译，不需要修改当前终端的 GOOS：

```powershell
go run ./scripts/build.go -os linux -arch amd64
```

运行生成的程序加 version 可查看版本、完整提交号、UTC 构建时间和 Go 版本。脚本从当前项目 Git 历史读取 v 开头的版本标签：标签处为 v0.1.0，之后为 v0.1.0-3-gabcdef0；无标签为 dev-提交号。存在未提交修改（含未跟踪文件，不含忽略文件）时追加 -dirty。无 Git 信息的源码构建显示 dev / unknown；普通 go run ./cmd version 也使用开发默认值。脚本自动读取模块名。新项目可通过 `git init` 建立自己的版本历史。CI 获取完整历史并使用同一脚本构建，但不自动发布。

单位/组件测试使用 miniredis 与 sqlmock，不连接业务数据库。使用全局资源的测试按顺序执行，安装独立测试连接并在结束后恢复，不使用 `t.Parallel()`。真实迁移、CRUD、多表关联和队列消费使用单独的集成测试。启动 Compose 后（PowerShell）：

```powershell
$env:GOTEMPLATE_TEST_MYSQL_ADDRESS='127.0.0.1:3306'
$env:GOTEMPLATE_TEST_MYSQL_USER='root'
$env:GOTEMPLATE_TEST_MYSQL_PASSWORD='local-test-password'
$env:GOTEMPLATE_TEST_REDIS_ADDRESS='127.0.0.1:6379'
$env:GOTEMPLATE_TEST_REDIS_DB='15'
go test -tags=integration ./integration -count=1 -timeout=3m
```

这些环境变量仅用于测试，不参与应用配置。集成测试创建随机命名的专用 MySQL 数据库并在结束时删除，需要创建/删除数据库权限；Redis 使用独占的测试 DB（默认 15，可通过 GOTEMPLATE_TEST_REDIS_DB 指定正整数），运行前必须为空，结束后清理本次测试数据。不要并发使用同一个测试 Redis DB；仅指向测试服务。未设置地址时测试会跳过。GitHub CI 包含 Windows/Linux 单元测试、Linux race 检查及 MySQL/Redis 服务集成测试。

更多内容：[API 文档](docs/api.md)、[开发约定](docs/development.md)、[部署说明](docs/deployment.md)。
