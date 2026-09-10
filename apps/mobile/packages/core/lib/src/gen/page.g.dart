// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'page.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$PagePayloadImpl _$$PagePayloadImplFromJson(Map<String, dynamic> json) =>
    _$PagePayloadImpl(
      component: json['component'] as String,
      props: json['props'] as Map<String, dynamic>,
      meta: PageMeta.fromJson(json['meta'] as Map<String, dynamic>),
      layout: LayoutPayload.fromJson(json['layout'] as Map<String, dynamic>),
      url: json['url'] as String,
      version: json['version'] as String,
    );

Map<String, dynamic> _$$PagePayloadImplToJson(_$PagePayloadImpl instance) =>
    <String, dynamic>{
      'component': instance.component,
      'props': instance.props,
      'meta': instance.meta,
      'layout': instance.layout,
      'url': instance.url,
      'version': instance.version,
    };
