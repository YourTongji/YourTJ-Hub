// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'publish.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$PublishCategoryPayloadImpl _$$PublishCategoryPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$PublishCategoryPayloadImpl(
  id: (json['id'] as num).toInt(),
  name: json['name'] as String,
  color: json['color'] as String,
);

Map<String, dynamic> _$$PublishCategoryPayloadImplToJson(
  _$PublishCategoryPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'color': instance.color,
};

_$PublishTopicPayloadImpl _$$PublishTopicPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$PublishTopicPayloadImpl(
  title: json['title'] as String,
  content: json['content'] as String,
  categoryIds: (json['categoryIds'] as List<dynamic>)
      .map((e) => (e as num).toInt())
      .toList(),
  topicStatus: (json['topicStatus'] as num).toInt(),
);

Map<String, dynamic> _$$PublishTopicPayloadImplToJson(
  _$PublishTopicPayloadImpl instance,
) => <String, dynamic>{
  'title': instance.title,
  'content': instance.content,
  'categoryIds': instance.categoryIds,
  'topicStatus': instance.topicStatus,
};

_$PublishPagePropsImpl _$$PublishPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$PublishPagePropsImpl(
  topicId: (json['topicId'] as num).toInt(),
  isEditing: json['isEditing'] as bool,
  categories: (json['categories'] as List<dynamic>)
      .map((e) => PublishCategoryPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  topic: PublishTopicPayload.fromJson(json['topic'] as Map<String, dynamic>),
);

Map<String, dynamic> _$$PublishPagePropsImplToJson(
  _$PublishPagePropsImpl instance,
) => <String, dynamic>{
  'topicId': instance.topicId,
  'isEditing': instance.isEditing,
  'categories': instance.categories,
  'topic': instance.topic,
};

_$CategoryHeaderPayloadImpl _$$CategoryHeaderPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CategoryHeaderPayloadImpl(
  id: (json['id'] as num).toInt(),
  name: json['name'] as String,
  description: json['description'] as String,
  icon: json['icon'] as String,
  color: json['color'] as String,
  url: json['url'] as String,
);

Map<String, dynamic> _$$CategoryHeaderPayloadImplToJson(
  _$CategoryHeaderPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'description': instance.description,
  'icon': instance.icon,
  'color': instance.color,
  'url': instance.url,
};

_$CategoryPagePropsImpl _$$CategoryPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$CategoryPagePropsImpl(
  category: CategoryHeaderPayload.fromJson(
    json['category'] as Map<String, dynamic>,
  ),
  sort: json['sort'] as String,
  tabs: (json['tabs'] as List<dynamic>)
      .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  topics: (json['topics'] as List<dynamic>)
      .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
);

Map<String, dynamic> _$$CategoryPagePropsImplToJson(
  _$CategoryPagePropsImpl instance,
) => <String, dynamic>{
  'category': instance.category,
  'sort': instance.sort,
  'tabs': instance.tabs,
  'topics': instance.topics,
  'pagination': instance.pagination,
};

_$AnnouncementItemPayloadImpl _$$AnnouncementItemPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$AnnouncementItemPayloadImpl(
  id: json['id'] as String,
  title: json['title'] as String,
  html: json['html'] as String,
);

Map<String, dynamic> _$$AnnouncementItemPayloadImplToJson(
  _$AnnouncementItemPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'title': instance.title,
  'html': instance.html,
};

_$AnnouncementPayloadImpl _$$AnnouncementPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$AnnouncementPayloadImpl(
  enabled: json['enabled'] as bool,
  html: json['html'] as String,
  publishedAt: json['publishedAt'] as String?,
  items: (json['items'] as List<dynamic>?)
      ?.map((e) => AnnouncementItemPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$$AnnouncementPayloadImplToJson(
  _$AnnouncementPayloadImpl instance,
) => <String, dynamic>{
  'enabled': instance.enabled,
  'html': instance.html,
  'publishedAt': instance.publishedAt,
  'items': instance.items,
};

_$HomePropsImpl _$$HomePropsImplFromJson(Map<String, dynamic> json) =>
    _$HomePropsImpl(
      sort: json['sort'] as String,
      tabs: (json['tabs'] as List<dynamic>)
          .map((e) => TabItemPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      topics: (json['topics'] as List<dynamic>)
          .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: PaginationPayload.fromJson(
        json['pagination'] as Map<String, dynamic>,
      ),
      announcement: AnnouncementPayload.fromJson(
        json['announcement'] as Map<String, dynamic>,
      ),
    );

Map<String, dynamic> _$$HomePropsImplToJson(_$HomePropsImpl instance) =>
    <String, dynamic>{
      'sort': instance.sort,
      'tabs': instance.tabs,
      'topics': instance.topics,
      'pagination': instance.pagination,
      'announcement': instance.announcement,
    };
