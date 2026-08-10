# ui_kit

yourtj 移动端设计系统(Flutter):设计 token、`ThemeData` 与 Gf* 组件。`forum_app` 的界面组件都经由此包暴露;应用页面不直接依赖 TDesign 的预发布 API。

## 所有权与边界

- **Token**:`lib/src/theme/tokens.json` 是 web 设计语言的移动端派生源(源:`apps/gooseforum/resource/src/styles/tokens.css`,1:1 镜像)。**改动 `tokens.css` 必须在同一提交更新 `tokens.json`**(契约式约束,见 docs/development/local-development.md)。
- **TDesign 隔离**:`tdesign_flutter` 锁定 `1.0.0-alpha.1`(0.2.7 无法在 Flutter 3.44 编译,见 pubspec.yaml 注释)。TDesign v1 API 只在本包内部(theme/ 与 components/)使用;`forum_app` / `core` / `auth` 不得 import `tdesign_flutter`。
- **组件分层**:`lib/src/components/` 下 `atoms/`(基础元素)、`business/`(业务组件)、`surfaces/`(容器与浮层)与顶层组件(导航/按钮/卡片等);主题在 `lib/src/theme/`。公开面统一由 `lib/ui_kit.dart` 导出。
- **依赖方向**:本包运行时只依赖 Flutter、锁定的 TDesign 与 `extended_image`(图片查看器);不依赖 `core` / `auth` / `forum_app`,不发请求、不持有业务状态。

## 主要组件

- 导航与动作:`GfBottomNavigation`(四目的地 + 中央发布动作)、`GfTabBar`、`GfAppBar`(TDesign TNavBar 封装,默认返回 + 显式 leading)、`GfButton` / `GfIconButton` / `GfFloatingAction`。
- 内容:`GfTopicCard` / `GfTopicRow`(对齐 Web TopicRow 语义)、`GfPostComposer`(Markdown 回复编辑器 + 图片动作)、`GfChatInput` / `GfMessageBubble` / `GfConversationRow`、`GfDraftRow` / `GfNotificationRow` / `GfSettingRow` / `GfUserCard`。
- 反馈与状态:`GfSkeleton`(结构化骨架)、`GfEmpty` / `GfStatusMessage` / `GfToast` / `GfAlertDialog` / `GfModal` / `GfBottomSheet` / `GfScrollToTop` / `GfLoadingIndicator`。
- 表单与展示:`GfInput` / `GfTextarea` / `GfSegmented` / `GfPillSwitch` / `GfSelectTag` / `GfAvatar` / `GfAvatarStack` / `GfBadge` / `GfChip` / `GfDivider` / `GfTooltip` / `GfAlert` / `GfDotGridBackground`。

## Token 与主题规则

- 主题数据来自 `lib/src/theme/tokens.json`(light/dark 双主题);`GfThemeData` 生成 `ThemeData`,`GfTheme.colorsOf(context)` 提供语义色。
- 新增/修改组件样式走 token,不硬编码色值;与 web `tokens.css` 保持 1:1。

## 验证

```bash
cd apps/mobile
melos run analyze        # 或 melos exec -- flutter analyze
melos run test           # 或 melos exec -- flutter test
```

测试:`test/tokens_test.dart`(token 完整性)、`test/components/`(组件行为)、`test/golden/`(golden 快照)。

## 边界

- `ui_kit` 不依赖 `core` / `auth` / `forum_app`;业务状态与请求归上层包。
- TDesign 预发布 API 不得泄漏到 `forum_app` / `core` / `auth`。
