# API 约定

管理接口前缀 `/api/admin`。除登录之外都要求 `Authorization: Bearer <token>`。管理员权限一致。认证只读取 Redis 会话并提供当前管理员 ID，需要管理员模型时由接口自行查询。带请求体的管理接口使用 JSON，请设置 `Content-Type: application/json`；列表分页参数仍通过 URL Query 传入。

## 响应

```json
{
  "code": "ok",
  "message": "ok",
  "data": {},
  "request_id": "..."
}
```

校验失败 HTTP 422，`code: validation_failed`，附加 `errors` 字段：

```json
{
  "code": "validation_failed",
  "message": "参数校验失败",
  "data": null,
  "errors": {"username": "账号为必填字段"},
  "request_id": "..."
}
```

400 格式/参数错误；401 未登录或凭证错误；404 不存在；409 用户名重复（code: username_exists）或最后管理员保护；429 登录限流；500 内部故障；503 服务未就绪；504 处理超时。500 响应不暴露底层数据库和 Redis 错误。

列表参数 `page` 默认 1、最大 1000000；`page_size` 默认 20、最大 100，均需为正整数。非整数或超出范围返回 HTTP 400、`code: invalid_pagination`。分页结果放在统一响应的 `data` 内，结构为 `data + pagination`：

```json
{
  "code": "ok",
  "message": "ok",
  "data": {
    "data": [],
    "pagination": {
      "current_page": 1,
      "last_page": 1,
      "per_page": 20,
      "total": 0,
      "count": 0
    }
  },
  "request_id": "..."
}
```

`current_page` 是请求页码，`last_page` 为总页数（向上取整，空数据时为 1），`per_page` 是每页条数，`total` 是总记录数，`count` 是当前页实际返回条数。越界页返回空数组，保留请求页码与真实总数/总页数。

## 管理接口

| 方法 | 路径 | 请求 / 说明 |
|---|---|---|
| POST | `/login` | `username`, `password`；返回 token、token_type、expires_at、admin |
| POST | `/logout` | 撤销当前 Token |
| GET | `/me` | 当前管理员 |
| PUT | `/me/password` | `current_password`, `password`；成功后全部会话失效 |
| GET | `/admins` | 分页列表 |
| POST | `/admins` | `username`, `password`, 可选 `display_name`, `enabled`（默认 true）；HTTP 201 |
| GET | `/admins/:id` | 管理员详情 |
| PUT | `/admins/:id` | 完整表单：`display_name`、`enabled` 都必填；名称不能为空；不允许修改 username |
| DELETE | `/admins/:id` | 删除管理员，撤销会话 |
| PUT | `/admins/:id/password` | `password`；重置密码，撤销其全部会话 |
| GET | `/login-logs` | 分页查询登录日志 |
| GET | `/settings` | 设置列表，包含 key、type、value、default，保留 JSON 类型 |
| PATCH | `/settings` | `{"values":{"site.name":"My site","site.maintenance":false}}` |

用户名是 3～64 位大小写英文字母或数字（不要求两者同时出现），按 MySQL 表的大小写不敏感排序规则判断唯一性。密码 6～30 个 **Unicode 字符**，允许中文、符号、空格，不自动裁剪；使用 Argon2id 存储。显示名称最多 100 个 Unicode 字符。管理员更新要求提交完整表单：display_name 不能为空，enabled 不可缺省或为 null，`enabled:false` 是有效值，会禁用账号。

登录按 IP 使用 Redis 计数限流，跨 HTTP 实例共享；默认每分钟 10 次（成功和失败请求都计数）。错误密码、未知账号、禁用账号返回相同提示。该基础策略不代替项目特定的验证码、MFA 或账号级风控。

所有时间输出 UTC RFC3339，精确到秒，例如 `2026-10-23T03:11:03Z`，包括登录响应的 `expires_at`。登录日志保留历史管理员 ID，不设置删除级联；第一版不自动清除登录日志，具体保留周期由项目定义。

健康接口不要求登录：`GET /health/live` 检查 HTTP 存活，`GET /health/ready` 检查 MySQL 和 Redis 可用性，不检查迁移版本、队列积压或外部业务依赖。

设置值类型支持 string、boolean、integer、string_array；空值和错误类型会被拒绝，设置专用校验规则仍有效。列表的 value 是当前有效值（数据库未保存则使用默认值），default 是代码默认值。修改成功后清空永久缓存；若数据库已提交但缓存失效失败，接口返回错误，数据库修改仍然保留，可执行 settings:clear-cache 重试清理。
