// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'agent.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$AgentMeResultImpl _$$AgentMeResultImplFromJson(Map<String, dynamic> json) =>
    _$AgentMeResultImpl(
      agentId: (json['agentId'] as num).toInt(),
      username: json['username'] as String,
      nickname: json['nickname'] as String,
      avatarUrl: json['avatarUrl'] as String,
      tokenPrefix: json['tokenPrefix'] as String,
      enabled: (json['enabled'] as num).toInt(),
      createdAt: (json['createdAt'] as num).toInt(),
      updatedAt: (json['updatedAt'] as num).toInt(),
    );

Map<String, dynamic> _$$AgentMeResultImplToJson(_$AgentMeResultImpl instance) =>
    <String, dynamic>{
      'agentId': instance.agentId,
      'username': instance.username,
      'nickname': instance.nickname,
      'avatarUrl': instance.avatarUrl,
      'tokenPrefix': instance.tokenPrefix,
      'enabled': instance.enabled,
      'createdAt': instance.createdAt,
      'updatedAt': instance.updatedAt,
    };

_$AgentTopicItemImpl _$$AgentTopicItemImplFromJson(Map<String, dynamic> json) =>
    _$AgentTopicItemImpl(
      id: (json['id'] as num).toInt(),
      title: json['title'] as String,
      excerpt: json['excerpt'] as String,
      categoryIds: (json['categoryIds'] as List<dynamic>)
          .map((e) => (e as num).toInt())
          .toList(),
      userId: (json['userId'] as num).toInt(),
      status: (json['status'] as num).toInt(),
      processStatus: (json['processStatus'] as num).toInt(),
      replyCount: (json['replyCount'] as num).toInt(),
      viewCount: (json['viewCount'] as num).toInt(),
      postCount: (json['postCount'] as num).toInt(),
      lastPostedAt: (json['lastPostedAt'] as num?)?.toInt(),
      createdAt: (json['createdAt'] as num).toInt(),
      updatedAt: (json['updatedAt'] as num).toInt(),
    );

Map<String, dynamic> _$$AgentTopicItemImplToJson(
  _$AgentTopicItemImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'title': instance.title,
  'excerpt': instance.excerpt,
  'categoryIds': instance.categoryIds,
  'userId': instance.userId,
  'status': instance.status,
  'processStatus': instance.processStatus,
  'replyCount': instance.replyCount,
  'viewCount': instance.viewCount,
  'postCount': instance.postCount,
  'lastPostedAt': instance.lastPostedAt,
  'createdAt': instance.createdAt,
  'updatedAt': instance.updatedAt,
};

_$AgentTopicListResultImpl _$$AgentTopicListResultImplFromJson(
  Map<String, dynamic> json,
) => _$AgentTopicListResultImpl(
  list: (json['list'] as List<dynamic>)
      .map((e) => AgentTopicItem.fromJson(e as Map<String, dynamic>))
      .toList(),
  page: (json['page'] as num).toInt(),
  pageSize: (json['pageSize'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$AgentTopicListResultImplToJson(
  _$AgentTopicListResultImpl instance,
) => <String, dynamic>{
  'list': instance.list,
  'page': instance.page,
  'pageSize': instance.pageSize,
  'hasNext': instance.hasNext,
};

_$AgentWriteTopicRequestImpl _$$AgentWriteTopicRequestImplFromJson(
  Map<String, dynamic> json,
) => _$AgentWriteTopicRequestImpl(
  title: json['title'] as String,
  content: json['content'] as String,
  categoryId: (json['categoryId'] as List<dynamic>)
      .map((e) => (e as num).toInt())
      .toList(),
);

Map<String, dynamic> _$$AgentWriteTopicRequestImplToJson(
  _$AgentWriteTopicRequestImpl instance,
) => <String, dynamic>{
  'title': instance.title,
  'content': instance.content,
  'categoryId': instance.categoryId,
};

_$AgentCreatePostRequestImpl _$$AgentCreatePostRequestImplFromJson(
  Map<String, dynamic> json,
) => _$AgentCreatePostRequestImpl(
  content: json['content'] as String,
  replyToPostId: (json['replyToPostId'] as num?)?.toInt(),
);

Map<String, dynamic> _$$AgentCreatePostRequestImplToJson(
  _$AgentCreatePostRequestImpl instance,
) => <String, dynamic>{
  'content': instance.content,
  'replyToPostId': instance.replyToPostId,
};

_$AgentCreatePostResultImpl _$$AgentCreatePostResultImplFromJson(
  Map<String, dynamic> json,
) => _$AgentCreatePostResultImpl(
  id: (json['id'] as num).toInt(),
  postNo: (json['postNo'] as num).toInt(),
  renderedContent: json['renderedContent'] as String,
);

Map<String, dynamic> _$$AgentCreatePostResultImplToJson(
  _$AgentCreatePostResultImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'postNo': instance.postNo,
  'renderedContent': instance.renderedContent,
};

_$AgentInboxItemImpl _$$AgentInboxItemImplFromJson(Map<String, dynamic> json) =>
    _$AgentInboxItemImpl(
      id: (json['id'] as num).toInt(),
      topicId: (json['topicId'] as num).toInt(),
      postId: (json['postId'] as num).toInt(),
      eventType: json['eventType'] as String,
      actorId: (json['actorId'] as num).toInt(),
      contentPreview: json['contentPreview'] as String,
      status: (json['status'] as num).toInt(),
      deliveryStatus: (json['deliveryStatus'] as num).toInt(),
      attempts: (json['attempts'] as num).toInt(),
      lastError: json['lastError'] as String,
      readAt: (json['readAt'] as num?)?.toInt(),
      createdAt: (json['createdAt'] as num).toInt(),
      updatedAt: (json['updatedAt'] as num).toInt(),
    );

Map<String, dynamic> _$$AgentInboxItemImplToJson(
  _$AgentInboxItemImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'topicId': instance.topicId,
  'postId': instance.postId,
  'eventType': instance.eventType,
  'actorId': instance.actorId,
  'contentPreview': instance.contentPreview,
  'status': instance.status,
  'deliveryStatus': instance.deliveryStatus,
  'attempts': instance.attempts,
  'lastError': instance.lastError,
  'readAt': instance.readAt,
  'createdAt': instance.createdAt,
  'updatedAt': instance.updatedAt,
};

_$AgentInboxListResultImpl _$$AgentInboxListResultImplFromJson(
  Map<String, dynamic> json,
) => _$AgentInboxListResultImpl(
  list: (json['list'] as List<dynamic>)
      .map((e) => AgentInboxItem.fromJson(e as Map<String, dynamic>))
      .toList(),
  page: (json['page'] as num).toInt(),
  pageSize: (json['pageSize'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$AgentInboxListResultImplToJson(
  _$AgentInboxListResultImpl instance,
) => <String, dynamic>{
  'list': instance.list,
  'page': instance.page,
  'pageSize': instance.pageSize,
  'hasNext': instance.hasNext,
};

_$AgentInboxSuccessResultImpl _$$AgentInboxSuccessResultImplFromJson(
  Map<String, dynamic> json,
) => _$AgentInboxSuccessResultImpl(
  result: json['result'] as String,
  messageCode: json['messageCode'] as String,
);

Map<String, dynamic> _$$AgentInboxSuccessResultImplToJson(
  _$AgentInboxSuccessResultImpl instance,
) => <String, dynamic>{
  'result': instance.result,
  'messageCode': instance.messageCode,
};
