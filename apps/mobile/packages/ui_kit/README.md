# ui_kit

YourTJ 移动端设计系统(Flutter):设计 token、`ThemeData` 与 Gf* 组件。`forum_app` 的界面组件都经由此包暴露;应用页面不直接依赖 TDesign 的预发布 API。

## 所有权与边界

- **Token**:`lib/src/theme/tokens.json` 是 web 设计语言的移动端派生源(源:`apps/gooseforum/resource/src/styles/tokens.css`,1:1 镜像)。**改动 `tokens.css` 必须在同一提交更新 `tokens.json`**(契约式约束,见 docs/development/local-development.md)。
- **TDesign 隔离**:`tdesign_flutter` 锁定 `1.0.0-alpha.1`(0.2.7 无法在 Flutter 3.44 编译,见 pubspec.yaml 注释)。TDesign v1 API 只在本包内部(theme/ 与 components/)使用;`forum_app` / `core` / `auth` 不得 import `tdesign_flutter`。
- **组件分层**:`lib/src/components/` 下 `atoms/`(基础元素)、`business/`(业务组件)、`surfaces/`(容器与浮层)与顶层组件(导航/按钮/卡片等);主题在 `lib/src/theme/`。公开面统一由 `lib/ui_kit.dart` 导出。
- **依赖方向**:本包运行时只依赖 Flutter、锁定的 TDesign 与 `extended_image`(图片查看器);不依赖 `core` / `auth` / `forum_app`,不发请求、不持有业务状态。

`GfInput` 使用 Flutter 原生 `TextField`，由 Gf token 保持外观，并透传 autofill hints、输入动作、
自动纠错、建议、大小写、焦点和回调。锁定的 TDesign 输入组件未透传这些原生表单属性；页面继续
只使用 `GfInput`，由业务层选择凭据/一次性验证码提示以及何时提交 autofill context。

## 主要组件

- 导航与动作:`GfBottomNavigation`(四个目的地,支持纯图标)、`GfTabBar`、`GfAppBar`(56px 居中导航栏,默认返回 + 显式 leading)、`GfButton` / `GfIconButton` / `GfFloatingAction`。
- 内容:`GfTopicCard` / `GfTopicRow`(对齐 Web TopicRow 语义)、`GfPostComposer`(Markdown 回复编辑器 + 图片动作)、`GfChatInput` / `GfMessageBubble` / `GfConversationRow`、`GfDraftRow` / `GfNotificationRow` / `GfSettingRow` / `GfUserCard`。
- 反馈与状态:`GfSkeleton`(结构化骨架)、`GfEmpty` / `GfStatusMessage` / `GfToast` / `GfAlertDialog` / `GfModal` / `GfBottomSheet` / `GfScrollToTop` / `GfLoadingIndicator`。
- 表单与展示:`GfInput` / `GfTextarea` / `GfSegmented` / `GfPillSwitch` / `GfSelectTag` / `GfAvatar` / `GfAvatarStack` / `GfBadge` / `GfChip` / `GfDivider` / `GfTooltip` / `GfAlert` / `GfDotGridBackground`。

## Token 与主题规则

- 主题数据来自 `lib/src/theme/tokens.json`(light/dark 双主题);`GfThemeData` 生成 `ThemeData`,`GfTheme.colorsOf(context)` 提供语义色。
- 新增/修改组件样式走 token,不硬编码色值;与 web `tokens.css` 保持 1:1。

## 验证

`Current`: 移动端按钮使用 44–56px 最小高度并允许文字换行增高；输入正文为 16px，
可点击分类保留 44px 触控区域。`GfTabBar.heightFor(context)` 供页面与悬浮栏同步计算大字体高度。
`GfEmpty` 支持说明和下一步操作，并适应短屏；资料统计按可用宽度和字号排布，设置行支持多行标题。
这些组件尺寸适配保留共享语义色与 token 镜像。

`Current`: `showGfBottomSheet` 统一使用 Flutter 的可滚动模态路由，沿用共享面板主题。
未指定高度时按内容收缩；`height` 是受可用视口约束的内容高度，长内容由调用方提供滚动容器。
安全区由入口消费一次，底部背景覆盖手势区；`keyboardAware: true` 负责键盘避让，
调用方不再叠加 `viewInsets`。`enableDrag: false` 可保护未保存的编辑。
`showGfAlertDialog` / `showGfModal` 用原生 `Dialog` 约束键盘上方的可用区域，保留 Gf/TDesign 内容样式。
回归测试包含刘海、底部手势区、长列表和键盘；应用层补充小屏双倍字号表单测试。

```bash
cd apps/mobile
melos run analyze        # 或 melos exec -- flutter analyze
melos run test           # 或 melos exec -- flutter test
```

测试:`test/tokens_test.dart`(token 完整性)、`test/components/`(组件行为)、`test/golden/`(golden 快照)。

## 边界

- `ui_kit` 不依赖 `core` / `auth` / `forum_app`;业务状态与请求归上层包。
- TDesign 预发布 API 不得泄漏到 `forum_app` / `core` / `auth`。

导航符号使用 `GfSymbol` 与共享 Lucide SVG,来源和许可见 [assets](assets/README.md)。页面交互与 Figma 入口以[移动端产品说明](../../../../docs/product/mobile-experience.md)为准。
