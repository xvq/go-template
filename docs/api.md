# API 约定

管理接口前缀 `/api/admin`。除登录之外都要求 `Authorization: Bearer <token>`。管理员权限一致。

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
  "errors": {"username": "请输入账号"},
  "request_id": "..."
}
```

400 格式/参数错误；401 未登录或凭证错误；404 不存在；409 账号重复或最后管理员保护；429 登录限流；500 内部故障；503 服务未就绪；504 处理超时。500 响应不暴露底层数据库和 Redis 错误。

列表参数 `page` 默认 1、最大 1000000；`page_size` 默认 20、最大 100。返回 `data: {items: [], total: 0, page: 1, page_size: 20}`。

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
| PATCH | `/admins/:id` | `display_name`、`enabled`，至少一个；不允许修改 username |
| DELETE | `/admins/:id` | 删除管理员，撤销会话 |
| PUT | `/admins/:id/password` | `password`；重置密码，撤销其全部会话 |
| GET | `/login-logs` | 分页查询登录日志 |
| GET | `/settings` | 设置列表，包含 key、type、sensitive 和非敏感 value |
| PATCH | `/settings` | `{"values":{"site.name":"My site","site.maintenance":false}}` |

用户名是 3～64 位字母、数字、下划线、点或连字符，按 MySQL 表的大小写不敏感排序规则判断唯一性。密码 12～72 **字节**，不自动裁剪空格。显示名称最多 100 个 Unicode 字符。PATCH 中未提供和 null 都表示不修改；空字符串可清空显示名称，`enabled:false` 会禁用账号。

登录按 IP 使用 Redis 计数限流，跨 HTTP 实例共享；默认每分钟 10 次（成功和失败请求都计数）。错误密码、未知账号、禁用账号返回相同提示。该基础策略不代替项目特定的验证码、MFA 或账号级风控。

所有时间输出 UTC RFC3339。登录日志保留历史管理员 ID，不设置删除级联；第一版不自动清除登录日志，具体保留周期由项目定义。

健康接口不要求登录：`GET /health/live` 检查 HTTP 存活，`GET /health/ready` 检查 MySQL 和 Redis 可用性，不检查迁移版本、队列积压或外部业务依赖。
