import 'dart:async';
import 'dart:io';
import 'dart:ui' show Tristate;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/stickers/sticker_image.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:ui_kit/ui_kit.dart';

import 'chat_visible_read_test.dart' show pumpChat, VisibleChatRepository;
import 'fixtures/page_fixtures.dart' show messagesPayloadJson, parsePayload;
import 'pages_behavior_test.dart' show CountingPageRepository, makeChatMessage;
import 'user_safety_test.dart' show Blocks;

class Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

GfApiClient _client() => GfApiClient(dio: Dio(), tokenStorage: Tokens());

class Reports extends PostRepository {
  Reports() : super(_client());
  final sent = <Map<String, Object>>[];
  @override
  Future<bool> report({
    required String targetType,
    required int targetId,
    required String reason,
    required String note,
  }) async {
    sent.add({
      'targetType': targetType,
      'targetId': targetId,
      'reason': reason,
      'note': note,
    });
    return true;
  }
}

class RecordingSendsChatRepository extends VisibleChatRepository {
  RecordingSendsChatRepository(super.client, {required super.messages});

  final sent = <(int, String)>[];
  final sentTypes = <int>[];
  final clientKeys = <String?>[];
  final replyTargets = <int?>[];
  int? requestedAroundId;
  List<ChatMessagePayload>? aroundMessages;
  bool aroundHasMoreAfter = false;
  Completer<ChatMessagesResponse>? newerResponse;
  int failures = 0;

  @override
  Future<ChatMessagesResponse> getMessages({
    required int convId,
    int beforeId = 0,
    int afterId = 0,
    int aroundId = 0,
    int limit = 30,
    Object? cancelToken,
  }) {
    if (aroundId > 0) {
      requestedAroundId = aroundId;
      return Future.value(
        ChatMessagesResponse(
          list: aroundMessages ?? [makeChatMessage(aroundId)],
          hasMoreBefore: false,
          hasMoreAfter: aroundHasMoreAfter,
          nextBeforeId: aroundId,
          latestId: aroundMessages?.last.id ?? aroundId,
        ),
      );
    }
    if (afterId > 0 && newerResponse != null) return newerResponse!.future;
    return super.getMessages(
      convId: convId,
      beforeId: beforeId,
      afterId: afterId,
      aroundId: aroundId,
      limit: limit,
      cancelToken: cancelToken,
    );
  }

  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 1,
    String? clientMessageId,
    int? replyToMessageId,
  }) async {
    sent.add((peerId, content));
    sentTypes.add(msgType);
    clientKeys.add(clientMessageId);
    if (failures > 0) {
      failures--;
      throw StateError('offline');
    }
    replyTargets.add(replyToMessageId);
    return 9;
  }
}

class _DelayedReplyRepository extends RecordingSendsChatRepository {
  _DelayedReplyRepository(super.client)
    : super(messages: [makeChatMessage(1), makeChatMessage(2)]);

  final result = Completer<int>();

  @override
  Future<int> sendMessage({
    required int peerId,
    required String content,
    int msgType = 1,
    String? clientMessageId,
    int? replyToMessageId,
  }) => result.future;
}

/// 1x1 transparent PNG served to `Image.network` so sticker widgets reach their
/// loaded state instead of the failed-image retry surface.
const List<int> _transparentPng = <int>[
  0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, //
  0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
  0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
  0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
  0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
  0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
  0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
  0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
  0x42, 0x60, 0x82,
];

class _ImageHttpClient implements HttpClient {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);

  @override
  Future<HttpClientRequest> getUrl(Uri url) async => _ImageHttpRequest();
}

class _ImageHttpRequest implements HttpClientRequest {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);

  @override
  final HttpHeaders headers = _ImageHttpHeaders();

  @override
  Future<HttpClientResponse> close() async => _ImageHttpResponse();
}

class _ImageHttpHeaders implements HttpHeaders {
  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}

class _ImageHttpResponse implements HttpClientResponse {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);

  @override
  int get statusCode => HttpStatus.ok;

  @override
  int get contentLength => _transparentPng.length;

  @override
  HttpClientResponseCompressionState get compressionState =>
      HttpClientResponseCompressionState.notCompressed;

  @override
  StreamSubscription<List<int>> listen(
    void Function(List<int> event)? onData, {
    Function? onError,
    void Function()? onDone,
    bool? cancelOnError,
  }) => Stream<List<int>>.value(_transparentPng).listen(
    onData,
    onError: onError,
    onDone: onDone,
    cancelOnError: cancelOnError,
  );
}

