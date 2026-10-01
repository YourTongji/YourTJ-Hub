import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  Future<void> pumpMenu(
    WidgetTester tester, {
    required ValueNotifier<bool> valid,
    required ValueChanged<String?> onResult,
    double scale = 1,
    double keyboard = 0,
    bool reduced = false,
    Rect source = const Rect.fromLTWH(270, 500, 30, 40),
    Widget? preview,
    List<String>? labels,
  }) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            textScaler: TextScaler.linear(scale),
            viewInsets: EdgeInsets.only(bottom: keyboard),
            disableAnimations: reduced,
          ),
          child: child!,
        ),
        home: Scaffold(
          body: Builder(
            builder: (context) => Center(
              child: TextButton(
                child: const Text('Open'),
                onPressed: () async {
                  final result = await showGfContextMenu<String>(
                    context,
                    sourceRect: source,
                    preview: preview ?? Text('Selected message\n' * 40),
                    semanticLabel: 'Message actions',
                    sourceValid: valid,
                    actions: [
                      for (var i = 0; i < (labels?.length ?? 6); i++)
                        GfContextAction(
                          value: '$i',
                          label: labels?[i] ?? 'Action $i with a long label',
                          symbol: 'copy',
                        ),
                    ],
                  );
                  onResult(result);
                },
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pump();
  }

  for (final top in [40.0, 450.0]) {
    testWidgets('action menu anchors to its button at $top without a sheet', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      String? selected;
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.dark),
          home: Scaffold(
            body: Stack(
              children: [
                Positioned(
                  top: top,
                  right: 16,
                  child: GfActionMenuButton<String>(
                    key: const Key('anchor'),
                    tooltip: 'Actions',
                    icon: const Icon(Icons.more_horiz),
                    onSelected: (value) => selected = value,
                    itemBuilder: (_) => const [
                      GfContextAction(
                        value: 'selected',
                        label: 'Current',
                        symbol: 'check',
                        selected: true,
                      ),
                      GfContextAction(
                        value: 'disabled',
                        label: 'Disabled',
                        symbol: 'copy',
                        enabled: false,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
      final anchor = tester.getRect(find.byKey(const Key('anchor')));
      await tester.tap(find.byTooltip('Actions'));
      await tester.pumpAndSettle();
      final menu = tester.getRect(find.byKey(const Key('gf-context-menu')));
      expect(find.byType(BottomSheet), findsNothing);
      expect(find.byKey(const Key('gf-context-preview')), findsNothing);
      expect(find.byType(Divider), findsNothing);
      expect(menu.right, closeTo(anchor.right, .01));
      if (top < 100) {
        expect(menu.top, closeTo(anchor.bottom + 8, .01));
      } else {
        expect(menu.bottom, closeTo(anchor.top - 8, .01));
      }
      expect(
        tester.widget<GfMenuItem>(find.byType(GfMenuItem).first).selected,
        isTrue,
      );
      await tester.tap(find.text('Disabled'));
      await tester.pumpAndSettle();
      expect(selected, isNull);
      expect(find.byKey(const Key('gf-context-menu')), findsOneWidget);
      await tester.tap(find.text('Current'));
      await tester.pumpAndSettle();
      expect(selected, 'selected');
    });
  }

  testWidgets('removing an action-menu source cancels its open route', (
    tester,
  ) async {
    var visible = true;
    var invoked = false;
    late StateSetter rebuild;
    await tester.pumpWidget(
      MaterialApp(
        home: StatefulBuilder(
          builder: (context, setState) {
            rebuild = setState;
            return Scaffold(
              body: visible
                  ? GfActionMenuButton<String>(
                      icon: const Icon(Icons.more_horiz),
                      tooltip: 'Actions',
                      onSelected: (_) => invoked = true,
                      itemBuilder: (_) => const [
                        GfContextAction(value: 'copy', label: 'Copy'),
                      ],
                    )
                  : const SizedBox(),
            );
          },
        ),
      ),
    );
    await tester.tap(find.byTooltip('Actions'));
    await tester.pumpAndSettle();
    rebuild(() => visible = false);
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('gf-context-menu')), findsNothing);
    expect(invoked, isFalse);
    expect(tester.takeException(), isNull);
  });

  testWidgets('menu fits its longest label without action separators', (
    tester,
  ) async {
    final valid = ValueNotifier(true);
    addTearDown(valid.dispose);
    await pumpMenu(
      tester,
      valid: valid,
      onResult: (_) {},
      labels: ['回复', '复制整条消息', '转发'],
    );
    await tester.pumpAndSettle();
    final menu = tester.getRect(find.byKey(const Key('gf-context-menu')));
    expect(menu.width, lessThan(200));
    // Longest text + 18px icon + 10px gap + 24px row + 8px surface insets.
    final label = tester.renderObject<RenderBox>(find.text('复制整条消息'));
    expect(
      menu.width,
      closeTo(label.getMaxIntrinsicWidth(double.infinity) + 60, .1),
    );
    final dividers = find.descendant(
      of: find.byKey(const Key('gf-context-menu')),
      matching: find.byType(Divider),
    );
    expect(dividers, findsNothing);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
  });

  testWidgets('selected bubble stays outside the glass action surface', (
    tester,
  ) async {
    final valid = ValueNotifier(true);
    addTearDown(valid.dispose);
    await pumpMenu(
      tester,
      valid: valid,
      onResult: (_) {},
      source: const Rect.fromLTWH(60, 120, 180, 48),
      preview: const SizedBox(height: 48, child: Text('Selected bubble')),
    );
    await tester.pumpAndSettle();
    final preview = find.text('Selected bubble');
    expect(
      find.ancestor(of: preview, matching: find.byType(GfLiquidSurface)),
      findsNothing,
    );
    final bubble = tester.getRect(find.byKey(const Key('gf-context-preview')));
    final menu = tester.getRect(find.byKey(const Key('gf-context-menu')));
    expect(bubble.width, 180);
    expect(bubble.left, 60);
    expect(menu.left, bubble.left);
    expect(
      menu.width,
      closeTo(
        tester
                .renderObject<RenderBox>(
                  find.text('Action 0 with a long label'),
                )
                .getMaxIntrinsicWidth(double.infinity) +
            60,
        .1,
      ),
    );
    expect(menu.top - bubble.bottom, 8);
    expect(bubble.overlaps(menu), isFalse);
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
  });

  testWidgets('only the modal backdrop blurs and cancellation removes it', (
    tester,
  ) async {
    final valid = ValueNotifier(true);
    addTearDown(valid.dispose);
    await pumpMenu(tester, valid: valid, onResult: (_) {});
    await tester.pumpAndSettle();
    final backdrop = find.byKey(const Key('gf-context-backdrop'));
    expect(backdrop, findsOneWidget);
    expect(tester.widget<BackdropFilter>(backdrop).enabled, isTrue);
    for (final key in ['gf-context-preview', 'gf-context-menu']) {
      expect(
        find.ancestor(of: find.byKey(Key(key)), matching: backdrop),
        findsNothing,
        reason: 'Selected content and action text must stay sharp',
      );
    }
    await tester.tapAt(const Offset(4, 4));
    await tester.pumpAndSettle();
    expect(backdrop, findsNothing);
    expect(find.text('Open'), findsOneWidget);
  });

  testWidgets('reduced effects disable the full-screen blur', (tester) async {
    final valid = ValueNotifier(true);
    addTearDown(valid.dispose);
    await pumpMenu(tester, valid: valid, onResult: (_) {}, reduced: true);
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<BackdropFilter>(find.byKey(const Key('gf-context-backdrop')))
          .enabled,
      isFalse,
    );
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
  });

  testWidgets('large labels and long preview fit above the keyboard', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final valid = ValueNotifier(true);
    final results = <String?>[];
    await pumpMenu(
      tester,
      valid: valid,
      onResult: results.add,
      scale: 2,
      keyboard: 280,
      source: const Rect.fromLTWH(24, 500, 260, 600),
    );
    await tester.pumpAndSettle();
    final rect = tester.getRect(find.byKey(const Key('gf-context-menu')));
    expect(rect.left, greaterThanOrEqualTo(12));
    expect(rect.right, lessThanOrEqualTo(308));
    expect(rect.top, greaterThanOrEqualTo(12));
    expect(rect.bottom, lessThanOrEqualTo(348));
    final previewRect = tester.getRect(
      find.byKey(const Key('gf-context-preview')),
    );
    expect(previewRect.overlaps(rect), isFalse);
    expect(previewRect.top, greaterThanOrEqualTo(12));
    expect(rect.top - previewRect.bottom, 8);
    expect(tester.takeException(), isNull);
    final last = find.text('Action 5 with a long label');
    await tester.ensureVisible(last);
    await tester.pumpAndSettle();
    expect(
      tester.getRect(find.byKey(const Key('gf-context-preview'))),
      previewRect,
      reason: 'Scrolling actions never scrolls the selected message',
    );
    await tester.tap(last);
    await tester.pumpAndSettle();
    expect(results, ['5']);
    valid.dispose();
  });

  testWidgets(
    'outgoing bubble keeps its right edge with an independent menu width',
    (tester) async {
      final valid = ValueNotifier(true);
      addTearDown(valid.dispose);
      await pumpMenu(
        tester,
        valid: valid,
        onResult: (_) {},
        source: const Rect.fromLTWH(640, 530, 100, 40),
        preview: const SizedBox(height: 40, child: Text('Sent')),
      );
      await tester.pumpAndSettle();
      final bubble = tester.getRect(
        find.byKey(const Key('gf-context-preview')),
      );
      final menu = tester.getRect(find.byKey(const Key('gf-context-menu')));
      expect(bubble.width, 100);
      expect(bubble.right, 740);
      expect(menu.right, bubble.right);
      expect(
        menu.width,
        closeTo(
          tester
                  .renderObject<RenderBox>(
                    find.text('Action 0 with a long label'),
                  )
                  .getMaxIntrinsicWidth(double.infinity) +
              60,
          .1,
        ),
      );
      expect(menu.top - bubble.bottom, 8);
      expect(menu.bottom, lessThanOrEqualTo(588));
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
    },
  );

  testWidgets('source invalidation closes the surface without an action', (
    tester,
  ) async {
    final valid = ValueNotifier(true);
    final results = <String?>[];
    await pumpMenu(tester, valid: valid, onResult: results.add);
    await tester.pumpAndSettle();
    valid.value = false;
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('gf-context-menu')), findsNothing);
    expect(results, [null]);
    valid.dispose();
  });

  testWidgets('keyboard can activate a menu action and Escape cancels', (
    tester,
  ) async {
    final valid = ValueNotifier(true);
    final results = <String?>[];
    await pumpMenu(tester, valid: valid, onResult: results.add, reduced: true);
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.tab);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();
    expect(results, ['0']);
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(results, ['0', null]);
    valid.dispose();
  });

  for (final fraction in [.1, .3, .7, 1.0]) {
    testWidgets('back cancels menu entry at $fraction', (tester) async {
      final valid = ValueNotifier(true);
      final results = <String?>[];
      await pumpMenu(tester, valid: valid, onResult: results.add);
      final preview = find.byKey(const Key('gf-context-preview'));
      expect(tester.getTopLeft(preview), const Offset(270, 500));
      await tester.pump(Duration(microseconds: (220000 * fraction).round()));
      final position = tester.getTopLeft(preview);
      await tester.binding.handlePopRoute();
      await tester.pump();
      expect(
        tester.getTopLeft(preview),
        position,
        reason:
            'Reversing starts from the current presentation, without a jump',
      );
      await tester.pumpAndSettle();
      expect(results, [null]);
      expect(find.byKey(const Key('gf-context-menu')), findsNothing);
      expect(tester.takeException(), isNull);
      valid.dispose();
    });
  }
}
