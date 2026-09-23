# PRD：Flutter App 今日课表桌面小组件（Android / iOS）

实施状态：`Current` — #759 设备快照、Projection、Flutter bridge、Android Glance 与 iOS WidgetKit/App Group 源码及发布配置已实现；`Partial` — 当前 Linux 环境没有 Android/iOS 设备或 Xcode，OEM、Doze、重启、系统添加流程及 iOS 真机渲染仍待设备矩阵验证。

> Doc type: Product Requirements Document
> Status: Draft
> Owner: Mobile / Campus
> Target branch baseline: `origin/dev@a0cba7e4`
> Related: #759（校园课表持久化 + 手动刷新）、#714（排课器多端同步，非本功能主数据链路）
> Research date: 2026-09-22

## 1. Problem Statement

YourTJ Flutter App 已有“校园 → 今日课表 / 我的课表”能力，但当前校园数据只存在于请求过程和最多 5 分钟的前台内存缓存中。用户冷启动、进程被系统回收或断网时，无法立即看到已有课表；更无法把今日课表作为系统桌面上的“一眼可读”信息持续展示。

Issue #759 已提出：至少把 profile / calendar / timetable / today 等校园数据持久化到设备，并将刷新语义调整为用户手动触发。这一变化为桌面小组件提供了必要的离线数据底座。

桌面小组件的核心约束是：Android AppWidget 与 iOS WidgetKit 都不能假设 Flutter 进程常驻，也不能把“定时后台联网”当作正确性前提。尤其在 Doze、iOS WidgetKit 刷新预算、HyperOS / HarmonyOS 等国产系统的强后台管控下，依赖后台请求会产生不可预测的空白、过期或刷新延迟。

因此，本功能的产品目标不是“让小组件在后台不停访问服务器”，而是：

1. 让绑定了同济校园身份且已经成功获取过课表的用户，在桌面无网络、App 被杀、系统省电时仍能查看**最后一次已确认的课表快照**；
2. 通过本地时间线推导“当前课程 / 下一节课 / 今日剩余课程”，使状态变化不依赖网络；
3. 当用户在 App 内手动刷新、身份变化或课表缓存被清除时，再同步更新 Widget 数据并请求系统重绘；
4. 在 Android / iOS 平台约束内提供稳定、克制、与 YourTJ 现有课表设计一致的体验。

## 2. Evidence & Current Architecture

### 2.1 当前仓库事实

- `CampusRepository` 当前明确是 request-scoped，并声明不把学校记录或凭据写入磁盘。
- `CampusMemoryCache` 只缓存 `profile / calendar / timetable / today / messages`，生命周期 5 分钟；进入后台即清空。
- `CampusController` 当前在进入标签、返回可见页和跨教学日时会重新拉取或检查数据。
- `docs/product/campus.md` 当前明确规定 Flutter 校园记录不进入离线库；#759 会改变这一产品边界，因此本功能必须依赖 #759 的决策与实现，而不是绕过它另建私有缓存。
- 官方课表与排课器已经共享节次表、网格和课程卡设计语义；Web `CampusTimetable.vue` 也明确要求官方课表只作展示，不写入排课器状态。
- Flutter App 已依赖 `drift`、`shared_preferences`、Riverpod，并有现成的 `offlineCacheEpochProvider` 用于账号/站点变化时隔离本地缓存。
- App 内已有 `CampusEvent`、`CampusTeachingDay`、`SectionTime`、默认 11/12 节作息、课程稳定配色槽位等可直接复用的领域模型与算法。

### 2.2 参考项目结论

#### TJ-Class-Schedule

用户指定参考项目的有效信息：

- 本地 SQLite 保存课程、学期、设置和临时停课；
- 今日课表、本周课表、教学周切换、下一节课倒计时；
- 桌面小组件展示下一节课、时间、地点、教师和倒计时；
- 明确为荣耀、华为、小米、OPPO、vivo 等国产 Android 提供后台设置说明；
- 功能结构中单独保留 `widget/` 作为桌面小组件数据同步层。

可借鉴的是“**本地课表作为唯一真源 + Widget 只读投影**”以及“下一节课优先”的展示策略，而不是其独立抓取学校页面的方式。

#### ClassWidget

- Flutter + SQLite + Riverpod；
- Android Widget 提供 2x1、4x2、4x4 三档；
- Widget 由 Kotlin AppWidget Provider / RemoteViews / BootReceiver 实现；
- Flutter 侧通过 `home_widget` 和 `workmanager` 同步；
- 有“Up Next”倒计时和跨午夜更新；
- 重启后重新计划 Widget 更新。

可借鉴的是尺寸分级、下一节课语义、跨午夜与重启恢复；但 YourTJ 不应把 WorkManager 周期联网作为数据正确性前提。

#### UntisPlus / 其他课表项目

- UntisPlus 使用 `home_widget`，强调课表离线本地存储与 home widget；
- VITTY 的典型产品形态是 Today + Next Class 两种 Widget；
- BSUIR Schedule 同时采用 iOS WidgetKit、Android Glance 风格 Widget，并以离线缓存作为 fallback。