/// Page payload whose layout viewer has no username (partial/legacy payloads),
/// used to exercise the self-label fallback instead of a real handle.
class _EmptyViewerPageRepository extends CountingPageRepository {
  _EmptyViewerPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) {
    if (path != '/messages') return super.fetch(path, cancelToken: cancelToken);
    final json = messagesPayloadJson();
    final layout = json['layout'] as Map<String, dynamic>;
    layout['viewer'] = <String, dynamic>{
      ...layout['viewer'] as Map<String, dynamic>,
      'username': '',
    };
    return Future.value(parsePayload(json));
  }
}

class _StickerRepository extends StickerRepository {
  _StickerRepository() : super(GfApiClient(dio: Dio(), tokenStorage: Tokens()));

  final saved = <String>[];

  @override
  Future<List<StickerItemPayload>> list() async => const [
    StickerItemPayload(name: 'smile', url: '/smile.png'),
  ];

  @override
  Future<List<StickerItemPayload>> resolve(List<String> names) async =>
      const <StickerItemPayload>[];

  @override
  Future<StickerItemPayload> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) async {
    saved.add(stickerName ?? '');
    return StickerItemPayload(name: stickerName ?? '', url: '/smile.png');
  }
}

Future<RecordingSendsChatRepository> pumpActions(
  WidgetTester tester, {
  List<ChatMessagePayload>? messages,
  Reports? reports,
  StickerLibrary? stickers,
  StickerCollection? stickerCollection,
  RecordingSendsChatRepository Function(GfApiClient client)? repositoryBuilder,
  List<Override> overrides = const <Override>[],
}) async {
  late final RecordingSendsChatRepository repository;
  await pumpChat(
    tester,
    repository: (client) => repository =
        repositoryBuilder?.call(client) ??
        RecordingSendsChatRepository(
          client,
          messages: messages ?? <ChatMessagePayload>[makeChatMessage(1)],
        ),
    stickers: stickers,
    stickerCollection: stickerCollection,
    overrides: [
      postRepositoryProvider.overrideWithValue(reports ?? Reports()),
      ...overrides,
    ],
  );
  return repository;
}

Future<void> openActions(WidgetTester tester, Finder bubble) async {
  await tester.longPress(bubble);
  await tester.pumpAndSettle();
}

final Finder _preview = find.byKey(const Key('chat-reply-preview'));

