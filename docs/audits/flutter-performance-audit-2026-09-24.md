# YourTJ Flutter App 综合性能审计报告

> 审计日期：2026-09-24
> 当前审计 HEAD：`7a564466275ecebce2cc0f3b4cc1c362fb90aced`（与审计时 `origin/dev` 一致）
> 旧报告基线：`91c0c864`
> 审计范围：`apps/mobile` 全部 Flutter/Dart 业务源码、Android/iOS 平台工程、依赖/资源/构建配置；必要处追到 Go 后端与 API contract 以核实真实数据契约
> 审计性质：只审计，不修改生产代码，不提交，不推送

## 0. 执行环境与模型说明

用户原要求在 CachyOS 上委派 Codex 使用 `gpt-6-luna`、`xhigh`、Fast 完成审计。实际 ACP 会话在调用 `gpt-6-luna` 时返回 HTTP 400：`The 'gpt-6-luna' model is not supported when using Codex with a ChatGPT account.`，因此 **不能声称本报告由 6 Luna 完成**。

后续全树扫描、Ponytail 审计与工具验证由 Codex `gpt-5.6-sol` 兜底继续，思考强度 `xhigh`，用户级 Fast 映射为 `service_tier=priority`。该 ACP run 在完成静态扫描和工具验证后以 `ACP_REMOTE_ERROR / Internal error` 中断，未自行写出最终 Markdown；本报告基于该 run 已固化到 worktree 的 `findings.md`、`progress.md`、实际命令输出，以及随后对关键源码行的复核整理完成。

这项执行差异只影响“由哪个模型生成文字”，不影响已经在本机实际执行的代码扫描、Flutter analyzer、构建与 analyze-size 结果。

---

# 1. Executive Summary

当前 App 的性能问题不是单点，而是四条链路叠加：

1. **启动窗口并发工作过多，但 4–8 秒“黑屏”不能直接归因于 Home 网络或图片。** `main()` 只做 `runApp`，Home API 和 Feed 图片都发生在 Flutter Widget 树启动之后。真正必须拆分测量的是 native launch / Flutter first rasterized frame / skeleton / Home data / first card / first media。Android 深色 LaunchTheme 本身就是黑色，可能把“合法启动表面”视觉上放大成黑屏，但改背景并不等于性能优化。
2. **Feed 图片链路是当前最明确、最值得优先处理的渲染热点。** 当前 Dart model、OpenAPI 与 Go payload 只提供原始图片 URL，没有 `thumbnailUrl`、`width`、`height`；服务端也不生成导数。`GfTopicCard` 先用 `NetworkImage.resolve` 解码图片拿比例，随后再以没有 `cacheWidth/cacheHeight` 的 `Image.network` 显示。Flutter ImageCache 通常可避免同 URL 的第二次网络下载，因此旧报告“必定下载两遍”需要修正；但全尺寸下载/解码、比例晚到导致的 layout change、像素内存占用是真问题。
3. **网络层最大的热路径浪费是每请求读取安全存储 Token，而不是缺一个“全局高级调度器”。** `GfApiClient` 的 `onRequest` 对几乎所有未显式带 Authorization 的请求执行 `await tokenStorage.read()`；同时 Shell unread、push 等路径还会在进入 repository 前再次读 token。应该先做 session-scoped 内存 token hot path、局部 single-flight/cancel/SWR，再考虑是否存在足够证据抽象共享 primitive。
4. **当前已有不少正确的生命周期保护，旧报告不能再笼统说“缺少缓存/取消”。** Home/Search/Topic/Campus 多处已经使用 generation/sequence/offlineCacheEpoch/CancelToken，Campus 还有 5 分钟前台内存缓存与 binding revision fence。真正缺口集中在：Home 冷启动首屏没有 SWR、Notifications 切筛选无 generation/cancel、Messages 轮询缺 in-flight guard、Schedule 三个会话元数据请求串行、Search/Course Detail 已加载大量结果后仍以非惰性 `Column`/children 构建。
5. **Campus 的 `status()` 不是偶然浪费，而是当前隐私契约。** 缓存明确规定“只有 fresh status 验证 binding 后才能恢复”，切后台、换账号、绑定 revision 改变都会清空。因此不能直接删掉 status gate。可以优化 status 本身、status 后的数据恢复/并行、减少重复元数据；若要短 lease，必须作为产品/隐私语义变更单独决策。
6. **Topic 详情存在一个旧报告没突出的问题：网络成功后先 await Drift 离线缓存写入，再发布 UI。** `_cachePut()` 的序列化/SQLite I/O 被放进首次有用内容路径，应改为先安全发布内存态，再在 epoch/generation fence 下异步缓存。
7. **包体积已经做了真实实验，但只能作为“参考基线”，不能冒充生产基线。** 本机 Flutter 3.47.5 / Android SDK 36 可用；`flutter analyze` 全量通过。原样 release 在 3.47.5 下因为 dev-only `integration_test` 被注册进 release Java registrant 而失败；临时排除该 dev dependency 后成功得到 arm64 `--analyze-size` 参考 AAB：`34,299,782 bytes`。本轮在 Flutter 3.44.9 下通过 release 专属 no-op shim 构建 AAB（90.1 MB，build-only key）；该构建未启用 analyze-size，仍不能作为 size baseline。生产 CI 固定 Flutter 3.44.9，且参考构建未包含生产 OEM push adapters，因此必须在生产同版本/同配置下再建立正式 size baseline。

## 本轮优先级结论

### P0：先做

- 启动分段 telemetry + 真机 profile/release benchmark；
- Token memory hot path，取消 API 热路径上的每请求 secure-storage read；
- Feed 媒体契约升级：导数缩略图 + intrinsic width/height + 目标 decode size；
- Home 第一页 SWR / cached first paint；
- Topic 详情把离线缓存写入移出 UI publish critical path；
- Android/iOS 各自建立可重复的启动基准，不再用“肉眼 4–8 秒”作为唯一指标。

### P1：随后做

- Notifications filter generation/cancel；
- Messages unread/chat polling local single-flight；
- Schedule session metadata `Future.wait` 并合并 setState；
- Search、Course Detail 长结果列表改真正 lazy sliver/list；
- Home/Search superseded GET 使用 CancelToken 取消，而不只是丢弃旧响应；
- Link Preview cover/favicon、media carousel、avatar 等小图补目标 decode size；
- Release size CI 预算、符号分离与依赖清理。

### P2：Profile 后决定

- 更细的 card-level rebuild 下沉；
- Markdown 超长内容 AST/model cache 或更深 sliver 化；
- Android Baseline Profile / Macrobenchmark；
- 全局 GET coalescing primitive（只有在更多重复调用点被证明后再抽象）；
- 是否需要磁盘级图片缓存库。

---

# 2. 审计范围、方法与证据等级

## 2.1 扫描规模

审计记录统计：

