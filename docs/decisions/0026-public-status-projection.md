# 将公开运行数据投影到论坛状态页

## Status

Accepted
Class: architecture

## Context and Problem Statement

YourTJ 需要在论坛内统一展示 Komari 指定节点的服务器指标、Umami 的网站访问概况与 Uptime Kuma 的服务可用性。
三套服务已有公开读取入口，数据结构和刷新语义不同，直接暴露管理员凭据没有必要。

## Decision Drivers

- 保持 Go + Vue 单二进制部署与现有导航、主题。
- 只展示配置节点与汇总流量，不扩大公开数据范围。
- 外部故障、旧采样与真实零值必须可区分。
- 不增加数据库、持久化采集任务或浏览器长期连接。

## Considered Options

- 嵌入三套服务的 iframe。
- 浏览器直接调用三套数据源。
- Go 服务端按固定字段聚合，Vue 渲染站内状态页。
- 独立部署状态服务并持久化采样历史。

## Decision Outcome

选择 Go 服务端聚合和 Vue 页面。服务端仅使用 Umami 公开共享令牌、Komari 公开 RPC 与 Uptime 已发布状态页的公开接口；
源地址与节点只由部署配置控制，拒绝重定向，接口响应只输出白名单字段。
采用固定三个访问统计区间、30 秒合并缓存、八秒请求上限与 15 分钟过期保留。
浏览器按来源标识缺失／过期数据，以独立访问检测作为服务可用性依据，明确主机运行时长
与可用率的区别；Uptime 百分比按已采集数据统计，最近检测条不冒充每日可用率。

## Pros and Cons of the Options

- iframe：接入简单，但无法统一布局、指标语义和故障状态，还耦合上游嵌入策略。
- 浏览器直连：减少后端代码，但每个访客都放大上游负载，共享令牌会交给浏览器，
  也受跨域限制影响。
- 服务端聚合：统一缓存、取消、降级和公开字段，复用现有单二进制；代价是维护三套
  适配器，论坛本身不可访问时无法独立提供状态页。
- 独立状态服务：可以脱离论坛报告故障，但增加部署和持久化系统，超出当前展示需求。

## Links

- [运行状态产品规范](../product/server-status.md)
- [Umami API](https://docs.umami.is/docs/api)
- [Umami 网站统计](https://docs.umami.is/docs/api/website-stats)
- [Komari JSON-RPC](https://www.komari.wiki/dev/rpc)
- [Uptime Kuma 公开状态接口](https://github.com/louislam/uptime-kuma/blob/e1371f402520e5df4be5be33aefd6e88663dd3c9/server/routers/status-page-router.js)
- [状态聚合实现](../../apps/gooseforum/app/service/statusservice/status.go)