共同模式：**Today / Next Class 是最适合课表桌面组件的两个信息层级；完整周课表不应直接塞入小尺寸组件。**

### 2.3 平台约束

#### Android

- App Widget 是由 Launcher 托管的系统组件，并非 Flutter Widget 树的一部分；
- Android 12+ 支持更精细的响应式/精确尺寸布局；
- 周期更新受系统限制，Doze / App Standby / OEM 电源策略会延迟后台任务；
- 对固定可预知事件，应优先依靠本地时间与系统触发刷新，而非高频周期任务；
- 用户主动在 App 内刷新数据时可主动请求 Widget 更新。

#### iOS

- WidgetKit Extension 在独立进程中用 SwiftUI 渲染，不保证常驻；
- 正确模式是提供 Timeline：系统依据 TimelineEntry 的时间切换内容；
- App 获得新数据后可以请求 WidgetCenter reload；
- 系统对 reload 有预算，不能依赖分钟级后台刷新；
- Widget 需要通过 App Group 读取主 App 写入的共享数据。

## 3. Goals

### 3.1 用户目标

1. 已绑定校园身份并至少成功获取过一次课表后，用户在桌面即可看到今天的课程，不必先打开 App。
2. 无网络、App 进程被杀或系统进入省电状态时，小组件仍显示最后一次可用课表。
3. 用户能一眼判断：现在是否上课、下一节课是什么、在哪里、几点开始、今天还剩几节。
4. 用户手动刷新 App 课表后，小组件在平台允许的合理时间内更新。
5. 不因账号切换、解绑、站点切换而泄漏上一账号课表。

### 3.2 产品目标

1. 建立一个可扩展的“校园本地投影 → 原生 Widget”桥接层，为未来锁屏组件、Apple Watch/Android Wear 等提供复用基础。
2. 不引入常驻后台服务、不增加服务器定时查询压力。
3. 视觉与现有 Web / Flutter 课表保持同一设计语言，而非另做一套“第三方小组件风格”。
4. 将 #759 的离线能力建设成明确、可测试的共享基础设施。

## 4. Non-Goals

本期明确不做：

- 不让 Widget 直接调用 YourTJ API 或同济官方 API；
- 不在 Widget Extension / AppWidget Provider 中持有论坛 token、学校 token 或任何凭据；
- 不做分钟级后台网络轮询；
- 不把成绩、消息正文、学号、校园凭据写入 Widget 共享存储；
- 不在首期做完整 7 天周课表缩略图；
- 不在首期允许直接从 Widget 修改排课方案；
- 不把 #714 的排课器云同步方案作为本功能前置依赖；
- 不承诺国产 Android 厂商“强杀”后仍有绝对精确的后台 reload；已有本地快照必须仍可显示；
- 不在首期做 Android 锁屏通知常驻、Live Activities / Dynamic Island。

## 5. Data Source Boundary

### 5.1 P0 主数据源：学校官方个人课表

本功能只面向“绑定了校园身份，并成功获取过学校课表”的用户。

主数据来源：

- `calendar`：学期、教学周信息；
- `timetable`：当前学期个人课表；
- `today`：服务端应用调休规则后的今日课表结果；
- 必要时 `profile` 仅用于主 App UI，不写入 Widget 数据，桌面组件默认不显示真实姓名。

### 5.2 与排课器方案隔离

`/schedule` 排课器是用户自建/选课方案，属于另一套本地 + 云同步模型。

P0 Widget **不得**：

- 从 `ScheduleStore` 自动选一个 plan 当作官方课表；
- 因 #714 的 revision/CAS 冲突影响桌面官方课表；
- 把官方课表写入排课器。

P2 可增加“数据源：官方课表 / 某排课方案”的高级配置，但必须在 Widget 配置中显式选择。

## 6. Offline-First Architecture

### 6.1 总体原则

推荐数据流：

```text
YourTJ / campus API
        │
        │ 用户在 App 内手动刷新
        ▼
CampusRepository
        │
        ▼
Campus Persistent Cache (#759)
  ├─ calendar
  ├─ timetable
  ├─ today
  └─ metadata(revision/account/site/fetchedAt)
        │
        │ 同事务/同一成功提交后生成
        ▼
ScheduleWidgetProjection (纯本地、最小字段)
        │
        ├──────── Android shared storage ───────► AppWidget/Glance
        │
        └──────── iOS App Group ────────────────► WidgetKit
```

Widget 原生层绝不访问网络，只读取 `ScheduleWidgetProjection`。

### 6.2 为什么要单独做 Projection

不能让原生 Widget 直接读整个 Drift 数据库或序列化完整 `CampusDataset`，原因：

- iOS Widget Extension 访问 Flutter Drift 数据库的路径、锁和 schema 演进复杂；
- Android 原生层直接复刻 Drift schema 会增加跨语言耦合；
- Widget 实际只需要少量字段；
- 最小投影更容易做原子覆盖、版本化、账号隔离和隐私审查。

建议共享投影采用一个**原子 JSON 文档**，写入后通过平台共享存储读取。

