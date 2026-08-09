// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$UserCardPayloadImpl _$$UserCardPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserCardPayloadImpl(
  userId: (json['userId'] as num).toInt(),
  username: json['username'] as String,
  nickname: json['nickname'] as String,
  avatarUrl: json['avatarUrl'] as String,
  profileCoverUrl: json['profileCoverUrl'] as String,
  bio: json['bio'] as String,
  signature: json['signature'] as String,
  websiteName: json['websiteName'] as String,
  website: json['website'] as String,
  prestige: (json['prestige'] as num).toInt(),
  externalInformation: (json['externalInformation'] as Map<String, dynamic>)
      .map(
        (k, e) => MapEntry(
          k,
          ExternalLinkPayload.fromJson(e as Map<String, dynamic>),
        ),
      ),
  isAdmin: json['isAdmin'] as bool,
  topicCount: (json['topicCount'] as num).toInt(),
  replyCount: (json['replyCount'] as num).toInt(),
  likeReceivedCount: (json['likeReceivedCount'] as num).toInt(),
  likeGivenCount: (json['likeGivenCount'] as num).toInt(),
  followerCount: (json['followerCount'] as num).toInt(),
  followingCount: (json['followingCount'] as num).toInt(),
  collectionCount: (json['collectionCount'] as num).toInt(),
  isOnline: json['isOnline'] as bool,
  isFollowing: json['isFollowing'] as bool,
  isSelf: json['isSelf'] as bool,
  badges: (json['badges'] as List<dynamic>)
      .map((e) => UserBadgePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  wornBadge: json['wornBadge'] == null
      ? null
      : UserBadgePayload.fromJson(json['wornBadge'] as Map<String, dynamic>),
  lastActiveTime: json['lastActiveTime'] as String,
  createdAt: json['createdAt'] as String,
);

Map<String, dynamic> _$$UserCardPayloadImplToJson(
  _$UserCardPayloadImpl instance,
) => <String, dynamic>{
  'userId': instance.userId,
  'username': instance.username,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'profileCoverUrl': instance.profileCoverUrl,
  'bio': instance.bio,
  'signature': instance.signature,
  'websiteName': instance.websiteName,
  'website': instance.website,
  'prestige': instance.prestige,
  'externalInformation': instance.externalInformation,
  'isAdmin': instance.isAdmin,
  'topicCount': instance.topicCount,
  'replyCount': instance.replyCount,
  'likeReceivedCount': instance.likeReceivedCount,
  'likeGivenCount': instance.likeGivenCount,
  'followerCount': instance.followerCount,
  'followingCount': instance.followingCount,
  'collectionCount': instance.collectionCount,
  'isOnline': instance.isOnline,
  'isFollowing': instance.isFollowing,
  'isSelf': instance.isSelf,
  'badges': instance.badges,
  'wornBadge': instance.wornBadge,
  'lastActiveTime': instance.lastActiveTime,
  'createdAt': instance.createdAt,
};

_$ExternalLinkPayloadImpl _$$ExternalLinkPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ExternalLinkPayloadImpl(link: json['link'] as String?);

Map<String, dynamic> _$$ExternalLinkPayloadImplToJson(
  _$ExternalLinkPayloadImpl instance,
) => <String, dynamic>{'link': instance.link};

_$UserActivityPayloadImpl _$$UserActivityPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserActivityPayloadImpl(
  id: (json['id'] as num).toInt(),
  action: (json['action'] as num).toInt(),
  subjectType: json['subjectType'] as String,
  subjectId: (json['subjectId'] as num).toInt(),
  contentPreview: json['contentPreview'] as String,
  url: json['url'] as String,
  label: json['label'] as String,
  createdAt: json['createdAt'] as String,
);

Map<String, dynamic> _$$UserActivityPayloadImplToJson(
  _$UserActivityPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'action': instance.action,
  'subjectType': instance.subjectType,
  'subjectId': instance.subjectId,
  'contentPreview': instance.contentPreview,
  'url': instance.url,
  'label': instance.label,
  'createdAt': instance.createdAt,
};

