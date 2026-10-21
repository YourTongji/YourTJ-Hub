import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// A composer stand-in above the slot, in a page that does not resize for the
/// keyboard, as the slot requires.
Widget _page(
  GfInputAccessory accessory,
  FocusNode panelFocus, {
  bool reduceMotion = false,
  Widget? panel,
}) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  builder: (context, child) => MediaQuery(
    data: MediaQuery.of(context).copyWith(disableAnimations: reduceMotion),
    child: child!,
  ),
  home: Scaffold(
    resizeToAvoidBottomInset: false,
    body: LayoutBuilder(
      builder: (context, box) => Column(
        children: [
          const Spacer(),
          const SizedBox(
            key: Key('editor'),
            width: double.infinity,
            height: 56,
          ),
          GfInputAccessorySlot(
            accessory: accessory,
            panelFocusNode: panelFocus,
            availableHeight: box.maxHeight,
            panelBuilder: (_) =>
                panel ?? const SizedBox.expand(key: Key('panel')),
          ),
        ],
      ),
    ),
  ),
);

void main() {
  late FocusNode panelFocus;

  setUp(() => panelFocus = FocusNode());
  tearDown(() => panelFocus.dispose());

  void useScreen(WidgetTester tester) {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
  }

  double editorBottom(WidgetTester tester) =>
      tester.getBottomLeft(find.byKey(const Key('editor'))).dy;

  /// Steps the IME inset over [frames] frames and records the editor edge.
  Future<List<double>> animateKeyboard(
    WidgetTester tester,
    double from,
    double to, {
    int frames = 10,
  }) async {
    final trace = <double>[editorBottom(tester)];
    for (var frame = 1; frame <= frames; frame++) {
      tester.view.viewInsets = FakeViewPadding(
        bottom: from + (to - from) * frame / frames,
      );
      await tester.pump(const Duration(milliseconds: 16));
      trace.add(editorBottom(tester));
    }
    return trace;
  }

  Future<void> settle(WidgetTester tester) async {
    await tester.pump(const Duration(milliseconds: 700));
    await tester.pumpAndSettle();
  }

  Future<double> openSettledKeyboard(WidgetTester tester) async {
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    await animateKeyboard(tester, 0, 300);
    await settle(tester);
    return editorBottom(tester);
  }

  testWidgets('keyboard and panel trade places without moving the editor', (
    tester,
  ) async {
    useScreen(tester);
    final anchor = await openSettledKeyboard(tester);
    expect(anchor, 844 - 300);

    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    final toPanel = await animateKeyboard(tester, 300, 0);
    await settle(tester);
    expect(toPanel, everyElement(anchor));
    expect(find.byKey(const Key('panel')), findsOneWidget);
    expect(
      tester.getSize(find.byKey(const Key('input-accessory-slot'))).height,
      300,
    );

    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    final toKeyboard = await animateKeyboard(tester, 0, 300);
    expect(toKeyboard, everyElement(anchor));
    // The panel stays under the rising keyboard until it is covered.
    await settle(tester);
    expect(find.byKey(const Key('panel')), findsNothing);
    expect(editorBottom(tester), anchor);
  });

  testWidgets('a reserved keyboard space holds the editor until it returns', (
    tester,
  ) async {
    useScreen(tester);
    final anchor = await openSettledKeyboard(tester);

    // A sheet borrows focus: the keyboard leaves, its space stays.
    await tester.pumpWidget(_page(GfInputAccessory.reserved, panelFocus));
    final leaving = await animateKeyboard(tester, 300, 0);
    await settle(tester);
    expect([...leaving, editorBottom(tester)], everyElement(anchor));
    expect(find.byKey(const Key('panel')), findsNothing);

    // Focus comes back and the keyboard fills the reserve.
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    final returning = await animateKeyboard(tester, 0, 300);
    await settle(tester);
    expect([...returning, editorBottom(tester)], everyElement(anchor));

    // Without a returning keyboard the reserve is released.
    await tester.pumpWidget(_page(GfInputAccessory.reserved, panelFocus));
    await animateKeyboard(tester, 300, 0);
    await tester.pumpWidget(_page(GfInputAccessory.none, panelFocus));
    await settle(tester);
    expect(editorBottom(tester), 844);
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    await settle(tester);
    expect(editorBottom(tester), 844, reason: 'no keyboard appeared');
  });

  testWidgets('a reversed switch follows the latest intent without bouncing', (
    tester,
  ) async {
    useScreen(tester);
    final anchor = await openSettledKeyboard(tester);
    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    final leaving = await animateKeyboard(tester, 300, 150, frames: 5);
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    final returning = await animateKeyboard(tester, 150, 300, frames: 5);
    await settle(tester);
    expect([...leaving, ...returning], everyElement(anchor));
    expect(find.byKey(const Key('panel')), findsNothing);
  });

  testWidgets('a shorter keyboard settles below the panel in one direction', (
    tester,
  ) async {
    useScreen(tester);
    await openSettledKeyboard(tester);
    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    await animateKeyboard(tester, 300, 0);
    await settle(tester);
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    // E.g. the user hid the suggestion strip: the keyboard is now 240 high.
    final trace = await animateKeyboard(tester, 0, 240);
    for (var i = 0; i < 12; i++) {
      await tester.pump(const Duration(milliseconds: 30));
      trace.add(editorBottom(tester));
    }
    await settle(tester);
    trace.add(editorBottom(tester));
    for (var i = 1; i < trace.length; i++) {
      expect(trace[i], greaterThanOrEqualTo(trace[i - 1] - .01));
    }
    expect(trace.last, 844 - 240);
  });

  testWidgets('without a keyboard the panel rises and leaves monotonically', (
    tester,
  ) async {
    useScreen(tester);
    await tester.pumpWidget(_page(GfInputAccessory.none, panelFocus));
    expect(editorBottom(tester), 844);

    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    final opening = <double>[editorBottom(tester)];
    for (var i = 0; i < 16; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      opening.add(editorBottom(tester));
    }
    for (var i = 1; i < opening.length; i++) {
      expect(opening[i], lessThanOrEqualTo(opening[i - 1] + .01));
    }
    expect(opening.last, lessThan(844 - 80));

    await tester.pumpWidget(_page(GfInputAccessory.none, panelFocus));
    final closing = <double>[editorBottom(tester)];
    for (var i = 0; i < 16; i++) {
      await tester.pump(const Duration(milliseconds: 16));
      closing.add(editorBottom(tester));
    }
    for (var i = 1; i < closing.length; i++) {
      expect(closing[i], greaterThanOrEqualTo(closing[i - 1] - .01));
    }
    expect(closing.last, 844);
    expect(find.byKey(const Key('panel')), findsNothing);
  });

  testWidgets('reduced motion opens and closes the panel in one frame', (
    tester,
  ) async {
    useScreen(tester);
    await tester.pumpWidget(
      _page(GfInputAccessory.none, panelFocus, reduceMotion: true),
    );
    await tester.pumpWidget(
      _page(GfInputAccessory.panel, panelFocus, reduceMotion: true),
    );
    final open = editorBottom(tester);
    await tester.pump(const Duration(milliseconds: 16));
    expect(editorBottom(tester), open);
    expect(open, lessThan(844 - 80));
    await tester.pumpWidget(
      _page(GfInputAccessory.none, panelFocus, reduceMotion: true),
    );
    await tester.pump();
    expect(editorBottom(tester), 844);
  });

  testWidgets('a hardware keyboard releases the held panel', (tester) async {
    useScreen(tester);
    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    await tester.pumpAndSettle();
    final withPanel = editorBottom(tester);
    await tester.pumpWidget(_page(GfInputAccessory.keyboard, panelFocus));
    await tester.pump(const Duration(milliseconds: 300));
    // Held while a software keyboard could still arrive.
    expect(editorBottom(tester), withPanel);
    await settle(tester);
    expect(editorBottom(tester), 844);
    expect(find.byKey(const Key('panel')), findsNothing);
  });

  testWidgets('a field inside the panel keeps it above its keyboard', (
    tester,
  ) async {
    useScreen(tester);
    final search = FocusNode();
    addTearDown(search.dispose);
    final panel = Column(
      children: [
        TextField(key: const Key('panel-search'), focusNode: search),
        const Expanded(child: SizedBox(key: Key('panel'))),
      ],
    );
    await tester.pumpWidget(
      _page(GfInputAccessory.panel, panelFocus, panel: panel),
    );
    panelFocus.requestFocus();
    await tester.pumpAndSettle();
    search.requestFocus();
    await tester.pump();
    await animateKeyboard(tester, 0, 300);
    await settle(tester);
    expect(
      tester.getBottomLeft(find.byKey(const Key('panel-search'))).dy,
      lessThanOrEqualTo(844 - 300),
    );
    expect(editorBottom(tester), lessThan(844 - 300 - 48));
    expect(find.byKey(const Key('panel-search')), findsOneWidget);
    tester.view.viewInsets = const FakeViewPadding();
  });

  testWidgets('the bottom safe area is filled once and kept under the panel', (
    tester,
  ) async {
    useScreen(tester);
    tester.view.viewPadding = const FakeViewPadding(bottom: 34);
    tester.view.padding = const FakeViewPadding(bottom: 34);
    await tester.pumpWidget(_page(GfInputAccessory.none, panelFocus));
    final slot = find.byKey(const Key('input-accessory-slot'));
    expect(tester.getSize(slot).height, 34);

    await tester.pumpWidget(_page(GfInputAccessory.panel, panelFocus));
    await tester.pumpAndSettle();
    // Panel content stops above the home indicator.
    expect(tester.getBottomLeft(find.byKey(const Key('panel'))).dy, 844 - 34);
  });
}