推荐 schema：

```json
{
  "schemaVersion": 2,
  "identity": {
    "siteKey": "prod",
    "accountScope": "<opaque-local-account-scope>",
    "bindingRevision": "<opaque-revision>"
  },
  "generatedAt": "2026-09-22T18:00:00+08:00",
  "schoolDate": "2026-09-22",
  "timezone": "Asia/Shanghai",
  "semester": {
    "id": "...",
    "week": 4
  },
  "sectionTimes": [
    {"section":1,"start":"08:00","end":"08:45"}
  ],
  "days": [
    {
      "date": "2026-09-22",
      "week": 4,
      "source": "server-adjusted",
      "kind": "makeup",
      "adjustmentLabel": "教学调整",
      "courses": [
        {
          "stableId": "...",
          "name": "高等数学",
          "teacher": "张老师",
          "room": "教室",
          "campus": "四平路",
          "startSection": 1,
          "endSection": 2,
          "startAt": "2026-09-22T08:00:00+08:00",
          "endAt": "2026-09-22T09:35:00+08:00",
          "colorSlot": 3
        }
      ]
    },
    {
      "date": "2026-09-23",
      "week": 4,
      "source": "local-resolved",
      "kind": "none",
      "courses": []
    }
  ]
}
```

说明：

- `accountScope` 使用本地不可逆/不具展示意义的 scope，不写用户名、姓名、学号；
- `bindingRevision` 仅用于一致性判断；
- `colorSlot` 直接复用现有 8 槽稳定 hash；
- `sectionTimes` 优先使用已缓存的服务器覆盖，缺失时使用 App 内建 11/12 节默认表；
- `days[0]` 必须采用服务端已经应用调休规则的 `today`，本地周课表不得覆盖；
- 成功刷新时从 `timetable + calendar + calendar-rules` 生成从当日起八天的紧凑窗口；原生端按 `Asia/Shanghai` 选择当前自然日与下一自然日；
- calendar-rules、校历或课表无法可靠解析时仍提交服务端当日结果，未来日写为 `source/kind = unknown`，不得“猜”普通教学周；
- schema 1 继续可读并安全降级为单日；可选文本在 Dart、Android、iOS 三层清洗，JSON null、字面量 `null`/`undefined` 和纯空白均不得显示。

### 6.3 原子写入

P0：

- 先在临时 key/file 生成完整 Projection；
- 校验 schema / date / identity；
- 原子替换正式值；
- 替换完成后才调用 Android/iOS reload；
- Widget 始终只读“最后一个完整版本”，不能读到半写状态。

## 7. Refresh & State Model

### 7.1 数据刷新原则

**网络数据刷新：只由主 App 明确触发。**

P0 触发：

1. 用户在校园页下拉刷新；
2. 用户点击刷新按钮；
3. 首次绑定完成后主 App 成功获取校园数据；
4. 用户重新授权后成功获取新数据。

不触发：

- Widget 自己打开网络；
- WorkManager 周期拉校园 API；
- WidgetKit TimelineProvider 拉 API；
- Android 每 15/30 分钟轮询服务端。

### 7.2 Widget UI 状态变化

课程状态是可由本地时间预知的，因此不需要联网。

状态机：

```text
before-first
  -> upcoming
  -> in-class
  -> break / upcoming
  -> in-class
  -> finished-for-today
  -> next-day
```

关键时间点：

- 每门课开始；
- 每门课结束；
- 下一门课倒计时文案的离散变化点；
- 上海午夜；
- 已知调休日边界。

**不要做每分钟原生重绘才能成立的倒计时。**

推荐：

- UI 显示“10:00 开始 · 还有 25 分钟”，可使用系统相对时间组件/文本能力时优先使用系统；
- 若平台无法保证连续倒计时，则用 5/10/15 分钟粒度的 Timeline/Alarm Entry；
- 精确到秒不是产品目标。

### 7.3 iOS Timeline

TimelineProvider 一次生成：

- 当前 entry；
- 当天后续每个课程 start/end 的 entry；
- 必要的倒计时文案变化 entry；
- 上海午夜 entry；
- policy 使用系统推荐的 after / atEnd 方式；
- App 写入新 Projection 后调用 `WidgetCenter.shared.reloadTimelines(ofKind:)`。

即使 iOS 不立即重新唤醒 App，已有 Timeline 仍可在课程开始/结束时切换。

### 7.4 Android 更新

P0 推荐：

- 数据变化：主 App 调 `HomeWidget.updateWidget` / AppWidgetManager notify；
- 日期边界：一次性 Alarm / 系统日期广播 + AppWidgetProvider；
- 重启：BootReceiver 只负责重建本地更新计划，不联网；
- 尺寸变化：`onAppWidgetOptionsChanged` 重新渲染；
- 避免把 `updatePeriodMillis` 当作主要更新机制。

WorkManager 可以作为**低优先级自愈**，例如一天 1 次检查共享投影是否需要重绘，但不能承担课表获取和准点切换的正确性。

## 8. Widget Information Architecture

