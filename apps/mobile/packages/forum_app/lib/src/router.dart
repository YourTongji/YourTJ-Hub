import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'navigation/auth_navigation.dart';
import 'navigation/session_overlays.dart';
import 'navigation/tab_scroll_registry.dart';
import 'navigation/route_visibility.dart';
import 'navigation/reading_chrome.dart';
import 'navigation/reading_window.dart';
import 'widgets/account_drawer.dart';
import 'pages/auth/login_page.dart';
import 'pages/admin/admin_page.dart';
import 'pages/campus/campus_page.dart';
import 'pages/campus/campus_explore_page.dart';
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
import 'pages/settings/schedule_widget_settings_page.dart';
import 'pages/topic/topic_page.dart';
import 'providers.dart';
import 'current_user.dart';
import 'realtime/foreground_realtime.dart';
import 'realtime/realtime_updates.dart';

extension on GfShellDestination {
  String get symbol => switch (this) {
    GfShellDestination.home => 'house',
    GfShellDestination.campus => 'graduation-cap',
    GfShellDestination.messages => 'mail',
    GfShellDestination.notifications => 'bell',
  };

  String label(AppLocalizations l10n) => switch (this) {
    GfShellDestination.home => l10n.navHome,
    GfShellDestination.campus => l10n.navCampus,
    GfShellDestination.messages => l10n.navMessages,
    GfShellDestination.notifications => l10n.notificationsTitle,
  };
}

/// Opens the account drawer from the leading content region while taking part
/// in the same arena as descendants. Horizontal rails, sliders and text fields
/// can claim their own drags; a vertical reading gesture is never cancelled.
class _DrawerSwipeGestureRecognizer extends HorizontalDragGestureRecognizer {
  _DrawerSwipeGestureRecognizer()
    : super(supportedDevices: const {PointerDeviceKind.touch}) {
    onlyAcceptDragOnThreshold = true;
  }

  double openingWidth = 0;
  final Map<int, Offset> _origins = {};

  @override
  bool isPointerAllowed(PointerEvent event) =>
      event.localPosition.dx >= 0 &&
      event.localPosition.dx <= openingWidth &&
      super.isPointerAllowed(event);

  @override
  void addAllowedPointer(PointerDownEvent event) {
    _origins[event.pointer] = event.position;
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    final origin = _origins[event.pointer];
    if (origin != null && event is PointerMoveEvent) {
      final delta = event.position - origin;
      final slop = computeHitSlop(event.kind, gestureSettings);
      if (delta.dx < -slop ||
          (delta.dy.abs() >= slop && delta.dy.abs() > delta.dx)) {
        resolve(GestureDisposition.rejected);
        return;
      }
    }
    if (event is PointerUpEvent || event is PointerCancelEvent) {
      _origins.remove(event.pointer);
    }
    super.handleEvent(event);
  }

  @override
  bool hasSufficientGlobalDistanceToAccept(
    PointerDeviceKind pointerDeviceKind,
    double? deviceTouchSlop,
  ) => globalDistanceMoved > computeHitSlop(pointerDeviceKind, gestureSettings);

  @override
  void rejectGesture(int pointer) {
    _origins.remove(pointer);
    super.rejectGesture(pointer);
  }

  @override
  void dispose() {
    _origins.clear();
    super.dispose();
  }
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

class _GfShellState extends ConsumerState<GfShell> with WidgetsBindingObserver {
  final _scaffoldKey = GlobalKey<ScaffoldState>();
  late ForegroundRealtimeCoordinator _realtime;
  late int _realtimeEpoch;
  bool _realtimeSessionStarted = false;
  bool _activeInTree = true;
  Future<void>? _unreadInFlight;
  bool _unreadDirty = false;
  CancelToken? _unreadCancel;
  bool _unreadNotifications = false;
  bool _unreadMessages = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    routeVisibilityChanges.addListener(_onRouteVisibilityChanged);
    _realtimeEpoch = ref.read(offlineCacheEpochProvider);
    _realtime = _createRealtime();
    unawaited(_purgeStaleOfflineCacheOnBoot());
    unawaited(_pollUnread());
    _onRouteVisibilityChanged();
  }

  ForegroundRealtimeCoordinator _createRealtime() {
    return ForegroundRealtimeCoordinator(
      readToken: ref.read(tokenStorageProvider).read,
      connect: ref.read(realtimeConnectProvider),
      onResync: () {
        if (!mounted || !_activeInTree) return;
        ref.read(realtimeInvalidationsProvider.notifier).resync();
        unawaited(_pollUnread());
      },
      onEvent: _handleRealtimeEvent,
      onFallbackTick: () {
        if (!mounted || !_activeInTree) return;
        ref.read(realtimeInvalidationsProvider.notifier).notifications();
        unawaited(_pollUnread());
      },
      onHealthChanged: (healthy) {
        if (mounted && _activeInTree) {
          ref.read(realtimeHealthyProvider.notifier).setHealthy(healthy);
        }
      },
    );
  }

