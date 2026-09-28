import 'dart:ui' show PointerDeviceKind;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show defaultTargetPlatform;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation/route_visibility.dart';
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

class _SwipeTabsPage extends StatefulWidget {
  const _SwipeTabsPage({this.initialIndex = 1});

  final int initialIndex;

  @override
  State<_SwipeTabsPage> createState() => _SwipeTabsPageState();
}

class _SwipeTabsPageState extends State<_SwipeTabsPage> {
  late int index = widget.initialIndex;
  final scroll = ScrollController();

  @override
  void dispose() {
    scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => RootSurface(
    swipeTabIndex: index,
    swipeTabCount: 3,
    onSwipeTabChanged: (value) => setState(() => index = value),
    title: 'Swipe tabs',
    body: (top, bottom) => ListView(
      controller: scroll,
      padding: EdgeInsets.only(top: top, bottom: bottom),
      children: [
        SizedBox(height: 80, child: Center(child: Text('Tab $index'))),
        for (var i = 0; i < 30; i++)
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
  Size size = const Size(390, 844),
  EdgeInsets padding = EdgeInsets.zero,
  TextDirection textDirection = TextDirection.ltr,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
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
          theme: gfThemeData(
            Brightness.light,
          ).copyWith(platform: defaultTargetPlatform),
          darkTheme: gfThemeData(
            Brightness.dark,
          ).copyWith(platform: defaultTargetPlatform),
          themeMode: ref.watch(themeModeProvider),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => Directionality(
            textDirection: textDirection,
            child: MediaQuery(
              data: MediaQuery.of(context).copyWith(
                textScaler: TextScaler.linear(scale),
                padding: padding,
                disableAnimations: true,
              ),
              child: child!,
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return container;
}

void _openDrawer() => accountDrawerLayerKey.currentState!.open();

void _closeDrawer() => accountDrawerLayerKey.currentState!.close();

bool _isDrawerOpen() => shellDrawerOpen.value;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    shellDrawerOpen.value = false;
  });

  testWidgets(
    'iOS drawer exposes a labelled semantic close action',
    (tester) async {
      final semantics = tester.ensureSemantics();
      try {
        await _mount(tester);
        _openDrawer();
        await tester.pumpAndSettle();
        tester.semantics.tap(find.semantics.byLabel('Dismiss'));
        await tester.pumpAndSettle();
        expect(_isDrawerOpen(), isFalse);
        expect(find.byType(Drawer), findsNothing);
      } finally {
        semantics.dispose();
      }
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  for (final direction in TextDirection.values) {
    testWidgets(
      'drawer keeps tracking a reversal past the drag origin ($direction)',
      (tester) async {
        await _mount(tester, textDirection: direction);
        final sign = direction == TextDirection.ltr ? 1.0 : -1.0;
        final gesture = await tester.startGesture(
          Offset(sign > 0 ? 100 : 290, 420),
        );
        await gesture.moveBy(Offset(100 * sign, 0));
        await tester.pump(const Duration(milliseconds: 100));
        expect(find.byType(Drawer), findsOneWidget);
        await gesture.moveBy(Offset(-130 * sign, 0));
        await tester.pump(const Duration(milliseconds: 100));
        expect(find.byType(Drawer), findsNothing);
        // The same accepted drag can open the panel again after returning to 0.
        await gesture.moveBy(Offset(260 * sign, 0));
        await tester.pump(const Duration(milliseconds: 100));
        await gesture.up();
        await tester.pumpAndSettle();
        expect(_isDrawerOpen(), isTrue);
        _closeDrawer();
        await tester.pumpAndSettle();
      },
      variant: TargetPlatformVariant({
        TargetPlatform.android,
        TargetPlatform.iOS,
      }),
    );
  }

  for (final mode in [ThemeMode.light, ThemeMode.dark]) {
    testWidgets(
      'drawer keeps full-height compact layout and scrolls at 2x ($mode)',
      (tester) async {
        final container = await _mount(
          tester,
          scale: 2,
          size: const Size(320, 568),
          padding: const EdgeInsets.only(top: 47, bottom: 34),
        );
        container.read(themeModeProvider.notifier).setMode(mode);
        _openDrawer();
        await tester.pumpAndSettle();
        final drawer = find.byType(Drawer);
        final rect = tester.getRect(drawer);
        expect(rect.left, 0);
        expect(rect.top, 0);
        expect(rect.width, closeTo(320 * .84, .01));
        expect(rect.bottom, 568);
        final panel = tester.widget<Drawer>(drawer);
        expect(
          (panel.shape! as RoundedRectangleBorder).borderRadius,
          BorderRadius.zero,
        );
        final entry = tester.widget<ListTile>(find.byType(ListTile).first);
        expect(entry.minTileHeight, 56);
        expect(entry.horizontalTitleGap, 16);
        expect((entry.leading! as GfSymbol).size, 24);
        expect((entry.title! as Text).style!.fontSize, 18);
        await tester.scrollUntilVisible(
          find.text('Appearance'),
          160,
          scrollable: find.descendant(
            of: drawer,
            matching: find.byType(Scrollable),
          ),
        );
        await tester.pumpAndSettle();
        expect(
          tester.getRect(find.text('Appearance')).bottom,
          lessThan(rect.bottom),
        );
        expect(tester.takeException(), isNull);
        _closeDrawer();
        await tester.pumpAndSettle();
      },
    );
  }

  testWidgets('right swipe opens from the expanded leading edge only', (
    tester,
  ) async {
    await _mount(tester);
    for (final x in [8.0, 80.0, 95.0, 195.0, 213.0]) {
      await tester.dragFrom(Offset(x, 320), const Offset(200, 2));
      await tester.pumpAndSettle();
      expect(_isDrawerOpen(), isTrue, reason: 'origin x=$x');
      _closeDrawer();
      await tester.pumpAndSettle();
    }
    for (final x in [215.0, 300.0, 389.0]) {
      await tester.dragFrom(Offset(x, 320), const Offset(110, 2));
      await tester.pumpAndSettle();
      expect(_isDrawerOpen(), isFalse, reason: 'origin x=$x');
    }
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('drawer follows a slow edge drag until release', (tester) async {
    await _mount(tester);
    final gesture = await tester.startGesture(const Offset(80, 320));
    for (var i = 0; i < 10; i++) {
      await gesture.moveBy(const Offset(5, 0));
      await tester.pump(const Duration(milliseconds: 100));
    }
    await tester.pump(const Duration(milliseconds: 300));
    final drawerLeft = tester.getRect(find.byType(Drawer)).left;
    expect(drawerLeft, greaterThan(-320));
    expect(drawerLeft, lessThan(0));

    await gesture.moveBy(const Offset(180, 0));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isTrue);
    _closeDrawer();
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('tab swipes cover the page and leave the drawer edge free', (
    tester,
  ) async {
    await _mount(tester, home: const _SwipeTabsPage());
    expect(find.text('Tab 1'), findsOneWidget);

    final state = tester.state<_SwipeTabsPageState>(
      find.byType(_SwipeTabsPage),
    );
    await tester.dragFrom(const Offset(100, 420), const Offset(0, -120));
    await tester.pumpAndSettle();
    expect(state.scroll.offset, greaterThan(0));

    await tester.dragFrom(const Offset(180, 28), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 2'), findsOneWidget);

    await tester.dragFrom(const Offset(320, 420), const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 1'), findsOneWidget);
    expect(_isDrawerOpen(), isFalse);

    await tester.dragFrom(const Offset(8, 420), const Offset(200, 0));
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isFalse);
    expect(find.text('Tab 0'), findsOneWidget);

    await tester.dragFrom(const Offset(8, 420), const Offset(200, 0));
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isTrue);
    expect(find.text('Tab 0'), findsOneWidget);
    _closeDrawer();
    await tester.pumpAndSettle();

    await tester.dragFrom(const Offset(112, 420), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 1'), findsOneWidget);
    expect(_isDrawerOpen(), isFalse);

    await tester.dragFrom(const Offset(112, 420), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 2'), findsOneWidget);
    expect(_isDrawerOpen(), isFalse);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('edge tab drag keeps processing moves after drawer yield', (
    tester,
  ) async {
    await _mount(tester, home: const _SwipeTabsPage(initialIndex: 0));
    final gesture = await tester.startGesture(const Offset(100, 420));
    await gesture.moveBy(const Offset(-36, 0));
    await tester.pump(const Duration(milliseconds: 40));
    await gesture.moveBy(const Offset(100, 0));
    await tester.pump(const Duration(milliseconds: 40));
    await gesture.up();
    await tester.pumpAndSettle();

    expect(find.text('Tab 0'), findsOneWidget);
    expect(_isDrawerOpen(), isFalse);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('RTL tab swipes follow reading order', (tester) async {
    await _mount(
      tester,
      home: const _SwipeTabsPage(initialIndex: 0),
      textDirection: TextDirection.rtl,
    );
    await tester.dragFrom(const Offset(200, 420), const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 1'), findsOneWidget);
    expect(_isDrawerOpen(), isFalse);

    await tester.dragFrom(const Offset(200, 420), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(find.text('Tab 0'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('drawer closes with an opposite-direction panel swipe', (
    tester,
  ) async {
    await _mount(tester);
    _openDrawer();
    await tester.pumpAndSettle();
    final drawer = tester.getRect(find.byType(Drawer));
    await tester.dragFrom(
      drawer.centerRight - const Offset(8, 0),
      const Offset(-180, 0),
    );
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isFalse);
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
    expect(_isDrawerOpen(), isFalse);
    await tester.dragFrom(const Offset(180, 320), const Offset(-100, 0));
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isFalse);
    await tester.dragFrom(const Offset(120, 450), const Offset(35, -180));
    await tester.pumpAndSettle();
    expect(state.scroll.offset, greaterThan(0));
    expect(_isDrawerOpen(), isFalse);
    await tester.dragFrom(
      const Offset(120, 320),
      const Offset(120, 0),
      kind: PointerDeviceKind.mouse,
    );
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isFalse);
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
    expect(_isDrawerOpen(), isFalse);
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
    expect(_isDrawerOpen(), isFalse);
    position.jumpTo(0);
    await tester.pumpAndSettle();
    await tester.dragFrom(origin, const Offset(100, 0));
    await tester.pumpAndSettle();
    expect(_isDrawerOpen(), isFalse);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('drawer theme choices stay open and repaint with the app', (
    tester,
  ) async {
    final container = await _mount(tester, scale: 2);
    _openDrawer();
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
      expect(_isDrawerOpen(), isTrue);
      expect(tester.takeException(), isNull);
    }
    Navigator.of(tester.element(find.byType(BottomSheet))).pop();
    await tester.pumpAndSettle();
    _closeDrawer();
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}
