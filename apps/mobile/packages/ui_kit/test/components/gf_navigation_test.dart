import 'dart:ui' show Tristate;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart'
    show BackdropFilterLayer, RenderDecoratedBox, RenderParagraph;
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('GfAppBar uses compact centered navigation chrome', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          appBar: GfAppBar(
            title: const Text('首页'),
            actions: <Widget>[
              GfIconButton(symbol: 'settings', onPressed: () {}),
            ],
          ),
        ),
      ),
    );

    expect(find.byType(AppBar), findsOneWidget);
    expect(find.text('首页'), findsOneWidget);
    final settings = find.byWidgetPredicate(
      (widget) => widget is GfSymbol && widget.name == 'settings',
    );
    expect(settings, findsOneWidget);

    final Finder buttonBox = find
        .ancestor(of: settings, matching: find.byType(SizedBox))
        .first;
    expect(tester.getSize(buttonBox), const Size.square(44));
  });

  testWidgets('GfAppBar keeps navigation chrome below the status safe area', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: MediaQuery(
          data: const MediaQueryData(padding: EdgeInsets.only(top: 44)),
          child: const Scaffold(appBar: GfAppBar(title: Text('安全区'))),
        ),
      ),
    );

    expect(tester.getTopLeft(find.text('安全区')).dy, greaterThanOrEqualTo(44));
    expect(tester.getSize(find.byType(AppBar)).height, 100);
  });

  testWidgets('GfBottomNavigation separates destinations and compose action', (
    tester,
  ) async {
    int selected = 0;
    int composeTaps = 0;
    await tester.pumpWidget(
      gfApp(
        GfBottomNavigation(
          currentIndex: selected,
          onSelected: (int value) => selected = value,
          onAction: () => composeTaps++,
          actionLabel: '发布',
          items: const <GfBottomNavigationItem>[
            GfBottomNavigationItem(
              label: '首页',
              symbol: 'house',
              selectedSymbol: 'house-filled',
            ),
            GfBottomNavigationItem(
              label: '搜索',
              symbol: 'search',
              selectedSymbol: 'search-filled',
            ),
            GfBottomNavigationItem(
              label: '消息',
              symbol: 'mail',
              selectedSymbol: 'mail-filled',
              badge: true,
            ),
            GfBottomNavigationItem(
              label: '我的',
              symbol: 'user-round',
              selectedSymbol: 'user-round-filled',
            ),
          ],
        ),
      ),
    );

    expect(find.text('首页'), findsOneWidget);
    expect(find.text('搜索'), findsOneWidget);
    expect(find.text('消息'), findsOneWidget);
    expect(find.text('我的'), findsOneWidget);
    expect(find.text('发布'), findsOneWidget);

    await tester.tap(find.text('我的'));
    expect(selected, 3);
    await tester.tap(find.text('发布'));
    expect(composeTaps, 1);
  });

  testWidgets('navigation exposes one selected unread action semantics node', (
    tester,
  ) async {
    final SemanticsHandle semantics = tester.ensureSemantics();
    try {
      await tester.pumpWidget(
        gfApp(
          GfBottomNavigation(
            currentIndex: 2,
            onSelected: (_) {},
            items: const <GfBottomNavigationItem>[
              GfBottomNavigationItem(
                label: '首页',
                symbol: 'house',
                selectedSymbol: 'house-filled',
              ),
              GfBottomNavigationItem(
                label: '校园',
                symbol: 'graduation-cap',
                selectedSymbol: 'graduation-cap-filled',
              ),
              GfBottomNavigationItem(
                label: '消息',
                symbol: 'mail',
                selectedSymbol: 'mail-filled',
                badge: true,
                badgeSemanticLabel: '有未读消息',
              ),
              GfBottomNavigationItem(
                label: '我的',
                symbol: 'user-round',
                selectedSymbol: 'user-round-filled',
              ),
            ],
          ),
        ),
      );

      final Finder unread = find.bySemanticsLabel('消息, 有未读消息');
      expect(unread, findsOneWidget);
      final SemanticsNode node = tester.getSemantics(unread);
      expect(node.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
      expect(
        node.getSemanticsData().flagsCollection.isSelected,
        Tristate.isTrue,
      );
    } finally {
      semantics.dispose();
    }
  });

  testWidgets('symbol destinations use ReIcon outline/filled variants', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        GfBottomNavigation(
          currentIndex: 0,
          onSelected: (_) {},
          showLabels: false,
          items: const <GfBottomNavigationItem>[
            GfBottomNavigationItem(
              label: '首页',
              symbol: 'house',
              selectedSymbol: 'house-filled',
            ),
            GfBottomNavigationItem(
              label: '校园',
              symbol: 'graduation-cap',
              selectedSymbol: 'graduation-cap-filled',
            ),
            GfBottomNavigationItem(
              label: '通知',
              symbol: 'bell',
              selectedSymbol: 'bell-filled',
            ),
            GfBottomNavigationItem(
              label: '消息',
              symbol: 'mail',
              selectedSymbol: 'mail-filled',
            ),
          ],
        ),
      ),
    );

    expect(
      tester
          .widgetList<GfSymbol>(find.byType(GfSymbol))
          .map((symbol) => symbol.name),
      ['house-filled', 'graduation-cap', 'bell', 'mail'],
    );
    expect(
      find.byKey(
        const ValueKey<String>('gf-bottom-navigation-indicator-house'),
      ),
      findsOneWidget,
    );
  });

  testWidgets('selection feedback remains interruptible between destinations', (
    tester,
  ) async {
    int currentIndex = 0;
    await tester.pumpWidget(
      gfApp(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter stateSetter) {
            return GfBottomNavigation(
              currentIndex: currentIndex,
              onSelected: (int value) =>
                  stateSetter(() => currentIndex = value),
              items: const <GfBottomNavigationItem>[
                GfBottomNavigationItem(
                  label: '首页',
                  symbol: 'house',
                  selectedSymbol: 'house-filled',
                ),
                GfBottomNavigationItem(
                  label: '校园',
                  symbol: 'graduation-cap',
                  selectedSymbol: 'graduation-cap-filled',
                ),
                GfBottomNavigationItem(
                  label: '通知',
                  symbol: 'bell',
                  selectedSymbol: 'bell-filled',
                ),
                GfBottomNavigationItem(
                  label: '消息',
                  symbol: 'mail',
                  selectedSymbol: 'mail-filled',
                ),
              ],
            );
          },
        ),
      ),
    );

    Color indicatorColor(String symbol) {
      final Finder indicator = find.byKey(
        ValueKey<String>('gf-bottom-navigation-indicator-$symbol'),
      );
      final Finder decorationFinder = find.descendant(
        of: indicator,
        matching: find.byType(DecoratedBox),
      );
      final RenderDecoratedBox renderBox = tester.renderObject(
        decorationFinder,
      );
      return (renderBox.decoration as BoxDecoration).color!;
    }

    final Color target = GfTheme.colorsOf(
      tester.element(find.text('校园')),
    ).primary.withValues(alpha: 0.12);
    const Color unselected = Colors.transparent;

    await tester.tap(find.text('校园'));
    expect(currentIndex, 1);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(indicatorColor('graduation-cap'), isNot(anyOf(unselected, target)));

    await tester.tap(find.text('首页'));
    expect(currentIndex, 0);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(indicatorColor('house'), isNot(anyOf(unselected, target)));

    await tester.pumpAndSettle();
    expect(indicatorColor('house'), target);
    expect(indicatorColor('graduation-cap'), unselected);
  });

  testWidgets('selection settles immediately when reduced motion is enabled', (
    tester,
  ) async {
    int currentIndex = 0;
    bool disableAnimations = true;
    late StateSetter update;
    const items = <GfBottomNavigationItem>[
      GfBottomNavigationItem(
        label: '首页',
        symbol: 'house',
        selectedSymbol: 'house-filled',
      ),
      GfBottomNavigationItem(
        label: '校园',
        symbol: 'graduation-cap',
        selectedSymbol: 'graduation-cap-filled',
      ),
      GfBottomNavigationItem(
        label: '通知',
        symbol: 'bell',
        selectedSymbol: 'bell-filled',
      ),
      GfBottomNavigationItem(
        label: '消息',
        symbol: 'mail',
        selectedSymbol: 'mail-filled',
      ),
    ];
    await tester.pumpWidget(
      gfApp(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter stateSetter) {
            update = stateSetter;
            return MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(disableAnimations: disableAnimations),
              child: GfBottomNavigation(
                currentIndex: currentIndex,
                onSelected: (int value) =>
                    stateSetter(() => currentIndex = value),
                items: items,
              ),
            );
          },
        ),
      ),
    );

    Color indicatorColor(String symbol) {
      final Finder indicator = find.byKey(
        ValueKey<String>('gf-bottom-navigation-indicator-$symbol'),
      );
      final Finder decorationFinder = find.descendant(
        of: indicator,
        matching: find.byType(DecoratedBox),
      );
      final RenderDecoratedBox renderBox = tester.renderObject(
        decorationFinder,
      );
      return (renderBox.decoration as BoxDecoration).color!;
    }

    final Color target = GfTheme.colorsOf(
      tester.element(find.text('首页')),
    ).primary.withValues(alpha: 0.12);
    const Color unselected = Colors.transparent;

    await tester.tap(find.text('校园'));
    await tester.pump();
    expect(indicatorColor('graduation-cap'), target);

    update(() => disableAnimations = false);
    await tester.pump();
    await tester.tap(find.text('首页'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(indicatorColor('house'), isNot(anyOf(unselected, target)));

    update(() => disableAnimations = true);
    await tester.pump(const Duration(milliseconds: 1));
    expect(indicatorColor('house'), target);
    expect(indicatorColor('graduation-cap'), unselected);
  });

  test('navigation metrics keep content and action offsets in one place', () {
    final metrics = GfBottomNavigation.metrics(
      safeAreaBottom: 34,
      barHeight: 72,
    );
    expect(metrics.barHeight, 72);
    expect(metrics.contentBottomInset, 114);
    expect(metrics.actionBottomInset, 88);

    final large = GfBottomNavigation.metrics(safeAreaBottom: 34, barHeight: 88);
    expect(large.barHeight, 88);
    expect(large.contentBottomInset, 130);
  });

  testWidgets('large German navigation labels stay legible at 320px', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 640);
    addTearDown(tester.view.reset);
    const labels = [
      'Startseite',
      'Campus',
      'Benachrichtigungen',
      'Nachrichten',
    ];
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(2)),
          child: Scaffold(
            bottomNavigationBar: GfBottomNavigation(
              currentIndex: 0,
              onSelected: (_) {},
              items: [
                for (final label in labels)
                  GfBottomNavigationItem(
                    label: label,
                    symbol: 'house',
                    selectedSymbol: 'house-filled',
                  ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    for (final label in labels) {
      final text = find.text(label);
      expect(text, findsOneWidget);
      expect(
        tester.renderObject<RenderParagraph>(text).didExceedMaxLines,
        isFalse,
        reason: label,
      );
      final target = find.ancestor(of: text, matching: find.byType(InkWell));
      expect(tester.getSize(target.first).width, greaterThanOrEqualTo(44));
      expect(tester.getSize(target.first).height, greaterThanOrEqualTo(44));
    }
  });

  testWidgets('high contrast navigation falls back to an opaque surface', (
    tester,
  ) async {
    await tester.pumpWidget(
      MediaQuery(
        data: const MediaQueryData(highContrast: true),
        child: MaterialApp(
          theme: gfThemeData(
            Brightness.light,
          ).copyWith(platform: TargetPlatform.iOS),
          home: Scaffold(
            bottomNavigationBar: GfBottomNavigation(
              currentIndex: 0,
              onSelected: (_) {},
              items: const <GfBottomNavigationItem>[
                GfBottomNavigationItem(
                  label: '首页',
                  symbol: 'house',
                  selectedSymbol: 'house-filled',
                ),
                GfBottomNavigationItem(
                  label: '校园',
                  symbol: 'graduation-cap',
                  selectedSymbol: 'graduation-cap-filled',
                ),
                GfBottomNavigationItem(
                  label: '通知',
                  symbol: 'bell',
                  selectedSymbol: 'bell-filled',
                ),
                GfBottomNavigationItem(
                  label: '消息',
                  symbol: 'mail',
                  selectedSymbol: 'mail-filled',
                ),
              ],
            ),
          ),
        ),
      ),
    );

    expect(tester.layers.whereType<BackdropFilterLayer>(), isEmpty);
    expect(tester.getSize(find.text('首页')), isNotNull);
  });

  testWidgets('normal iOS navigation uses the glass material path', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: Scaffold(
          bottomNavigationBar: GfBottomNavigation(
            currentIndex: 0,
            onSelected: (_) {},
            items: const <GfBottomNavigationItem>[
              GfBottomNavigationItem(
                label: '首页',
                symbol: 'house',
                selectedSymbol: 'house-filled',
              ),
              GfBottomNavigationItem(
                label: '校园',
                symbol: 'graduation-cap',
                selectedSymbol: 'graduation-cap-filled',
              ),
              GfBottomNavigationItem(
                label: '通知',
                symbol: 'bell',
                selectedSymbol: 'bell-filled',
              ),
              GfBottomNavigationItem(
                label: '消息',
                symbol: 'mail',
                selectedSymbol: 'mail-filled',
              ),
            ],
          ),
        ),
      ),
    );

    expect(tester.layers.whereType<BackdropFilterLayer>(), isNotEmpty);
  });

  testWidgets('reduced motion navigation keeps the solid material path', (
    tester,
  ) async {
    await tester.pumpWidget(
      MediaQuery(
        data: const MediaQueryData(disableAnimations: true),
        child: MaterialApp(
          theme: gfThemeData(
            Brightness.light,
          ).copyWith(platform: TargetPlatform.iOS),
          home: Scaffold(
            bottomNavigationBar: GfBottomNavigation(
              currentIndex: 0,
              onSelected: (_) {},
              items: const <GfBottomNavigationItem>[
                GfBottomNavigationItem(
                  label: '首页',
                  symbol: 'house',
                  selectedSymbol: 'house-filled',
                ),
                GfBottomNavigationItem(
                  label: '校园',
                  symbol: 'graduation-cap',
                  selectedSymbol: 'graduation-cap-filled',
                ),
                GfBottomNavigationItem(
                  label: '通知',
                  symbol: 'bell',
                  selectedSymbol: 'bell-filled',
                ),
                GfBottomNavigationItem(
                  label: '消息',
                  symbol: 'mail',
                  selectedSymbol: 'mail-filled',
                ),
              ],
            ),
          ),
        ),
      ),
    );

    expect(tester.layers.whereType<BackdropFilterLayer>(), isEmpty);
  });

  testWidgets(
    'GfBottomNavigation rejects invalid destination counts at runtime',
    (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfBottomNavigation(
            currentIndex: 0,
            onSelected: (_) {},
            items: const <GfBottomNavigationItem>[
              GfBottomNavigationItem(
                label: '首页',
                symbol: 'house',
                selectedSymbol: 'house-filled',
              ),
              GfBottomNavigationItem(
                label: '搜索',
                symbol: 'search',
                selectedSymbol: 'search-filled',
              ),
              GfBottomNavigationItem(
                label: '消息',
                symbol: 'mail',
                selectedSymbol: 'mail-filled',
              ),
            ],
          ),
        ),
      );

      final Object? error = tester.takeException();
      expect(error, isA<FlutterError>());
      expect('$error', contains('requires exactly four destinations'));
    },
  );

  testWidgets('GfScrollToTop appears after threshold and returns to start', (
    tester,
  ) async {
    final GfScrollToTopController controller = GfScrollToTopController();
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: GfScrollToTop(
            semanticLabel: '返回顶部',
            controller: controller,
            threshold: 100,
            builder: (BuildContext context, ScrollController scrollController) {
              return ListView.builder(
                controller: scrollController,
                itemCount: 30,
                itemExtent: 60,
                itemBuilder: (BuildContext context, int index) =>
                    Text('第 $index 项'),
              );
            },
          ),
        ),
      ),
    );

    expect(controller.isAttached, isTrue);
    expect(find.byTooltip('返回顶部'), findsNothing);
    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();
    expect(find.byTooltip('返回顶部'), findsOneWidget);

    final Future<void> scroll = controller.scrollToTop();
    await tester.pumpAndSettle();
    await scroll;
    expect(find.text('第 0 项'), findsOneWidget);
    expect(find.byTooltip('返回顶部'), findsNothing);
  });
}