- `apps/mobile` tracked 文件：664；
- Dart 文件：390；
- `lib/` Dart：268；
- 排除 Freezed / JSON / l10n 等生成物后：231 个业务 Dart，约 52,997 行；
- Android tracked 文件：34；
- iOS tracked 文件：51；
- 资源类文件：189。

重点阅读覆盖：启动入口、Provider/Riverpod、Home/Feed、Topic、Search、Notifications、Messages、Campus、Schedule、Courses、Markdown/Link Preview、图片上传/展示、Drift/offline cache、Dio/client/interceptor、Android/iOS launch、CI/release workflow、pubspec/lock，以及 Feed 图片相关后端/API contract。

## 2.2 证据等级

本文所有结论分三类：

- **[静态确认]**：当前 HEAD 源码/配置可直接证明；
- **[合理推断]**：机制上高度可能影响性能，但没有真机 timeline 不能量化；
- **[必须实测]**：只有 profile/release 真机、FrameTiming、Perfetto/Instruments 或网络 trace 才能确认。

避免把“代码看起来复杂”直接写成“就是它导致掉帧”。

## 2.3 实际执行的验证

本机：

- Flutter `3.47.5`
- Dart `3.13.4`
- DevTools `2.60.0`
- Android SDK `36`
- 生产 release workflow 固定 Flutter `3.44.9`
- 无 Android 真机/模拟器；仅 Linux desktop device
- Linux 无 Xcode / CocoaPods，因此 iOS build/profile/analyze-size 无法执行

实际执行：

```bash
/opt/flutter/bin/flutter doctor -v
/opt/flutter/bin/flutter devices
/opt/flutter/bin/flutter pub deps --style=compact
/opt/flutter/bin/flutter pub get
/opt/flutter/bin/flutter analyze
flutter build appbundle --release --analyze-size ...
```

`flutter analyze` 结果：

```text
No issues found! (ran in 9.5s)
```

原样 release 在本机 Flutter 3.47.5 失败：dev-only `integration_test` 被 release registrant 引用，Java 编译缺失 `dev.flutter.plugins.integration_test`。临时排除该 dev dependency 后，arm64 参考 analyze-size 构建成功；临时改动已恢复。

本轮用 mise Flutter 3.44.9 再现同一问题。原因是 Flutter 生成的 release registrant 保留 dev plugin 调用，而 Gradle release classpath 已排除该 plugin。现通过 `android/app/src/release/java/.../IntegrationTestPlugin.java` 提供 release-only 空实现；dev dependency 保留，debug/profile 仍链接真实 `integration_test`。本机 3.44.9 release AAB 与 debug APK 均已成功构建。

---

# 3. 旧报告逐项复核矩阵

| 旧报告关键结论 | 当前状态 | 当前 HEAD 证据 | 修正后的判断 |
|---|---|---|---|
| `main()` 不存在长串同步初始化 | **确认** | `apps/mobile/packages/forum_app/lib/main.dart:7` 只有 `runApp(const ProviderScope(child: GfApp()))` | 4–8 秒黑屏不能简单归因于 `main()` 阻塞 |
| 首屏网络/图片能解释全部 4–8 秒黑屏 | **结论不充分** | Home `_load()` 在 Widget 初始化后；Feed media 更晚 | 必须拆 TTID/Flutter first frame/TTF Home data/first media；原生 launch 与 Flutter 内容慢不能混成一项 |
| 每 API 请求读取 secure storage token | **确认** | `gf_api_client.dart:31-39` | P0 热路径优化成立 |
| Feed 图片会“下载两遍” | **部分确认/需修正** | `gf_topic_card.dart:155` 先 resolve；`:694` 再 `Image.network` | Flutter ImageCache 可能复用网络结果，不能声称必定二次下载；但全尺寸 decode + late ratio + relayout 是确定成本 |
| Feed 没缩略图/width/height contract | **确认** | 全链路 Dart/OpenAPI/Go 扫描只有原图 URL；后端原样存储 | P0，应服务端生成导数并回传 intrinsic size |
| `cached_network_image` 未使用 | **确认** | pubspec 有 direct dep，业务 Dart 零直接引用 | `delete:` 候选；不要因为删它就顺手把 Feed 全部改用同库 |
| Home 缺第一屏 SWR | **确认** | `HomePage.initState -> _load()`；无首屏持久数据恢复 | P0，优先“已有内容先画，再静默刷新” |
| Topic 返回 Home 再刷新第一页 | **确认** | `home_page.dart:180-254` | 已先 merge return state，随后仍 GET 第一页；可做 freshness window/cancel，但要保留一致性 |
| Campus 每次进入的 status 是无意义重复 | **已证伪其“无意义”表述** | `campus_memory_cache.dart:35` 明确 fresh status 后才 restore；`campus_state.dart:119-149/202-207` | status gate 是隐私硬门；不能仅以性能理由移除 |
| Campus 缺请求取消/生命周期 | **已过时/不成立** | `CancelToken`、generation、cache generation、revision fence、foreground clear | 应保留；只优化 gate 之后的并行/元数据/请求数 |
| 整个 App 都缺请求生命周期 | **部分确认，表述过宽** | Home/Search/Topic/Campus 已有 sequence/generation/epoch；Notifications/Messages 存具体缺口 | 做局部补洞，不造 GlobalNetworkScheduler |
| 大文件/大 StatefulWidget 就是 jank | **结论不充分** | 多个大页面已使用 builder/sliver/key/RepaintBoundary | 只能通过 DevTools Track builds/layout/paint 定位；当前明确非惰性热点是 Search/Course Detail 等 |
| Markdown 需要整体重写 | **不支持** | 当前已有 body memoization、主图 `cacheWidth`、link preview deferred loading | 保留现有实现，只对 cover/favicon/media decode size 和真机 layout 热点做增量优化 |
| 宿主无 Flutter SDK，无法分析包体积 | **已过时** | `/opt/flutter` 可用，已完成 analyze 与参考 AAB size analysis | 现在已有参考数字，但正式基线仍要用生产 3.44.9/完整 push 配置 |
| `sqlite3_flutter_libs` EOL 需处理 | **确认但需谨慎** | lock 中 `0.6.0+eol`，还有 `sqlcipher_flutter_libs 0.7.0+eol` | 先做依赖树/DB 构建迁移验证，不能只靠源码 import grep 直接删 native asset provider |
| Android 黑色启动表面值得调整 | **部分确认** | night `LaunchTheme`/`NormalTheme` 使用 `Theme.Black.NoTitleBar` | 可改善视觉连续性，但不能作为启动性能修复 |
| P99 <= 16.7ms 等是 Flutter 官方标准 | **需要修正** | Flutter 官方只提供帧预算/分析工具，项目目标需自行设 | 60/90/120Hz 理论预算 16.67/11.11/8.33ms；具体 P95/P99/jank 目标写成 YourTJ 工程 SLO |

---

# 4. 启动链路审计

## 4.1 当前启动关键路径

### T0：Native process / launch surface

Android：

