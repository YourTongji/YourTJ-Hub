# 会话吊销：JWT jti + user_sessions 表

## Status
Accepted
Class: architecture

## Context and Problem Statement

- 论坛 JWT 自签会话凭证此前仅有 TokenVersion 全局失效机制，无法按设备/会话粒度吊销，也无法向用户展示登录设备列表。
- issue #8 要求"查看并吊销其他设备会话"，会话管理需要会话粒度的服务端状态。

## Decision Drivers

- 吊销必须即时生效，不能等待 token 自然过期。
- 必须支持按设备粒度列出与吊销，同时保留全局失效兜底。
- 多实例部署下状态必须共享。

## Considered Options

- JWT 增加 `jti` claim + `user_sessions` 表承载会话状态（选定）。
- 纯 JWT 短过期 + refresh token 轮换。
- 服务端全量会话表、JWT 退化为不透明 token。

## Decision Outcome

1. 论坛 JWT 增加 `jti` claim，映射 `user_sessions` 表（jti 唯一索引）承载会话状态。
2. 中心校验点 `JWTAuthGetUserId` 要求所有 token 映射有效会话行；吊销即时生效（无需等 token 自然过期）。
3. 续签保留同一 jti 并同步会话过期时间，保证"当前会话"在列表中稳定。
4. 单会话吊销删除对应行；"退出所有设备"删除全部行 + `TokenVersion++` 双保险。
5. `TokenVersion` 保留为全局失效兜底（改密/封禁语义不变）。
6. challenge token（TOTP 2FA 第二因素）不写会话表，仅专用中间件接受，普通 API 天然拒绝。

### Consequences

- Good: 吊销即时生效；会话列表返回掩码 IP 与设备标签，隐私受控。
- Trade-off: 每次登录产生一行会话记录；过期行在 serve 启动时清理。
- 多实例部署时 jti 校验依赖共享数据库，单二进制部署形态不受影响。

## Pros and Cons of the Options

### jti + 会话表（选定）
- Good: 保留 JWT 无状态校验路径，仅增加一次索引查询；粒度到设备。
- Bad: 登录/校验路径引入 DB 依赖与会话行生命周期管理。

### 短过期 + refresh 轮换
- Good: 无服务端状态。
- Bad: 吊销延迟到 refresh 边界，且无法列出设备列表，不满足 issue #8。

### 全量不透明会话
- Good: 语义最简单。
- Bad: 放弃 JWT 既有生态（claims、TOTP challenge token 复用），改动面大。

## Links

- issue #8
- 契约：packages/api-contract（session management list/revoke/revoke-all）