首期提供两个 Widget kind（Android/iOS 命名保持一致）：

### 8.1 “下一节课” Widget

定位：最小、最高频、最适合手机主屏幕。

#### Small / Android 2x1 / iOS systemSmall

显示：

- 状态标签：正在上课 / 下一节 / 今日课程已结束 / 今天暂无课程安排；
- 课程名（最多 2 行）；
- 开始时间或“进行至 09:35”；
- 地点（存在时）；
- 一个小型课程色条/色点。

不显示：

- 教师默认隐藏；
- 教学周；
- 完整周课表；
- 用户姓名。

点击：打开 App `/campus?tab=today&focus=<stableId>`。

### 8.2 “今日课表” Widget

#### Medium / Android 4x2 compact / iOS systemMedium

显示：

- 顶部：周几 / 第 N 周 / 日期；
- 当前或下一节突出；
- 今天最多显示可完整容纳的 3 门课程；
- 每项按课程名、地点与教师、时间三层展示；
- Android 默认尺寸为 4x3，显示今天与明天双栏；缩小到 150dp 以下时切换为紧凑的今日列表；
- 今日与明日列表只显示教室和教师，不展示校区；地点教师行使用次级灰度，时间行使用稍大字号和中等字重；
- 当前课程使用主色边框/背景状态；
- 超出时显示“还有 N 门课程”。

#### Large / Android 4x4 / iOS systemLarge

显示：

- “今天｜明天”双栏，两侧可独立显示课程或空状态；
- 每门课程按课程名、地点与教师、时间三层展示；
- 课程稳定色槽；
- 调休/放假标签；
- 空间不足时只显示可完整容纳的课程，并以“还有 N 门课程”收尾；
- 底部轻量状态：“更新于 17:42”。

课程名允许合理两行；核心课程信息不使用单行截断或省略号。次级字段以整字段省略或换行处理，不显示半截字符串。

Large 不显示完整 7 天网格，理由：

- 小组件的目标是 glanceability；
- 7 列周网格在手机桌面字号过小；
- YourTJ App 内已有完整周课表。

## 9. Visual Design

### 9.1 与现有 YourTJ 对齐

- 复用现有 8 槽课程稳定色；
- 浅/深色分别沿用 Web/Flutter 课程色语义；
- 大圆角卡片，但遵循平台 Widget 容器安全边距；
- 表面使用半透明层次与细微高光；Android 12+ 以壁纸动态色板染色、约 90% 不透明度显示，iOS 使用系统原生材质；文字对比度优先于透明效果；
- 不使用紫色渐变；
- 当前课程强调使用品牌主色/高对比描边；
- 每日标题按本地化短日期、教学周、星期顺序展示（如“9月23日 · 第4周 · 星期三”），星期依据课表日期和 `Asia/Shanghai` 时区计算；
- 在现有标题行末尾放置小尺寸 YourTJ 图形标记，浅色使用品牌蓝、深色使用反白版本，不额外占用课程内容行；
- Android 大尺寸课表保留手势滚动，不显示滚动条指示器；
- 字体以系统字体为主，避免 Widget Extension 自带字体导致包体和加载问题；
- 图标仅使用平台原生/SF Symbols/Android Vector 或已内置资产。

### 9.2 Android 12+ 动态色

- 表面使用系统壁纸派生的中性色背景与低比例主色染色；
- 课程色槽仍保留，避免课程身份色被动态色完全覆盖；
- 低于 Android 12 回退 YourTJ 固定主题。

### 9.3 iOS Rendering Mode

- 支持 light/dark；
- 考虑 tinted/accented rendering，核心信息不能依赖背景彩色块才能识别；
- colorSlot 同时对应一条形状/位置语义，保证被系统染色后仍可区分。

## 10. Empty / Error / Privacy States

必须定义以下状态：

1. **未绑定校园**
   文案：“绑定同济账号后显示课表”
   点击进入校园绑定页。

2. **已绑定但从未成功获取课表**
   文案：“打开 YourTJ 获取课表”
   不在 Widget 内尝试联网。

3. **有离线快照，当前断网**
   正常显示课程；底部仅在数据超过阈值时显示“离线数据”。

4. **快照较旧**
   当 `generatedAt` > 7 天未更新或学期边界异常：显示“课表可能已更新”，支持文案为“打开 YourTJ 刷新课表”。
   仍保留已有课程，不直接清空。

5. **解绑 / 登出 / 切换账号 / 切换站点**
   立即清空 Projection + reload Widget；绝不能保留上一身份内容。

6. **学校授权失效**
   若主 App 已确认授权失效：清空或降级为“需要重新授权”。
   若只是暂时网络错误：保留离线快照，不清空。

7. **今日无课 / 放假**
   正常正向状态，不使用错误样式。

8. **调休**
   显示“调休”标签和管理员规则名称（若已知），课程以服务端今日结果为准。

## 11. Battery & Background Policy

### 11.1 Android 通用

产品原则：**默认情况下不要求用户关闭省电优化。**

原因：

