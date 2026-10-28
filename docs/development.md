# 开发约定

## 新增 API

1. 在 `internal/app/<应用>/request/` 新建请求结构，使用 go-playground/validator 的 `validate` 标签和中文显示名 `label`。
2. 在应用 `service/` 实现业务规则和事务，直接使用 `core.DB/Redis/Session/Config/Log/HTTP/Queue`，返回模型或业务结果。
3. 在应用 `resource/` 定义公开的 JSON 结构与模型转换函数。
4. 在 `handler/` 接收已校验的请求指针，调用 service，再通过 resource 转换，交给 response 输出。
5. 在应用 `routes.go` 显式注册，用公共 request 包装有参数的 handler；中间件必须放在包装后的最终 handler **之前**。
6. 更新接口文档；重要业务规则和故障修复按需补自动测试，不要求简单 CRUD 都增加测试。

管理端请求体统一使用 JSON：请求结构只写 json 标签，路由使用 request.JSON。特殊接口按实际来源使用 form/query 标签以及对应包装器。json 是 JSON 编解码器识别的标签，form/query 是 Fiber 绑定器识别的标签，标签内容不是我们自定义的解析协议。

公共 `internal/core/request` 提供 `Body/JSON/Form/Query` 泛型路由包装器。Body 根据 Content-Type 选择绑定方式，JSON/Form/Query 明确指定来源；每次请求创建独立参数结构，绑定或校验失败不会进入 handler。保持原生 fiber.Ctx，认证身份仍由 Auth 中间件通过 Locals 保存。

```go
// routes.go：request 是公共包，adminrequest 是应用请求包。
api.Post("/login", middleware.LoginLimit(), request.JSON(h.Login))

// handler：不再手动绑定或调用校验器。
func (h *Admin) Login(c fiber.Ctx, req *adminrequest.Login) error {
    // 使用 req.Username、req.Password 调用 service。
    return nil
}

// app/admin/request
type Login struct {
    Username string `json:"username" label:"账号" validate:"required,alphanum,min=3,max=64"`
    Password string `json:"password" label:"密码" validate:"required,min=6,max=30"`
}
```

不要用 GORM 模型接收请求，否则容易意外开放内部字段。格式错误返回 HTTP 400；规则失败返回 HTTP 422、validation_failed，errors 的键保持 JSON 字段名，label 只用于中文提示。当前注册简体中文翻译，并未实现按请求切换语言。

公共校验器位于 `internal/core/infrastructure/validator`，在 NewServer 和 admin:create 执行时显式调用 Init，一次注册后只读使用；其他直接校验请求的命令也应先 Init，再调用 validator.Struct(&req)。帮助命令不初始化。仅使用内置校验规则，不注册自定义规则。用户名使用 required,alphanum,min=3,max=64；所有密码字段统一使用 required,min=6,max=30，长度按 Unicode 码点计数，内容不限，不自动裁剪空格。

密码直接使用 github.com/alexedwards/argon2id 的 CreateHash 和 ComparePasswordAndHash。参数统一定义在 `internal/core/infrastructure/password/params.go`，通过 `&passwordhash.Params` 使用（passwordhash 为该包的导入别名），运行期间视为只读，不增加 YAML 配置。使用固定参数（19 MiB 内存、2 次迭代、并行度 1、16 字节随机盐、32 字节结果），盐和参数随哈希一起存储；新建、登录、改密、重置和不存在账号的占位哈希均使用 Argon2id，不保留旧算法兼容。部署时按登录并发与资源预算评估参数。

管理员更新采用完整表单：display_name 使用 required,max=100，enabled 使用 *bool 加 required，拒绝缺省/null，同时允许 false。校验后服务接收具体 string/bool，并明确更新两个字段，不再区分局部更新字段。复杂校验可注册自定义规则；数据库唯一性、权限和事务等业务规则仍放在 service。

ID、分页、Bearer Token 通过 request.ID(c)、request.Pagination(c)、request.Token(c) 在需要时读取，不使用全局参数中间件或自定义 Context。request.Token 仅提取 Authorization 请求头中的 Bearer Token，缺失或格式错误返回空字符串；会话和账号有效性仍由 Auth 中间件及 service 验证。

