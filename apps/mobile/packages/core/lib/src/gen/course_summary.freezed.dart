// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'course_summary.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

CourseAiSummaryRepresentativeReview
_$CourseAiSummaryRepresentativeReviewFromJson(Map<String, dynamic> json) {
  return _CourseAiSummaryRepresentativeReview.fromJson(json);
}

/// @nodoc
mixin _$CourseAiSummaryRepresentativeReview {
  String get excerpt => throw _privateConstructorUsedError;
  String get sentiment => throw _privateConstructorUsedError;

  /// Serializes this CourseAiSummaryRepresentativeReview to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseAiSummaryRepresentativeReview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseAiSummaryRepresentativeReviewCopyWith<
    CourseAiSummaryRepresentativeReview
  >
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseAiSummaryRepresentativeReviewCopyWith<$Res> {
  factory $CourseAiSummaryRepresentativeReviewCopyWith(
    CourseAiSummaryRepresentativeReview value,
    $Res Function(CourseAiSummaryRepresentativeReview) then,
  ) =
      _$CourseAiSummaryRepresentativeReviewCopyWithImpl<
        $Res,
        CourseAiSummaryRepresentativeReview
      >;
  @useResult
  $Res call({String excerpt, String sentiment});
}

/// @nodoc
class _$CourseAiSummaryRepresentativeReviewCopyWithImpl<
  $Res,
  $Val extends CourseAiSummaryRepresentativeReview
>
    implements $CourseAiSummaryRepresentativeReviewCopyWith<$Res> {
  _$CourseAiSummaryRepresentativeReviewCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseAiSummaryRepresentativeReview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? excerpt = null, Object? sentiment = null}) {
    return _then(
      _value.copyWith(
            excerpt: null == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String,
            sentiment: null == sentiment
                ? _value.sentiment
                : sentiment // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseAiSummaryRepresentativeReviewImplCopyWith<$Res>
    implements $CourseAiSummaryRepresentativeReviewCopyWith<$Res> {
  factory _$$CourseAiSummaryRepresentativeReviewImplCopyWith(
    _$CourseAiSummaryRepresentativeReviewImpl value,
    $Res Function(_$CourseAiSummaryRepresentativeReviewImpl) then,
  ) = __$$CourseAiSummaryRepresentativeReviewImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String excerpt, String sentiment});
}

/// @nodoc
class __$$CourseAiSummaryRepresentativeReviewImplCopyWithImpl<$Res>
    extends
        _$CourseAiSummaryRepresentativeReviewCopyWithImpl<
          $Res,
          _$CourseAiSummaryRepresentativeReviewImpl
        >
    implements _$$CourseAiSummaryRepresentativeReviewImplCopyWith<$Res> {
  __$$CourseAiSummaryRepresentativeReviewImplCopyWithImpl(
    _$CourseAiSummaryRepresentativeReviewImpl _value,
    $Res Function(_$CourseAiSummaryRepresentativeReviewImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseAiSummaryRepresentativeReview
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? excerpt = null, Object? sentiment = null}) {
    return _then(
      _$CourseAiSummaryRepresentativeReviewImpl(
        excerpt: null == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String,
        sentiment: null == sentiment
            ? _value.sentiment
            : sentiment // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseAiSummaryRepresentativeReviewImpl
    implements _CourseAiSummaryRepresentativeReview {
  const _$CourseAiSummaryRepresentativeReviewImpl({
    required this.excerpt,
    required this.sentiment,
  });

  factory _$CourseAiSummaryRepresentativeReviewImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$CourseAiSummaryRepresentativeReviewImplFromJson(json);

  @override
  final String excerpt;
  @override
  final String sentiment;

  @override
  String toString() {
    return 'CourseAiSummaryRepresentativeReview(excerpt: $excerpt, sentiment: $sentiment)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseAiSummaryRepresentativeReviewImpl &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt) &&
            (identical(other.sentiment, sentiment) ||
                other.sentiment == sentiment));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, excerpt, sentiment);

  /// Create a copy of CourseAiSummaryRepresentativeReview
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseAiSummaryRepresentativeReviewImplCopyWith<
    _$CourseAiSummaryRepresentativeReviewImpl
  >
  get copyWith =>
      __$$CourseAiSummaryRepresentativeReviewImplCopyWithImpl<
        _$CourseAiSummaryRepresentativeReviewImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseAiSummaryRepresentativeReviewImplToJson(this);
  }
}

abstract class _CourseAiSummaryRepresentativeReview
    implements CourseAiSummaryRepresentativeReview {
  const factory _CourseAiSummaryRepresentativeReview({
    required final String excerpt,
    required final String sentiment,
  }) = _$CourseAiSummaryRepresentativeReviewImpl;

  factory _CourseAiSummaryRepresentativeReview.fromJson(
    Map<String, dynamic> json,
  ) = _$CourseAiSummaryRepresentativeReviewImpl.fromJson;

  @override
  String get excerpt;
  @override
  String get sentiment;

  /// Create a copy of CourseAiSummaryRepresentativeReview
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseAiSummaryRepresentativeReviewImplCopyWith<
    _$CourseAiSummaryRepresentativeReviewImpl
  >
  get copyWith => throw _privateConstructorUsedError;
}

CourseAiSummaryPayload _$CourseAiSummaryPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CourseAiSummaryPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseAiSummaryPayload {
  String get consensus => throw _privateConstructorUsedError;
  List<String> get keywords => throw _privateConstructorUsedError;
  List<String> get pros => throw _privateConstructorUsedError;
  List<String> get cons => throw _privateConstructorUsedError;
  List<CourseAiSummaryRepresentativeReview> get representativeReviews =>
      throw _privateConstructorUsedError;

  /// Serializes this CourseAiSummaryPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseAiSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseAiSummaryPayloadCopyWith<CourseAiSummaryPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseAiSummaryPayloadCopyWith<$Res> {
  factory $CourseAiSummaryPayloadCopyWith(
    CourseAiSummaryPayload value,
    $Res Function(CourseAiSummaryPayload) then,
  ) = _$CourseAiSummaryPayloadCopyWithImpl<$Res, CourseAiSummaryPayload>;
  @useResult
  $Res call({
    String consensus,
    List<String> keywords,
    List<String> pros,
    List<String> cons,
    List<CourseAiSummaryRepresentativeReview> representativeReviews,
  });
}

/// @nodoc
class _$CourseAiSummaryPayloadCopyWithImpl<
  $Res,
  $Val extends CourseAiSummaryPayload
>
    implements $CourseAiSummaryPayloadCopyWith<$Res> {
  _$CourseAiSummaryPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseAiSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? consensus = null,
    Object? keywords = null,
    Object? pros = null,
    Object? cons = null,
    Object? representativeReviews = null,
  }) {
    return _then(
      _value.copyWith(
            consensus: null == consensus
                ? _value.consensus
                : consensus // ignore: cast_nullable_to_non_nullable
                      as String,
            keywords: null == keywords
                ? _value.keywords
                : keywords // ignore: cast_nullable_to_non_nullable
                      as List<String>,
            pros: null == pros
                ? _value.pros
                : pros // ignore: cast_nullable_to_non_nullable
                      as List<String>,
            cons: null == cons
                ? _value.cons
                : cons // ignore: cast_nullable_to_non_nullable
                      as List<String>,
            representativeReviews: null == representativeReviews
                ? _value.representativeReviews
                : representativeReviews // ignore: cast_nullable_to_non_nullable
                      as List<CourseAiSummaryRepresentativeReview>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseAiSummaryPayloadImplCopyWith<$Res>
    implements $CourseAiSummaryPayloadCopyWith<$Res> {
  factory _$$CourseAiSummaryPayloadImplCopyWith(
    _$CourseAiSummaryPayloadImpl value,
    $Res Function(_$CourseAiSummaryPayloadImpl) then,
  ) = __$$CourseAiSummaryPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String consensus,
    List<String> keywords,
    List<String> pros,
    List<String> cons,
    List<CourseAiSummaryRepresentativeReview> representativeReviews,
  });
}

/// @nodoc
class __$$CourseAiSummaryPayloadImplCopyWithImpl<$Res>
    extends
        _$CourseAiSummaryPayloadCopyWithImpl<$Res, _$CourseAiSummaryPayloadImpl>
    implements _$$CourseAiSummaryPayloadImplCopyWith<$Res> {
  __$$CourseAiSummaryPayloadImplCopyWithImpl(
    _$CourseAiSummaryPayloadImpl _value,
    $Res Function(_$CourseAiSummaryPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseAiSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? consensus = null,
    Object? keywords = null,
    Object? pros = null,
    Object? cons = null,
    Object? representativeReviews = null,
  }) {
    return _then(
      _$CourseAiSummaryPayloadImpl(
        consensus: null == consensus
            ? _value.consensus
            : consensus // ignore: cast_nullable_to_non_nullable
                  as String,
        keywords: null == keywords
            ? _value._keywords
            : keywords // ignore: cast_nullable_to_non_nullable
                  as List<String>,
        pros: null == pros
            ? _value._pros
            : pros // ignore: cast_nullable_to_non_nullable
                  as List<String>,
        cons: null == cons
            ? _value._cons
            : cons // ignore: cast_nullable_to_non_nullable
                  as List<String>,
        representativeReviews: null == representativeReviews
            ? _value._representativeReviews
            : representativeReviews // ignore: cast_nullable_to_non_nullable
                  as List<CourseAiSummaryRepresentativeReview>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseAiSummaryPayloadImpl implements _CourseAiSummaryPayload {
  const _$CourseAiSummaryPayloadImpl({
    required this.consensus,
    required final List<String> keywords,
    required final List<String> pros,
    required final List<String> cons,
    required final List<CourseAiSummaryRepresentativeReview>
    representativeReviews,
  }) : _keywords = keywords,
       _pros = pros,
       _cons = cons,
       _representativeReviews = representativeReviews;

  factory _$CourseAiSummaryPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseAiSummaryPayloadImplFromJson(json);

  @override
  final String consensus;
  final List<String> _keywords;
  @override
  List<String> get keywords {
    if (_keywords is EqualUnmodifiableListView) return _keywords;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_keywords);
  }

  final List<String> _pros;
  @override
  List<String> get pros {
    if (_pros is EqualUnmodifiableListView) return _pros;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_pros);
  }

  final List<String> _cons;
  @override
  List<String> get cons {
    if (_cons is EqualUnmodifiableListView) return _cons;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_cons);
  }

  final List<CourseAiSummaryRepresentativeReview> _representativeReviews;
  @override
  List<CourseAiSummaryRepresentativeReview> get representativeReviews {
    if (_representativeReviews is EqualUnmodifiableListView)
      return _representativeReviews;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_representativeReviews);
  }

  @override
  String toString() {
    return 'CourseAiSummaryPayload(consensus: $consensus, keywords: $keywords, pros: $pros, cons: $cons, representativeReviews: $representativeReviews)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseAiSummaryPayloadImpl &&
            (identical(other.consensus, consensus) ||
                other.consensus == consensus) &&
            const DeepCollectionEquality().equals(other._keywords, _keywords) &&
            const DeepCollectionEquality().equals(other._pros, _pros) &&
            const DeepCollectionEquality().equals(other._cons, _cons) &&
            const DeepCollectionEquality().equals(
              other._representativeReviews,
              _representativeReviews,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    consensus,
    const DeepCollectionEquality().hash(_keywords),
    const DeepCollectionEquality().hash(_pros),
    const DeepCollectionEquality().hash(_cons),
    const DeepCollectionEquality().hash(_representativeReviews),
  );

  /// Create a copy of CourseAiSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseAiSummaryPayloadImplCopyWith<_$CourseAiSummaryPayloadImpl>
  get copyWith =>
      __$$CourseAiSummaryPayloadImplCopyWithImpl<_$CourseAiSummaryPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseAiSummaryPayloadImplToJson(this);
  }
}

abstract class _CourseAiSummaryPayload implements CourseAiSummaryPayload {
  const factory _CourseAiSummaryPayload({
    required final String consensus,
    required final List<String> keywords,
    required final List<String> pros,
    required final List<String> cons,
    required final List<CourseAiSummaryRepresentativeReview>
    representativeReviews,
  }) = _$CourseAiSummaryPayloadImpl;

  factory _CourseAiSummaryPayload.fromJson(Map<String, dynamic> json) =
      _$CourseAiSummaryPayloadImpl.fromJson;

  @override
  String get consensus;
  @override
  List<String> get keywords;
  @override
  List<String> get pros;
  @override
  List<String> get cons;
  @override
  List<CourseAiSummaryRepresentativeReview> get representativeReviews;

  /// Create a copy of CourseAiSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseAiSummaryPayloadImplCopyWith<_$CourseAiSummaryPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseAiSummaryResult _$CourseAiSummaryResultFromJson(
  Map<String, dynamic> json,
) {
  return _CourseAiSummaryResult.fromJson(json);
}

/// @nodoc
mixin _$CourseAiSummaryResult {
  String get status => throw _privateConstructorUsedError;
  CourseAiSummaryPayload? get summary => throw _privateConstructorUsedError;
  String? get generatedAt => throw _privateConstructorUsedError;
  String? get model => throw _privateConstructorUsedError;

  /// Serializes this CourseAiSummaryResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseAiSummaryResultCopyWith<CourseAiSummaryResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseAiSummaryResultCopyWith<$Res> {
  factory $CourseAiSummaryResultCopyWith(
    CourseAiSummaryResult value,
    $Res Function(CourseAiSummaryResult) then,
  ) = _$CourseAiSummaryResultCopyWithImpl<$Res, CourseAiSummaryResult>;
  @useResult
  $Res call({
    String status,
    CourseAiSummaryPayload? summary,
    String? generatedAt,
    String? model,
  });

  $CourseAiSummaryPayloadCopyWith<$Res>? get summary;
}

/// @nodoc
class _$CourseAiSummaryResultCopyWithImpl<
  $Res,
  $Val extends CourseAiSummaryResult
>
    implements $CourseAiSummaryResultCopyWith<$Res> {
  _$CourseAiSummaryResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? status = null,
    Object? summary = freezed,
    Object? generatedAt = freezed,
    Object? model = freezed,
  }) {
    return _then(
      _value.copyWith(
            status: null == status
                ? _value.status
                : status // ignore: cast_nullable_to_non_nullable
                      as String,
            summary: freezed == summary
                ? _value.summary
                : summary // ignore: cast_nullable_to_non_nullable
                      as CourseAiSummaryPayload?,
            generatedAt: freezed == generatedAt
                ? _value.generatedAt
                : generatedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
            model: freezed == model
                ? _value.model
                : model // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $CourseAiSummaryPayloadCopyWith<$Res>? get summary {
    if (_value.summary == null) {
      return null;
    }

    return $CourseAiSummaryPayloadCopyWith<$Res>(_value.summary!, (value) {
      return _then(_value.copyWith(summary: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$CourseAiSummaryResultImplCopyWith<$Res>
    implements $CourseAiSummaryResultCopyWith<$Res> {
  factory _$$CourseAiSummaryResultImplCopyWith(
    _$CourseAiSummaryResultImpl value,
    $Res Function(_$CourseAiSummaryResultImpl) then,
  ) = __$$CourseAiSummaryResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String status,
    CourseAiSummaryPayload? summary,
    String? generatedAt,
    String? model,
  });

  @override
  $CourseAiSummaryPayloadCopyWith<$Res>? get summary;
}

/// @nodoc
class __$$CourseAiSummaryResultImplCopyWithImpl<$Res>
    extends
        _$CourseAiSummaryResultCopyWithImpl<$Res, _$CourseAiSummaryResultImpl>
    implements _$$CourseAiSummaryResultImplCopyWith<$Res> {
  __$$CourseAiSummaryResultImplCopyWithImpl(
    _$CourseAiSummaryResultImpl _value,
    $Res Function(_$CourseAiSummaryResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? status = null,
    Object? summary = freezed,
    Object? generatedAt = freezed,
    Object? model = freezed,
  }) {
    return _then(
      _$CourseAiSummaryResultImpl(
        status: null == status
            ? _value.status
            : status // ignore: cast_nullable_to_non_nullable
                  as String,
        summary: freezed == summary
            ? _value.summary
            : summary // ignore: cast_nullable_to_non_nullable
                  as CourseAiSummaryPayload?,
        generatedAt: freezed == generatedAt
            ? _value.generatedAt
            : generatedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
        model: freezed == model
            ? _value.model
            : model // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseAiSummaryResultImpl implements _CourseAiSummaryResult {
  const _$CourseAiSummaryResultImpl({
    required this.status,
    this.summary,
    this.generatedAt,
    this.model,
  });

  factory _$CourseAiSummaryResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseAiSummaryResultImplFromJson(json);

  @override
  final String status;
  @override
  final CourseAiSummaryPayload? summary;
  @override
  final String? generatedAt;
  @override
  final String? model;

  @override
  String toString() {
    return 'CourseAiSummaryResult(status: $status, summary: $summary, generatedAt: $generatedAt, model: $model)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseAiSummaryResultImpl &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.summary, summary) || other.summary == summary) &&
            (identical(other.generatedAt, generatedAt) ||
                other.generatedAt == generatedAt) &&
            (identical(other.model, model) || other.model == model));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, status, summary, generatedAt, model);

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseAiSummaryResultImplCopyWith<_$CourseAiSummaryResultImpl>
  get copyWith =>
      __$$CourseAiSummaryResultImplCopyWithImpl<_$CourseAiSummaryResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseAiSummaryResultImplToJson(this);
  }
}

abstract class _CourseAiSummaryResult implements CourseAiSummaryResult {
  const factory _CourseAiSummaryResult({
    required final String status,
    final CourseAiSummaryPayload? summary,
    final String? generatedAt,
    final String? model,
  }) = _$CourseAiSummaryResultImpl;

  factory _CourseAiSummaryResult.fromJson(Map<String, dynamic> json) =
      _$CourseAiSummaryResultImpl.fromJson;

  @override
  String get status;
  @override
  CourseAiSummaryPayload? get summary;
  @override
  String? get generatedAt;
  @override
  String? get model;

  /// Create a copy of CourseAiSummaryResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseAiSummaryResultImplCopyWith<_$CourseAiSummaryResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}
