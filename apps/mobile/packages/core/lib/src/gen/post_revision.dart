import 'topic.dart';

/// Mirrors the controlled post-revisions contract, including masked snapshots.
class PostRevision {
  const PostRevision({
    required this.version,
    required this.editor,
    required this.content,
    required this.renderedHTML,
    required this.processStatus,
    required this.createdAt,
  });
  final int version;
  final UserBriefPayload editor;
  final String content;
  final String renderedHTML;
  final int processStatus;
  final String createdAt;
  factory PostRevision.fromJson(Map<String, dynamic> json) => PostRevision(
    version: (json['version'] as num).toInt(),
    editor: UserBriefPayload.fromJson(json['editor'] as Map<String, dynamic>),
    content: json['content'] as String,
    renderedHTML: json['renderedHTML'] as String,
    processStatus: (json['processStatus'] as num).toInt(),
    createdAt: json['createdAt'] as String,
  );
}

class PostRevisionPage {
  const PostRevisionPage({
    required this.postId,
    required this.versions,
    required this.hasMore,
    required this.beforeVersion,
  });
  final int postId;
  final List<PostRevision> versions;
  final bool hasMore;
  final int beforeVersion;
  factory PostRevisionPage.fromJson(Map<String, dynamic> json) =>
      PostRevisionPage(
        postId: (json['postId'] as num).toInt(),
        versions: (json['versions'] as List)
            .map((item) => PostRevision.fromJson(item as Map<String, dynamic>))
            .toList(),
        hasMore: json['hasMore'] as bool,
        beforeVersion: (json['beforeVersion'] as num).toInt(),
      );
}
