import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/topic/mention_search.dart';
import 'package:forum_app/src/providers.dart';

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
  SearchPageProps result;
  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) async {
    expect(query, 'au');
    expect(scope, 'users');
    return result;
  }
}

void main() {
  const result = SearchPageProps(
    query: 'au',
    scope: 'users',
    topics: [],
    users: [
      UserSearchPayload(
        id: 2,
        username: 'author',
        nickname: 'Author',
        avatarUrl: '',
        bio: '',
      ),
    ],
    categories: [],
    courses: [],
    total: 1,
    usersTotal: 1,
    categoriesTotal: 0,
    coursesTotal: 0,
    totalPages: 1,
    pagination: PaginationPayload(
      page: 1,
      nextPage: 0,
      hasNext: false,
      nextUrl: '',
    ),
  );

  test(
    'user search maps candidates and distinguishes unavailable scope from empty results',
    () async {
      final topics = _Topics(result);
      final container = ProviderContainer(
        overrides: [topicRepositoryProvider.overrideWithValue(topics)],
      );
      addTearDown(container.dispose);
      final search = container.read(mentionUserSearchProvider);
      expect((await search('au')).single.username, 'author');
      topics.result = result.copyWith(searchUnavailable: true);
      await expectLater(search('au'), throwsException);
      topics.result = result.copyWith(failedScopes: ['users']);
      await expectLater(search('au'), throwsException);
      topics.result = result.copyWith(failedScopes: ['topics']);
      expect((await search('au')).single.id, 2);
      topics.result = result.copyWith(users: []);
      expect(await search('au'), isEmpty);
    },
  );
}
