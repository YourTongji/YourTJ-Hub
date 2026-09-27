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
  test('markVisible sends exact IDs and decodes the contract acknowledgement', () async {
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
  });

  test('read-state lookup accepts mixed-direction IDs without text bodies', () async {
    final dio = Dio();
    dio.httpClientAdapter = MockAdapter((request) async {
      expect(request.path, '/api/forum/chat/message-read-states');
      expect(request.data, {
        'convId': 37701,
        'messageIds': [39003, 39001, 39002],
      });
      return ResponseData(200, _fixture('chat-message-read-states-success.json'));
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
  });

  test('invalid explicit IDs return a business error', () async {
    final dio = Dio();
    dio.httpClientAdapter = MockAdapter((request) async =>
        ResponseData(200, _fixture('chat-mark-read-failed.json')));
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