  void _onRouteVisibilityChanged() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _activeInTree) {
        unawaited(_ensureRealtimeForCurrentSession());
      }
    });
  }

  Future<void> _ensureRealtimeForCurrentSession() async {
    if (!mounted ||
        !_activeInTree ||
        (WidgetsBinding.instance.lifecycleState != null &&
            WidgetsBinding.instance.lifecycleState !=
                AppLifecycleState.resumed) ||
        !routeIsUncovered(context)) {
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    if (_realtimeEpoch != epoch) {
      _realtime.stop();
      _realtimeEpoch = epoch;
      _realtimeSessionStarted = false;
      _realtime = _createRealtime();
    }
    if (_realtimeSessionStarted) return;
    try {
      final token = await ref.read(tokenStorageProvider).read();
      if (!mounted ||
          !_activeInTree ||
          epoch != ref.read(offlineCacheEpochProvider) ||
          !routeIsUncovered(context) ||
          (WidgetsBinding.instance.lifecycleState != null &&
              WidgetsBinding.instance.lifecycleState !=
                  AppLifecycleState.resumed)) {
        return;
      }
      if (token == null || token.isEmpty) return;
      _realtime.start();
      _realtimeSessionStarted = true;
      unawaited(_pollUnread());
    } catch (_) {
      // Token storage errors leave the public shell usable; another route or
      // foreground transition can retry the connection.
    }
  }

  @override
  void deactivate() {
    _realtime.stop();
    _realtimeSessionStarted = false;
    _activeInTree = false;
    super.deactivate();
  }

  @override
  void activate() {
    super.activate();
    _activeInTree = true;
    _onRouteVisibilityChanged();
  }

  @override
  void dispose() {
    _activeInTree = false;
    _realtime.stop();
    _unreadCancel?.cancel('shell disposed');
    shellDrawerOpen.value = false;
    routeVisibilityChanges.removeListener(_onRouteVisibilityChanged);
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      unawaited(_ensureRealtimeForCurrentSession());
    } else {
      _realtime.stop();
      _realtimeSessionStarted = false;
      _unreadCancel?.cancel('application backgrounded');
    }
  }

  void _handleRealtimeEvent(ForumSseFrame frame) {
    if (!mounted) return;
    switch (frame.event) {
      case 'chat.changed':
        try {
          final event = ForumRealtimeChatChanged.fromJson(
            jsonDecode(frame.data) as Map<String, dynamic>,
          );
          ref.read(realtimeInvalidationsProvider.notifier).chat(event.convId);
        } catch (_) {
          // Unknown or malformed hints cannot replace REST as truth.
        }
      case 'notifications.changed':
        ref.read(realtimeInvalidationsProvider.notifier).notifications();
      case 'unread.changed':
        unawaited(_pollUnread());
      case 'session.invalidated':
        ref.read(apiClientProvider).onUnauthorized?.call();
    }
  }

  /// 启动兜底:无令牌(上次 401 清库可能被进程中断)时清空离线缓存,
  /// 防止未登录态读取上一账号残留的私信/话题。清理失败不影响启动。
  Future<void> _purgeStaleOfflineCacheOnBoot() async {
    try {
      if (await hasSessionToken(ref.read(tokenStorageProvider))) return;
      await clearOfflineCacheQuietly(
        ref.read(offlineTopicCacheProvider),
        ref.read(offlineChatCacheProvider),
        ref.read(scheduleWidgetBridgeProvider),
      );
    } catch (_) {
      // 兜底清理失败(缓存不可用)不阻塞启动。
    }
  }

  Future<void> _pollUnread() {
    if (_unreadInFlight case final inFlight?) {
      _unreadDirty = true;
      return inFlight;
    }
    final request = _fetchUnread();
    _unreadInFlight = request;
    unawaited(
      request.whenComplete(() {
        _unreadInFlight = null;
        if (_unreadDirty && mounted && _activeInTree) {
          _unreadDirty = false;
          unawaited(_pollUnread());
        }
      }),
    );
    return request;
  }

  Future<void> _fetchUnread() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final cancel = _unreadCancel = CancelToken();
    try {
      final String? token = await ref.read(tokenStorageProvider).read();
      if (token == null || token.isEmpty) return;
      final status = await ref
          .read(notificationRepositoryProvider)
          .getUnreadStatus(cancelToken: cancel);
      if (!mounted ||
          cancel.isCancelled ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
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
    } finally {
      if (identical(_unreadCancel, cancel)) _unreadCancel = null;
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
    ref.listen(offlineCacheEpochProvider, (int? previous, int next) {
      if (next != previous) {
        _realtime.stop();
        _realtimeSessionStarted = false;
        _unreadCancel?.cancel('session changed');
        _unreadDirty = false;
      }
    });

    final destinations = [
      for (final destination in GfShellDestination.values)
        GfBottomNavigationItem(
          symbol: destination.symbol,
          selectedSymbol: '${destination.symbol}-filled',
          label: destination.label(l10n),
          badge: destination == GfShellDestination.notifications
              ? _unreadNotifications
              : destination == GfShellDestination.messages && _unreadMessages,
        ),
    ];

    final chrome = ref.watch(readingChromeProvider);
    final duration = MediaQuery.disableAnimationsOf(context)
        ? Duration.zero
        : const Duration(milliseconds: 200);
    return Scaffold(
      key: _scaffoldKey,
      drawer: const AccountDrawer(),
      // The opening gesture belongs below the drawer overlay so descendants
      // can win it. The native drawer still handles dragging an open panel shut.
      drawerEnableOpenDragGesture: false,
      onDrawerChanged: (open) {
        shellDrawerOpen.value = open;
        ref.read(readingChromeProvider).show();
        if (open) ref.invalidate(accountCardProvider);
      },
      body: RawGestureDetector(
        behavior: HitTestBehavior.translucent,
        excludeFromSemantics: true,
        gestures: {
          _DrawerSwipeGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                _DrawerSwipeGestureRecognizer
              >(
                _DrawerSwipeGestureRecognizer.new,
                (recognizer) => recognizer
                  ..openingWidth = MediaQuery.sizeOf(context).width * .55
                  ..onStart = (_) {
                    if (_scaffoldKey.currentState?.isDrawerOpen != true) {
                      _scaffoldKey.currentState?.openDrawer();
                    }
                  },
              ),
        },
        child: NotificationListener<ScrollNotification>(
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
          child: ReadingWindow(
            maxContentWidth: widget.navigationShell.currentIndex == 1
                ? 1120
                : 720,
            rail: ReadingNavigationRail(
              currentIndex: widget.navigationShell.currentIndex,
              onSelected: _selectDestination,
              items: destinations,
            ),
            bottomNavigation: AnimatedSlide(
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
                    items: destinations,
                  ),
                ),
              ),
            ),
            child: widget.navigationShell,
          ),
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
final appSessionOverlays = SessionOverlayRegistry();
final GoRouter appRouter = GoRouter(
  navigatorKey: appNavigatorKey,
  initialLocation: '/',
  observers: [VisibilityRouteObserver(), appSessionOverlays.observer()],
  redirect: (context, state) => authNavigationRedirect(
    requested: state.uri,
    previousLocation: appRouter.routerDelegate.currentConfiguration.isEmpty
        ? null
        : appRouter.state.uri.toString(),
    tokenStorage: ProviderScope.containerOf(
      context,
      listen: false,
    ).read(tokenStorageProvider),
  ),
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
          observers: [VisibilityRouteObserver(), appSessionOverlays.observer()],
          routes: <RouteBase>[
            GoRoute(path: '/', builder: (_, _) => const HomePage()),
          ],
        ),
        StatefulShellBranch(
          observers: [VisibilityRouteObserver(), appSessionOverlays.observer()],
          routes: <RouteBase>[
            GoRoute(path: '/campus', builder: (_, _) => const CampusPage()),
          ],
        ),
        StatefulShellBranch(
          observers: [VisibilityRouteObserver(), appSessionOverlays.observer()],
          routes: [
            GoRoute(
              path: '/notifications',
              builder: (_, _) => const NotificationsPage(),
            ),
          ],
        ),
        StatefulShellBranch(
          observers: [VisibilityRouteObserver(), appSessionOverlays.observer()],
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
        localDraftKey: state.uri.queryParameters['local'],
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
    for (final stream in ['following', 'followers'])
      GoRoute(
        path: '/u/:userId/$stream',
        builder: (_, state) => ProfilePage.connections(
          userId: int.parse(state.pathParameters['userId']!),
          initialStream: stream,
        ),
      ),
    GoRoute(
      path: '/u/:userId',
      builder: (BuildContext context, GoRouterState state) =>
          ProfilePage(userId: int.parse(state.pathParameters['userId']!)),
    ),
    GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
    GoRoute(
      path: '/settings/widgets',
      builder: (_, _) => const ScheduleWidgetSettingsPage(),
    ),
    GoRoute(
      path: '/settings/:section',
      builder: (_, state) => SettingsPage(
        initialSection: state.pathParameters['section'],
        autoEditProfile: state.uri.queryParameters['edit'] == '1',
      ),
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
      builder: (_, state) => switch (state.uri.queryParameters['stream']) {
        'following' || 'followers' => ProfilePage.connections(
          initialStream: state.uri.queryParameters['stream']!,
        ),
        'bookmarks' => const ProfilePage(initialStream: 'bookmarks'),
        _ => const ProfilePage(),
      },
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
    GoRoute(path: '/campus/official', redirect: (_, _) => '/campus'),
    GoRoute(
      path: '/campus/explore',
      builder: (_, _) => const CampusExplorePage(),
    ),
    GoRoute(path: '/admin', builder: (_, _) => const AdminPage()),
    GoRoute(path: '/login', builder: (_, _) => const LoginPage()),
    GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
    GoRoute(path: '/schedule', builder: (_, _) => const SchedulePage()),
    GoRoute(
      path: '/courses',
      builder: (_, state) =>
          CourseCatalogPage(initialQuery: state.uri.queryParameters['q'] ?? ''),
    ),
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
    GoRoute(
      path: '/wiki/search',
      builder: (_, state) =>
          WikiSearchPage(initialQuery: state.uri.queryParameters['q'] ?? ''),
    ),
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
