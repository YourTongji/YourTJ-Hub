import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/format.dart';
import 'package:forum_app/src/messages/forwarded_message.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'chat_visible_read_test.dart' show pumpChat;
import 'fixtures/page_fixtures.dart' show messagesPayloadJson, parsePayload;
import 'pages_behavior_test.dart' show CountingPageRepository, MemTokenStorage;

class _ForwardPreviewRepository extends CountingPageRepository {
  _ForwardPreviewRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()));
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path != '/messages') return super.fetch(path, cancelToken: cancelToken);
    final payload = messagesPayloadJson();
    final conversations = (payload['props'] as Map)['conversations'] as List;
    (conversations.first as Map)['lastMsg'] =
        '[Chat history]\nBob: [:sticker:smile:]';
    return parsePayload(payload);
  }
}

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'outgoing history text and chevron inherit white ($brightness)',
      (tester) async {
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: gfThemeData(brightness),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const Scaffold(
                body: GfMessageBubble(
                  text: '[Chat history]',
                  mine: true,
                  content: ForwardedMessageCard(
                    bundle: ChatForwardBundle(
                      version: 1,
                      messages: [
                        ChatForwardEntry(
                          senderName: 'Bob',
                          content: 'Preview',
                          createdAt: '2026-09-28T09:00:00Z',
                          msgType: 1,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        final texts = tester.widgetList<RichText>(
          find.descendant(
            of: find.byType(ForwardedMessageCard),
            matching: find.byType(RichText),
          ),
        );
        expect(texts.length, 3, reason: 'title, preview and count');
        for (final text in texts) {
          expect((text.text as TextSpan).style!.color, Colors.white);
        }
        expect(
          tester.widget<GfSymbol>(find.byType(GfSymbol)).color,
          Colors.white,
        );
      },
    );
  }

  testWidgets('nested history keeps each sender avatar and opens as a card', (
    tester,
  ) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    const bundle = ChatForwardBundle(
      version: 1,
      messages: [
        ChatForwardEntry(
          senderName: 'Forwarder',
          avatarUrl: '/static/pic/6.webp',
          content: '[Chat history]\nAlice: first\nBob: second',
          createdAt: '2026-09-28T10:00:00Z',
          msgType: 4,
          forwarded: ChatForwardBundle(
            version: 1,
            messages: [
              ChatForwardEntry(
                senderName: 'Alice',
                avatarUrl: '/static/pic/3.webp',
                content: 'first',
                createdAt: '2026-09-28T09:00:00Z',
                msgType: 1,
              ),
              ChatForwardEntry(
                senderName: 'Bob',
                avatarUrl: '/static/pic/4.webp',
                content: 'second',
                createdAt: '2026-09-28T09:01:00Z',
                msgType: 1,
              ),
            ],
          ),
        ),
      ],
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ForwardedMessagesPage(bundle: bundle, ownerEpoch: 0),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(ForwardedMessageCard), findsOneWidget);
    expect(
      tester.widget<GfAvatar>(find.byType(GfAvatar)).src,
      endsWith('/static/pic/6.webp'),
    );
    await tester.tap(find.byType(ForwardedMessageCard));
    await tester.pumpAndSettle();
    expect(find.text('first'), findsOneWidget);
    expect(find.text('second'), findsOneWidget);
    expect(
      tester.widgetList<GfAvatar>(find.byType(GfAvatar)).map((a) => a.src),
      [endsWith('/static/pic/3.webp'), endsWith('/static/pic/4.webp')],
    );
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.text('Forwarder'), findsOneWidget);
    expect(find.byType(ForwardedMessageCard), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'conversation preview includes the forwarded content on one line',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});
      await pumpChat(
        tester,
        targetUserId: null,
        overrides: [
          pageRepositoryProvider.overrideWithValue(_ForwardPreviewRepository()),
        ],
      );
      final row = tester.widget<GfConversationRow>(
        find.byType(GfConversationRow).first,
      );
      expect(row.lastMessage, '[Chat history] Bob: [Animated sticker]');
    },
  );

  testWidgets(
    'snapshot detail shows a single full timestamp and hides on account change',
    (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      const time = '2026-09-28T01:00:00Z';
      const bundle = ChatForwardBundle(
        version: 1,
        messages: [
          ChatForwardEntry(
            senderName: 'Alice',
            content: 'copied private body',
            createdAt: time,
            msgType: 1,
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ForwardedMessagesPage(bundle: bundle, ownerEpoch: 0),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text(formatDateTime(time)), findsOneWidget);
      expect(find.text('copied private body'), findsOneWidget);
      expect(find.byType(GfAvatar), findsOneWidget);
      expect(find.byType(GfMessageBubble), findsOneWidget);
      expect(find.byType(GfChatInput), findsNothing);
      expect(
        tester.getTopLeft(find.byType(GfAvatar)).dx,
        lessThan(tester.getTopLeft(find.text('copied private body')).dx),
      );
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      expect(find.text('copied private body'), findsNothing);
      expect(find.text('Alice'), findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('history entries expose only a one-entry copy action', (
    tester,
  ) async {
    final calls = <MethodCall>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        calls.add(call);
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
      ProviderScope(
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ForwardedMessagesPage(
            ownerEpoch: 0,
            bundle: ChatForwardBundle(
              version: 1,
              messages: [
                ChatForwardEntry(
                  senderName: 'Alice',
                  content: 'single entry body',
                  createdAt: '2026-09-28T01:00:00Z',
                  msgType: 1,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.longPress(find.text('single entry body'));
    await tester.pumpAndSettle();
    expect(find.text('Copy entire message'), findsOneWidget);
    expect(find.text('Reply'), findsNothing);
    expect(find.text('Report message'), findsNothing);
    await tester.tap(find.text('Copy entire message'));
    await tester.pumpAndSettle();
    final copy = calls.singleWhere(
      (call) => call.method == 'Clipboard.setData',
    );
    expect(
      (copy.arguments as Map<Object?, Object?>)['text'],
      'single entry body',
    );
    await tester.pumpWidget(const SizedBox());
  });
}
