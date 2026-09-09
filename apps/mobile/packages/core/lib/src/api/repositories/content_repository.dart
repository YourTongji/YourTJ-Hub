import '../../gen/user_content.dart';
import '../gf_api_client.dart';

class ContentRepository {
  ContentRepository(this._client);
  final GfApiClient _client;
  static const _base = '/api/forum/user';

  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) => _client.get(
    '$_base/${deleted ? 'deleted-content' : 'my-content'}',
    queryParameters: {
      'contentType': contentType,
      'cursorId': cursor,
      'limit': 20,
    },
    parser: (json) => UserContentPage.fromJson(json as Map<String, dynamic>),
  );

  Future<void> restore(UserContentItem item) async {
    await _client.post<Object?>(
      '$_base/content-restore',
      body: {'contentType': item.contentType, 'contentId': item.id},
    );
  }

  Future<List<ContentDeletionResult>> delete({
    required String contentType,
    required List<int> ids,
    String? password,
  }) => _client.post(
    '$_base/content-batch-delete',
    body: {
      'contentType': contentType,
      'contentIds': ids,
      if (password != null) ...{'force': true, 'password': password},
    },
    parser: (json) => ((json as Map<String, dynamic>)['results'] as List)
        .map(
          (item) =>
              ContentDeletionResult.fromJson(item as Map<String, dynamic>),
        )
        .toList(),
  );

  Future<void> purge(
    UserContentItem item, {
    String? password,
  }) async {
    await _client.post<Object?>(
      '$_base/content-purge',
      body: {
        'contentType': item.contentType,
        'contentId': item.id,
        if (password != null) ...{'force': true, 'password': password},
      },
    );
  }

  Future<void> closeAccount({
    required String mode,
    required String password,
  }) async {
    await _client.post<Object?>(
      '$_base/account-close',
      body: {'mode': mode, 'password': password},
    );
  }
}
