import 'package:freezed_annotation/freezed_annotation.dart';

part 'course_summary.freezed.dart';
part 'course_summary.g.dart';

@freezed
abstract class CourseAiSummaryRepresentativeReview
    with _$CourseAiSummaryRepresentativeReview {
  const factory CourseAiSummaryRepresentativeReview({
    required String excerpt,
    required String sentiment,
  }) = _CourseAiSummaryRepresentativeReview;

  factory CourseAiSummaryRepresentativeReview.fromJson(
    Map<String, dynamic> json,
  ) => _$CourseAiSummaryRepresentativeReviewFromJson(json);
}

@freezed
abstract class CourseAiSummaryPayload with _$CourseAiSummaryPayload {
  const factory CourseAiSummaryPayload({
    required String consensus,
    required List<String> keywords,
    required List<String> pros,
    required List<String> cons,
    required List<CourseAiSummaryRepresentativeReview> representativeReviews,
  }) = _CourseAiSummaryPayload;

  factory CourseAiSummaryPayload.fromJson(Map<String, dynamic> json) =>
      _$CourseAiSummaryPayloadFromJson(json);
}

@freezed
abstract class CourseAiSummaryResult with _$CourseAiSummaryResult {
  const factory CourseAiSummaryResult({
    required String status,
    CourseAiSummaryPayload? summary,
    String? generatedAt,
    String? model,
  }) = _CourseAiSummaryResult;

  factory CourseAiSummaryResult.fromJson(Map<String, dynamic> json) =>
      _$CourseAiSummaryResultFromJson(json);
}
