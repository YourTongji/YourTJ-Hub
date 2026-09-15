import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/brand_mark.dart';
import 'package:forum_app/src/widgets/root_surface.dart';

import 'fixtures/page_fixtures.dart';

UserCardPayload accountCard() => UserCardPayload.fromJson(
  (userProfilePayloadJson()['props'] as Map<String, dynamic>)['user']
      as Map<String, dynamic>,
).copyWith(nickname: '校园漫游者', followingCount: 108, followerCount: 13);

Future<({GoRouter router, ProviderContainer container})> pumpNavigation(
  WidgetTester tester, {
  Locale locale = const Locale('zh'),
  double textScale = 1,
  Future<UserCardPayload> Function()? loadCard,
  bool signedIn = true,
  VoidCallback? onAction,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = const Size(390, 844);
  addTearDown(tester.view.reset);
  final layout = LayoutPayload.fromJson(minimalLayoutJson());
  final container = ProviderContainer(
    overrides: [
      accountLayoutProvider.overrideWith(
        (_) async => layout.copyWith(
          viewer: layout.viewer.copyWith(
            id: 1,
            username: 'yzxoi',
            isAuthenticated: signedIn,
            canAccessAdmin: false,
            isModerator: false,
          ),
        ),
      ),
      accountCardProvider(
        1,
      ).overrideWith((_) => loadCard?.call() ?? Future.value(accountCard())),
    ],
  );
  addTearDown(container.dispose);
  final router = GoRouter(
    routes: [
      GoRoute(
        path: '/',
        builder: (_, _) => Scaffold(
          drawer: const AccountDrawer(),
          body: RootSurface(
            title: 'YourTJ',
            titleWidget: const YourTjMark(),
            onAction: onAction,
            body: (top, bottom) => ListView(
              padding: EdgeInsets.only(top: top, bottom: bottom),
              children: const [
                SizedBox(height: 300, child: Center(child: Text('校园动态'))),
              ],
            ),
          ),
        ),
      ),
      for (final path in [
        '/publish',
        '/profile',
        '/drafts',
        '/my-content',
        '/recycle-bin',
        '/settings',
      ])
        GoRoute(
          path: path,
          builder: (_, state) =>
              Scaffold(body: Text('destination:${state.uri}')),
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
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(textScale)),
          child: child!,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (router: router, container: container);
}

void main() {
  for (final value in [2, 3, 1]) {
    testWidgets('compose first level routes type $value only after choosing', (
      tester,
    ) async {
      final app = await pumpNavigation(tester);
      await tester.tap(find.byType(FloatingActionButton));
      await tester.pumpAndSettle();
      expect(app.router.state.uri.path, '/');
      for (final type in [2, 3, 1]) {
        expect(find.byKey(ValueKey('compose-$type')), findsOneWidget);
      }
      await tester.tap(find.byKey(ValueKey('compose-$value')));
      await tester.pumpAndSettle();
      expect(app.router.state.uri.toString(), '/publish?type=$value');
      expect(find.byKey(const ValueKey('compose-2')), findsNothing);
    });
  }

  testWidgets('compose closes with backdrop, close button and system back', (
    tester,
  ) async {
    final app = await pumpNavigation(tester);
    for (final dismissal in ['backdrop', 'close', 'back']) {
      await tester.tap(find.byType(FloatingActionButton));
      await tester.pumpAndSettle();
      switch (dismissal) {
        case 'backdrop':
          await tester.tapAt(const Offset(20, 200));
        case 'close':
          await tester.tap(find.byTooltip('关闭'));
        case 'back':
          await tester.binding.handlePopRoute();
      }
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('compose-2')), findsNothing);
      expect(app.router.state.uri.path, '/');
    }
  });

  testWidgets('new conversation keeps its direct action', (tester) async {
    var calls = 0;
    await pumpNavigation(tester, onAction: () => calls++);
    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    expect(calls, 1);
    expect(find.byKey(const ValueKey('compose-2')), findsNothing);
  });

  testWidgets('account shows real connections and flat content destinations', (
    tester,
  ) async {
    final app = await pumpNavigation(tester);
    await tester.tap(find.byTooltip('我的'));
    await tester.pumpAndSettle();
    expect(find.text('校园漫游者'), findsOneWidget);
    expect(find.text('108 正在关注'), findsOneWidget);
    expect(find.text('13 关注者'), findsOneWidget);
    expect(find.byType(ExpansionTile), findsNothing);
    for (final title in ['草稿箱', '我的内容', '回收站']) {
      expect(find.text(title), findsOneWidget);
    }
    await tester.tap(find.text('108 正在关注'));
    await tester.pumpAndSettle();
    expect(app.router.state.uri.toString(), '/profile?stream=following');
  });

  testWidgets('connection failures offer retry without inventing zero counts', (
    tester,
  ) async {
    var attempts = 0;
    final app = await pumpNavigation(
      tester,
      loadCard: () async {
        if (++attempts == 1) throw StateError('offline');
        return accountCard();
      },
    );
    await tester.tap(find.byTooltip('我的'));
    await tester.pumpAndSettle();
    expect(find.text('— 正在关注'), findsOneWidget);
    await tester.tap(find.byTooltip('重试'));
    await tester.pumpAndSettle();
    expect(find.text('13 关注者'), findsOneWidget);
    await tester.tap(find.text('13 关注者'));
    await tester.pumpAndSettle();
    expect(app.router.state.uri.toString(), '/profile?stream=followers');
  });

  testWidgets('signing out hides previous account and connection data', (
    tester,
  ) async {
    final app = await pumpNavigation(tester);
    await tester.tap(find.byTooltip('我的'));
    await tester.pumpAndSettle();
    final nextLayout = Completer<LayoutPayload>();
    app.container.updateOverrides([
      accountLayoutProvider.overrideWith((_) => nextLayout.future),
      accountCardProvider(1).overrideWith((_) async => accountCard()),
    ]);
    app.container.invalidate(accountLayoutProvider);
    await tester.pumpAndSettle();
    expect(app.container.read(accountLayoutProvider).isLoading, isTrue);
    expect(find.text('108 正在关注'), findsNothing);
    expect(find.text('校园漫游者'), findsNothing);
    nextLayout.complete(LayoutPayload.fromJson(minimalLayoutJson()));
    await tester.pumpAndSettle();
    expect(find.text('108 正在关注'), findsNothing);
    expect(find.text('校园漫游者'), findsNothing);
  });

  testWidgets('large text keeps menu and drawer usable at narrow widths', (
    tester,
  ) async {
    await pumpNavigation(tester, locale: const Locale('de'), textScale: 2);
    tester.view.physicalSize = const Size(320, 640);
    await tester.pumpAndSettle();
    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    await tester.tap(find.byType(AccountAvatar));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.byType(ExpansionTile), findsNothing);
  });
}