- `values/styles.xml` 的 `LaunchTheme` 使用 Light；
- `values-night/styles.xml` 的 `LaunchTheme` / `NormalTheme` 使用 `Theme.Black.NoTitleBar`；
- 当前没有专门的 Android 12+ `values-v31` SplashScreen 定制证据。

iOS：

- `LaunchScreen.storyboard` 是静态 launch screen；
- `AppDelegate.didFinishLaunchingWithOptions` 没发现启动期同步网络/文件 I/O；
- APNs 路径和 Android JPush 初始化链不同。

**[合理推断]** 深色 Android 启动时，用户看到一段黑色 native launch surface 是可能的。它会影响“黑屏”主观观感，但如果 TTID 本身很长，换颜色只遮掩而不是解决。

### T1：Flutter entry

`apps/mobile/packages/forum_app/lib/main.dart:7`：

```dart
runApp(const ProviderScope(child: GfApp()));
```

没有 await 数据库、网络或 SDK 初始化。

### T2：GfApp build

`app.dart:28-35`：

- watch `themeModeProvider`；
- watch `siteThemeProvider`；
- watch `pushBootstrapProvider`；
- 构建 `MaterialApp.router`。

`siteThemeProvider` 自己会从 SharedPreferences 恢复并后台刷新站点主题；push bootstrap 也会进入 session/prefs/native driver 路径。

### T3：Shell + Home 同时启动

`router.dart:81-88`：

```dart
unawaited(_purgeStaleOfflineCacheOnBoot());
_pollUnread();
_unreadTimer = Timer.periodic(const Duration(seconds: 30), ...);
```

同时 Home `initState()` 会 `_load()`。

`_pollUnread()` 先 `tokenStorage.read()`，随后 repository GET 又会经过 `GfApiClient.onRequest` 再读 secure storage token，因此存在重复平台存储访问。

### T4：first useful Home content

Home 首次进入没有数据级 first-page cache：

```text
empty/skeleton
→ secure-storage/token gate
→ Home GET
→ JSON parse
→ list build
→ media URL request/decode
```

而 UpdateHost 已经正确地把自动更新检查放到 `addPostFrameCallback` 之后，说明“所有后台任务都在首帧前”并不成立。真正需要后移的是仍和首个有用 Home 内容竞争的非关键动作，而不是机械地把所有 async 都延后。

## 4.2 启动问题的真实优先级

### P0-A：先把“4–8 秒”拆成可观测指标

至少记录：

```text
T0 native process / launch start
T1 Flutter firstFrameRasterized
T2 Home skeleton/first route painted
T3 Home response received
T4 Home props parsed + first card committed
T5 first media image painted
T6 primary async content complete / fully usable
```

建议事件名：

```text
app_start_flutter_first_frame_ms
app_start_home_shell_visible_ms
app_start_home_data_ms
app_start_first_card_ms
app_start_first_media_ms
```

Android 同时用 TTID/TTFD 语义；TTFD 可以在主要异步内容完成时接入 `reportFullyDrawn` 对齐 Android startup 工具。

### P0-B：把 token read 从每请求热路径移出

当前 `gf_api_client.dart:31-39`：

```dart
final token = options.headers.containsKey('Authorization')
    ? null
    : await tokenStorage.read();
```

建议实现一个极薄的 session token holder：

```text
SecureStorage → 只在 session hydrate / login / renew / logout 时读写
Memory token   → 每个 request interceptor 直接读取
```

必须保持：

- login 成功 token 持久化后同步更新 memory；
- `New-Token` renewal 同步更新；
- 401 / logout 清 memory + storage；
- session-bound cleanup 显式 Authorization 的语义不变；
- `offlineCacheEpoch` / currentUser invalidation 不变。

**预期影响：** 减少每次 GET/POST 前的 platform channel + keystore/keychain wait，尤其是 Campus 等并行多请求场景。

### P0-C：首帧和首个 Home 内容之间只保留必要工作

建议测量后逐项后移/合并：

- unread 首次 poll 可在 first frame 后；
- theme/local prefs 恢复如果要避免闪烁可保留，但共享同一个 prefs instance/restore future，避免多个 provider 重复 `SharedPreferences.getInstance()`；
- push 若未 opt-in 或当前 session 无 token，应尽早 cheap-exit；
- update check 已经 post-frame，保留；
- 离线缓存的跨账号安全清理不能为了性能删除。

---

# 5. Feed 图片、缓存与首屏媒体

## 5.1 当前问题已被全链路确认

当前 Feed 媒体 contract：

```text
Go backend / API contract / Dart payload
→ 原图 URL
→ 无 thumbnail URL
→ 无 width/height
```

上传路径校验后原样存储；没有服务端派生尺寸。移动端选图自身会做一定限宽，但 Web 上传仍可能是更大原图，所以客户端不能假定 Feed URL 已经是“屏幕合适尺寸”。

`gf_topic_card.dart:148-169`：

```dart
_stream = NetworkImage(url).resolve(...);
_listener = ImageStreamListener((info, synchronousCall) {
  final ratio = info.image.width / info.image.height;
  setState(() => _ratio = ratio);
});
```

真正显示在 `:694-704`：

```dart
Image.network(url, fit: fit)
```

没有 `cacheWidth/cacheHeight`。

## 5.2 机制影响

### 网络

Flutter `Image.network` 默认会走 ImageCache。`NetworkImage.resolve` 与后续同 URL `Image.network` 很可能共享已解析的 image stream/cache，所以 **不应写成“必定发两个 HTTP 请求”**。

### Decode / Memory

但为了拿宽高，图片必须先可解码；之后展示仍没有目标 decode size。一个远大于屏幕的原图会以接近原始像素尺寸进入解码/缓存流程。

Flutter 官方说明图片在内存中是未压缩位图；4K RGBA 可超过 30MB。Feed 连续多张大图非常容易造成 raster/decode 压力和 ImageCache 内存波动。

### Layout

初始 `_ratio = 1.5`，真实图片 decode 后再 `setState` 替换比例：

```text
placeholder geometry
→ decode finishes
→ ratio changes
→ card relayout
→ surrounding list geometry changes
```

快速滚动时，这类晚到的 layout change 比“有没有 const”更值得优先处理。

## 5.3 P0 方案：把尺寸信息前移到服务端契约

推荐 API：

```json
{
  "url": "original-or-full-url",
  "width": 4032,
  "height": 3024,
  "variants": {
    "small": "...",
    "medium": "...",
    "large": "..."
  }
}
```

或者最小改动：

```json
{
  "url": "...",
  "thumbnailUrl": "...",
  "width": 4032,
  "height": 3024
}
```

Flutter 直接：

```text
intrinsic width/height
→ 提前确定 AspectRatio
→ 选择最接近实际 physical pixel 的 thumbnail
→ cacheWidth/cacheHeight 限制 decode
```

随后可以删除 `_observeImage()` 整套 ImageStream probe。

### 变体建议

不要用固定“所有设备 360px”这种逻辑。根据：

```text
logical card width × devicePixelRatio
```

