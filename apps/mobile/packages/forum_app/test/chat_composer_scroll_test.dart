import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'chat_visible_read_test.dart' show pumpChat;
import 'fixtures/sticker_fixtures.dart';
import 'pages_behavior_test.dart' show MemTokenStorage, makeChatMessage;

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  testWidgets('a short conversation keeps its last message above the panel', (
    tester,
  ) async {
    await pumpChat(
      tester,
      messages: List.generate(4, (i) => makeChatMessage(i + 1)),
    );
    final list = find.byType(ListView).last;
    final scroll = tester.widget<ListView>(list).controller!;
    expect(scroll.position.maxScrollExtent, 0);
    await tester.tap(find.byTooltip('Emoji'));
    await tester.pumpAndSettle();
    expect(
      tester.getRect(find.text('消息 4')).bottom,
      lessThan(tester.getRect(list).bottom),
    );
    expect(scroll.position.extentAfter, closeTo(0, 1));
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(scroll.offset, 0);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });

  for (final readingHistory in [false, true]) {
    testWidgets(
      'composer resize preserves visible messages, history=$readingHistory',
      (tester) async {
        final repository = ComposerStickerRepository(
          GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()),
        );
        final library = StickerLibrary(repository);
        addTearDown(library.dispose);
        await pumpChat(
          tester,
          stickers: library,
          stickerCollection: StickerCollection(repository, library),
          messages: List.generate(
            60,
            (i) => makeChatMessage(
              i + 1,
            ).copyWith(content: '消息 ${i + 1}${'\n不同高度的历史内容' * (i % 3)}'),
          ),
        );
        final list = find.byType(ListView).last;
        final scroll = tester.widget<ListView>(list).controller!;
        if (readingHistory) {
          scroll.jumpTo(scroll.position.maxScrollExtent / 2);
          await tester.pumpAndSettle();
        }
        final originalOffset = scroll.offset;
        final viewport = tester.getRect(list);
        final visible = tester
            .widgetList<GfMessageBubble>(find.byType(GfMessageBubble))
            .where((bubble) {
              final bounds = tester.getRect(find.byKey(bubble.bubbleKey!));
              return bounds.top >= viewport.top &&
                  bounds.bottom <= viewport.bottom;
            })
            .toList();
        final anchor = find.byKey(visible.last.bubbleKey!);
        final bottomGap = viewport.bottom - tester.getRect(anchor).bottom;

        void expectAnchor() {
          final bounds = tester.getRect(anchor);
          final viewport = tester.getRect(list);
          expect(bounds.bottom, closeTo(viewport.bottom - bottomGap, 1));
          expect(bounds.top, greaterThanOrEqualTo(viewport.top));
          if (!readingHistory) {
            expect(scroll.position.extentAfter, closeTo(0, 1));
          }
          expect(tester.takeException(), isNull);
        }

        await tester.tap(find.byTooltip('Emoji'));
        await tester.pumpAndSettle();
        expectAnchor();

        // Draft previews also consume list height without a window-metrics event.
        await tester.tap(find.text('Smile').last);
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('sticker-draft-preview')), findsOneWidget);
        expectAnchor();

        // Switching back to the keyboard must preserve the same history anchor.
        await tester.tap(find.byTooltip('Keyboard'));
        tester.view.viewInsets = FakeViewPadding(
          bottom: 250 * tester.view.devicePixelRatio,
        );
        addTearDown(tester.view.resetViewInsets);
        await tester.pumpAndSettle();
        expectAnchor();

        tester
            .widget<GfChatInput>(find.byType(GfChatInput))
            .controller!
            .clear();
        FocusManager.instance.primaryFocus?.unfocus();
        tester.view.resetViewInsets();
        await tester.pumpAndSettle();
        expectAnchor();
        expect(scroll.offset, closeTo(originalOffset, 1));
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(seconds: 1));
      },
    );
  }
}
