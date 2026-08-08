import 'package:freezed_annotation/freezed_annotation.dart';

import 'common.dart';
import 'topic.dart';

part 'user.freezed.dart';
part 'user.g.dart';

@freezed
abstract class UserCardPayload with _$UserCardPayload {
  const factory UserCardPayload({
    required int userId,
    required String username,
    required String nickname,
    required String avatarUrl,
    required String profileCoverUrl,
    required String bio,
    required String signature,
    required String websiteName,
    required String website,
    required int prestige,
    required Map<String, ExternalLinkPayload> externalInformation,
    required bool isAdmin,
    required int topicCount,
    required int replyCount,
    required int likeReceivedCount,
    required int likeGivenCount,
    required int followerCount,
    required int followingCount,
    required int collectionCount,
    required bool isOnline,
    required bool isFollowing,
    required bool isSelf,
    required List<UserBadgePayload> badges,
    UserBadgePayload? wornBadge,
    required String lastActiveTime,
    required String createdAt,
  }) = _UserCardPayload;

  factory UserCardPayload.fromJson(Map<String, dynamic> json) =>
      _$UserCardPayloadFromJson(json);
}

@freezed
abstract class ExternalLinkPayload with _$ExternalLinkPayload {
  const factory ExternalLinkPayload({
    String? link,
  }) = _ExternalLinkPayload;

  factory ExternalLinkPayload.fromJson(Map<String, dynamic> json) =>
      _$ExternalLinkPayloadFromJson(json);
}

@freezed
abstract class UserActivityPayload with _$UserActivityPayload {
  const factory UserActivityPayload({
    required int id,
    required int action,
    required String subjectType,
    required int subjectId,
    required String contentPreview,
    required String url,
    required String label,
    required String createdAt,
  }) = _UserActivityPayload;

  factory UserActivityPayload.fromJson(Map<String, dynamic> json) =>
      _$UserActivityPayloadFromJson(json);
}

@freezed
abstract class UserLikePayload with _$UserLikePayload {
  const factory UserLikePayload({
    required int id,
    required int topicId,
    required String title,
    required String url,
    required String likedAt,
  }) = _UserLikePayload;

  factory UserLikePayload.fromJson(Map<String, dynamic> json) =>
      _$UserLikePayloadFromJson(json);
}

@freezed
abstract class UserBookmarkPayload with _$UserBookmarkPayload {
  const factory UserBookmarkPayload({
    required int id,
    required String type,
    required int topicId,
    int? postId,
    int? postNo,
    required String title,
    String? excerpt,
    required String url,
    required String bookmarkedAt,
  }) = _UserBookmarkPayload;

  factory UserBookmarkPayload.fromJson(Map<String, dynamic> json) =>
      _$UserBookmarkPayloadFromJson(json);
}

@freezed
abstract class UserConnectionPayload with _$UserConnectionPayload {
  const factory UserConnectionPayload({
    required int id,
    required String username,
    required String nickname,
    required String avatarUrl,
    required String bio,
    required String url,
  }) = _UserConnectionPayload;

  factory UserConnectionPayload.fromJson(Map<String, dynamic> json) =>
      _$UserConnectionPayloadFromJson(json);
}

@freezed
abstract class UserProfileProps with _$UserProfileProps {
  const factory UserProfileProps({
    required UserCardPayload user,
    required String section,
    required String activityTab,
    required List<TabItemPayload> tabs,
    required List<TabItemPayload> activityTabs,
    required PaginationPayload pagination,
    required List<UserBadgePayload> badges,
    required List<TopicPayload> topics,
    required List<UserActivityPayload> activities,
    required List<UserLikePayload> likes,
    required List<UserBookmarkPayload> bookmarks,
    required List<UserConnectionPayload> following,
    required List<UserConnectionPayload> followers,
    required bool isOwnProfile,
    required bool canMessage,
    required bool canFollow,
    required String messageUrl,
    required String settingsUrl,
  }) = _UserProfileProps;

  factory UserProfileProps.fromJson(Map<String, dynamic> json) =>
      _$UserProfilePropsFromJson(json);
}

@freezed
abstract class SettingsUserPayload with _$SettingsUserPayload {
  const factory SettingsUserPayload({
    required int id,
    required String username,
    required String email,
    required String nickname,
    required String locale,
    required String avatarUrl,
    required String profileCoverUrl,
    required String bio,
    required String signature,
    required String websiteName,
    required String website,
    required int prestige,
    required String createdAt,
    required Map<String, ExternalLinkPayload> externalInformation,
    required String wornBadgeCode,
    required List<UserBadgePayload> badges,
    required List<UserBadgePayload> wearableBadges,
    UserBadgePayload? wornBadge,
  }) = _SettingsUserPayload;

  factory SettingsUserPayload.fromJson(Map<String, dynamic> json) =>
      _$SettingsUserPayloadFromJson(json);
}

@freezed
abstract class SettingsStatsPayload with _$SettingsStatsPayload {
  const factory SettingsStatsPayload({
    required int topicCount,
    required int replyCount,
    required int followerCount,
    required int followingCount,
    required int likeReceivedCount,
    required int likeGivenCount,
    required int collectionCount,
    required String createdAt,
  }) = _SettingsStatsPayload;

  factory SettingsStatsPayload.fromJson(Map<String, dynamic> json) =>
      _$SettingsStatsPayloadFromJson(json);
}

@freezed
abstract class SettingsPageProps with _$SettingsPageProps {
  const factory SettingsPageProps({
    required SettingsUserPayload user,
    required SettingsStatsPayload stats,
    required List<TabItemPayload> tabs,
  }) = _SettingsPageProps;

  factory SettingsPageProps.fromJson(Map<String, dynamic> json) =>
      _$SettingsPagePropsFromJson(json);
}
