import '../../gen/search.dart';
import '../../gen/topic.dart';
import '../../gen/mention_target.dart';
import '../gf_api_client.dart';
import 'post_repository.dart';

import 'package:dio/dio.dart';

/// 话题相关接口:搜索、帖子窗口、写话题、话题状态、点赞/收藏/关注。
class TopicRepository {
  TopicRepository(this._client);

  final GfApiClient _client;

  /// 搜索(q/scope/page)。注意:蜜罐字段(website)绝不发送。
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
    CancelToken? cancelToken,
  }) {
    return _client.get<SearchPageProps>(
      '/api/forum/search',
      cancelToken: cancelToken,
      queryParameters: {
        'q': query,
        if (scope.isNotEmpty) 'scope': scope,
        'page': page,
      },
      parser: (json) => SearchPageProps.fromJson(json as Map<String, dynamic>),
    );
  }

  /// Search public human and Agent identities for editor autocomplete only.
  Future<List<MentionTarget>> mentionTargets({
    required String query,
    int limit = 20,
    CancelToken? cancelToken,
  }) {
    return _client.get<List<MentionTarget>>(
      '/api/forum/mention-targets',
      cancelToken: cancelToken,
      queryParameters: {'q': query, 'limit': limit},
      parser: (json) => (json as List<dynamic>)
          .map((item) => MentionTarget.fromJson(item as Map<String, dynamic>))
          .toList(),
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
      parser: (json) =>
          PostWindowPayload.fromJson(json as Map<String, dynamic>),
    );
  }

  /// 创建/编辑话题,成功返回 topicId。
  Future<int> writeTopic({
    required int topicId,
    required String title,
    required String content,
    required List<int> categoryIds,
    required int topicStatus,
    int contentType = 3,
    List<String>? images,
    String? captchaId,
    String? captchaCode,
  }) async {
    final WriteTopicResult result = await writeTopicResult(
      topicId: topicId,
      title: title,
      content: content,
      categoryIds: categoryIds,
      topicStatus: topicStatus,
      contentType: contentType,
      images: images,
      captchaId: captchaId,
      captchaCode: captchaCode,
    );
    return result.id;
  }

  /// 创建/编辑话题并保留成功信封元数据:内容转入人工审核(敏感词或 AI 图文
  /// 审查,issue #975)时 [WriteTopicResult.pendingReview] 为 true。
  Future<WriteTopicResult> writeTopicResult({
    required int topicId,
    required String title,
    required String content,
    required List<int> categoryIds,
    required int topicStatus,
    int contentType = 3,
    List<String>? images,
    String? captchaId,
    String? captchaCode,
  }) async {
    final response = await _client.postEnvelope<int>(
      '/api/forum/topics/write',
      body: {
        'topicId': topicId,
        'title': title,
        'content': content,
        'categoryId': categoryIds,
        'topicStatus': topicStatus,
        'contentType': contentType == 0 ? 3 : contentType,
        'images': ?images,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty)
          'captchaCode': captchaCode,
      },
      parser: (json) => json is int ? json : (json as num?)?.toInt() ?? topicId,
    );
    return WriteTopicResult(
      id: response.result ?? topicId,
      pendingReview: isPendingReviewCode(response.messageCode),
      checking: response.messageCode == checkingMessageCode,
    );
  }

  /// 更新话题状态(0 普通 / 1 置顶)。
  Future<bool> updateTopicStatus({
    required int topicId,
    required int topicStatus,
  }) async {
    await _client.post<Object?>(
      '/api/forum/topics/status',
      body: {'topicId': topicId, 'topicStatus': topicStatus},
    );
    return true;
  }

  Future<void> deleteTopic({required int topicId}) async {
    await _client.post<Object?>(
      '/api/forum/topics/delete',
      body: {'topicId': topicId},
    );
  }

  Future<void> moderate({required int topicId, required bool ban}) async {
    await _client.post<Object?>(
      '/api/forum/moderation/topic-status',
      body: {'topicId': topicId, 'action': ban ? 'ban' : 'unban'},
    );
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
  Future<bool> bookmarkTopic({
    required int topicId,
    required int action,
  }) async {
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
  Future<bool> followUser({
    required int userId,
    required bool isFollowing,
  }) async {
    await _client.post<Object?>(
      '/api/forum/follow-user',
      body: {'id': userId, 'action': isFollowing ? 2 : 1},
    );
    return true;
  }
}
