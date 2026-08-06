# 身份、登录与账号生命周期

> 文档类型：产品规范
>
> 状态：Active（认证链路 `Planned`，数字 ID 前提已验证）
>
> 负责人：Platform maintainers、Security reviewer
>
> 最近核验：2026-08-06

## 身份模型

- **身份唯一来源 = Casdoor（OIDC）**。论坛不维护独立密码体系；Casdoor 签发 id_token，
  `sub` 为数字用户 ID（uint64）。
- **数字 ID 是硬约束**：credit（积分）的 `GetID()` 用 `strconv.ParseUint` 解析 sub。
  UUID 会解析失败并回落为 0，导致所有用户互相覆盖。Casdoor 必须：
  1. 应用 SignupItems 的 ID 规则设为 `Incremental`（用户自助注册得自增数字 ID）；且
  2. 管理员建号时显式传数字 `id`。
- 论坛 JWT 只是**会话凭证**（HS256，自签，可加 refresh_token），不是身份事实源；
  用户封禁/状态变更以 Casdoor 为准，server 通过 exchange 时同步本地投影。

## 登录流程

### Web

标准 OIDC 授权码流程：浏览器重定向到 Casdoor → 回调 → server 用 code 换 id_token →
验签（iss/aud/nonce/exp）→ 查找或创建本地用户 → 签发论坛 JWT（HttpOnly cookie 或
Authorization header 均可，落地时定）。

### Mobile（Flutter）

1. appauth + PKCE → Casdoor 授权页（外部浏览器）→ 回调拿 code；
2. token endpoint 换 id_token（验 nonce）；
3. `POST /api/auth/oidc/exchange`（body: idToken, nonce）→ server 验签 → 返回论坛 JWT；
4. JWT 存 Keychain/Keystore（flutter_secure_storage）；id_token 只存内存不落盘。

## 账号生命周期

- 注册：Casdoor 自助注册（Incremental ID）或管理员建号（显式数字 id）。
- 登录态：论坛 JWT 7 天有效（沿用 GooseForum 量级，落地时可调）；支持 refresh_token 则
  短 access + 长 refresh；改密码/被踢 → TokenVersion 机制使旧 token 失效。
- 封禁/停用：以 Casdoor 的 active/forbidden 为准；server 在 exchange 与每次校验时同步。
- 删除/导出：`Planned`（遵循产品原则 12：落库前回答用途、可见者、保留期、导出、删除）。

## 安全要点

- PKCE 必开（appauth 默认）；nonce 防重放（exchange 端点必须校验）。
- id_token 不落盘；论坛 JWT 进系统安全存储。
- 防枚举：登录错误不区分"用户不存在/密码错误"（Casdoor 侧配置）。
- 会话撤销：TokenVersion 或 refresh 轮换（落地时定，决策记录见项目 note）。
