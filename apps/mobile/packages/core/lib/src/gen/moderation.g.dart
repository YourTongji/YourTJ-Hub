// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'moderation.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$ModerationPagePropsImpl _$$ModerationPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationPagePropsImpl(
  categoryTabs: (json['categoryTabs'] as List<dynamic>)
      .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  topics: (json['topics'] as List<dynamic>)
      .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$ModerationPagePropsImplToJson(
  _$ModerationPagePropsImpl instance,
) => <String, dynamic>{
  'categoryTabs': instance.categoryTabs,
  'topics': instance.topics,
  'pagination': instance.pagination,
};

_$ModerationLogSubjectImpl _$$ModerationLogSubjectImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationLogSubjectImpl(
  type: json['type'] as String,
  id: (json['id'] as num).toInt(),
  title: json['title'] as String,
  url: json['url'] as String?,
  excerpt: json['excerpt'] as String?,
);

Map<String, dynamic> _$$ModerationLogSubjectImplToJson(
  _$ModerationLogSubjectImpl instance,
) => <String, dynamic>{
  'type': instance.type,
  'id': instance.id,
  'title': instance.title,
  'url': instance.url,
  'excerpt': instance.excerpt,
};

_$ModerationLogActorImpl _$$ModerationLogActorImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationLogActorImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  avatarUrl: json['avatarUrl'] as String,
);

Map<String, dynamic> _$$ModerationLogActorImplToJson(
  _$ModerationLogActorImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'avatarUrl': instance.avatarUrl,
};

_$ModerationLogItemImpl _$$ModerationLogItemImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationLogItemImpl(
  id: (json['id'] as num).toInt(),
  action: json['action'] as String,
  actor: ModerationLogActor.fromJson(json['actor'] as Map<String, dynamic>),
  subject: ModerationLogSubject.fromJson(
    json['subject'] as Map<String, dynamic>,
  ),
  categories: (json['categories'] as List<dynamic>)
      .map((e) => CategoryBriefPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  messageCode: json['messageCode'] as String,
  params: json['params'] as Map<String, dynamic>,
  createdAt: json['createdAt'] as String,
);

Map<String, dynamic> _$$ModerationLogItemImplToJson(
  _$ModerationLogItemImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'action': instance.action,
  'actor': instance.actor,
  'subject': instance.subject,
  'categories': instance.categories,
  'messageCode': instance.messageCode,
  'params': instance.params,
  'createdAt': instance.createdAt,
};

_$ModerationLogListResponseImpl _$$ModerationLogListResponseImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationLogListResponseImpl(
  items: (json['items'] as List<dynamic>)
      .map((e) => ModerationLogItem.fromJson(e as Map<String, dynamic>))
      .toList(),
  nextCursor: (json['nextCursor'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$ModerationLogListResponseImplToJson(
  _$ModerationLogListResponseImpl instance,
) => <String, dynamic>{
  'items': instance.items,
  'nextCursor': instance.nextCursor,
  'hasNext': instance.hasNext,
};

_$ModerationReportItemImpl _$$ModerationReportItemImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationReportItemImpl(
  id: (json['id'] as num).toInt(),
  targetType: json['targetType'] as String,
  targetId: (json['targetId'] as num).toInt(),
  targetUrl: json['targetUrl'] as String,
  title: json['title'] as String,
  excerpt: json['excerpt'] as String,
  reason: json['reason'] as String,
  note: json['note'] as String,
  status: json['status'] as String,
  resolution: json['resolution'] as String,
  reporter: ModerationLogActor.fromJson(
    json['reporter'] as Map<String, dynamic>,
  ),
  handler: ModerationLogActor.fromJson(json['handler'] as Map<String, dynamic>),
  categories: (json['categories'] as List<dynamic>)
      .map((e) => CategoryBriefPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  createdAt: json['createdAt'] as String,
  handledAt: json['handledAt'] as String?,
);

Map<String, dynamic> _$$ModerationReportItemImplToJson(
  _$ModerationReportItemImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'targetType': instance.targetType,
  'targetId': instance.targetId,
  'targetUrl': instance.targetUrl,
  'title': instance.title,
  'excerpt': instance.excerpt,
  'reason': instance.reason,
  'note': instance.note,
  'status': instance.status,
  'resolution': instance.resolution,
  'reporter': instance.reporter,
  'handler': instance.handler,
  'categories': instance.categories,
  'createdAt': instance.createdAt,
  'handledAt': instance.handledAt,
};

_$ModerationReportListResponseImpl _$$ModerationReportListResponseImplFromJson(
  Map<String, dynamic> json,
) => _$ModerationReportListResponseImpl(
  items: (json['items'] as List<dynamic>)
      .map((e) => ModerationReportItem.fromJson(e as Map<String, dynamic>))
      .toList(),
  nextCursor: (json['nextCursor'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$ModerationReportListResponseImplToJson(
  _$ModerationReportListResponseImpl instance,
) => <String, dynamic>{
  'items': instance.items,
  'nextCursor': instance.nextCursor,
  'hasNext': instance.hasNext,
};
