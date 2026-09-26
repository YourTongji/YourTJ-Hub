import 'dart:ui' show PointerDeviceKind;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/root_surface.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'pages_behavior_test.dart' show MemTokenStorage, NoopCache;

class _GesturePage extends StatefulWidget {
  const _GesturePage();

  @override
  State<_GesturePage> createState() => _GesturePageState();
}

class _GesturePageState extends State<_GesturePage> {
  final scroll = ScrollController();
  double value = .5;

  @override
  void dispose() {
    scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => RootSurface(
    title: 'Home',
    body: (top, bottom) => ListView(
      key: const Key('drawer-vertical-list'),
      controller: scroll,
      padding: EdgeInsets.only(top: top, bottom: bottom),
      children: [
        const SizedBox(height: 80, child: Text('Ordinary reading content')),
        Slider(
          key: const Key('drawer-slider'),
          value: value,
          onChanged: (next) => setState(() => value = next),
        ),
        for (var i = 0; i < 40; i++)
          SizedBox(height: 64, child: Text('Reading row $i')),
      ],
    ),
  );
}

class _HomePages extends PageRepository {
  _HomePages()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );

  @override
  Future<PagePayload> home({String sort = '', Object? cancelToken}) async {
    final data = homePayloadJson();
    (data['layout']['sidebar'] as Map)['categories'] = [
      for (var i = 1; i <= 12; i++)
        {
          'id': i,
          'label': 'Category $i',
          'url': '/c/category/$i',
          'color': '#888888',
        },
    ];
    return parsePayload(data);
  }
}

Future<ProviderContainer> _mount(
  WidgetTester tester, {
  Widget home = const _GesturePage(),
  PageRepository? pages,
  double scale = 1,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = const Size(390, 844);
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
      if (pages != null) pageRepositoryProvider.overrideWithValue(pages),
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
              routes: [GoRoute(path: path, builder: (_, _) => home)],
            ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: Consumer(
        builder: (_, ref, _) => MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          darkTheme: gfThemeData(Brightness.dark),
          themeMode: ref.watch(themeModeProvider),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: TextScaler.linear(scale),
              disableAnimations: true,
            ),
            child: child!,
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return container;
}

ScaffoldState _shell(WidgetTester tester) => tester.state<ScaffoldState>(
  find.byWidgetPredicate(
    (widget) => widget is Scaffold && widget.drawer is AccountDrawer,
  ),
);

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  testWidgets('right swipe opens from the leading 55 percent of content', (
    tester,
  ) async {
    await _mount(tester);
    for (final x in [8.0, 195.0, 213.0]) {
      await tester.dragFrom(Offset(x, 320), const Offset(110, 2));
      await tester.pumpAndSettle();
      expect(_shell(tester).isDrawerOpen, isTrue, reason: 'origin x=$x');
      _shell(tester).closeDrawer();
      await tester.pumpAndSettle();
    }
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('drawer leaves vertical, leftward and trailing swipes alone', (
    tester,
  ) async {
    await _mount(tester);
    final state = tester.state<_GesturePageState>(find.byType(_GesturePage));
    await tester.dragFrom(const Offset(230, 320), const Offset(120, 0));
    await tester.pumpAndSettle();
    expect(_shell(tester).isDrawerOpen, isFalse);
    await tester.dragFrom(const Offset(180, 320), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(_shell(tester).isDrawerOpen, isFalse);
    await tester.dragFrom(const Offset(120, 450), const Offset(35, -180));
    await tester.pumpAndSettle();
    expect(state.scroll.offset, greaterThan(0));
    expect(_shell(tester).isDrawerOpen, isFalse);
    await tester.dragFrom(
      const Offset(120, 320),
      const Offset(120, 0),
      kind: PointerDeviceKind.mouse,
    );
    await tester.pumpAndSettle();
    expect(_shell(tester).isDrawerOpen, isFalse);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('a horizontal slider owns its drag inside the opening area', (
    tester,
  ) async {
    await _mount(tester);
    final state = tester.state<_GesturePageState>(find.byType(_GesturePage));
    final slider = tester.getRect(find.byKey(const Key('drawer-slider')));
    await tester.dragFrom(slider.center, const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(state.value, greaterThan(.5));
    expect(_shell(tester).isDrawerOpen, isFalse);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('home category rail keeps horizontal scrolling at either end', (
    tester,
  ) async {
    await _mount(tester, home: const HomePage(), pages: _HomePages());
    final rail = find.byKey(const ValueKey('home-category-rail'));
    final position = tester
        .state<ScrollableState>(
          find.descendant(of: rail, matching: find.byType(Scrollable)),
        )
        .position;
    position.jumpTo(200);
    await tester.pumpAndSettle();
    final origin = Offset(120, tester.getCenter(rail).dy);
    await tester.dragFrom(origin, const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(position.pixels, lessThan(200));
    expect(_shell(tester).isDrawerOpen, isFalse);
    position.jumpTo(0);
    await tester.pumpAndSettle();
    await tester.dragFrom(origin, const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(_shell(tester).isDrawerOpen, isFalse);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('drawer theme choices stay open and repaint with the app', (
    tester,
  ) async {
    final container = await _mount(tester, scale: 2);
    _shell(tester).openDrawer();
    await tester.pumpAndSettle();
    final l10n = AppLocalizations.of(tester.element(find.byType(GfShell)));
    final appearance = find.text(l10n.settingsAppearance);
    await tester.ensureVisible(appearance);
    await tester.tap(appearance);
    await tester.pumpAndSettle();
    for (final mode in [ThemeMode.dark, ThemeMode.light, ThemeMode.system]) {
      final label = switch (mode) {
        ThemeMode.dark => l10n.settingsThemeDark,
        ThemeMode.light => l10n.settingsThemeLight,
        ThemeMode.system => l10n.settingsLanguageSystem,
      };
      final choice = find.descendant(
        of: find.byType(BottomSheet),
        matching: find.text(label),
      );
      await tester.ensureVisible(choice);
      await tester.tap(choice);
      await tester.pumpAndSettle();
      expect(container.read(themeModeProvider), mode);
      expect(find.byType(BottomSheet), findsOneWidget);
      final expected = mode == ThemeMode.dark
          ? Brightness.dark
          : Brightness.light;
      expect(Theme.of(tester.element(choice)).brightness, expected);
      final material = tester.widget<Material>(
        find.ancestor(of: choice, matching: find.byType(Material)).first,
      );
      expect(material.color, GfColors.forBrightness(expected).base100);
      expect(_shell(tester).isDrawerOpen, isTrue);
      expect(tester.takeException(), isNull);
    }
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}
