import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/images/image_upload.dart';
import 'package:forum_app/src/messages/chat_image.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:forum_app/src/providers.dart';
import 'package:image_picker/image_picker.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'chat_message_actions_test.dart' show RecordingSendsChatRepository;
import 'chat_send_failure_test.dart' show GatedChatRepository;
import 'chat_visible_read_test.dart' show pumpChat;
import 'fixtures/page_fixtures.dart' show messagesPayloadJson, parsePayload;
import 'pages_behavior_test.dart' show MemTokenStorage, makeChatMessage;

final Uint8List _png = base64Decode(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAACklEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg==',
);

/// A real 4x4 JPEG: the chat preview decodes the picked bytes, so the test
/// payload must be a valid image, not just a magic-number header.
final Uint8List _jpeg = base64Decode(
  '/9j/2wCEAAUDBAQEAwUEBAQFBQUGBwwIBwcHBw8LCwkMEQ8SEhEPERETFhwXExQaFRERGCEYGh0dHx8fExciJCIeJBweHx4BBQUFBwYHDggIDh4UERQeHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHv/AABEIAAQABAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APMaKKK+UPuD/9k=',
);

class _ImagePreviewPageRepository extends PageRepository {
  _ImagePreviewPageRepository(this.preview)
    : super(GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()));

  final String preview;

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    final json = messagesPayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    final conversations = props['conversations'] as List;
    (conversations.first as Map<String, dynamic>)['lastMsg'] = preview;
    return parsePayload(json);
  }
}

class _Photos extends ImagePicker {
  Completer<XFile?>? gate;
  XFile? file = XFile.fromData(_png, name: 'photo.png', path: 'photo.png');
  @override
  Future<XFile?> pickImage({
    required ImageSource source,
    double? maxWidth,
    double? maxHeight,
    int? imageQuality,
    CameraDevice preferredCameraDevice = CameraDevice.rear,
    bool requestFullMetadata = true,
  }) async {
    expect(source, ImageSource.gallery);
    expect(requestFullMetadata, isFalse);
    return gate == null ? file : gate!.future;
  }
}

class _Files extends FileRepository {
  _Files() : super(GfApiClient(dio: Dio(), tokenStorage: MemTokenStorage()));
  int calls = 0;
  bool fail = false;
  Completer<String>? gate;
  final filenames = <String>[];
  @override
  Future<String> uploadImage({
    required List<int> bytes,
    required String filename,
  }) async {
    calls++;
    filenames.add(filename);
    if (fail) throw StateError('upload unavailable');
    return gate == null ? '/file/img/photo.png' : gate!.future;
  }
}

class _ImageStore extends ChatDraftStore {
  bool failWrites = false;
  @override
  Future<void> writeImage(
    String scope,
    int peerId,
    String clientMessageId,
    Map<String, Object> data, {
    required int generation,
  }) async {
    if (failWrites) throw StateError('secure storage unavailable');
    return super.writeImage(
      scope,
      peerId,
      clientMessageId,
      data,
      generation: generation,
    );
  }
}

