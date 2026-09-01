import 'package:freezed_annotation/freezed_annotation.dart';

import 'common.dart';
import 'topic.dart';

part 'search.freezed.dart';
part 'search.g.dart';

@freezed
abstract class UserSearchPayload with _$UserSearchPayload {
  const factory UserSearchPayload({
    required int id,
    required String username,
    required String nickname,
    required String avatarUrl,
    required String bio,
  }) = _UserSearchPayload;

  factory UserSearchPayload.fromJson(Map<String, dynamic> json) =>
      _$UserSearchPayloadFromJson(json);
}

@freezed
abstract class CategorySearchPayload with _$CategorySearchPayload {
  const factory CategorySearchPayload({
    required int id,
    required String name,
    required String slug,
    required String icon,
    required String color,
    required String desc,
  }) = _CategorySearchPayload;

  factory CategorySearchPayload.fromJson(Map<String, dynamic> json) =>
      _$CategorySearchPayloadFromJson(json);
}

@freezed
abstract class CourseSearchPayload with _$CourseSearchPayload {
  const factory CourseSearchPayload({
    required int id,
    required String primaryCode,
    required String name,
    required String department,
    required int creditX10,
    // (code, teacher) 复合身份：卡片身份教师（无教师时省略）。
    int? teacherId,
    String? teacherName,
    List<String>? aliases,
    List<String>? instructors,
    List<String>? terms,
    List<String>? campus,
    double? ratingAvg,
    int? reviewCount,
  }) = _CourseSearchPayload;

  factory CourseSearchPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseSearchPayloadFromJson(json);
}

@freezed
abstract class SearchPageProps with _$SearchPageProps {
  const factory SearchPageProps({
    required String query,
    required String scope,
    required List<TopicPayload> topics,
    required List<UserSearchPayload> users,
    required List<CategorySearchPayload> categories,
    required List<CourseSearchPayload> courses,
    required int total,
    required int usersTotal,
    required int categoriesTotal,
    required int coursesTotal,
    required int totalPages,
    required PaginationPayload pagination,
    List<String>? failedScopes,
    bool? searchUnavailable,
  }) = _SearchPageProps;

  factory SearchPageProps.fromJson(Map<String, dynamic> json) =>
      _$SearchPagePropsFromJson(json);
}
