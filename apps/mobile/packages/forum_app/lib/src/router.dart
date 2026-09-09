import 'dart:async';

import 'package:core/core.dart';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'navigation/tab_scroll_registry.dart';
import 'navigation/reading_chrome.dart';
import 'widgets/account_drawer.dart';
import 'pages/auth/login_page.dart';
import 'pages/admin/admin_page.dart';
import 'pages/campus/campus_page.dart';
import 'pages/content/content_page.dart';
import 'pages/category/category_page.dart';
import 'pages/courses/catalog_page.dart';
import 'pages/courses/detail_page.dart';
import 'pages/courses/my_reviews_page.dart';
import 'pages/drafts/drafts_page.dart';
import 'pages/home/home_page.dart';
import 'pages/info/site_info_page.dart';
import 'pages/messages/messages_page.dart';
import 'pages/notifications/notifications_page.dart';
import 'pages/profile/profile_page.dart';
import 'pages/publish/publish_page.dart';
import 'pages/wiki/wiki_home_page.dart';
import 'pages/wiki/wiki_page.dart';
import 'pages/wiki/wiki_search_page.dart';
import 'pages/schedule/schedule_page.dart';
import 'pages/search/search_page.dart';
import 'pages/settings/settings_page.dart';
import 'pages/topic/topic_page.dart';
import 'providers.dart';
import 'current_user.dart';

extension on GfShellDestination {
  IconData get icon => switch (this) {
    GfShellDestination.home => Icons.home_outlined,
    GfShellDestination.campus => Icons.school_outlined,
    GfShellDestination.messages => Icons.forum_outlined,
    GfShellDestination.notifications => Icons.notifications_outlined,
  };

  IconData get activeIcon => switch (this) {
    GfShellDestination.home => Icons.home,
    GfShellDestination.campus => Icons.school,
    GfShellDestination.messages => Icons.forum,
    GfShellDestination.notifications => Icons.notifications,
  };

