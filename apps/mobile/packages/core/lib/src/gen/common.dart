import 'package:freezed_annotation/freezed_annotation.dart';

part 'common.freezed.dart';
part 'common.g.dart';

@freezed
abstract class PageMeta with _$PageMeta {
  const factory PageMeta({
    required String title,
    String? description,
    String? canonical,
    String? prevUrl,
    String? nextUrl,
    String? robots,
    OpenGraphMeta? openGraph,
    TwitterMeta? twitter,
    Map<String, dynamic>? jsonLd,
  }) = _PageMeta;

  factory PageMeta.fromJson(Map<String, dynamic> json) => _$PageMetaFromJson(json);
}

@freezed
abstract class OpenGraphMeta with _$OpenGraphMeta {
  const factory OpenGraphMeta({
    String? title,
    String? description,
    String? type,
    String? url,
    String? siteName,
    String? image,
    String? publishedTime,
    String? modifiedTime,
    String? author,
    String? section,
    List<String>? tags,
  }) = _OpenGraphMeta;

  factory OpenGraphMeta.fromJson(Map<String, dynamic> json) => _$OpenGraphMetaFromJson(json);
}

@freezed
abstract class TwitterMeta with _$TwitterMeta {
  const factory TwitterMeta({
    String? card,
    String? title,
    String? description,
    String? image,
  }) = _TwitterMeta;

  factory TwitterMeta.fromJson(Map<String, dynamic> json) => _$TwitterMetaFromJson(json);
}

@freezed
abstract class PaginationPayload with _$PaginationPayload {
  const factory PaginationPayload({
    required int page,
    required int nextPage,
    required bool hasNext,
    required String nextUrl,
  }) = _PaginationPayload;

  factory PaginationPayload.fromJson(Map<String, dynamic> json) =>
      _$PaginationPayloadFromJson(json);
}

@freezed
abstract class TabItemPayload with _$TabItemPayload {
  const factory TabItemPayload({
    required String key,
    String? label,
    required String url,
    required bool active,
  }) = _TabItemPayload;

  factory TabItemPayload.fromJson(Map<String, dynamic> json) => _$TabItemPayloadFromJson(json);
}

@freezed
abstract class ErrorPageProps with _$ErrorPageProps {
  const factory ErrorPageProps({
    required String code,
    required String title,
    String? messageCode,
    Map<String, dynamic>? params,
  }) = _ErrorPageProps;

  factory ErrorPageProps.fromJson(Map<String, dynamic> json) => _$ErrorPagePropsFromJson(json);
}
