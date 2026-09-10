/// 课程目录列表契约镜像（`GET /api/forum/courses` 的 result）。
///
/// 列表项复用 `content_pages.dart` 的 CourseSummaryPayload（与契约
/// CourseSummary 同形）；本文件只补列表包装。
library;

import 'package:freezed_annotation/freezed_annotation.dart';

import 'content_pages.dart';

part 'course_catalog.freezed.dart';
part 'course_catalog.g.dart';

@freezed
abstract class CourseListResultPayload with _$CourseListResultPayload {
  const factory CourseListResultPayload({
    required List<CourseSummaryPayload> list,
    required int page,
    required int size,
    required int total,
    required bool hasNext,
  }) = _CourseListResultPayload;

  factory CourseListResultPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseListResultPayloadFromJson(json);
}
