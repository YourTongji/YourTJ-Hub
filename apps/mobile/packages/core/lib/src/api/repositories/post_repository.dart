import '../gf_api_client.dart';
import '../../gen/post_revision.dart';

/// 创建/编辑帖子的成功结果。
/// 成功信封上的待审 messageCode(issue #975):敏感词转审与 AI 图文审查转审
/// 共用,不暴露触因。内容在人工审核通过前不公开。
const String pendingReviewMessageCode = 'content.moderation.pendingReview';

/// 创建/编辑话题的成功结果。
class WriteTopicResult {
  const WriteTopicResult({required this.id, this.pendingReview = false});

  final int id;

  /// 话题已转入人工审核,通过前不公开。
  final bool pendingReview;
}

class CreatePostResult {
  const CreatePostResult({
    required this.id,
    this.postNo,
    required this.renderedContent,
    this.pendingReview = false,
  });

  final int id;
  final int? postNo;
  final String renderedContent;

  /// 回复已转入人工审核,通过前不公开。
  final bool pendingReview;
}

/// 更新帖子的成功结果。
class UpdatePostResult {
  const UpdatePostResult({
    required this.id,
    this.postNo,
    required this.content,
    required this.renderedContent,
    required this.updatedAt,
    this.lastEditorId,
    this.lastEditedAt,
    this.revisionCount,
    this.pendingReview = false,
  });

  final int id;
  final int? postNo;
  final String content;
  final String renderedContent;
  final String updatedAt;

  /// 最后编辑者/时间与版本数：编辑过才非空（首楼编辑开放后由服务端返回）。
  final int? lastEditorId;
  final String? lastEditedAt;
  final int? revisionCount;

  /// 编辑后的内容已转入人工审核,通过前不公开。
  final bool pendingReview;
}

/// 帖子相关接口:创建/更新/删除/点赞/收藏/举报。
class PostRepository {
  PostRepository(this._client);

  final GfApiClient _client;

  /// 创建回复。replyToPostId 为 0 表示直接回复主题。
  /// 注意:蜜罐字段(website)绝不发送。
  Future<CreatePostResult> createPost({
    required int topicId,
    required String content,
    int replyToPostId = 0,
    String? captchaId,
    String? captchaCode,
  }) async {
    final response = await _client.postEnvelope<CreatePostResult>(
      '/api/forum/posts/create',
      body: {
        'topicId': topicId,
        'content': content,
        'replyToPostId': replyToPostId,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty)
          'captchaCode': captchaCode,
      },
      parser: (json) => CreatePostResult(
        id: (json as Map<String, dynamic>)['id'] as int,
        postNo: (json['postNo'] as num?)?.toInt(),
        renderedContent: (json['renderedContent'] as String?) ?? '',
      ),
    );
    final CreatePostResult created = response.result!;
    return CreatePostResult(
      id: created.id,
      postNo: created.postNo,
      renderedContent: created.renderedContent,
      pendingReview: response.messageCode == pendingReviewMessageCode,
    );
  }

  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    final response = await _client.postEnvelope<UpdatePostResult>(
      '/api/forum/posts/update',
      body: {'postId': postId, 'content': content},
      parser: (json) => UpdatePostResult(
        id: (json as Map<String, dynamic>)['id'] as int,
        postNo: (json['postNo'] as num?)?.toInt(),
        content: (json['content'] as String?) ?? '',
        renderedContent: (json['renderedContent'] as String?) ?? '',
        updatedAt: (json['updatedAt'] as String?) ?? '',
        lastEditorId: (json['lastEditorId'] as num?)?.toInt(),
        lastEditedAt: json['lastEditedAt'] as String?,
        revisionCount: (json['revisionCount'] as num?)?.toInt(),
      ),
    );
    final UpdatePostResult updated = response.result!;
    return UpdatePostResult(
      id: updated.id,
      postNo: updated.postNo,
      content: updated.content,
      renderedContent: updated.renderedContent,
      updatedAt: updated.updatedAt,
      lastEditorId: updated.lastEditorId,
      lastEditedAt: updated.lastEditedAt,
      revisionCount: updated.revisionCount,
      pendingReview: response.messageCode == pendingReviewMessageCode,
    );
  }

  Future<bool> deletePost({required int postId}) async {
    await _client.post<Object?>(
      '/api/forum/posts/delete',
      body: {'postId': postId},
    );
    return true;
  }

  Future<PostRevisionPage> revisions({
    required int postId,
    int beforeVersion = 0,
  }) => _client.get(
    '/api/forum/posts/revisions',
    queryParameters: {
      'postId': postId,
      'beforeVersion': beforeVersion,
      'limit': 20,
    },
    parser: (json) => PostRevisionPage.fromJson(json as Map<String, dynamic>),
  );

  Future<void> moderate({required int postId, required bool ban}) async {
    await _client.post<Object?>(
      '/api/forum/moderation/post-status',
      body: {'postId': postId, 'action': ban ? 'ban' : 'unban'},
    );
  }

  /// action: 1 点赞, 2 取消点赞。
  Future<bool> likePost({required int postId, required int action}) async {
    await _client.post<Object?>(
      '/api/forum/posts/like',
      body: {'postId': postId, 'action': action},
    );
    return true;
  }

  /// action: 1 收藏, 2 取消收藏。
  Future<bool> bookmarkPost({required int postId, required int action}) async {
    await _client.post<Object?>(
      '/api/forum/posts/bookmark',
      body: {'postId': postId, 'action': action},
    );
    return true;
  }

  /// 举报帖子/话题。
  Future<bool> report({
    required String targetType,
    required int targetId,
    required String reason,
    required String note,
  }) async {
    await _client.post<Object?>(
      '/api/forum/report',
      body: {
        'targetType': targetType,
        'targetId': targetId,
        'reason': reason,
        'note': note,
      },
    );
    return true;
  }
}
