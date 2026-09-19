import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/pages/campus/campus_explore_page.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/campus_shortcuts.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart'
    show EmptySessionsUserRepository, settingsPayloadJson;
import 'pages_smoke_test.dart' show MemoryTokenStorage, FakeTopicRepository;

class SurfacePageRepository extends PageRepository {
  SurfacePageRepository(super.client);
  @override
  Future<PagePayload> fetch(String path) async => PagePayload.fromJson(
    path == '/settings' ? settingsPayloadJson() : homePayloadJson(),
  );
}

class EmptyNotificationsRepository extends NotificationRepository {
  EmptyNotificationsRepository(super.client);
  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) async => const NotificationListResponse(
    items: [],
    nextCursor: 0,
    hasNext: false,
    unreadCount: 0,
  );
}

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'discovery routes and scaled settings fit a small ${brightness.name} phone',
      (tester) async {
        tester.view.physicalSize = const Size(320, 640);
        tester.view.devicePixelRatio = 1;
        tester.view.padding = const FakeViewPadding(top: 44, bottom: 34);
        tester.view.viewPadding = const FakeViewPadding(top: 44, bottom: 34);
        addTearDown(tester.view.reset);
        final client = GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'http://fake.local',
        );
        final container = ProviderContainer(
          overrides: [
            notificationRepositoryProvider.overrideWithValue(
              EmptyNotificationsRepository(client),
            ),
            pageRepositoryProvider.overrideWithValue(
              SurfacePageRepository(client),
            ),
            topicRepositoryProvider.overrideWithValue(
              FakeTopicRepository(client),
            ),
            userRepositoryProvider.overrideWithValue(
              EmptySessionsUserRepository(client),
            ),
            tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
            accountLayoutProvider.overrideWith(
              (_) async => LayoutPayload.fromJson(minimalLayoutJson()),
            ),
            campusCoursesProvider.overrideWith(
              (_) async => const CourseListResultPayload(
                list: [],
                page: 1,
                size: 3,
                total: 0,
                hasNext: false,
              ),
            ),
          ],
        );
        addTearDown(container.dispose);
        final router = GoRouter(
          initialLocation: '/search',
          routes: [
            GoRoute(path: '/search', builder: (_, _) => const SearchPage()),
            GoRoute(
              path: '/campus',
              builder: (_, _) => const CampusExplorePage(),
            ),
            GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
            GoRoute(
              path: '/notifications',
              builder: (_, _) => const NotificationsPage(),
            ),
            for (final path in ['/', '/courses', '/schedule', '/wiki'])
              GoRoute(
                path: path,
                builder: (_, _) => Scaffold(body: Text('destination:$path')),
              ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: gfThemeData(brightness),
              locale: const Locale('de'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: const TextScaler.linear(2)),
                child: child!,
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(
          find.byType(CampusShortcuts),
          200,
          scrollable: find
              .descendant(
                of: find.byType(ListView),
                matching: find.byType(Scrollable),
              )
              .first,
        );
        expect(find.byType(CampusShortcuts), findsOneWidget);
        expect(tester.takeException(), isNull);
        final l10n = AppLocalizations.of(
          tester.element(find.byType(CampusShortcuts)),
        );
        for (final (label, path) in [
          (l10n.coursesTitle, '/courses'),
          (l10n.scheduleTitle, '/schedule'),
          (l10n.wikiTitle, '/wiki'),
        ]) {
          final action = find.descendant(
            of: find.byType(CampusShortcuts),
            matching: find.text(label),
          );
          await tester.ensureVisible(action);
          await tester.tap(action);
          await tester.pumpAndSettle();
          expect(router.state.uri.path, path);
          router.pop();
          await tester.pumpAndSettle();
        }
        router.go('/campus');
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        router.go('/settings');
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final settingsTabs = tester.getRect(find.byType(GfTabBar));
        expect(settingsTabs.height, greaterThan(48));
        router.go('/notifications');
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        expect(
          tester.getRect(find.byIcon(Icons.notifications_none_rounded)).top,
          greaterThanOrEqualTo(tester.getRect(find.byType(GfTabBar)).bottom),
        );
        await tester.ensureVisible(find.text(l10n.navHome));
        await tester.pumpAndSettle();
        await tester.tap(find.text(l10n.navHome));
        await tester.pumpAndSettle();
        expect(router.state.uri.path, '/');
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      },
    );
  }
}
