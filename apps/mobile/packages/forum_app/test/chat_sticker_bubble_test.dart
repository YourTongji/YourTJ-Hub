import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/src/messages/chat_outbox.dart';
import 'package:forum_app/l10n/app_localizations.dart';

import 'chat_visible_read_test.dart' show pumpChat;
import 'pages_behavior_test.dart' show MemTokenStorage, makeChatMessage;

const _smile = StickerItemPayload(name: 'smile', url: '/smile.png');

class _Stickers extends StickerRepository {
  _Stickers() : super(GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()));

  Completer<List<StickerItemPayload>>? pending;
  @override
  Future<List<StickerItemPayload>> list() async => pending?.future ?? [_smile];
  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async => [];
}

Container _surface(WidgetTester tester) {
  final bubble = tester.widget<GfMessageBubble>(find.byType(GfMessageBubble));
  expect(bubble.bubbleKey, isNotNull);
  expect(tester.getSize(find.byKey(bubble.bubbleKey!)).isEmpty, isFalse);
  expect(bubble.time, isNotEmpty);
  return tester.widget<Container>(find.byKey(bubble.bubbleKey!));
}

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  for (final mine in [false, true]) {
    for (final (text, plain) in [
      ('[:sticker:smile:]', true),
      (' \n[:sticker:smile:] [:sticker:smile:]\n ', true),
      ('Hello [:sticker:smile:]', false),
      ('[:sticker:unknown:]', false),
      ('[:sticker:smile:] [:sticker:unknown:]', false),
    ]) {
      testWidgets('chat sticker surface mine=$mine content=$text', (
        tester,
      ) async {
        final library = StickerLibrary(_Stickers());
        addTearDown(library.dispose);
        final chat = await pumpChat(
          tester,
          stickers: library,
          messages: [makeChatMessage(1).copyWith(content: text, isSelf: mine)],
        );
        final surface = _surface(tester);
        expect(surface.decoration == null, plain);
        expect(
          surface.padding ?? EdgeInsets.zero,
          plain
              ? EdgeInsets.zero
              : const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        );
        await tester.pump(const Duration(milliseconds: 400));
        await tester.pumpAndSettle();
        if (!mine) {
          expect(chat.repo.batches.expand((ids) => ids), contains(1));
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
      });
    }
  }

  testWidgets(
    'failed sticker outbox keeps its retry outside a frameless body',
    (tester) async {
      final library = StickerLibrary(_Stickers());
      addTearDown(library.dispose);
      final chat = await pumpChat(tester, stickers: library, messages: []);
      chat.container
          .read(chatOutboxProvider(2))
          .enqueue('[:sticker:smile:]', 0);
      await tester.pumpAndSettle();
      final bubble = tester.widget<GfMessageBubble>(
        find.byType(GfMessageBubble),
      );
      expect(bubble.showBubble, isFalse);
      expect(bubble.mine, isTrue);
      final l10n = AppLocalizations.of(
        tester.element(find.byType(GfMessageBubble)),
      );
      expect(find.text(l10n.messagesFailed), findsOneWidget);
      expect(find.text(l10n.messagesRetry), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'resolution removes the bubble without replacing its visibility key',
    (tester) async {
      final repository = _Stickers()..pending = Completer();
      final library = StickerLibrary(repository);
      addTearDown(library.dispose);
      await pumpChat(
        tester,
        stickers: library,
        messages: [makeChatMessage(1).copyWith(content: '[:sticker:smile:]')],
      );
      expect(_surface(tester).decoration, isNotNull);
      final key = tester
          .widget<GfMessageBubble>(find.byType(GfMessageBubble))
          .bubbleKey;
      repository.pending!.complete([_smile]);
      await tester.pumpAndSettle();
      expect(_surface(tester).decoration, isNull);
      expect(
        tester.widget<GfMessageBubble>(find.byType(GfMessageBubble)).bubbleKey,
        same(key),
      );
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
