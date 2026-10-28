# 开发指南

本文说明项目的扩展位置和已有能力。目录职责见 [README](../README.md)，接口契约见 [API 文档](api.md)。修改前先查找已有实现，复用公共能力。

## 注册入口

以下路径均相对于 `internal/`。项目采用显式注册，不使用 `init()` 自动发现业务。

| 入口 | 负责什么 |
| --- | --- |
| `command/register.go` | 项目有哪些业务命令 |
| `job/register.go` | 消费哪些队列任务 |
| `schedule/register.go` | 运行哪些定时任务 |
| `core/route/register.go` | 挂载哪些应用路由 |
| `core/middleware/register.go` | 全局中间件及执行顺序 |
| `core/bootstrap/dependencies.go` | 各常驻进程需要哪些资源 |

具体应用在自己的 `routes.go` 中创建 service、handler 并注册接口。全局中间件依次为请求生命周期、panic 恢复、可选 CORS，之后注册健康检查和应用路由；应用中间件放在最终 handler 之前。

## 新增接口

1. 在 `app/<应用>/request` 定义请求结构，使用 `validate` 和 `label` 标签，不直接用数据库模型接收请求。
2. 在应用 `service` 中实现业务规则、查询与事务，返回模型或业务结果。
3. 在应用 `resource` 中选择输出字段、转换模型，不查询数据库。
4. 在 `handler` 中调用服务、转换结果并使用统一响应。
5. 在应用 `routes.go` 注册接口，更新 API 文档，为重要规则或故障修复补充测试。

管理端请求体统一使用 JSON，结构只写 `json` 标签，路由使用 `request.JSON`：

```go
// 应用 request 包
type UpdateAdmin struct {
    DisplayName string `json:"display_name" label:"名称" validate:"required,max=100"`
    Enabled     *bool  `json:"enabled" label:"启用状态" validate:"required"`
}

// 应用 routes.go；request 为 internal/core/request
api.Put("/admins/:id", request.JSON(h.Update))
```

包装器先绑定、校验，再调用签名为 `func(fiber.Ctx, *T) error` 的 handler。其他请求来源使用现有 `request.Query/Form/Body` 和对应标签。公共校验器在 HTTP 启动时初始化；自行执行校验的 CLI 命令先调用 `validator.Init()`。字段错误以 JSON 字段名为键，`label` 用于中文提示。

handler 按需使用 `request.ID`、`request.Pagination`、`request.Token`，将 `c.Context()` 传给 service。成功响应使用 `response.OK/Created`，错误直接返回给统一错误处理器；公共业务错误放在 `internal/apperror`。

### 资源转换

资源文件按表名单数命名，例如 `admin.go`、`admin_login_log.go`。输出类型和无状态转换类型保持私有，公开一个共享入口：

```go
// base 为 internal/core/resource
type admin struct {
    ID       uint64 `json:"id"`
    Username string `json:"username"`
}

type adminResource struct{}

func (adminResource) transform(value model.Admin) admin {
    return admin{ID: value.ID, Username: value.Username}
}

var AdminResource = base.New(adminResource{}.transform)
```

通过 `AdminResource.Item(&row)`、`.Collection(rows)`、`.Page(result)` 使用，不重复编写包装方法。`Item(nil)` 返回 nil，空集合输出 `[]`。嵌套资源调用其他资源入口，关联由 service 预加载，避免循环转换。登录等组合响应直接在 handler 中组织。

## 公共资源与已有工具

基础设施通过 `core.Config/DB/Redis/Session/Log/HTTP/Queue` 使用，不通过业务构造函数逐层注入。bootstrap 按命令初始化资源，运行期间视为只读；请求状态、查询链和事务留在局部变量中。业务不得关闭共享连接。

