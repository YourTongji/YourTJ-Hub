import 'package:freezed_annotation/freezed_annotation.dart';

import 'common.dart';
import 'topic.dart';

part 'notification.freezed.dart';
part 'notification.g.dart';

@freezed
abstract class NotificationPayload with _$NotificationPayload {
  const factory NotificationPayload({
    required int id,
    required String eventType,
    required bool isRead,
    required String createdAt,
    required String title,
    required String content,
    required NotificationActorPayload actor,
    NotificationTopicPayload? topic,
    required NotificationInnerPayload payload,
  }) = _NotificationPayload;

  factory NotificationPayload.fromJson(Map<String, dynamic> json) =>
      _$NotificationPayloadFromJson(json);
}

@freezed
abstract class NotificationActorPayload with _$NotificationActorPayload {
  const factory NotificationActorPayload({
    required int id,
    required String username,
    String? avatarUrl,
  }) = _NotificationActorPayload;

  factory NotificationActorPayload.fromJson(Map<String, dynamic> json) =>
      _$NotificationActorPayloadFromJson(json);
}

@freezed
abstract class NotificationTopicPayload with _$NotificationTopicPayload {
  const factory NotificationTopicPayload({
    required int id,
    required String title,
    required String url,
  }) = _NotificationTopicPayload;

  factory NotificationTopicPayload.fromJson(Map<String, dynamic> json) =>
      _$NotificationTopicPayloadFromJson(json);
}

@freezed
abstract class NotificationInnerPayload with _$NotificationInnerPayload {
  const factory NotificationInnerPayload({
    String? title,
    String? content,
    String? templateKey,
    NotificationTemplateParams? templateParams,
    required int actorId,
    String? actorName,
    int? topicId,
    int? postId,
    String? topicTitle,
    NotificationMetadata? metadata,
  }) = _NotificationInnerPayload;

  factory NotificationInnerPayload.fromJson(Map<String, dynamic> json) =>
      _$NotificationInnerPayloadFromJson(json);
}

@freezed
abstract class NotificationTemplateParams with _$NotificationTemplateParams {
  const factory NotificationTemplateParams({
    String? preview,
    String? followerName,
    String? badgeCode,
    String? badgeName,
  }) = _NotificationTemplateParams;

  factory NotificationTemplateParams.fromJson(Map<String, dynamic> json) =>
      _$NotificationTemplateParamsFromJson(json);
}

@freezed
abstract class NotificationMetadata with _$NotificationMetadata {
  const factory NotificationMetadata({
    String? followerName,
    String? badgeCode,
    String? badgeName,
    String? badgeIconUrl,
    String? profileUrl,
  }) = _NotificationMetadata;

  factory NotificationMetadata.fromJson(Map<String, dynamic> json) =>
      _$NotificationMetadataFromJson(json);
}

@freezed
abstract class NotificationListResponse with _$NotificationListResponse {
  const factory NotificationListResponse({
    required List<NotificationPayload> items,
    required int nextCursor,
    required bool hasNext,
    required int unreadCount,
  }) = _NotificationListResponse;

  factory NotificationListResponse.fromJson(Map<String, dynamic> json) =>
      _$NotificationListResponseFromJson(json);
}

@freezed
abstract class NotificationsPageProps with _$NotificationsPageProps {
  const factory NotificationsPageProps({
    required int total,
    required int unreadCount,
    required List<NotificationPayload> notifications,
    required PaginationPayload pagination,
  }) = _NotificationsPageProps;

  factory NotificationsPageProps.fromJson(Map<String, dynamic> json) =>
      _$NotificationsPagePropsFromJson(json);
}

@freezed
abstract class DraftPayload with _$DraftPayload {
  const factory DraftPayload({
    required int id,
    required String title,
    required String description,
    required String editUrl,
    required int replyCount,
    required int viewCount,
    required int processStatus,
    required String updatedAt,
    required String createdAt,
    required List<CategoryBriefPayload> categories,
  }) = _DraftPayload;

  factory DraftPayload.fromJson(Map<String, dynamic> json) => _$DraftPayloadFromJson(json);
}

@freezed
abstract class DraftsPageProps with _$DraftsPageProps {
  const factory DraftsPageProps({
    required int total,
    required List<DraftPayload> drafts,
    required PaginationPayload pagination,
  }) = _DraftsPageProps;

  factory DraftsPageProps.fromJson(Map<String, dynamic> json) =>
      _$DraftsPagePropsFromJson(json);
}
