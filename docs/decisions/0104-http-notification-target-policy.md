# HTTP 通知回调目标地址的策略边界

## Status

Accepted
Class: bug-fix

## Context and Problem Statement

管理员 HTTP 通知（通用 JSON／飞书机器人／AstrBot）早于 `safefetch` 存在：投递用裸 `http.Client`，
只在构造请求时检查 `scheme ∈ {http, https}` 与主机名非空，保存端点时完全不校验目标。`SiteManager`
因此可以让服务端向任意主机端口 POST 固定 JSON 报文，并借状态码差异做端口探测，
或持续把含主题／评论摘要与审批卡片 token 链接的领域事件投递到该地址（issue #1280）。

同批次上线的 Agent Webhook 已走 `safefetch`（每次连接重新解析并 pin、禁代理与连接池复用、
禁重定向、协议／端口限制、头白名单、超时与报文上限）。需要决定：通知这条既有路径接入同一套
出站加固时，目标地址策略取到多严。

## Decision Drivers

- 通知目标由管理员显式配置，但仍是服务端发起的出站请求，必须受同一层边界约束。
- 企业内网自建接收端（如内网 AstrBot、公司 webhook 网关）是合法且常见的部署方式，不该被迫改架构。
- 不得为通知另造一套发请求逻辑：URL／DNS／代理／重定向／超时策略在两条路径上会漂移。
- 已保存的历史端点（可能指向 127.0.0.1）在投递时同样要拦。

## Considered Options

- 复用 `safefetch`，但新增“通知目标”策略：放行 http 与任意端口、放行 RFC1918／ULA 私网，
  拒绝环回、链路本地（含云元数据 `169.254.169.254`）、unspecified、multicast 与 IPv4-mapped 字面量（本决策）。
- 完全按 Agent Webhook 的既有策略（仅 https:443、仅公网）：会直接切断内网通知部署。
- 默认封禁全部私网，另加显式配置开关放行内网目标：多一个部署必答题和一条配置漂移面。
- 只在保存时校验目标，投递仍用裸 client：DNS rebinding 仍可绕过。

## Decision Outcome

选择第一种：`safefetch` 增加通知目标策略，保存与投递两处都校验。

- `safefetch.ValidateNotifyURL` 只做 URL／主机名层判定：允许 http 与 https、任意端口、查询串；
  拒绝用户信息、fragment、`localhost`／`*.localhost`／带 zone 的地址、纯数字主机名写法，
  以及按策略被拒绝的 IP 字面量。
- `safefetch.Client.CheckNotifyTarget` 只在保存端点时解析 DNS 并逐个校验应答地址；
  测试发送与正式投递都不预先查 DNS，而是在 `safefetch.Client.DoNotify` 拨号时重新解析并 pin
  接受的那个地址（保存时的结果只用于尽早拒绝，不是投递的前置条件）。
  `safefetch.Client.DoNotify` 投递时复用同一 pin dialer（每次连接重新解析、禁代理、禁用连接池、
  不跟随重定向），返回活响应供通道解析 2xx 与业务码。
- 地址策略：私网（RFC1918／ULA）放行；环回、链路本地 unicast/multicast、unspecified、multicast、
  IPv4-mapped IPv6 与既有保留段（含 CGNAT `100.64/10`）一律拒绝。取舍在代码中以
  `shortcut:` 注释记录。
- 投递路径仍是单一通道：`httpnotifyservice` 的 `X-Goose-*` 与 HMAC 签名、AstrBot Bearer 头、
  飞书签名语义不变，只把“怎么发出去”换成 `DoNotify`。
- 运维后果：升级前已保存、指向 `127.0.0.1` 的端点仍留在配置里不做迁移，但每次投递都会被拒；
  连续 3 次失败后自动熔断并停用该端点。管理员需要把地址改成论坛宿主机的内网地址（如
  `http://192.168.x.x:9966/send`）才能恢复。
- 失败信息只有固定文案，不回显解析到的内部地址，也不回显 URL：保存端点被拒时返回消息码
  `common.request.invalidParams`；测试发送结果与端点 `lastError` 是
  `invalid notify url (expected http(s)://host[:port]/path on a non-local target)`（URL 层被拒）
  或 `safe fetch: <class>`（拨号期被拒，如 `safe fetch: blocked`）。这是为避免把校验错误变成新的
  探测通道。

## Pros and Cons of the Options

- 通知目标策略（选定）：内网通知继续可用，环回与云元数据被关闭，出站加固只有一份实现；
  代价是放行 RFC1918 后仍保留一定的内网横向移动面——若日后管理员角色向不受信的多租户开放，
  需要改为显式配置开关（记录在代码注释的 `shortcut:` 里）。
- 复用 Agent Webhook 严格策略：边界最窄；代价是内网自建接收端全部失效，等于用安全修复换掉一个既有部署能力。
- 私网封禁 + 配置开关：租户隔离更好；代价是多一条必须解释的配置、一份部署文档和开关本身的误配风险，
  而当前管理员就是站点运维者本人。
- 仅保存时校验：改动更小；代价是 DNS rebinding（保存时解析为公网、投递时解析为环回）仍成立，
  而钉住地址的成本只是复用已有 dialer。

## Links

- [issue #1280](https://github.com/YourTongji/YourTJ-Hub/issues/1280)
- [0096 Agent Webhook 静态 Authorization 头如何穿过 safefetch 边界](0096-agent-webhook-static-authorization-forwarding.md)
- [0067 HTTP 通知的有限重试与熔断](0067-bounded-http-notification-retries.md)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
