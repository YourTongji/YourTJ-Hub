import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'navigation/tab_scroll_registry.dart';
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
import 'current_user.dart';

extension on GfShellDestination {
  IconData get icon => switch (this) {
    GfShellDestination.home => Icons.home_outlined,
    GfShellDestination.search => Icons.search_outlined,
    GfShellDestination.messages => Icons.forum_outlined,
    GfShellDestination.profile => Icons.person_outline,
  };

  IconData get activeIcon => switch (this) {
    GfShellDestination.home => Icons.home,
    GfShellDestination.search => Icons.search,
    GfShellDestination.messages => Icons.forum,
    GfShellDestination.profile => Icons.person,
  };

  String label(AppLocalizations l10n) => switch (this) {
    GfShellDestination.home => l10n.navHome,
    GfShellDestination.search => l10n.navSearch,
    GfShellDestination.messages => l10n.navMessages,
    GfShellDestination.profile => l10n.navProfile,
  };
}

/// Persistent mobile shell with four navigation destinations and one compose
/// action. Each branch owns its own navigator and state; compose is pushed as
/// a global page rather than kept alive as a destination.
class GfShell extends ConsumerStatefulWidget {
  const GfShell({super.key, required this.navigationShell});

  final StatefulNavigationShell navigationShell;

  @override
  ConsumerState<GfShell> createState() => _GfShellState();
}

class _GfShellState extends ConsumerState<GfShell> {
  Timer? _unreadTimer;
  bool _unreadNotifications = false;
  bool _unreadMessages = false;

  @override
  void initState() {
    super.initState();
    unawaited(_purgeStaleOfflineCacheOnBoot());
    _pollUnread();
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

  /// 启动兜底:无令牌(上次 401 清库可能被进程中断)时清空离线缓存,
  /// 防止未登录态读取上一账号残留的私信/话题。清理失败不影响启动。
  Future<void> _purgeStaleOfflineCacheOnBoot() async {
    try {
      if (await hasSessionToken(ref.read(tokenStorageProvider))) return;
      await clearOfflineCacheQuietly(
        ref.read(offlineTopicCacheProvider),
        ref.read(offlineChatCacheProvider),
      );
    } catch (_) {
      // 兜底清理失败(缓存不可用)不阻塞启动。
    }
  }

  Future<void> _pollUnread() async {
    try {
      final String? token = await ref.read(tokenStorageProvider).read();
      if (token == null || token.isEmpty) return;
    } catch (_) {
      return;
    }
    try {
      final status = await ref
          .read(notificationRepositoryProvider)
          .getUnreadStatus();
      if (!mounted) return;
      if (_unreadNotifications == status.notifications &&
          _unreadMessages == status.messages) {
        return;
      }
      setState(() {
        _unreadNotifications = status.notifications;
        _unreadMessages = status.messages;
      });
    } catch (_) {
      // Unread state is best-effort and never blocks navigation.
    }
  }

  void _selectDestination(int index) {
    if (index == widget.navigationShell.currentIndex) {
      ref
          .read(tabScrollRegistryProvider)
          .scrollToTop(GfShellDestination.values[index]);
      return;
    }
    widget.navigationShell.goBranch(index);
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    ref.listen(unauthorizedEventsProvider, (int? previous, int next) {
      if (next > (previous ?? 0) &&
          mounted &&
          GoRouter.of(context).state.uri.path != '/login') {
        // 401 即会话边界:使缓存的当前用户身份失效(旧账号 id 不再被
        // 后续新 shell 读取),用 go 替换导航栈销毁保留旧账号内存态的
        // shell;重新登录后 go('/') 得到全新 shell。
        ref.invalidate(currentUserProvider);
        context.go('/login');
      }
    });

    return Scaffold(
      body: widget.navigationShell,
      bottomNavigationBar: GfBottomNavigation(
        currentIndex: widget.navigationShell.currentIndex,
        onSelected: _selectDestination,
        onAction: () => context.push('/publish'),
        actionLabel: l10n.navPublish,
        items: <GfBottomNavigationItem>[
          for (final GfShellDestination destination
              in GfShellDestination.values)
            GfBottomNavigationItem(
              icon: destination.icon,
              selectedIcon: destination.activeIcon,
              label: destination.label(l10n),
              badge: destination == GfShellDestination.messages
                  ? _unreadMessages
                  : destination == GfShellDestination.profile
                  ? _unreadNotifications
                  : false,
            ),
        ],
      ),
    );
  }
}

/// Resolves the topic being edited from both the mobile link and the
/// server-authored draft edit URL.
int? publishTopicIdFromUri(Uri uri) {
  final String rawTopicId =
      uri.queryParameters['topicId'] ?? uri.queryParameters['id'] ?? '';
  return int.tryParse(rawTopicId);
}

final GoRouter appRouter = GoRouter(
  initialLocation: '/',
  routes: <RouteBase>[
    StatefulShellRoute.indexedStack(
      builder:
          (
            BuildContext context,
            GoRouterState state,
            StatefulNavigationShell navigationShell,
          ) => GfShell(navigationShell: navigationShell),
      branches: <StatefulShellBranch>[
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(path: '/', builder: (_, _) => const HomePage()),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(path: '/search', builder: (_, _) => const SearchPage()),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/messages',
              builder: (BuildContext context, GoRouterState state) =>
                  MessagesPage(
                    targetUserId: int.tryParse(
                      state.uri.queryParameters['userId'] ?? '',
                    ),
                    targetUsername: state.uri.queryParameters['username'] ?? '',
                    targetAvatarUrl: state.uri.queryParameters['avatar'] ?? '',
                  ),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(path: '/profile', builder: (_, _) => const ProfilePage()),
          ],
        ),
      ],
    ),
    GoRoute(
      path: '/publish',
      builder: (BuildContext context, GoRouterState state) =>
          PublishPage(topicId: publishTopicIdFromUri(state.uri)),
    ),
    GoRoute(
      path: '/c/:slug/:id',
      builder: (BuildContext context, GoRouterState state) => CategoryPage(
        slug: state.pathParameters['slug']!,
        categoryId: int.parse(state.pathParameters['id']!),
      ),
    ),
    GoRoute(
      path: '/p/:postId',
      builder: (BuildContext context, GoRouterState state) =>
          TopicPage(topicId: int.parse(state.pathParameters['postId']!)),
    ),
    GoRoute(
      path: '/u/:userId',
      builder: (BuildContext context, GoRouterState state) =>
          ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
    ),
    GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
    GoRoute(
      path: '/notifications',
      builder: (_, _) => const NotificationsPage(),
    ),
    GoRoute(path: '/login', builder: (_, _) => const LoginPage()),
    GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
  ],
);
