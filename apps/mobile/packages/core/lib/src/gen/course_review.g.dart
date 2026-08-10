// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'course_review.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ReviewAuthorPayloadImpl _$$ReviewAuthorPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ReviewAuthorPayloadImpl(
  kind: json['kind'] as String,
  label: json['label'] as String,
);

Map<String, dynamic> _$$ReviewAuthorPayloadImplToJson(
  _$ReviewAuthorPayloadImpl instance,
) => <String, dynamic>{'kind': instance.kind, 'label': instance.label};

_$ReviewViewerPayloadImpl _$$ReviewViewerPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ReviewViewerPayloadImpl(
  canEdit: json['canEdit'] as bool,
  canDelete: json['canDelete'] as bool,
  isHelpful: json['isHelpful'] as bool,
);

Map<String, dynamic> _$$ReviewViewerPayloadImplToJson(
  _$ReviewViewerPayloadImpl instance,
) => <String, dynamic>{
  'canEdit': instance.canEdit,
  'canDelete': instance.canDelete,
  'isHelpful': instance.isHelpful,
};

_$ReviewPayloadImpl _$$ReviewPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ReviewPayloadImpl(
  id: (json['id'] as num).toInt(),
  offeringId: (json['offeringId'] as num).toInt(),
  rating: (json['rating'] as num?)?.toInt(),
  contentHtml: json['contentHtml'] as String,
  author: ReviewAuthorPayload.fromJson(json['author'] as Map<String, dynamic>),
  viewer: ReviewViewerPayload.fromJson(json['viewer'] as Map<String, dynamic>),
  helpfulCount: (json['helpfulCount'] as num).toInt(),
  createdAt: json['createdAt'] as String,
  updatedAt: json['updatedAt'] as String,
);

Map<String, dynamic> _$$ReviewPayloadImplToJson(_$ReviewPayloadImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'offeringId': instance.offeringId,
      'rating': instance.rating,
      'contentHtml': instance.contentHtml,
      'author': instance.author,
      'viewer': instance.viewer,
      'helpfulCount': instance.helpfulCount,
      'createdAt': instance.createdAt,
      'updatedAt': instance.updatedAt,
    };

_$CreateCourseReviewInputImpl _$$CreateCourseReviewInputImplFromJson(
  Map<String, dynamic> json,
) => _$CreateCourseReviewInputImpl(
  offeringId: (json['offeringId'] as num).toInt(),
  rating: (json['rating'] as num).toInt(),
  content: json['content'] as String,
  isAnonymous: json['isAnonymous'] as bool,
);

Map<String, dynamic> _$$CreateCourseReviewInputImplToJson(
  _$CreateCourseReviewInputImpl instance,
) => <String, dynamic>{
  'offeringId': instance.offeringId,
  'rating': instance.rating,
  'content': instance.content,
  'isAnonymous': instance.isAnonymous,
};

_$UpdateCourseReviewInputImpl _$$UpdateCourseReviewInputImplFromJson(
  Map<String, dynamic> json,
) => _$UpdateCourseReviewInputImpl(
  rating: (json['rating'] as num?)?.toInt(),
  content: json['content'] as String?,
  isAnonymous: json['isAnonymous'] as bool?,
);

Map<String, dynamic> _$$UpdateCourseReviewInputImplToJson(
  _$UpdateCourseReviewInputImpl instance,
) => <String, dynamic>{
  'rating': instance.rating,
  'content': instance.content,
  'isAnonymous': instance.isAnonymous,
};

_$ModerationCourseReviewReportItemImpl
_$$ModerationCourseReviewReportItemImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationCourseReviewReportItemImpl(
  id: (json['id'] as num).toInt(),
  reviewId: (json['reviewId'] as num).toInt(),
  reason: json['reason'] as String,
  note: json['note'] as String,
  status: json['status'] as String,
  resolution: json['resolution'] as String,
  excerpt: json['excerpt'] as String,
  reporter: UserBriefPayload.fromJson(json['reporter'] as Map<String, dynamic>),
  handler: UserBriefPayload.fromJson(json['handler'] as Map<String, dynamic>),
  createdAt: json['createdAt'] as String,
  handledAt: json['handledAt'] as String?,
  reportCount: (json['reportCount'] as num).toInt(),
);

Map<String, dynamic> _$$ModerationCourseReviewReportItemImplToJson(
  _$ModerationCourseReviewReportItemImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'reviewId': instance.reviewId,
  'reason': instance.reason,
  'note': instance.note,
  'status': instance.status,
  'resolution': instance.resolution,
  'excerpt': instance.excerpt,
  'reporter': instance.reporter,
  'handler': instance.handler,
  'createdAt': instance.createdAt,
  'handledAt': instance.handledAt,
  'reportCount': instance.reportCount,
};

_$ModerationCourseReviewReportListResponseImpl
_$$ModerationCourseReviewReportListResponseImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationCourseReviewReportListResponseImpl(
  items: (json['items'] as List<dynamic>)
      .map(
        (e) => ModerationCourseReviewReportItem.fromJson(
          e as Map<String, dynamic>,
        ),
      )
      .toList(),
  nextCursor: (json['nextCursor'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$ModerationCourseReviewReportListResponseImplToJson(
  _$ModerationCourseReviewReportListResponseImpl instance,
) => <String, dynamic>{
  'items': instance.items,
  'nextCursor': instance.nextCursor,
  'hasNext': instance.hasNext,
};

_$CourseReviewAuthorRevealPayloadImpl
_$$CourseReviewAuthorRevealPayloadImplFromJson(Map<String, dynamic> json) =>
    _$CourseReviewAuthorRevealPayloadImpl(
      reviewId: (json['reviewId'] as num).toInt(),
      authorUserId: (json['authorUserId'] as num?)?.toInt(),
      username: json['username'] as String?,
      nickname: json['nickname'] as String?,
      isAnonymous: json['isAnonymous'] as bool,
      source: json['source'] as String,
    );

Map<String, dynamic> _$$CourseReviewAuthorRevealPayloadImplToJson(
  _$CourseReviewAuthorRevealPayloadImpl instance,
) => <String, dynamic>{
  'reviewId': instance.reviewId,
  'authorUserId': instance.authorUserId,
  'username': instance.username,
  'nickname': instance.nickname,
  'isAnonymous': instance.isAnonymous,
  'source': instance.source,
};
