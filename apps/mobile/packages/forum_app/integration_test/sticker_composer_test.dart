import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_picker.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import '../test/fixtures/page_fixtures.dart';
import '../test/pages_behavior_test.dart'
    show PollingChatRepository, makeChatMessage;
import '../test/pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

class _Pages extends FakePageRepository {
  _Pages(super.client);
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (!path.startsWith('/publish')) {
      return super.fetch(path, cancelToken: cancelToken);
    }
    return PagePayload.fromJson({
      'component': PageComponent.publish,
      'props': {
        'topicId': 0,
        'isEditing': false,
        'categories': [
          {'id': 1, 'name': '演示分类', 'color': '#2563eb'},
        ],
        'topic': {
          'title': '',
          'content': '',
          'categoryIds': [],
          'topicStatus': 0,
        },
      },
      'meta': {'title': '表情输入演示'},
      'layout': minimalLayoutJson(),
      'url': '/publish',
      'version': '1',
    });
  }
}

class _Stickers extends StickerRepository {
  _Stickers(super.client, this.url);
  final String url;
  @override
  Future<List<StickerItemPayload>> list() async => [
    StickerItemPayload(name: 'demo', url: url, displayName: '演示表情'),
  ];
  @override
  Future<List<StickerItemPayload>> mine() async => [];
  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async => [];
}

/// Synthetic drafts and a local image server; no account or remote writes.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'native sticker input stays above its panel and previews images',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final bytes = (await rootBundle.load(
        'assets/launcher/icon.png',
      )).buffer.asUint8List();
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      server.listen((request) async {
        request.response.headers.contentType = ContentType('image', 'png');
        request.response.add(bytes);
        await request.response.close();
      });
      addTearDown(() => server.close(force: true));
      final storage = MemoryTokenStorage();
      await storage.write('synthetic-session');
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: storage,
        baseUrl: 'http://127.0.0.1:${server.port}',
      );
      final stickers = _Stickers(client, '${client.baseUrl}/demo.png');
      final library = StickerLibrary(stickers);
      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(client),
          tokenStorageProvider.overrideWithValue(storage),
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: '演示账号'),
          ),
          pageRepositoryProvider.overrideWithValue(_Pages(client)),
          chatRepositoryProvider.overrideWithValue(
            PollingChatRepository(
              client,
              messages: List.generate(
                60,
                (i) => makeChatMessage(i + 1).copyWith(
                  content: '演示消息 ${i + 1}${'\n用于验证阅读位置的内容' * (i % 3)}',
                ),
              ),
            ),
          ),
          topicRepositoryProvider.overrideWithValue(
            FakeTopicRepository(client),
          ),
          offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
          offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
          stickerLibraryProvider.overrideWithValue(library),
          stickerCollectionProvider.overrideWith(
            (ref) => StickerCollection(stickers, library),
          ),
        ],
      );
      addTearDown(container.dispose);
      addTearDown(library.dispose);
      Future<void> show(Widget page) async {
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              locale: const Locale('zh'),
              home: page,
            ),
          ),
        );
        await tester.pumpAndSettle();
      }

      for (final surface in [
        'reply',
        'moment',
        'article',
        'chat',
        'chat-history',
      ]) {
        await show(switch (surface) {
          'reply' => const TopicPage(topicId: 100),
          'moment' => const PublishPage(initialContentType: 2),
          'article' => const PublishPage(initialContentType: 3),
          _ => const MessagesPage(targetUserId: 2),
        });
        void Function()? checkChatPosition;
        if (surface.startsWith('chat')) {
          final list = find.byType(ListView).last;
          final scroll = tester.widget<ListView>(list).controller!;
          if (surface == 'chat-history') {
            scroll.jumpTo(scroll.position.maxScrollExtent / 2);
            await tester.pumpAndSettle();
          }
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
          final gap = viewport.bottom - tester.getRect(anchor).bottom;
          checkChatPosition = () {
            expect(
              tester.getRect(anchor).bottom,
              closeTo(tester.getRect(list).bottom - gap, 1),
            );
            expect(anchor.hitTestable(), findsOneWidget);
            if (surface == 'chat') {
              expect(scroll.position.extentAfter, closeTo(0, 1));
            }
          };
        }
        if (surface == 'reply') {
          await tester.tap(find.byTooltip('参与讨论'));
          await tester.pumpAndSettle();
        }
        await tester.tap(
          find.byTooltip(surface.startsWith('chat') ? '表情' : '表情库'),
        );
        await tester.pumpAndSettle();
        checkChatPosition?.call();
        await tester.tap(find.text('演示表情').last);
        await tester.pumpAndSettle();
        expect(
          find.byType(StickerPicker),
          findsOneWidget,
          reason: '$surface keeps the picker open after the first insertion',
        );
        checkChatPosition?.call();
        await tester.tap(find.text('演示表情').last);
        await tester.pumpAndSettle();
        expect(
          find.byType(StickerPicker),
          findsOneWidget,
          reason: '$surface keeps the picker open after repeated insertions',
        );
        final preview = find.byKey(const Key('sticker-draft-preview'));
        expect(
          find.descendant(of: preview, matching: find.byType(StickerImage)),
          findsNWidgets(2),
        );
        expect(find.byType(BottomSheet), findsNothing);
        expect(tester.takeException(), isNull);
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot('sticker-$surface');
        }
        await tester.tap(find.byTooltip('键盘'));
        await tester.pumpAndSettle();
        expect(find.byType(StickerPicker), findsNothing);
        checkChatPosition?.call();
        if (surface.startsWith('chat')) {
          tester
              .widget<GfChatInput>(find.byType(GfChatInput))
              .controller!
              .clear();
          FocusManager.instance.primaryFocus?.unfocus();
          await tester.pumpAndSettle();
          checkChatPosition?.call();
        }
      }
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
}
