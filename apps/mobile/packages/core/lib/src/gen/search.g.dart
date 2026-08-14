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

_$CourseSearchPayloadImpl _$$CourseSearchPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseSearchPayloadImpl(
  id: (json['id'] as num).toInt(),
  primaryCode: json['primaryCode'] as String,
  name: json['name'] as String,
  department: json['department'] as String,
  creditX10: (json['creditX10'] as num).toInt(),
  aliases: (json['aliases'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  instructors: (json['instructors'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
  terms: (json['terms'] as List<dynamic>?)?.map((e) => e as String).toList(),
  campus: (json['campus'] as List<dynamic>?)?.map((e) => e as String).toList(),
  ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
  reviewCount: (json['reviewCount'] as num?)?.toInt(),
);

Map<String, dynamic> _$$CourseSearchPayloadImplToJson(
  _$CourseSearchPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'primaryCode': instance.primaryCode,
  'name': instance.name,
  'department': instance.department,
  'creditX10': instance.creditX10,
  'aliases': instance.aliases,
  'instructors': instance.instructors,
  'terms': instance.terms,
  'campus': instance.campus,
  'ratingAvg': instance.ratingAvg,
  'reviewCount': instance.reviewCount,
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
  courses: (json['courses'] as List<dynamic>)
      .map((e) => CourseSearchPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  total: (json['total'] as num).toInt(),
  usersTotal: (json['usersTotal'] as num).toInt(),
  categoriesTotal: (json['categoriesTotal'] as num).toInt(),
  coursesTotal: (json['coursesTotal'] as num).toInt(),
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
  'courses': instance.courses,
  'total': instance.total,
  'usersTotal': instance.usersTotal,
  'categoriesTotal': instance.categoriesTotal,
  'coursesTotal': instance.coursesTotal,
  'totalPages': instance.totalPages,
  'pagination': instance.pagination,
  'failedScopes': instance.failedScopes,
  'searchUnavailable': instance.searchUnavailable,
};
