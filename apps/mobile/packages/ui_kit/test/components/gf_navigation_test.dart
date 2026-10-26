import 'dart:ui' show Tristate;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart' show RenderParagraph;
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
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
          currentIndex: 0,
          onSelected: (int value) => selected = value,
          onAction: () => composeTaps++,
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
            GfBottomNavigationItem(
              label: '我的',
              symbol: 'user-round',
              selectedSymbol: 'user-round-filled',
            ),
          ],
        ),
      ),
    );

    for (final String label in <String>['首页', '搜索', '消息', '我的']) {
      expect(
        find.descendant(of: _restingLayer, matching: find.text(label)),
        findsOneWidget,
      );
    }
    expect(find.text('发布'), findsOneWidget);
    expect(
      tester.getRect(find.text('发布')).left,
      greaterThan(tester.getRect(_restingLayer).right),
      reason: 'the compose action floats beside the dock',
    );

    await tester.tap(_target(3));
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

  testWidgets('the lens shows the filled glyph over the outline layer', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(_dock(index: 0)));

    List<String> symbols(Finder layer) => tester
        .widgetList<GfSymbol>(
          find.descendant(of: layer, matching: find.byType(GfSymbol)),
        )
        .map((symbol) => symbol.name)
        .toList();
    expect(symbols(_restingLayer), ['house', 'graduation-cap', 'bell', 'mail']);
    expect(symbols(_selectedLayer), [
      'house-filled',
      'graduation-cap-filled',
      'bell-filled',
      'mail-filled',
    ]);
    expect(
      tester.getCenter(_lens).dx,
      moreOrLessEquals(tester.getCenter(_target(0)).dx, epsilon: 0.5),
    );
  });

  testWidgets('icon-only destinations stay compact and name themselves', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(_dock(index: 0)));

    expect(find.text('校园'), findsNothing);
    expect(find.bySemanticsLabel('校园'), findsOneWidget);
    expect(tester.getSize(_restingLayer).height, 56);
    expect(tester.getSize(_target(1)).height, greaterThanOrEqualTo(44));
    await tester.longPress(_target(1));
    await tester.pumpAndSettle();
    expect(find.text('校园'), findsOneWidget);
  });

  testWidgets('the dock floats inside the margins and lets content through', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 800);
    addTearDown(tester.view.reset);
    int contentTaps = 0;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: MediaQuery(
          data: const MediaQueryData(
            size: Size(390, 800),
            padding: EdgeInsets.only(bottom: 34),
          ),
          child: Stack(
            children: [
              Positioned.fill(
                child: GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onTap: () => contentTaps++,
                ),
              ),
              Positioned(left: 0, right: 0, bottom: 0, child: _dock(index: 0)),
            ],
          ),
        ),
      ),
    );

    // The layers sit inside the dock's 5px horizontal inset.
    final Rect layer = tester.getRect(_restingLayer);
    final Rect dock = Rect.fromLTRB(
      layer.left - 5,
      layer.top,
      layer.right + 5,
      layer.bottom,
    );
    expect(dock.bottom, moreOrLessEquals(800 - 22, epsilon: 0.5));
    expect(dock.left, greaterThanOrEqualTo(20));
    expect(dock.center.dx, moreOrLessEquals(195, epsilon: 0.5));
    await tester.tapAt(Offset(dock.left - 8, dock.center.dy));
    expect(contentTaps, 1, reason: 'the margins beside the dock stay content');

    // A press swells the dock's paint, never the area it takes touches in.
    final TestGesture hold = await tester.startGesture(
      tester.getCenter(_target(1)),
    );
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tapAt(Offset(dock.left - 1.5, dock.center.dy));
    expect(contentTaps, 2);
    await hold.up();
    await tester.pumpAndSettle();
  });

  testWidgets('selection feedback remains interruptible between destinations', (
    tester,
  ) async {
    int currentIndex = 0;
    await tester.pumpWidget(
      gfApp(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter setState) => _dock(
            index: currentIndex,
            onSelected: (int value) => setState(() => currentIndex = value),
          ),
        ),
      ),
    );
    double lens() => tester.getCenter(_lens).dx;
    double slot(int index) => tester.getCenter(_target(index)).dx;

    await tester.tap(_target(2));
    expect(currentIndex, 2);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    final double midway = lens();
    expect(midway, greaterThan(slot(0) + 4));
    expect(midway, lessThan(slot(2) - 4));
    expect(
      tester.getSize(_lens).width,
      greaterThan(tester.getSize(_target(0)).width),
      reason: 'the lens stretches while it travels',
    );

    await tester.tap(_target(0));
    expect(currentIndex, 0);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 16));
    expect(
      (lens() - midway).abs(),
      lessThan(slot(1) - slot(0)),
      reason: 'a new target redirects the lens from where it is',
    );

    await tester.pumpAndSettle();
    expect(lens(), moreOrLessEquals(slot(0), epsilon: 0.5));
    expect(tester.getSize(_lens), _restingLens(tester, 0));
  });

  testWidgets('selection settles immediately when reduced motion is enabled', (
    tester,
  ) async {
    int currentIndex = 0;
    bool disableAnimations = true;
    late StateSetter update;
    await tester.pumpWidget(
      gfApp(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter setState) {
            update = setState;
            return MediaQuery(
              data: MediaQuery.of(
                context,
              ).copyWith(disableAnimations: disableAnimations),
              child: _dock(
                index: currentIndex,
                onSelected: (int value) => setState(() => currentIndex = value),
              ),
            );
          },
        ),
      ),
    );
    double lens() => tester.getCenter(_lens).dx;
    double slot(int index) => tester.getCenter(_target(index)).dx;

    await tester.tap(_target(1));
    await tester.pump();
    expect(lens(), moreOrLessEquals(slot(1), epsilon: 0.5));

    update(() => disableAnimations = false);
    await tester.pump();
    await tester.tap(_target(3));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 80));
    expect(lens(), allOf(greaterThan(slot(1) + 4), lessThan(slot(3) - 4)));

    update(() => disableAnimations = true);
    await tester.pump(const Duration(milliseconds: 1));
    expect(lens(), moreOrLessEquals(slot(3), epsilon: 0.5));
    expect(tester.getSize(_lens), _restingLens(tester, 3));
  });

  testWidgets('dragging scrubs the lens and selects where it is let go', (
    tester,
  ) async {
    final List<int> selections = <int>[];
    int currentIndex = 0;
    int ticks = 0;
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (MethodCall call) async {
        if (call.method == 'HapticFeedback.vibrate' &&
            call.arguments == 'HapticFeedbackType.selectionClick') {
          ticks++;
        }
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    await tester.pumpWidget(
      gfApp(
        StatefulBuilder(
          builder: (BuildContext context, StateSetter setState) => _dock(
            index: currentIndex,
            onSelected: (int value) => setState(() {
              selections.add(value);
              currentIndex = value;
            }),
          ),
        ),
      ),
    );
    double slot(int index) => tester.getCenter(_target(index)).dx;

    final TestGesture gesture = await tester.startGesture(
      tester.getCenter(_target(0)),
    );
    for (int step = 1; step <= 10; step++) {
      await gesture.moveTo(
        Offset(
          slot(0) + (slot(2) - slot(0)) * step / 10,
          tester.getCenter(_target(0)).dy,
        ),
      );
      await tester.pump(const Duration(milliseconds: 16));
    }
    expect(selections, isEmpty, reason: 'nothing is selected mid-drag');
    expect(ticks, 2, reason: 'one tick per destination crossed');
    expect(
      tester.getSize(_lens).height,
      greaterThan(_restingLens(tester, 0).height),
      reason: 'a dragged lens lifts past its resting size',
    );
    await tester.pump(const Duration(milliseconds: 300));
    await gesture.up();
    await tester.pumpAndSettle();
    expect(selections, [2]);
    expect(tester.getCenter(_lens).dx, moreOrLessEquals(slot(2), epsilon: 0.5));
    expect(tester.getSize(_lens), _restingLens(tester, 2));

    // Letting go where it started selects nothing, so the parent does not
    // treat it as a repeated tap.
    await tester.timedDrag(
      _target(2),
      const Offset(24, 0),
      const Duration(milliseconds: 400),
    );
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpAndSettle();
    expect(selections, [2]);
  });

  testWidgets('a lens the parent does not accept returns to the selection', (
    tester,
  ) async {
    await tester.pumpWidget(gfApp(_dock(index: 0, onSelected: (_) {})));

    await tester.tap(_target(3));
    await tester.pumpAndSettle();
    expect(
      tester.getCenter(_lens).dx,
      moreOrLessEquals(tester.getCenter(_target(0)).dx, epsilon: 0.5),
    );
  });

  test('navigation metrics keep content and action offsets in one place', () {
    final metrics = GfBottomNavigation.metrics(
      safeAreaBottom: 34,
      barHeight: 56,
    );
    expect(metrics.barHeight, 56);
    expect(metrics.contentBottomInset, 102);
    expect(metrics.actionBottomInset, 72);

    final overIndicator = GfBottomNavigation.metrics(
      safeAreaBottom: 34,
      barHeight: 56,
      homeIndicator: true,
    );
    expect(GfBottomNavigation.bottomMarginFor(34, homeIndicator: true), 22);
    expect(overIndicator.contentBottomInset, 90);
    expect(overIndicator.actionBottomInset, 60);

    final edgeless = GfBottomNavigation.metrics(
      safeAreaBottom: 0,
      barHeight: 88,
    );
    expect(edgeless.contentBottomInset, 112);
    expect(edgeless.actionBottomInset, 116);
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
    for (int i = 0; i < labels.length; i++) {
      final text = find.descendant(
        of: _restingLayer,
        matching: find.text(labels[i]),
      );
      expect(text, findsOneWidget);
      expect(
        tester.renderObject<RenderParagraph>(text).didExceedMaxLines,
        isFalse,
        reason: labels[i],
      );
      expect(
        tester.getRect(text).bottom,
        lessThanOrEqualTo(tester.getRect(_restingLayer).bottom),
        reason: '${labels[i]} fits inside the dock',
      );
      expect(tester.getSize(_target(i)).width, greaterThanOrEqualTo(44));
      expect(tester.getSize(_target(i)).height, greaterThanOrEqualTo(44));
    }
  });

  for (final (String name, MediaQueryData data, bool glass)
      in <(String, MediaQueryData, bool)>[
        ('a normal Android dock', const MediaQueryData(), true),
        ('high contrast', const MediaQueryData(highContrast: true), false),
        (
          'reduced motion',
          const MediaQueryData(disableAnimations: true),
          false,
        ),
        (
          'accessible navigation',
          const MediaQueryData(accessibleNavigation: true),
          false,
        ),
      ]) {
    testWidgets('$name ${glass ? 'uses glass' : 'keeps the opaque dock'}', (
      tester,
    ) async {
      await tester.pumpWidget(
        MediaQuery(
          data: data,
          child: MaterialApp(
            theme: gfThemeData(
              Brightness.light,
            ).copyWith(platform: TargetPlatform.android),
            home: Scaffold(bottomNavigationBar: _dock(index: 0)),
          ),
        ),
      );

      expect(
        find.byType(BackdropFilter),
        glass ? findsOneWidget : findsNothing,
      );
      expect(find.bySemanticsLabel('首页'), findsOneWidget);
    });
  }

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

final Finder _restingLayer = find.byKey(
  const ValueKey<String>('gf-bottom-navigation-resting-layer'),
);
final Finder _selectedLayer = find.byKey(
  const ValueKey<String>('gf-bottom-navigation-selected-layer'),
);
final Finder _lens = find.byKey(
  const ValueKey<String>('gf-bottom-navigation-lens'),
);
// A resting lens spans its destination's column, inset 5px from the rim.
Size _restingLens(WidgetTester tester, int index) {
  final Size target = tester.getSize(_target(index));
  return Size(target.width, target.height - 10);
}

Finder _target(int index) =>
    find.byKey(ValueKey<String>('gf-bottom-navigation-target-$index'));

Widget _dock({required int index, ValueChanged<int>? onSelected}) =>
    GfBottomNavigation(
      currentIndex: index,
      onSelected: onSelected ?? (_) {},
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
    );