- 本功能不依赖后台联网；
- 桌面 Widget 本身由 Launcher 托管；
- 课程 start/end 的可预知状态由本地 alarm / widget update 计划实现；
- 即使更新触发被延迟，用户仍能看到已有快照。

只有当设备实测出现“跨午夜不切日 / 系统长期不执行 Widget 更新”等问题时，才在设置页提供“桌面小组件刷新诊断”。

### 11.2 小米 / Redmi / HyperOS / MIUI

官方支持文档明确存在：

- Background autostart；
- Battery saver / No restrictions；
- 后台锁定。

YourTJ 不应首次启用 Widget 就强制用户全部打开。

P1 “刷新诊断”可按机型提供可复制路径：

- 设置 → 应用 → 权限 → 后台自启动；
- 设置 → 电池 → App battery saver → YourTJ → 无限制（具体文案因系统版本变化）；
- 最近任务锁定仅作为故障排查建议。

### 11.3 华为 / HarmonyOS / EMUI

官方支持文档明确存在：

- App launch → 手动管理；
- Auto-launch / Secondary launch / Run in background；
- Battery optimization → Don't allow；
- 最近任务锁定。

同样仅在诊断页按需提示，不作为 P0 使用门槛。

### 11.4 OPPO / vivo / 荣耀

不同系统版本设置项命名变化较大。

P0：

- 不硬编码“一定存在”的菜单路径；
- 在诊断页以“允许后台活动 / 自启动 / 不限制电量 / 最近任务锁定”四类概念说明；
- 机型可识别时提供品牌文案；
- 文案应标记“路径可能因系统版本不同而变化”。

### 11.5 iOS

- 不要求用户为 Widget 打开 Background App Refresh 才能显示本地 Timeline；
- 用户从 App Switcher 强制退出 App 后，不承诺主 App 能后台获取新内容；
- Widget 仍展示 App Group 中最后一次成功快照；
- 新数据由用户再次打开 App 并手动刷新后写入；
- 使用 WidgetKit Timeline，而不是后台保活。

## 12. Flutter / Native Technical Plan

### 12.1 Flutter package

推荐新增：

```text
apps/mobile/packages/forum_app/lib/src/campus_widget/
  schedule_widget_projection.dart
  schedule_widget_projection_builder.dart
  schedule_widget_bridge.dart
  schedule_widget_settings.dart
```

职责：

- 从 #759 的持久化 campus cache 读取已确认数据；
- 构建最小 Projection；
- 根据身份 epoch / bindingRevision 校验；
- 写入平台共享存储；
- 请求原生 Widget reload；
- 处理 logout / unbind / site change 的清除。

### 12.2 依赖建议

首选：`home_widget`。

原因：

- Android/iOS 双平台；
- 提供 Flutter → 原生共享数据、Widget 更新、点击 Deep Link 桥；
- 社区成熟度较高；
- 仍允许两端使用原生布局，不强迫把 Flutter Widget 截图当主实现；
- 新版本已支持 generated widget 能力，但本项目 P0 建议保持原生端显式实现，便于控制平台行为。

不建议把较新的 iOS-only `flutter_home_widget` 作为主依赖，因为 P0 需要 Android + iOS 同一桥接层。

### 12.3 Android 实现

建议二选一：

**A. Jetpack Glance（优先）**

优点：

- Compose 风格；
- 响应式尺寸支持较自然；
- Android 12+ 更易做 Material 3/动态色；
- 结构可测试性比大型 RemoteViews XML 更好。

缺点：

- 仍需 Kotlin 原生；
- 需要确认当前 Flutter Android Gradle/Kotlin 配置兼容性。

**B. RemoteViews**

优点：

- 最成熟，兼容范围清晰；
- 对 2x1 / 4x2 / 4x4 课表列表足够。

缺点：

- 响应式和复杂列表写法更笨重。

P0 决策建议：若现有 minSdk 与依赖允许，使用 **Glance**；否则 RemoteViews。不要为了“纯 Flutter”而把整个 Widget 渲染成位图。

Android 组件：

```text
android/app/src/main/kotlin/.../widget/
  NextClassWidget.kt
  TodayScheduleWidget.kt
  ScheduleWidgetDataStore.kt
  ScheduleWidgetReceiver.kt
  WidgetUpdateScheduler.kt
  BootReceiver.kt
```

### 12.4 iOS 实现

新增 Widget Extension：

```text
ios/ScheduleWidgets/
  ScheduleWidgetsBundle.swift
  NextClassWidget.swift
  TodayScheduleWidget.swift
  ScheduleTimelineProvider.swift
  ScheduleWidgetModels.swift
  ScheduleWidgetStore.swift
```

- SwiftUI + WidgetKit；
- App 与 Extension 共享 App Group；
- 只从 App Group 读取 JSON Projection；
- TimelineProvider 根据课表生成时间线；
- 点击用统一 deep link 回 Flutter；
- 不在 Extension 中发网络请求。

### 12.5 Deep Link

统一：

- `yourtj://campus/today`
- `yourtj://campus/course/<stableId>?date=YYYY-MM-DD`

