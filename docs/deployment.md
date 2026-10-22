# 部署

## 二进制与配置

构建 `./cmd`，复制二进制和独立的 `config.yaml` 到服务器。SQL 已嵌入程序，不需要额外复制迁移目录。运行用户需有日志目录写入权限，配置文件只授予必要的读取权限。

```sh
./app -c /etc/my-app/config.yaml migrate up
./app -c /etc/my-app/config.yaml server
```

分别通过 systemd、容器或进程管理器运行 server、schedule 和可选 queue:work。只部署一个 schedule；queue:work 可以多实例，但 `queue.concurrency` 是每个进程的并发数。

SIGINT/SIGTERM 触发退出：HTTP 停止接收并等待在途请求，超出期限后取消上下文；调度停止并等待任务；Asynq 按配置等待任务并恢复未完成任务。所有业务长任务都必须遵守 context。Windows 本地使用 Ctrl+C。

## 反向代理

默认只监听 127.0.0.1:8080。需要容器或外部访问时显式修改 host。TLS 通常在 Nginx/Caddy 等反向代理终止。只有配置了 server.trusted_proxies 的来源才能影响客户端 IP，不要直接信任所有 X-Forwarded-For。限制到实际代理 IP/CIDR，并由代理覆盖/规范化转发头。

跨域默认关闭。需要浏览器跨域时在 server.cors_origins 声明允许来源；当前使用 Bearer 认证，不启用跨域 Cookie 凭证。

## Redis

会话、登录限流和设置缓存使用 redis.prefix 区分项目。Asynq 使用自己的键名，需要单独的逻辑 DB 或实例来区分项目。逻辑 DB 不能隔离容量、淘汰策略或故障。

如果 Redis 同时保存队列，不应按普通可丢缓存配置：启用适合业务的持久化，使用 noeviction 并监控容量、错误与积压。需要更严格资源隔离时把 queue.redis 指向独立实例。Redis 重启后的会话/任务保留程度取决于持久化和恢复配置。

第一版客户端配置面向受保护网络中的单机 Redis（不支持 Cluster），MySQL/Redis 的专用 TLS/哨兵拓扑配置尚未内置。需要连接要求 TLS 的托管服务时，在部署前扩展连接配置与测试，不能通过关闭证书校验绕过。

## 队列工具

Asynqmon/Asynq CLI 可独立运行，配置同一个队列 Redis 地址、认证和 DB。它们不经过业务 API，也不自动继承管理员登录态。放在受控网络或认证代理后面。使用与当前 Asynq 数据格式兼容的工具版本。

## 更新

发布前备份数据库，检查迁移 SQL，然后显式执行 migrate up。MySQL DDL 失败后先检查实际表结构；确认修复完成才使用 migrate force。down 默认仅一步，需要 --yes。不要把回滚命令当作数据备份恢复。

设置更新通常立即使缓存失效；Redis 失败时允许缓存 TTL 内的短暂旧值。对不能接受此窗口的业务设置，应由项目实现更严格的读取策略。
