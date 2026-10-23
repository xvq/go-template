# 开发约定

## 新增 API

1. 在 `internal/app/<应用>/validator/` 新建请求结构与 `Validate() error`，使用 ozzo-validation。
2. 在应用 `service/` 实现业务规则和事务，使用传入的 `*gorm.DB`、会话/设置服务等依赖，返回模型或业务结果。
3. 在应用 `resource/` 定义公开的 JSON 结构与模型转换函数。
4. 在 `handler/` 绑定、校验参数，调用 service，再通过 resource 转换，交给 response 输出。
5. 在应用 `routes.go` 显式注册；中间件必须放在最终 handler **之前**。
6. 增加必要的行为测试和接口文档。

JSON、Form 绑定调用 `validator.Body(c, &req)`。Query 可显式 `c.Bind().Query(&req)` 后调用 `req.Validate()`。不要用 GORM 模型接收请求，否则容易意外开放内部字段。

## 服务与资源组织

管理业务放在 `internal/app/admin/service`，网站业务放在各自应用目录。邮件、支付、短信、存储等复用能力分别放在 `internal/service/<能力>`；当前内置 `settings`。bootstrap 显式组装依赖，新增能力时不要求创建统一接口或注册中心。

业务服务不声明 JSON 输出契约。例如登录返回包含 Token、有效期和管理员模型的 `service.LoginResult`，handler 直接用 `fiber.Map` 拼装登录响应，其中管理员模型调用 `resource.AdminItem`。resource 专注数据库返回数据的字段转换，不为登录等组合响应单独定义资源。命令和任务也能直接使用业务服务。

管理员资源的调用方式：

```go
resource.AdminItem(admin)                // 单条模型
base.Collection(rows, resource.AdminItem) // 集合
base.Paginate(result, resource.AdminItem)     // 分页
```

这里 `resource` 是应用资源包，`base` 是 `internal/resource` 的导入别名。每种资源只需定义输出结构、编写一次单条转换函数；集合和分页直接调用公共泛型助手，不再编写各资源专用的 Collection/Paginate 包装函数。分页服务返回 `base.Page[T]{Items, Total, Page, PageSize}`，公共助手保留 `items/total/page/page_size` 四个 JSON 字段。总数由 service 查询，不能从模型切片自动推断；空集合统一输出 `[]`，越界页仍保留真实总数。

资源可以组合：关联字段使用其他资源类型，调用对应单条转换函数；嵌套列表使用 `base.Collection(rows, transform)`。可选关联可以使用 `base.Item(modelPointer, transform)`，nil 转为 null（字段带 omitempty 时省略）。Go 用有类型的字段组合，避免匿名嵌入造成 JSON 字段意外平铺。

resource 只转换已经加载的数据，不查询数据库、不触发懒加载；关联需要在 service 里用 Preload 等方式提前加载。公开字段显式列出，密码哈希、认证版本等内部字段不进入响应。

## 数据库

数据库日期时间统一使用 UTC、秒精度 `DATETIME`；Go 模型使用 `time.Time`，GORM 自动时间在写入前截断到秒，避免数据库舍入和创建响应携带小数。API 时间使用 UTC RFC3339 秒精度。Redis 会话的 `created_at`、`expires_at` 和索引分数统一使用 Unix 秒时间戳，使用 `EXAT` / `EXPIREAT` 按同一到期时刻清理会话与索引；会话有效期不足整秒的部分截去。耗时、超时等 duration 不受此约定影响。

`db.WithContext(ctx)` 传递超时/取消。service 决定事务边界：

```go
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 所有属于同一事务的操作都使用 tx
    return tx.Model(&record).Updates(map[string]any{"enabled": false}).Error
})
```

更新零值使用明确的 map/Select，不能假设结构体 Updates 会写入 false、0 或空字符串。区分 `gorm.ErrRecordNotFound`、重复约束和基础设施故障。批量列表必须分页并设置上限。

