import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation/shell_sections.dart';
import 'package:forum_app/src/navigation/tab_scroll_registry.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show MemTokenStorage, NoopCache;

void main() {
  test('a requested section is taken once, by its own destination', () {
    final sections = ShellSections();
    addTearDown(sections.dispose);
    var notified = 0;
    sections.addListener(() => notified++);

    sections.request(GfShellDestination.campus, 'timetable');
    expect(notified, 1);
    expect(sections.take(GfShellDestination.notifications), isNull);
    expect(sections.take(GfShellDestination.campus), 'timetable');
    expect(sections.take(GfShellDestination.campus), isNull);
  });

  test('reporting the shown section is silent', () {
    final sections = ShellSections();
    addTearDown(sections.dispose);
    var notified = 0;
    sections.addListener(() => notified++);

    sections.report(GfShellDestination.notifications, 'likes');
    expect(sections.currentOf(GfShellDestination.notifications), 'likes');
    expect(notified, 0);
  });

  testWidgets('dock menus check a section only on their own page', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(MemTokenStorage()),
        currentUserProvider.overrideWith(
          (_) async => const CurrentUser(id: 1, username: 'alice'),
        ),
        offlineTopicCacheProvider.overrideWithValue(NoopCache()),
        offlineChatCacheProvider.overrideWithValue(NoopCache()),
        accountLayoutProvider.overrideWith(
          (_) async => LayoutPayload.fromJson(minimalLayoutJson()),
        ),
      ],
    );
    addTearDown(container.dispose);
    final router = GoRouter(
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (_, _, shell) => GfShell(navigationShell: shell),
          branches: [
            for (final path in ['/', '/campus', '/notifications', '/messages'])
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: path,
                    builder: (_, _) => Scaffold(body: Text(path)),
                  ),
                ],
              ),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    List<String> checked(GfShellDestination destination) {
      final GfBottomNavigation navigation = tester.widget(
        find.byType(GfBottomNavigation),
      );
      return [
        for (final entry in navigation.items[destination.index].menu!())
          if (entry.selected) entry.label,
      ];
    }

    final sections = container.read(shellSectionsProvider)
      ..report(GfShellDestination.campus, 'academics')
      ..report(GfShellDestination.notifications, 'likes');
    expect(checked(GfShellDestination.campus), isEmpty);
    expect(checked(GfShellDestination.notifications), isEmpty);

    router.go('/campus');
    await tester.pumpAndSettle();
    expect(checked(GfShellDestination.campus), ['学业记录']);
    expect(checked(GfShellDestination.notifications), isEmpty);

    router.go('/notifications');
    await tester.pumpAndSettle();
    expect(checked(GfShellDestination.campus), isEmpty);
    expect(checked(GfShellDestination.notifications), ['赞']);

    router.go('/');
    await tester.pumpAndSettle();
    expect(checked(GfShellDestination.campus), isEmpty);
    expect(checked(GfShellDestination.notifications), isEmpty);
    expect(sections.currentOf(GfShellDestination.campus), 'academics');
  });
}