### 时间输入与统一时区

`app.timezone` 是业务日期范围等计算与 cron 调度的统一时区，默认 `Asia/Hong_Kong`；使用明确的 IANA 名称或 `UTC`，不使用系统相关的 `Local`。配置执行校验会检查该值，help 不加载配置；数据库会话和写入、API 时间输出继续使用 UTC，不改变进程的 `time.Local`。

应用请求中的时间字段直接使用标准 `time.Time`（可选字段使用 `*time.Time`）。JSON、Query、Form 及 Body 包装器利用标准类型的 UnmarshalText/UnmarshalJSON 原生解析 RFC3339，再执行 validator，无需自定义类型、转换方法或扫描结构体。

只接受带明确偏移或 Z 的 RFC3339，例如 `2026-10-28T00:00:00+08:00`、`2026-10-27T16:00:00Z`，允许小数秒。不接受 `2026-10-28 00:00:00` 或纯日期，不按 `app.timezone` 补全缺失时区；客户端负责选择时区并明确表达时间点。

绑定保留输入的时间偏移，不额外归一化；标准时间比较按实际时刻进行，不受表示偏移影响。非法格式或日期返回 HTTP 400，必填或时间顺序不符返回 HTTP 422。未传的可选字段为 nil，JSON null 同样表示 nil；空字符串不是未传，返回 400。通过 URL 查询或表单传入偏移时，用标准参数编码工具编码，避免 `+` 被解释为空格。MySQL 驱动使用现有连接的 UTC location 编码时间参数，故带偏移的标准时间可以直接传给查询工具；资源输出仍显式遵守 UTC 约定。

```go
// 应用 request：示例要求起止时间都提供，并校验结束晚于开始。
type ListLoginLogs struct {
    Start *time.Time `json:"start" query:"start" label:"开始时间" validate:"required"`
    End   *time.Time `json:"end" query:"end" label:"结束时间" validate:"required,gtfield=Start"`
}

// handler：直接传递标准 *time.Time，不需要类型转换。
filter := service.LoginLogFilter{
    Start: req.Start,
    End:   req.End,
}

// service 只接收标准时间，不依赖 request 包。
type LoginLogFilter struct {
    Start *time.Time
    End   *time.Time
}

db.Scopes(
    query.GreaterOrEqual("created_at", filter.Start),
    query.LessThan("created_at", filter.End),
)
```

筛选使用包含开始、不包含结束的范围。输入绑定不自动计算今日/本周/本月等业务范围，也不自动为现有 API 增加时间筛选参数。

## 公共资源与生命周期

应用路由统一在 `internal/core/route/register.go` 的 `Register(app)` 中显式登记，当前依次注册 admin、web。新增应用时在这里添加注册调用；具体服务和 handler 的组装仍留在应用自己的 `routes.go`。bootstrap 在全局中间件与健康检查注册后调用该入口，并传播注册错误。

公共中间件统一在 `internal/core/middleware/register.go` 的 `Register(app, ctx, onError)` 中显式注册，顺序为请求生命周期、panic 恢复、按配置启用的 CORS，bootstrap 在注册任何路由前调用该入口。请求生命周期实现位于同目录的 `request_lifecycle.go`，负责 context、超时、请求 ID 和访问日志；管理员认证和登录限流位于 `internal/app/admin/middleware`，继续在管理员路由中按需使用。

框架代码统一收在 `internal/core`：`bootstrap` 管理启动与清理，`config` 管理配置，`infrastructure` 提供底层实现，`testutil` 提供测试辅助，`request/response/resource` 提供公共 HTTP 请求、响应和资源转换能力；根包 `core` 保留全局资源入口。子目录各自是独立 Go 包，不会自动引用父包。底层工厂和配置包不反向依赖 core，避免循环引用。

基础设施统一从 `internal/core` 访问：

```go
core.DB.WithContext(ctx).First(&record, id)
core.Redis.Get(ctx, key)
core.Session.Validate(ctx, token)
core.Log.InfoContext(ctx, "任务完成")
core.HTTP.R().SetContext(ctx).Get(url)
core.Queue.Enqueue(ctx, taskType, payload)
core.Config.App.Name
```

