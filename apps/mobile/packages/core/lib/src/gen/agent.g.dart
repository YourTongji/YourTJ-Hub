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
