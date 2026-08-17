// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$NotificationPayloadImpl _$$NotificationPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationPayloadImpl(
  id: (json['id'] as num).toInt(),
  eventType: json['eventType'] as String,
  isRead: json['isRead'] as bool,
  createdAt: json['createdAt'] as String,
  title: json['title'] as String,
  content: json['content'] as String,
  actor: NotificationActorPayload.fromJson(
    json['actor'] as Map<String, dynamic>,
  ),
  topic: json['topic'] == null
      ? null
      : NotificationTopicPayload.fromJson(
          json['topic'] as Map<String, dynamic>,
        ),
  payload: NotificationInnerPayload.fromJson(
    json['payload'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$NotificationPayloadImplToJson(
  _$NotificationPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'eventType': instance.eventType,
  'isRead': instance.isRead,
  'createdAt': instance.createdAt,
  'title': instance.title,
  'content': instance.content,
  'actor': instance.actor,
  'topic': instance.topic,
  'payload': instance.payload,
};

_$NotificationActorPayloadImpl _$$NotificationActorPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationActorPayloadImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  avatarUrl: json['avatarUrl'] as String?,
);

Map<String, dynamic> _$$NotificationActorPayloadImplToJson(
  _$NotificationActorPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'avatarUrl': instance.avatarUrl,
};

_$NotificationTopicPayloadImpl _$$NotificationTopicPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationTopicPayloadImpl(
  id: (json['id'] as num).toInt(),
  title: json['title'] as String,
  url: json['url'] as String,
);

Map<String, dynamic> _$$NotificationTopicPayloadImplToJson(
  _$NotificationTopicPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'title': instance.title,
  'url': instance.url,
};

_$NotificationInnerPayloadImpl _$$NotificationInnerPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationInnerPayloadImpl(
  title: json['title'] as String?,
  content: json['content'] as String?,
  templateKey: json['templateKey'] as String?,
  templateParams: json['templateParams'] == null
      ? null
      : NotificationTemplateParams.fromJson(
          json['templateParams'] as Map<String, dynamic>,
        ),
  actorId: (json['actorId'] as num).toInt(),
  actorName: json['actorName'] as String?,
  topicId: (json['topicId'] as num?)?.toInt(),
  postId: (json['postId'] as num?)?.toInt(),
  topicTitle: json['topicTitle'] as String?,
  metadata: json['metadata'] == null
      ? null
      : NotificationMetadata.fromJson(json['metadata'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$NotificationInnerPayloadImplToJson(
  _$NotificationInnerPayloadImpl instance,
) => <String, dynamic>{
  'title': instance.title,
  'content': instance.content,
  'templateKey': instance.templateKey,
  'templateParams': instance.templateParams,
  'actorId': instance.actorId,
  'actorName': instance.actorName,
  'topicId': instance.topicId,
  'postId': instance.postId,
  'topicTitle': instance.topicTitle,
  'metadata': instance.metadata,
};

_$NotificationTemplateParamsImpl _$$NotificationTemplateParamsImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationTemplateParamsImpl(
  preview: json['preview'] as String?,
);

Map<String, dynamic> _$$NotificationTemplateParamsImplToJson(
  _$NotificationTemplateParamsImpl instance,
) => <String, dynamic>{
  'preview': instance.preview,
};

_$NotificationMetadataImpl _$$NotificationMetadataImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationMetadataImpl(
  followerName: json['followerName'] as String?,
  badgeCode: json['badgeCode'] as String?,
  badgeName: json['badgeName'] as String?,
  badgeIconUrl: json['badgeIconUrl'] as String?,
  profileUrl: json['profileUrl'] as String?,
);

Map<String, dynamic> _$$NotificationMetadataImplToJson(
  _$NotificationMetadataImpl instance,
) => <String, dynamic>{
  'followerName': instance.followerName,
  'badgeCode': instance.badgeCode,
  'badgeName': instance.badgeName,
  'badgeIconUrl': instance.badgeIconUrl,
  'profileUrl': instance.profileUrl,
};

_$NotificationListResponseImpl _$$NotificationListResponseImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationListResponseImpl(
  items: (json['items'] as List<dynamic>)
      .map((e) => NotificationPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  nextCursor: (json['nextCursor'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
  unreadCount: (json['unreadCount'] as num).toInt(),
);

Map<String, dynamic> _$$NotificationListResponseImplToJson(
  _$NotificationListResponseImpl instance,
) => <String, dynamic>{
  'items': instance.items,
  'nextCursor': instance.nextCursor,
  'hasNext': instance.hasNext,
  'unreadCount': instance.unreadCount,
};

_$NotificationsPagePropsImpl _$$NotificationsPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$NotificationsPagePropsImpl(
  total: (json['total'] as num).toInt(),
  unreadCount: (json['unreadCount'] as num).toInt(),
  notifications: (json['notifications'] as List<dynamic>)
      .map((e) => NotificationPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$NotificationsPagePropsImplToJson(
  _$NotificationsPagePropsImpl instance,
) => <String, dynamic>{
  'total': instance.total,
  'unreadCount': instance.unreadCount,
  'notifications': instance.notifications,
  'pagination': instance.pagination,
};

_$DraftPayloadImpl _$$DraftPayloadImplFromJson(Map<String, dynamic> json) =>
    _$DraftPayloadImpl(
      id: (json['id'] as num).toInt(),
      title: json['title'] as String,
      description: json['description'] as String,
      editUrl: json['editUrl'] as String,
      replyCount: (json['replyCount'] as num).toInt(),
      viewCount: (json['viewCount'] as num).toInt(),
      processStatus: (json['processStatus'] as num).toInt(),
      updatedAt: json['updatedAt'] as String,
      createdAt: json['createdAt'] as String,
      categories: (json['categories'] as List<dynamic>)
          .map((e) => CategoryBriefPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$DraftPayloadImplToJson(_$DraftPayloadImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'description': instance.description,
      'editUrl': instance.editUrl,
      'replyCount': instance.replyCount,
      'viewCount': instance.viewCount,
      'processStatus': instance.processStatus,
      'updatedAt': instance.updatedAt,
      'createdAt': instance.createdAt,
      'categories': instance.categories,
    };

_$DraftsPagePropsImpl _$$DraftsPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$DraftsPagePropsImpl(
  total: (json['total'] as num).toInt(),
  drafts: (json['drafts'] as List<dynamic>)
      .map((e) => DraftPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$DraftsPagePropsImplToJson(
  _$DraftsPagePropsImpl instance,
) => <String, dynamic>{
  'total': instance.total,
  'drafts': instance.drafts,
  'pagination': instance.pagination,
};