`bootstrap.Open(ctx, config, bootstrap.Options{...})` 在命令执行时显式初始化。`Name` 是日志名称；`Database/Redis/HTTP/Queue` 为 bool 字段，只有设为 true 才校验并初始化对应资源，省略默认 false。Config、Log 总是准备。未请求的 DB/Redis/HTTP 为 nil，不能使用；未请求的 Queue 为禁用客户端，投递返回 `taskqueue.ErrDisabled`，即使配置启用了队列也不会自动连接。

资源管理不承担 server/schedule 的纯配置校验：server 命令启动处调用 `c.Validate("server")`，schedule 命令启动处调用 `c.Validate("schedule")`，再初始化所需资源。帮助、hello、version 不调用 Open，不依赖配置或外部服务。

Open 成功后才一次发布全局资源；失败关闭已创建的资源。一个进程同时只允许一组活动资源，避免覆盖和泄漏。不在 `init()` 或首次业务调用时隐式连接。底层客户端工厂保留显式创建参数，由启动代码调用；业务层不传递基础设施依赖。

返回的 `Resources` 只负责清理，命令继续使用 `defer r.Close()`。先停止 HTTP 请求、调度任务或队列消费者，再关闭公共连接。Close 可重复调用，关闭后清空全局入口。业务不自行 Close、不替换全局变量；Config 含 map/slice，读取后也不能修改其内部集合。禁止把查询链或事务 tx 写回 `core.DB`，请求状态只放局部变量或 context。

## 服务与资源组织

不依赖全局基础设施或业务规则的公共工具按用途放在 `internal/util/<用途>`。字符串工具使用 `str`，例如 `str.Truncate(value, n)` 按 Unicode 码点保留前 n 位，不追加省略号；n 小于等于零返回空字符串。它不按字节截断，也不保证组合字符或 emoji 序列完整。只提取已有的通用能力，不建立混放所有函数的大 common/utils 包。

管理业务放在 `internal/app/admin/service`，网站业务放在各自应用目录。邮件、支付、短信、存储等复用能力分别放在 `internal/service/<能力>`；当前内置 `settings`。bootstrap 只负责基础设施与服务生命周期，不集中组装具体业务服务。具体业务对象按需在应用路由中创建，新增业务不需要修改 bootstrap，也不要求统一接口或注册中心。

业务服务不声明 JSON 输出契约。例如登录返回包含 Token、有效期和管理员模型的 `service.LoginResult`，handler 直接用 `fiber.Map` 拼装登录响应，其中管理员模型调用 `resource.AdminResource.Item(&out.Admin)`。resource 专注数据库返回数据的字段转换，不为登录等组合响应单独定义资源。命令和任务也能直接使用业务服务。

管理员资源按表名单数命名，例如 admins 对应 admin.go / AdminResource，admin_login_logs 对应 admin_login_log.go / AdminLoginLogResource，settings 对应 setting.go / SettingResource。输出类型使用私有名称，例如 admin、adminLoginLog、setting，仅公开资源入口。字段仍保持导出供 JSON 编码，同包资源可以直接引用私有类型组成嵌套结构。

```go
resource.AdminResource.Item(&admin)      // 单条或可选模型，nil 输出 null
resource.AdminResource.Collection(rows) // 集合
resource.AdminResource.Page(result)     // 分页
```

这里 resource 是应用资源包。公共 internal/core/resource 定义泛型 Resource[T,R] 以及 New 构造函数。业务资源只定义输出结构和私有空结构体的 transform 方法，再声明只读的公共资源入口：

```go
type adminResource struct{}

func (adminResource) transform(value model.Admin) admin {
    return admin{ID: value.ID, Username: value.Username}
}

var AdminResource = base.New(adminResource{}.transform)
```

base 是 internal/core/resource 的导入别名。Item/Collection/Page 统一由泛型 Resource 提供，业务不重复包装，也不公开 transform。共享资源只保存无状态转换函数，初始化后不得替换或修改，不保存请求、模型、数据库连接或分页结果。

