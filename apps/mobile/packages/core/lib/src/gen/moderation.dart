import 'package:freezed_annotation/freezed_annotation.dart';

import 'common.dart';
import 'topic.dart';

part 'moderation.freezed.dart';
part 'moderation.g.dart';

@freezed
abstract class ModerationPageProps with _$ModerationPageProps {
  const factory ModerationPageProps({
    required List<TabItemPayload> categoryTabs,
    required List<TopicPayload> topics,
    required PaginationPayload pagination,
  }) = _ModerationPageProps;

  factory ModerationPageProps.fromJson(Map<String, dynamic> json) =>
      _$ModerationPagePropsFromJson(json);
}

@freezed
abstract class ModerationLogSubject with _$ModerationLogSubject {
  const factory ModerationLogSubject({
    required String type,
    required int id,
    required String title,
    String? url,
    String? excerpt,
  }) = _ModerationLogSubject;

  factory ModerationLogSubject.fromJson(Map<String, dynamic> json) =>
      _$ModerationLogSubjectFromJson(json);
}

@freezed
abstract class ModerationLogActor with _$ModerationLogActor {
  const factory ModerationLogActor({
    required int id,
    required String username,
    required String avatarUrl,
  }) = _ModerationLogActor;

  factory ModerationLogActor.fromJson(Map<String, dynamic> json) =>
      _$ModerationLogActorFromJson(json);
}

@freezed
abstract class ModerationLogItem with _$ModerationLogItem {
  const factory ModerationLogItem({
    required int id,
    required String action,
    required ModerationLogActor actor,
    required ModerationLogSubject subject,
    required List<CategoryBriefPayload> categories,
    required String messageCode,
    required Map<String, dynamic> params,
    required String createdAt,
  }) = _ModerationLogItem;

  factory ModerationLogItem.fromJson(Map<String, dynamic> json) =>
      _$ModerationLogItemFromJson(json);
}

@freezed
abstract class ModerationLogListResponse with _$ModerationLogListResponse {
  const factory ModerationLogListResponse({
    required List<ModerationLogItem> items,
    required int nextCursor,
    required bool hasNext,
  }) = _ModerationLogListResponse;

  factory ModerationLogListResponse.fromJson(Map<String, dynamic> json) =>
      _$ModerationLogListResponseFromJson(json);
}

@freezed
abstract class ModerationReportItem with _$ModerationReportItem {
  const factory ModerationReportItem({
    required int id,
    required String targetType,
    required int targetId,
    required String targetUrl,
    required String title,
    required String excerpt,
    required String reason,
    required String note,
    required String status,
    required String resolution,
    required ModerationLogActor reporter,
    required ModerationLogActor handler,
    required List<CategoryBriefPayload> categories,
    required String createdAt,
    String? handledAt,
  }) = _ModerationReportItem;

  factory ModerationReportItem.fromJson(Map<String, dynamic> json) =>
      _$ModerationReportItemFromJson(json);
}

@freezed
abstract class ModerationReportListResponse with _$ModerationReportListResponse {
  const factory ModerationReportListResponse({
    required List<ModerationReportItem> items,
    required int nextCursor,
    required bool hasNext,
  }) = _ModerationReportListResponse;

  factory ModerationReportListResponse.fromJson(Map<String, dynamic> json) =>
      _$ModerationReportListResponseFromJson(json);
}
