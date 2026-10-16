# 校园隐私内容的后台卸载触发时机

## Status
Accepted
Class: bug-fix

## Context and Problem Statement

校园私密页在离开前台时丢弃成绩、消息正文等隐私内容，避免它们留在应用切换器缩略图等系统快照中。此前实现把任何非 `resumed` 生命周期状态都当作后台：Android 拉下通知栏、控制中心等瞬时遮挡（`inactive`，内容仍在屏上）也会卸载隐私子树、suspend 数据并清空前台内存缓存，收起遮挡后整页骨架屏重载并重新读取学校数据（issue #1109 期间暴露）。需要明确：哪些生命周期状态才算"被系统覆盖"。

## Decision Drivers

- 瞬时遮挡（通知栏、控制中心、来电、分屏失焦）时内容仍在屏上，不应触发整页重载与重复学校请求。
- 真正退后台时仍要丢弃隐私正文、取消在途请求、清空前台内存缓存，恢复时先核验绑定再取数。
- 不新增持久化，不改变成绩、消息正文与学校凭据不进入设备快照的既有边界。
- 判定要能用 widget test 断言，并与框架的帧调度语义一致（`SchedulerBinding.handleAppLifecycleStateChanged` 在 `resumed`/`inactive` 启用帧、在 `hidden`/`paused`/`detached` 停用帧）。

## Considered Options

- 沿用"非 `resumed` 即卸载"：`inactive` 同样卸载、suspend、清缓存。
- 仅 `paused`/`hidden`/`detached` 触发卸载，`inactive` 保持前台（本决策）。
- `inactive` 仅视觉卸载：渲染骨架屏但不 suspend、不清缓存。

## Decision Outcome

选择仅 `paused`/`hidden`/`detached` 触发卸载，`inactive` 视为仍在前台：`campus_page`、`campus_private_surface` 与 `campus_memory_cache` 统一通过 `isCampusForeground`（`apps/mobile/packages/forum_app/lib/src/pages/campus/campus_lifecycle.dart`）判定。Android 在通知栏下拉、应用切换器可见时给出 `inactive`，这是框架文档化的行为；`hidden` 是进入 `paused` 前合成的状态，因此退后台的清理路径不变。

恢复前台时仍先核验绑定状态，再请求当前栏目数据；后台已清空的缓存不会复活旧数据（`clear()` 递增 generation 并使 `revision` 失效，迟到响应被版本隔离拒绝）。

已知取舍：Flutter 自 `hidden`/`paused` 起停用帧调度，卸载产生的空白帧不会绘制，应用切换器缩略图可能保留含隐私内容的最后一帧。旧实现在 `inactive`（帧仍启用）即渲染空白帧，缩略图防护更强；本决策优先消除瞬时遮挡的重载与重复请求，缩略图防护的缺口改由"恢复前台重新核验、缓存已清"承担，并记录为已知限制。若后续需要兼顾，采用备选方案"`inactive` 仅视觉卸载"。

## Pros and Cons of the Options

- 非 `resumed` 即卸载：缩略图防护最强，实现最简单；代价是通知栏下拉等瞬时遮挡导致整页骨架屏重载、重复读取学校数据，并放大平台差异带来的误判（issue #1109 的用户体感问题）。
- 仅 `paused`/`hidden`/`detached` 触发：与框架帧调度语义一致，消除瞬时遮挡的重载；代价是切换器缩略图防护变弱（见 Decision Outcome 的取舍说明）。
- `inactive` 仅视觉卸载：可同时保住缩略图与前台数据；但收起遮挡时仍会闪一次骨架屏，且需要维护"视觉卸载"与"数据卸载"两套状态，复杂度更高，暂不采用。

## Links

- [用户反馈 #1109](https://github.com/YourTongji/YourTJ-Hub/issues/1109)
- [校园产品与保留规则](../product/campus.md)
- [0033 校园数据的短期前台内存复用](0033-campus-foreground-memory-cache.md)
- [0035 校园白名单设备快照与原生桌面课表](0035-campus-device-snapshot-and-schedule-widgets.md)
