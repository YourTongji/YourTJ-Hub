import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation_style.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show MemTokenStorage, NoopCache;

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<GfNavigationStyle> restored(Map<String, Object> saved) async {
    SharedPreferences.setMockInitialValues(saved);
    final container = ProviderContainer();
    addTearDown(container.dispose);
    container.read(navigationStyleProvider);
    await Future<void>.delayed(Duration.zero);
    return container.read(navigationStyleProvider);
  }

  test('restores the saved style and ignores unknown values', () async {
    expect(await restored({}), GfNavigationStyle.dock);
    expect(
      await restored({NavigationStyleNotifier.prefsKey: 'bar'}),
      GfNavigationStyle.bar,
    );
    expect(
      await restored({NavigationStyleNotifier.prefsKey: 'ribbon'}),
      GfNavigationStyle.dock,
    );
  });

  test('a choice persists and survives a new session', () async {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    container
        .read(navigationStyleProvider.notifier)
        .setStyle(GfNavigationStyle.bar);
    await Future<void>.delayed(Duration.zero);
    expect(
      (await SharedPreferences.getInstance()).getString(
        NavigationStyleNotifier.prefsKey,
      ),
      'bar',
    );
    expect(
      await restored({NavigationStyleNotifier.prefsKey: 'bar'}),
      GfNavigationStyle.bar,
    );
  });

  testWidgets('the shell measures the classic bar and switches live', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    tester.view.padding = const FakeViewPadding(top: 44, bottom: 34);
    tester.view.viewPadding = const FakeViewPadding(top: 44, bottom: 34);
    addTearDown(tester.view.reset);
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(MemTokenStorage()),
        currentUserProvider.overrideWith((_) async => null),
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
                    builder: (_, _) => RootSurface(
                      title: path,
                      body: (top, bottom) => ListView(
                        key: ValueKey('body-$path'),
                        padding: EdgeInsets.only(top: top, bottom: bottom),
                        children: const [SizedBox(height: 2000)],
                      ),
                    ),
                  ),
                ],
              ),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);
    container
        .read(navigationStyleProvider.notifier)
        .setStyle(GfNavigationStyle.bar);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => AppNavigationStyleScope(child: child!),
        ),
      ),
    );
    await tester.pumpAndSettle();

    double contentBottom() => tester
        .widget<ListView>(find.byKey(const ValueKey('body-/')))
        .padding!
        .resolve(TextDirection.ltr)
        .bottom;

    expect(
      tester.getRect(find.byType(GfNavigationBarSurface)),
      const Rect.fromLTRB(0, 844 - 34 - 56, 390, 844),
    );
    expect(contentBottom(), 56 + 34 + 8);

    container
        .read(navigationStyleProvider.notifier)
        .setStyle(GfNavigationStyle.dock);
    await tester.pumpAndSettle();
    expect(find.byType(GfNavigationBarSurface), findsNothing);
    expect(
      contentBottom(),
      GfBottomNavigation.metrics(
        safeAreaBottom: 34,
        barHeight: 56,
      ).contentBottomInset,
    );
    expect(tester.takeException(), isNull);
  });
}
