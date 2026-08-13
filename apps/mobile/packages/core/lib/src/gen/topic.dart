import 'package:freezed_annotation/freezed_annotation.dart';

part 'topic.freezed.dart';
part 'topic.g.dart';

@freezed
abstract class UserBriefPayload with _$UserBriefPayload {
  const factory UserBriefPayload({
    required int id,
    required String username,
    String? nickname,
    required String avatarUrl,
    UserBadgePayload? wornBadge,
  }) = _UserBriefPayload;

  factory UserBriefPayload.fromJson(Map<String, dynamic> json) =>
      _$UserBriefPayloadFromJson(json);
}

@freezed
abstract class BadgePayload with _$BadgePayload {
  const factory BadgePayload({
    required String code,
    required String type,
    required String grantMode,
    required String name,
    required String description,
    required String iconType,
    required String iconKey,
    required String iconUrl,
    required String color,
    required String level,
    required bool isEnabled,
    required bool isWearable,
    required int sortOrder,
  }) = _BadgePayload;

  factory BadgePayload.fromJson(Map<String, dynamic> json) =>
      _$BadgePayloadFromJson(json);
}

@freezed
abstract class UserBadgePayload with _$UserBadgePayload {
  const factory UserBadgePayload({
    required String code,
    required String type,
    required String grantMode,
    required String name,
    required String description,
    required String iconType,
    required String iconKey,
    required String iconUrl,
    required String color,
    required String level,
    required bool isEnabled,
    required bool isWearable,
    required int sortOrder,
    required String source,
    required String reason,
    required String grantedAt,
  }) = _UserBadgePayload;

  factory UserBadgePayload.fromJson(Map<String, dynamic> json) =>
      _$UserBadgePayloadFromJson(json);
}

@freezed
abstract class CategoryBriefPayload with _$CategoryBriefPayload {
  const factory CategoryBriefPayload({
    required int id,
    required String name,
    required String url,
    required String color,
  }) = _CategoryBriefPayload;

  factory CategoryBriefPayload.fromJson(Map<String, dynamic> json) =>
      _$CategoryBriefPayloadFromJson(json);
}

@freezed
abstract class TopicPayload with _$TopicPayload {
  const factory TopicPayload({
    required int id,
    required String title,
    required String description,
    String? firstImageUrl,
    List<String>? images,
    required String url,
    required UserBriefPayload author,
    required List<UserBriefPayload> participants,
    required List<CategoryBriefPayload> categories,
    required int replyCount,
    required int viewCount,
    required int pinWeight,
    required int processStatus,
    required String activityText,
    required String lastUpdateTime,
    bool? unseen,
  }) = _TopicPayload;

  factory TopicPayload.fromJson(Map<String, dynamic> json) =>
      _$TopicPayloadFromJson(json);
}

@freezed
abstract class TopicDetailPayload with _$TopicDetailPayload {
  const factory TopicDetailPayload({
    required int id,
    required String title,
    required String description,
    required String url,
    required int topicStatus,
    required int processStatus,
    required UserBriefPayload author,
    required List<UserBriefPayload> participants,
    required List<CategoryBriefPayload> categories,
    required int replyCount,
    required int maxPostNo,
    required int viewCount,
    required int likeCount,
    required bool isLiked,
    required bool isBookmarked,
    required bool isWatched,
    required String createdAt,
    required String updatedAt,
  }) = _TopicDetailPayload;

  factory TopicDetailPayload.fromJson(Map<String, dynamic> json) =>
      _$TopicDetailPayloadFromJson(json);
}

@freezed
abstract class PostPayload with _$PostPayload {
  const factory PostPayload({
    required int id,
    required int topicId,
    required int postNo,
    required String content,
    required String renderedContent,
    required int processStatus,
    required bool isHidden,
    required bool canModerate,
    required UserBriefPayload author,
    required String createdAt,
    int? replyToPostId,
    int? replyToUserId,
    String? replyToUsername,
    required bool isOwnPost,
    String? updatedAt,
    UserBriefPayload? lastEditor,
    String? lastEditedAt,
    required int revisionCount,
    required int likeCount,
    required bool isLiked,
    required bool isBookmarked,
  }) = _PostPayload;

  factory PostPayload.fromJson(Map<String, dynamic> json) =>
      _$PostPayloadFromJson(json);
}

@freezed
abstract class ReplyTargetPayload with _$ReplyTargetPayload {
  const factory ReplyTargetPayload({
    required int id,
    int? postNo,
    required UserBriefPayload author,
    String? renderedContent,
    bool? unavailable,
  }) = _ReplyTargetPayload;

  factory ReplyTargetPayload.fromJson(Map<String, dynamic> json) =>
      _$ReplyTargetPayloadFromJson(json);
}

@freezed
abstract class PostWindowPayload with _$PostWindowPayload {
  const factory PostWindowPayload({
    required List<PostPayload> posts,
    required List<ReplyTargetPayload> replyTargets,
    int? anchorPostId,
    int? beforePostNo,
    int? afterPostNo,
    required bool hasBefore,
    required bool hasAfter,
    required int total,
    required int maxPostNo,
  }) = _PostWindowPayload;

  factory PostWindowPayload.fromJson(Map<String, dynamic> json) =>
      _$PostWindowPayloadFromJson(json);
}

@freezed
abstract class TopicDetailProps with _$TopicDetailProps {
  const factory TopicDetailProps({
    required TopicDetailPayload topic,
    required PostWindowPayload postStream,
    required List<TopicPayload> hotTopics,
    required TopicDetailPermissions permissions,
  }) = _TopicDetailProps;

  factory TopicDetailProps.fromJson(Map<String, dynamic> json) =>
      _$TopicDetailPropsFromJson(json);
}

@freezed
abstract class TopicDetailPermissions with _$TopicDetailPermissions {
  const factory TopicDetailPermissions({
    required bool isOwnTopic,
    required bool canPost,
    required bool canModerateTopic,
  }) = _TopicDetailPermissions;

  factory TopicDetailPermissions.fromJson(Map<String, dynamic> json) =>
      _$TopicDetailPermissionsFromJson(json);
}
