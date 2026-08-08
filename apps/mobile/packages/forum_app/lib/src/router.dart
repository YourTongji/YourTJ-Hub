import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'pages/auth/login_page.dart';
import 'pages/category/category_page.dart';
import 'pages/drafts/drafts_page.dart';
import 'pages/home/home_page.dart';
import 'pages/messages/messages_page.dart';
import 'pages/notifications/notifications_page.dart';
import 'pages/profile/profile_page.dart';
import 'pages/publish/publish_page.dart';
import 'pages/search/search_page.dart';
import 'pages/settings/settings_page.dart';
import 'pages/topic/topic_page.dart';
import 'providers.dart';

/// 应用底部导航项(对齐 web AppShell 移动端导航)。
/// label 在 build 内经 AppLocalizations 映射(zh/en 一期)。
enum GfTab {
  home(Icons.home_outlined, Icons.home),
  search(Icons.search, Icons.search),
  publish(Icons.edit, Icons.edit),
  messages(Icons.forum_outlined, Icons.forum),
  profile(Icons.person_outline, Icons.person);

  const GfTab(this.icon, this.activeIcon);

  final IconData icon;
  final IconData activeIcon;

  /// 当前语言下的导航标签。
  String label(AppLocalizations l10n) => switch (this) {
    GfTab.home => l10n.navHome,
    GfTab.search => l10n.navSearch,
    GfTab.publish => l10n.navPublish,
    GfTab.messages => l10n.navMessages,
    GfTab.profile => l10n.navProfile,
  };
}

/// 底部导航壳:五个 tab 对应 web 端移动端导航入口。
///
/// 轮询 unread-status 驱动消息/通知角标(与后端无 WebSocket 的现状一致)。
class GfShell extends ConsumerStatefulWidget {
  const GfShell({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<GfShell> createState() => _GfShellState();
}

class _GfShellState extends ConsumerState<GfShell> {
  int _index = 0;
  Timer? _unreadTimer;

  /// 未读状态:通知/私信角标。
  bool _unreadNotifications = false;
  bool _unreadMessages = false;

  static const List<String> _paths = [
    '/',
    '/search',
    '/publish',
    '/messages',
    '/profile',
  ];

  @override
  void initState() {
    super.initState();
    _pollUnread();
    // 前台 30s 轮询 unread-status(通知/私信角标)。
    _unreadTimer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => _pollUnread(),
    );
  }

  @override
  void dispose() {
    _unreadTimer?.cancel();
    super.dispose();
  }

  Future<void> _pollUnread() async {
    // 未登录不轮询(避免匿名 401 被误判为会话失效)。
    try {
      final token = await ref.read(tokenStorageProvider).read();
      if (token == null || token.isEmpty) return;
    } catch (_) {
      return;
    }
    try {
      final status = await ref
          .read(notificationRepositoryProvider)
          .getUnreadStatus();
      if (mounted) {
        setState(() {
          _unreadNotifications = status.notifications;
          _unreadMessages = status.messages;
        });
      }
    } catch (_) {
      // 网络失败时静默。
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final int routeIndex = _paths.indexOf(GoRouterState.of(context).uri.path);
    final int effectiveIndex = routeIndex < 0 ? _index : routeIndex;

    // 401 会话失效:清会话并回登录页(ref.listen 必须在 build 中注册)。
    ref.listen(unauthorizedEventsProvider, (prev, next) {
      if (next > (prev ?? 0) && mounted) {
        context.go('/login');
      }
    });

    return Scaffold(
      body: IndexedStack(
        index: effectiveIndex,
        children: [
          widget.child,
          const SearchPage(),
          const PublishPage(),
          const MessagesPage(),
          const ProfilePage(),
        ],
      ),
      bottomNavigationBar: GfBottomNavigation(
        currentIndex: effectiveIndex,
        onSelected: (int i) {
          setState(() => _index = i);
          final String path = _paths[i];
          if (GoRouter.of(context).state.uri.path != path) {
            context.go(path);
          }
        },
        items: <GfBottomNavigationItem>[
          GfBottomNavigationItem(
            icon: GfTab.home.icon,
            selectedIcon: GfTab.home.activeIcon,
            label: GfTab.home.label(l10n),
          ),
          GfBottomNavigationItem(
            icon: GfTab.search.icon,
            selectedIcon: GfTab.search.activeIcon,
            label: GfTab.search.label(l10n),
          ),
          GfBottomNavigationItem(
            icon: GfTab.publish.icon,
            selectedIcon: GfTab.publish.activeIcon,
            label: GfTab.publish.label(l10n),
          ),
          GfBottomNavigationItem(
            icon: GfTab.messages.icon,
            selectedIcon: GfTab.messages.activeIcon,
            label: GfTab.messages.label(l10n),
            badge: _unreadMessages,
          ),
          GfBottomNavigationItem(
            icon: GfTab.profile.icon,
            selectedIcon: GfTab.profile.activeIcon,
            label: GfTab.profile.label(l10n),
            badge: _unreadNotifications,
          ),
        ],
      ),
    );
  }
}

/// 路由表:与 web 端路径语义对应。
///
/// - `/` 首页
/// - `/c/:slug/:id` 分类页
/// - `/p/:postId` 话题详情(web: /p/post/:id)
/// - `/u/:userId` 用户主页
/// - `/search` `/messages` `/publish` 底部 tab
/// - `/settings` 设置页
/// - `/drafts` 草稿列表
final GoRouter appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    ShellRoute(
      builder: (context, state, child) => GfShell(child: child),
      routes: [
        GoRoute(path: '/', builder: (context, state) => const HomePage()),
        GoRoute(
          path: '/search',
          builder: (context, state) => const SearchPage(),
        ),
        GoRoute(
          path: '/publish',
          builder: (context, state) => const PublishPage(),
        ),
        GoRoute(
          path: '/messages',
          builder: (context, state) => const MessagesPage(),
        ),
        GoRoute(
          path: '/profile',
          builder: (context, state) => const ProfilePage(),
        ),
      ],
    ),
    GoRoute(
      path: '/c/:slug/:id',
      builder: (context, state) => CategoryPage(
        slug: state.pathParameters['slug']!,
        categoryId: int.parse(state.pathParameters['id']!),
      ),
    ),
    GoRoute(
      path: '/p/:postId',
      builder: (context, state) =>
          TopicPage(topicId: int.parse(state.pathParameters['postId']!)),
    ),
    GoRoute(
      path: '/u/:userId',
      builder: (context, state) =>
          ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
    ),
    GoRoute(
      path: '/settings',
      builder: (context, state) => const SettingsPage(),
    ),
    GoRoute(
      path: '/notifications',
      builder: (context, state) => const NotificationsPage(),
    ),
    GoRoute(path: '/login', builder: (context, state) => const LoginPage()),
    GoRoute(path: '/drafts', builder: (context, state) => const DraftsPage()),
  ],
);
