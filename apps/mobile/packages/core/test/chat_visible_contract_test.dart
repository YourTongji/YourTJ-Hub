import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Storage implements TokenStorage {
  @override
  Future<String?> read() async => 'test-session';

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

Map<String, dynamic> _fixture(String name) =>
    jsonDecode(
          File(
            '../../../../packages/api-contract/fixtures/$name',
          ).readAsStringSync(),
        )
        as Map<String, dynamic>;

void main() {
  test(
    'nested history and both avatar levels survive the wire/cache roundtrip',
    () {
      final result =
          _fixture('chat-messages-success.json')['result']
              as Map<String, dynamic>;
      final response = ChatMessagesResponse.fromJson(result);
      final parent = response.list.last.forwarded!.messages.single;
      expect(parent.avatarUrl, '/static/pic/6.webp');
      expect(parent.forwarded!.messages.single.avatarUrl, '/static/pic/3.webp');
      expect(parent.forwarded!.messages.single.content, 'hello');
      expect(
        ChatMessagesResponse.fromJson(jsonDecode(jsonEncode(response))),
        response,
      );
    },
  );
  test(
    'forward sends source IDs and stable identity and decodes its acknowledgement',
    () async {
      final dio = Dio();
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/forum/chat/forward');
        expect(request.data, {
          'convId': 1,
          'peerId': 3,
          'messageIds': [2, 4],
          'mode': 'merged',
          'clientForwardId': 'stable-batch',
        });
        return ResponseData(200, _fixture('chat-forward-success.json'));
      });
      final result =
          await ChatRepository(
            GfApiClient(dio: dio, tokenStorage: _Storage()),
          ).forwardMessages(
            convId: 1,
            peerId: 3,
            messageIds: [2, 4],
            merged: true,
            clientForwardId: 'stable-batch',
          );
      expect(result.convId, 1001);
      expect(result.messageIds, [2001]);
    },
  );
  test(
    'legacy messages omit a bundle and structured copies survive JSON roundtrip',
    () {
      final json = {
        'id': 1,
        'senderId': 2,
        'content': 'fallback',
        'msgType': 4,
        'isRead': 0,
        'isSelf': false,
        'createdAt': '2026-09-28T01:00:00Z',
      };
      expect(ChatMessagePayload.fromJson(json).forwarded, isNull);
      json['forwarded'] = {
        'version': 1,
        'messages': [
          {
            'senderName': 'Alice',
            'avatarUrl': '/static/pic/3.webp',
            'content': 'body',
            'createdAt': '2026-09-28T01:00:00Z',
            'msgType': 1,
          },
        ],
      };
      final message = ChatMessagePayload.fromJson(json);
      expect(
        ChatMessagePayload.fromJson(jsonDecode(jsonEncode(message))),
        message,
      );
      expect(message.forwarded!.messages.single.content, 'body');
      expect(
        message.forwarded!.messages.single.avatarUrl,
        '/static/pic/3.webp',
      );
      final legacy = Map<String, dynamic>.from(
        (json['forwarded'] as Map)['messages'][0] as Map,
      )..remove('avatarUrl');
      expect(ChatForwardEntry.fromJson(legacy).avatarUrl, isEmpty);
    },
  );

  test(
    'markVisible sends exact IDs and decodes the contract acknowledgement',
    () async {
      final dio = Dio();
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.method, 'POST');
        expect(request.path, '/api/forum/chat/mark-visible');
        expect(request.data, {
          'convId': 27701,
          'messageIds': [29003, 29001, 29003],
        });
        return ResponseData(200, _fixture('chat-mark-visible-success.json'));
      });
      final repository = ChatRepository(
        GfApiClient(dio: dio, tokenStorage: _Storage()),
      );
      final result = await repository.markVisible(
        convId: 27701,
        messageIds: [29003, 29001, 29003],
      );
      expect(result.convId, 27701);
      expect(result.acknowledgedMessageIds, [29003, 29001]);
      expect(result.unreadCount, 1);
    },
  );

  test(
    'read-state lookup accepts mixed-direction IDs without text bodies',
    () async {
      final dio = Dio();
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/forum/chat/message-read-states');
        expect(request.data, {
          'convId': 37701,
          'messageIds': [39003, 39001, 39002],
        });
        return ResponseData(
          200,
          _fixture('chat-message-read-states-success.json'),
        );
      });
      final repository = ChatRepository(
        GfApiClient(dio: dio, tokenStorage: _Storage()),
      );
      final result = await repository.messageReadStates(
        convId: 37701,
        messageIds: [39003, 39001, 39002],
      );
      expect(result.items.map((item) => item.id), [39003, 39001, 39002]);
      expect(result.items.map((item) => item.isRead), [1, 1, 0]);
      expect(result.unreadCount, 1);
    },
  );

  test('invalid explicit IDs return a business error', () async {
    final dio = Dio();
    dio.httpClientAdapter = MockAdapter(
      (request) async =>
          ResponseData(200, _fixture('chat-mark-read-failed.json')),
    );
    final repository = ChatRepository(
      GfApiClient(dio: dio, tokenStorage: _Storage()),
    );
    await expectLater(
      repository.markVisible(convId: 27701, messageIds: [1]),
      throwsA(
        isA<ApiException>().having(
          (error) => error.messageCode,
          'messageCode',
          'chat.markRead.failed',
        ),
      ),
    );
  });
}
