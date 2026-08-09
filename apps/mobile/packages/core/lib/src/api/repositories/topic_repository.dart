import '../../gen/search.dart';
import '../../gen/topic.dart';
import '../gf_api_client.dart';

/// 话题相关接口:搜索、帖子窗口、写话题、话题状态、点赞/收藏/关注。
class TopicRepository {
  TopicRepository(this._client);

  final GfApiClient _client;

  /// 搜索(q/scope/page)。注意:蜜罐字段(website)绝不发送。
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
  }) {
    return _client.get<SearchPageProps>(
      '/api/forum/search',
      queryParameters: {
        'q': query,
        if (scope.isNotEmpty) 'scope': scope,
        'page': page,
      },
      parser: (json) => SearchPageProps.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 帖子窗口(游标分页)。
  Future<PostWindowPayload> getPostWindow({
    required int topicId,
    int? anchorPostId,
    int? anchorPostNo,
    int? beforePostNo,
    int? afterPostNo,
    int? limit,
  }) {
    return _client.get<PostWindowPayload>(
      '/api/forum/posts/window',
      queryParameters: {
        'topicId': topicId,
        'anchorPostId': ?anchorPostId,
        'anchorPostNo': ?anchorPostNo,
        'beforePostNo': ?beforePostNo,
        'afterPostNo': ?afterPostNo,
        'limit': ?limit,
      },
      parser: (json) => PostWindowPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 创建/编辑话题,成功返回 topicId。
  Future<int> writeTopic({
    required int topicId,
    required String title,
    required String content,
    required List<int> categoryIds,
    required int topicStatus,
    String? captchaId,
    String? captchaCode,
  }) {
    return _client.post<int>(
      '/api/forum/topics/write',
      body: {
        'topicId': topicId,
        'title': title,
        'content': content,
        'categoryId': categoryIds,
        'topicStatus': topicStatus,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty) 'captchaCode': captchaCode,
      },
      parser: (json) => json is int ? json : (json as num?)?.toInt() ?? topicId,
    );
  }

  /// 更新话题状态(0 普通 / 1 置顶)。
  Future<bool> updateTopicStatus({required int topicId, required int topicStatus}) async {
    await _client.post<Object?>(
      '/api/forum/topics/status',
      body: {'topicId': topicId, 'topicStatus': topicStatus},
    );
    return true;
  }

  /// action: 1 点赞, 2 取消点赞。
  Future<bool> likeTopic({required int topicId, required int action}) async {
    await _client.post<Object?>(
      '/api/forum/topics/like',
      body: {'topicId': topicId, 'action': action},
    );
    return true;
  }

  /// action: 1 收藏, 2 取消收藏。
  Future<bool> bookmarkTopic({required int topicId, required int action}) async {
    await _client.post<Object?>(
      '/api/forum/topics/bookmark',
      body: {'topicId': topicId, 'action': action},
    );
    return true;
  }

  /// action: 1 关注, 2 取消关注。
  Future<bool> watchTopic({required int topicId, required int action}) async {
    await _client.post<Object?>(
      '/api/forum/topics/watch',
      body: {'topicId': topicId, 'action': action},
    );
    return true;
  }

  /// 关注/取消关注用户。isFollowing 为 true 表示当前已关注(取消),否则关注。
  Future<bool> followUser({required int userId, required bool isFollowing}) async {
    await _client.post<Object?>(
      '/api/forum/follow-user',
      body: {'id': userId, 'action': isFollowing ? 2 : 1},
    );
    return true;
  }
}