_$UserLikePayloadImpl _$$UserLikePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserLikePayloadImpl(
  id: (json['id'] as num).toInt(),
  topicId: (json['topicId'] as num).toInt(),
  title: json['title'] as String,
  url: json['url'] as String,
  likedAt: json['likedAt'] as String,
);

Map<String, dynamic> _$$UserLikePayloadImplToJson(
  _$UserLikePayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'topicId': instance.topicId,
  'title': instance.title,
  'url': instance.url,
  'likedAt': instance.likedAt,
};

_$UserBookmarkPayloadImpl _$$UserBookmarkPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserBookmarkPayloadImpl(
  id: (json['id'] as num).toInt(),
  type: json['type'] as String,
  topicId: (json['topicId'] as num).toInt(),
  postId: (json['postId'] as num?)?.toInt(),
  postNo: (json['postNo'] as num?)?.toInt(),
  title: json['title'] as String,
  excerpt: json['excerpt'] as String?,
  url: json['url'] as String,
  bookmarkedAt: json['bookmarkedAt'] as String,
);

Map<String, dynamic> _$$UserBookmarkPayloadImplToJson(
  _$UserBookmarkPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'type': instance.type,
  'topicId': instance.topicId,
  'postId': instance.postId,
  'postNo': instance.postNo,
  'title': instance.title,
  'excerpt': instance.excerpt,
  'url': instance.url,
  'bookmarkedAt': instance.bookmarkedAt,
};

_$UserConnectionPayloadImpl _$$UserConnectionPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserConnectionPayloadImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  nickname: json['nickname'] as String,
  avatarUrl: json['avatarUrl'] as String,
  bio: json['bio'] as String,
  url: json['url'] as String,
);

Map<String, dynamic> _$$UserConnectionPayloadImplToJson(
  _$UserConnectionPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'bio': instance.bio,
  'url': instance.url,
};