| 能力 | 位置 / 用法 |
| --- | --- |
| 请求绑定与参数 | `core/request`：JSON/Form/Query/Body、ID、Pagination、Token |
| 统一响应与错误 | `core/response`：OK、Created、ErrorHandler |
| 资源转换 | `core/resource`：New、Item、Collection、Page |
| 可选查询条件 | `util/query/filter.go`：原生 GORM Scopes |
| 分页查询 | `util/query/pagination.go`：Paginate、Page[T]、NewPage |
| 字符串截取 | `util/str`：Truncate，按 Unicode 码点截取 |
| HTTP 请求 | 按需启用后使用 `core.HTTP.R()`，基于 Resty |
| 会话 | `core.Session`：Issue、Validate、Revoke、RevokeAll |
| 系统设置 | `service/settings`：类型化读取、保存、列表与缓存清理 |

### 查询与分页

直接使用 GORM 的查询、事务和关联能力。可选条件复用 `query.Search/Equal/In/GreaterThan/GreaterOrEqual/LessThan/LessOrEqual`：

```go
// service 中的查询片段；keyword 为 string，enabled 为 *bool。
db := core.DB.WithContext(ctx).Model(&model.Admin{}).
    Scopes(
        query.Search(keyword, "username", "display_name"),
        query.Equal("enabled", enabled),
    ).Order("id DESC")
return query.Paginate[model.Admin](db, page, size)
```

指针为 nil 时跳过条件，false、0 等零值仍有效；空 `In` 切片跳过筛选，因此不能用于可能为空的权限白名单。`Search` 将多个字段组成 OR 条件并把 LIKE 通配符当作普通输入。列名由服务代码指定，不接受客户端任意列名或 SQL 表达式。

HTTP 层使用 `request.Pagination` 解析页码；service 返回 `query.Page[T]`，`query.Paginate` 负责范围校验、Count 和分页查询，调用前设置 Model/Table 与稳定排序。handler 使用 `Resource.Page` 仅转换数据、保留分页信息。复杂聚合需要自定义计数时，可自行查询后用 `query.NewPage` 组装结果，并负责校验分页参数。

模型统一放在 `internal/model`，每个表一个文件。关联在 service 中显式加载；更新 false、0、空字符串时使用明确的 map/Select。新增表结构添加版本化 SQL 迁移，不在服务启动时 AutoMigrate，不修改已发布迁移。

### 时间与请求生命周期

`app.timezone` 是业务计算和调度的统一时区，默认 `Asia/Shanghai`；数据库存储和 API 时间使用 UTC，不修改 `time.Local`。

请求时间使用标准 `time.Time` / `*time.Time`，通过原生绑定接受带偏移或 Z 的 RFC3339。客户端明确时区，服务直接接收标准时间，不套用应用时区或增加自定义转换类型。

将真实 `context.Context` 传入 DB、HTTP 和任务操作，不保留 Fiber Ctx。流式接口使用 `response.Stream`，传入应用生命周期 context 并设置有限时长；不要复用 handler 返回后已取消的请求 context。流式使用方法见 [stream.go](../internal/core/response/stream.go)。

共享 HTTP 客户端默认不初始化（`core.HTTP` 为 nil）；需要时在对应进程依赖或业务命令的 `bootstrap.Options` 中设置 `HTTP: true`。启用后用 `core.HTTP.R().SetContext(ctx)` 发起请求，不在运行时修改客户端配置；特定上游需要认证、重试等配置时，可通过 `infrastructure/httpclient.New` 创建专用客户端。

## 管理员与系统设置

管理员认证仅验证 Redis 会话及 admin 身份域，在 Locals 保存 ID；handler 用 `middleware.CurrentID` 获取，需要模型时再查询。密码使用 Argon2id，参数位于 `core/infrastructure/password/params.go`。

管理员更新为完整表单；不能删除或禁用最后一个启用的管理员。改密、重置密码、启用状态变化和删除账号会撤销全部会话。密码验证与更新在同一事务行锁内完成；数据库提交与 Redis 撤销不是分布式事务，撤销失败必须返回错误。

设置定义集中在 `internal/service/settings/definitions.go`，声明 key、类型、默认值和可选校验。使用包函数 `String/Bool/Int/Strings/Set/Update/List/ClearCache`，不创建服务实例。支持 `string/boolean/integer/string_array`，不做隐式类型转换；未保存时使用代码默认值。`site.name`、`site.maintenance` 是示例，后者不会自动阻止请求。

