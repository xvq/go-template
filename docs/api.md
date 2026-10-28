# API 文档

管理接口前缀为 `/api/admin`，除登录外均需请求头 `Authorization: Bearer <token>`。所有管理员权限一致。带请求体的接口使用 JSON，请设置 `Content-Type: application/json`。

## 响应约定

成功响应默认 HTTP 200，创建管理员为 HTTP 201：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {},
  "request_id": "..."
}
```

无返回数据的操作仍返回响应信封，`data` 为 null。时间输出为 UTC RFC3339，例如 `2026-10-28T03:00:00Z`。

校验失败返回 HTTP 422，`errors` 的键为请求字段名：

```json
{
  "code": "validation_failed",
  "message": "参数校验失败",
  "data": null,
  "errors": {"username": "账号为必填字段"},
  "request_id": "..."
}
```

| HTTP 状态 | code | 含义 |
| --- | --- | --- |
| 400 | `http_error` | 请求格式或 ID 无效 |
| 400 | `invalid_pagination` | 分页参数无效 |
| 401 | `unauthorized` | 未登录、会话失效或登录凭证错误 |
| 404 | `not_found` | 业务记录不存在 |
| 409 | `username_exists` / `last_admin` | 用户名重复 / 最后一个启用管理员保护 |
| 422 | `validation_failed` / `invalid_setting` | 字段校验失败 / 设置项或值无效 |
| 429 | `rate_limited` | 登录请求过于频繁 |
| 500 | `internal_error` | 内部故障，不返回底层错误详情 |
| 503 | `internal_error` / `canceled` | 健康检查未就绪 / 请求取消 |
| 504 | `timeout` | 处理超时 |

### 分页

管理员列表和登录日志列表接受 URL 参数 `page`（默认 1，范围 1～1000000）及 `page_size`（默认 20，范围 1～100）。必须为整数；未传或空值使用默认值。当前没有其他列表筛选参数。

分页结果位于响应的 `data` 中：

```json
{
  "data": [],
  "pagination": {
    "current_page": 1,
    "last_page": 1,
    "per_page": 20,
    "total": 0,
    "count": 0
  }
}
```

`total` 是总记录数，`count` 是本页条数；空数据的 `last_page` 为 1。超出末页返回空数组，保留请求页码和真实总数。

## 登录与当前身份

下表路径均相对于 `/api/admin`。

| 方法 | 路径 | JSON 请求体 | 返回 data |
| --- | --- | --- | --- |
| POST | `/login` | `username`、`password` | Token 信息和管理员对象 |
| POST | `/logout` | 无 | null |
| GET | `/me` | 无 | 管理员对象 |
| PUT | `/me/password` | `current_password`、`password` | null |

登录返回的 `data`：

```json
{
  "token": "<token>",
  "token_type": "Bearer",
  "expires_at": "2026-10-29T03:00:00Z",
  "admin": {
    "id": 1,
    "username": "admin",
    "display_name": "管理员",
    "enabled": true,
    "created_at": "2026-10-28T03:00:00Z",
    "updated_at": "2026-10-28T03:00:00Z"
  }
}
```

会话有效期固定，不自动续期，默认 24 小时。默认允许多会话；设置 `auth.admin.allow_multiple_sessions: false` 后，新登录会撤销旧会话。退出仅撤销当前 Token，本人改密后全部会话失效，需要重新登录。

登录按 IP 限流，默认每分钟 10 次，成功和失败请求均计数。未知账号、密码错误和账号禁用返回相同的凭证错误。

## 管理员

| 方法 | 路径 | JSON 请求体 / 说明 | 返回 data |
| --- | --- | --- | --- |
| GET | `/admins` | 分页参数 | 管理员分页列表 |
| POST | `/admins` | 必填 `username`、`password`；可选 `display_name`、`enabled` | 新管理员对象，HTTP 201 |
| GET | `/admins/:id` | 正整数 ID | 管理员对象 |
| PUT | `/admins/:id` | 必填 `display_name`、`enabled` | 更新后的管理员对象 |
| DELETE | `/admins/:id` | 删除并撤销全部会话 | null |
| PUT | `/admins/:id/password` | 必填 `password`，重置密码并撤销全部会话 | null |
| GET | `/login-logs` | 分页参数 | 登录日志分页列表 |

字段规则：

- `username`：3～64 位英文字母或数字，创建后不可修改，唯一性不区分大小写。
- 所有密码字段：6～30 个 Unicode 码点，允许空格和符号，不自动裁剪。
- `display_name`：最多 100 个 Unicode 码点；创建可省略，更新必须非空。
- `enabled`：创建时省略或 null 默认为 true；更新必须提供布尔值，false 有效。

更新管理员要求完整表单，不支持局部更新。不能删除或禁用最后一个启用的管理员。修改启用状态会撤销该管理员全部会话。

管理员对象字段为 `id`、`username`、`display_name`、`enabled`、`created_at`、`updated_at`，不包含密码哈希。

登录日志对象字段为 `id`、`admin_id`（可为 null）、`username`、`success`、`reason`、`ip`、`user_agent`、`created_at`。当前原因值为 `success`、`invalid_credentials`、`session_unavailable`。删除管理员不会删除历史日志；模板不自动清理日志。

## 系统设置

`GET /settings` 返回设置数组，每项包含 `key`、`type`、`value`、`default`：

```json
[
  {"key":"site.name","type":"string","value":"Go Template","default":"Go Template"},
  {"key":"site.maintenance","type":"boolean","value":false,"default":false}
]
```

`value` 是当前有效值，未保存时使用代码默认值；`default` 始终是代码默认值。`site.name` 不能仅包含空白，最多 100 个 Unicode 码点；`site.maintenance` 是示例开关，不自动拦截请求。

`PATCH /settings` 修改一个或多个已声明的设置，成功返回 `data: null`：

```json
{
  "values": {
    "site.name": "My site",
    "site.maintenance": false
  }
}
```

`values` 不能为空。仅允许代码中已声明的 key；值严格匹配声明类型（string、boolean、integer、string_array），不接受 null 或隐式转换。

管理员会话撤销、设置缓存失效均在相关数据库修改后执行。若 Redis 操作失败，接口可能返回错误但数据库修改已经提交；不要将错误响应理解为数据库必定回滚。设置缓存恢复方法见 [部署文档](deployment.md#维护)。

## 健康检查

无需认证：

| 方法与路径 | 检查内容 | 成功 data |
| --- | --- | --- |
| `GET /health/live` | HTTP 存活 | `{"status":"ok"}` |
| `GET /health/ready` | MySQL、Redis 可用 | `{"status":"ready"}` |

就绪检查不检查迁移版本、队列积压或外部业务依赖。