_$UserProfilePropsImpl _$$UserProfilePropsImplFromJson(
  Map<String, dynamic> json,
) => _$UserProfilePropsImpl(
  user: UserCardPayload.fromJson(json['user'] as Map<String, dynamic>),
  section: json['section'] as String,
  activityTab: json['activityTab'] as String,
  tabs: (json['tabs'] as List<dynamic>)
      .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  activityTabs: (json['activityTabs'] as List<dynamic>)
      .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
  badges: (json['badges'] as List<dynamic>)
      .map((e) => UserBadgePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  topics: (json['topics'] as List<dynamic>)
      .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  activities: (json['activities'] as List<dynamic>)
      .map((e) => UserActivityPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  likes: (json['likes'] as List<dynamic>)
      .map((e) => UserLikePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  bookmarks: (json['bookmarks'] as List<dynamic>)
      .map((e) => UserBookmarkPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  following: (json['following'] as List<dynamic>)
      .map((e) => UserConnectionPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  followers: (json['followers'] as List<dynamic>)
      .map((e) => UserConnectionPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  isOwnProfile: json['isOwnProfile'] as bool,
  canMessage: json['canMessage'] as bool,
  canFollow: json['canFollow'] as bool,
  messageUrl: json['messageUrl'] as String,
  settingsUrl: json['settingsUrl'] as String,
);

Map<String, dynamic> _$$UserProfilePropsImplToJson(
  _$UserProfilePropsImpl instance,
) => <String, dynamic>{
  'user': instance.user,
  'section': instance.section,
  'activityTab': instance.activityTab,
  'tabs': instance.tabs,
  'activityTabs': instance.activityTabs,
  'pagination': instance.pagination,
  'badges': instance.badges,
  'topics': instance.topics,
  'activities': instance.activities,
  'likes': instance.likes,
  'bookmarks': instance.bookmarks,
  'following': instance.following,
  'followers': instance.followers,
  'isOwnProfile': instance.isOwnProfile,
  'canMessage': instance.canMessage,
  'canFollow': instance.canFollow,
  'messageUrl': instance.messageUrl,
  'settingsUrl': instance.settingsUrl,
};

_$SettingsUserPayloadImpl _$$SettingsUserPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SettingsUserPayloadImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  email: json['email'] as String,
  nickname: json['nickname'] as String,
  locale: json['locale'] as String,
  avatarUrl: json['avatarUrl'] as String,
  profileCoverUrl: json['profileCoverUrl'] as String,
  bio: json['bio'] as String,
  signature: json['signature'] as String,
  websiteName: json['websiteName'] as String,
  website: json['website'] as String,
  prestige: (json['prestige'] as num).toInt(),
  createdAt: json['createdAt'] as String,
  externalInformation: (json['externalInformation'] as Map<String, dynamic>)
      .map(
        (k, e) => MapEntry(
          k,
          ExternalLinkPayload.fromJson(e as Map<String, dynamic>),
        ),
      ),
  wornBadgeCode: json['wornBadgeCode'] as String,
  badges: (json['badges'] as List<dynamic>)
      .map((e) => UserBadgePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  wearableBadges: (json['wearableBadges'] as List<dynamic>)
      .map((e) => UserBadgePayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  wornBadge: json['wornBadge'] == null
      ? null
      : UserBadgePayload.fromJson(json['wornBadge'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$SettingsUserPayloadImplToJson(
  _$SettingsUserPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'email': instance.email,
  'nickname': instance.nickname,
  'locale': instance.locale,
  'avatarUrl': instance.avatarUrl,
  'profileCoverUrl': instance.profileCoverUrl,
  'bio': instance.bio,
  'signature': instance.signature,
  'websiteName': instance.websiteName,
  'website': instance.website,
  'prestige': instance.prestige,
  'createdAt': instance.createdAt,
  'externalInformation': instance.externalInformation,
  'wornBadgeCode': instance.wornBadgeCode,
  'badges': instance.badges,
  'wearableBadges': instance.wearableBadges,
  'wornBadge': instance.wornBadge,
};

_$SettingsStatsPayloadImpl _$$SettingsStatsPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SettingsStatsPayloadImpl(
  topicCount: (json['topicCount'] as num).toInt(),
  replyCount: (json['replyCount'] as num).toInt(),
  followerCount: (json['followerCount'] as num).toInt(),
  followingCount: (json['followingCount'] as num).toInt(),
  likeReceivedCount: (json['likeReceivedCount'] as num).toInt(),
  likeGivenCount: (json['likeGivenCount'] as num).toInt(),
  collectionCount: (json['collectionCount'] as num).toInt(),
  createdAt: json['createdAt'] as String,
);

Map<String, dynamic> _$$SettingsStatsPayloadImplToJson(
  _$SettingsStatsPayloadImpl instance,
) => <String, dynamic>{
  'topicCount': instance.topicCount,
  'replyCount': instance.replyCount,
  'followerCount': instance.followerCount,
  'followingCount': instance.followingCount,
  'likeReceivedCount': instance.likeReceivedCount,
  'likeGivenCount': instance.likeGivenCount,
  'collectionCount': instance.collectionCount,
  'createdAt': instance.createdAt,
};

_$SettingsPagePropsImpl _$$SettingsPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$SettingsPagePropsImpl(
  user: SettingsUserPayload.fromJson(json['user'] as Map<String, dynamic>),
  stats: SettingsStatsPayload.fromJson(json['stats'] as Map<String, dynamic>),
  tabs: (json['tabs'] as List<dynamic>)
      .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$SettingsPagePropsImplToJson(
  _$SettingsPagePropsImpl instance,
) => <String, dynamic>{
  'user': instance.user,
  'stats': instance.stats,
  'tabs': instance.tabs,
};
