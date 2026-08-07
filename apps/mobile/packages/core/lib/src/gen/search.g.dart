// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'search.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$UserSearchPayloadImpl _$$UserSearchPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserSearchPayloadImpl(
  id: (json['id'] as num).toInt(),
  username: json['username'] as String,
  nickname: json['nickname'] as String,
  avatarUrl: json['avatarUrl'] as String,
  bio: json['bio'] as String,
);

Map<String, dynamic> _$$UserSearchPayloadImplToJson(
  _$UserSearchPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'username': instance.username,
  'nickname': instance.nickname,
  'avatarUrl': instance.avatarUrl,
  'bio': instance.bio,
};

_$CategorySearchPayloadImpl _$$CategorySearchPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CategorySearchPayloadImpl(
  id: (json['id'] as num).toInt(),
  name: json['name'] as String,
  slug: json['slug'] as String,
  icon: json['icon'] as String,
  color: json['color'] as String,
  desc: json['desc'] as String,
);

Map<String, dynamic> _$$CategorySearchPayloadImplToJson(
  _$CategorySearchPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'slug': instance.slug,
  'icon': instance.icon,
  'color': instance.color,
  'desc': instance.desc,
};

_$SearchPagePropsImpl _$$SearchPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$SearchPagePropsImpl(
  query: json['query'] as String,
  scope: json['scope'] as String,
  topics: (json['topics'] as List<dynamic>)
      .map((e) => TopicPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  users: (json['users'] as List<dynamic>)
      .map((e) => UserSearchPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  categories: (json['categories'] as List<dynamic>)
      .map((e) => CategorySearchPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  total: (json['total'] as num).toInt(),
  usersTotal: (json['usersTotal'] as num).toInt(),
  categoriesTotal: (json['categoriesTotal'] as num).toInt(),
  totalPages: (json['totalPages'] as num).toInt(),
  pagination: PaginationPayload.fromJson(
    json['pagination'] as Map<String, dynamic>,
  ),
  failedScopes: (json['failedScopes'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  searchUnavailable: json['searchUnavailable'] as bool?,
);

Map<String, dynamic> _$$SearchPagePropsImplToJson(
  _$SearchPagePropsImpl instance,
) => <String, dynamic>{
  'query': instance.query,
  'scope': instance.scope,
  'topics': instance.topics,
  'users': instance.users,
  'categories': instance.categories,
  'total': instance.total,
  'usersTotal': instance.usersTotal,
  'categoriesTotal': instance.categoriesTotal,
  'totalPages': instance.totalPages,
  'pagination': instance.pagination,
  'failedScopes': instance.failedScopes,
  'searchUnavailable': instance.searchUnavailable,
};