分页服务返回 query.Page[T]{Data, Pagination}（位于 internal/util/query），对应 {data: [], pagination: {current_page, last_page, per_page, total, count}}。分页元数据在查询层计算完成；AdminResource.Page(result) 仅转换 Data、原样保留 Pagination，返回同样的 query.Page[R] 结构，放在统一响应的 data 内。空集合统一输出 []，越界页保留请求页码和真实总数。last_page 向上取整，空数据时为 1；count 是当前页实际条数。

资源可以组合：在 transform 内调用其他资源的 Item(modelPointer) 或 Collection(rows)。例如登录日志关联管理员时可调用 AdminResource.Item(log.Admin)，nil 转为 null（输出字段带 omitempty 时省略）。关联数据必须由 service 提前加载，转换过程不执行查询；不增加尚未需要的模型关联。避免双向资源转换无限递归，必要时分成简要资源和详情资源。

resource 只转换已经加载的数据，不查询数据库、不触发懒加载；关联需要在 service 里用 Preload 等方式提前加载。公开字段显式列出，密码哈希等内部字段不进入响应。

## 可选筛选条件

`internal/util/query/filter.go` 提供可组合的原生 GORM Scope，只追加条件、不执行查询、不依赖 HTTP 或 core。字段名由 service 在代码中指定，支持 `table.column` 或表别名前缀，按标识符引用；不接受 SQL 表达式或前端指定的任意字段。条件值使用参数绑定。

| 工具 | 条件 | 跳过规则 |
| --- | --- | --- |
| Search(keyword, columns...) | 多个文本字段的 OR 子串搜索，整组加括号 | 关键词 TrimSpace 后为空，或未提供字段 |
| Equal(column, value) | 等于 | 指针为 nil |
| In(column, values) | 集合成员匹配 | 切片为 nil 或长度为 0 |
| GreaterThan(column, value) | 大于 | 指针为 nil |
| GreaterOrEqual(column, value) | 大于等于 | 指针为 nil |
| LessThan(column, value) | 小于 | 指针为 nil |
| LessOrEqual(column, value) | 小于等于 | 指针为 nil |

Equal 和比较工具接受类型化指针，保留 0、false、空字符串等已传值。In 接受类型化切片，空集合表示不启用这个可选筛选，不应用于必须限制访问范围的权限集合。Search 将 `%`、`_` 和转义字符 `!` 当作普通输入，使用显式 `ESCAPE '!'`，不依赖 MySQL 的反斜杠转义模式；搜索大小写行为由字段排序规则决定。

例如 service 定义自己的筛选参数，再交给 Scopes：

```go
type AdminFilter struct {
    Keyword string
    Enabled *bool
    IDs     []uint64
    MinID   *uint64
    MaxID   *uint64
}

db := core.DB.WithContext(ctx).Model(&model.Admin{}).
    Scopes(
        query.Search(filter.Keyword, "admins.username", "admins.display_name"),
        query.Equal("admins.enabled", filter.Enabled),
        query.In("admins.id", filter.IDs),
        query.GreaterOrEqual("admins.id", filter.MinID),
        query.LessThan("admins.id", filter.MaxID),
    ).Order("admins.id DESC")

return query.Paginate[model.Admin](db, page, size)
```

HTTP request 使用指针区分未传与零值，由匹配来源的 request 包装器绑定、校验；handler 转成 service 的筛选参数。工具不解析日期、不决定时区，也不增加排序或关联查询约定；时间比较可传入已处理好的 `*time.Time`。现有管理员列表接口的参数和行为不因新增工具自动改变。

## 分页查询

普通分页调用 `internal/util/query` 的 `Paginate[T]`，统一执行 Count、Limit/Offset 查询并返回 `query.Page[T]`。例如 service：

```go
func (s *AdminService) List(ctx context.Context, page, size int) (query.Page[model.Admin], error) {
    return query.Paginate[model.Admin](
        core.DB.WithContext(ctx).Model(&model.Admin{}).Order("id DESC"),
        page, size,
    )
}
```