如果课程详情没有独立路由，则 P0 只跳今日页并高亮/滚动对应课程。

## 13. #759 Integration Contract

桌面 Widget PR 不应先于 #759 私自落盘校园数据。

#759 至少需要向本功能提供：

1. 持久化白名单中包含 `calendar / timetable / today`；
2. 账号、site、bindingRevision 隔离；
3. logout / unbind / switch account / switch site 清理；
4. 手动刷新成功后有可监听的“持久快照已提交”事件；
5. 暂时网络失败不删除最后成功数据；
6. 上海午夜后的陈旧 `today` 不再当作“今日真值”；
7. schema 可版本迁移；
8. 能区分“没有数据”和“数据为空（今日无课）”。

Widget 只在上述持久化提交成功后构建 Projection。

## 14. User Stories

### US-01 添加下一节课 Widget（P0）

As a 已绑定校园且有课表的学生，I want 在桌面看到下一节课 so that 我无需打开 App 即可知道接下来去哪上课。

Acceptance:

- Given 已有有效本地课表快照
- When 添加“下一节课” Widget
- Then 立即显示当前/下一节课程，不发网络请求。

### US-02 离线查看（P0）

As a 处于弱网/断网环境的学生，I want Widget 仍显示已获取的课程 so that 校园网络异常时课表仍可用。

Acceptance:

- Given App 已成功获取过课表
- When 断网并杀掉 App
- Then Widget 仍显示最后成功快照。

### US-03 课程状态自动推进（P0）

As a 学生，I want Widget 随时间从“下一节”切换到“正在上课/下一节/今日结束” so that 信息始终符合当前时段。

Acceptance:

- Given 当日 10:00–11:35 有课程
- When 系统时间跨过 10:00
- Then Widget 不依赖网络即可切换到“正在上课”。
- When 系统时间跨过 11:35
- Then Widget 切换到下一课程或“今日已结束”。

### US-04 手动刷新同步（P0）

As a 用户，I want 在 App 手动刷新后桌面 Widget 更新 so that 调课或学校课表变化能反映到桌面。

Acceptance:

- Given Widget 正在显示旧 Projection
- When App 校园页手动刷新成功并完成持久化
- Then 新 Projection 原子覆盖旧值并请求系统 reload。

### US-05 身份隔离（P0）

As a 多账号/切站点用户，I want Widget 不显示上一账号课表 so that 私密课程信息不串号。

Acceptance:

- Given A 账号已生成 Widget 数据
- When logout / 切换到 B / 解绑
- Then Widget 共享存储立即清除并切到未配置状态。

### US-06 调休（P0）

As a 学生，I want Widget 与 App 今日页使用同一调休结果 so that 节假日和补课不会互相矛盾。

Acceptance:

- Given 服务端 `today` 已应用管理员调休规则
- When 生成 Projection
- Then Widget 使用该结果而不是自行按 weekday 猜测。

## 15. Requirements (MoSCoW)

### P0 Must Have

- #759 提供持久化官方课表底座；
- Next Class + Today Schedule 两个 Widget kind；
- Android + iOS；
- 全离线读取 Projection；
- App 手动刷新后主动 reload；
- start/end/跨日状态本地时间推进；
- 账号/site/binding 隔离与清理；
- Light/Dark；
- Small/Medium，Android 至少 2x1/4x2；iOS systemSmall/systemMedium；
- 点击打开校园今日页；
- 调休结果一致；
- 无课、无缓存、未绑定、授权失效、旧数据状态；
- 自动化测试覆盖 Projection、状态机、身份清理和跨午夜。

### P1 Should Have

- Android Large / iOS systemLarge；
- Widget 配置：显示教师、显示校园、紧凑/舒展；
- Android 12+ 动态色；
- 国产 Android “刷新诊断”页；
- 重启后恢复本地刷新计划；
- stale badge 与上次刷新时间；
- 大字体/无障碍增强；
- iOS tinted rendering 适配。

### P2 Could Have

- Widget 数据源可选排课器 plan；
- iOS Lock Screen accessory；
- Android 锁屏/快捷状态；
- Apple Watch / Wear OS；
- Live Activity / Dynamic Island 上课状态；
- 交互式切换今天/明天。

### Won't Have (本期)

- Widget 后台网络同步；
- Widget 修改课程；
- 全周 7 列网格；
- 成绩、消息正文；
- 秒级倒计时；
- 强制用户申请“忽略电池优化”。

## 16. Acceptance Criteria Matrix

