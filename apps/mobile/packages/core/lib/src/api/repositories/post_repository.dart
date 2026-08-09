import '../gf_api_client.dart';

/// 创建/编辑帖子的成功结果。
class CreatePostResult {
  const CreatePostResult({
    required this.id,
    this.postNo,
    required this.renderedContent,
  });

  final int id;
  final int? postNo;
  final String renderedContent;
}

/// 更新帖子的成功结果。
class UpdatePostResult {
  const UpdatePostResult({
    required this.id,
    this.postNo,
    required this.content,
    required this.renderedContent,
    required this.updatedAt,
  });

  final int id;
  final int? postNo;
  final String content;
  final String renderedContent;
  final String updatedAt;
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
  }) {
    return _client.post<CreatePostResult>(
      '/api/forum/posts/create',
      body: {
        'topicId': topicId,
        'content': content,
        'replyToPostId': replyToPostId,
        if (captchaId != null && captchaId.isNotEmpty) 'captchaId': captchaId,
        if (captchaCode != null && captchaCode.isNotEmpty) 'captchaCode': captchaCode,
      },
      parser: (json) => CreatePostResult(
        id: (json as Map<String, dynamic>)['id'] as int,
        postNo: (json['postNo'] as num?)?.toInt(),
        renderedContent: (json['renderedContent'] as String?) ?? '',
      ),
    );
  }

  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) {
    return _client.post<UpdatePostResult>(
      '/api/forum/posts/update',
      body: {'postId': postId, 'content': content},
      parser: (json) => UpdatePostResult(
        id: (json as Map<String, dynamic>)['id'] as int,
        postNo: (json['postNo'] as num?)?.toInt(),
        content: (json['content'] as String?) ?? '',
        renderedContent: (json['renderedContent'] as String?) ?? '',
        updatedAt: (json['updatedAt'] as String?) ?? '',
      ),
    );
  }

  Future<bool> deletePost({required int postId}) async {
    await _client.post<Object?>('/api/forum/posts/delete', body: {'postId': postId});
    return true;
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
      body: {'targetType': targetType, 'targetId': targetId, 'reason': reason, 'note': note},
    );
    return true;
  }
}
