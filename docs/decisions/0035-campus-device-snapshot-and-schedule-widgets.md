# 校园白名单设备快照与原生桌面课表

## Status
Accepted
Class: architecture

## Context and Problem Statement

校园页仅有五分钟前台内存缓存，进程重启或网络失败后无法阅读最后一次成功数据，也无法为 Android 与 iOS 桌面课表提供可靠的离线来源。需要在不保存学校凭据、成绩或消息正文的前提下，提供按身份隔离的设备快照和原生 Widget。

## Decision Drivers

- 网络失败保留最后一次完整成功结果，且 Widget 永不联网。
- 数据按 API site、论坛数字账号和校园绑定版本隔离，不能跨账号或换绑复用。
- logout、unbind、换绑、切号、切站点与明确授权失效立即清理。
- 原生端只读版本化最小投影，不耦合 Flutter Drift schema 或排课器本地方案。
- 上海教学日、调休、放假、上下课时刻和跨午夜状态可在离线状态推进。
- Android 与 iOS 采用各自稳定的原生 Widget 技术，并保留可验证的发布签名配置。

## Considered Options

- 保留 [0033](0033-campus-foreground-memory-cache.md) 的五分钟前台内存缓存。
- 用现有 Drift 数据库保存受限校园白名单快照，再生成独立原子 Widget Projection。
- 让 Android/iOS 直接读取完整 Drift 数据库或排课器 `ScheduleStore`。

## Decision Outcome

选择 Drift 白名单快照并取代 [0033](0033-campus-foreground-memory-cache.md)。`profile`、`calendar`、`timetable`、`today` 以单个版本化 JSON 文档原子替换；行键为规范化 API origin 与论坛数字账号，行内携带校园 `bindingRevision`。普通网络失败保留最后成功文档；损坏或不支持的 schema 安全删除。成绩、消息列表/正文、学校凭据、token 与 cookie 不进入该快照。

主 App 只在用户手动刷新、首次绑定或重新授权后的完整白名单读取全部成功时提交快照。可见页面的分钟时钟只更新显示和执行上海午夜失效，不发起网络轮询。会话清理、解绑/换绑成功、身份切换、site 不匹配和明确授权失效同时清除校园快照与 Widget 数据。

每次快照提交后由 Dart 生成 `ScheduleWidgetProjection` schema 2。投影保留服务端已解析的当日结果，并在主 App 成功读取管理员校历规则时生成从当日起八天的紧凑窗口；Android/iOS 按上海自然日从窗口选择今天与明天。规则、校历或课表无法可靠解析时仍提交服务端当日结果，未来日期明确标为 `unknown`，不猜普通教学周。原生 decoder 继续接受 schema 1，并安全降级为单日数据。

投影只含不可展示的 site/account scope、绑定版本、生成时间、教学日期、学期周、节次、调休状态和课程显示字段，不含 token、cookie、学号、邮箱、姓名、成绩或消息正文。Dart 生成端及 Android/iOS 读取端均清洗可选文本，JSON null、字面量 `null`/`undefined` 与纯空白不会成为可见文案。写入采用临时 key 校验后替换正式 key，再请求系统 reload；刷新失败保留上一个完整投影。

Flutter 与原生桥使用 `home_widget`。Android 使用与当前 minSdk 24、AGP 9.0.1、Kotlin 2.3.20 兼容的 Jetpack Glance 1.2.0；iOS 使用 WidgetKit/SwiftUI 与 App Group。原生 Widget 不包含任何网络客户端，只按 Projection、课程起止时刻和上海午夜建立本地 timeline/alarm。官方校园课表是唯一数据源，排课器 `/schedule` 与其 `ScheduleStore` 不在本数据域中。

## Pros and Cons of the Options

- 前台内存：隐私边界最窄，但无法离线恢复、无法支持系统桌面 Widget，网络故障也会丢失展示。
- Drift 快照 + 最小 Projection：提供完整离线链路和清晰清理边界；代价是维护 schema、原子提交和双平台原生渲染。
- 原生直读 Drift/排课器：少一层复制，但把原生端绑定到完整数据库和无关数据域，扩大隐私面与迁移风险，因此拒绝。

## Links

- [被取代的前台内存缓存决策](0033-campus-foreground-memory-cache.md)
- [校园产品与保留规则](../product/campus.md)
- [移动端体验](../product/mobile-experience.md)
- [Issue #759](https://github.com/YourTongji/YourTJ-Hub/issues/759)
- [Issue #760](https://github.com/YourTongji/YourTJ-Hub/issues/760)