| 场景 | 预期 |
|---|---|
| 首次安装、未绑定 | Widget 显示绑定引导，不联网 |
| 已绑定但未拉过课表 | 显示“打开 App 获取一次课表” |
| 有缓存 + App 被杀 | Widget 正常可读 |
| 有缓存 + 飞行模式 | Widget 正常可读 |
| 课程开始/结束 | 本地自动切换状态 |
| 上海午夜 | 进入新日期状态；没有可验证新日数据时不把昨日 today 当今日 |
| 今日无课 | 正常“今日无课” |
| 法定假日 | 显示假日/无课 |
| 调休补课 | 与 App 今日页一致 |
| 手动刷新成功 | Projection 更新并 reload |
| 手动刷新失败 | 保留最后成功 Projection，不清空 |
| logout | 立即清除 |
| 切换账号 | 不出现上一账号内容 |
| 解绑 | 清除并显示重新绑定 |
| App 更新 schema | 旧 Projection 可迁移或安全降级 |
| Android 重启 | Widget 仍能读快照，并恢复必要更新计划 |
| Android Doze | 不依赖联网，已有数据不消失 |
| iOS 低电量/后台受限 | 已生成 Timeline 仍可展示 |
| 深色模式 | 对比度可读 |
| Widget 缩放 | Android 响应式布局不溢出 |
| 字号放大 | 关键课程名/时间不被完全裁掉 |

## 17. Testing Strategy

### 17.1 Dart Unit Tests

- Projection builder；
- `CampusEvent -> WidgetCourse` 映射；
- 稳定 `colorSlot`；
- section time fallback；
- 当前/下一节状态机；
- 上海时区；
- 跨午夜；
- 空课表；
- 调休 today 优先；
- stale 判断；
- identity mismatch；
- schema migration。

### 17.2 Flutter Integration Tests

- #759 持久层提交后生成 Projection；
- 手动 refresh 成功触发 bridge；
- refresh 失败保留旧 Projection；
- logout/unbind/site switch 清理；
- deep link 返回校园今日页。

### 17.3 Android Tests

- Small/Medium/Large snapshot；
- Widget picker preview across API 24–30 (`previewImage`), API 31–34 (`previewLayout`) and API 35+ (generated Glance preview), checking the default 4×3 proportions and both color schemes；
- API 26、31、当前 target API；
- resize；
- reboot receiver；
- doze 模拟；
- locale / timezone；
- dark mode；
- Android 12 dynamic color（P1）；
- 小米/华为至少各一台真机验证，不把后台权限作为正常路径前提。

### 17.4 iOS Tests

- systemSmall/systemMedium/systemLarge；
- Widget gallery preview；
- App Group 读写；
- Timeline entry 排序；
- WidgetCenter reload；
- cold start deep link；
- dark / tinted；
- App 被终止情况下读取最后快照；
- iPhone 真机至少覆盖当前最低支持 iOS 与最新 iOS。

## 18. Metrics

本功能不应为采集数据破坏校园隐私边界。

建议只记录**不含课程内容**的匿名功能事件，且遵循校园页现有遥测策略；如果校园域仍禁止 Umami，则在本地调试/用户反馈阶段评估，不新增生产遥测。

可选指标：

- Widget 功能设置页打开次数；
- “请求添加 Widget”按钮点击（若系统支持）；
- Projection 生成成功/失败的本地诊断计数；
- reload 请求失败率（不含课程内容）。

成功标准（发布后 2–4 周，前提是允许采集）：

- 已启用 Widget 的用户中，Projection 生成错误率 < 1%；
- 身份串号/解绑后仍显示旧课表：0 个已确认缺陷；
- 课程 start/end 本地状态推进错误：0 个 P0；
- Widget 导致的后台网络请求：0。

## 19. Rollout Plan

### Phase 0 — #759 离线底座

- 更新校园数据持久化决策；
- Drift 表/schema；
- 手动刷新语义；
- identity 生命周期；
- cache migration/clear。

### Phase 1 — Bridge + Next Class

- Projection builder；
- home_widget bridge；
- Android Small / iOS systemSmall；
- deep link；
- 清理语义；
- 基础测试。

### Phase 2 — Today Schedule

- Medium；
- Timeline/Alarm；
- 调休；
- stale 状态；
- dark mode；
- 真机测试。

### Phase 3 — Large + OEM polish

- Large；
- Android 12+ 动态色；
- 国产安卓诊断页；
- iOS tinted/accessibility；
- 发布文档。

## 20. Risks & Mitigations

### Risk A：#759 最终不批准校园数据落盘

已解决：维护者批准 #759 白名单数据设备持久化，并由 [MADR 0035](../decisions/0035-campus-device-snapshot-and-schedule-widgets.md) 取代 MADR 0033。Widget 只从同一 Drift 快照派生 Projection，没有另建校园数据旁路。

### Risk B：旧课表误导用户

缓解：

- 显示 stale 状态；
- 学期/日期边界严格校验；
- 不因网络失败删除最后数据；
- 但跨学期明显过期时提升刷新提示权重。

### Risk C：调休规则更新但用户没刷新

由于产品明确选择“获取课表后完全离线”，这是可接受边界。

缓解：

- 显示上次刷新时间；
- App 进入校园页时提示数据较旧，但不强制自动请求；
- 用户手动刷新后立即同步 Widget。

### Risk D：OEM 后台限制

通过“预计算本地状态 + Launcher/WidgetKit 托管展示”把影响降到最低。OEM 权限仅作为诊断，而不是功能成立条件。

### Risk E：共享存储泄漏