关联显式加载，例如 `Preload("Items.Product")`，不要在每条记录的循环里单独查关联。筛选主记录、聚合和关联排序按需使用 Join/子查询/SQL。响应对象不直接暴露密码等内部字段。

新增表结构写入下一个版本的 `.up.sql` / `.down.sql`。已发布迁移不修改。服务不调用 `AutoMigrate`；集成测试里的临时关联模型是特例。

## 会话与设置

会话服务接受身份域与 ID，禁止把普通用户 Token 当管理员 Token 使用。多会话开关在签发时应用；通过数据库 auth_version/status 处理管理员安全状态变更。新增用户业务应自行设计状态校验，不直接复用管理员模型。

设置在 `SettingDefinitions()` 声明。使用 `SettingsService.String/Bool` 获取明确类型，调用方处理读取错误。敏感字段在管理列表中省略 value；数据库内的设置值本身未加密。添加新类型时同时添加校验规则和对应类型读取方法。

## HTTP 客户端

`infrastructure/httpclient.New(config)` 返回独立 Resty 客户端，适合按上游设置 BaseURL、认证和超时。初始化完成后不要在并发请求中修改共享客户端配置；动态参数使用 `client.R()`。

```go
var result UpstreamResponse
resp, err := client.R().
    SetContext(ctx).
    SetHeader("Authorization", "Bearer "+key).
    SetBody(payload).
    SetResult(&result).
    Post("https://upstream.example/api")
if err != nil { return err }       // 网络/超时错误
if resp.IsError() { /* 处理 HTTP 4xx/5xx */ }
```

表单使用 SetFormData，文件上传使用 SetFile/SetMultipartField。默认没有重试，不自动记录请求正文和凭证。需要重试时在特定上游客户端显式配置并确认业务可以重复执行。

流式读取使用 `SetDoNotParseResponse(true)`，调用方负责关闭 `resp.RawBody()`。普通请求的 context 会在 handler 返回时取消，不能交给之后才开始读取的响应流。

`response.Stream` 为流创建独立、有限时长的 context，并在流关闭时取消上游。传入**应用生命周期 context**，open 回调使用该 context 发起上游请求、检查状态后返回 body。发生 HTTP 错误时先关闭上游 body，再返回错误。不要把 Fiber Ctx 放进后台 goroutine。

Fiber 基于 fasthttp：不要假定浏览器断开会像 net/http 一样立即取消请求 context。普通请求靠期限和服务退出控制；流在读写失败/关闭时清理，读不到数据的上游仍需依靠配置的超时。流式接口应单独设计最长运行时间，同时调整服务器/反向代理的写超时；默认 60 秒适合普通接口，并不意味着无限流。

## 队列

在 `internal/job/` 定义任务类型、JSON 参数和处理函数，在 `job.Register` 登记。服务接收 `*taskqueue.Client` 后调用：

```go
info, err := queue.Enqueue(ctx, "mail:send", payload,
    asynq.ProcessIn(10*time.Second),
    asynq.MaxRetry(3),
    asynq.Timeout(time.Minute),
)
```

返回 nil 表示处理成功；返回错误触发重试；不可恢复的参数错误可以包装 `asynq.SkipRetry`。消费者收到的 context 必须传给 DB/HTTP 操作，并在长循环中检查取消。超时不会强杀不响应 context 的代码。

模板只负责基础设施，不实现业务幂等、事务 outbox 或补偿。任务可能重复执行，项目按业务需要处理。Asynq 存储键不使用 `redis.prefix`，跨项目必须隔离队列 DB/实例。

任务需要数据库等依赖时，在 `internal/infrastructure/cli/queue_work.go` 的 `bootstrap.Open` 增加所需组件，并传给 `job.Register`。默认示例只需日志，因此 queue:work 不要求 MySQL。

## 定时任务

