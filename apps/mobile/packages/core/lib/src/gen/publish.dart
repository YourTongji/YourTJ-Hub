import 'package:freezed_annotation/freezed_annotation.dart';

import 'common.dart';
import 'topic.dart';

part 'publish.freezed.dart';
part 'publish.g.dart';

@freezed
abstract class PublishCategoryPayload with _$PublishCategoryPayload {
  const factory PublishCategoryPayload({
    required int id,
    required String name,
    required String color,
  }) = _PublishCategoryPayload;

  factory PublishCategoryPayload.fromJson(Map<String, dynamic> json) =>
      _$PublishCategoryPayloadFromJson(json);
}

@freezed
abstract class PublishTopicPayload with _$PublishTopicPayload {
  const factory PublishTopicPayload({
    required String title,
    required String content,
    @Default(<int>[]) List<int> categoryIds,
    required int topicStatus,
  }) = _PublishTopicPayload;

  factory PublishTopicPayload.fromJson(Map<String, dynamic> json) =>
      _$PublishTopicPayloadFromJson(json);
}

@freezed
abstract class PublishPageProps with _$PublishPageProps {
  const factory PublishPageProps({
    required int topicId,
    required bool isEditing,
    required List<PublishCategoryPayload> categories,
    required PublishTopicPayload topic,
  }) = _PublishPageProps;

  factory PublishPageProps.fromJson(Map<String, dynamic> json) =>
      _$PublishPagePropsFromJson(json);
}

@freezed
abstract class CategoryHeaderPayload with _$CategoryHeaderPayload {
  const factory CategoryHeaderPayload({
    required int id,
    required String name,
    required String description,
    required String icon,
    required String color,
    required String url,
  }) = _CategoryHeaderPayload;

  factory CategoryHeaderPayload.fromJson(Map<String, dynamic> json) =>
      _$CategoryHeaderPayloadFromJson(json);
}

@freezed
abstract class CategoryPageProps with _$CategoryPageProps {
  const factory CategoryPageProps({
    required CategoryHeaderPayload category,
    required String sort,
    required List<TabItemPayload> tabs,
    required List<TopicPayload> topics,
    required PaginationPayload pagination,
  }) = _CategoryPageProps;

  factory CategoryPageProps.fromJson(Map<String, dynamic> json) =>
      _$CategoryPagePropsFromJson(json);
}

@freezed
abstract class AnnouncementItemPayload with _$AnnouncementItemPayload {
  const factory AnnouncementItemPayload({
    required String id,
    required String title,
    required String html,
  }) = _AnnouncementItemPayload;

  factory AnnouncementItemPayload.fromJson(Map<String, dynamic> json) =>
      _$AnnouncementItemPayloadFromJson(json);
}

@freezed
abstract class AnnouncementPayload with _$AnnouncementPayload {
  const factory AnnouncementPayload({
    required bool enabled,
    required String html,
    String? publishedAt,
    List<AnnouncementItemPayload>? items,
  }) = _AnnouncementPayload;

  factory AnnouncementPayload.fromJson(Map<String, dynamic> json) =>
      _$AnnouncementPayloadFromJson(json);
}

@freezed
abstract class HomeProps with _$HomeProps {
  const factory HomeProps({
    required String sort,
    required List<TabItemPayload> tabs,
    required List<TopicPayload> topics,
    required PaginationPayload pagination,
    required AnnouncementPayload announcement,
  }) = _HomeProps;

  factory HomeProps.fromJson(Map<String, dynamic> json) =>
      _$HomePropsFromJson(json);
}
