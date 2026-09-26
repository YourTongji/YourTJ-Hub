# ui_kit

YourTJ 移动端设计系统(Flutter):设计 token、`ThemeData` 与 Gf* 组件。`forum_app` 的界面组件都经由此包暴露;由仓库维护外观，复用 Flutter 原生输入、焦点、选择与可访问性能力。

## 所有权与边界

- **Token**:`lib/src/theme/tokens.json` 是 web 设计语言的移动端派生源(源:`apps/gooseforum/resource/src/styles/tokens.css`,1:1 镜像)。**改动 `tokens.css` 必须在同一提交更新 `tokens.json`**(契约式约束,见 docs/development/local-development.md)。
- **组件实现**:Gf 组件直接组合 Flutter 原生控件，不依赖第三方组件库的默认皮肤；圆角、留白、语义色、禁用和选中状态由本包统一维护，见 [0039](../../../../docs/decisions/0039-native-gf-component-foundation.md)。
- **组件分层**:`lib/src/components/` 下 `atoms/`(基础元素)、`business/`(业务组件)、`surfaces/`(容器与浮层)与顶层组件(导航/按钮/卡片等);主题在 `lib/src/theme/`。公开面统一由 `lib/ui_kit.dart` 导出。
- **依赖方向**:本包运行时只依赖 Flutter、`flutter_svg`(符号) 与 `extended_image`(图片查看器);不依赖 `core` / `auth` / `forum_app`,不发请求、不持有业务状态。

`GfInput` 使用 Flutter 原生 `TextField`，由 Gf token 保持外观，并透传 autofill hints、输入动作、
自动纠错、建议、大小写、焦点和回调。普通表单使用共享 `InputDecorationTheme` 的灰底16px圆角；
浮动标签留在填充区域内，聚焦轮廓无缺口；需要校验的 `TextFormField` 继承同一主题。由业务层选择凭据/一次性验证码提示及提交 autofill context。

## 主要组件

- 导航与动作:`GfBottomNavigation`(四个目的地,支持纯图标)、`GfTabBar`、`GfAppBar`(56px 居中导航栏,默认返回 + 显式 leading)、`GfButton` / `GfIconButton` / `GfFloatingAction`。
- 内容:`GfTopicCard` / `GfTopicRow`(对齐 Web TopicRow 语义)、`GfPostComposer`(Markdown 回复编辑器 + 图片动作)、`GfChatInput` / `GfMessageBubble` / `GfConversationRow`、`GfDraftRow` / `GfNotificationRow` / `GfSettingRow` / `GfUserCard`。
- 反馈与状态:`GfSkeleton`(结构化骨架)、`GfEmpty` / `GfStatusMessage` / `GfToast` / `GfAlertDialog` / `GfModal` / `GfBottomSheet` / `GfScrollToTop` / `GfLoadingIndicator`。
- 表单与展示:`GfInput` / `GfTextarea` / `GfSegmented` / `GfPillSwitch` / `GfSelectTag` / `GfAvatar` / `GfAvatarStack` / `GfBadge` / `GfChip` / `GfDivider` / `GfTooltip` / `GfAlert` / `GfDotGridBackground`。

`Current`: `GfBadgeMedallion` 统一徽章的圆形外圈、高光和内盘；暗色主题保留浅色内盘以呈现服务端固定颜色图案。
`GfUserCard` 的 `coloredBadges` 使用纯图标入口，保留名称语义、提示和详情回调；
`GfAchievementCard` 使用同一徽章图案、居中名称与说明，随字号增高，点击详情由应用层处理。

## Token 与主题规则

- 主题数据来自 `lib/src/theme/tokens.json`(light/dark 双主题);`GfThemeData` 生成 `ThemeData`,`GfTheme.colorsOf(context)` 提供语义色。
- 新增/修改组件样式走 token,不硬编码色值;与 web `tokens.css` 保持 1:1。

## 验证

`Current`: 移动端按钮和分类标签分别约束可见背景与触控区域，并允许文字增大时增高；
具体尺寸见[移动端展示规范](../../../../docs/product/mobile-experience.md#language-and-presentation)。
`GfChip.tapTargetHeightFor(context)` 与 `GfTabBar.heightFor(context)` 供横向分类栏、页面和悬浮栏
同步计算大字体高度。
`GfEmpty` 支持说明和下一步操作，并适应短屏；资料统计按可用宽度和字号排布，设置行支持多行标题。
这些组件尺寸适配保留共享语义色与 token 镜像。

`Current`: `showGfBottomSheet` 统一使用 Flutter 的可滚动模态路由，沿用共享面板主题。
未指定高度时按内容收缩；`height` 是受可用视口约束的内容高度，长内容由调用方提供滚动容器。
安全区由入口消费一次，底部背景覆盖手势区；`keyboardAware: true` 负责键盘避让，
调用方不再叠加 `viewInsets`。`enableDrag: false` 可保护未保存的编辑。
`showGfAlertDialog` / `showGfModal` 用原生 `Dialog` 约束键盘上方的可用区域，使用24px圆角、原生焦点和单层滚动内容。
回归测试包含刘海、底部手势区、长列表和键盘；应用层补充小屏双倍字号表单测试。

```bash
cd apps/mobile
melos run analyze        # 或 melos exec -- flutter analyze
melos run test           # 或 melos exec -- flutter test
```

测试:`test/tokens_test.dart`(token 完整性)、`test/components/`(组件行为)、`test/golden/`(golden 快照)。

## 边界

- `ui_kit` 不依赖 `core` / `auth` / `forum_app`;业务状态与请求归上层包。
- 页面通过 Gf API 复用组件；特殊编辑器保留原生能力，并明确其场景样式。

导航与操作符号使用 `GfSymbol` 和 ReIcon SVG；品牌与功能字形例外、映射和许可见 [assets](assets/README.md)。页面交互与 Figma 入口以[移动端产品说明](../../../../docs/product/mobile-experience.md)为准。
Shared headers use a centered 18px semibold title and `GfSymbol` back action. Buttons use pill
shapes, separate painted/48px hit bounds, and state-aware disabled palettes. `showGfBottomSheet` owns the root navigator, a 640px
width bound, safe areas, optional keyboard avoidance and a 28px drag-handle area inside its height.
Builders supply content without adding keyboard insets again; use `showDragHandle: false` for
non-draggable editing surfaces. Navigation symbols share the 24px drawing grid and switch from
ReIcon Outline to the matching Filled weight when selected.

Search and chat use filled capsules. Reply inputs grow from one to four lines above a flat icon
toolbar; article publishing keeps an open writing canvas. Message bubbles use 20px corners and
size against the conversation pane. `showBubble: false` removes the surface and padding for
standalone stickers while retaining content bounds, alignment and time. Menus and segmented choices grow with large text; action
icons use `GfSymbol` and retain at least 44px independent targets. Component-specific geometry
is owned here and does not change the Web/mobile token mirror.

`GfGlassSurface` / `GfGlassIconButton` 为封面上的操作提供局部毛玻璃圆形底板、细描边和至少 44px 点击区域，沿用原有按钮语义。`GfUserCardHeader` 将封面、重叠头像和操作带放在同一绘制层，可用于折叠导航；`GfUserCard(showHeader: false)` 只展示后续资料。公开页与编辑页使用同一封面尺寸计算。