选最近的服务端 variant，例如 360 / 720 / 1080 / 1440 档。

## 5.4 磁盘图片缓存：不是第一步

`cached_network_image` 当前 direct dep 零业务引用，可以删。

是否之后重新引入统一 disk image cache，应在下列问题处理后用数据决定：

1. 服务端是否已有合理导数；
2. decode size 是否受控；
3. 同一进程内 Flutter ImageCache 命中是否足够；
4. App 重启后首屏图片的网络字节与 T5 是否仍是主要瓶颈。

不要为了“缓存”先引入另一层不可控 cache manager。

---

# 6. Home Feed 数据与 SWR

## 6.1 当前状态

`HomePage.initState()` 恢复 feed mode 后直接 `_load()`；没有 cold-start first-page data cache。

当前已有正确保护：

- `_loadSequence` 防旧响应覆盖；
- `offlineCacheEpoch` 防跨账号结果落入新 session；
- “加载更多”有独立状态；
- 从 Topic 返回先通过 `topicReturnStatesProvider` merge 本地互动状态。

因此旧报告“Home 完全没有请求生命周期”不准确。

## 6.2 仍存在的浪费

### 冷启动

每次都从 skeleton 等 Home GET。

### Topic return

`home_page.dart:180-254` 已经先 merge 了 unseen / like / bookmark / reply / view 等状态，随后仍然无条件 GET 第一页并按 id 原位刷新。

### sort / refresh superseded request

sequence 会丢弃旧响应，但旧请求本身仍继续占网络、服务端、JSON parse。

## 6.3 P0/P1 实现

### 第一页 SWR

key：

```text
home:first-page:<sort>:<account/public-context-version>
```

流程：

```text
进入 Home
→ 有可信 cache：立即 hydrate 列表
→ 后台 GET
→ 成功原位更新
→ 失败保留旧内容并轻提示/静默
```

注意：

- 如果 Home 含账号相关字段，缓存必须绑定 `offlineCacheEpoch` / account fence；
- 公共匿名内容和 viewer-specific interaction 可拆缓存层，避免跨账号泄漏；
- TTL 不要伪装成标准，建议先用 3–10 分钟实验并以 stale correctness/流量决定。

### Topic return freshness window

返回详情后：

```text
merge local return state
→ if last_home_refresh sufficiently recent: no GET
→ else silent first-page refresh
```

建议 30–60 秒仅作为实验起点，不是固定标准。

### Cancel superseded

Home sort/refresh、Search query/scope 切换可复用现有 Dio `CancelToken`：

```text
new intent → cancel previous read → start next
```

sequence/generation 仍保留作为最终 correctness fence；cancel 是资源优化，不替代 fence。

---

# 7. 滚动 / Jank / Rebuild / Layout

## 7.1 不能归罪给 ListView 本身

Home/topic/profile/messages 多数路径已经使用 builder/sliver、稳定 key 或局部 RepaintBoundary。`GfTopicList` 采用 lazy list，方向正确。

不要做以下“优化”：

- 因为 `HomePage` 有 setState 就全面 Riverpod family 化；
- 因为某文件 2000 行就判断运行时慢；
- 未 profile 就调大/调小 `cacheExtent`；
- 为 Feed 手写 RenderObject。

## 7.2 当前明确热点

### A. Feed 原图 decode + late ratio relayout — P0

见第 5 节。

### B. Search 已加载结果是非惰性 children — P1

`search_page.dart:570-618`：外层 `ListView(children: [...])`，内部 `GfCard -> Column`，再由 `_UserRows` / `_TopicRows` 构造当前已加载全部条目。

分页越多，任何父级 rebuild 都会重新创建大量 row widgets。

建议改：

```text
CustomScrollView
SliverToBoxAdapter(headers)
SliverList(users/topics)
SliverGrid(categories)
SliverToBoxAdapter(load-more footer)
```

### C. Course Detail reviews 非惰性 — P1

`courses/detail_page.dart:443` 使用 `ListView(children: ...)`，Reviews section 在 `:568` 对 `_reviews` 全量 `for` 构建 `_ReviewRow`。分页追加后，已加载 review 全部处在一个 Column children 树中。

建议把 detail page 改为组合 sliver，至少 review 区使用 `SliverList.builder`。

### D. 页面级互动 setState — P2，先 Profile

Home 点赞/收藏等会导致页面局部重建，但当前没有真机证据表明它比图片 decode 更重。只有在 DevTools Track Widget Builds 显示大量可见卡片重建时，再下沉 per-topic state/select。

## 7.3 帧预算与验证

理论预算：

- 60Hz：16.67ms
- 90Hz：11.11ms
- 120Hz：8.33ms

这些是显示刷新间隔，不是 YourTJ 的官方合格线。

建议 YourTJ 自己设工程目标，例如：

```text
scroll jank < 1%
P95/P99 frame duration 按目标设备分档
```

必须标注为项目 SLO，而不是 Flutter 官方阈值。

联合采集：

- `SchedulerBinding.addTimingsCallback` / FrameTiming；
- DevTools Performance UI/Raster frame chart；
- Track Widget Builds / layouts / paints；
- Highlight repaints；
- memory / ImageCache 波动；
- 网络 waterfall；
- Android Perfetto 结合 image decode / GC / thread scheduling。

---

# 8. 网络、Cache、SWR、Lazy 与 Request Lifecycle

## 8.1 已有的正确基础

当前不是“从零开始”：

- Dio/client 单实例 provider；
- `CancelToken` 已用于 Campus 等路径；
- Home `_loadSequence`；
- Search `_generation`；
- Topic `_windowGeneration`；
- `offlineCacheEpoch` 跨账号 fence；
- Campus generation + cache generation + binding revision；
- Topic return local merge；
- Link Preview deferred loading。

这些都应 `keep:`。

## 8.2 P0：Token hot path

见启动节。该项是最通用、证据最直接的网络优化。

## 8.3 P1：Messages 轮询 single-flight

Shell unread：30 秒一次；消息页面还有约 15 秒轮询。审计发现轮询入口没有统一 in-flight guard，而某 receive 请求 timeout 可到 20 秒。

慢网场景可能：

```text
poll N 未结束
→ timer N+1 又发起相同读取
```

最小修复：

```dart
if (_polling) return;
_polling = true;
try { ... } finally { _polling = false; }
```

不要先建全局 priority queue。

## 8.4 P1：Notifications filter correctness + resource waste

`notifications_page.dart:52-66` 的 `_load` 没有 generation/cancel；筛选变化 `:155-156` 立即 `_load()`。

快速切：

```text
all → replies → likes
```

旧 `all` 响应可能晚到覆盖新筛选结果。

这里不是只“性能不好”，还是潜在 stale response correctness 问题。

修复：

- `_generation++`；
- capture filter + generation；
- 新筛选 cancel 旧 GET；
- apply response 前验证 filter/generation/epoch。

## 8.5 P1：Schedule 三元数据串行

`schedule_page.dart:213-252` 依次：

