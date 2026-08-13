// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'topic.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$UserBriefPayloadImpl _$$UserBriefPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserBriefPayloadImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  nickname: json['nickname'] as String?,
  avatarUrl: json['avatarUrl'] as String,
  wornBadge: json['wornBadge'] == null
      ? null
      : UserBadgePayload.fromJson(json['wornBadge'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$UserBriefPayloadImplToJson(
  _$UserBriefPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'wornBadge': instance.wornBadge,
};

_$BadgePayloadImpl _$$BadgePayloadImplFromJson(Map<String, dynamic> json) =>
    _$BadgePayloadImpl(
      code: json['code'] as String,
      type: json['type'] as String,
      grantMode: json['grantMode'] as String,
      name: json['name'] as String,
      description: json['description'] as String,
      iconType: json['iconType'] as String,
      iconKey: json['iconKey'] as String,
      iconUrl: json['iconUrl'] as String,
      color: json['color'] as String,
      level: json['level'] as String,
      isEnabled: json['isEnabled'] as bool,
      isWearable: json['isWearable'] as bool,
      sortOrder: (json['sortOrder'] as num).toInt(),
    );

Map<String, dynamic> _$$BadgePayloadImplToJson(_$BadgePayloadImpl instance) =>
    <String, dynamic>{
      'code': instance.code,
      'type': instance.type,
      'grantMode': instance.grantMode,
      'name': instance.name,
      'description': instance.description,
      'iconType': instance.iconType,
      'iconKey': instance.iconKey,
      'iconUrl': instance.iconUrl,
      'color': instance.color,
      'level': instance.level,
      'isEnabled': instance.isEnabled,
      'isWearable': instance.isWearable,
      'sortOrder': instance.sortOrder,
    };

_$UserBadgePayloadImpl _$$UserBadgePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserBadgePayloadImpl(
  code: json['code'] as String,
  type: json['type'] as String,
  grantMode: json['grantMode'] as String,
  name: json['name'] as String,
  description: json['description'] as String,
  iconType: json['iconType'] as String,
  iconKey: json['iconKey'] as String,
  iconUrl: json['iconUrl'] as String,
  color: json['color'] as String,
  level: json['level'] as String,
  isEnabled: json['isEnabled'] as bool,
  isWearable: json['isWearable'] as bool,
  sortOrder: (json['sortOrder'] as num).toInt(),
  source: json['source'] as String,
  reason: json['reason'] as String,
  grantedAt: json['grantedAt'] as String,
);

Map<String, dynamic> _$$UserBadgePayloadImplToJson(
  _$UserBadgePayloadImpl instance,
) => <String, dynamic>{
  'code': instance.code,
  'type': instance.type,
  'grantMode': instance.grantMode,
  'name': instance.name,
  'description': instance.description,
  'iconType': instance.iconType,
  'iconKey': instance.iconKey,
  'iconUrl': instance.iconUrl,
  'color': instance.color,
  'level': instance.level,
  'isEnabled': instance.isEnabled,
  'isWearable': instance.isWearable,
  'sortOrder': instance.sortOrder,
  'source': instance.source,
  'reason': instance.reason,
  'grantedAt': instance.grantedAt,
};

_$CategoryBriefPayloadImpl _$$CategoryBriefPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CategoryBriefPayloadImpl(
  id: (json['id'] as num).toInt(),
  name: json['name'] as String,
  url: json['url'] as String,
  color: json['color'] as String,
);

Map<String, dynamic> _$$CategoryBriefPayloadImplToJson(
  _$CategoryBriefPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'url': instance.url,
  'color': instance.color,
};

_$TopicPayloadImpl _$$TopicPayloadImplFromJson(Map<String, dynamic> json) =>
    _$TopicPayloadImpl(
      id: (json['id'] as num).toInt(),
      title: json['title'] as String,
      description: json['description'] as String,
      firstImageUrl: json['firstImageUrl'] as String?,
      images: (json['images'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList(),
      url: json['url'] as String,
      author: UserBriefPayload.fromJson(json['author'] as Map<String, dynamic>),
      participants: (json['participants'] as List<dynamic>)
          .map((e) => UserBriefPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      categories: (json['categories'] as List<dynamic>)
          .map((e) => CategoryBriefPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      replyCount: (json['replyCount'] as num).toInt(),
      viewCount: (json['viewCount'] as num).toInt(),
      pinWeight: (json['pinWeight'] as num).toInt(),
      processStatus: (json['processStatus'] as num).toInt(),
      activityText: json['activityText'] as String,
      lastUpdateTime: json['lastUpdateTime'] as String,
      unseen: json['unseen'] as bool?,
    );

Map<String, dynamic> _$$TopicPayloadImplToJson(_$TopicPayloadImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'description': instance.description,
      'firstImageUrl': instance.firstImageUrl,
      'images': instance.images,
      'url': instance.url,
      'author': instance.author,
      'participants': instance.participants,
      'categories': instance.categories,
      'replyCount': instance.replyCount,
      'viewCount': instance.viewCount,
      'pinWeight': instance.pinWeight,
      'processStatus': instance.processStatus,
      'activityText': instance.activityText,
      'lastUpdateTime': instance.lastUpdateTime,
      'unseen': instance.unseen,
    };

_$TopicDetailPayloadImpl _$$TopicDetailPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$TopicDetailPayloadImpl(
  id: (json['id'] as num).toInt(),
  title: json['title'] as String,
  description: json['description'] as String,
  url: json['url'] as String,
  topicStatus: (json['topicStatus'] as num).toInt(),
  processStatus: (json['processStatus'] as num).toInt(),
  author: UserBriefPayload.fromJson(json['author'] as Map<String, dynamic>),
  participants: (json['participants'] as List<dynamic>)
      .map((e) => UserBriefPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  categories: (json['categories'] as List<dynamic>)
      .map((e) => CategoryBriefPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  replyCount: (json['replyCount'] as num).toInt(),
  maxPostNo: (json['maxPostNo'] as num).toInt(),
  viewCount: (json['viewCount'] as num).toInt(),
  likeCount: (json['likeCount'] as num).toInt(),
  isLiked: json['isLiked'] as bool,
  isBookmarked: json['isBookmarked'] as bool,
  isWatched: json['isWatched'] as bool,
  createdAt: json['createdAt'] as String,
  updatedAt: json['updatedAt'] as String,
);

Map<String, dynamic> _$$TopicDetailPayloadImplToJson(
  _$TopicDetailPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'title': instance.title,
  'description': instance.description,
  'url': instance.url,
  'topicStatus': instance.topicStatus,
  'processStatus': instance.processStatus,
  'author': instance.author,
  'participants': instance.participants,
  'categories': instance.categories,
  'replyCount': instance.replyCount,
  'maxPostNo': instance.maxPostNo,
  'viewCount': instance.viewCount,
  'likeCount': instance.likeCount,
  'isLiked': instance.isLiked,
  'isBookmarked': instance.isBookmarked,
  'isWatched': instance.isWatched,
  'createdAt': instance.createdAt,
  'updatedAt': instance.updatedAt,
};

_$PostPayloadImpl _$$PostPayloadImplFromJson(Map<String, dynamic> json) =>
    _$PostPayloadImpl(
      id: (json['id'] as num).toInt(),
      topicId: (json['topicId'] as num).toInt(),
      postNo: (json['postNo'] as num).toInt(),
      content: json['content'] as String,
      renderedContent: json['renderedContent'] as String,
      processStatus: (json['processStatus'] as num).toInt(),
      isHidden: json['isHidden'] as bool,
      canModerate: json['canModerate'] as bool,
      author: UserBriefPayload.fromJson(json['author'] as Map<String, dynamic>),
      createdAt: json['createdAt'] as String,
      replyToPostId: (json['replyToPostId'] as num?)?.toInt(),
      replyToUserId: (json['replyToUserId'] as num?)?.toInt(),
      replyToUsername: json['replyToUsername'] as String?,
      isOwnPost: json['isOwnPost'] as bool,
      updatedAt: json['updatedAt'] as String?,
      lastEditor: json['lastEditor'] == null
          ? null
          : UserBriefPayload.fromJson(
              json['lastEditor'] as Map<String, dynamic>,
            ),
      lastEditedAt: json['lastEditedAt'] as String?,
      revisionCount: (json['revisionCount'] as num?)?.toInt(),
      likeCount: (json['likeCount'] as num).toInt(),
      isLiked: json['isLiked'] as bool,
      isBookmarked: json['isBookmarked'] as bool,
    );

Map<String, dynamic> _$$PostPayloadImplToJson(_$PostPayloadImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'topicId': instance.topicId,
      'postNo': instance.postNo,
      'content': instance.content,
      'renderedContent': instance.renderedContent,
      'processStatus': instance.processStatus,
      'isHidden': instance.isHidden,
      'canModerate': instance.canModerate,
      'author': instance.author,
      'createdAt': instance.createdAt,
      'replyToPostId': instance.replyToPostId,
      'replyToUserId': instance.replyToUserId,
      'replyToUsername': instance.replyToUsername,
      'isOwnPost': instance.isOwnPost,
      'updatedAt': instance.updatedAt,
      'lastEditor': instance.lastEditor,
      'lastEditedAt': instance.lastEditedAt,
      'revisionCount': instance.revisionCount,
      'likeCount': instance.likeCount,
      'isLiked': instance.isLiked,
      'isBookmarked': instance.isBookmarked,
    };

_$ReplyTargetPayloadImpl _$$ReplyTargetPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$ReplyTargetPayloadImpl(
  id: (json['id'] as num).toInt(),
  postNo: (json['postNo'] as num?)?.toInt(),
  author: UserBriefPayload.fromJson(json['author'] as Map<String, dynamic>),
  renderedContent: json['renderedContent'] as String?,
  unavailable: json['unavailable'] as bool?,
);

Map<String, dynamic> _$$ReplyTargetPayloadImplToJson(
  _$ReplyTargetPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'postNo': instance.postNo,
  'author': instance.author,
  'renderedContent': instance.renderedContent,
  'unavailable': instance.unavailable,
};

_$PostWindowPayloadImpl _$$PostWindowPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$PostWindowPayloadImpl(
  posts: (json['posts'] as List<dynamic>)
      .map((e) => PostPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  replyTargets: (json['replyTargets'] as List<dynamic>)
      .map((e) => ReplyTargetPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  anchorPostId: (json['anchorPostId'] as num?)?.toInt(),
  beforePostNo: (json['beforePostNo'] as num?)?.toInt(),
  afterPostNo: (json['afterPostNo'] as num?)?.toInt(),
  hasBefore: json['hasBefore'] as bool,
  hasAfter: json['hasAfter'] as bool,
  total: (json['total'] as num).toInt(),
  maxPostNo: (json['maxPostNo'] as num).toInt(),
);

Map<String, dynamic> _$$PostWindowPayloadImplToJson(
  _$PostWindowPayloadImpl instance,
) => <String, dynamic>{
  'posts': instance.posts,
  'replyTargets': instance.replyTargets,
  'anchorPostId': instance.anchorPostId,
  'beforePostNo': instance.beforePostNo,
  'afterPostNo': instance.afterPostNo,
  'hasBefore': instance.hasBefore,
  'hasAfter': instance.hasAfter,
  'total': instance.total,
  'maxPostNo': instance.maxPostNo,
};

_$TopicDetailPropsImpl _$$TopicDetailPropsImplFromJson(
  Map<String, dynamic> json,
) => _$TopicDetailPropsImpl(
  topic: TopicDetailPayload.fromJson(json['topic'] as Map<String, dynamic>),
  postStream: PostWindowPayload.fromJson(
    json['postStream'] as Map<String, dynamic>,
  ),
  hotTopics: (json['hotTopics'] as List<dynamic>)
      .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  permissions: TopicDetailPermissions.fromJson(
    json['permissions'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$TopicDetailPropsImplToJson(
  _$TopicDetailPropsImpl instance,
) => <String, dynamic>{
  'topic': instance.topic,
  'postStream': instance.postStream,
  'hotTopics': instance.hotTopics,
  'permissions': instance.permissions,
};

_$TopicDetailPermissionsImpl _$$TopicDetailPermissionsImplFromJson(
  Map<String, dynamic> json,
) => _$TopicDetailPermissionsImpl(
  isOwnTopic: json['isOwnTopic'] as bool,
  canPost: json['canPost'] as bool,
  canModerateTopic: json['canModerateTopic'] as bool,
);

Map<String, dynamic> _$$TopicDetailPermissionsImplToJson(
  _$TopicDetailPermissionsImpl instance,
) => <String, dynamic>{
  'isOwnTopic': instance.isOwnTopic,
  'canPost': instance.canPost,
  'canModerateTopic': instance.canModerateTopic,
};
