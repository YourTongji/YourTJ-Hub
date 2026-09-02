import 'package:freezed_annotation/freezed_annotation.dart';

import 'topic.dart';

part 'course_review.freezed.dart';
part 'course_review.g.dart';

@freezed
abstract class ReviewAuthorPayload with _$ReviewAuthorPayload {
  const factory ReviewAuthorPayload({
    required String kind,
    required String label,
  }) = _ReviewAuthorPayload;

  factory ReviewAuthorPayload.fromJson(Map<String, dynamic> json) =>
      _$ReviewAuthorPayloadFromJson(json);
}

@freezed
abstract class ReviewViewerPayload with _$ReviewViewerPayload {
  const factory ReviewViewerPayload({
    required bool canEdit,
    required bool canDelete,
    required bool isHelpful,
  }) = _ReviewViewerPayload;

  factory ReviewViewerPayload.fromJson(Map<String, dynamic> json) =>
      _$ReviewViewerPayloadFromJson(json);
}

@freezed
abstract class ReviewPayload with _$ReviewPayload {
  const factory ReviewPayload({
    required int id,
    required int offeringId,
    int? rating,
    required String content,
    required String contentHtml,
    required ReviewAuthorPayload author,
    required ReviewViewerPayload viewer,
    required int helpfulCount,
    required String createdAt,
    required String updatedAt,
    double? offeringRatingAvg,
    int? offeringReviewCount,
  }) = _ReviewPayload;

  factory ReviewPayload.fromJson(Map<String, dynamic> json) =>
      _$ReviewPayloadFromJson(json);
}

@freezed
abstract class ReviewListResult with _$ReviewListResult {
  const factory ReviewListResult({
    required List<ReviewPayload> list,
    String? nextCursor,
    required int total,
  }) = _ReviewListResult;

  factory ReviewListResult.fromJson(Map<String, dynamic> json) =>
      _$ReviewListResultFromJson(json);
}

@freezed
abstract class CreateCourseReviewInput with _$CreateCourseReviewInput {
  const factory CreateCourseReviewInput({
    required int offeringId,
    required int rating,
    required String content,
    required bool isAnonymous,
  }) = _CreateCourseReviewInput;

  factory CreateCourseReviewInput.fromJson(Map<String, dynamic> json) =>
      _$CreateCourseReviewInputFromJson(json);
}

@freezed
abstract class UpdateCourseReviewInput with _$UpdateCourseReviewInput {
  const factory UpdateCourseReviewInput({
    int? rating,
    String? content,
    bool? isAnonymous,
  }) = _UpdateCourseReviewInput;

  factory UpdateCourseReviewInput.fromJson(Map<String, dynamic> json) =>
      _$UpdateCourseReviewInputFromJson(json);
}

@freezed
abstract class ModerationCourseReviewReportItem
    with _$ModerationCourseReviewReportItem {
  const factory ModerationCourseReviewReportItem({
    required int id,
    required int reviewId,
    required String reason,
    required String note,
    required String status,
    required String resolution,
    required String excerpt,
    required UserBriefPayload reporter,
    required UserBriefPayload handler,
    required String createdAt,
    String? handledAt,
    required int reportCount,
  }) = _ModerationCourseReviewReportItem;

  factory ModerationCourseReviewReportItem.fromJson(
    Map<String, dynamic> json,
  ) => _$ModerationCourseReviewReportItemFromJson(json);
}

@freezed
abstract class ModerationCourseReviewReportListResponse
    with _$ModerationCourseReviewReportListResponse {
  const factory ModerationCourseReviewReportListResponse({
    required List<ModerationCourseReviewReportItem> items,
    required int nextCursor,
    required bool hasNext,
  }) = _ModerationCourseReviewReportListResponse;

  factory ModerationCourseReviewReportListResponse.fromJson(
    Map<String, dynamic> json,
  ) => _$ModerationCourseReviewReportListResponseFromJson(json);
}

@freezed
abstract class CourseReviewAuthorRevealPayload
    with _$CourseReviewAuthorRevealPayload {
  const factory CourseReviewAuthorRevealPayload({
    required int reviewId,
    int? authorUserId,
    String? username,
    String? nickname,
    required bool isAnonymous,
    required String source,
  }) = _CourseReviewAuthorRevealPayload;

  factory CourseReviewAuthorRevealPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseReviewAuthorRevealPayloadFromJson(json);
}

@freezed
abstract class RelatedCourseItem with _$RelatedCourseItem {
  const factory RelatedCourseItem({
    required int id,
    required String primaryCode,
    required String name,
    required String department,
    String? teacherName,
    List<String>? instructors,
    required double ratingAvg,
    required int ratingCount,
    required int reviewCount,
  }) = _RelatedCourseItem;

  factory RelatedCourseItem.fromJson(Map<String, dynamic> json) =>
      _$RelatedCourseItemFromJson(json);
}

@freezed
abstract class RelationItem with _$RelationItem {
  const factory RelationItem({
    required int relationId,
    required int fromCourseId,
    required String fromName,
    required int toCourseId,
    required String toName,
    required String relationType,
    required String status,
    required String direction,
  }) = _RelationItem;

  factory RelationItem.fromJson(Map<String, dynamic> json) =>
      _$RelationItemFromJson(json);
}

@freezed
abstract class CourseRelatedResult with _$CourseRelatedResult {
  const factory CourseRelatedResult({
    required List<RelatedCourseItem> teacherOtherCourses,
    required List<RelatedCourseItem> sameCourseOtherTeachers,
    // 本卡已确认的沿革关系（approved/merged；原名标注与旧卡跳转）。
    List<RelationItem>? lineage,
  }) = _CourseRelatedResult;

  factory CourseRelatedResult.fromJson(Map<String, dynamic> json) =>
      _$CourseRelatedResultFromJson(json);
}
