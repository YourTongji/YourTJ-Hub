# Web Push + Service Worker 站内通知推送（第二通知通道）

## Status
Accepted
Class: feature

## Context and Problem Statement

论坛站内通知（event_notification）仅在用户打开页面时可见；页面关闭后评论/回复/关注/点赞/徽章/wiki 更新全部丢失触达。issue #444 要求「页面关闭也能收到」的第二通知通道。前置 PR #449 已落地页面内 Web Notification（open-tab 通道），但依赖页面存活。

## Decision Drivers

- 页面完全关闭时仍可送达；点击推送深链到对应内容。
- 服务端异步发送，不阻塞通知创建方。
- 未配置凭据的实例必须整体关闭该通道（fail-closed），不能报错或半可用。
- 中国大陆网络环境下浏览器推送通道可达性差异必须如实定位。

## Considered Options

- Web Push + Service Worker（VAPID），浏览器标准通道（选定）。
- 邮件通知作为第二通道。
- 移动端原生推送先行（后续由 [0010](0010-mobile-route-a-native-alignment.md) 以 APNs/FCM 并行补齐）。

## Decision Outcome

1. 通道：Web Push（VAPID）+ 根 scope Service Worker（`/sw.js`，无 fetch handler 的纯 push 形态）+ `/manifest.webmanifest`（app.gohtml head 引入）；设置页「推送通知」开关（权限 + subscribe/unsubscribe），带能力检测与优雅降级。
2. 触发：通知创建后服务端触发推送，覆盖 `notificationservice/send.go` 6 个 Send*（comment/post_reply/topic_post/badge/like/follow）与 `wikiservice` `notifyWatchers`（wiki_updated）；异步经 taskQueue outbox + 专用 worker 发送。
3. 配置：实例级 `[webpush] vapid_public_key/vapid_private_key`（走 [0006](0006-deploy-config-as-artifact.md) Environments 管线）；未配置 = 通道关闭；新增 `webpush-keys` cobra 生成子命令。
4. 库与协议：`github.com/SherClockHolmes/webpush-go`（master ≥ 2026-04-22 AuthScheme 提交，v1.4.0 tag 不支持 Chrome `WebPush` scheme）；Chrome 端点（fcm.googleapis.com）→ `AuthScheme: WebPush`，Firefox/Apple → RFC 8292 `vapid` 默认；TTL 显式 24h（库默认 0 = 立即过期）。
5. 生命周期：subscribe 的 applicationServerKey 只收 65B P-256 未压缩点（base64url）；账号注销（anonymize + delete 双 mode）删除该用户全部订阅；发送遇 404/410 删除失效订阅；401/403/400 不重试（配置错），429/5xx 记日志。
6. 送达定位：Safari 需 iOS 16.4+ 添加到主屏幕（manifest `display: standalone`）/macOS 13+ 直接可用；中国大陆 Chrome/FCM 无可靠直连，Firefox/Apple 端点通常可达 → 定位为增强通道，站内通知与应用内轮询保持主通道语义。

### Consequences

- Good: 页面关闭仍可触达；通道随 VAPID 配置存在与否自动开关，零配置实例行为不变。
- Trade-off: 大陆 Chrome 用户基本无增强通道（已知接受）；Safari 受 PWA 安装约束。
- 后续移动端原生推送（APNs/FCM）与之并行成双通道，同文案单源、同触发点、同清理语义。

## Pros and Cons of the Options

### Web Push + SW（选定）
- Good: 浏览器标准、免装 App、免租推送服务；凭据自持（VAPID 自签）。
- Bad: 大陆 FCM 不可达；iOS 需 PWA 安装；Service Worker 生命周期需管理。

### 邮件通知
- Good: 无浏览器兼容问题。
- Bad: 触达延迟高、深链体验差、SMTP 配置与反垃圾负担，不适合高频站内互动。

### 原生推送先行
- Good: 送达率最高。
- Bad: 需要应用商店分发与原生客户端就绪，周期长；Web 端触达缺口无法等待。

## Links

- issue #444
- PR #449（本决策随 PR 落地合入 dev）
- 调研与蓝图：宿主 Blueprint「Web Push + Service Worker 站内通知推送」（nte_06dad7e39001MfBs9CMkYYBzBd）
- 并行通道：[0010](0010-mobile-route-a-native-alignment.md)