`internal/schedule/register.go` 统一显式注册任务，`internal/infrastructure/scheduler/task.go` 定义 `Name / Spec / Timeout / Run(ctx)`，同目录的 `runner.go` 负责调度和退出。内置 `heartbeat.go` 示例：每分钟执行一次、超时 5 秒，注入日志依赖并记录 debug 心跳；调度器还会记录 info 级别的执行完成日志。不需要时删除 `heartbeat.go` 及 `Register` 中的 `newHeartbeat(log)` 注册项即可。空注册表下进程保持运行，等待退出信号。

新增任务时在 `internal/schedule/` 创建独立文件，提供返回 `scheduler.Task` 的构造函数（例如 `newCleanup(...) scheduler.Task`），然后在 `Register` 的列表中添加 `newCleanup(...)`。具体执行逻辑留在任务自己的文件里，不使用 `init()` 自动注册。需要服务依赖时给 `Register` 增加明确的参数，由 `internal/infrastructure/cli/schedule.go` 组装并传入。

cron 使用五段表达式（分钟粒度），也支持 `@every 30s`。同一进程内同一任务上次未结束时跳过本次，异常被记录；不存在跨进程分布式锁，因此只运行一个 schedule 进程。

需要投递任务时，让 schedule 命令按配置启用 queue 组件，并把 `r.Queue` 传入任务构造函数；需要数据库时同理增加 database 组件。初始化失败要返回错误，不要静默跳过。

## 自定义命令

在 `internal/command/` 创建 Cobra 命令，并在 `internal/command/register.go` 的 `Register` 显式登记。参考 `hello`、`admin:create`、`queue:example`。只加载需要的资源，使用 `cmd.Context()`、返回 error，禁止从业务代码直接 os.Exit。预期错误不用 panic。

`internal/infrastructure/cli/root.go` 负责根命令、公共参数和内置命令注册，并通过 `command.Register(&path)` 加入业务命令。cli 目录保留 server、queue:work、schedule、migrate、config:init、version；业务目录 `internal/command` 保留 admin:create、hello、queue:example 及注册文件。每个命令独立文件，migrate 的 up/down/version/force 子命令放在同一个文件。

新增命令时在自己的文件提供 `newXxx() *cobra.Command` 构造函数，再加入 `command.Register` 返回的列表。需要配置路径时接收 `path *string`，在 RunE 执行时读取 `*path`，以获得解析后的 `--config` 参数。构造函数只定义命令及参数，资源初始化和清理留在命令执行逻辑中；不使用 `init()` 自动注册。仅该命令使用的辅助函数也留在对应文件，例如管理员密码读取。

依赖方向：cli 引用 command 和 schedule；command 不引用 cli。schedule 引用 scheduler 的任务类型；scheduler 不引用具体任务目录。

## 测试

HTTP 未识别的内部错误和显式 5xx 错误向前端返回通用提示；服务端记录 request_id、方法、路径、错误类型和脱敏后的 error。panic 额外记录当前 goroutine 的函数名、源文件和行号，不记录函数参数；普通错误不采集堆栈。业务错误按固定公开提示返回。

错误文本会过滤常见凭证格式、URL、带引号的值及已配置的数据库/Redis 密码，并限制长度。这是防御措施，不能识别任意业务秘密：不要将请求正文、完整 SQL、Token 或第三方响应正文拼入 error。请求恢复不覆盖自行启动的 goroutine，也无法改写已经发送的流式响应。

- 默认测试不连接真实外部服务，使用 sqlmock 与 miniredis 验证错误路径和 Redis Lua 行为。
- `integration` 构建标签提供真实 MySQL/Redis 验证：迁移上下行、管理 API、密码撤销、设置缓存、四表嵌套预加载、实际队列消费。
- Mock 测试不能证明 MySQL 事务隔离或真实 Redis 的全部语义，发布前执行 CI 集成测试。
- 不对生产数据库执行测试；所有自动建表只允许出现在随机创建的测试库。


版本查询不加载配置或服务。通过 go run ./scripts/build.go 构建时，脚本读取 Git 元数据，通过 -ldflags -X 注入 cli/version.go 的字符串变量；不手工修改这些默认变量，不在运行时修改它们。发布构建应使用干净工作区及完整 Git 标签历史。