  String label(AppLocalizations l10n) => switch (this) {
    GfShellDestination.home => l10n.navHome,
    GfShellDestination.campus => l10n.navCampus,
    GfShellDestination.messages => l10n.navMessages,
    GfShellDestination.notifications => l10n.notificationsTitle,
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
    ref.read(readingChromeProvider).show();
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

    final chrome = ref.watch(readingChromeProvider);
    final duration = MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : const Duration(milliseconds: 200);
    return Scaffold(
      drawer: const AccountDrawer(),
      onDrawerChanged: (_) => ref.read(readingChromeProvider).show(),
      body: NotificationListener<ScrollNotification>(
        onNotification: (notification) {
          if (notification.depth != 0 ||
              notification.metrics.axis != Axis.vertical) {
            return false;
          }
          if (notification is ScrollUpdateNotification) {
            ref
                .read(readingChromeProvider)
                .update(
                  notification.scrollDelta ?? 0,
                  notification.metrics.pixels,
                  locked:
                      MediaQuery.viewInsetsOf(context).bottom > 0 ||
                      ModalRoute.of(context)?.isCurrent == false,
                );
          }
          return false;
        },
        child: Stack(
          children: [
            Positioned.fill(child: widget.navigationShell),
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: AnimatedSlide(
                offset: chrome.hidden ? const Offset(0, 1) : Offset.zero,
                duration: duration,
                curve: Curves.easeOut,
                child: IgnorePointer(
                  ignoring: chrome.hidden,
                  child: ExcludeSemantics(
                    excluding: chrome.hidden,
                    child: GfBottomNavigation(
                      currentIndex: widget.navigationShell.currentIndex,
                      onSelected: _selectDestination,
                      showLabels: false,
                      items: [
                        for (final destination in GfShellDestination.values)
                          GfBottomNavigationItem(
                            icon: destination.icon,
                            selectedIcon: destination.activeIcon,
                            symbol: switch (destination) {
                              GfShellDestination.home => 'house',
                              GfShellDestination.campus => 'graduation-cap',
                              GfShellDestination.notifications => 'bell',
                              GfShellDestination.messages => 'mail',
                            },
                            label: destination.label(l10n),
                            badge:
                                destination == GfShellDestination.notifications
                                ? _unreadNotifications
                                : destination == GfShellDestination.messages &&
                                      _unreadMessages,
                          ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
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

final appNavigatorKey = GlobalKey<NavigatorState>();
final GoRouter appRouter = GoRouter(
  navigatorKey: appNavigatorKey,
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
            GoRoute(path: '/campus', builder: (_, _) => const CampusPage()),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/notifications',
              builder: (_, _) => const NotificationsPage(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/messages',
              redirect: (_, state) =>
                  int.tryParse(state.uri.queryParameters['userId'] ?? '') !=
                      null
                  ? Uri(
                      path: '/chat',
                      queryParameters: state.uri.queryParameters,
                    ).toString()
                  : null,
              builder: (_, _) => const MessagesPage(),
            ),
          ],
        ),
      ],
    ),
    GoRoute(
      path: '/chat',
      redirect: (_, state) =>
          (int.tryParse(state.uri.queryParameters['userId'] ?? '') ?? 0) > 0
          ? null
          : '/messages',
      builder: (_, state) => MessagesPage(
        targetUserId: int.tryParse(state.uri.queryParameters['userId'] ?? ''),
        targetUsername: state.uri.queryParameters['username'] ?? '',
        targetAvatarUrl: state.uri.queryParameters['avatar'] ?? '',
      ),
    ),
    GoRoute(path: '/search', builder: (_, _) => const SearchPage()),
    GoRoute(
      path: '/publish',
      builder: (BuildContext context, GoRouterState state) => PublishPage(
        topicId: publishTopicIdFromUri(state.uri),
        initialContentType: switch (state.uri.queryParameters['type'] ??
            state.uri.queryParameters['contentType']) {
          '1' || 'question' => 1,
          '3' || 'article' => 3,
          _ => 2,
        },
      ),
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
      builder: (BuildContext context, GoRouterState state) => TopicPage(
        topicId: int.parse(state.pathParameters['postId']!),
        initialPostNo: int.tryParse(state.uri.queryParameters['postNo'] ?? ''),
      ),
    ),
    GoRoute(
      path: '/u/:userId',
      builder: (BuildContext context, GoRouterState state) =>
          ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
    ),
    GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
    GoRoute(
      path: '/settings/:section',
      builder: (_, state) =>
          SettingsPage(initialSection: state.pathParameters['section']),
    ),
    GoRoute(
      path: '/my-course-reviews',
      builder: (_, _) => const MyCourseReviewsPage(),
    ),
    GoRoute(path: '/my-content', builder: (_, _) => const ContentPage()),
    GoRoute(
      path: '/recycle-bin',
      builder: (_, _) => const ContentPage(deleted: true),
    ),
    GoRoute(
      path: '/profile',
      builder: (_, state) => ProfilePage(
        initialStream: state.uri.queryParameters['stream'] == 'bookmarks'
            ? 'bookmarks'
            : 'timeline',
      ),
    ),
    GoRoute(
      path: '/moderation',
      builder: (_, _) => const AdminPage(target: MobileWebTarget.moderation),
    ),
    GoRoute(path: '/about', builder: (_, _) => const SiteInfoIndexPage()),
    for (final kind in SiteInfoKind.values)
      GoRoute(
        path: '/${kind.name}',
        builder: (_, _) => SiteInfoPage(kind: kind),
      ),
    GoRoute(
      path: '/moderation/courses',
      builder: (_, _) =>
          const AdminPage(target: MobileWebTarget.courseManagement),
    ),
    GoRoute(
      path: '/moderation/course-reviews',
      builder: (_, _) => const AdminPage(target: MobileWebTarget.courseReviews),
    ),
    GoRoute(path: '/admin', builder: (_, _) => const AdminPage()),
    GoRoute(path: '/login', builder: (_, _) => const LoginPage()),
    GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
    GoRoute(path: '/schedule', builder: (_, _) => const SchedulePage()),
    GoRoute(path: '/courses', builder: (_, _) => const CourseCatalogPage()),
    GoRoute(
      path: '/courses/:courseId',
      builder: (BuildContext context, GoRouterState state) => CourseDetailPage(
        courseId: int.parse(state.pathParameters['courseId']!),
        focusReviewId: int.tryParse(
          state.uri.queryParameters['reviewId'] ?? '',
        ),
        focusOfferingId: int.tryParse(
          state.uri.queryParameters['offeringId'] ?? '',
        ),
      ),
    ),
    GoRoute(path: '/wiki/search', builder: (_, _) => const WikiSearchPage()),
    GoRoute(path: '/wiki', builder: (_, _) => const WikiHomePage()),
    GoRoute(
      // 多段 wiki 路径（如 /wiki/guide/getting-started）经 (.*) 通配捕获；
      // go_router 对 path 参数自动 percent-decode，页面内按段重新编码。
      path: '/wiki/:wikiPath(.*)',
      builder: (BuildContext context, GoRouterState state) => WikiPage(
        wikiPath: state.pathParameters['wikiPath']!,
        // state.uri.fragment 保留 percent-encoded 态;解码后再传给 scrollToAnchor。
        initialAnchor: decodeWikiAnchor(state.uri.fragment),
      ),
    ),
  ],
);
