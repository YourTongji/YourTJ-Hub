// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'course_catalog.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$CourseListResultPayloadImpl _$$CourseListResultPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseListResultPayloadImpl(
  list: (json['list'] as List<dynamic>)
      .map((e) => CourseSummaryPayload.fromJson(e as Map<String, dynamic>))
      .toList(),
  page: (json['page'] as num).toInt(),
  size: (json['size'] as num).toInt(),
  total: (json['total'] as num).toInt(),
  hasNext: json['hasNext'] as bool,
);

Map<String, dynamic> _$$CourseListResultPayloadImplToJson(
  _$CourseListResultPayloadImpl instance,
) => <String, dynamic>{
  'list': instance.list,
  'page': instance.page,
  'size': instance.size,
  'total': instance.total,
  'hasNext': instance.hasNext,
};
