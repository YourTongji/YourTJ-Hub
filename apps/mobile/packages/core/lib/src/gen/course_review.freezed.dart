// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'course_review.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

ReviewAuthorPayload _$ReviewAuthorPayloadFromJson(Map<String, dynamic> json) {
  return _ReviewAuthorPayload.fromJson(json);
}

/// @nodoc
mixin _$ReviewAuthorPayload {
  String get kind => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;

  /// Serializes this ReviewAuthorPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ReviewAuthorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReviewAuthorPayloadCopyWith<ReviewAuthorPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReviewAuthorPayloadCopyWith<$Res> {
  factory $ReviewAuthorPayloadCopyWith(
    ReviewAuthorPayload value,
    $Res Function(ReviewAuthorPayload) then,
  ) = _$ReviewAuthorPayloadCopyWithImpl<$Res, ReviewAuthorPayload>;
  @useResult
  $Res call({String kind, String label});
}

/// @nodoc
class _$ReviewAuthorPayloadCopyWithImpl<$Res, $Val extends ReviewAuthorPayload>
    implements $ReviewAuthorPayloadCopyWith<$Res> {
  _$ReviewAuthorPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReviewAuthorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? kind = null, Object? label = null}) {
    return _then(
      _value.copyWith(
            kind: null == kind
                ? _value.kind
                : kind // ignore: cast_nullable_to_non_nullable
                      as String,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ReviewAuthorPayloadImplCopyWith<$Res>
    implements $ReviewAuthorPayloadCopyWith<$Res> {
  factory _$$ReviewAuthorPayloadImplCopyWith(
    _$ReviewAuthorPayloadImpl value,
    $Res Function(_$ReviewAuthorPayloadImpl) then,
  ) = __$$ReviewAuthorPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String kind, String label});
}

/// @nodoc
class __$$ReviewAuthorPayloadImplCopyWithImpl<$Res>
    extends _$ReviewAuthorPayloadCopyWithImpl<$Res, _$ReviewAuthorPayloadImpl>
    implements _$$ReviewAuthorPayloadImplCopyWith<$Res> {
  __$$ReviewAuthorPayloadImplCopyWithImpl(
    _$ReviewAuthorPayloadImpl _value,
    $Res Function(_$ReviewAuthorPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ReviewAuthorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? kind = null, Object? label = null}) {
    return _then(
      _$ReviewAuthorPayloadImpl(
        kind: null == kind
            ? _value.kind
            : kind // ignore: cast_nullable_to_non_nullable
                  as String,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ReviewAuthorPayloadImpl implements _ReviewAuthorPayload {
  const _$ReviewAuthorPayloadImpl({required this.kind, required this.label});

  factory _$ReviewAuthorPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ReviewAuthorPayloadImplFromJson(json);

  @override
  final String kind;
  @override
  final String label;

  @override
  String toString() {
    return 'ReviewAuthorPayload(kind: $kind, label: $label)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReviewAuthorPayloadImpl &&
            (identical(other.kind, kind) || other.kind == kind) &&
            (identical(other.label, label) || other.label == label));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, kind, label);

  /// Create a copy of ReviewAuthorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReviewAuthorPayloadImplCopyWith<_$ReviewAuthorPayloadImpl> get copyWith =>
      __$$ReviewAuthorPayloadImplCopyWithImpl<_$ReviewAuthorPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ReviewAuthorPayloadImplToJson(this);
  }
}

abstract class _ReviewAuthorPayload implements ReviewAuthorPayload {
  const factory _ReviewAuthorPayload({
    required final String kind,
    required final String label,
  }) = _$ReviewAuthorPayloadImpl;

  factory _ReviewAuthorPayload.fromJson(Map<String, dynamic> json) =
      _$ReviewAuthorPayloadImpl.fromJson;

  @override
  String get kind;
  @override
  String get label;

  /// Create a copy of ReviewAuthorPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReviewAuthorPayloadImplCopyWith<_$ReviewAuthorPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ReviewViewerPayload _$ReviewViewerPayloadFromJson(Map<String, dynamic> json) {
  return _ReviewViewerPayload.fromJson(json);
}

/// @nodoc
mixin _$ReviewViewerPayload {
  bool get canEdit => throw _privateConstructorUsedError;
  bool get canDelete => throw _privateConstructorUsedError;
  bool get isHelpful => throw _privateConstructorUsedError;

  /// Serializes this ReviewViewerPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ReviewViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReviewViewerPayloadCopyWith<ReviewViewerPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReviewViewerPayloadCopyWith<$Res> {
  factory $ReviewViewerPayloadCopyWith(
    ReviewViewerPayload value,
    $Res Function(ReviewViewerPayload) then,
  ) = _$ReviewViewerPayloadCopyWithImpl<$Res, ReviewViewerPayload>;
  @useResult
  $Res call({bool canEdit, bool canDelete, bool isHelpful});
}

/// @nodoc
class _$ReviewViewerPayloadCopyWithImpl<$Res, $Val extends ReviewViewerPayload>
    implements $ReviewViewerPayloadCopyWith<$Res> {
  _$ReviewViewerPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReviewViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? canEdit = null,
    Object? canDelete = null,
    Object? isHelpful = null,
  }) {
    return _then(
      _value.copyWith(
            canEdit: null == canEdit
                ? _value.canEdit
                : canEdit // ignore: cast_nullable_to_non_nullable
                      as bool,
            canDelete: null == canDelete
                ? _value.canDelete
                : canDelete // ignore: cast_nullable_to_non_nullable
                      as bool,
            isHelpful: null == isHelpful
                ? _value.isHelpful
                : isHelpful // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ReviewViewerPayloadImplCopyWith<$Res>
    implements $ReviewViewerPayloadCopyWith<$Res> {
  factory _$$ReviewViewerPayloadImplCopyWith(
    _$ReviewViewerPayloadImpl value,
    $Res Function(_$ReviewViewerPayloadImpl) then,
  ) = __$$ReviewViewerPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool canEdit, bool canDelete, bool isHelpful});
}

/// @nodoc
class __$$ReviewViewerPayloadImplCopyWithImpl<$Res>
    extends _$ReviewViewerPayloadCopyWithImpl<$Res, _$ReviewViewerPayloadImpl>
    implements _$$ReviewViewerPayloadImplCopyWith<$Res> {
  __$$ReviewViewerPayloadImplCopyWithImpl(
    _$ReviewViewerPayloadImpl _value,
    $Res Function(_$ReviewViewerPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ReviewViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? canEdit = null,
    Object? canDelete = null,
    Object? isHelpful = null,
  }) {
    return _then(
      _$ReviewViewerPayloadImpl(
        canEdit: null == canEdit
            ? _value.canEdit
            : canEdit // ignore: cast_nullable_to_non_nullable
                  as bool,
        canDelete: null == canDelete
            ? _value.canDelete
            : canDelete // ignore: cast_nullable_to_non_nullable
                  as bool,
        isHelpful: null == isHelpful
            ? _value.isHelpful
            : isHelpful // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ReviewViewerPayloadImpl implements _ReviewViewerPayload {
  const _$ReviewViewerPayloadImpl({
    required this.canEdit,
    required this.canDelete,
    required this.isHelpful,
  });

  factory _$ReviewViewerPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ReviewViewerPayloadImplFromJson(json);

  @override
  final bool canEdit;
  @override
  final bool canDelete;
  @override
  final bool isHelpful;

  @override
  String toString() {
    return 'ReviewViewerPayload(canEdit: $canEdit, canDelete: $canDelete, isHelpful: $isHelpful)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReviewViewerPayloadImpl &&
            (identical(other.canEdit, canEdit) || other.canEdit == canEdit) &&
            (identical(other.canDelete, canDelete) ||
                other.canDelete == canDelete) &&
            (identical(other.isHelpful, isHelpful) ||
                other.isHelpful == isHelpful));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, canEdit, canDelete, isHelpful);

  /// Create a copy of ReviewViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReviewViewerPayloadImplCopyWith<_$ReviewViewerPayloadImpl> get copyWith =>
      __$$ReviewViewerPayloadImplCopyWithImpl<_$ReviewViewerPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ReviewViewerPayloadImplToJson(this);
  }
}

abstract class _ReviewViewerPayload implements ReviewViewerPayload {
  const factory _ReviewViewerPayload({
    required final bool canEdit,
    required final bool canDelete,
    required final bool isHelpful,
  }) = _$ReviewViewerPayloadImpl;

  factory _ReviewViewerPayload.fromJson(Map<String, dynamic> json) =
      _$ReviewViewerPayloadImpl.fromJson;

  @override
  bool get canEdit;
  @override
  bool get canDelete;
  @override
  bool get isHelpful;

  /// Create a copy of ReviewViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReviewViewerPayloadImplCopyWith<_$ReviewViewerPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ReviewPayload _$ReviewPayloadFromJson(Map<String, dynamic> json) {
  return _ReviewPayload.fromJson(json);
}

/// @nodoc
mixin _$ReviewPayload {
  int get id => throw _privateConstructorUsedError;
  int get offeringId => throw _privateConstructorUsedError;
  int? get rating => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  String get contentHtml => throw _privateConstructorUsedError;
  ReviewAuthorPayload get author => throw _privateConstructorUsedError;
  ReviewViewerPayload get viewer => throw _privateConstructorUsedError;
  int get helpfulCount => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  String get updatedAt => throw _privateConstructorUsedError;
  double? get offeringRatingAvg => throw _privateConstructorUsedError;
  int? get offeringReviewCount => throw _privateConstructorUsedError;

  /// Serializes this ReviewPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReviewPayloadCopyWith<ReviewPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReviewPayloadCopyWith<$Res> {
  factory $ReviewPayloadCopyWith(
    ReviewPayload value,
    $Res Function(ReviewPayload) then,
  ) = _$ReviewPayloadCopyWithImpl<$Res, ReviewPayload>;
  @useResult
  $Res call({
    int id,
    int offeringId,
    int? rating,
    String content,
    String contentHtml,
    ReviewAuthorPayload author,
    ReviewViewerPayload viewer,
    int helpfulCount,
    String createdAt,
    String updatedAt,
    double? offeringRatingAvg,
    int? offeringReviewCount,
  });

  $ReviewAuthorPayloadCopyWith<$Res> get author;
  $ReviewViewerPayloadCopyWith<$Res> get viewer;
}

/// @nodoc
class _$ReviewPayloadCopyWithImpl<$Res, $Val extends ReviewPayload>
    implements $ReviewPayloadCopyWith<$Res> {
  _$ReviewPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? offeringId = null,
    Object? rating = freezed,
    Object? content = null,
    Object? contentHtml = null,
    Object? author = null,
    Object? viewer = null,
    Object? helpfulCount = null,
    Object? createdAt = null,
    Object? updatedAt = null,
    Object? offeringRatingAvg = freezed,
    Object? offeringReviewCount = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            offeringId: null == offeringId
                ? _value.offeringId
                : offeringId // ignore: cast_nullable_to_non_nullable
                      as int,
            rating: freezed == rating
                ? _value.rating
                : rating // ignore: cast_nullable_to_non_nullable
                      as int?,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            contentHtml: null == contentHtml
                ? _value.contentHtml
                : contentHtml // ignore: cast_nullable_to_non_nullable
                      as String,
            author: null == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as ReviewAuthorPayload,
            viewer: null == viewer
                ? _value.viewer
                : viewer // ignore: cast_nullable_to_non_nullable
                      as ReviewViewerPayload,
            helpfulCount: null == helpfulCount
                ? _value.helpfulCount
                : helpfulCount // ignore: cast_nullable_to_non_nullable
                      as int,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String,
            offeringRatingAvg: freezed == offeringRatingAvg
                ? _value.offeringRatingAvg
                : offeringRatingAvg // ignore: cast_nullable_to_non_nullable
                      as double?,
            offeringReviewCount: freezed == offeringReviewCount
                ? _value.offeringReviewCount
                : offeringReviewCount // ignore: cast_nullable_to_non_nullable
                      as int?,
          )
          as $Val,
    );
  }

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ReviewAuthorPayloadCopyWith<$Res> get author {
    return $ReviewAuthorPayloadCopyWith<$Res>(_value.author, (value) {
      return _then(_value.copyWith(author: value) as $Val);
    });
  }

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ReviewViewerPayloadCopyWith<$Res> get viewer {
    return $ReviewViewerPayloadCopyWith<$Res>(_value.viewer, (value) {
      return _then(_value.copyWith(viewer: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ReviewPayloadImplCopyWith<$Res>
    implements $ReviewPayloadCopyWith<$Res> {
  factory _$$ReviewPayloadImplCopyWith(
    _$ReviewPayloadImpl value,
    $Res Function(_$ReviewPayloadImpl) then,
  ) = __$$ReviewPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int offeringId,
    int? rating,
    String content,
    String contentHtml,
    ReviewAuthorPayload author,
    ReviewViewerPayload viewer,
    int helpfulCount,
    String createdAt,
    String updatedAt,
    double? offeringRatingAvg,
    int? offeringReviewCount,
  });

  @override
  $ReviewAuthorPayloadCopyWith<$Res> get author;
  @override
  $ReviewViewerPayloadCopyWith<$Res> get viewer;
}

/// @nodoc
class __$$ReviewPayloadImplCopyWithImpl<$Res>
    extends _$ReviewPayloadCopyWithImpl<$Res, _$ReviewPayloadImpl>
    implements _$$ReviewPayloadImplCopyWith<$Res> {
  __$$ReviewPayloadImplCopyWithImpl(
    _$ReviewPayloadImpl _value,
    $Res Function(_$ReviewPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? offeringId = null,
    Object? rating = freezed,
    Object? content = null,
    Object? contentHtml = null,
    Object? author = null,
    Object? viewer = null,
    Object? helpfulCount = null,
    Object? createdAt = null,
    Object? updatedAt = null,
    Object? offeringRatingAvg = freezed,
    Object? offeringReviewCount = freezed,
  }) {
    return _then(
      _$ReviewPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        offeringId: null == offeringId
            ? _value.offeringId
            : offeringId // ignore: cast_nullable_to_non_nullable
                  as int,
        rating: freezed == rating
            ? _value.rating
            : rating // ignore: cast_nullable_to_non_nullable
                  as int?,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        contentHtml: null == contentHtml
            ? _value.contentHtml
            : contentHtml // ignore: cast_nullable_to_non_nullable
                  as String,
        author: null == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as ReviewAuthorPayload,
        viewer: null == viewer
            ? _value.viewer
            : viewer // ignore: cast_nullable_to_non_nullable
                  as ReviewViewerPayload,
        helpfulCount: null == helpfulCount
            ? _value.helpfulCount
            : helpfulCount // ignore: cast_nullable_to_non_nullable
                  as int,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String,
        offeringRatingAvg: freezed == offeringRatingAvg
            ? _value.offeringRatingAvg
            : offeringRatingAvg // ignore: cast_nullable_to_non_nullable
                  as double?,
        offeringReviewCount: freezed == offeringReviewCount
            ? _value.offeringReviewCount
            : offeringReviewCount // ignore: cast_nullable_to_non_nullable
                  as int?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ReviewPayloadImpl implements _ReviewPayload {
  const _$ReviewPayloadImpl({
    required this.id,
    required this.offeringId,
    this.rating,
    required this.content,
    required this.contentHtml,
    required this.author,
    required this.viewer,
    required this.helpfulCount,
    required this.createdAt,
    required this.updatedAt,
    this.offeringRatingAvg,
    this.offeringReviewCount,
  });

  factory _$ReviewPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ReviewPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int offeringId;
  @override
  final int? rating;
  @override
  final String content;
  @override
  final String contentHtml;
  @override
  final ReviewAuthorPayload author;
  @override
  final ReviewViewerPayload viewer;
  @override
  final int helpfulCount;
  @override
  final String createdAt;
  @override
  final String updatedAt;
  @override
  final double? offeringRatingAvg;
  @override
  final int? offeringReviewCount;

  @override
  String toString() {
    return 'ReviewPayload(id: $id, offeringId: $offeringId, rating: $rating, content: $content, contentHtml: $contentHtml, author: $author, viewer: $viewer, helpfulCount: $helpfulCount, createdAt: $createdAt, updatedAt: $updatedAt, offeringRatingAvg: $offeringRatingAvg, offeringReviewCount: $offeringReviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReviewPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.offeringId, offeringId) ||
                other.offeringId == offeringId) &&
            (identical(other.rating, rating) || other.rating == rating) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.contentHtml, contentHtml) ||
                other.contentHtml == contentHtml) &&
            (identical(other.author, author) || other.author == author) &&
            (identical(other.viewer, viewer) || other.viewer == viewer) &&
            (identical(other.helpfulCount, helpfulCount) ||
                other.helpfulCount == helpfulCount) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.offeringRatingAvg, offeringRatingAvg) ||
                other.offeringRatingAvg == offeringRatingAvg) &&
            (identical(other.offeringReviewCount, offeringReviewCount) ||
                other.offeringReviewCount == offeringReviewCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    offeringId,
    rating,
    content,
    contentHtml,
    author,
    viewer,
    helpfulCount,
    createdAt,
    updatedAt,
    offeringRatingAvg,
    offeringReviewCount,
  );

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReviewPayloadImplCopyWith<_$ReviewPayloadImpl> get copyWith =>
      __$$ReviewPayloadImplCopyWithImpl<_$ReviewPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ReviewPayloadImplToJson(this);
  }
}

abstract class _ReviewPayload implements ReviewPayload {
  const factory _ReviewPayload({
    required final int id,
    required final int offeringId,
    final int? rating,
    required final String content,
    required final String contentHtml,
    required final ReviewAuthorPayload author,
    required final ReviewViewerPayload viewer,
    required final int helpfulCount,
    required final String createdAt,
    required final String updatedAt,
    final double? offeringRatingAvg,
    final int? offeringReviewCount,
  }) = _$ReviewPayloadImpl;

  factory _ReviewPayload.fromJson(Map<String, dynamic> json) =
      _$ReviewPayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get offeringId;
  @override
  int? get rating;
  @override
  String get content;
  @override
  String get contentHtml;
  @override
  ReviewAuthorPayload get author;
  @override
  ReviewViewerPayload get viewer;
  @override
  int get helpfulCount;
  @override
  String get createdAt;
  @override
  String get updatedAt;
  @override
  double? get offeringRatingAvg;
  @override
  int? get offeringReviewCount;

  /// Create a copy of ReviewPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReviewPayloadImplCopyWith<_$ReviewPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ReviewListResult _$ReviewListResultFromJson(Map<String, dynamic> json) {
  return _ReviewListResult.fromJson(json);
}

/// @nodoc
mixin _$ReviewListResult {
  List<ReviewPayload> get list => throw _privateConstructorUsedError;
  String? get nextCursor => throw _privateConstructorUsedError;
  int get total => throw _privateConstructorUsedError;

  /// Serializes this ReviewListResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ReviewListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReviewListResultCopyWith<ReviewListResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReviewListResultCopyWith<$Res> {
  factory $ReviewListResultCopyWith(
    ReviewListResult value,
    $Res Function(ReviewListResult) then,
  ) = _$ReviewListResultCopyWithImpl<$Res, ReviewListResult>;
  @useResult
  $Res call({List<ReviewPayload> list, String? nextCursor, int total});
}

/// @nodoc
class _$ReviewListResultCopyWithImpl<$Res, $Val extends ReviewListResult>
    implements $ReviewListResultCopyWith<$Res> {
  _$ReviewListResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReviewListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? nextCursor = freezed,
    Object? total = null,
  }) {
    return _then(
      _value.copyWith(
            list: null == list
                ? _value.list
                : list // ignore: cast_nullable_to_non_nullable
                      as List<ReviewPayload>,
            nextCursor: freezed == nextCursor
                ? _value.nextCursor
                : nextCursor // ignore: cast_nullable_to_non_nullable
                      as String?,
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ReviewListResultImplCopyWith<$Res>
    implements $ReviewListResultCopyWith<$Res> {
  factory _$$ReviewListResultImplCopyWith(
    _$ReviewListResultImpl value,
    $Res Function(_$ReviewListResultImpl) then,
  ) = __$$ReviewListResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<ReviewPayload> list, String? nextCursor, int total});
}

/// @nodoc
class __$$ReviewListResultImplCopyWithImpl<$Res>
    extends _$ReviewListResultCopyWithImpl<$Res, _$ReviewListResultImpl>
    implements _$$ReviewListResultImplCopyWith<$Res> {
  __$$ReviewListResultImplCopyWithImpl(
    _$ReviewListResultImpl _value,
    $Res Function(_$ReviewListResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ReviewListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? nextCursor = freezed,
    Object? total = null,
  }) {
    return _then(
      _$ReviewListResultImpl(
        list: null == list
            ? _value._list
            : list // ignore: cast_nullable_to_non_nullable
                  as List<ReviewPayload>,
        nextCursor: freezed == nextCursor
            ? _value.nextCursor
            : nextCursor // ignore: cast_nullable_to_non_nullable
                  as String?,
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ReviewListResultImpl implements _ReviewListResult {
  const _$ReviewListResultImpl({
    required final List<ReviewPayload> list,
    this.nextCursor,
    required this.total,
  }) : _list = list;

  factory _$ReviewListResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$ReviewListResultImplFromJson(json);

  final List<ReviewPayload> _list;
  @override
  List<ReviewPayload> get list {
    if (_list is EqualUnmodifiableListView) return _list;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_list);
  }

  @override
  final String? nextCursor;
  @override
  final int total;

  @override
  String toString() {
    return 'ReviewListResult(list: $list, nextCursor: $nextCursor, total: $total)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReviewListResultImpl &&
            const DeepCollectionEquality().equals(other._list, _list) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor) &&
            (identical(other.total, total) || other.total == total));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_list),
    nextCursor,
    total,
  );

  /// Create a copy of ReviewListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReviewListResultImplCopyWith<_$ReviewListResultImpl> get copyWith =>
      __$$ReviewListResultImplCopyWithImpl<_$ReviewListResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ReviewListResultImplToJson(this);
  }
}

abstract class _ReviewListResult implements ReviewListResult {
  const factory _ReviewListResult({
    required final List<ReviewPayload> list,
    final String? nextCursor,
    required final int total,
  }) = _$ReviewListResultImpl;

  factory _ReviewListResult.fromJson(Map<String, dynamic> json) =
      _$ReviewListResultImpl.fromJson;

  @override
  List<ReviewPayload> get list;
  @override
  String? get nextCursor;
  @override
  int get total;

  /// Create a copy of ReviewListResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReviewListResultImplCopyWith<_$ReviewListResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CreateCourseReviewInput _$CreateCourseReviewInputFromJson(
  Map<String, dynamic> json,
) {
  return _CreateCourseReviewInput.fromJson(json);
}

/// @nodoc
mixin _$CreateCourseReviewInput {
  int get offeringId => throw _privateConstructorUsedError;
  int get rating => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  bool get isAnonymous => throw _privateConstructorUsedError;

  /// Serializes this CreateCourseReviewInput to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CreateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CreateCourseReviewInputCopyWith<CreateCourseReviewInput> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CreateCourseReviewInputCopyWith<$Res> {
  factory $CreateCourseReviewInputCopyWith(
    CreateCourseReviewInput value,
    $Res Function(CreateCourseReviewInput) then,
  ) = _$CreateCourseReviewInputCopyWithImpl<$Res, CreateCourseReviewInput>;
  @useResult
  $Res call({int offeringId, int rating, String content, bool isAnonymous});
}

/// @nodoc
class _$CreateCourseReviewInputCopyWithImpl<
  $Res,
  $Val extends CreateCourseReviewInput
>
    implements $CreateCourseReviewInputCopyWith<$Res> {
  _$CreateCourseReviewInputCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CreateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? offeringId = null,
    Object? rating = null,
    Object? content = null,
    Object? isAnonymous = null,
  }) {
    return _then(
      _value.copyWith(
            offeringId: null == offeringId
                ? _value.offeringId
                : offeringId // ignore: cast_nullable_to_non_nullable
                      as int,
            rating: null == rating
                ? _value.rating
                : rating // ignore: cast_nullable_to_non_nullable
                      as int,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            isAnonymous: null == isAnonymous
                ? _value.isAnonymous
                : isAnonymous // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CreateCourseReviewInputImplCopyWith<$Res>
    implements $CreateCourseReviewInputCopyWith<$Res> {
  factory _$$CreateCourseReviewInputImplCopyWith(
    _$CreateCourseReviewInputImpl value,
    $Res Function(_$CreateCourseReviewInputImpl) then,
  ) = __$$CreateCourseReviewInputImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int offeringId, int rating, String content, bool isAnonymous});
}

/// @nodoc
class __$$CreateCourseReviewInputImplCopyWithImpl<$Res>
    extends
        _$CreateCourseReviewInputCopyWithImpl<
          $Res,
          _$CreateCourseReviewInputImpl
        >
    implements _$$CreateCourseReviewInputImplCopyWith<$Res> {
  __$$CreateCourseReviewInputImplCopyWithImpl(
    _$CreateCourseReviewInputImpl _value,
    $Res Function(_$CreateCourseReviewInputImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CreateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? offeringId = null,
    Object? rating = null,
    Object? content = null,
    Object? isAnonymous = null,
  }) {
    return _then(
      _$CreateCourseReviewInputImpl(
        offeringId: null == offeringId
            ? _value.offeringId
            : offeringId // ignore: cast_nullable_to_non_nullable
                  as int,
        rating: null == rating
            ? _value.rating
            : rating // ignore: cast_nullable_to_non_nullable
                  as int,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        isAnonymous: null == isAnonymous
            ? _value.isAnonymous
            : isAnonymous // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CreateCourseReviewInputImpl implements _CreateCourseReviewInput {
  const _$CreateCourseReviewInputImpl({
    required this.offeringId,
    required this.rating,
    required this.content,
    required this.isAnonymous,
  });

  factory _$CreateCourseReviewInputImpl.fromJson(Map<String, dynamic> json) =>
      _$$CreateCourseReviewInputImplFromJson(json);

  @override
  final int offeringId;
  @override
  final int rating;
  @override
  final String content;
  @override
  final bool isAnonymous;

  @override
  String toString() {
    return 'CreateCourseReviewInput(offeringId: $offeringId, rating: $rating, content: $content, isAnonymous: $isAnonymous)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CreateCourseReviewInputImpl &&
            (identical(other.offeringId, offeringId) ||
                other.offeringId == offeringId) &&
            (identical(other.rating, rating) || other.rating == rating) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.isAnonymous, isAnonymous) ||
                other.isAnonymous == isAnonymous));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, offeringId, rating, content, isAnonymous);

  /// Create a copy of CreateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CreateCourseReviewInputImplCopyWith<_$CreateCourseReviewInputImpl>
  get copyWith =>
      __$$CreateCourseReviewInputImplCopyWithImpl<
        _$CreateCourseReviewInputImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CreateCourseReviewInputImplToJson(this);
  }
}

abstract class _CreateCourseReviewInput implements CreateCourseReviewInput {
  const factory _CreateCourseReviewInput({
    required final int offeringId,
    required final int rating,
    required final String content,
    required final bool isAnonymous,
  }) = _$CreateCourseReviewInputImpl;

  factory _CreateCourseReviewInput.fromJson(Map<String, dynamic> json) =
      _$CreateCourseReviewInputImpl.fromJson;

  @override
  int get offeringId;
  @override
  int get rating;
  @override
  String get content;
  @override
  bool get isAnonymous;

  /// Create a copy of CreateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CreateCourseReviewInputImplCopyWith<_$CreateCourseReviewInputImpl>
  get copyWith => throw _privateConstructorUsedError;
}

UpdateCourseReviewInput _$UpdateCourseReviewInputFromJson(
  Map<String, dynamic> json,
) {
  return _UpdateCourseReviewInput.fromJson(json);
}

/// @nodoc
mixin _$UpdateCourseReviewInput {
  int? get rating => throw _privateConstructorUsedError;
  String? get content => throw _privateConstructorUsedError;
  bool? get isAnonymous => throw _privateConstructorUsedError;

  /// Serializes this UpdateCourseReviewInput to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UpdateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UpdateCourseReviewInputCopyWith<UpdateCourseReviewInput> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UpdateCourseReviewInputCopyWith<$Res> {
  factory $UpdateCourseReviewInputCopyWith(
    UpdateCourseReviewInput value,
    $Res Function(UpdateCourseReviewInput) then,
  ) = _$UpdateCourseReviewInputCopyWithImpl<$Res, UpdateCourseReviewInput>;
  @useResult
  $Res call({int? rating, String? content, bool? isAnonymous});
}

/// @nodoc
class _$UpdateCourseReviewInputCopyWithImpl<
  $Res,
  $Val extends UpdateCourseReviewInput
>
    implements $UpdateCourseReviewInputCopyWith<$Res> {
  _$UpdateCourseReviewInputCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UpdateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? rating = freezed,
    Object? content = freezed,
    Object? isAnonymous = freezed,
  }) {
    return _then(
      _value.copyWith(
            rating: freezed == rating
                ? _value.rating
                : rating // ignore: cast_nullable_to_non_nullable
                      as int?,
            content: freezed == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String?,
            isAnonymous: freezed == isAnonymous
                ? _value.isAnonymous
                : isAnonymous // ignore: cast_nullable_to_non_nullable
                      as bool?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UpdateCourseReviewInputImplCopyWith<$Res>
    implements $UpdateCourseReviewInputCopyWith<$Res> {
  factory _$$UpdateCourseReviewInputImplCopyWith(
    _$UpdateCourseReviewInputImpl value,
    $Res Function(_$UpdateCourseReviewInputImpl) then,
  ) = __$$UpdateCourseReviewInputImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int? rating, String? content, bool? isAnonymous});
}

/// @nodoc
class __$$UpdateCourseReviewInputImplCopyWithImpl<$Res>
    extends
        _$UpdateCourseReviewInputCopyWithImpl<
          $Res,
          _$UpdateCourseReviewInputImpl
        >
    implements _$$UpdateCourseReviewInputImplCopyWith<$Res> {
  __$$UpdateCourseReviewInputImplCopyWithImpl(
    _$UpdateCourseReviewInputImpl _value,
    $Res Function(_$UpdateCourseReviewInputImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UpdateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? rating = freezed,
    Object? content = freezed,
    Object? isAnonymous = freezed,
  }) {
    return _then(
      _$UpdateCourseReviewInputImpl(
        rating: freezed == rating
            ? _value.rating
            : rating // ignore: cast_nullable_to_non_nullable
                  as int?,
        content: freezed == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String?,
        isAnonymous: freezed == isAnonymous
            ? _value.isAnonymous
            : isAnonymous // ignore: cast_nullable_to_non_nullable
                  as bool?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UpdateCourseReviewInputImpl implements _UpdateCourseReviewInput {
  const _$UpdateCourseReviewInputImpl({
    this.rating,
    this.content,
    this.isAnonymous,
  });

  factory _$UpdateCourseReviewInputImpl.fromJson(Map<String, dynamic> json) =>
      _$$UpdateCourseReviewInputImplFromJson(json);

  @override
  final int? rating;
  @override
  final String? content;
  @override
  final bool? isAnonymous;

  @override
  String toString() {
    return 'UpdateCourseReviewInput(rating: $rating, content: $content, isAnonymous: $isAnonymous)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UpdateCourseReviewInputImpl &&
            (identical(other.rating, rating) || other.rating == rating) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.isAnonymous, isAnonymous) ||
                other.isAnonymous == isAnonymous));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, rating, content, isAnonymous);

  /// Create a copy of UpdateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UpdateCourseReviewInputImplCopyWith<_$UpdateCourseReviewInputImpl>
  get copyWith =>
      __$$UpdateCourseReviewInputImplCopyWithImpl<
        _$UpdateCourseReviewInputImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$UpdateCourseReviewInputImplToJson(this);
  }
}

abstract class _UpdateCourseReviewInput implements UpdateCourseReviewInput {
  const factory _UpdateCourseReviewInput({
    final int? rating,
    final String? content,
    final bool? isAnonymous,
  }) = _$UpdateCourseReviewInputImpl;

  factory _UpdateCourseReviewInput.fromJson(Map<String, dynamic> json) =
      _$UpdateCourseReviewInputImpl.fromJson;

  @override
  int? get rating;
  @override
  String? get content;
  @override
  bool? get isAnonymous;

  /// Create a copy of UpdateCourseReviewInput
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UpdateCourseReviewInputImplCopyWith<_$UpdateCourseReviewInputImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ModerationCourseReviewReportItem _$ModerationCourseReviewReportItemFromJson(
  Map<String, dynamic> json,
) {
  return _ModerationCourseReviewReportItem.fromJson(json);
}

/// @nodoc
mixin _$ModerationCourseReviewReportItem {
  int get id => throw _privateConstructorUsedError;
  int get reviewId => throw _privateConstructorUsedError;
  String get reason => throw _privateConstructorUsedError;
  String get note => throw _privateConstructorUsedError;
  String get status => throw _privateConstructorUsedError;
  String get resolution => throw _privateConstructorUsedError;
  String get excerpt => throw _privateConstructorUsedError;
  UserBriefPayload get reporter => throw _privateConstructorUsedError;
  UserBriefPayload get handler => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  String? get handledAt => throw _privateConstructorUsedError;
  int get reportCount => throw _privateConstructorUsedError;

  /// Serializes this ModerationCourseReviewReportItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationCourseReviewReportItemCopyWith<ModerationCourseReviewReportItem>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationCourseReviewReportItemCopyWith<$Res> {
  factory $ModerationCourseReviewReportItemCopyWith(
    ModerationCourseReviewReportItem value,
    $Res Function(ModerationCourseReviewReportItem) then,
  ) =
      _$ModerationCourseReviewReportItemCopyWithImpl<
        $Res,
        ModerationCourseReviewReportItem
      >;
  @useResult
  $Res call({
    int id,
    int reviewId,
    String reason,
    String note,
    String status,
    String resolution,
    String excerpt,
    UserBriefPayload reporter,
    UserBriefPayload handler,
    String createdAt,
    String? handledAt,
    int reportCount,
  });

  $UserBriefPayloadCopyWith<$Res> get reporter;
  $UserBriefPayloadCopyWith<$Res> get handler;
}

/// @nodoc
class _$ModerationCourseReviewReportItemCopyWithImpl<
  $Res,
  $Val extends ModerationCourseReviewReportItem
>
    implements $ModerationCourseReviewReportItemCopyWith<$Res> {
  _$ModerationCourseReviewReportItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? reviewId = null,
    Object? reason = null,
    Object? note = null,
    Object? status = null,
    Object? resolution = null,
    Object? excerpt = null,
    Object? reporter = null,
    Object? handler = null,
    Object? createdAt = null,
    Object? handledAt = freezed,
    Object? reportCount = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            reviewId: null == reviewId
                ? _value.reviewId
                : reviewId // ignore: cast_nullable_to_non_nullable
                      as int,
            reason: null == reason
                ? _value.reason
                : reason // ignore: cast_nullable_to_non_nullable
                      as String,
            note: null == note
                ? _value.note
                : note // ignore: cast_nullable_to_non_nullable
                      as String,
            status: null == status
                ? _value.status
                : status // ignore: cast_nullable_to_non_nullable
                      as String,
            resolution: null == resolution
                ? _value.resolution
                : resolution // ignore: cast_nullable_to_non_nullable
                      as String,
            excerpt: null == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String,
            reporter: null == reporter
                ? _value.reporter
                : reporter // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            handler: null == handler
                ? _value.handler
                : handler // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            handledAt: freezed == handledAt
                ? _value.handledAt
                : handledAt // ignore: cast_nullable_to_non_nullable
                      as String?,
            reportCount: null == reportCount
                ? _value.reportCount
                : reportCount // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get reporter {
    return $UserBriefPayloadCopyWith<$Res>(_value.reporter, (value) {
      return _then(_value.copyWith(reporter: value) as $Val);
    });
  }

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get handler {
    return $UserBriefPayloadCopyWith<$Res>(_value.handler, (value) {
      return _then(_value.copyWith(handler: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ModerationCourseReviewReportItemImplCopyWith<$Res>
    implements $ModerationCourseReviewReportItemCopyWith<$Res> {
  factory _$$ModerationCourseReviewReportItemImplCopyWith(
    _$ModerationCourseReviewReportItemImpl value,
    $Res Function(_$ModerationCourseReviewReportItemImpl) then,
  ) = __$$ModerationCourseReviewReportItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int reviewId,
    String reason,
    String note,
    String status,
    String resolution,
    String excerpt,
    UserBriefPayload reporter,
    UserBriefPayload handler,
    String createdAt,
    String? handledAt,
    int reportCount,
  });

  @override
  $UserBriefPayloadCopyWith<$Res> get reporter;
  @override
  $UserBriefPayloadCopyWith<$Res> get handler;
}

/// @nodoc
class __$$ModerationCourseReviewReportItemImplCopyWithImpl<$Res>
    extends
        _$ModerationCourseReviewReportItemCopyWithImpl<
          $Res,
          _$ModerationCourseReviewReportItemImpl
        >
    implements _$$ModerationCourseReviewReportItemImplCopyWith<$Res> {
  __$$ModerationCourseReviewReportItemImplCopyWithImpl(
    _$ModerationCourseReviewReportItemImpl _value,
    $Res Function(_$ModerationCourseReviewReportItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? reviewId = null,
    Object? reason = null,
    Object? note = null,
    Object? status = null,
    Object? resolution = null,
    Object? excerpt = null,
    Object? reporter = null,
    Object? handler = null,
    Object? createdAt = null,
    Object? handledAt = freezed,
    Object? reportCount = null,
  }) {
    return _then(
      _$ModerationCourseReviewReportItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        reviewId: null == reviewId
            ? _value.reviewId
            : reviewId // ignore: cast_nullable_to_non_nullable
                  as int,
        reason: null == reason
            ? _value.reason
            : reason // ignore: cast_nullable_to_non_nullable
                  as String,
        note: null == note
            ? _value.note
            : note // ignore: cast_nullable_to_non_nullable
                  as String,
        status: null == status
            ? _value.status
            : status // ignore: cast_nullable_to_non_nullable
                  as String,
        resolution: null == resolution
            ? _value.resolution
            : resolution // ignore: cast_nullable_to_non_nullable
                  as String,
        excerpt: null == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String,
        reporter: null == reporter
            ? _value.reporter
            : reporter // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        handler: null == handler
            ? _value.handler
            : handler // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        handledAt: freezed == handledAt
            ? _value.handledAt
            : handledAt // ignore: cast_nullable_to_non_nullable
                  as String?,
        reportCount: null == reportCount
            ? _value.reportCount
            : reportCount // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationCourseReviewReportItemImpl
    implements _ModerationCourseReviewReportItem {
  const _$ModerationCourseReviewReportItemImpl({
    required this.id,
    required this.reviewId,
    required this.reason,
    required this.note,
    required this.status,
    required this.resolution,
    required this.excerpt,
    required this.reporter,
    required this.handler,
    required this.createdAt,
    this.handledAt,
    required this.reportCount,
  });

  factory _$ModerationCourseReviewReportItemImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$ModerationCourseReviewReportItemImplFromJson(json);

  @override
  final int id;
  @override
  final int reviewId;
  @override
  final String reason;
  @override
  final String note;
  @override
  final String status;
  @override
  final String resolution;
  @override
  final String excerpt;
  @override
  final UserBriefPayload reporter;
  @override
  final UserBriefPayload handler;
  @override
  final String createdAt;
  @override
  final String? handledAt;
  @override
  final int reportCount;

  @override
  String toString() {
    return 'ModerationCourseReviewReportItem(id: $id, reviewId: $reviewId, reason: $reason, note: $note, status: $status, resolution: $resolution, excerpt: $excerpt, reporter: $reporter, handler: $handler, createdAt: $createdAt, handledAt: $handledAt, reportCount: $reportCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationCourseReviewReportItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.reviewId, reviewId) ||
                other.reviewId == reviewId) &&
            (identical(other.reason, reason) || other.reason == reason) &&
            (identical(other.note, note) || other.note == note) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.resolution, resolution) ||
                other.resolution == resolution) &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt) &&
            (identical(other.reporter, reporter) ||
                other.reporter == reporter) &&
            (identical(other.handler, handler) || other.handler == handler) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.handledAt, handledAt) ||
                other.handledAt == handledAt) &&
            (identical(other.reportCount, reportCount) ||
                other.reportCount == reportCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    reviewId,
    reason,
    note,
    status,
    resolution,
    excerpt,
    reporter,
    handler,
    createdAt,
    handledAt,
    reportCount,
  );

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationCourseReviewReportItemImplCopyWith<
    _$ModerationCourseReviewReportItemImpl
  >
  get copyWith =>
      __$$ModerationCourseReviewReportItemImplCopyWithImpl<
        _$ModerationCourseReviewReportItemImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationCourseReviewReportItemImplToJson(this);
  }
}

abstract class _ModerationCourseReviewReportItem
    implements ModerationCourseReviewReportItem {
  const factory _ModerationCourseReviewReportItem({
    required final int id,
    required final int reviewId,
    required final String reason,
    required final String note,
    required final String status,
    required final String resolution,
    required final String excerpt,
    required final UserBriefPayload reporter,
    required final UserBriefPayload handler,
    required final String createdAt,
    final String? handledAt,
    required final int reportCount,
  }) = _$ModerationCourseReviewReportItemImpl;

  factory _ModerationCourseReviewReportItem.fromJson(
    Map<String, dynamic> json,
  ) = _$ModerationCourseReviewReportItemImpl.fromJson;

  @override
  int get id;
  @override
  int get reviewId;
  @override
  String get reason;
  @override
  String get note;
  @override
  String get status;
  @override
  String get resolution;
  @override
  String get excerpt;
  @override
  UserBriefPayload get reporter;
  @override
  UserBriefPayload get handler;
  @override
  String get createdAt;
  @override
  String? get handledAt;
  @override
  int get reportCount;

  /// Create a copy of ModerationCourseReviewReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationCourseReviewReportItemImplCopyWith<
    _$ModerationCourseReviewReportItemImpl
  >
  get copyWith => throw _privateConstructorUsedError;
}

ModerationCourseReviewReportListResponse
_$ModerationCourseReviewReportListResponseFromJson(Map<String, dynamic> json) {
  return _ModerationCourseReviewReportListResponse.fromJson(json);
}

/// @nodoc
mixin _$ModerationCourseReviewReportListResponse {
  List<ModerationCourseReviewReportItem> get items =>
      throw _privateConstructorUsedError;
  int get nextCursor => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this ModerationCourseReviewReportListResponse to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationCourseReviewReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationCourseReviewReportListResponseCopyWith<
    ModerationCourseReviewReportListResponse
  >
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationCourseReviewReportListResponseCopyWith<$Res> {
  factory $ModerationCourseReviewReportListResponseCopyWith(
    ModerationCourseReviewReportListResponse value,
    $Res Function(ModerationCourseReviewReportListResponse) then,
  ) =
      _$ModerationCourseReviewReportListResponseCopyWithImpl<
        $Res,
        ModerationCourseReviewReportListResponse
      >;
  @useResult
  $Res call({
    List<ModerationCourseReviewReportItem> items,
    int nextCursor,
    bool hasNext,
  });
}

/// @nodoc
class _$ModerationCourseReviewReportListResponseCopyWithImpl<
  $Res,
  $Val extends ModerationCourseReviewReportListResponse
>
    implements $ModerationCourseReviewReportListResponseCopyWith<$Res> {
  _$ModerationCourseReviewReportListResponseCopyWithImpl(
    this._value,
    this._then,
  );

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationCourseReviewReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
  }) {
    return _then(
      _value.copyWith(
            items: null == items
                ? _value.items
                : items // ignore: cast_nullable_to_non_nullable
                      as List<ModerationCourseReviewReportItem>,
            nextCursor: null == nextCursor
                ? _value.nextCursor
                : nextCursor // ignore: cast_nullable_to_non_nullable
                      as int,
            hasNext: null == hasNext
                ? _value.hasNext
                : hasNext // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ModerationCourseReviewReportListResponseImplCopyWith<$Res>
    implements $ModerationCourseReviewReportListResponseCopyWith<$Res> {
  factory _$$ModerationCourseReviewReportListResponseImplCopyWith(
    _$ModerationCourseReviewReportListResponseImpl value,
    $Res Function(_$ModerationCourseReviewReportListResponseImpl) then,
  ) = __$$ModerationCourseReviewReportListResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<ModerationCourseReviewReportItem> items,
    int nextCursor,
    bool hasNext,
  });
}

/// @nodoc
class __$$ModerationCourseReviewReportListResponseImplCopyWithImpl<$Res>
    extends
        _$ModerationCourseReviewReportListResponseCopyWithImpl<
          $Res,
          _$ModerationCourseReviewReportListResponseImpl
        >
    implements _$$ModerationCourseReviewReportListResponseImplCopyWith<$Res> {
  __$$ModerationCourseReviewReportListResponseImplCopyWithImpl(
    _$ModerationCourseReviewReportListResponseImpl _value,
    $Res Function(_$ModerationCourseReviewReportListResponseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationCourseReviewReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$ModerationCourseReviewReportListResponseImpl(
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<ModerationCourseReviewReportItem>,
        nextCursor: null == nextCursor
            ? _value.nextCursor
            : nextCursor // ignore: cast_nullable_to_non_nullable
                  as int,
        hasNext: null == hasNext
            ? _value.hasNext
            : hasNext // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationCourseReviewReportListResponseImpl
    implements _ModerationCourseReviewReportListResponse {
  const _$ModerationCourseReviewReportListResponseImpl({
    required final List<ModerationCourseReviewReportItem> items,
    required this.nextCursor,
    required this.hasNext,
  }) : _items = items;

  factory _$ModerationCourseReviewReportListResponseImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$ModerationCourseReviewReportListResponseImplFromJson(json);

  final List<ModerationCourseReviewReportItem> _items;
  @override
  List<ModerationCourseReviewReportItem> get items {
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_items);
  }

  @override
  final int nextCursor;
  @override
  final bool hasNext;

  @override
  String toString() {
    return 'ModerationCourseReviewReportListResponse(items: $items, nextCursor: $nextCursor, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationCourseReviewReportListResponseImpl &&
            const DeepCollectionEquality().equals(other._items, _items) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_items),
    nextCursor,
    hasNext,
  );

  /// Create a copy of ModerationCourseReviewReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationCourseReviewReportListResponseImplCopyWith<
    _$ModerationCourseReviewReportListResponseImpl
  >
  get copyWith =>
      __$$ModerationCourseReviewReportListResponseImplCopyWithImpl<
        _$ModerationCourseReviewReportListResponseImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationCourseReviewReportListResponseImplToJson(this);
  }
}

abstract class _ModerationCourseReviewReportListResponse
    implements ModerationCourseReviewReportListResponse {
  const factory _ModerationCourseReviewReportListResponse({
    required final List<ModerationCourseReviewReportItem> items,
    required final int nextCursor,
    required final bool hasNext,
  }) = _$ModerationCourseReviewReportListResponseImpl;

  factory _ModerationCourseReviewReportListResponse.fromJson(
    Map<String, dynamic> json,
  ) = _$ModerationCourseReviewReportListResponseImpl.fromJson;

  @override
  List<ModerationCourseReviewReportItem> get items;
  @override
  int get nextCursor;
  @override
  bool get hasNext;

  /// Create a copy of ModerationCourseReviewReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationCourseReviewReportListResponseImplCopyWith<
    _$ModerationCourseReviewReportListResponseImpl
  >
  get copyWith => throw _privateConstructorUsedError;
}

CourseReviewAuthorRevealPayload _$CourseReviewAuthorRevealPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CourseReviewAuthorRevealPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseReviewAuthorRevealPayload {
  int get reviewId => throw _privateConstructorUsedError;
  int? get authorUserId => throw _privateConstructorUsedError;
  String? get username => throw _privateConstructorUsedError;
  String? get nickname => throw _privateConstructorUsedError;
  bool get isAnonymous => throw _privateConstructorUsedError;
  String get source => throw _privateConstructorUsedError;

  /// Serializes this CourseReviewAuthorRevealPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseReviewAuthorRevealPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseReviewAuthorRevealPayloadCopyWith<CourseReviewAuthorRevealPayload>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseReviewAuthorRevealPayloadCopyWith<$Res> {
  factory $CourseReviewAuthorRevealPayloadCopyWith(
    CourseReviewAuthorRevealPayload value,
    $Res Function(CourseReviewAuthorRevealPayload) then,
  ) =
      _$CourseReviewAuthorRevealPayloadCopyWithImpl<
        $Res,
        CourseReviewAuthorRevealPayload
      >;
  @useResult
  $Res call({
    int reviewId,
    int? authorUserId,
    String? username,
    String? nickname,
    bool isAnonymous,
    String source,
  });
}

/// @nodoc
class _$CourseReviewAuthorRevealPayloadCopyWithImpl<
  $Res,
  $Val extends CourseReviewAuthorRevealPayload
>
    implements $CourseReviewAuthorRevealPayloadCopyWith<$Res> {
  _$CourseReviewAuthorRevealPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseReviewAuthorRevealPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? reviewId = null,
    Object? authorUserId = freezed,
    Object? username = freezed,
    Object? nickname = freezed,
    Object? isAnonymous = null,
    Object? source = null,
  }) {
    return _then(
      _value.copyWith(
            reviewId: null == reviewId
                ? _value.reviewId
                : reviewId // ignore: cast_nullable_to_non_nullable
                      as int,
            authorUserId: freezed == authorUserId
                ? _value.authorUserId
                : authorUserId // ignore: cast_nullable_to_non_nullable
                      as int?,
            username: freezed == username
                ? _value.username
                : username // ignore: cast_nullable_to_non_nullable
                      as String?,
            nickname: freezed == nickname
                ? _value.nickname
                : nickname // ignore: cast_nullable_to_non_nullable
                      as String?,
            isAnonymous: null == isAnonymous
                ? _value.isAnonymous
                : isAnonymous // ignore: cast_nullable_to_non_nullable
                      as bool,
            source: null == source
                ? _value.source
                : source // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseReviewAuthorRevealPayloadImplCopyWith<$Res>
    implements $CourseReviewAuthorRevealPayloadCopyWith<$Res> {
  factory _$$CourseReviewAuthorRevealPayloadImplCopyWith(
    _$CourseReviewAuthorRevealPayloadImpl value,
    $Res Function(_$CourseReviewAuthorRevealPayloadImpl) then,
  ) = __$$CourseReviewAuthorRevealPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int reviewId,
    int? authorUserId,
    String? username,
    String? nickname,
    bool isAnonymous,
    String source,
  });
}

/// @nodoc
class __$$CourseReviewAuthorRevealPayloadImplCopyWithImpl<$Res>
    extends
        _$CourseReviewAuthorRevealPayloadCopyWithImpl<
          $Res,
          _$CourseReviewAuthorRevealPayloadImpl
        >
    implements _$$CourseReviewAuthorRevealPayloadImplCopyWith<$Res> {
  __$$CourseReviewAuthorRevealPayloadImplCopyWithImpl(
    _$CourseReviewAuthorRevealPayloadImpl _value,
    $Res Function(_$CourseReviewAuthorRevealPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseReviewAuthorRevealPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? reviewId = null,
    Object? authorUserId = freezed,
    Object? username = freezed,
    Object? nickname = freezed,
    Object? isAnonymous = null,
    Object? source = null,
  }) {
    return _then(
      _$CourseReviewAuthorRevealPayloadImpl(
        reviewId: null == reviewId
            ? _value.reviewId
            : reviewId // ignore: cast_nullable_to_non_nullable
                  as int,
        authorUserId: freezed == authorUserId
            ? _value.authorUserId
            : authorUserId // ignore: cast_nullable_to_non_nullable
                  as int?,
        username: freezed == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String?,
        nickname: freezed == nickname
            ? _value.nickname
            : nickname // ignore: cast_nullable_to_non_nullable
                  as String?,
        isAnonymous: null == isAnonymous
            ? _value.isAnonymous
            : isAnonymous // ignore: cast_nullable_to_non_nullable
                  as bool,
        source: null == source
            ? _value.source
            : source // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseReviewAuthorRevealPayloadImpl
    implements _CourseReviewAuthorRevealPayload {
  const _$CourseReviewAuthorRevealPayloadImpl({
    required this.reviewId,
    this.authorUserId,
    this.username,
    this.nickname,
    required this.isAnonymous,
    required this.source,
  });

  factory _$CourseReviewAuthorRevealPayloadImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$CourseReviewAuthorRevealPayloadImplFromJson(json);

  @override
  final int reviewId;
  @override
  final int? authorUserId;
  @override
  final String? username;
  @override
  final String? nickname;
  @override
  final bool isAnonymous;
  @override
  final String source;

  @override
  String toString() {
    return 'CourseReviewAuthorRevealPayload(reviewId: $reviewId, authorUserId: $authorUserId, username: $username, nickname: $nickname, isAnonymous: $isAnonymous, source: $source)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseReviewAuthorRevealPayloadImpl &&
            (identical(other.reviewId, reviewId) ||
                other.reviewId == reviewId) &&
            (identical(other.authorUserId, authorUserId) ||
                other.authorUserId == authorUserId) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.isAnonymous, isAnonymous) ||
                other.isAnonymous == isAnonymous) &&
            (identical(other.source, source) || other.source == source));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    reviewId,
    authorUserId,
    username,
    nickname,
    isAnonymous,
    source,
  );

  /// Create a copy of CourseReviewAuthorRevealPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseReviewAuthorRevealPayloadImplCopyWith<
    _$CourseReviewAuthorRevealPayloadImpl
  >
  get copyWith =>
      __$$CourseReviewAuthorRevealPayloadImplCopyWithImpl<
        _$CourseReviewAuthorRevealPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseReviewAuthorRevealPayloadImplToJson(this);
  }
}

abstract class _CourseReviewAuthorRevealPayload
    implements CourseReviewAuthorRevealPayload {
  const factory _CourseReviewAuthorRevealPayload({
    required final int reviewId,
    final int? authorUserId,
    final String? username,
    final String? nickname,
    required final bool isAnonymous,
    required final String source,
  }) = _$CourseReviewAuthorRevealPayloadImpl;

  factory _CourseReviewAuthorRevealPayload.fromJson(Map<String, dynamic> json) =
      _$CourseReviewAuthorRevealPayloadImpl.fromJson;

  @override
  int get reviewId;
  @override
  int? get authorUserId;
  @override
  String? get username;
  @override
  String? get nickname;
  @override
  bool get isAnonymous;
  @override
  String get source;

  /// Create a copy of CourseReviewAuthorRevealPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseReviewAuthorRevealPayloadImplCopyWith<
    _$CourseReviewAuthorRevealPayloadImpl
  >
  get copyWith => throw _privateConstructorUsedError;
}

RelatedCourseItem _$RelatedCourseItemFromJson(Map<String, dynamic> json) {
  return _RelatedCourseItem.fromJson(json);
}

/// @nodoc
mixin _$RelatedCourseItem {
  int get id => throw _privateConstructorUsedError;
  String get primaryCode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get department => throw _privateConstructorUsedError;
  List<String>? get instructors => throw _privateConstructorUsedError;
  double get ratingAvg => throw _privateConstructorUsedError;
  int get ratingCount => throw _privateConstructorUsedError;
  int get reviewCount => throw _privateConstructorUsedError;

  /// Serializes this RelatedCourseItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of RelatedCourseItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $RelatedCourseItemCopyWith<RelatedCourseItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RelatedCourseItemCopyWith<$Res> {
  factory $RelatedCourseItemCopyWith(
    RelatedCourseItem value,
    $Res Function(RelatedCourseItem) then,
  ) = _$RelatedCourseItemCopyWithImpl<$Res, RelatedCourseItem>;
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    List<String>? instructors,
    double ratingAvg,
    int ratingCount,
    int reviewCount,
  });
}

/// @nodoc
class _$RelatedCourseItemCopyWithImpl<$Res, $Val extends RelatedCourseItem>
    implements $RelatedCourseItemCopyWith<$Res> {
  _$RelatedCourseItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of RelatedCourseItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? instructors = freezed,
    Object? ratingAvg = null,
    Object? ratingCount = null,
    Object? reviewCount = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            primaryCode: null == primaryCode
                ? _value.primaryCode
                : primaryCode // ignore: cast_nullable_to_non_nullable
                      as String,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            department: null == department
                ? _value.department
                : department // ignore: cast_nullable_to_non_nullable
                      as String,
            instructors: freezed == instructors
                ? _value.instructors
                : instructors // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            ratingAvg: null == ratingAvg
                ? _value.ratingAvg
                : ratingAvg // ignore: cast_nullable_to_non_nullable
                      as double,
            ratingCount: null == ratingCount
                ? _value.ratingCount
                : ratingCount // ignore: cast_nullable_to_non_nullable
                      as int,
            reviewCount: null == reviewCount
                ? _value.reviewCount
                : reviewCount // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$RelatedCourseItemImplCopyWith<$Res>
    implements $RelatedCourseItemCopyWith<$Res> {
  factory _$$RelatedCourseItemImplCopyWith(
    _$RelatedCourseItemImpl value,
    $Res Function(_$RelatedCourseItemImpl) then,
  ) = __$$RelatedCourseItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    List<String>? instructors,
    double ratingAvg,
    int ratingCount,
    int reviewCount,
  });
}

/// @nodoc
class __$$RelatedCourseItemImplCopyWithImpl<$Res>
    extends _$RelatedCourseItemCopyWithImpl<$Res, _$RelatedCourseItemImpl>
    implements _$$RelatedCourseItemImplCopyWith<$Res> {
  __$$RelatedCourseItemImplCopyWithImpl(
    _$RelatedCourseItemImpl _value,
    $Res Function(_$RelatedCourseItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of RelatedCourseItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? instructors = freezed,
    Object? ratingAvg = null,
    Object? ratingCount = null,
    Object? reviewCount = null,
  }) {
    return _then(
      _$RelatedCourseItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        primaryCode: null == primaryCode
            ? _value.primaryCode
            : primaryCode // ignore: cast_nullable_to_non_nullable
                  as String,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        department: null == department
            ? _value.department
            : department // ignore: cast_nullable_to_non_nullable
                  as String,
        instructors: freezed == instructors
            ? _value._instructors
            : instructors // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        ratingAvg: null == ratingAvg
            ? _value.ratingAvg
            : ratingAvg // ignore: cast_nullable_to_non_nullable
                  as double,
        ratingCount: null == ratingCount
            ? _value.ratingCount
            : ratingCount // ignore: cast_nullable_to_non_nullable
                  as int,
        reviewCount: null == reviewCount
            ? _value.reviewCount
            : reviewCount // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$RelatedCourseItemImpl implements _RelatedCourseItem {
  const _$RelatedCourseItemImpl({
    required this.id,
    required this.primaryCode,
    required this.name,
    required this.department,
    final List<String>? instructors,
    required this.ratingAvg,
    required this.ratingCount,
    required this.reviewCount,
  }) : _instructors = instructors;

  factory _$RelatedCourseItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$RelatedCourseItemImplFromJson(json);

  @override
  final int id;
  @override
  final String primaryCode;
  @override
  final String name;
  @override
  final String department;
  final List<String>? _instructors;
  @override
  List<String>? get instructors {
    final value = _instructors;
    if (value == null) return null;
    if (_instructors is EqualUnmodifiableListView) return _instructors;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final double ratingAvg;
  @override
  final int ratingCount;
  @override
  final int reviewCount;

  @override
  String toString() {
    return 'RelatedCourseItem(id: $id, primaryCode: $primaryCode, name: $name, department: $department, instructors: $instructors, ratingAvg: $ratingAvg, ratingCount: $ratingCount, reviewCount: $reviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RelatedCourseItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.primaryCode, primaryCode) ||
                other.primaryCode == primaryCode) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.department, department) ||
                other.department == department) &&
            const DeepCollectionEquality().equals(
              other._instructors,
              _instructors,
            ) &&
            (identical(other.ratingAvg, ratingAvg) ||
                other.ratingAvg == ratingAvg) &&
            (identical(other.ratingCount, ratingCount) ||
                other.ratingCount == ratingCount) &&
            (identical(other.reviewCount, reviewCount) ||
                other.reviewCount == reviewCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    primaryCode,
    name,
    department,
    const DeepCollectionEquality().hash(_instructors),
    ratingAvg,
    ratingCount,
    reviewCount,
  );

  /// Create a copy of RelatedCourseItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$RelatedCourseItemImplCopyWith<_$RelatedCourseItemImpl> get copyWith =>
      __$$RelatedCourseItemImplCopyWithImpl<_$RelatedCourseItemImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$RelatedCourseItemImplToJson(this);
  }
}

abstract class _RelatedCourseItem implements RelatedCourseItem {
  const factory _RelatedCourseItem({
    required final int id,
    required final String primaryCode,
    required final String name,
    required final String department,
    final List<String>? instructors,
    required final double ratingAvg,
    required final int ratingCount,
    required final int reviewCount,
  }) = _$RelatedCourseItemImpl;

  factory _RelatedCourseItem.fromJson(Map<String, dynamic> json) =
      _$RelatedCourseItemImpl.fromJson;

  @override
  int get id;
  @override
  String get primaryCode;
  @override
  String get name;
  @override
  String get department;
  @override
  List<String>? get instructors;
  @override
  double get ratingAvg;
  @override
  int get ratingCount;
  @override
  int get reviewCount;

  /// Create a copy of RelatedCourseItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$RelatedCourseItemImplCopyWith<_$RelatedCourseItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

RelatedTeacherOfferingItem _$RelatedTeacherOfferingItemFromJson(
  Map<String, dynamic> json,
) {
  return _RelatedTeacherOfferingItem.fromJson(json);
}

/// @nodoc
mixin _$RelatedTeacherOfferingItem {
  int get offeringId => throw _privateConstructorUsedError;
  String? get termCode => throw _privateConstructorUsedError;
  String? get termName => throw _privateConstructorUsedError;
  String? get campus => throw _privateConstructorUsedError;
  List<String>? get instructors => throw _privateConstructorUsedError;
  double get ratingAvg => throw _privateConstructorUsedError;
  int get ratingCount => throw _privateConstructorUsedError;
  int get reviewCount => throw _privateConstructorUsedError;

  /// Serializes this RelatedTeacherOfferingItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of RelatedTeacherOfferingItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $RelatedTeacherOfferingItemCopyWith<RelatedTeacherOfferingItem>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RelatedTeacherOfferingItemCopyWith<$Res> {
  factory $RelatedTeacherOfferingItemCopyWith(
    RelatedTeacherOfferingItem value,
    $Res Function(RelatedTeacherOfferingItem) then,
  ) =
      _$RelatedTeacherOfferingItemCopyWithImpl<
        $Res,
        RelatedTeacherOfferingItem
      >;
  @useResult
  $Res call({
    int offeringId,
    String? termCode,
    String? termName,
    String? campus,
    List<String>? instructors,
    double ratingAvg,
    int ratingCount,
    int reviewCount,
  });
}

/// @nodoc
class _$RelatedTeacherOfferingItemCopyWithImpl<
  $Res,
  $Val extends RelatedTeacherOfferingItem
>
    implements $RelatedTeacherOfferingItemCopyWith<$Res> {
  _$RelatedTeacherOfferingItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of RelatedTeacherOfferingItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? offeringId = null,
    Object? termCode = freezed,
    Object? termName = freezed,
    Object? campus = freezed,
    Object? instructors = freezed,
    Object? ratingAvg = null,
    Object? ratingCount = null,
    Object? reviewCount = null,
  }) {
    return _then(
      _value.copyWith(
            offeringId: null == offeringId
                ? _value.offeringId
                : offeringId // ignore: cast_nullable_to_non_nullable
                      as int,
            termCode: freezed == termCode
                ? _value.termCode
                : termCode // ignore: cast_nullable_to_non_nullable
                      as String?,
            termName: freezed == termName
                ? _value.termName
                : termName // ignore: cast_nullable_to_non_nullable
                      as String?,
            campus: freezed == campus
                ? _value.campus
                : campus // ignore: cast_nullable_to_non_nullable
                      as String?,
            instructors: freezed == instructors
                ? _value.instructors
                : instructors // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            ratingAvg: null == ratingAvg
                ? _value.ratingAvg
                : ratingAvg // ignore: cast_nullable_to_non_nullable
                      as double,
            ratingCount: null == ratingCount
                ? _value.ratingCount
                : ratingCount // ignore: cast_nullable_to_non_nullable
                      as int,
            reviewCount: null == reviewCount
                ? _value.reviewCount
                : reviewCount // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$RelatedTeacherOfferingItemImplCopyWith<$Res>
    implements $RelatedTeacherOfferingItemCopyWith<$Res> {
  factory _$$RelatedTeacherOfferingItemImplCopyWith(
    _$RelatedTeacherOfferingItemImpl value,
    $Res Function(_$RelatedTeacherOfferingItemImpl) then,
  ) = __$$RelatedTeacherOfferingItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int offeringId,
    String? termCode,
    String? termName,
    String? campus,
    List<String>? instructors,
    double ratingAvg,
    int ratingCount,
    int reviewCount,
  });
}

/// @nodoc
class __$$RelatedTeacherOfferingItemImplCopyWithImpl<$Res>
    extends
        _$RelatedTeacherOfferingItemCopyWithImpl<
          $Res,
          _$RelatedTeacherOfferingItemImpl
        >
    implements _$$RelatedTeacherOfferingItemImplCopyWith<$Res> {
  __$$RelatedTeacherOfferingItemImplCopyWithImpl(
    _$RelatedTeacherOfferingItemImpl _value,
    $Res Function(_$RelatedTeacherOfferingItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of RelatedTeacherOfferingItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? offeringId = null,
    Object? termCode = freezed,
    Object? termName = freezed,
    Object? campus = freezed,
    Object? instructors = freezed,
    Object? ratingAvg = null,
    Object? ratingCount = null,
    Object? reviewCount = null,
  }) {
    return _then(
      _$RelatedTeacherOfferingItemImpl(
        offeringId: null == offeringId
            ? _value.offeringId
            : offeringId // ignore: cast_nullable_to_non_nullable
                  as int,
        termCode: freezed == termCode
            ? _value.termCode
            : termCode // ignore: cast_nullable_to_non_nullable
                  as String?,
        termName: freezed == termName
            ? _value.termName
            : termName // ignore: cast_nullable_to_non_nullable
                  as String?,
        campus: freezed == campus
            ? _value.campus
            : campus // ignore: cast_nullable_to_non_nullable
                  as String?,
        instructors: freezed == instructors
            ? _value._instructors
            : instructors // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        ratingAvg: null == ratingAvg
            ? _value.ratingAvg
            : ratingAvg // ignore: cast_nullable_to_non_nullable
                  as double,
        ratingCount: null == ratingCount
            ? _value.ratingCount
            : ratingCount // ignore: cast_nullable_to_non_nullable
                  as int,
        reviewCount: null == reviewCount
            ? _value.reviewCount
            : reviewCount // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$RelatedTeacherOfferingItemImpl implements _RelatedTeacherOfferingItem {
  const _$RelatedTeacherOfferingItemImpl({
    required this.offeringId,
    this.termCode,
    this.termName,
    this.campus,
    final List<String>? instructors,
    required this.ratingAvg,
    required this.ratingCount,
    required this.reviewCount,
  }) : _instructors = instructors;

  factory _$RelatedTeacherOfferingItemImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$RelatedTeacherOfferingItemImplFromJson(json);

  @override
  final int offeringId;
  @override
  final String? termCode;
  @override
  final String? termName;
  @override
  final String? campus;
  final List<String>? _instructors;
  @override
  List<String>? get instructors {
    final value = _instructors;
    if (value == null) return null;
    if (_instructors is EqualUnmodifiableListView) return _instructors;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final double ratingAvg;
  @override
  final int ratingCount;
  @override
  final int reviewCount;

  @override
  String toString() {
    return 'RelatedTeacherOfferingItem(offeringId: $offeringId, termCode: $termCode, termName: $termName, campus: $campus, instructors: $instructors, ratingAvg: $ratingAvg, ratingCount: $ratingCount, reviewCount: $reviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RelatedTeacherOfferingItemImpl &&
            (identical(other.offeringId, offeringId) ||
                other.offeringId == offeringId) &&
            (identical(other.termCode, termCode) ||
                other.termCode == termCode) &&
            (identical(other.termName, termName) ||
                other.termName == termName) &&
            (identical(other.campus, campus) || other.campus == campus) &&
            const DeepCollectionEquality().equals(
              other._instructors,
              _instructors,
            ) &&
            (identical(other.ratingAvg, ratingAvg) ||
                other.ratingAvg == ratingAvg) &&
            (identical(other.ratingCount, ratingCount) ||
                other.ratingCount == ratingCount) &&
            (identical(other.reviewCount, reviewCount) ||
                other.reviewCount == reviewCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    offeringId,
    termCode,
    termName,
    campus,
    const DeepCollectionEquality().hash(_instructors),
    ratingAvg,
    ratingCount,
    reviewCount,
  );

  /// Create a copy of RelatedTeacherOfferingItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$RelatedTeacherOfferingItemImplCopyWith<_$RelatedTeacherOfferingItemImpl>
  get copyWith =>
      __$$RelatedTeacherOfferingItemImplCopyWithImpl<
        _$RelatedTeacherOfferingItemImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RelatedTeacherOfferingItemImplToJson(this);
  }
}

abstract class _RelatedTeacherOfferingItem
    implements RelatedTeacherOfferingItem {
  const factory _RelatedTeacherOfferingItem({
    required final int offeringId,
    final String? termCode,
    final String? termName,
    final String? campus,
    final List<String>? instructors,
    required final double ratingAvg,
    required final int ratingCount,
    required final int reviewCount,
  }) = _$RelatedTeacherOfferingItemImpl;

  factory _RelatedTeacherOfferingItem.fromJson(Map<String, dynamic> json) =
      _$RelatedTeacherOfferingItemImpl.fromJson;

  @override
  int get offeringId;
  @override
  String? get termCode;
  @override
  String? get termName;
  @override
  String? get campus;
  @override
  List<String>? get instructors;
  @override
  double get ratingAvg;
  @override
  int get ratingCount;
  @override
  int get reviewCount;

  /// Create a copy of RelatedTeacherOfferingItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$RelatedTeacherOfferingItemImplCopyWith<_$RelatedTeacherOfferingItemImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseRelatedResult _$CourseRelatedResultFromJson(Map<String, dynamic> json) {
  return _CourseRelatedResult.fromJson(json);
}

/// @nodoc
mixin _$CourseRelatedResult {
  List<RelatedCourseItem> get teacherOtherCourses =>
      throw _privateConstructorUsedError;
  List<RelatedTeacherOfferingItem> get sameCourseOtherTeachers =>
      throw _privateConstructorUsedError;

  /// Serializes this CourseRelatedResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseRelatedResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseRelatedResultCopyWith<CourseRelatedResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseRelatedResultCopyWith<$Res> {
  factory $CourseRelatedResultCopyWith(
    CourseRelatedResult value,
    $Res Function(CourseRelatedResult) then,
  ) = _$CourseRelatedResultCopyWithImpl<$Res, CourseRelatedResult>;
  @useResult
  $Res call({
    List<RelatedCourseItem> teacherOtherCourses,
    List<RelatedTeacherOfferingItem> sameCourseOtherTeachers,
  });
}

/// @nodoc
class _$CourseRelatedResultCopyWithImpl<$Res, $Val extends CourseRelatedResult>
    implements $CourseRelatedResultCopyWith<$Res> {
  _$CourseRelatedResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseRelatedResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? teacherOtherCourses = null,
    Object? sameCourseOtherTeachers = null,
  }) {
    return _then(
      _value.copyWith(
            teacherOtherCourses: null == teacherOtherCourses
                ? _value.teacherOtherCourses
                : teacherOtherCourses // ignore: cast_nullable_to_non_nullable
                      as List<RelatedCourseItem>,
            sameCourseOtherTeachers: null == sameCourseOtherTeachers
                ? _value.sameCourseOtherTeachers
                : sameCourseOtherTeachers // ignore: cast_nullable_to_non_nullable
                      as List<RelatedTeacherOfferingItem>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseRelatedResultImplCopyWith<$Res>
    implements $CourseRelatedResultCopyWith<$Res> {
  factory _$$CourseRelatedResultImplCopyWith(
    _$CourseRelatedResultImpl value,
    $Res Function(_$CourseRelatedResultImpl) then,
  ) = __$$CourseRelatedResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<RelatedCourseItem> teacherOtherCourses,
    List<RelatedTeacherOfferingItem> sameCourseOtherTeachers,
  });
}

/// @nodoc
class __$$CourseRelatedResultImplCopyWithImpl<$Res>
    extends _$CourseRelatedResultCopyWithImpl<$Res, _$CourseRelatedResultImpl>
    implements _$$CourseRelatedResultImplCopyWith<$Res> {
  __$$CourseRelatedResultImplCopyWithImpl(
    _$CourseRelatedResultImpl _value,
    $Res Function(_$CourseRelatedResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseRelatedResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? teacherOtherCourses = null,
    Object? sameCourseOtherTeachers = null,
  }) {
    return _then(
      _$CourseRelatedResultImpl(
        teacherOtherCourses: null == teacherOtherCourses
            ? _value._teacherOtherCourses
            : teacherOtherCourses // ignore: cast_nullable_to_non_nullable
                  as List<RelatedCourseItem>,
        sameCourseOtherTeachers: null == sameCourseOtherTeachers
            ? _value._sameCourseOtherTeachers
            : sameCourseOtherTeachers // ignore: cast_nullable_to_non_nullable
                  as List<RelatedTeacherOfferingItem>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseRelatedResultImpl implements _CourseRelatedResult {
  const _$CourseRelatedResultImpl({
    required final List<RelatedCourseItem> teacherOtherCourses,
    required final List<RelatedTeacherOfferingItem> sameCourseOtherTeachers,
  }) : _teacherOtherCourses = teacherOtherCourses,
       _sameCourseOtherTeachers = sameCourseOtherTeachers;

  factory _$CourseRelatedResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseRelatedResultImplFromJson(json);

  final List<RelatedCourseItem> _teacherOtherCourses;
  @override
  List<RelatedCourseItem> get teacherOtherCourses {
    if (_teacherOtherCourses is EqualUnmodifiableListView) {
      return _teacherOtherCourses;
    }
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_teacherOtherCourses);
  }

  final List<RelatedTeacherOfferingItem> _sameCourseOtherTeachers;
  @override
  List<RelatedTeacherOfferingItem> get sameCourseOtherTeachers {
    if (_sameCourseOtherTeachers is EqualUnmodifiableListView) {
      return _sameCourseOtherTeachers;
    }
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_sameCourseOtherTeachers);
  }

  @override
  String toString() {
    return 'CourseRelatedResult(teacherOtherCourses: $teacherOtherCourses, sameCourseOtherTeachers: $sameCourseOtherTeachers)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseRelatedResultImpl &&
            const DeepCollectionEquality().equals(
              other._teacherOtherCourses,
              _teacherOtherCourses,
            ) &&
            const DeepCollectionEquality().equals(
              other._sameCourseOtherTeachers,
              _sameCourseOtherTeachers,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_teacherOtherCourses),
    const DeepCollectionEquality().hash(_sameCourseOtherTeachers),
  );

  /// Create a copy of CourseRelatedResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseRelatedResultImplCopyWith<_$CourseRelatedResultImpl> get copyWith =>
      __$$CourseRelatedResultImplCopyWithImpl<_$CourseRelatedResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseRelatedResultImplToJson(this);
  }
}

abstract class _CourseRelatedResult implements CourseRelatedResult {
  const factory _CourseRelatedResult({
    required final List<RelatedCourseItem> teacherOtherCourses,
    required final List<RelatedTeacherOfferingItem> sameCourseOtherTeachers,
  }) = _$CourseRelatedResultImpl;

  factory _CourseRelatedResult.fromJson(Map<String, dynamic> json) =
      _$CourseRelatedResultImpl.fromJson;

  @override
  List<RelatedCourseItem> get teacherOtherCourses;
  @override
  List<RelatedTeacherOfferingItem> get sameCourseOtherTeachers;

  /// Create a copy of CourseRelatedResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseRelatedResultImplCopyWith<_$CourseRelatedResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
