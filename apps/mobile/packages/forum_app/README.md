# forum_app

yourtj 移动端论坛客户端(Flutter)。`apps/mobile` melos 工作区的入口应用包,依赖 `core`(契约/API 客户端/markdown 转换)、`auth`(登录与 token 存储)、`ui_kit`(设计 token 与 Gf* 组件)。

## 当前交互架构

- **入口与主题**:`lib/main.dart` → `GfApp`(`lib/src/app.dart`)。主题严格来自 ui_kit 设计 token(web `tokens.css` 的 1:1 镜像),light/dark 双主题默认跟随系统,设置页可手动切换;l10n 为 zh/en,跟随系统语言。
- **导航**(`lib/src/router.dart`,go_router):`StatefulShellRoute.indexedStack` 四个持久分支——首页 `/`、搜索 `/search`、消息 `/messages`、我的 `/profile`,每分支独立 navigator 与状态;底部导航中央为全局发布动作,`/publish` 以推入页存在(不常驻)。其余路由:`/c/:slug/:id` 分类、`/p/:postId` 话题详情、`/u/:userId` 用户主页、`/settings`、`/notifications`、`/login`、`/drafts`。
- **页面**(`lib/src/pages/`):home / category / topic / profile / search / messages / notifications / publish / drafts / settings / login。推入页带显式 44dp 返回按钮;长页面支持回到顶部(`GfScrollToTop` + `lib/src/navigation/tab_scroll_registry.dart`)。
- **内容与编辑**:Web 对齐的列表/卡片话题流(`lib/src/widgets/topic_list.dart`,复用 `GfTopicRow` / `GfTopicCard`);话题详情为帖子流 + Markdown 渲染 + 图片查看器 + 点赞/收藏/评论 + 轻量回复;全局 Markdown 发布编辑器(`publish_page.dart`)窄屏编辑/预览切换、宽屏双栏、格式与图片工具栏,支持草稿保存与编辑回填;结构化骨架屏(`lib/src/widgets/skeletons.dart`)与统一状态视图(`status_views.dart`)。
- **离线**:`lib/src/offline/drift_cache.dart` 基于 drift 缓存已浏览话题与 IM 会话。
- **运行配置**(`lib/src/app_config.dart`):经 `--dart-define` 注入 `YOURTJ_OIDC_ISSUER` / `YOURTJ_OIDC_CLIENT_ID` / `YOURTJ_API_BASE_URL`;默认内建 OIDC issuer 为 `http://localhost:5234/api/oauth`,API baseUrl 为空时 Android 模拟器走 `10.0.2.2`。

## 关键页面

| 路由 | 页面 | 说明 |
|---|---|---|
| `/` | HomePage | 话题流 |
| `/search` | SearchPage | 聚合搜索 + scope tabs |
| `/messages` | MessagesPage | IM 会话与聊天 |
| `/profile` | ProfilePage | 用户主页/个人资料 |
| `/u/:userId` | ProfilePage | 用户主页(他人视角) |
| `/publish` | PublishPage | 全局发布/编辑 |
| `/p/:postId` | TopicPage | 话题详情 + 回复 |
| `/c/:slug/:id` | CategoryPage | 分类话题 |
| `/drafts` | DraftsPage | 草稿列表 |
| `/notifications` | NotificationsPage | 通知 |
| `/settings` | SettingsPage | 设置 |
| `/login` | LoginPage | OIDC 登录 |

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

## 边界

- 后端访问只经 `core` 的 API 客户端/repository;业务状态归本包(Riverpod,`lib/src/providers.dart`)。
- 不直接依赖 TDesign:`forum_app` 不 import `tdesign_flutter`,组件统一走 `ui_kit` 的 Gf* API。
- 契约镜像位于 `core/lib/src/gen/*.dart`(见 docs/architecture/contracts-and-data.md)。