1. `calendars()`；
2. `sectionTimes()`；
3. `latestUpdate()`；

每个成功还单独 setState/notify。

三者互不依赖，适合：

```dart
final results = await Future.wait([...]);
if (!mounted) return;
setState(() { apply all available results; });
```

个别失败可用独立 result wrapper/try helper 保持现有“失败不阻塞”的语义。

## 8.6 Request coalescing：只做薄 primitive

当前可证明的共享需求：

- session/token hydrate；
- section time / session metadata；
- unread/polling 类 GET；
- 某些短时间重复 GET。

如果实现共享 in-flight map，只支持：

```text
GET + deterministic request key + same account/session fence
```

请求完成立即 remove。

不要合并 POST/PUT/DELETE，不要把 retry/cache/priority 全塞进一个 `GlobalNetworkScheduler`。

## 8.7 Retry policy

建议仅对幂等 GET 的 transient failure：

- connect timeout；
- socket disconnect；
- 少量 502/503/504；

使用有上限的 exponential backoff + jitter。

用户主动刷新应该可以立即发起，不必被后台 retry 队列卡住。

写操作默认不自动 retry，避免重复副作用。

---

# 9. Campus 专项

## 9.1 现有缓存比旧报告描述得更完整

`campus_memory_cache.dart`：

- TTL 5 分钟；
- 只缓存 `profile/calendar/timetable/today/messages`；
- 不缓存 credentials、grades、notice bodies；
- daily key 校验校历日期；
- binding revision 变更清空；
- background 立即 clear；
- `offlineCacheEpoch` / repository 更换会替换 cache。

这是合理的私有数据边界，应保留。

## 9.2 status gate 是明确设计

关键注释：

```dart
/// Called only after a fresh status request has verified this binding.
```

`CampusController.refresh()` 先：

```dart
final status = await repository.status(...);
final restored = cache.restore(binding?.revision);
```

`enterTab()`：

```dart
await refresh(reuseCache: true);
```

因此“返回 Campus 总有一次 status”不是 cache bug，而是当前产品/隐私契约。

## 9.3 正确优化方向

### 保持 status hard gate

不要直接“缓存 status 60 秒然后跳过”。若要引入 verified-binding lease，必须先修改产品/隐私文档并回答：

- 用户后台解绑后多久必须失效？
- 同账号不同设备/绑定 revision 改变如何发现？
- app resume 是否强制 fresh status？
- authorizationRequired 如何立即切断旧数据？

在没有这个产品决策前，本报告不推荐绕过 gate。

### status 后立即恢复 fresh memory cache

当前已经做到。进一步优化：

- status endpoint 保持最小 payload/最短服务端路径；
- 只拉 stale/missing dataset；
- tab 数据使用现有 `Future.wait`；
- 课节时间等与私有 data 无直接依赖的元数据可独立缓存/并行；
- UI 显示 non-sensitive shell/skeleton 不必等全部 dataset。

### 真正应测的 Campus 指标

```text
campus_status_rtt_ms
campus_status_to_cached_content_ms
campus_status_to_fresh_content_ms
campus_requests_per_enter
campus_requests_per_tab_switch
```

目标应是：

- fresh cache 命中时，status 之后立刻出数据；
- 同一次 verified session 下 tab 切换不重复拉 fresh dataset；
- background/account/revision 边界绝不复用私有缓存。

---

# 10. Topic / Offline Cache 专项

## 10.1 新发现：网络成功先写 Drift，再 setState

`topic_page.dart:193-205`：

```dart
if (epoch == ref.read(offlineCacheEpochProvider)) {
  await _cachePut(widget.topicId, payload.toJson());
}
...
setState(() {
  _page = AsyncValue.data(props);
});
```

`_cachePut` 最终：

```dart
await ref.read(offlineTopicCacheProvider).put(topicId, json);
```

也就是说，用户已经拿到网络结果，但必须等完整 payload 序列化/Drift 写入完成，UI 才发布。

## 10.2 P0/P1 修正

建议：

```text
network payload
→ parse/validate
→ generation/epoch check
→ publish UI immediately
→ unawaited guarded cache write
```

后台写入仍需：

- capture epoch/generation/topic id；
- 写前/写后检查 session fence；
- cache failure silent；
- logout/401 clear semantics 不变。

这是一类典型“把非必须持久化移出 critical path”的优化，收益比为了 topic page 拆十几个 widget 更明确。

---

# 11. Markdown、Link Preview、WebView/Quill 等重组件

## 11.1 现有实现值得保留

Markdown 当前已经有：

- 内容 body memoization；
- 主文图片 width-based `cacheWidth`；
- Sticker 仅需要时解析；
- Link Preview post-frame / near viewport 延迟加载；
- `Scrollable.recommendDeferredLoadingForContext()`；
- preview batch future 共享。

因此“换 Markdown 引擎”不是当前合理 P0。

## 11.2 仍可优化

### Link Preview cover/favicon

当前 cover/favicon 部分 `Image.network` 仍缺目标 decode size。

- favicon：按 16–32 logical px × DPR；
- cover：按实际 card width × DPR；
- media carousel：同理选导数/目标 decode size。

### 超长 Markdown

`MarkdownWidget(shrinkWrap: true)` 之类嵌入外层 scroll 的布局成本需要真机验证。若 DevTools 明确显示 layout 超时，再考虑：

- AST/model cache；
- post-level RepaintBoundary；
- 超长内容折叠；
- 更深的 sliver/lazy 内容组织。

## 11.3 依赖不能误删

`webview_flutter`、Quill、Markdown、Drift、`image`、HTML renderer 等都有真实功能引用。Ponytail 审计不能因为“库大”就建议删除。

---

# 12. Android 专项

## 12.1 启动

当前 night theme：

```xml
<style name="LaunchTheme" parent="@android:style/Theme.Black.NoTitleBar">
<style name="NormalTheme" parent="@android:style/Theme.Black.NoTitleBar">
```

建议：

- Android 12+ 使用现代 SplashScreen 配置；
- launch background 与 Flutter root background 做视觉连续；
- 不人为延长 splash；
- 用 TTID/TTFD/Perfetto 测真实速度。

## 12.2 Push

审计确认主进程已避免无条件 JPush auto InitProvider；真实初始化受 Dart consent/config 流程控制。这是正确的启动治理，应该保留。

## 12.3 Release shrink / ABI / symbols

项目 `build.gradle.kts` 的 release block只配置 signing，未在项目层显式设置 `minifyEnabled` / `shrinkResources`。

注意：这不等于“当前产物一定完全没 shrink”，Flutter/AGP 插件可能设置其他默认行为；正式动作应先检查 effective Gradle/R8 输出，再决定是否显式开启。

当前生产 workflow：

```bash
flutter build apk --release --split-per-abi ...
```

优点：三个 ABI 分开交付，用户不会下载所有 ABI native libs。

缺口：