handler 使用 `resource.AdminResource.Page(result)` 转换，resource 不查询数据库。函数接收已经组装好的 GORM 查询，继续支持 Where、Scopes、Select、Preload 等原生能力，不依赖 core 或 HTTP 请求。调用前设置 Model（或 Table）和稳定排序。query.Paginate 统一校验 page 范围 1～1000000、size 范围 1～100，非法参数在查询前返回 apperror.ErrInvalidPagination。HTTP 入口的 request.Pagination 负责默认值（1/20）、字符串转整数和相同的范围校验，非法参数立即返回同一个错误；查询层保留校验，保护 CLI 等非 HTTP 调用。错误经 service/handler 原样传递，由 response.ErrorHandler 统一转换为 HTTP 400、code: invalid_pagination。CLI 等调用方也可以通过 errors.Is 识别该错误。query.NewPage 只包装自定义查询结果，不执行查询或校验参数。

计数与取数分别创建 GORM 会话，保留传入查询的条件和 context，不修改原查询；分页函数负责 Limit/Offset，计数时移除已有分页条件。不要在 Scopes 中再次设置分页条件。数据库错误原样返回，空页保留实际总数。两次查询默认不保证并发修改下的同一快照；需要时由调用方提供合适隔离级别的事务查询。

计数沿用 GORM Count 的语义。关联重复行、分组聚合等需要特殊计数时，在 service 显式查询后调用 `query.NewPage(rows, total, page, size)` 整理数据和元数据，同样交给对应资源的 `Page` 方法输出，不强行通过通用函数处理。

## 数据库

数据库模型放在 `internal/model`，一个模型一个文件，按表名单数命名，例如 admin.go、admin_login_log.go、setting.go；所有文件继续属于 model 包。

数据库日期时间统一使用 UTC、秒精度 `DATETIME`；Go 模型使用 `time.Time`，GORM 自动时间在写入前截断到秒，避免数据库舍入和创建响应携带小数。API 时间使用 UTC RFC3339 秒精度。Redis 会话的 `created_at`、`expires_at` 和索引分数统一使用 Unix 秒时间戳，使用 `EXAT` / `EXPIREAT` 按同一到期时刻清理会话与索引；会话有效期不足整秒的部分截去。耗时、超时等 duration 不受此约定影响。

`core.DB.WithContext(ctx)` 传递超时/取消。service 决定事务边界：

```go
err := core.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    // 所有属于同一事务的操作都使用 tx
    return tx.Model(&record).Updates(map[string]any{"enabled": false}).Error
})
```

更新零值使用明确的 map/Select，不能假设结构体 Updates 会写入 false、0 或空字符串。区分 `gorm.ErrRecordNotFound`、重复约束和基础设施故障。批量列表必须分页并设置上限。

关联显式加载，例如 `Preload("Items.Product")`，不要在每条记录的循环里单独查关联。筛选主记录、聚合和关联排序按需使用 Join/子查询/SQL。响应对象不直接暴露密码等内部字段。

新增表结构写入下一个版本的 `.up.sql` / `.down.sql`。已发布迁移不修改。服务不调用 `AutoMigrate`；集成测试里的临时关联模型是特例。

## 会话与设置

`core.Session` 在 `Options.Redis: true` 且 Redis 初始化成功后自动创建，复用 `core.Redis` 连接池，无需额外的 Session 选项。未初始化 Redis 时为 nil；业务仅在需要会话功能时启用 Redis。Session 不持有独立连接，无需单独关闭，`Resources.Close()` 关闭 Redis 并清空 Session。管理员创建等纯数据库操作不依赖 Session。

会话服务接受身份域与 ID，不存储认证版本。Auth() 只验证 Redis 会话和 admin 身份域，再通过私有 Locals 键保存 ID；handler 通过 middleware.CurrentID(c) 读取，认证不查询数据库。/me 等需要模型的接口自行调用 service.Get。登录仍查询 MySQL 校验密码和 enabled，禁止把普通用户 Token 当管理员 Token 使用。

多会话开关在签发时应用；修改/重置密码、切换启用状态和删除账号后调用 RevokeAll 删除所有 Redis 会话。撤销错误必须向调用方返回，不能只记录日志后返回成功。数据库修改已提交时，Redis 撤销失败不会回滚数据库；数据库与 Redis 操作不保证跨存储原子性，并发登录与撤销存在竞态边界。直接修改数据库也不会自动撤销会话。这是纯 Redis 认证的取舍，不增加认证版本、补偿任务或数据库逐请求校验。

