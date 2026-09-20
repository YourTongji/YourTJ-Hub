# 同济官方身份作为独立私密校园连接

## Status
Superseded by [0032](0032-tongji-login-and-registration.md)
Class: architecture

## Context and Problem Statement

YourTJ 需要复用 OneTJ 已获授权的同济开放平台能力，并支持官方身份绑定、解绑与换绑。论坛用户仍是唯一账号源；学校凭据需要自动刷新且不能泄漏到浏览器、公开资料或同步到 dev 的生产快照。

## Decision Drivers

- 一账号一官方身份，双向唯一；失败不能破坏原绑定。
- 保留 Go + Vue 单二进制部署和现有论坛会话。
- 支持真实课表、学业记录和学校消息，避免复制整套 Flutter 状态存储与遥测。
- 把身份占用和短期学校访问授权分开，刷新失败仍保留身份。

## Considered Options

- Go 服务端独立校园连接 + Vue 工作台 + App 已登录 WebView 入口。
- 将学校身份直接加入论坛 OAuth 登录提供方。
- 浏览器或 Flutter 客户端直接保存 OneTJ token，并复用完整应用。

## Decision Outcome

选择服务端独立校园连接。复用 OneTJ 的客户端授权、scope 与接口字段知识，由 YourTJ 自有回调处理 state、PKCE S256、nonce、RS256 验签和学校标识校验。学校认证结果经用户确认后绑定，不自动创建论坛用户或替换其登录方式。

数据库以论坛 user ID 为主键、以加密密钥之外的 HMAC 密钥生成学校身份唯一键；凭据 AES-GCM 加密并绑定 user ID。绑定换绑用版本条件和唯一约束提交。授权尝试只在进程内保留十分钟；多实例共享事务存储不属于当前支持模型。实时校园数据不落库，前端收到字段白名单投影。生产到 dev 的同步排除绑定表数据。

Web 提供完整校园工作台；App 复用已登录网页并同步契约，原生页面仍为 Partial。连接不是论坛登录方式，也不是公开身份徽章。

## Pros and Cons of the Options

- 服务端连接：一个安全实现服务多端，学校凭据不下发，统一刷新和解绑；代价是服务端承担凭据保管，部署需要独立 secrets 和快照过滤。
- OAuth 登录提供方：能够一键进入论坛，但把校园数据授权与账号恢复/注册混合，改变当前用户确认的账号语义，因此不采用。
- 客户端直连：容易移植 OneTJ 页面，但重复多端刷新与加密逻辑，浏览器存储令牌，且无法可靠维护全站双向唯一；完整移植还带来不需要的状态管理和遥测依赖。

## Links

- [校园产品语义](../product/campus.md)
- [校园运维](../operations/campus.md)
- [OneTJ 认证与数据服务](https://github.com/oierxjn/OneTJ/blob/e43099a046e522be16e0a19bc1c2a6288d1532c7/lib/services/tongji.dart)
- [学校 OAuth 授权码文档](https://api.tongji.edu.cn/docs/intro/develop/authentication_code)
