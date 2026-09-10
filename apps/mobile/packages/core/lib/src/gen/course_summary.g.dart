// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'course_summary.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$CourseAiSummaryRepresentativeReviewImpl
_$$CourseAiSummaryRepresentativeReviewImplFromJson(Map<String, dynamic> json) =>
    _$CourseAiSummaryRepresentativeReviewImpl(
      excerpt: json['excerpt'] as String,
      sentiment: json['sentiment'] as String,
    );

Map<String, dynamic> _$$CourseAiSummaryRepresentativeReviewImplToJson(
  _$CourseAiSummaryRepresentativeReviewImpl instance,
) => <String, dynamic>{
  'excerpt': instance.excerpt,
  'sentiment': instance.sentiment,
};

_$CourseAiSummaryPayloadImpl _$$CourseAiSummaryPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$CourseAiSummaryPayloadImpl(
  consensus: json['consensus'] as String,
  keywords: (json['keywords'] as List<dynamic>)
      .map((e) => e as String)
      .toList(),
  pros: (json['pros'] as List<dynamic>).map((e) => e as String).toList(),
  cons: (json['cons'] as List<dynamic>).map((e) => e as String).toList(),
  representativeReviews: (json['representativeReviews'] as List<dynamic>)
      .map(
        (e) => CourseAiSummaryRepresentativeReview.fromJson(
          e as Map<String, dynamic>,
        ),
      )
      .toList(),
);

Map<String, dynamic> _$$CourseAiSummaryPayloadImplToJson(
  _$CourseAiSummaryPayloadImpl instance,
) => <String, dynamic>{
  'consensus': instance.consensus,
  'keywords': instance.keywords,
  'pros': instance.pros,
  'cons': instance.cons,
  'representativeReviews': instance.representativeReviews,
};

_$CourseAiSummaryResultImpl _$$CourseAiSummaryResultImplFromJson(
  Map<String, dynamic> json,
) => _$CourseAiSummaryResultImpl(
  status: json['status'] as String,
  summary: json['summary'] == null
      ? null
      : CourseAiSummaryPayload.fromJson(
          json['summary'] as Map<String, dynamic>,
        ),
  generatedAt: json['generatedAt'] as String?,
  model: json['model'] as String?,
);

Map<String, dynamic> _$$CourseAiSummaryResultImplToJson(
  _$CourseAiSummaryResultImpl instance,
) => <String, dynamic>{
  'status': instance.status,
  'summary': instance.summary,
  'generatedAt': instance.generatedAt,
  'model': instance.model,
};
