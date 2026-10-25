import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;
import 'push_repository_test.dart' show Tokens;

void main() {
  test('follow-user returns the applied state for mutual follows', () async {
    final dio = Dio();
    addTearDown(dio.close);
    dio.httpClientAdapter = MockAdapter((request) async {
      return ResponseData(200, {
        'code': 0,
        'result': {'isFollowing': true, 'isMutualFollowing': true},
      });
    });
    final repo = TopicRepository(GfApiClient(dio: dio, tokenStorage: Tokens()));

    final applied = await repo.followUser(userId: 42, isFollowing: false);
    expect(applied.isFollowing, isTrue);
    expect(applied.isMutualFollowing, isTrue);
  });

  test(
    'follow-user falls back to the optimistic value for a legacy body',
    () async {
      final dio = Dio();
      addTearDown(dio.close);
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(200, {'code': 0, 'result': true});
      });
      final repo = TopicRepository(
        GfApiClient(dio: dio, tokenStorage: Tokens()),
      );

      final unfollow = await repo.followUser(userId: 42, isFollowing: true);
      expect(unfollow.isFollowing, isFalse);
      expect(unfollow.isMutualFollowing, isFalse);
    },
  );

  test('unfollow response never reports a mutual follow', () async {
    final dio = Dio();
    addTearDown(dio.close);
    dio.httpClientAdapter = MockAdapter((request) async {
      return ResponseData(200, {
        'code': 0,
        'result': {'isFollowing': false, 'isMutualFollowing': false},
      });
    });
    final repo = TopicRepository(GfApiClient(dio: dio, tokenStorage: Tokens()));

    final applied = await repo.followUser(userId: 42, isFollowing: true);
    expect(applied.isFollowing, isFalse);
    expect(applied.isMutualFollowing, isFalse);
  });
}
