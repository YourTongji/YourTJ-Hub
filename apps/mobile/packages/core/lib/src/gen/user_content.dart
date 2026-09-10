/// Mirrors the user-content OpenAPI domain. Permissions come from the server.
class UserContentItem {
  const UserContentItem({
    required this.id,
    required this.contentType,
    required this.title,
    this.excerpt = '',
    this.createdAt = '',
    this.deletedAt = '',
    this.topicId,
    this.postNo,
    this.canRestore = false,
    this.canPermanent = false,
  });
  final int id;
  final String contentType;
  final String title;
  final String excerpt;
  final String createdAt;
  final String deletedAt;
  final int? topicId;
  final int? postNo;
  final bool canRestore;
  final bool canPermanent;
  factory UserContentItem.fromJson(Map<String, dynamic> json) =>
      UserContentItem(
        id: (json['id'] as num).toInt(),
        contentType: json['contentType'] as String,
        title: json['title'] as String,
        excerpt: json['excerpt'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        deletedAt: json['deletedAt'] as String? ?? '',
        topicId: (json['topicId'] as num?)?.toInt(),
        postNo: (json['postNo'] as num?)?.toInt(),
        canRestore: json['canRestore'] == true,
        canPermanent: json['canPermanent'] == true,
      );
}

class UserContentPage {
  const UserContentPage({
    required this.items,
    required this.hasMore,
    required this.nextCursorId,
  });
  final List<UserContentItem> items;
  final bool hasMore;
  final int nextCursorId;
  factory UserContentPage.fromJson(Map<String, dynamic> json) =>
      UserContentPage(
        items: (json['items'] as List? ?? [])
            .map(
              (item) => UserContentItem.fromJson(item as Map<String, dynamic>),
            )
            .toList(),
        hasMore: json['hasMore'] == true,
        nextCursorId: (json['nextCursorId'] as num?)?.toInt() ?? 0,
      );
}

class ContentDeletionResult {
  const ContentDeletionResult({
    required this.contentId,
    required this.success,
    this.message = '',
  });
  final int contentId;
  final bool success;
  final String message;
  factory ContentDeletionResult.fromJson(Map<String, dynamic> json) =>
      ContentDeletionResult(
        contentId: (json['contentId'] as num).toInt(),
        success: json['success'] == true,
        message: json['message'] as String? ?? '',
      );
}
