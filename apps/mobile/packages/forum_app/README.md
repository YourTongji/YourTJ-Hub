# forum_app

YourTJ 移动端论坛客户端(Flutter)。`apps/mobile` melos 工作区的入口应用包,依赖 `core`(契约/API 客户端/markdown 转换)、`auth`(登录与 token 存储)、`ui_kit`(设计 token 与 Gf* 组件)。

## 当前交互架构

- **入口与主题**:`lib/main.dart` → `GfApp`(`lib/src/app.dart`)。主题严格来自 ui_kit 设计 token(web `tokens.css` 的 1:1 镜像),light/dark 双主题默认跟随系统,设置页可手动切换;l10n 为 zh/en,跟随系统语言。
- **导航**(`lib/src/router.dart`, go_router):首页、校园、通知、私信四个持久分支,各自保留 navigator 与滚动状态;搜索、发布、个人资料、会话、课程、排课器和 Wiki 以推入页呈现。底部为有可访问名称的图标,根页面导航随向下阅读收起、向上阅读恢复;头像打开账号抽屉。发布/新会话使用固定含义的 FAB,帖内回复与楼层跳转使用底部操作栏。
- **内容与编辑**:按图片方向展示的信息流;瞬间/提问使用图片轮播加正文,文章使用折叠工具栏的富文本编辑器;预览后选择分类,支持服务端草稿和离开确认。个人页的各个 Tab 请求真实活动流,内容管理和回收站按服务端权限提供操作。
- **账号与管理**:资料编辑保留网站、语言和社交链接;账号、安全与隐私控制位于独立设置页。完整 Web 管理与审核工作台通过受限的原生 WebView 访问,会话仅通过请求头交接。公共站点信息从校园或设置页的“关于社区”进入。
- **交互规格**:具体行为、权限和验证边界见[移动端体验](../../../../docs/product/mobile-experience.md);路由以 `lib/src/router.dart` 为准。
- **离线**:`lib/src/offline/drift_cache.dart` 基于 drift 缓存已浏览话题与 IM 会话。
- **运行配置**(`lib/src/app_config.dart`):经 `--dart-define` 注入 `YOURTJ_OIDC_ISSUER` / `YOURTJ_OIDC_CLIENT_ID` / `YOURTJ_API_BASE_URL`;默认内建 OIDC issuer 为 `http://localhost:5234/api/oauth`,API baseUrl 为空时 Android 模拟器走 `10.0.2.2`。

## 运行与验证

前置:Flutter SDK + melos;工作区脚本定义在 `apps/mobile/pubspec.yaml` 的 `melos:` 键。

```bash
cd apps/mobile
melos bootstrap          # 首次或依赖变更后
melos run analyze        # 全包静态检查
melos run test           # 全包测试
```

调试运行(在 `apps/mobile/packages/forum_app` 下):

```bash
flutter run --dart-define=YOURTJ_OIDC_ISSUER=http://localhost:5234/api/oauth \
            --dart-define=YOURTJ_OIDC_CLIENT_ID=yourtj-mobile
```

CI(`ci-mobile`)对 `apps/mobile/**` 运行同一组 bootstrap / analyze / test。

## 本地设备集成测试

`integration_test/local_journey_test.dart` 使用真实 API、系统安全存储和 WebView。只有显式提供测试账号密码才运行,并拒绝非本地服务地址。准备隔离的本地数据库、一个有管理权限的演示账号和至少一个分区;演示站点关闭登录与新用户发帖验证码要求,并设定足够的每日发帖额度。测试会保存草稿、裁切上传封面并恢复原封面,以及新增后删除一个友情链接以验证后台表单;不会清空数据库。还会验证 28 个后台页面与两个课程工作台的登录衔接和移动视口。

在 `forum_app` 目录运行,设备 ID 从 `flutter devices` 获取:

```bash
flutter test integration_test/local_journey_test.dart -d "$YOURTJ_TEST_DEVICE" \
  --dart-define=YOURTJ_API_BASE_URL=http://127.0.0.1:15234 \
  --dart-define="YOURTJ_TEST_USERNAME=$YOURTJ_TEST_USERNAME" \
  --dart-define="YOURTJ_TEST_PASSWORD=$YOURTJ_TEST_PASSWORD"
```

Android 模拟器先运行 `adb -s "$YOURTJ_TEST_DEVICE" reverse tcp:15234 tcp:15234`,
然后同样使用 `http://127.0.0.1:15234`。该地址需与隔离服务的站点 URL 一致;localhost
也使浏览器安全上下文 API 可用于本地开发。生产环境使用 HTTPS。此测试独立于无后端的
widget/契约测试;通过它不代表所有原生文件选择、分享或发布平台均已验证。

保存设备截图时,使用同目录的 driver;截图写入本地输出目录:

```bash
YOURTJ_TEST_ARTIFACTS=/absolute/path/to/screenshots flutter drive \
  --driver=test_driver/integration_driver.dart \
  --target=integration_test/local_journey_test.dart -d "$YOURTJ_TEST_DEVICE" \
  --dart-define=YOURTJ_API_BASE_URL=http://127.0.0.1:15234 \
  --dart-define="YOURTJ_TEST_USERNAME=$YOURTJ_TEST_USERNAME" \
  --dart-define="YOURTJ_TEST_PASSWORD=$YOURTJ_TEST_PASSWORD" \
  --dart-define=YOURTJ_TEST_SCREENSHOTS=true
```

可选的原生文件往返检查需人为操作系统 UI。在上述命令增加
`--dart-define=YOURTJ_TEST_FILES=true`,测试会创建导出,在日志出现 `NATIVE_EXPORT_OPEN`
后检查并关闭系统分享面板;出现 `NATIVE_PICKER_OPEN` 后选择预先放入设备“下载”或
“我的 iPhone”目录的 `mobile-picker-check.json`。内容可为 `{"fixture":"file selector"}`。
测试验证 Web 表单收到该文件,不会提交导入。`YOURTJ_TEST_FILES_ONLY=true` 可跳过已覆盖的
后台模块遍历,缩短这项单独检查;文件选择等待上限为五分钟。

## 服务端错误文案

`lib/src/server_message_catalog.dart` 从 Web 的 `server` / `serverMessages` 目录生成,保留
错误码和参数插值。更新 Web 文案后,在仓库根目录运行
`node apps/mobile/tools/generate_server_messages.mjs`;`pnpm check:i18n` 同时检查移动端文案是否同步。

## 边界

- 后端访问只经 `core` 的 API 客户端/repository;业务状态归本包(Riverpod,`lib/src/providers.dart`)。
- 不直接依赖 TDesign:`forum_app` 不 import `tdesign_flutter`,组件统一走 `ui_kit` 的 Gf* API。
- 契约镜像位于 `core/lib/src/gen/*.dart`(见 docs/architecture/contracts-and-data.md)。
