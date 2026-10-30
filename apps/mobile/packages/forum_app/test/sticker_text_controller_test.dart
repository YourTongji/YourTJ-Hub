import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_suggestion_bar.dart';
import 'package:forum_app/src/widgets/stickers/sticker_text_controller.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/sticker_fixtures.dart';

class _MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

class _SuggestionRepository extends ComposerStickerRepository {
  _SuggestionRepository(super.client);

  @override
  Future<List<StickerOfficialPackPayload>> officialPacks() async => [];

  @override
  Future<List<StickerMarketPackPayload>> myPacks() async => [];
}

GfApiClient _client() => GfApiClient(
  dio: Dio(BaseOptions(baseUrl: 'http://test')),
  tokenStorage: _MemoryTokenStorage(),
);

StickerLibrary _library() {
  final library = StickerLibrary(ComposerStickerRepository(_client()))
    ..remember([composerSticker]);
  addTearDown(library.dispose);
  return library;
}

const _token = '[:sticker:smile:]';

StickerTextEditingController _controller(String text) {
  final controller = StickerTextEditingController(text: text)
    ..library = _library();
  addTearDown(controller.dispose);
  return controller;
}

/// Simulates the platform reporting one edit, as EditableText would.
void _edit(
  StickerTextEditingController controller,
  String text,
  TextSelection selection,
) => controller.value = TextEditingValue(text: text, selection: selection);

Widget _field(StickerTextEditingController controller, {Widget? above}) =>
    MaterialApp(
      theme: gfThemeData(Brightness.light),
      home: Scaffold(
        body: Column(
          children: [
            ?above,
            TextField(
              controller: controller,
              style: const TextStyle(fontSize: 16),
            ),
          ],
        ),
      ),
    );

