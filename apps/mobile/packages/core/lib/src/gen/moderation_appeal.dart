/// 审核申诉契约镜像（`POST|GET /api/forum/content/appeals`、
/// `POST /api/forum/moderation/appeals`、`POST /api/forum/moderation/appeal-action`，#1114）。
///
/// OpenAPI 仍是事实源（packages/api-contract/paths/moderation-appeals.yaml）；
/// Dart 生成尚未启用，本文件按其他手写镜像的约定维护：镜像不含作者/裁决者身份字段，
/// 与 Web 生成类型（packages/client/src/gen）逐项对应。
library;

/// 一条申诉记录（提交说明 + 受理/裁决结论）。
class ModerationAppealItem {
  const ModerationAppealItem({
    required this.id,
    required this.kind,
    required this.targetId,
    required this.revisionId,
    required this.topicId,
    required this.postNo,
    required this.title,
    required this.content,
    required this.images,
    required this.reason,
    this.originalReviewReason = '',
    required this.status,
    required this.reviewReason,
    required this.createdAt,
    this.reviewedAt,
  });

  final int id;
  final String kind;
  final int targetId;
  final int revisionId;
  final int topicId;
  final int postNo;
  final String title;
  final String content;
  final List<String> images;
  final String reason;

  /// 被申诉的原审核说明，供复核时对照。
  final String originalReviewReason;

  /// pending / approved / rejected / stale（stale 由服务端按内容现状计算）。
  final String status;
  final String reviewReason;
  final String createdAt;
  final String? reviewedAt;

  factory ModerationAppealItem.fromJson(Map<String, dynamic> json) =>
      ModerationAppealItem(
        id: (json['id'] as num).toInt(),
        kind: json['kind'] as String,
        targetId: (json['targetId'] as num).toInt(),
        revisionId: (json['revisionId'] as num).toInt(),
        topicId: (json['topicId'] as num).toInt(),
        postNo: (json['postNo'] as num).toInt(),
        title: json['title'] as String? ?? '',
        content: json['content'] as String? ?? '',
        images: (json['images'] as List? ?? []).cast<String>(),
        reason: json['reason'] as String? ?? '',
        originalReviewReason: json['originalReviewReason'] as String? ?? '',
        status: json['status'] as String,
        reviewReason: json['reviewReason'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        reviewedAt: json['reviewedAt'] as String?,
      );
}

/// 内容管理列表里最新版本的申诉概况（`MyContentItem.appeal`，ModerationAppealOwnerSummary）。
class ModerationAppealOwnerSummary {
  const ModerationAppealOwnerSummary({
    required this.status,
    required this.reason,
    this.reviewReason = '',
    this.createdAt = '',
    this.reviewedAt,
  });

  /// pending / approved / rejected / stale。
  final String status;
  final String reason;
  final String reviewReason;
  final String createdAt;
  final String? reviewedAt;

  factory ModerationAppealOwnerSummary.fromJson(Map<String, dynamic> json) =>
      ModerationAppealOwnerSummary(
        status: json['status'] as String,
        reason: json['reason'] as String? ?? '',
        reviewReason: json['reviewReason'] as String? ?? '',
        createdAt: json['createdAt'] as String? ?? '',
        reviewedAt: json['reviewedAt'] as String?,
      );
}

/// GET /api/forum/content/appeals 的 data（作者历史）。
class ModerationAppealHistoryResult {
  const ModerationAppealHistoryResult({required this.items});

  final List<ModerationAppealItem> items;

  factory ModerationAppealHistoryResult.fromJson(Map<String, dynamic> json) =>
      ModerationAppealHistoryResult(
        items: (json['items'] as List? ?? [])
            .map(
              (item) => ModerationAppealItem.fromJson(item as Map<String, dynamic>),
            )
            .toList(),
      );
}

/// POST /api/forum/moderation/appeals 的 data（版主队列）。
class ModerationAppealQueueResult {
  const ModerationAppealQueueResult({
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
  });

  final List<ModerationAppealItem> items;
  final int total;
  final int page;
  final int pageSize;

  factory ModerationAppealQueueResult.fromJson(Map<String, dynamic> json) =>
      ModerationAppealQueueResult(
        items: (json['items'] as List? ?? [])
            .map(
              (item) => ModerationAppealItem.fromJson(item as Map<String, dynamic>),
            )
            .toList(),
        total: (json['total'] as num?)?.toInt() ?? 0,
        page: (json['page'] as num?)?.toInt() ?? 1,
        pageSize: (json['pageSize'] as num?)?.toInt() ?? 20,
      );
}

/// POST /api/forum/content/appeals 请求体。
class ModerationAppealSubmitRequest {
  const ModerationAppealSubmitRequest({
    required this.kind,
    required this.id,
    required this.revisionId,
    required this.reason,
  });

  final String kind;
  final int id;
  final int revisionId;
  final String reason;

  Map<String, dynamic> toJson() => {
    'kind': kind,
    'id': id,
    'revisionId': revisionId,
    'reason': reason,
  };
}

/// POST /api/forum/moderation/appeals 请求体。
class ModerationAppealQueueRequest {
  const ModerationAppealQueueRequest({
    this.page = 1,
    this.pageSize = 20,
    this.status = 'pending',
  });

  final int page;
  final int pageSize;

  /// pending / approved / rejected / stale。
  final String status;

  Map<String, dynamic> toJson() => {
    'page': page,
    'pageSize': pageSize,
    'status': status,
  };
}

/// POST /api/forum/moderation/appeal-action 请求体。
class ModerationAppealActionRequest {
  const ModerationAppealActionRequest({
    required this.id,
    required this.approve,
    this.reason = '',
  });

  final int id;
  final bool approve;

  /// 拒绝申诉时必填。
  final String reason;

  Map<String, dynamic> toJson() => {
    'id': id,
    'approve': approve,
    'reason': reason,
  };
}