本人改密和重置密码在事务内锁定目标账号（SELECT FOR UPDATE）；本人改密在持锁状态下验证旧密码再更新，避免并发请求使用已经失效的旧密码覆盖新密码。新增用户业务应自行设计状态校验，不直接复用管理员模型。

设置定义集中在 `internal/service/settings/definitions.go`：SettingDefinition 包含 Key、Type、Default（JSON 原始值）和可选 Validate 函数。String/Bool/Int/Strings 分别读取 string、boolean、integer、string_array，返回 string/bool/int/[]string。类型严格校验，不接受字符串转数字/布尔值，数组不允许混入其他类型或 null；空数组是否允许由具体设置规则决定。不新增无关业务示例；现有 site.name、site.maintenance 保留。

内部 all 读取整份设置，List 返回按定义顺序排列的 Entry{Key,Type,Value,Default}。后台资源输出 key/type/value/default，不再有敏感值过滤机制。值和默认值均保留真实 JSON 类型，复制 RawMessage 后返回，避免调用方修改原数据。设置服务无请求状态，不缓存进程内数据，不需要创建或注入服务实例。

Redis Hash `settings` 永久保存完整快照，字段名为设置 key，每个字段的值采用 JSON 编码。单值 getter 先检查 key 是否已定义、getter 是否匹配定义类型，再通过 HGET 只取目标字段；命中后只检查目标值的 JSON 类型，不检查其他字段。List 使用 HGETALL 检查设置项是否齐全及所有值的 JSON 类型。两种缓存命中路径均不执行设置专用 Validate 规则，但仍拒绝 null、错误类型、非法 JSON 等值。目标字段缺失或类型无效，或列表快照不完整/类型无效时，读取整份 MySQL 数据，以代码默认值为基础用数据库记录覆盖，对默认值和数据库值执行完整类型与规则校验，再通过版本保护的 Lua 整份回填，不做单字段回填。默认值本身不会插入数据库，只有 Set/Update 才执行事务 upsert。Redis 不可用时回源数据库；没有初始化 Redis 时每次直接读取数据库。

`settings:version` 是无过期时间的字符串计数器。回源前读取版本（不存在视为 0），回填 Lua 在版本相同时一次替换整个 Hash，不设置过期时间。它防止旧回源结果重新进入共享缓存，不保证已经开始的读取立刻拿到最新值。List 检查到新增/删除定义导致快照不齐全时会重新加载，单值读取在目标字段缺失时才重建。

Set(ctx,key,value) 将 Go 值编码为 JSON 后调用 Update；Update 校验所有输入，再在 MySQL 事务中写入。提交成功后调用 ClearCache：Lua 原子递增版本并删除 Hash。ClearCache 需要已初始化 Redis，不查数据库，最多尝试 3 次、总期限 3 秒。Update 在数据库提交后脱离原请求的取消信号进行这次有界失效；最终失败返回错误，MySQL 已提交的修改不会自动回滚。不存在 TTL 自动修复机制，可修复 Redis 后执行手动清理命令。

`go run ./cmd settings:clear-cache` 在执行时通过 bootstrap.Options{Name:"settings:clear-cache",Redis:true} 初始化资源，再调用相同的 ClearCache。help 不读取配置，不要求 MySQL，也不添加 HTTP 清缓存接口或其他业务能力。

## HTTP 客户端

普通请求直接使用 `core.HTTP.R()`。需要独立上游配置时，`internal/core/infrastructure/httpclient` 的 `New(config)` 返回专用 Resty 客户端，适合设置 BaseURL、认证和超时；专用客户端由创建者负责关闭空闲连接。初始化完成后不要在并发请求中修改共享客户端配置；动态参数使用 `client.R()`。

