import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

import 'catalog.dart';
import 'mock.dart';
import 'screens.dart';

/// Preview with `flutter run --route '/settings?theme=dark&y=640'`.
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final route = Uri.tryParse(
    WidgetsBinding.instance.platformDispatcher.defaultRouteName,
  );
  galleryLaunchPage = route?.pathSegments.lastOrNull;
  galleryInitialScroll =
      double.tryParse(route?.queryParameters['y'] ?? '') ?? 0;
  if (route?.queryParameters['theme'] == 'dark') {
    galleryThemeMode.value = ThemeMode.dark;
  }
  runApp(const GalleryApp());
}

String? galleryLaunchPage;

final ValueNotifier<ThemeMode> galleryThemeMode = ValueNotifier(
  ThemeMode.light,
);

class GalleryPage {
  const GalleryPage(
    this.id,
    this.title,
    this.subtitle,
    this.symbol,
    this.build,
  );
  final String id;
  final String title;
  final String subtitle;
  final String symbol;
  final WidgetBuilder build;
}

final List<GalleryPage> catalogPages = [
  GalleryPage(
    'foundations',
    '基础',
    '颜色角色 · 字号 · 圆角 · 间距 · 阴影',
    'palette',
    (_) => const FoundationsPage(),
  ),
  GalleryPage(
    'buttons',
    '按钮与操作',
    '按钮 · 图标按钮 · 浮动按钮 · 导航文字操作',
    'square-pen',
    (_) => const ButtonsPage(),
  ),
  GalleryPage(
    'inputs',
    '表单与选择',
    '输入框 · 搜索 · 分段 · 标签 · 开关',
    'type',
    (_) => const InputsPage(),
  ),
  GalleryPage(
    'feedback',
    '标签与反馈',
    '徽标 · 计数 · 分类 · 提示条 · 轻提示 · 空状态',
    'circle-alert',
    (_) => const FeedbackPage(),
  ),
  GalleryPage(
    'lists',
    '列表与分组',
    '分组列表 · 设置行 · 通知行 · 会话行',
    'list',
    (_) => const ListsPage(),
  ),
  GalleryPage(
    'overlays',
    '浮层',
    '对话框 · 底部面板 · 菜单 · 操作表',
    'layers-filled',
    (_) => const OverlaysPage(),
  ),
  GalleryPage(
    'content',
    '内容',
    '帖子卡片 · 消息气泡 · 资料卡',
    'file-text',
    (_) => const ContentPage(),
  ),
];

final List<GalleryPage> screenPages = [
  GalleryPage(
    'profile-edit',
    '编辑资料',
    '分组表单 · 封面头像 · 字数统计',
    'pen-line',
    (_) => const ProfileEditScreen(),
  ),
  GalleryPage(
    'settings',
    '设置',
    '账号卡片 · 分组设置 · 当前值',
    'settings',
    (_) => const SettingsScreen(),
  ),
  GalleryPage(
    'profile',
    '个人主页',
    '封面 · 资料卡 · 标签页 · 帖子',
    'user-round',
    (_) => const ProfileScreen(),
  ),
  GalleryPage(
    'home',
    '首页',
    '排序标签 · 分类 · 帖子流 · 底部导航',
    'house',
    (_) => const HomeScreen(),
  ),
  GalleryPage(
    'notifications',
    '通知',
    '筛选 · 通知行 · 未读状态',
    'bell',
    (_) => const NotificationsScreen(),
  ),
  GalleryPage(
    'chat',
    '私信对话',
    '气泡 · 时间分组 · 输入栏',
    'message-circle',
    (_) => const ChatScreen(),
  ),
  GalleryPage(
    'login',
    '登录',
    '品牌 · 分段 · 表单 · 主操作',
    'login4',
    (_) => const LoginScreen(),
  ),
  GalleryPage(
    'dialog',
    '对话框示例',
    '在页面上打开确认对话框',
    'circle-help',
    (_) => const DialogDemoScreen(),
  ),
  GalleryPage(
    'sheet',
    '底部面板示例',
    '在页面上打开选择面板',
    'layers-filled',
    (_) => const SheetDemoScreen(),
  ),
];

class GalleryApp extends StatelessWidget {
  const GalleryApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<ThemeMode>(
      valueListenable: galleryThemeMode,
      builder: (context, mode, _) => MaterialApp(
        title: 'Gf Gallery',
        debugShowCheckedModeBanner: false,
        theme: gfThemeData(Brightness.light),
        darkTheme: gfThemeData(Brightness.dark),
        themeMode: mode,
        home: _launchHome() ?? const GalleryIndex(),
        onGenerateRoute: _route,
        builder: (context, child) {
          final dark = Theme.of(context).brightness == Brightness.dark;
          return AnnotatedRegion<SystemUiOverlayStyle>(
            value: dark
                ? SystemUiOverlayStyle.light
                : SystemUiOverlayStyle.dark,
            child: child!,
          );
        },
      ),
    );
  }

  Widget? _launchHome() {
    final id = galleryLaunchPage;
    if (id == null) return null;
    final page = [...catalogPages, ...screenPages].where((p) => p.id == id);
    return page.isEmpty ? null : Builder(builder: page.first.build);
  }

  Route<dynamic>? _route(RouteSettings settings) {
    final uri = Uri.tryParse(settings.name ?? '');
    if (uri == null) return null;
    final segments = uri.pathSegments;
    if (segments.isEmpty) return null;
    final id = segments.last;
    galleryInitialScroll = double.tryParse(uri.queryParameters['y'] ?? '') ?? 0;
    final theme = uri.queryParameters['theme'];
    if (theme != null) {
      galleryThemeMode.value = theme == 'dark'
          ? ThemeMode.dark
          : ThemeMode.light;
    }
    final page = [...catalogPages, ...screenPages].where((p) => p.id == id);
    if (page.isEmpty) return null;
    return MaterialPageRoute(builder: page.first.build, settings: settings);
  }
}

class GalleryIndex extends StatelessWidget {
  const GalleryIndex({super.key});

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    Widget row(GalleryPage page) => GfSettingRow(
      leading: GfIconTile(page.symbol, size: 32),
      title: page.title,
      description: page.subtitle,
      onTap: () =>
          Navigator.of(context).push(MaterialPageRoute(builder: page.build)),
    );
    return Scaffold(
      backgroundColor: colors.canvasGrouped,
      appBar: GfAppBar(
        title: const GfText('Gf 组件库'),
        showDivider: false,
        backgroundColor: colors.canvasGrouped,
        actions: [
          GfIconButton(
            symbol: dark ? 'sun' : 'moon',
            tooltip: '切换主题',
            color: colors.textPrimary,
            onPressed: () => galleryThemeMode.value = dark
                ? ThemeMode.light
                : ThemeMode.dark,
          ),
          const SizedBox(width: GfSpace.xs),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.only(top: GfSpace.sm, bottom: GfSpace.xxxl),
        children: [
          GfListSection(
            header: '组件',
            dividerIndent: GfSpace.lg + 32 + GfSpace.md,
            children: [for (final page in catalogPages) row(page)],
          ),
          const SizedBox(height: GfSpace.section),
          GfListSection(
            header: '示例页面',
            footer: '示例页面只组合组件库，不含业务逻辑与网络请求以外的数据。',
            dividerIndent: GfSpace.lg + 32 + GfSpace.md,
            children: [for (final page in screenPages) row(page)],
          ),
        ],
      ),
    );
  }
}