- 最小 Projection；
- 不存姓名/学号/token；
- identity scope；
- logout/unbind 原子清空；
- iOS App Group 仅本 App + Widget Extension；
- Android storage 使用 app-private / home_widget 推荐共享机制，不暴露 exported provider。

## 21. Security & Privacy Checklist

- [x] Widget Projection 不含 token、cookie、学号、邮箱、姓名；
- [x] 不含成绩、消息正文；
- [x] 不含完整 CampusDataset 原始响应；
- [x] 课程名、教师、教室属于用户主动要求在桌面展示的私密信息；
- [x] 首次启用 Widget 时明确提醒“课表会显示在系统桌面，解锁设备的人可能看到”；
- [x] 可在 App 设置中一键“清除桌面课表数据”；
- [x] logout / unbind / switch account / switch site 必须清除；
- [x] iOS Device Lock 场景已评估：P0 不提供锁屏 accessory，Projection 保持最小且无身份/凭据；系统可能在锁屏时保留桌面快照，因此以首次隐私提示和一键清除为边界。若未来增加锁屏 Widget，须单独复核 App Group file-protection 与占位策略；
- [x] Android 不声明不必要的后台/精确闹钟权限；当前仅使用可延迟的一次性本地更新，不申请 exact alarm。

## 22. Product Settings

建议入口：

`我的 → 设置 → 桌面小组件`

内容：

- 当前状态：已准备 / 无课表 / 需要重新授权；
- “刷新小组件数据”（只从本地持久化快照重建，不联网）；
- “打开系统小组件选择器 / 添加到桌面”（平台支持时）；
- 隐私提示；
- App 设置 → 外观：小组件背景透明度 0%–15%，默认 9%，仅调整背景层；
- P1：显示教师、显示校园、倒计时样式；
- P1：刷新诊断；
- “清除桌面课表数据”。

不要在这个页面再实现一套“同步学校课表”按钮；真正网络刷新仍在校园页手动完成。

## 23. Open Questions

### Resolved for this implementation

1. **#759 持久化范围**：包含 `profile / calendar / timetable / today`；`profile` 仅供登录后私密页面，Widget Projection 不含姓名。

2. **Drift 边界**：复用现有数据库实例，新增单文档 `campus_snapshots` 表，按 site/account 分区并携带 bindingRevision。

3. **iOS 最低版本**：Runner 保持 iOS 13；Widget Extension 从 WidgetKit 可用的 iOS 14 起提供 small/medium/large，iOS 16 accented 与 iOS 17 container background 使用 availability guard。

4. **Android 兼容性**：实测当前 Flutter minSdk 24、AGP 9.0.1、Kotlin 2.3.20 可编译 Glance 1.2.0，因此采用 Glance。

### Non-blocking

5. 姓名问候：不支持。

6. Large 显示教师/校区/教室；Small 隐藏教师，Medium 保持紧凑。

7. stale 阈值：7 天；跨上海日期先进入需要刷新状态。

8. 锁屏隐私：P0 不提供锁屏 accessory；后续如增加该 family，单独进行 data-protection 评审。

## 24. Recommended Engineering Decision

**推荐主方案：`#759 Drift 持久化 → Dart 生成最小 Widget Projection → home_widget 负责 Flutter/原生桥 → Android Glance（不兼容则 RemoteViews） + iOS WidgetKit/SwiftUI → 本地 Timeline/Alarm 推进状态。**

这个方案的关键价值：

- 离线是默认能力，不是 fallback；
- 不依赖后台联网、推送或常驻进程；
- 对国产 Android 杀后台天然更鲁棒；
- 不增加服务器负载；
- 不复制课表业务模型；
- 不把 Widget 和 #714 排课器云同步耦合；
- 未来能自然扩展锁屏 Widget、Watch、Live Activity。

## 25. Research Sources

- YourTJ Hub issue #759: https://github.com/YourTongji/YourTJ-Hub/issues/759
- YourTJ Hub issue #714: https://github.com/YourTongji/YourTJ-Hub/issues/714
- TJ-Class-Schedule: https://github.com/qp338113/TJ-Class-Schedule
- ClassWidget: https://github.com/karbburn/ClassWidget
- UntisPlus: https://github.com/ninocss/UntisPlus
- VITTY: https://github.com/GDGVIT/vitty-app
- BSUIR Schedule: https://github.com/vazonhub/bsuir-schedule
- home_widget: https://github.com/ABausG/home_widget
- home_widget docs: https://docs.page/abausg/home_widget
- Android App Widgets: https://developer.android.com/develop/ui/views/appwidgets
- Android Widget layouts: https://developer.android.com/develop/ui/views/appwidgets/layouts
- Apple WidgetKit: https://developer.apple.com/documentation/widgetkit
- Apple Keeping a widget up to date: https://developer.apple.com/documentation/widgetkit/keeping-a-widget-up-to-date
- Xiaomi background autostart: https://www.mi.com/global/support/faq/details/KA-507611/
- Huawei background apps: https://consumer.huawei.com/en/support/content/en-us00428704/
- Apple Background App Refresh: https://support.apple.com/118408