void main() {
  group('atomic deletion', () {
    test('backspace after a token removes the whole token', () {
      final controller = _controller('a$_token b');
      final end = 1 + _token.length;
      controller.selection = TextSelection.collapsed(offset: end);
      _edit(
        controller,
        'a${_token.substring(0, _token.length - 1)} b',
        TextSelection.collapsed(offset: end - 1),
      );
      expect(controller.text, 'a b');
      expect(controller.selection, const TextSelection.collapsed(offset: 1));
    });

    test('forward delete before a token removes the whole token', () {
      final controller = _controller('a${_token}b');
      controller.selection = const TextSelection.collapsed(offset: 1);
      _edit(
        controller,
        'a${_token.substring(1)}b',
        const TextSelection.collapsed(offset: 1),
      );
      expect(controller.text, 'ab');
      expect(controller.selection, const TextSelection.collapsed(offset: 1));
    });

    test('backspace removes a half-typed fragment in one step', () {
      final controller = _controller('hi [:sticker:smi');
      controller.selection = TextSelection.collapsed(
        offset: controller.text.length,
      );
      _edit(
        controller,
        'hi [:sticker:sm',
        TextSelection.collapsed(offset: controller.text.length - 1),
      );
      expect(controller.text, 'hi ');
    });

    test('clearing the field works with the caret anywhere', () {
      final controller = _controller('$_token hi');
      controller.selection = const TextSelection.collapsed(offset: 0);
      controller.clear();
      expect(controller.text, isEmpty);
    });

    test('ordinary text still deletes one character at a time', () {
      final controller = _controller('hello');
      controller.selection = const TextSelection.collapsed(offset: 5);
      _edit(controller, 'hell', const TextSelection.collapsed(offset: 4));
      expect(controller.text, 'hell');
    });
  });

  group('caret snapping', () {
    test('a caret moving into a token lands on its far side', () {
      final controller = _controller('a${_token}b');
      controller.selection = const TextSelection.collapsed(offset: 0);
      controller.selection = const TextSelection.collapsed(offset: 3);
      expect(controller.selection.baseOffset, 1 + _token.length);
    });

    test('stepping back from beyond a token lands before it', () {
      final controller = _controller('a${_token}b');
      final end = 1 + _token.length;
      controller.selection = TextSelection.collapsed(offset: end);
      controller.selection = TextSelection.collapsed(offset: end - 1);
      expect(controller.selection.baseOffset, 1);
    });

    test('selections widen to whole tokens', () {
      final controller = _controller('a${_token}b');
      controller.selection = const TextSelection(
        baseOffset: 0,
        extentOffset: 4,
      );
      expect(
        controller.selection,
        TextSelection(baseOffset: 0, extentOffset: 1 + _token.length),
      );
    });

    test('unresolved tokens behave as text', () {
      final controller = _controller('a[:sticker:unknown:]b');
      controller.selection = const TextSelection.collapsed(offset: 4);
      expect(controller.selection.baseOffset, 4);
    });
  });

  group('inline rendering', () {
    testWidgets('a resolved token draws as an image at 2.5x the font size', (
      tester,
    ) async {
      final controller = _controller('a${_token}b');
      await tester.pumpWidget(_field(controller));

      final image = find.byType(StickerImage);
      expect(image, findsOneWidget);
      expect(tester.widget<StickerImage>(image).size, 16 * 2.5);
      expect(controller.text, 'a${_token}b');

      // The caret after the token sits one image width past the caret
      // before it: the hidden token characters take no room.
      final editable = tester.renderObject<RenderEditable>(
        find.descendant(
          of: find.byType(EditableText),
          matching: find.byWidgetPredicate(
            (widget) => widget.runtimeType.toString() == '_Editable',
          ),
        ),
      );
      final before = editable.getLocalRectForCaret(
        const TextPosition(offset: 1),
      );
      final after = editable.getLocalRectForCaret(
        TextPosition(offset: 1 + _token.length),
      );
      expect(after.left - before.left, closeTo(16 * 2.5, 1.5));
    });

    testWidgets('tokens in code and unknown tokens stay text', (tester) async {
      final controller = _controller(
        '`$_token` [:sticker:unknown:]\n```\n$_token\n```',
      );
      await tester.pumpWidget(_field(controller));
      expect(find.byType(StickerImage), findsNothing);
    });

    testWidgets('deleting the token removes the image', (tester) async {
      final controller = _controller('a$_token');
      await tester.pumpWidget(_field(controller));
      expect(find.byType(StickerImage), findsOneWidget);

      await tester.tap(find.byType(TextField));
      controller.selection = TextSelection.collapsed(
        offset: controller.text.length,
      );
      await tester.pump();
      await tester.sendKeyEvent(LogicalKeyboardKey.backspace);
      await tester.pump();
      expect(controller.text, 'a');
      expect(find.byType(StickerImage), findsNothing);
    });
  });

  group('line layout', () {
    const lines = 'aa$_token\nbb$_token cc\ndd';

    List<Rect> stickerRects(WidgetTester tester) => [
      for (final element in find.byType(StickerImage).evaluate())
        tester.getRect(find.byWidget(element.widget)),
    ];

    Rect caretRect(WidgetTester tester, int offset) {
      final editable = tester.renderObject<RenderEditable>(
        find.descendant(
          of: find.byType(EditableText),
          matching: find.byWidgetPredicate(
            (widget) => widget.runtimeType.toString() == '_Editable',
          ),
        ),
      );
      return editable
          .getLocalRectForCaret(TextPosition(offset: offset))
          .shift(editable.localToGlobal(Offset.zero));
    }

    for (final scale in [1.0, 1.3]) {
      testWidgets('sticker lines grow instead of overlapping at $scale', (
        tester,
      ) async {
        final controller = _controller(lines);
        await tester.pumpWidget(
          MediaQuery(
            data: MediaQueryData(textScaler: TextScaler.linear(scale)),
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              home: Scaffold(
                body: Align(
                  alignment: Alignment.bottomCenter,
                  child: GfPostComposer(
                    controller: controller,
                    onPublish: () {},
                    publishLabel: 'Send',
                    hintText: 'Reply',
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();

        final rects = stickerRects(tester);
        expect(rects, hasLength(2));
        // Each image keeps its own line: no overlap with the next line's
        // image or with the plain text line after it.
        expect(rects[1].top, greaterThanOrEqualTo(rects[0].bottom - 0.5));
        final lastLine = caretRect(tester, lines.length);
        expect(lastLine.top, greaterThanOrEqualTo(rects[1].bottom - 0.5));
        // Plain lines keep the text height.
        expect(lastLine.height, closeTo(16 * scale * 1.4, 1));
      });
    }

    testWidgets('the compact composer shows a whole sticker line', (
      tester,
    ) async {
      final controller = _controller('aa$_token');
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: Align(
              alignment: Alignment.bottomCenter,
              child: GfPostComposer(
                controller: controller,
                compactEditor: true,
                onPublish: () {},
                publishLabel: 'Send',
                hintText: 'Reply',
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final sticker = stickerRects(tester).single;
      final field = tester.getRect(find.byType(EditableText));
      expect(field.top, lessThanOrEqualTo(sticker.top + 0.5));
      expect(field.bottom, greaterThanOrEqualTo(sticker.bottom - 0.5));
    });

    testWidgets('chat keeps a whole sticker line while its panel is open', (
      tester,
    ) async {
      final controller = _controller('aa$_token');
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: Align(
              alignment: Alignment.bottomCenter,
              child: GfChatInput(
                controller: controller,
                onSend: (_) {},
                emojiLabel: 'Stickers',
                accessoryBuilder: (_) => const SizedBox(height: 240),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byTooltip('Stickers'));
      await tester.pumpAndSettle();
      final sticker = stickerRects(tester).single;
      final field = tester.getRect(find.byType(EditableText));
      expect(field.top, lessThanOrEqualTo(sticker.top + 0.5));
      expect(field.bottom, greaterThanOrEqualTo(sticker.bottom - 0.5));
    });

    test('plain text keeps the default line strut', () {
      final controller = _controller('hello');
      expect(controller.inlineExtentScale, isNull);
      expect(GfInlineContentController.lineScale(controller, 1.4), 1.4);
      controller.text = 'hi $_token';
      expect(controller.inlineExtentScale, StickerImage.inlineScale);
      expect(
        GfInlineContentController.lineScale(controller, 1.4),
        StickerImage.inlineScale,
      );
      expect(
        GfInlineContentController.strutFor(TextEditingController()),
        isNull,
      );
    });
  });

  testWidgets('typing a fragment offers candidates that complete it', (
    tester,
  ) async {
    final library = _library();
    final repository = _SuggestionRepository(_client());
    final controller = StickerTextEditingController()..library = library;
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          stickerLibraryProvider.overrideWithValue(library),
          stickerCollectionProvider.overrideWith(
            (ref) => StickerCollection(repository, library),
          ),
        ],
        child: _field(
          controller,
          above: StickerSuggestionBar.text(controller: controller),
        ),
      ),
    );
    expect(find.byKey(const Key('sticker-suggestion-bar')), findsNothing);

    await tester.enterText(find.byType(TextField), 'hi [:sticker:sm');
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('sticker-suggestion-bar')), findsOneWidget);

    await tester.tap(find.byKey(const Key('sticker-suggestion-smile')));
    await tester.pumpAndSettle();
    expect(controller.text, 'hi $_token');
    expect(controller.selection.baseOffset, controller.text.length);
    expect(find.byKey(const Key('sticker-suggestion-bar')), findsNothing);
    expect(
      find.descendant(
        of: find.byType(TextField),
        matching: find.byType(StickerImage),
      ),
      findsOneWidget,
    );
  });
}
