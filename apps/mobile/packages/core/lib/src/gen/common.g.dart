// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'common.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$PageMetaImpl _$$PageMetaImplFromJson(Map<String, dynamic> json) =>
    _$PageMetaImpl(
      title: json['title'] as String,
      description: json['description'] as String?,
      canonical: json['canonical'] as String?,
      prevUrl: json['prevUrl'] as String?,
      nextUrl: json['nextUrl'] as String?,
      robots: json['robots'] as String?,
      openGraph: json['openGraph'] == null
          ? null
          : OpenGraphMeta.fromJson(json['openGraph'] as Map<String, dynamic>),
      twitter: json['twitter'] == null
          ? null
          : TwitterMeta.fromJson(json['twitter'] as Map<String, dynamic>),
      jsonLd: json['jsonLd'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$$PageMetaImplToJson(_$PageMetaImpl instance) =>
    <String, dynamic>{
      'title': instance.title,
      'description': instance.description,
      'canonical': instance.canonical,
      'prevUrl': instance.prevUrl,
      'nextUrl': instance.nextUrl,
      'robots': instance.robots,
      'openGraph': instance.openGraph,
      'twitter': instance.twitter,
      'jsonLd': instance.jsonLd,
    };

_$OpenGraphMetaImpl _$$OpenGraphMetaImplFromJson(Map<String, dynamic> json) =>
    _$OpenGraphMetaImpl(
      title: json['title'] as String?,
      description: json['description'] as String?,
      type: json['type'] as String?,
      url: json['url'] as String?,
      siteName: json['siteName'] as String?,
      image: json['image'] as String?,
      publishedTime: json['publishedTime'] as String?,
      modifiedTime: json['modifiedTime'] as String?,
      author: json['author'] as String?,
      section: json['section'] as String?,
      tags: (json['tags'] as List<dynamic>?)?.map((e) => e as String).toList(),
    );

Map<String, dynamic> _$$OpenGraphMetaImplToJson(_$OpenGraphMetaImpl instance) =>
    <String, dynamic>{
      'title': instance.title,
      'description': instance.description,
      'type': instance.type,
      'url': instance.url,
      'siteName': instance.siteName,
      'image': instance.image,
      'publishedTime': instance.publishedTime,
      'modifiedTime': instance.modifiedTime,
      'author': instance.author,
      'section': instance.section,
      'tags': instance.tags,
    };

_$TwitterMetaImpl _$$TwitterMetaImplFromJson(Map<String, dynamic> json) =>
    _$TwitterMetaImpl(
      card: json['card'] as String?,
      title: json['title'] as String?,
      description: json['description'] as String?,
      image: json['image'] as String?,
    );

Map<String, dynamic> _$$TwitterMetaImplToJson(_$TwitterMetaImpl instance) =>
    <String, dynamic>{
      'card': instance.card,
      'title': instance.title,
      'description': instance.description,
      'image': instance.image,
    };

_$PaginationPayloadImpl _$$PaginationPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$PaginationPayloadImpl(
  page: (json['page'] as num).toInt(),
  nextPage: (json['nextPage'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
  nextUrl: json['nextUrl'] as String,
);

Map<String, dynamic> _$$PaginationPayloadImplToJson(
  _$PaginationPayloadImpl instance,
) => <String, dynamic>{
  'page': instance.page,
  'nextPage': instance.nextPage,
  'hasNext': instance.hasNext,
  'nextUrl': instance.nextUrl,
};

_$TabItemPayloadImpl _$$TabItemPayloadImplFromJson(Map<String, dynamic> json) =>
    _$TabItemPayloadImpl(
      key: json['key'] as String,
      label: json['label'] as String?,
      url: json['url'] as String,
      active: json['active'] as bool,
    );

Map<String, dynamic> _$$TabItemPayloadImplToJson(
  _$TabItemPayloadImpl instance,
) => <String, dynamic>{
  'key': instance.key,
  'label': instance.label,
  'url': instance.url,
  'active': instance.active,
};

_$ErrorPagePropsImpl _$$ErrorPagePropsImplFromJson(Map<String, dynamic> json) =>
    _$ErrorPagePropsImpl(
      code: json['code'] as String,
      title: json['title'] as String,
      messageCode: json['messageCode'] as String?,
      params: json['params'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$$ErrorPagePropsImplToJson(
  _$ErrorPagePropsImpl instance,
) => <String, dynamic>{
  'code': instance.code,
  'title': instance.title,
  'messageCode': instance.messageCode,
  'params': instance.params,
};