- 没有持久化 `--analyze-size` baseline；
- 没有 size budget regression gate；
- 没看到 `--split-debug-info` 的 release 体积治理；
- 没有 AAB Play-like download size 的持续追踪。

## 12.4 Baseline Profiles

可以作为 P2：

- Macrobenchmark startup；
- FrameTimingMetric；
- Baseline Profile；

但必须在 Dart/图片/网络关键热路径处理后再评估收益，不能用平台 profile 掩盖 Feed 原图 decode。

---

# 13. iOS 专项

## 13.1 Launch

`AppDelegate` 没有发现首启同步网络或文件 I/O；LaunchScreen 是静态 storyboard，方向合理。

建议真机使用 Instruments / Xcode launch diagnostics 分：

```text
process launch
→ Flutter engine / first rasterized frame
→ Home useful content
```

LaunchScreen 只保持简洁、接近首屏即可，不应放进复杂逻辑。

## 13.2 Build / size

Linux 当前无法构建 iOS，因此：

- 没有本轮真实 IPA/App Store size；
- 没有 iOS `--analyze-size` 数字；
- 必须在 macOS CI 或本地 Mac 以 release 构建建立正式 baseline。

Release 工程已使用 whole-module optimization，并保留 dSYM；这是正确方向。

## 13.3 图片与内存

iOS 同样受 Flutter image decode/cache 机制影响，因此服务端导数 + intrinsic dimensions 是跨平台 P0，不是 Android-only 修复。

---

# 14. 包体积审计

## 14.1 参考 arm64 AAB

临时排除 Flutter 3.47.5 下导致 release registrant 失败的 `integration_test` dev dependency 后：

```text
AAB: 34,299,782 bytes
Flutter console: ~34.3 MB
```

Flutter 3.44.9 下另构建了 90.1 MB AAB，但未使用 `--analyze-size`，不可与上述 arm64 参考数字直接比较。

**这不是生产正式基线**，原因：

- 本机 Flutter 3.47.5，生产 CI 3.44.9；
- 临时排除了 dev plugin；
- 未带生产 OEM push adapters；
- AAB 内包含约 14MB 压缩 native debug symbols 与约 2MB obfuscation metadata；
- AAB archive size 不等于终端下载/安装体积。

## 14.2 最大项（参考构建）

| 项 | 参考大小 |
|---|---:|
| `libapp.so` | 16,778,120 B |
| `libflutter.so` | 11,747,864 B |
| `classes.dex` | 4,441,536 B |
| `libsqlite3.so` | 1,732,360 B |
| `libjutils.so` | 1,525,304 B |

Dart AOT attribution 主要项：

| package | 参考 accounted size |
|---|---:|
| `flutter` | ~4 MB |
| `forum_app` | ~1 MB |
| `highlight` | ~1 MB |
| `image` | ~865 KB |
| `core` | ~640 KB |
| `flutter_quill` | ~469 KB |
| `tldts` trie | ~457 KB |

这些数字用于定位，不代表可以直接“删库省同等 MB”。

## 14.3 Tree shaking 实际有效

构建日志确认字体 tree shaking：

```text
TDesign:       559,612 → 912 B
CupertinoIcons:257,628 → 848 B
MaterialIcons:1,645,184 → 18,724 B
```

所以不能拿源字体文件大小直接当包体积。

## 14.4 可直接清理

### `delete: cached_network_image`

pubspec direct dep，业务零引用。

### `delete: cupertino_icons`（低收益）

业务零直接引用；实际字体已经 tree-shake 到很小，所以这是依赖卫生，不是性能主项。

## 14.5 需要迁移验证，不可直接删

`sqlite3_flutter_libs 0.6.0+eol`、lock 中 `sqlcipher_flutter_libs 0.7.0+eol`。

步骤：

1. `flutter pub deps` 明确依赖来源；
2. 对照当前 Drift/sqlite3 推荐组合；
3. Android release + DB open/migration/offline cache tests；
4. 比较 analyze-size native libs；
5. iOS 同样验证。

## 14.6 Release size CI 建议

每次正式 release 保存：

```text
flutter version
commit SHA
ABI/AAB/APK size
DevTools size-analysis JSON
largest 20 components
symbols/artifacts separately
```

对 PR 不一定做硬 gate，但 release branch 可设：

```text
compressed artifact Δ > X%
或 top package Δ > Y KB
→ require review
```

X/Y 应由连续几次真实发布分布决定，不在本报告凭空指定。

---

# 15. Ponytail Whole-repo 审计

按收益排序：

1. **`shrink:` 删除 `GfTopicCard._observeImage()` 整条 ratio probe。** 替换为 API intrinsic width/height + thumbnail variant。`[apps/mobile/packages/ui_kit/lib/src/components/gf_topic_card.dart]`
2. **`delete:` 删除零引用 `cached_network_image` direct dependency。** replacement: nothing。`[apps/mobile/packages/forum_app/pubspec.yaml]`
3. **`native:` Feed/Markdown/preview 图片优先使用 Flutter ImageCache + `cacheWidth/cacheHeight` / ResizeImage 语义。** 不再手写额外内存缓存。
4. **`native:` 长列表优先 `ListView.builder` / SliverList / SliverGrid。** Search/Course Detail 不需要新列表框架。
5. **`native:` Schedule 三个独立 future 使用 `Future.wait` + 一次发布。** 不建 session meta scheduler。
6. **`yagni:` 不创建 `GlobalNetworkScheduler` / PriorityRequestQueue / UniversalFetchPolicy。** 当前证据支持的是 token hot path、局部 cancel、single-flight、SWR。
7. **`shrink:` Topic 网络成功后先 publish UI，再 best-effort offline write。** 删除用户等待 cache write 的 critical-path dependency。
8. **`delete:` `cupertino_icons` 零直接引用可清理。** 体积收益很小，主要是依赖卫生。
9. **`shrink:` ui_kit 重复 Lucide license 资源可去重。** 不是性能主项。

### keep

- Campus binding status hard gate；
- Campus revision/account/foreground fences；
- `offlineCacheEpoch`；
- Home/Search/Topic generation/sequence correctness fence；
- Markdown body memoization；
- deferred Link Preview；
- Home/Topic local interaction merge；
- builder/sliver lists 已经存在的地方；
- update check post-frame；
- Android push consent-gated init。

Ponytail 的目标是删掉无谓工作，而不是为了“简化”删除安全/隐私边界。

---

# 16. P0 / P1 / P2 TODO

## P0-1 启动性能基准

**位置**：App bootstrap + telemetry layer；Android/iOS native diagnostics。
**工作量**：M。
**依赖**：无。
**风险**：低。

TODO：

- [x] 埋 firstFrameRasterized；
- [x] Home skeleton/data/card/media milestones；
- [x] Android `reportFullyDrawn` 与 Home 首内容帧对齐；
- [x] 记录 device/refresh-rate/build-mode；
- [x] cold/warm/hot 分开（iOS 记录 cold/hot）；
- [ ] 生产 release/profile 真机测 cold/warm/hot 各 30 次分布。