void main() {
  testWidgets('chat more menu owns block and unblock confirmation', (
    tester,
  ) async {
    final blocks = Blocks();
    await pumpChat(
      tester,
      messages: [makeChatMessage(1)],
      overrides: [userRepositoryProvider.overrideWithValue(blocks)],
    );
    expect(find.byTooltip('Block user'), findsNothing);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    expect(blocks.writes, isEmpty);
    await tester.tap(find.text('Block user'));
    await tester.pumpAndSettle();
    expect(blocks.writes, isEmpty);
    await tester.tap(find.widgetWithText(FilledButton, 'Block user'));
    await tester.pumpAndSettle();
    expect(blocks.writes, [true]);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Unblock user'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Unblock user'));
    await tester.pumpAndSettle();
    expect(blocks.writes, [true, false]);
  });

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  Future<void> dispose(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  }

  testWidgets('long-press on a peer message opens the action menu', (
    tester,
  ) async {
    await pumpActions(tester);
    expect(find.text('Reply'), findsNothing);
    expect(find.text('Copy entire message'), findsNothing);
    await openActions(tester, find.text('消息 1'));
    expect(find.text('Reply'), findsOneWidget);
    expect(find.text('Copy entire message'), findsOneWidget);
    expect(find.text('Report message'), findsOneWidget);
    expect(find.text('Save to my stickers'), findsNothing);
    expect(tester.takeException(), isNull);
    await dispose(tester);
  });

  testWidgets('own messages offer reply and copy without report', (
    tester,
  ) async {
    await pumpActions(
      tester,
      messages: [makeChatMessage(1).copyWith(isSelf: true)],
    );
    await openActions(tester, find.text('消息 1'));
    expect(find.text('Reply'), findsOneWidget);
    expect(find.text('Copy entire message'), findsOneWidget);
    expect(find.text('Report message'), findsNothing);
    await dispose(tester);
  });

  testWidgets(
    'forwarded history can be replied to but not copied or collected',
    (tester) async {
      final repository = await pumpActions(
        tester,
        messages: [
          makeChatMessage(
            1,
          ).copyWith(msgType: 4, content: '[Chat history]\nprivate snapshot'),
        ],
      );
      await openActions(tester, find.textContaining('Chat history'));
      expect(find.text('Reply'), findsOneWidget);
      expect(find.text('Forward'), findsOneWidget);
      expect(find.text('Select messages'), findsOneWidget);
      expect(find.text('Report message'), findsOneWidget);
      expect(find.text('Copy entire message'), findsNothing);
      expect(find.text('Save to my stickers'), findsNothing);
      await tester.tap(find.text('Reply'));
      await tester.pumpAndSettle();
      expect(
        find.descendant(of: _preview, matching: find.text('[Chat history]')),
        findsOneWidget,
      );
      await tester.enterText(
        find.descendant(
          of: find.byType(GfChatInput),
          matching: find.byType(TextField),
        ),
        '收到',
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('chat-send')));
      await tester.pumpAndSettle();
      expect(repository.sent, [(2, '> @bob: [Chat history]\n\n收到')]);
      await dispose(tester);
    },
  );

  testWidgets('the inline report link is gone and reporting keeps its target', (
    tester,
  ) async {
    final reports = Reports();
    await pumpActions(tester, reports: reports);
    expect(find.widgetWithText(TextButton, 'Report message'), findsNothing);

    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Report message'));
    await tester.pumpAndSettle();
    expect(find.textContaining('rest of the conversation'), findsOneWidget);
    await tester.enterText(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(TextField),
      ),
      '这条消息持续骚扰我',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
    await tester.pumpAndSettle();
    expect(reports.sent, [
      {
        'targetType': 'chat_message',
        'targetId': 1,
        'reason': 'abuse',
        'note': '这条消息持续骚扰我',
      },
    ]);
    await dispose(tester);
  });

  testWidgets(
    'left swipe replies once and focuses the unchanged draft',
    (tester) async {
      await pumpActions(tester);
      final input = find.descendant(
        of: find.byType(GfChatInput),
        matching: find.byType(TextField),
      );
      await tester.enterText(input, 'draft to preserve');
      await tester.pumpAndSettle();
      await tester.drag(find.text('消息 1'), const Offset(-28, 0));
      await tester.pumpAndSettle();
      expect(_preview, findsNothing);
      await tester.drag(find.text('消息 1'), const Offset(95, 0));
      await tester.pumpAndSettle();
      expect(_preview, findsNothing);
      final gesture = await tester.startGesture(
        tester.getCenter(find.text('消息 1')),
      );
      // iOS may coalesce a fast swipe into a single move before pointer-up.
      await gesture.moveBy(const Offset(-100, 0));
      await gesture.up();
      await tester.pumpAndSettle();
      expect(_preview, findsOneWidget);
      expect(
        tester.widget<TextField>(input).controller!.text,
        'draft to preserve',
      );
      expect(tester.widget<TextField>(input).focusNode!.hasFocus, isTrue);
      await dispose(tester);
    },
    variant: TargetPlatformVariant({
      TargetPlatform.iOS,
      TargetPlatform.android,
    }),
  );

  testWidgets('multi-select cancels with system back and preserves the draft', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await pumpActions(
      tester,
      messages: [makeChatMessage(1), makeChatMessage(2)],
    );
    final input = find.descendant(
      of: find.byType(GfChatInput),
      matching: find.byType(TextField),
    );
    await tester.enterText(input, 'draft survives selection');
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Select messages'));
    await tester.pumpAndSettle();
    expect(find.byType(Checkbox), findsNWidgets(2));
    expect(
      tester
          .widgetList<Checkbox>(find.byType(Checkbox))
          .every((checkbox) => checkbox.shape is CircleBorder),
      isTrue,
    );
    expect(find.byType(GfChatInput), findsNothing);
    final unselectedRow = find.bySemanticsLabel('@bob: 消息 2');
    expect(
      tester.getSemantics(unselectedRow).flagsCollection.isEnabled,
      Tristate.isTrue,
    );
    semantics.dispose();
    await tester.tap(find.text('消息 2'));
    await tester.pump();
    expect(
      tester.widgetList<Checkbox>(find.byType(Checkbox)).every((c) => c.value!),
      isTrue,
    );
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.byType(Checkbox), findsNothing);
    expect(
      tester.widget<TextField>(input).controller!.text,
      'draft survives selection',
    );
    await dispose(tester);
  });

  testWidgets(
    'recipient selection fits narrow screens with a search keyboard and large text',
    (tester) async {
      await pumpActions(tester);
      await openActions(tester, find.text('消息 1'));
      await tester.tap(find.text('Forward'));
      await tester.pumpAndSettle();
      expect(find.byType(CheckboxListTile), findsWidgets);
      expect(
        tester
            .widgetList<CheckboxListTile>(find.byType(CheckboxListTile))
            .every((tile) => tile.checkboxShape is CircleBorder),
        isTrue,
      );
      await tester.binding.setSurfaceSize(const Size(320, 560));
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      addTearDown(tester.view.resetViewInsets);
      tester.view.viewInsets = const FakeViewPadding(bottom: 240);
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      expect(find.byType(FilledButton).last.hitTestable(), findsOneWidget);
      tester.view.resetViewInsets();
      await dispose(tester);
    },
  );

  testWidgets(
    'quote return disappears after manually scrolling to the bottom',
    (tester) async {
      await pumpActions(
        tester,
        messages: [
          for (var id = 1; id < 40; id++) makeChatMessage(id),
          makeChatMessage(40).copyWith(
            content: '> @bob: first message\n\nsource',
            replyToMessageId: 1,
          ),
        ],
      );
      await tester.tap(find.text('first message'));
      await tester.pumpAndSettle();
      final returnButton = find.byKey(const Key('chat-reply-return'));
      expect(returnButton, findsOneWidget);
      final list = find.byType(ListView).last;
      await tester.drag(list, const Offset(0, -250));
      await tester.pumpAndSettle();
      expect(returnButton, findsOneWidget);

      await tester.drag(list, const Offset(0, -3000));
      await tester.pumpAndSettle();
      expect(find.text('source').hitTestable(), findsOneWidget);
      expect(tester.widget<ListView>(list).controller!.position.extentAfter, 0);
      expect(returnButton, findsNothing);
      await tester.drag(list, const Offset(0, 200));
      await tester.pumpAndSettle();
      expect(returnButton, findsNothing);
      await dispose(tester);
    },
  );

  testWidgets('quote return remains at a page boundary until the real bottom', (
    tester,
  ) async {
    final source = makeChatMessage(
      60,
    ).copyWith(content: '> @bob: old target\n\nsource', replyToMessageId: 15);
    final repository = await pumpActions(tester, messages: [source]);
    repository.aroundMessages = [
      for (var id = 1; id <= 30; id++) makeChatMessage(id),
    ];
    repository.aroundHasMoreAfter = true;
    repository.newerResponse = Completer<ChatMessagesResponse>();
    await tester.tap(find.text('old target'));
    await tester.pumpAndSettle();
    final list = find.byType(ListView).last;
    final returnButton = find.byKey(const Key('chat-reply-return'));
    await tester.drag(list, const Offset(0, -3000));
    await tester.pumpAndSettle();
    expect(tester.widget<ListView>(list).controller!.position.extentAfter, 0);
    expect(returnButton, findsOneWidget);

    repository.newerResponse!.complete(
      ChatMessagesResponse(
        list: [for (var id = 31; id < 60; id++) makeChatMessage(id), source],
        hasMoreBefore: false,
        hasMoreAfter: false,
        nextBeforeId: 0,
        latestId: 60,
      ),
    );
    await tester.pumpAndSettle();
    expect(returnButton, findsOneWidget);
    await tester.drag(list, const Offset(0, -3000));
    await tester.pumpAndSettle();
    expect(find.text('source').hitTestable(), findsOneWidget);
    expect(returnButton, findsNothing);
    await dispose(tester);
  });

  for (final reducedMotion in [false, true]) {
    testWidgets(
      'selection rail can exit during reveal (reduced motion: $reducedMotion)',
      (tester) async {
        tester.platformDispatcher.accessibilityFeaturesTestValue =
            FakeAccessibilityFeatures(disableAnimations: reducedMotion);
        addTearDown(
          tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
        );
        await pumpActions(tester);
        final message = find.text('消息 1');
        final originalLeft = tester.getTopLeft(message).dx;
        await openActions(tester, message);
        await tester.tap(find.text('Select messages'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 60));
        final selectingLeft = tester.getTopLeft(message).dx;
        if (reducedMotion) {
          expect(selectingLeft, closeTo(originalLeft + 44, 0.01));
        } else {
          expect(selectingLeft, greaterThan(originalLeft));
          expect(selectingLeft, lessThan(originalLeft + 44));
        }
        await tester.binding.handlePopRoute();
        await tester.pumpAndSettle();
        expect(find.byType(Checkbox), findsNothing);
        expect(tester.getTopLeft(message).dx, closeTo(originalLeft, 0.01));
        expect(find.byType(GfChatInput), findsOneWidget);
        expect(tester.takeException(), isNull);
        await dispose(tester);
      },
    );
  }

  testWidgets('copying a message writes the whole content to the clipboard', (
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
    await pumpActions(tester);
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Copy entire message'));
    await tester.pumpAndSettle();
    final copy = calls.singleWhere(
      (call) => call.method == 'Clipboard.setData',
    );
    expect((copy.arguments as Map<Object?, Object?>)['text'], '消息 1');
    expect(find.text('Copied'), findsOneWidget);
    await dispose(tester);
  });

  testWidgets('choosing reply shows a dismissible quote preview', (
    tester,
  ) async {
    await pumpActions(tester);
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    expect(_preview, findsOneWidget);
    expect(
      find.descendant(of: _preview, matching: find.text('@bob')),
      findsOneWidget,
    );
    expect(
      find.descendant(of: _preview, matching: find.text('消息 1')),
      findsOneWidget,
    );
    expect(
      tester.getRect(_preview).bottom,
      lessThanOrEqualTo(
        tester.getRect(find.byKey(const Key('chat-input-surface'))).top,
      ),
    );
    await tester.tap(find.byTooltip('Cancel reply'));
    await tester.pumpAndSettle();
    expect(_preview, findsNothing);
    await dispose(tester);
  });

  testWidgets('sending a reply quotes the excerpt and clears the preview', (
    tester,
  ) async {
    final repository = await pumpActions(tester);
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.descendant(
        of: find.byType(GfChatInput),
        matching: find.byType(TextField),
      ),
      '收到',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(repository.sent, [(2, '> @bob: 消息 1\n\n收到')]);
    expect(repository.replyTargets, [1]);
    expect(_preview, findsNothing);
    await dispose(tester);
  });

  testWidgets(
    'quote tap loads an old target, highlights it, and returns to source',
    (tester) async {
      final repository = await pumpActions(
        tester,
        messages: [
          makeChatMessage(2).copyWith(
            content: '> @bob: older content\n\nanswer',
            replyToMessageId: 1,
          ),
        ],
      );

      await tester.tap(find.text('older content'));
      await tester.pumpAndSettle();
      expect(repository.requestedAroundId, 1);
      expect(find.byKey(const Key('chat-reply-return')), findsOneWidget);
      expect(find.text('消息 1'), findsOneWidget);

      await tester.tap(find.byKey(const Key('chat-reply-return')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('chat-reply-return')), findsNothing);
      expect(find.text('answer'), findsOneWidget);
      await dispose(tester);
    },
  );

  testWidgets(
    'quote reveals an unbuilt target inside a full variable-height window',
    (tester) async {
      final repository = await pumpActions(
        tester,
        messages: [
          for (var id = 100; id < 130; id++) makeChatMessage(id),
          makeChatMessage(130).copyWith(
            content: '> @bob: old target\n\nsource answer',
            replyToMessageId: 15,
          ),
        ],
      );
      repository.aroundMessages = [
        for (var id = 1; id <= 30; id++)
          makeChatMessage(id).copyWith(
            content: id == 15
                ? 'target body'
                : 'row $id\n${'long line\n' * (id % 5 + 1)}',
          ),
      ];
      final sourceY = tester.getCenter(find.text('source answer')).dy;
      await tester.tap(find.text('old target'));
      await tester.pumpAndSettle();
      expect(repository.requestedAroundId, 15);
      expect(find.text('target body').hitTestable(), findsOneWidget);
      await tester.tap(find.byKey(const Key('chat-reply-return')));
      await tester.pumpAndSettle();
      expect(find.text('source answer').hitTestable(), findsOneWidget);
      expect(
        tester.getCenter(find.text('source answer')).dy,
        closeTo(sourceY, 2),
      );
      await dispose(tester);
    },
  );

  testWidgets(
    'quote reveals a loaded offscreen target without another request',
    (tester) async {
      final repository = await pumpActions(
        tester,
        messages: [
          for (var id = 1; id < 40; id++) makeChatMessage(id),
          makeChatMessage(40).copyWith(
            content: '> @bob: first message\n\nsource',
            replyToMessageId: 1,
          ),
        ],
      );
      expect(find.text('消息 1'), findsNothing);
      repository.batches.clear();
      await tester.tap(find.text('first message'));
      await tester.pumpAndSettle();
      expect(repository.requestedAroundId, isNull);
      expect(find.text('消息 1').hitTestable(), findsOneWidget);
      expect(
        repository.batches
            .expand((ids) => ids)
            .any((id) => id >= 15 && id <= 25),
        isFalse,
        reason: 'rows traversed while seeking are not marked read',
      );
      await dispose(tester);
    },
  );

  for (final reducedMotion in [false, true]) {
    testWidgets(
      'quote return restores the reading position with reduced motion $reducedMotion',
      (tester) async {
        final repository = await pumpActions(
          tester,
          messages: [
            for (var id = 1; id < 40; id++) makeChatMessage(id),
            makeChatMessage(40).copyWith(
              content: '> @bob: first message\n\nsource',
              replyToMessageId: 1,
            ),
          ],
        );
        final controller = tester
            .widget<ListView>(find.byType(ListView).last)
            .controller!;
        final sourceOffset = controller.offset;
        final sourceY = tester.getCenter(find.text('source')).dy;
        await tester.tap(find.text('first message'));
        await tester.pumpAndSettle();
        final quotedOffset = controller.offset;
        expect(quotedOffset, lessThan(sourceOffset));
        if (reducedMotion) {
          tester.platformDispatcher.accessibilityFeaturesTestValue =
              FakeAccessibilityFeatures(disableAnimations: true);
          addTearDown(
            tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
          );
          await tester.pump();
        }
        repository.batches.clear();

        await tester.tap(find.byKey(const Key('chat-reply-return')));
        await tester.pump();
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));
        if (reducedMotion) {
          expect(controller.offset, closeTo(sourceOffset, 2));
        } else {
          expect(controller.offset, greaterThan(quotedOffset));
          expect(controller.offset, lessThan(sourceOffset - 2));
          expect(repository.batches, isEmpty);
        }
        await tester.pumpAndSettle();
        expect(find.text('source').hitTestable(), findsOneWidget);
        expect(tester.getCenter(find.text('source')).dy, closeTo(sourceY, 2));
        expect(find.byKey(const Key('chat-reply-return')), findsNothing);
        await dispose(tester);
      },
    );
  }

  testWidgets('quoting own message labels the quote with the viewer username', (
    tester,
  ) async {
    final repository = await pumpActions(
      tester,
      messages: [makeChatMessage(1).copyWith(isSelf: true)],
    );
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    expect(
      find.descendant(of: _preview, matching: find.text('@viewer')),
      findsOneWidget,
    );
    await tester.enterText(
      find.descendant(
        of: find.byType(GfChatInput),
        matching: find.byType(TextField),
      ),
      '收到',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(repository.sent, [(2, '> @viewer: 消息 1\n\n收到')]);
    await dispose(tester);
  });

  testWidgets('own-message quotes fall back to the localized self label', (
    tester,
  ) async {
    final repository = await pumpActions(
      tester,
      messages: [makeChatMessage(1).copyWith(isSelf: true)],
      overrides: [
        pageRepositoryProvider.overrideWithValue(
          _EmptyViewerPageRepository(_client()),
        ),
      ],
    );
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    expect(
      find.descendant(of: _preview, matching: find.text('You')),
      findsOneWidget,
    );
    await tester.enterText(
      find.descendant(
        of: find.byType(GfChatInput),
        matching: find.byType(TextField),
      ),
      '收到',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(repository.sent, [(2, '> You: 消息 1\n\n收到')]);
    await dispose(tester);
  });

  testWidgets('a failed reply keeps its quote and retries the same entry', (
    tester,
  ) async {
    final repository = await pumpActions(
      tester,
      repositoryBuilder: (client) =>
          RecordingSendsChatRepository(client, messages: [makeChatMessage(1)])
            ..failures = 1,
    );
    await openActions(tester, find.text('消息 1'));
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.descendant(
        of: find.byType(GfChatInput),
        matching: find.byType(TextField),
      ),
      '收到',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(repository.sent, [(2, '> @bob: 消息 1\n\n收到')]);
    expect(
      _preview,
      findsOneWidget,
      reason: 'a failed reply keeps its quote attached for the retry',
    );

    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(repository.sent, [
      (2, '> @bob: 消息 1\n\n收到'),
      (2, '> @bob: 消息 1\n\n收到'),
    ]);
    expect(find.text('> @bob: 消息 1\n\n收到'), findsNothing);
    expect(find.text('@bob'), findsOneWidget);
    expect(find.text('收到'), findsOneWidget);
    expect(_preview, findsNothing);
    await dispose(tester);
  });

  for (final replaceQuote in [false, true]) {
    testWidgets(
      replaceQuote
          ? 'late failed reply preserves a newer cancelled quote selection'
          : 'late failed reply cannot attach its quote to a newer draft',
      (tester) async {
        late _DelayedReplyRepository repository;
        await pumpActions(
          tester,
          repositoryBuilder: (client) =>
              repository = _DelayedReplyRepository(client),
        );
        await openActions(tester, find.text('消息 1'));
        await tester.tap(find.text('Reply'));
        await tester.pumpAndSettle();
        final input = find.descendant(
          of: find.byType(GfChatInput),
          matching: find.byType(TextField),
        );
        await tester.enterText(input, 'first reply');
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('chat-send')));
        await tester.pump();
        if (replaceQuote) {
          await openActions(tester, find.text('消息 2'));
          await tester.tap(find.text('Reply'));
          await tester.pumpAndSettle();
          await tester.tap(find.byTooltip('Cancel reply'));
        } else {
          await tester.enterText(input, 'new unrelated draft');
        }
        await tester.pump(const Duration(milliseconds: 500));
        expect(_preview, findsNothing);
        repository.result.completeError(StateError('offline'));
        await tester.pumpAndSettle();
        final quoteCount = _preview.evaluate().length;
        final text = tester.widget<TextField>(input).controller!.text;
        await dispose(tester);
        expect(text, replaceQuote ? 'first reply' : 'new unrelated draft');
        expect(
          quoteCount,
          0,
          reason: 'old failure cannot restore a stale quote',
        );
      },
    );
  }

  for (final chooseNewerQuote in [false, true]) {
    testWidgets(
      chooseNewerQuote
          ? 'successful bubble retry preserves a newer quote'
          : 'successful bubble retry clears its restored quote',
      (tester) async {
        final repository = await pumpActions(
          tester,
          repositoryBuilder: (client) => RecordingSendsChatRepository(
            client,
            messages: [makeChatMessage(1), makeChatMessage(2)],
          )..failures = 1,
        );
        await openActions(tester, find.text('消息 1'));
        await tester.tap(find.text('Reply'));
        await tester.pumpAndSettle();
        await tester.enterText(
          find.descendant(
            of: find.byType(GfChatInput),
            matching: find.byType(TextField),
          ),
          'reply body',
        );
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(const Key('chat-send')));
        await tester.pumpAndSettle();
        expect(_preview, findsOneWidget);
        if (chooseNewerQuote) {
          await openActions(tester, find.text('消息 2'));
          await tester.tap(find.text('Reply'));
          await tester.pumpAndSettle();
        }
        await tester.tap(find.text('Retry sending'));
        await tester.pumpAndSettle();
        expect(repository.sent, [
          (2, '> @bob: 消息 1\n\nreply body'),
          (2, '> @bob: 消息 1\n\nreply body'),
        ]);
        final quoteCount = _preview.evaluate().length;
        if (chooseNewerQuote) {
          expect(
            find.descendant(of: _preview, matching: find.text('消息 2')),
            findsOneWidget,
          );
        }
        await dispose(tester);
        expect(
          quoteCount,
          chooseNewerQuote ? 1 : 0,
          reason: 'an acknowledged retry must release only its own quote',
        );
      },
    );
  }

  testWidgets('barrier and system back dismiss the menu without a reply', (
    tester,
  ) async {
    await pumpActions(tester);
    final composer = find.descendant(
      of: find.byType(GfChatInput),
      matching: find.byType(TextField),
    );
    await tester.enterText(composer, '草稿');
    await tester.pumpAndSettle();
    expect(tester.widget<TextField>(composer).focusNode!.hasFocus, isTrue);

    await openActions(tester, find.text('消息 1'));
    expect(
      tester.widget<TextField>(composer).focusNode!.hasFocus,
      isFalse,
      reason: 'the menu owns focus while it is open',
    );
    await tester.tapAt(const Offset(8, 8));
    await tester.pumpAndSettle();
    expect(find.text('Reply'), findsNothing);
    expect(_preview, findsNothing);
    expect(
      FocusManager.instance.primaryFocus?.hasFocus,
      isTrue,
      reason: 'dismissing the menu leaves the page with focus',
    );
    await tester.enterText(composer, '草稿 2');
    await tester.pumpAndSettle();
    expect(tester.widget<TextField>(composer).controller!.text, '草稿 2');

    await openActions(tester, find.text('消息 1'));
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.text('Reply'), findsNothing);
    expect(_preview, findsNothing);
    expect(tester.widget<TextField>(composer).controller!.text, '草稿 2');
    await dispose(tester);
  });

  testWidgets('menu actions are labelled for screen readers', (tester) async {
    final handle = tester.ensureSemantics();
    await pumpActions(tester);
    await openActions(tester, find.text('消息 1'));
    expect(find.bySemanticsLabel('Reply'), findsOneWidget);
    expect(find.bySemanticsLabel('Report message'), findsOneWidget);
    final node = tester.getSemantics(find.bySemanticsLabel('Reply'));
    expect(node.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
    handle.dispose();
    await dispose(tester);
  });

  testWidgets('Escape dismisses the action menu', (tester) async {
    await pumpActions(tester);
    await openActions(tester, find.text('消息 1'));
    expect(find.text('Reply'), findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.text('Reply'), findsNothing);
    expect(_preview, findsNothing);
    await dispose(tester);
  });

  testWidgets('dragging a bubble scrolls instead of opening the menu', (
    tester,
  ) async {
    await pumpActions(
      tester,
      messages: List.generate(30, (index) => makeChatMessage(index + 1)),
    );
    final list = find.byType(ListView).last;
    final scroll = tester.widget<ListView>(list).controller!;
    final before = scroll.offset;
    await tester.drag(find.text('消息 30'), const Offset(0, 160));
    await tester.pumpAndSettle();
    expect(find.text('Reply'), findsNothing);
    expect(scroll.offset, lessThan(before));
    await dispose(tester);
  });

  testWidgets('sticker-only peer messages still open the action menu', (
    tester,
  ) async {
    debugNetworkImageHttpClientProvider = () => _ImageHttpClient();
    final repository = _StickerRepository();
    final library = StickerLibrary(repository);
    addTearDown(library.dispose);
    final collection = StickerCollection(repository, library);
    try {
      await pumpActions(
        tester,
        stickers: library,
        stickerCollection: collection,
        messages: [makeChatMessage(1).copyWith(content: '[:sticker:smile:]')],
      );
      await openActions(tester, find.byType(StickerImage));
      expect(find.text('Reply'), findsOneWidget);
      expect(find.text('Report message'), findsOneWidget);
      expect(find.text('Save to my stickers'), findsOneWidget);
      await tester.tap(find.text('Save to my stickers'));
      await tester.pumpAndSettle();
      expect(repository.saved, ['smile']);
    } finally {
      debugNetworkImageHttpClientProvider = null;
    }
    await dispose(tester);
  });

  testWidgets('outbox stickers retain their collection action', (tester) async {
    debugNetworkImageHttpClientProvider = () => _ImageHttpClient();
    final repository = _StickerRepository();
    final library = StickerLibrary(repository);
    addTearDown(library.dispose);
    final collection = StickerCollection(repository, library);
    try {
      await pumpActions(
        tester,
        stickers: library,
        stickerCollection: collection,
      );
      await tester.enterText(find.byType(TextField), '[:sticker:smile:]');
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('chat-send')));
      await tester.pumpAndSettle();
      final pending = find.byWidgetPredicate(
        (widget) =>
            widget.key is ValueKey<String> &&
            (widget.key! as ValueKey<String>).value.startsWith('pending-'),
      );
      expect(pending, findsOneWidget);
      await openActions(
        tester,
        find.descendant(of: pending, matching: find.byType(StickerImage)),
      );
      expect(find.text('Save to my stickers'), findsOneWidget);
      await tester.tap(find.text('Save to my stickers'));
      await tester.pumpAndSettle();
      expect(repository.saved, ['smile']);
    } finally {
      debugNetworkImageHttpClientProvider = null;
    }
    await dispose(tester);
  });

  testWidgets('a failed sticker image still opens the action menu', (
    tester,
  ) async {
    final repository = _StickerRepository();
    final library = StickerLibrary(repository);
    addTearDown(library.dispose);
    final collection = StickerCollection(repository, library);
    // No debug image client: Image.network fails in the test binding, so the
    // sticker renders its unavailable state. The dwell below delivers a read
    // receipt and rebuilds the row, exactly like polls do in production.
    await pumpActions(
      tester,
      stickers: library,
      stickerCollection: collection,
      messages: [makeChatMessage(1).copyWith(content: '[:sticker:smile:]')],
    );
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    await openActions(tester, find.byType(StickerImage));
    expect(find.text('Reply'), findsOneWidget);
    expect(find.text('Report message'), findsOneWidget);
    expect(find.text('Save to my stickers'), findsOneWidget);
    await dispose(tester);
  });

  testWidgets('the menu and a long quote preview fit a narrow viewport', (
    tester,
  ) async {
    final content = '长内容' * 80;
    await pumpActions(
      tester,
      messages: [makeChatMessage(1).copyWith(content: content)],
    );
    await tester.binding.setSurfaceSize(const Size(320, 600));
    await tester.pumpAndSettle();
    await openActions(tester, find.text(content));
    expect(find.text('Reply'), findsOneWidget);
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    expect(_preview, findsOneWidget);
    expect(tester.takeException(), isNull);
    await dispose(tester);
  });
}
