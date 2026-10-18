# 外链免确认白名单由管理端配置

## Status

Accepted
Class: feature

## Context and Problem Statement

用户反馈（[issue #1200](https://github.com/YourTongji/YourTJ-Hub/issues/1200)）：点击主流站点外链时，
每个浏览会话首次都要经过「即将离开 YourTJ」确认层；「不再询问」只写入 sessionStorage，会话结束即失效，
产品层面没有可信站点名单。现有实现把豁免严格限制在 normal_external 风险上（suspicious/blocked
永不免除），扩展点清晰，但「名单由谁维护、匹配粒度多细」需要决策：内置静态名单改动最小；管理员配置
运营灵活但要新增设置面与契约；用户自选持久化体验最个人化，却引入长期信任语义与撤销入口。

## Decision Drivers

- 用户体验：主流可信站点（技术资料、内容平台）不应在每个会话反复确认。
- 安全边界：确认层是钓鱼链接的最后一道提醒，豁免面必须可解释、可审计、默认从严。
- 两端一致：Web 与 Flutter 使用同一份清单与同一豁免判定，避免行为分叉。
- 运维现实：可信站点集合随时间变化，需要站点运营者自行增删，不应依赖发版。
- 兼容存量：已上线站点升级后不应因缺少新字段退回「全部链接都确认」。

## Considered Options

- 内置静态名单随前端打包（首期最小改动方案）。
- 管理员可配置：站点设置内的白名单列表（选定）。
- 用户自选并持久化：个人长期信任与撤销入口。

## Decision Outcome

白名单是站点设置的一部分（`siteSettings.trustedExternalDomains`），默认值为内置清单（主流技术站点
与国内内容平台）。命中判定按主机名后缀且必须落在点边界上（条目覆盖其子域，反向不成立），仅豁免
normal_external；suspicious/blocked 始终确认或拦截。保存期由 `bundles/domainpolicy` 整单校验：
拒绝带协议、路径、端口、通配符、IP、单标签与公网后缀（如 `github.io`）的条目，上限 200 条，条目
归一化为小写（IDN 转 punycode）；读取期过滤历史脏值，并在键缺失或为 null 时补齐默认清单。保存请求
缺省该字段（部署期间仍打开的旧管理端页面）同样按「保留现有清单」处理，只有显式空数组表示「不豁免
任何域名」，避免一次无关保存静默停用白名单。Web 由 SSR payload（`layout.site.trustedExternalDomains`）
下发，Flutter 启动时读取公开端点 `GET /api/external-link-policy`，两端共用同一份清单与同一豁免边界；
两端都在 punycode 形式上比较（Flutter 需把 `Uri.host` 的百分号编码还原后再编码非 ASCII 标签），
保证 IDN 条目在两端一致命中。标签与目标不符的链接在 Flutter 侧按源文本预先判为 suspicious
（Markdown AST 与渲染后 HTML 两条链路：帖子/评论、Wiki 与课评正文；富文本渲染器只把 href 交给点击
回调，聊天是纯文本、可点文本即 URL），与 Web 的渲染后文本判定等价，因此同样不被白名单豁免。

不选内置静态名单：站点集合变化必须发版，无法按站点定制，与「运营可维护」矛盾。
不选用户自选持久化：引入长期信任语义、撤销入口与跨设备一致性问题，交互与安全面最重，且无法表达
站点运营意图。

## Pros and Cons of the Options

### 管理员可配置（选定）

- Good: 站点运营者自行增删，无需发版；默认清单覆盖常见需求。
- Good: 存量站点升级即用（缺键补默认），且显式清空仍可表达「从严」。
- Good: 豁免面受服务端校验与读取期过滤约束，公网后缀等危险条目不可能生效。
- Bad: 新增设置面、公开端点与契约/Dart 镜像，改动面大于纯前端常量。
- Bad: 按主机名后缀匹配会连带豁免条目的子域；收录承载用户生成内容的域名时由运营权衡，
  默认清单避开按租户隔离的私网后缀（`github.io` 一类条目也不会被接受）。

### 内置静态名单随前端打包

- Good: 改动最小，无后端与契约面，随常规 PR 更新。
- Bad: 任何增删都要发版；不同站点无法按自身受众调整。

### 用户自选并持久化

- Good: 最贴合个人偏好，一次信任长期有效。
- Bad: 需要长期信任的存储、撤销与同步设计；无法表达站点推荐的可信站点集合。

## Links

- [issue #1200](https://github.com/YourTongji/YourTJ-Hub/issues/1200)
- [论坛体验：链接、Markdown 与表情](../product/forum.md#links-markdown-and-stickers)
- [公开端点契约](../../packages/api-contract/paths/external-link-policy.yaml)
- [服务端条目校验](../../apps/gooseforum/app/bundles/domainpolicy/domainpolicy.go)
- [Web 豁免判定](../../apps/gooseforum/resource/src/runtime/external-link-guard.ts)
- [Flutter 豁免判定](../../apps/mobile/packages/forum_app/lib/src/link_navigation.dart)