本地改造落地了 Timeline 事件和 Android/iOS 平台信息采集；数据不上传、不持久化。尚未连接真机完成分布采样，不能据此报告启动耗时数字。

验收：

- 能回答黑屏主要落在 native→first frame 还是 first frame→Home content；
- splash 改色不会被算作性能提升。

## P0-2 Session token hot path

**位置**：`packages/core/lib/src/api/gf_api_client.dart`、auth/session providers。
**工作量**：M。
**风险**：中（认证边界）。

TODO：

- memory token holder；
- startup hydrate 一次；
- login/renew/logout/401 同步；
- tests：账号切换、401、cleanup explicit Authorization、并发请求。

验收：普通 API GET/POST 不再每次调用 secure storage。

## P0-3 Feed media contract

**位置**：Go upload/media pipeline、API contract、Dart payload、`GfTopicCard`。
**工作量**：L。
**风险**：中（兼容旧数据/缓存/CDN）。

TODO：

- [x] 上传后生成静态 JPEG/PNG/BMP variants，并持久化 intrinsic width/height；超大图跳过 derivative，GIF/WebP 保留原图；
- [x] Topic API 返回可选 dimensions/variants；OpenAPI、生成 TypeScript、fixtures 和文档同步；
- [x] Flutter 按 logical width × DPR 选择最近 variant，并设置 cacheWidth/cacheHeight；
- [x] 删除 `_observeImage()`；无 metadata 的旧 payload、缓存和 CDN URL 仍走原图 fallback。

验收：Feed 首屏不再为了比例先 decode 原图；滚动时卡片高度不因图片到达跳变。

## P0-4 Home first-page SWR

**位置**：`home_page.dart` + 轻量 repository/cache。
**工作量**：M。
**风险**：中（账号态字段）。

TODO：

- cache schema/fence；
- hydrate old content；
- silent network refresh；
- stale error retention；
- logout clear。

验收：有缓存的重复启动，Flutter 首帧后能快速展示旧 Feed，不等待网络空白。

## P0-5 Topic cache write 脱离 UI critical path

**位置**：`pages/topic/topic_page.dart:193-205`。
**工作量**：S。
**风险**：低-中（session fence）。

验收：网络 response parse 后 UI 先发布；离线 cache 失败不延迟/覆盖页面。

## P1-1 Notifications generation + cancel

**工作量**：S。
**风险**：低。

验收：快速切 filter 永远不会被旧响应覆盖；旧 GET 被取消。

## P1-2 Messages/Shell poll single-flight

**工作量**：S。
**风险**：低。

验收：弱网下任一 poll key 同时最多一个 in-flight request。

## P1-3 Schedule meta parallelization

**工作量**：S。
**风险**：低。

验收：calendars/sectionTimes/latestUpdate 并行；失败仍各自降级；合并 UI publish。

## P1-4 Search/Course Detail lazy list

**工作量**：M。
**风险**：中（滚动位置、refresh、focus review）。

验收：加载多页后，不会在一次父级 build 中同步创建全部已加载 rows。

## P1-5 Release size governance

**工作量**：M。
**风险**：低。

Android native-build CI 现对 arm64 Release APK 生成 analyze-size JSON，并保存 APK 字节数、Flutter 版本与 commit SHA 为 90 天 artifact。仍需收集同配置多次 CI 运行建立历史基线；AAB、其他 ABI 和 iOS 基线未覆盖，不应用旧 SDK 数字设阈值。

## P2

- card interaction state 精细下沉（仅在 rebuild profiler 证明后）；
- Markdown 超长内容进一步 lazy；
- Android Baseline Profile；
- disk image cache；
- generic GET coalescing primitive；
- isolate/compute for JSON parse（仅当 timeline 明确主 isolate parse 超预算）。

---

# 17. Quantitative Benchmark 方案

## 17.1 设备矩阵

至少：

- 中低端 Android 60Hz；
- 主流 Android 120Hz；
- 一台 Android 厂商深度定制 ROM；
- iPhone 较旧一代 60Hz；
- 当前主力 iPhone 高刷机型。

## 17.2 启动实验

每个 build：

```text
cold start × 30
warm start × 30
hot/resume × 30
```

记录：

- TTID / firstFrameRasterized；
- first Home card；
- first media；
- fully drawn；
- request count / bytes；
- secure storage read count。

Android vitals 把 cold >=5s、warm >=2s、hot >=1.5s 定义为 excessive startup 平台阈值；这是 Android vitals 的标准，不是 YourTJ 自定义 SLO。

## 17.3 Feed scroll

测试场景：

- 无图 feed；
- 1 图/帖；
- 3+ 图/帖；
- 混合超大原图；
- 已缓存/未缓存；
- Wi‑Fi/限速网络。

采集：

```text
UI frame duration
Raster frame duration
jank count
image decode events
GC/memory
network bytes
ImageCache live/pending size
```

比较修复前后：

- original URL vs thumbnail；
- ratio probe vs metadata；
- no cacheWidth vs target decode；
- cold Home vs SWR Home。

## 17.4 Campus

每种场景记录请求 waterfall：

1. 冷进入；
2. 5 分钟内返回；
3. tab today→timetable→today；
4. background/resume；
5. binding revision 改变；
6. logout/login different account；
7. authorizationRequired。

要求性能优化不得让 4–7 的隐私边界退化。

## 17.5 Size

Android：

```bash
flutter build appbundle --release --analyze-size
flutter build apk --release --split-per-abi --analyze-size
```

生产同 SDK、同 dart-defines、同 OEM push adapters。

iOS（macOS）：

```bash
flutter build ios --release --analyze-size
```

并保存 App Store / archive size 数据。

---

# 18. QA / 验收 Checklist

## 启动

- [x] cold/warm/hot 分开统计（iOS 没有单独 warm 分类）；
- [x] `firstFrameRasterized` 可观测；
- [x] Home shell/data/card/media 各自有时间点；
- [ ] Android TTID/TTFD 真机样本可复现；
- [ ] iOS launch 与 Flutter first content 可区分；
- [ ] splash 视觉修复不被计入性能收益；
- [ ] 首帧前无新增非关键同步 I/O；
- [ ] update check 继续 post-frame；
- [ ] push consent 语义不变。

## Feed / Image

- [x] API 返回 width/height；
- [x] Feed 使用 derivative URL；
- [x] Flutter 根据 DPR 选 variant；
- [x] `cacheWidth/cacheHeight` 按目标像素设置；
- [x] `GfTopicCard` 不再 resolve 图片探比例；
- [x] 新媒体在图片未到达前使用 intrinsic ratio 固定 layout；
- [ ] 4K 原图不会作为普通 Feed thumbnail decode；
- [ ] error/legacy image 无尺寸时有兼容 fallback；
- [ ] 多图 carousel 也受 decode sizing 约束。

## Home

