import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/topic/mention_search.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/app_config.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _Topics extends TopicRepository {
  _Topics(this.result)
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Tokens(),
          baseUrl: 'http://fake',
        ),
      );
  List<MentionTarget> result;
  @override
  Future<List<MentionTarget>> mentionTargets({
    required String query,
    int limit = 20,
    CancelToken? cancelToken,
  }) async {
    expect(query, 'au');
    expect(limit, 20);
    return result;
  }
}

void main() {
  const result = <MentionTarget>[
    MentionTarget(
      userId: 2,
      username: 'author',
      nickname: 'Author',
      avatarUrl: '',
      actorType: 'human',
    ),
    MentionTarget(
      userId: 3,
      username: 'assistant',
      nickname: 'Assistant',
      avatarUrl: '/bot.png',
      actorType: 'bot',
    ),
  ];

  test('mention target search maps public human and bot candidates', () async {
    final topics = _Topics(result);
    final container = ProviderContainer(
      overrides: [topicRepositoryProvider.overrideWithValue(topics)],
    );
    addTearDown(container.dispose);
    final search = container.read(mentionUserSearchProvider);
    final candidates = await search('au');
    expect(candidates.map((candidate) => candidate.id), [2, 3]);
    expect(candidates.map((candidate) => candidate.actorType), [
      'human',
      'bot',
    ]);
    final baseUrl = AppConfig.apiBaseUrl.isEmpty
        ? GfApiClient.defaultBaseUrl
        : AppConfig.apiBaseUrl;
    expect(candidates.last.avatarUrl, '$baseUrl/bot.png');
  });
}