```go
var result UpstreamResponse
resp, err := core.HTTP.R().
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

在 `internal/job/` 为每个任务创建独立文件，定义任务类型、JSON 参数和处理函数，在 `internal/job/register.go` 的 `Register()` 任务列表中显式登记。Register 返回 ServeMux 和列表长度，注册与启动提示的任务类型数来自同一份列表。示例处理函数放在 `example.go`；删除示例时同时移除对应注册项，保留注册入口。直接使用公共投递客户端：

```go
info, err := core.Queue.Enqueue(ctx, "mail:send", payload,
    asynq.ProcessIn(10*time.Second),
    asynq.MaxRetry(3),
    asynq.Timeout(time.Minute),
)
```

返回 nil 表示处理成功；返回错误触发重试；不可恢复的参数错误可以包装 `asynq.SkipRetry`。消费者收到的 context 必须传给 DB/HTTP 操作，并在长循环中检查取消。超时不会强杀不响应 context 的代码。

模板只负责基础设施，不实现业务幂等、事务 outbox 或补偿。任务可能重复执行，项目按业务需要处理。会话、限流、设置缓存和 Asynq 使用固定用途的键名，跨项目通过独立 Redis DB/实例隔离。

任务需要数据库等依赖时，在 `internal/core/bootstrap/dependencies.go` 的 `QueueWorkOptions()` 启用所需资源，任务直接使用 `core.DB` 等公共资源。默认示例只需日志，因此 queue:work 不要求 MySQL。

## 定时任务

`internal/schedule/register.go` 统一显式注册任务，`internal/core/infrastructure/scheduler/task.go` 定义 `Name / Spec / Timeout / Run(ctx)`，同目录的 `runner.go` 负责调度和退出。内置 `heartbeat.go` 示例：每分钟执行一次、超时 5 秒，使用 `core.Log` 记录 debug 心跳；调度器还会记录 info 级别的执行完成日志。不需要时删除 `heartbeat.go` 及 `Register` 中的 `newHeartbeat()` 注册项即可。空注册表下进程保持运行，等待退出信号。

新增任务时在 `internal/schedule/` 创建独立文件，提供返回 `scheduler.Task` 的构造函数（例如 `newCleanup(...) scheduler.Task`），然后在 `Register` 的列表中添加 `newCleanup(...)`。具体执行逻辑留在任务自己的文件里，不使用 `init()` 自动注册。任务通过 core 访问公共基础设施，无需给 Register 或任务构造函数增加 DB/Redis/Log 参数。

cron 使用五段表达式（分钟粒度），也支持 `@every 30s`，统一按 `app.timezone` 调度；schedule 配置仅保留退出等待等调度配置。同一进程内同一任务上次未结束时跳过本次，异常被记录；不存在跨进程分布式锁，因此只运行一个 schedule 进程。

需要投递任务时，在 `internal/core/bootstrap/dependencies.go` 的 `ScheduleOptions()` 设置 `Queue: true`，队列资源是否启用仍由 `queue.enabled` 决定，任务直接调用 `core.Queue.Enqueue`；需要数据库时同理设置 `Database: true`。初始化失败要返回错误，不要静默跳过。

## 自定义命令

在 `internal/command/` 创建 Cobra 命令，并在 `internal/command/register.go` 的 `Register` 显式登记。参考 `hello`、`admin:create`、`queue:example`。只加载需要的资源，使用 `cmd.Context()`、返回 error，禁止从业务代码直接 os.Exit。预期错误不用 panic。

`internal/core/infrastructure/cli/root.go` 负责根命令、公共参数和内置命令注册，并通过 `command.Register(&path)` 加入业务命令。cli 目录保留 server、queue:work、schedule、migrate、config:init、version；业务目录 `internal/command` 保留 admin:create、hello、queue:example 及注册文件。每个命令独立文件，migrate 的 up/down/version/force 子命令放在同一个文件。

新增命令时在自己的文件提供 `newXxx() *cobra.Command` 构造函数，再加入 `command.Register` 返回的列表。需要配置路径时接收 `path *string`，在 RunE 执行时读取 `*path`，以获得解析后的 `--config` 参数。构造函数只定义命令及参数，资源初始化和清理留在命令执行逻辑中；不使用 `init()` 自动注册。仅该命令使用的辅助函数也留在对应文件，例如管理员密码读取。

server、schedule、queue:work 三个框架进程的资源依赖集中在 `internal/core/bootstrap/dependencies.go`，分别由 `ServerOptions(c)`、`ScheduleOptions()`、`QueueWorkOptions()` 返回。ServerOptions 接收已加载的配置，根据 `c.Queue.Enabled` 选择 Queue；这些函数不读取配置文件、不连接资源，初始化和清理仍在原启动流程中执行。新增定时任务或队列任务需要额外资源时，修改对应函数。

三个常驻命令在启动成功后通过 `cmd.ErrOrStderr()` 打印单行英文提示，与日志配置独立，同时保留 info 启动日志。HTTP 使用实际绑定的监听地址；Scheduler 完成任务校验、注册并启动后通过回调通知命令层；Queue Worker 在 Asynq Start 成功后打印任务类型数和配置的最大并发数，并记录 `queue worker started` 日志。启动失败不打印成功提示，help 不启动资源或打印这些提示。bootstrap 和 scheduler 通过启动回调通知命令层，不直接持有 Cobra 命令。

业务命令的依赖继续在命令自己的执行代码中声明，不在 root.go 建立另一份清单。例如同时使用数据库和 Redis：

```go
c, err := config.Load(*path)
if err != nil { return err }
r, err := bootstrap.Open(cmd.Context(), c, bootstrap.Options{
    Name:     "my-command",
    Database: true,
    Redis:    true,
})
if err != nil { return err }
defer r.Close()
// 从这里开始使用 core.DB、core.Redis，无需从 r 取出再传给业务。
```

迁移命令仍通过 OpenMigrator 创建专用迁移连接（开启多语句执行），由命令自己关闭，不复用普通 GORM 连接。

依赖方向：cli 引用 command 和 schedule；command 不引用 cli。schedule 引用 scheduler 的任务类型；scheduler 不引用具体任务目录。

## 测试

HTTP 未识别的内部错误和显式 5xx 错误向前端返回通用提示；服务端记录 request_id、方法、路径、错误类型和脱敏后的 error。panic 额外记录当前 goroutine 的函数名、源文件和行号，不记录函数参数；普通错误不采集堆栈。业务错误按固定公开提示返回。

错误文本会过滤常见凭证格式、URL、带引号的值及已配置的数据库/Redis 密码，并限制长度。这是防御措施，不能识别任意业务秘密：不要将请求正文、完整 SQL、Token 或第三方响应正文拼入 error。请求恢复不覆盖自行启动的 goroutine，也无法改写已经发送的流式响应。

- 手工开发可以使用隔离的测试 MySQL/Redis 联调；自动测试优先保护认证、事务、缓存一致性和已修复的问题，不追求覆盖率或要求简单功能逐一补测试。
- 使用全局资源的测试必须串行，不调用 `t.Parallel()`；`internal/core/testutil.UseCore` 安装测试连接并在清理时恢复所有全局变量。连接由测试创建者关闭；后台请求/任务必须先停止。不同包的测试进程有各自的全局变量，真实 DB/Redis 数据仍需隔离。
- 默认测试不连接真实外部服务，使用 sqlmock 与 miniredis 验证错误路径和 Redis Lua 行为。
- `integration` 构建标签使用独占且初始为空的 Redis 测试 DB（默认 15，GOTEMPLATE_TEST_REDIS_DB 可指定其他正整数，禁止与其他服务并发共用），结束后清理该 DB 的测试数据；MySQL 使用随机临时库。提供真实 MySQL/Redis 验证：迁移上下行、管理 API、密码撤销、设置缓存、四表嵌套预加载、实际队列消费。
- Mock 测试不能证明 MySQL 事务隔离或真实 Redis 的全部语义，发布前执行 CI 集成测试。
- 不对生产数据库执行测试；所有自动建表只允许出现在随机创建的测试库。


版本查询不加载配置或服务。通过 go run ./scripts/build.go 构建时，脚本读取 Git 元数据，通过 -ldflags -X 注入 cli/version.go 的字符串变量；不手工修改这些默认变量，不在运行时修改它们。发布构建应使用干净工作区及完整 Git 标签历史。