设置由 MySQL 保存，Redis 永久 Hash 缓存完整快照，版本号防止旧查询覆盖新缓存。写入后统一清理缓存，不自行写 Redis 或添加 TTL。缓存不可用时读取可回源 MySQL；提交后的失效失败会返回错误，恢复方法见部署文档。

## 命令、队列与定时任务

每个业务命令在 `internal/command` 中独立建文件并注册，参考 `admin_create.go`。仅在执行时读取配置和初始化资源，帮助命令不得依赖配置或外部服务：

```go
// 命令 RunE 中；path 为配置路径。
c, err := config.Load(*path)
if err != nil { return err }
r, err := bootstrap.Open(cmd.Context(), c, bootstrap.Options{
    Name: "my-command", Database: true, Redis: true,
})
if err != nil { return err }
defer r.Close()
// 使用 core.DB、core.Redis 等已选择的资源。
```

`Options` 包含 Name、Database、Redis、HTTP、Queue；Config 和 Log 总是准备，Session 随 Redis 自动准备并复用连接池。先停止请求和后台任务，再关闭公共资源。

常驻进程的依赖在 `bootstrap/dependencies.go` 中调整：`ServerOptions(c)`、`ScheduleOptions()`、`QueueWorkOptions()`。默认 server 使用 DB、Redis 并按配置选择 Queue，HTTP 客户端按需启用；心跳调度只需日志，示例消费者只需队列和日志。新增任务需要其他资源时，同时更新对应声明。

### 队列

每个任务在 `internal/job` 中独立定义任务类型、payload 和处理函数，并在 `register.go` 注册。参考 `example.go`，通过 `core.Queue.Enqueue(ctx, taskType, payload, options...)` 投递，支持 Asynq 的延迟、重试和超时选项。

队列默认禁用，禁用时入队返回 `taskqueue.ErrDisabled`。启用后单独运行 `queue:work`。任务可能重复执行，幂等和数据库与入队的一致性由具体项目实现；处理函数必须响应 context 取消。

### 定时任务

每个任务在 `internal/schedule` 独立建文件，返回 `scheduler.Task{Name, Spec, Timeout, Run}` 并注册。`heartbeat.go` 是可删除的示例，删除时同步移除注册项。

调度支持五段 cron 表达式和 `@every`，使用 `app.timezone`。同一进程内跳过重叠执行并恢复任务 panic；没有跨进程锁，只运行一个 schedule 实例。任务可以调用服务或投递队列，所需资源在 `ScheduleOptions()` 声明。

## 测试与验证

Go 代码修改后执行 gofmt、`go test ./...`、`go vet ./...`。测试重点是业务规则、认证、事务、缓存一致性和故障修复，不为简单 CRUD 追求覆盖率。

默认测试使用 sqlmock/miniredis，不连接真实服务。替换全局资源的测试使用 `core/testutil.UseCore`，串行执行，停止后台工作后再恢复，不使用 `t.Parallel()`。

真实联调使用可丢弃的 MySQL/Redis。启动根目录 Compose 后，可在 PowerShell 中运行：

```powershell
$env:GOTEMPLATE_TEST_MYSQL_ADDRESS='127.0.0.1:3306'
$env:GOTEMPLATE_TEST_MYSQL_USER='root'
$env:GOTEMPLATE_TEST_MYSQL_PASSWORD='local-test-password'
$env:GOTEMPLATE_TEST_REDIS_ADDRESS='127.0.0.1:6379'
$env:GOTEMPLATE_TEST_REDIS_DB='15'
go test -tags=integration ./integration -count=1 -timeout=3m
```

测试创建并删除随机 MySQL 数据库，需要建库/删库权限；Redis 必须是独占、初始为空的测试 DB（默认 15），结束后清理。禁止连接生产服务或并发使用同一个测试 DB。未配置服务地址的测试会跳过，跳过不代表验证通过。这些环境变量只用于测试。

CI 执行 Windows/Linux 单元测试、构建、vet、Linux race 检查和真实 MySQL/Redis 集成测试。