void main() {
  late _Photos photos;
  late _Files files;
  late _ImageStore imageStore;
  var imageStoreCreated = false;
  late RecordingSendsChatRepository chat;

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
    photos = _Photos();
    files = _Files();
    imageStoreCreated = false;
  });

  Future<ProviderContainer> pump(
    WidgetTester tester, {
    List<ChatMessagePayload>? messages,
    Locale locale = const Locale('en'),
  }) async {
    if (!imageStoreCreated) {
      imageStore = _ImageStore();
      imageStoreCreated = true;
    }
    final result = await pumpChat(
      tester,
      locale: locale,
      repository: (client) => chat = RecordingSendsChatRepository(
        client,
        messages: messages ?? [makeChatMessage(1)],
      ),
      overrides: [
        imagePickerProvider.overrideWithValue(photos),
        fileRepositoryProvider.overrideWithValue(files),
        chatDraftStoreProvider.overrideWithValue(imageStore),
      ],
    );
    return result.container;
  }

  Future<void> choose(WidgetTester tester) async {
    await tester.tap(find.byKey(const Key('chat-attach')));
    await tester.pumpAndSettle();
    expect(find.text('Add sticker'), findsOneWidget);
    await tester.tap(find.text('Add image'));
    await tester.pumpAndSettle();
  }

  Future<void> dispose(WidgetTester tester) async {
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pumpAndSettle();
  }

  testWidgets('untrusted image origins never create a network image', (
    tester,
  ) async {
    await pump(
      tester,
      messages: [
        makeChatMessage(
          1,
        ).copyWith(content: 'https://tracker.example/pixel.png', msgType: 2),
      ],
    );
    expect(find.byType(ChatImage), findsNothing);
    expect(find.text('https://tracker.example/pixel.png'), findsOneWidget);
    await dispose(tester);
  });

  testWidgets('chat thumbnail bounds decoded device pixels', (tester) async {
    await pump(
      tester,
      messages: [
        makeChatMessage(1).copyWith(content: '/file/img/photo.png', msgType: 2),
      ],
    );
    final image = tester.widget<GfNetworkImage>(
      find.descendant(
        of: find.byType(ChatImage),
        matching: find.byType(GfNetworkImage),
      ),
    );
    final ratio = MediaQuery.devicePixelRatioOf(
      tester.element(find.byType(ChatImage)),
    );
    expect(image.cacheWidth, (240 * ratio).ceil());
    expect(image.cacheHeight, (180 * ratio).ceil());
    await dispose(tester);
  });

  testWidgets('a pending text send prevents starting an image upload', (
    tester,
  ) async {
    late GatedChatRepository gated;
    await pumpChat(
      tester,
      repository: (client) =>
          gated = GatedChatRepository(client, messages: [makeChatMessage(1)]),
      overrides: [
        imagePickerProvider.overrideWithValue(photos),
        fileRepositoryProvider.overrideWithValue(files),
      ],
    );
    await tester.enterText(find.byType(TextField).last, 'first');
    await tester.pump();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pump();
    expect(gated.pending, hasLength(1));
    await choose(tester);
    expect(find.byType(ChatImagePreview), findsNothing);
    expect(files.calls, 0);
    Navigator.of(tester.element(find.text('Add image'))).pop();
    gated.pending.single.complete(5);
    await tester.pumpAndSettle();
    await dispose(tester);
  });

  testWidgets('received image quotes render in the reader language', (
    tester,
  ) async {
    await pump(
      tester,
      locale: const Locale('zh'),
      messages: [
        makeChatMessage(1).copyWith(content: '> @bob: [Image]\n\nnice photo'),
      ],
    );
    expect(find.text('[图片]'), findsOneWidget);
    expect(find.text('[Image]'), findsNothing);
    await dispose(tester);
  });

  test('image trust compares exact configured origins', () {
    const base = 'https://forum.example';
    for (final value in [
      '/file/img/photo.png',
      'https://forum.example/file/img/a',
      'https://forum.example:443/a',
    ]) {
      expect(isChatImageUrl(value, baseUrl: base), isTrue, reason: value);
    }
    for (final value in [
      '//tracker.example/a',
      'https://forum.example.evil/a',
      'https://user@forum.example/a',
      'https://forum.example:444/a',
      'http://forum.example/a',
      'data:image/png;base64,a',
      '/\\tracker.example/a',
      'https://tracker.example/a',
    ]) {
      expect(isChatImageUrl(value, baseUrl: base), isFalse, reason: value);
    }
    expect(
      isChatImageUrl(
        'https://cdn.example/a',
        baseUrl: base,
        assetOrigins: ['https://cdn.example'],
      ),
      isTrue,
    );
    expect(
      isChatImageUrl(
        'https://cdn.example/a',
        baseUrl: base,
        assetOrigins: [
          'https://cdn.example/path',
          '//cdn.example',
          'https://cdn.example?x=1',
        ],
      ),
      isFalse,
    );
  });

  testWidgets('German image reply persists a locale-independent marker', (
    tester,
  ) async {
    await pump(
      tester,
      locale: const Locale('de'),
      messages: [
        makeChatMessage(1).copyWith(content: '/file/img/photo.png', msgType: 2),
      ],
    );
    await tester.longPress(find.byType(ChatImage));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Antworten'));
    await tester.pumpAndSettle();
    expect(find.text('[Bild]'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, 'schön');
    await tester.pump();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(chat.sent.single.$2, contains('[Image]\n\nschön'));
    await dispose(tester);
  });

  for (final (content, preview) in [
    ('/file/2026/09/30/photo.png', '[Image]'),
    ('https://cdn.example.com/PHOTO.JPEG?token=abc#image', '[Image]'),
    ('https://cdn.example.com/animation.webp', '[Image]'),
    ('https://example.com/topic/123', 'https://example.com/topic/123'),
    (
      'https://example.com/?file=photo.png',
      'https://example.com/?file=photo.png',
    ),
    ('photo: /file/photo.png', 'photo: /file/photo.png'),
  ]) {
    testWidgets('conversation image preview handles $content', (tester) async {
      await pumpChat(
        tester,
        targetUserId: null,
        overrides: [
          pageRepositoryProvider.overrideWithValue(
            _ImagePreviewPageRepository(content),
          ),
        ],
      );
      final row = tester
          .widgetList<GfConversationRow>(find.byType(GfConversationRow))
          .firstWhere((row) => row.name == 'bob');
      expect(row.lastMessage, preview);
      expect(find.text(preview), findsOneWidget);
      await dispose(tester);
    });
  }

  testWidgets('re-encoded picker output uploads under the sniffed jpg name', (
    tester,
  ) async {
    photos.file = XFile.fromData(
      _jpeg,
      name: 'scaled_photo.png',
      path: 'scaled_photo.png',
    );
    await pump(tester);
    await choose(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pumpAndSettle();
    expect(files.calls, 1);
    expect(files.filenames.single, 'scaled_photo.jpg');
    await dispose(tester);
  });

  testWidgets(
    'image selection previews before upload and cancel preserves draft',
    (tester) async {
      await pump(tester);
      await tester.enterText(find.byType(TextField).last, 'unsent text');
      await choose(tester);
      expect(find.byType(ChatImagePreview), findsOneWidget);
      expect(files.calls, 0);
      expect(chat.sent, isEmpty);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(files.calls, 0);
      expect(find.text('unsent text'), findsOneWidget);
      await dispose(tester);
    },
  );

  testWidgets(
    'upload failure retains preview; failed image send retries without reupload or losing text',
    (tester) async {
      await pump(tester);
      await tester.enterText(find.byType(TextField).last, 'keep my draft');
      await choose(tester);
      files.fail = true;
      await tester.tap(find.widgetWithText(FilledButton, 'Send'));
      await tester.pumpAndSettle();
      expect(find.byType(ChatImagePreview), findsOneWidget);
      expect(chat.sent, isEmpty);
      files.fail = false;
      chat.failures = 1;
      await tester.tap(find.widgetWithText(FilledButton, 'Retry'));
      await tester.pumpAndSettle();
      expect(find.byType(ChatImagePreview), findsNothing);
      expect(files.calls, 2);
      expect(chat.sentTypes, [2]);
      expect(chat.sent.single.$2, '/file/img/photo.png');
      expect(find.byType(ChatImage), findsOneWidget);
      expect(find.text('keep my draft'), findsOneWidget);
      await tester.tap(find.text('Retry sending'));
      await tester.pumpAndSettle();
      expect(files.calls, 2);
      expect(chat.sentTypes, [2, 2]);
      expect(chat.clientKeys[1], chat.clientKeys[0]);
      expect(find.text('keep my draft'), findsOneWidget);
      await dispose(tester);
    },
  );

  testWidgets('late photo selection after account change never uploads', (
    tester,
  ) async {
    final container = await pump(tester);
    photos.gate = Completer<XFile?>();
    await choose(tester);
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    photos.gate!.complete(photos.file);
    await tester.pumpAndSettle();
    expect(files.calls, 0);
    expect(chat.sent, isEmpty);
    await dispose(tester);
  });

  testWidgets('late upload after account change never sends an image', (
    tester,
  ) async {
    final container = await pump(tester);
    await choose(tester);
    files.gate = Completer<String>();
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    files.gate!.complete('/file/img/photo.png');
    await tester.pumpAndSettle();
    expect(chat.sent, isEmpty);
    expect(find.byType(Image), findsNothing);
    await dispose(tester);
    await pump(tester);
    expect(
      find.byType(ChatImage),
      findsOneWidget,
      reason: 'the upload remains recoverable by its original account',
    );
    expect(chat.sent, isEmpty);
    await dispose(tester);
  });

  testWidgets('a recovery write failure retries the same uploaded URL', (
    tester,
  ) async {
    await pump(tester);
    await choose(tester);
    imageStore.failWrites = true;
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pumpAndSettle();
    expect(files.calls, 1);
    expect(chat.sent, isEmpty);
    expect(find.byType(ChatImagePreview), findsOneWidget);
    imageStore.failWrites = false;
    await tester.tap(find.widgetWithText(FilledButton, 'Retry'));
    await tester.pumpAndSettle();
    expect(files.calls, 1);
    expect(chat.sentTypes, [2]);
    await dispose(tester);
  });

  testWidgets('explicit data erasure fences a late uploaded recovery record', (
    tester,
  ) async {
    final container = await pump(tester);
    await choose(tester);
    files.gate = Completer<String>();
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();
    expect(files.calls, 1);
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await imageStore.clearAll();
    files.gate!.complete('/file/img/photo.png');
    await tester.pumpAndSettle();
    expect(chat.sent, isEmpty);
    await dispose(tester);
    await pump(tester);
    expect(find.byType(ChatImage), findsNothing);
    await dispose(tester);
  });

  test(
    'image recovery is private to its account and peer and participates in erasure',
    () async {
      imageStore = _ImageStore();
      final generation = imageStore.imageGeneration;
      const data = {
        'content': '/file/img/photo.png',
        'afterId': 0,
        'clientMessageId': 'abc',
      };
      await imageStore.writeImage(
        'site:1',
        2,
        'abc',
        data,
        generation: generation,
      );
      expect(await imageStore.readImages('site:1', 2), [data]);
      expect(await imageStore.readImages('site:2', 2), isEmpty);
      expect(await imageStore.readImages('site:1', 3), isEmpty);
      expect(await imageStore.read('site:1'), isEmpty);
      expect((await imageStore.usage()).count, 1);
      await imageStore.clearAccount('site:1');
      expect(await imageStore.readImages('site:1', 2), isEmpty);
      await expectLater(
        imageStore.writeImage('site:1', 2, 'abc', data, generation: generation),
        throwsStateError,
      );
    },
  );

  testWidgets(
    'failed image send survives a process restart and retries the same intent',
    (tester) async {
      await pump(tester);
      await choose(tester);
      chat.failures = 1;
      await tester.tap(find.widgetWithText(FilledButton, 'Send'));
      await tester.pumpAndSettle();
      final key = chat.clientKeys.single;
      await dispose(tester);
      await pump(tester);
      expect(find.byType(ChatImage), findsOneWidget);
      expect(chat.sent, isEmpty);
      await tester.tap(find.text('Retry sending'));
      await tester.pumpAndSettle();
      expect(chat.clientKeys, [key]);
      expect(files.calls, 1);
      await dispose(tester);
      await pump(tester);
      expect(
        find.byType(ChatImage),
        findsNothing,
        reason: 'acknowledgement removes the recovery record',
      );
      await dispose(tester);
    },
  );

  testWidgets('replying to an image uses a readable quote and its message ID', (
    tester,
  ) async {
    await pump(
      tester,
      messages: [
        makeChatMessage(1).copyWith(content: '/file/img/photo.png', msgType: 2),
      ],
    );
    await tester.longPress(find.byType(ChatImage));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Reply'));
    await tester.pumpAndSettle();
    expect(find.text('[Image]'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, 'nice photo');
    await tester.pump();
    await tester.tap(find.byKey(const Key('chat-send')));
    await tester.pumpAndSettle();
    expect(chat.sent.single.$2, contains('[Image]\n\nnice photo'));
    expect(chat.replyTargets, [1]);
    expect(chat.sentTypes, [1]);
    await dispose(tester);
  });

  testWidgets(
    'received images open the viewer and unsafe image content stays text',
    (tester) async {
      await pump(
        tester,
        messages: [
          makeChatMessage(
            1,
          ).copyWith(content: '/file/img/photo.png', msgType: 2),
          makeChatMessage(
            2,
          ).copyWith(content: 'javascript:alert(1)', msgType: 2),
        ],
      );
      expect(find.byType(ChatImage), findsOneWidget);
      expect(find.text('javascript:alert(1)'), findsOneWidget);
      await tester.tap(find.byType(ChatImage));
      await tester.pumpAndSettle();
      expect(find.byType(GfImageViewer), findsOneWidget);
      await dispose(tester);
    },
  );
}
