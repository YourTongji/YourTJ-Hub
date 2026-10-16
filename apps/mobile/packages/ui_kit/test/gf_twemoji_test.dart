import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart'
    show debugDefaultTargetPlatformOverride;
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void _captureClipboard(WidgetTester tester, void Function(String?) onText) {
  tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
    SystemChannels.platform,
    (call) async {
      if (call.method == 'Clipboard.setData') {
        onText((call.arguments as Map)['text'] as String?);
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
}

Future<void> _selectAllAndCopy(WidgetTester tester) async {
  final region = tester.state<SelectableRegionState>(
    find.byType(SelectableRegion),
  );
  region.selectAll(SelectionChangedCause.keyboard);
  await tester.pump();
  final area = find.byType(GfSelectionArea);
  final text = area.evaluate().isNotEmpty
      ? find.descendant(of: area, matching: find.byType(RichText)).first
      : find.byType(EditableText);
  Actions.invoke(tester.element(text), CopySelectionTextIntent.copy);
  await tester.pump();
}

void main() {
  group('Twemoji sequence matching', () {
    test('matches complete mixed-language and composed emoji sequences', () {
      const text = '中文 A😀 🐦‍🔥 🇺🇸 1️⃣ 👩🏽‍💻!';
      final segments = parseTwemojiText(text);
      final emoji = segments
          .where((segment) => segment.isEmoji)
          .map((segment) => segment.value)
          .toList();

      expect(emoji, <String>['😀', '🐦‍🔥', '🇺🇸', '1️⃣', '👩🏽‍💻']);
      expect(segments.map((segment) => segment.value).join(), text);
      expect(twemojiVersion, '17.0.3');
    });

    test(
      'matches optional variation selectors without changing source text',
      () {
        expect(parseTwemojiText('❤\uFE0F'), <TwemojiSegment>[
          const TwemojiSegment.emoji('❤\uFE0F', '2764.png'),
        ]);
      },
    );

    testWidgets('fast candidate scan keeps symbol, flag and keycap artwork', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: GfText('© ® ™ ❤️ 🐦‍🔥 🇺🇸 1️⃣ #️⃣')),
        ),
      );
      await tester.pump();

      expect(find.byType(Image), findsNWidgets(8));
      expect(find.bySemanticsLabel('🐦‍🔥'), findsOneWidget);
      expect(find.bySemanticsLabel('🇺🇸'), findsOneWidget);
    });

    testWidgets('ordinary digits are not treated as keycap Emoji', (
      tester,
    ) async {
      await tester.pumpWidget(
        const MaterialApp(home: Scaffold(body: GfText('2026 and 514'))),
      );
      await tester.pump();
      expect(find.byType(Image), findsNothing);
      expect(find.text('2026 and 514'), findsOneWidget);
    });

    test('keeps unsupported joined sequences together as original text', () {
      expect(parseTwemojiText('🐦‍🛹'), <TwemojiSegment>[
        const TwemojiSegment.text('🐦‍🛹'),
      ]);
    });

    test(
      'copy maps an inline image placeholder to its complete source sequence',
      () {
        const span = TextSpan(text: '前文 🐦‍🔥 后文');
        // "前文 " occupies three UTF-16 units; the image is one visual unit.
        expect(twemojiCopyTextForDisplayRange(span, 3, 4), '🐦‍🔥');
        expect(twemojiCopyTextForDisplayRange(span, 0, 4), '前文 🐦‍🔥');
      },
    );

    test('preserves native WidgetSpan placeholders and source offsets', () {
      const span = TextSpan(
        children: <InlineSpan>[
          TextSpan(text: 'a'),
          WidgetSpan(child: SizedBox(width: 8, height: 8)),
          TextSpan(text: '🐦‍🔥z'),
        ],
      );

      expect(twemojiCopyTextForDisplayRange(span, 0, 4), 'a\uFFFC🐦‍🔥z');
      expect(twemojiCopyTextForDisplayRange(span, 2, 3), '🐦‍🔥');
    });
  });

  testWidgets('GfText paints one accessible inline image per full sequence', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: GfText('中文 🐦‍🔥!'))),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
    final image = tester.widget<Image>(find.byType(Image));
    final resizedImage = image.image as ResizeImage;
    final assetImage = resizedImage.imageProvider as AssetImage;
    expect(assetImage.assetName, contains('1f426-200d-1f525.png'));
    expect(assetImage.package, 'ui_kit');
    expect(resizedImage.width, 72);
    expect(resizedImage.height, 72);
    expect(find.bySemanticsLabel('🐦‍🔥'), findsOneWidget);
    semantics.dispose();
  });

  testWidgets('decorated names keep Twemoji artwork colors', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: Column(
            children: [
              GfDecoratedName(
                '昵称 🐦‍🔥',
                colors: <Color>[Colors.red, Colors.blue],
              ),
              GfDecoratedName('渐变昵称', colors: <Color>[Colors.red, Colors.blue]),
            ],
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
    expect(find.bySemanticsLabel('🐦‍🔥'), findsOneWidget);
    expect(find.byType(ShaderMask), findsOneWidget);
    final emojiElement = tester.element(find.byType(Image));
    expect(emojiElement.findAncestorWidgetOfExactType<ShaderMask>(), isNull);
  });

  testWidgets('selectable Emoji copy restores its complete source sequence', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);

    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: GfSelectableText('中文\uE000 A🐦‍🔥B\uF8FF')),
      ),
    );
    await tester.pump();

    tester
        .state<EditableTextState>(find.byType(EditableText))
        .selectAll(SelectionChangedCause.keyboard);
    await tester.pump();
    final editableFocus = tester.element(
      find
          .descendant(
            of: find.byType(EditableText),
            matching: find.byType(Focus),
          )
          .first,
    );
    Actions.invoke(editableFocus, CopySelectionTextIntent.copy);
    await tester.pump();

    expect(copied, '中文\uE000 A🐦‍🔥B\uF8FF');
  });

  testWidgets(
    'selection area preserves PUA characters on either side of Emoji',
    (tester) async {
      const before = '\uE000 first 🐦‍🔥';
      const after = 'second 🐦‍🔥\uE001\uF8FF';
      final source = '$before$after';
      String? copied;
      _captureClipboard(tester, (text) => copied = text);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GfSelectionArea(
              selectionSource: source,
              child: const Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[GfText(before), GfText(after)],
              ),
            ),
          ),
        ),
      );
      await tester.pump();

      expect(find.byType(Image), findsNWidgets(2));
      await _selectAllAndCopy(tester);
      expect(copied, source);
    },
  );

  testWidgets('selection area keeps native WidgetSpan U+FFFC intact', (
    tester,
  ) async {
    const source = 'before 🐦‍🔥 after';
    String? copied;
    _captureClipboard(tester, (text) => copied = text);

    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: GfSelectionArea(
            selectionSource: source,
            child: GfText.rich(
              TextSpan(
                children: <InlineSpan>[
                  TextSpan(text: 'before 🐦‍🔥'),
                  WidgetSpan(child: SizedBox(width: 8, height: 8)),
                  TextSpan(text: ' after'),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    await _selectAllAndCopy(tester);
    expect(copied, 'before 🐦‍🔥 after');
  });

  testWidgets('selection area desktop Ctrl+C copies source Unicode', (
    tester,
  ) async {
    const source = '\uE000 before 🐦‍🔥 after \uF8FF';
    String? copied;
    final focusNode = FocusNode();
    addTearDown(focusNode.dispose);
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: GfSelectionArea(
            selectionSource: source,
            focusNode: focusNode,
            child: GfText(source),
          ),
        ),
      ),
    );
    await tester.pump();

    final region = tester.state<SelectableRegionState>(
      find.byType(SelectableRegion),
    );
    region.selectAll(SelectionChangedCause.keyboard);
    focusNode.requestFocus();
    await tester.pump();
    await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
    await tester.sendKeyEvent(LogicalKeyboardKey.keyC);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
    await tester.pump();

    expect(copied, source);
  });

  testWidgets('selection area re-registers source after content rebuild', (
    tester,
  ) async {
    String source = '🐦‍🔥';
    String? copied;
    _captureClipboard(tester, (text) => copied = text);

    await tester.pumpWidget(
      MaterialApp(
        home: StatefulBuilder(
          builder: (context, setState) => Scaffold(
            body: Column(
              children: <Widget>[
                TextButton(
                  onPressed: () => setState(() => source = '\uE000🐦‍🔥'),
                  child: const Text('change'),
                ),
                GfSelectionArea(selectionSource: source, child: GfText(source)),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('change'));
    await tester.pumpAndSettle();

    expect(find.byType(Image), findsOneWidget);
    await _selectAllAndCopy(tester);
    expect(copied, '\uE000🐦‍🔥');
  });

  testWidgets('selection copy safely falls back when valid text fills PUA pool', (
    tester,
  ) async {
    final source =
        '${String.fromCharCodes(List<int>.generate(0x1900, (i) => 0xe000 + i))}'
        '${String.fromCharCode(0xf0000)}${String.fromCharCode(0x10fffd)}🐦‍🔥';
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: GfSelectionArea(selectionSource: source, child: GfText(source)),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(Image), findsNothing);
    await _selectAllAndCopy(tester);
    expect(copied, source);
  });

  testWidgets(
    'selection toolbar copy restores PUA and Emoji on Android and iOS',
    (tester) async {
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      for (final platform in <TargetPlatform>[
        TargetPlatform.android,
        TargetPlatform.iOS,
      ]) {
        debugDefaultTargetPlatformOverride = platform;
        String? copied;
        _captureClipboard(tester, (text) => copied = text);
        const source = '\uE000 A🐦‍🔥B\uF8FF';
        await tester.pumpWidget(
          const MaterialApp(
            home: Scaffold(
              body: GfSelectionArea(
                selectionSource: source,
                child: GfText(source),
              ),
            ),
          ),
        );
        await tester.pump();

        final region = tester.state<SelectableRegionState>(
          find.byType(SelectableRegion),
        );
        region.selectAll(SelectionChangedCause.longPress);
        await tester.pump();
        final selectionArea = tester.widget<SelectionArea>(
          find.byType(SelectionArea),
        );
        final toolbar =
            selectionArea.contextMenuBuilder!(
                  tester.element(find.byType(SelectionArea)),
                  region,
                )
                as AdaptiveTextSelectionToolbar;
        toolbar.buttonItems!
            .singleWhere((item) => item.type == ContextMenuButtonType.copy)
            .onPressed!();
        await tester.pump();

        expect(copied, source, reason: '$platform toolbar copy');
        await tester.pumpWidget(const SizedBox.shrink());
      }
      debugDefaultTargetPlatformOverride = null;
    },
  );

  testWidgets('Emoji SelectableText retains native selection parameters', (
    tester,
  ) async {
    void onTap() {}
    var customMenuBuilt = false;
    Widget customMenu(BuildContext context, EditableTextState state) {
      customMenuBuilt = true;
      return AdaptiveTextSelectionToolbar.buttonItems(
        anchors: state.contextMenuAnchors,
        buttonItems: state.contextMenuButtonItems,
      );
    }

    String? copied;
    _captureClipboard(tester, (text) => copied = text);

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: GfSelectableText(
            '\uE000🐦‍🔥 selectable\uF8FF',
            showCursor: true,
            autofocus: true,
            onTap: onTap,
            contextMenuBuilder: customMenu,
          ),
        ),
      ),
    );
    await tester.pump();

    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    expect(selectable.showCursor, isTrue);
    expect(selectable.autofocus, isTrue);
    expect(selectable.onTap, same(onTap));
    expect(selectable.contextMenuBuilder, isNotNull);
    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.longPress);
    await tester.pump();
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    expect(customMenuBuilt, isTrue);
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();
    expect(copied, '\uE000🐦‍🔥 selectable\uF8FF');
  });

  testWidgets('SelectableText toolbar copy restores Unicode Emoji', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: GfSelectableText('\uE000A🐦‍🔥B\uF8FF')),
      ),
    );
    await tester.pump();

    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.longPress);
    await tester.pump();
    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();

    expect(copied, '\uE000A🐦‍🔥B\uF8FF');
  });

  testWidgets('desktop Meta+C copies the source Unicode selection', (
    tester,
  ) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
    try {
      String? copied;
      _captureClipboard(tester, (text) => copied = text);
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: GfSelectableText('before 🐦‍🔥 after')),
        ),
      );
      await tester.pump();

      final editable = tester.state<EditableTextState>(
        find.byType(EditableText),
      );
      editable.selectAll(SelectionChangedCause.keyboard);
      editable.widget.focusNode.requestFocus();
      await tester.pump();
      await tester.sendKeyDownEvent(LogicalKeyboardKey.metaLeft);
      await tester.sendKeyEvent(LogicalKeyboardKey.keyC);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.metaLeft);
      await tester.pump();

      expect(copied, 'before 🐦‍🔥 after');
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  testWidgets('SelectableText copy preserves native WidgetSpan offsets', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: GfSelectableText.rich(
            TextSpan(
              children: <InlineSpan>[
                TextSpan(text: 'a'),
                WidgetSpan(child: SizedBox(width: 8, height: 8)),
                TextSpan(text: '🐦‍🔥z'),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.longPress);
    await tester.pump();
    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();

    expect(copied, 'a\uFFFC🐦‍🔥z');
  });

  testWidgets('SelectableText.rich accepts a root WidgetSpan without Emoji', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: GfSelectableText.rich(
            WidgetSpan(child: SizedBox(width: 8, height: 8)),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(tester.takeException(), isNull);
    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    expect(selectable.textSpan, isA<TextSpan>());
    expect(selectable.textSpan!.children!.single, isA<WidgetSpan>());
    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.keyboard);
    await tester.pump();
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();

    expect(copied, '\uFFFC');
  });

  testWidgets(
    'SelectableText.rich accepts a root WidgetSpan with Emoji child',
    (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: GfSelectableText.rich(WidgetSpan(child: GfText('widget 😀'))),
          ),
        ),
      );
      await tester.pump();

      expect(tester.takeException(), isNull);
      expect(find.byType(Image), findsOneWidget);
      expect(find.bySemanticsLabel('😀'), findsOneWidget);
    },
  );

  testWidgets('nested WidgetSpan without Emoji keeps selectable copy offsets', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: GfSelectableText.rich(
            TextSpan(
              children: <InlineSpan>[
                TextSpan(text: 'before'),
                WidgetSpan(child: SizedBox(width: 8, height: 8)),
                TextSpan(text: 'after'),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.keyboard);
    await tester.pump();
    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();

    expect(copied, 'before\uFFFCafter');
  });

  testWidgets('nested WidgetSpan with Emoji keeps selectable copy offsets', (
    tester,
  ) async {
    String? copied;
    _captureClipboard(tester, (text) => copied = text);
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: GfSelectableText.rich(
            TextSpan(
              children: <InlineSpan>[
                TextSpan(text: 'before '),
                WidgetSpan(child: SizedBox(width: 8, height: 8)),
                TextSpan(text: ' 🐦‍🔥 after'),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
    final editable = tester.state<EditableTextState>(find.byType(EditableText));
    editable.selectAll(SelectionChangedCause.keyboard);
    await tester.pump();
    final selectable = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    final toolbar =
        selectable.contextMenuBuilder!(
              tester.element(find.byType(EditableText)),
              editable,
            )
            as AdaptiveTextSelectionToolbar;
    toolbar.buttonItems!
        .singleWhere((item) => item.type == ContextMenuButtonType.copy)
        .onPressed!();
    await tester.pump();

    expect(copied, 'before \uFFFC 🐦‍🔥 after');
  });
}