- [ ] 首屏 cache 绑定 account/session fence；
- [ ] logout/401 不可展示前账号 viewer-specific 数据；
- [ ] cached content 立即可见；
- [ ] refresh 失败保留内容；
- [ ] sort 切换取消旧请求；
- [ ] Topic return 不无条件重复刷新 fresh Home。

## Network

- [ ] 普通 API request 不每次读 secure storage；
- [ ] token renewal/logout/401 memory/storage 一致；
- [ ] Notifications filter 不被 stale response 覆盖；
- [ ] polling 有 in-flight guard；
- [ ] POST/PUT/DELETE 默认无自动 retry；
- [ ] GET retry 只针对 transient failure；
- [ ] CancelToken 不替代 generation/session correctness fence。

## Campus

- [ ] fresh status hard gate 保留；
- [ ] cache 只在 verified binding 后 restore；
- [ ] background clear 保留；
- [ ] revision change clear；
- [ ] account/site epoch clear；
- [ ] grades/notice bodies 不进入该 5min cache；
- [ ] status 后 fresh cache 立即可见；
- [ ] tab switch 不重复拉 fresh dataset；
- [ ] 请求数量可通过 waterfall 断言。

## Topic offline cache

- [ ] UI 不等待 Drift write；
- [ ] background write 有 epoch/topic fence；
- [ ] cache failure 不影响在线阅读；
- [ ] 网络失败仍能按当前语义回退离线 cache。

## 滚动

- [ ] 60/90/120Hz 设备都有 frame timing；
- [ ] Search 多页结果 lazy；
- [ ] Course reviews 多页结果 lazy；
- [ ] 不因全局 `setState` 猜测就大改架构；
- [ ] DevTools 证据能定位每个后续 rebuild 优化。

## 包体积

- [x] 本机 release 构建使用 Flutter 3.44.9，与 CI 固定 SDK 对齐；
- [x] release AAB 通过插件注册编译并用临时 build-only key 构建；debug APK 仍使用真实 `integration_test` 插件；
- [x] arm64 APK analyze-size JSON 被 CI 保存；
- [ ] Android 每 ABI / AAB 有历史曲线；
- [ ] iOS 有正式 baseline；
- [ ] `cached_network_image` 零引用依赖删除；
- [ ] sqlite EOL 迁移经过 DB/migration tests；
- [ ] symbol/debug metadata 与用户下载 size 分开统计。

---

# 19. 必须真机验证的实验矩阵

| 假设 | 静态证据 | 必须怎么测 | 通过标准 |
|---|---|---|---|
| 4–8 秒主要发生在 Flutter first frame 前 | `main()` 不阻塞，但 Android launch surface 存在 | cold start + firstFrameRasterized + Perfetto | 能明确分段，不再凭肉眼 |
| Feed 原图 decode 是 jank 主因 | 无 derivative/size + no target decode | 大图 feed FrameTiming + memory | thumbnail 方案显著降低 raster/decode/memory 峰值 |
| secure storage read 显著拖慢多请求页面 | 每 request await read | instrument token read + request start | memory token 后 request dispatch latency 降低 |
| Home SWR 改善首屏感知 | 当前无 first-page hydrate | 有/无 cache cold app process 对比 | first useful Home 明显提前，correctness 不退化 |
| Topic Drift write 延迟 first content | 当前 await cachePut before setState | timeline around DB write | publish-before-cache 使 TTF content 降低 |
| Search/Course non-lazy 导致多页 jank | children/Column 全量 build | 5–10 页数据 Track Builds | sliver 后 build count / frame time 降低 |
| Android black launch 是主要“黑屏”来源 | night LaunchTheme black | 同时记录 native launch 与 Flutter first frame | 只改善视觉时不误报性能收益 |
| Baseline Profile 值得做 | 当前无数据 | Macrobenchmark before/after | 有稳定、可重复收益才保留 |

---

# 20. 官方资料（检索日期：2026-09-24）

## Flutter 图片 / 内存

- https://api.flutter.dev/flutter/widgets/Image/Image.network.html
- https://api.flutter.dev/flutter/widgets/Image-class.html

要点：网络图片使用 Flutter 图片缓存；未固定尺寸可能造成加载期 layout change；`cacheWidth/cacheHeight` 可提示 engine 目标尺寸解码、降低 ImageCache 内存。

## Flutter 首帧 / Performance

- https://api.flutter.dev/flutter/widgets/WidgetsBinding/firstFrameRasterized.html
- https://docs.flutter.dev/perf/best-practices
- https://docs.flutter.dev/tools/devtools/performance

要点：profile/真实设备/timeline 才是性能判断基础；60Hz 单帧理论预算约 16.7ms。

## Flutter 包体积

- https://docs.flutter.dev/perf/app-size
- https://docs.flutter.dev/tools/devtools/app-size

要点：Debug 大小无代表性；使用 release `--analyze-size` 和 DevTools attribution；AOT/tree shaking 会删除不可达 Dart 代码。

## Android 启动

- https://developer.android.com/topic/performance/issues/launch-time
- https://developer.android.com/topic/performance/appstartup/analysis-optimization
- https://developer.android.com/topic/performance/appstartup/best-practices
- https://developer.android.com/develop/ui/views/launch/splash-screen
- https://docs.flutter.dev/platform-integration/android/splash-screen
- https://developer.android.com/google/play/vitals/launch-time

## iOS 启动

- https://developer.apple.com/documentation/xcode/reducing-your-app-s-launch-time
- https://developer.apple.com/documentation/xcode/specifying-your-apps-launch-screen

## Android Baseline Profiles

- https://developer.android.com/topic/performance/baselineprofiles/overview
- https://developer.android.com/topic/performance/baselineprofiles/measure-baselineprofile

---

# 21. 最终结论

当前 YourTJ Flutter App **不需要一次“大型性能架构重构”**。

最值得做的是删除几处已经被源码证实的无谓工作：

```text
每请求 secure-storage token read
Feed 为拿比例先 decode 原图
Feed 下载/解码远大于显示尺寸的媒体
Home 冷启动只能等网络
Topic 网络成功后先等离线 DB 写入
弱网轮询允许重叠
几个长列表仍全量 children build
Schedule 独立元数据串行
```

同时，应保留当前已经存在且正确的边界：

```text
Campus fresh status privacy gate
Campus/account/revision/foreground cache fence
offlineCacheEpoch
Home/Search/Topic generation correctness fence
Markdown memoization + deferred link previews
push consent-gated native initialization
```

建议实施顺序固定为：

1. **先测清 TTID / Flutter first frame / Home data / first media；**
2. **再做 token hot path；**
3. **再改 Feed 媒体 contract 与 decode；**
4. **再做 Home SWR + Topic cache-write critical path；**
5. **再处理局部 poll/cancel/lazy-list；**
6. **最后根据真机 timeline 决定是否需要更深的 state/render 重构或 Baseline Profile。**

这样做符合 Ponytail 的原则：优先删除工作、复用 Flutter/平台能力、保留已经证明有价值的安全 fence，不为了“性能架构”引入一套比问题本身更复杂的新框架。
