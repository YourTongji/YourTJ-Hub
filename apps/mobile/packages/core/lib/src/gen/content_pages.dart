import 'package:freezed_annotation/freezed_annotation.dart';
import 'common.dart';

part 'content_pages.freezed.dart';
part 'content_pages.g.dart';

@freezed
abstract class LinksPageProps with _$LinksPageProps {
  const factory LinksPageProps({
    required List<LinkGroupPayload> groups,
    required int totalCount,
  }) = _LinksPageProps;

  factory LinksPageProps.fromJson(Map<String, dynamic> json) =>
      _$LinksPagePropsFromJson(json);
}

@freezed
abstract class LinkGroupPayload with _$LinkGroupPayload {
  const factory LinkGroupPayload({
    required String name,
    required String emoji,
    required String color,
    required List<FriendLinkPayload> links,
  }) = _LinkGroupPayload;

  factory LinkGroupPayload.fromJson(Map<String, dynamic> json) =>
      _$LinkGroupPayloadFromJson(json);
}

@freezed
abstract class FriendLinkPayload with _$FriendLinkPayload {
  const factory FriendLinkPayload({
    required String name,
    required String desc,
    required String url,
    required String logoUrl,
  }) = _FriendLinkPayload;

  factory FriendLinkPayload.fromJson(Map<String, dynamic> json) =>
      _$FriendLinkPayloadFromJson(json);
}

@freezed
abstract class SponsorsPageProps with _$SponsorsPageProps {
  const factory SponsorsPageProps({
    required List<SponsorSectionPayload> sections,
    required int totalCount,
    required SponsorsPageIntroPayload content,
    required SponsorsContactPayload contact,
    required List<SponsorsRulePayload> rules,
  }) = _SponsorsPageProps;

  factory SponsorsPageProps.fromJson(Map<String, dynamic> json) =>
      _$SponsorsPagePropsFromJson(json);
}

@freezed
abstract class SponsorSectionPayload with _$SponsorSectionPayload {
  const factory SponsorSectionPayload({
    required String key,
    required String label,
    required String tone,
    required List<SponsorPayload> sponsors,
  }) = _SponsorSectionPayload;

  factory SponsorSectionPayload.fromJson(Map<String, dynamic> json) =>
      _$SponsorSectionPayloadFromJson(json);
}

@freezed
abstract class SponsorPayload with _$SponsorPayload {
  const factory SponsorPayload({
    required String name,
    required String message,
    required String link,
    required String avatarUrl,
  }) = _SponsorPayload;

  factory SponsorPayload.fromJson(Map<String, dynamic> json) =>
      _$SponsorPayloadFromJson(json);
}

@freezed
abstract class SponsorsPageIntroPayload with _$SponsorsPageIntroPayload {
  const factory SponsorsPageIntroPayload({
    required String title,
    required String description,
  }) = _SponsorsPageIntroPayload;

  factory SponsorsPageIntroPayload.fromJson(Map<String, dynamic> json) =>
      _$SponsorsPageIntroPayloadFromJson(json);
}

@freezed
abstract class SponsorsContactPayload with _$SponsorsContactPayload {
  const factory SponsorsContactPayload({
    required String title,
    required String description,
    required String buttonText,
    required String buttonLink,
  }) = _SponsorsContactPayload;

  factory SponsorsContactPayload.fromJson(Map<String, dynamic> json) =>
      _$SponsorsContactPayloadFromJson(json);
}

@freezed
abstract class SponsorsRulePayload with _$SponsorsRulePayload {
  const factory SponsorsRulePayload({required String content}) =
      _SponsorsRulePayload;

  factory SponsorsRulePayload.fromJson(Map<String, dynamic> json) =>
      _$SponsorsRulePayloadFromJson(json);
}

@freezed
abstract class TermsPageProps with _$TermsPageProps {
  const factory TermsPageProps({
    required bool enabled,
    required String contentHtml,
  }) = _TermsPageProps;

  factory TermsPageProps.fromJson(Map<String, dynamic> json) =>
      _$TermsPagePropsFromJson(json);
}

@freezed
abstract class CourseSummaryPayload with _$CourseSummaryPayload {
  const factory CourseSummaryPayload({
    required int id,
    required String primaryCode,
    required String name,
    required String department,
    required int creditX10,
    required double ratingAvg,
    required int ratingCount,
    required int reviewCount,
    List<String>? aliases,
    List<String>? instructors,
    List<String>? recentTerms,
  }) = _CourseSummaryPayload;

  factory CourseSummaryPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseSummaryPayloadFromJson(json);
}

@freezed
abstract class CourseCatalogPageProps with _$CourseCatalogPageProps {
  const factory CourseCatalogPageProps({
    required CourseCatalogQueryPayload query,
    required List<CourseSummaryPayload> courses,
    required PaginationPayload pagination,
    required List<String> departments,
  }) = _CourseCatalogPageProps;

  factory CourseCatalogPageProps.fromJson(Map<String, dynamic> json) =>
      _$CourseCatalogPagePropsFromJson(json);
}

@freezed
abstract class CourseCatalogQueryPayload with _$CourseCatalogQueryPayload {
  const factory CourseCatalogQueryPayload({
    String? keyword,
    String? department,
    String? term,
    String? campus,
    String? instructor,
    bool? onlyWithReviews,
    String? sortBy,
    required int page,
    required int size,
  }) = _CourseCatalogQueryPayload;

  factory CourseCatalogQueryPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseCatalogQueryPayloadFromJson(json);
}

@freezed
abstract class CourseOfferingPayload with _$CourseOfferingPayload {
  const factory CourseOfferingPayload({
    required int id,
    required String termCode,
    String? termName,
    String? campus,
    String? faculty,
    List<String>? instructors,
  }) = _CourseOfferingPayload;

  factory CourseOfferingPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseOfferingPayloadFromJson(json);
}

@freezed
abstract class CourseDetailPageProps with _$CourseDetailPageProps {
  const factory CourseDetailPageProps({required CourseDetailPayload course}) =
      _CourseDetailPageProps;

  factory CourseDetailPageProps.fromJson(Map<String, dynamic> json) =>
      _$CourseDetailPagePropsFromJson(json);
}

@freezed
abstract class CourseDetailPayload with _$CourseDetailPayload {
  const factory CourseDetailPayload({
    required int id,
    required String primaryCode,
    required String name,
    required String department,
    required int creditX10,
    List<String>? aliases,
    List<CourseOfferingPayload>? offerings,
  }) = _CourseDetailPayload;

  factory CourseDetailPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseDetailPayloadFromJson(json);
}
