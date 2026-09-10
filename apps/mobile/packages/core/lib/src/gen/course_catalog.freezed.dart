// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'course_catalog.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

CourseListResultPayload _$CourseListResultPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CourseListResultPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseListResultPayload {
  List<CourseSummaryPayload> get list => throw _privateConstructorUsedError;
  int get page => throw _privateConstructorUsedError;
  int get size => throw _privateConstructorUsedError;
  int get total => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this CourseListResultPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseListResultPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseListResultPayloadCopyWith<CourseListResultPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseListResultPayloadCopyWith<$Res> {
  factory $CourseListResultPayloadCopyWith(
    CourseListResultPayload value,
    $Res Function(CourseListResultPayload) then,
  ) = _$CourseListResultPayloadCopyWithImpl<$Res, CourseListResultPayload>;
  @useResult
  $Res call({
    List<CourseSummaryPayload> list,
    int page,
    int size,
    int total,
    bool hasNext,
  });
}

/// @nodoc
class _$CourseListResultPayloadCopyWithImpl<
  $Res,
  $Val extends CourseListResultPayload
>
    implements $CourseListResultPayloadCopyWith<$Res> {
  _$CourseListResultPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseListResultPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? size = null,
    Object? total = null,
    Object? hasNext = null,
  }) {
    return _then(
      _value.copyWith(
            list: null == list
                ? _value.list
                : list // ignore: cast_nullable_to_non_nullable
                      as List<CourseSummaryPayload>,
            page: null == page
                ? _value.page
                : page // ignore: cast_nullable_to_non_nullable
                      as int,
            size: null == size
                ? _value.size
                : size // ignore: cast_nullable_to_non_nullable
                      as int,
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
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
abstract class _$$CourseListResultPayloadImplCopyWith<$Res>
    implements $CourseListResultPayloadCopyWith<$Res> {
  factory _$$CourseListResultPayloadImplCopyWith(
    _$CourseListResultPayloadImpl value,
    $Res Function(_$CourseListResultPayloadImpl) then,
  ) = __$$CourseListResultPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<CourseSummaryPayload> list,
    int page,
    int size,
    int total,
    bool hasNext,
  });
}

/// @nodoc
class __$$CourseListResultPayloadImplCopyWithImpl<$Res>
    extends
        _$CourseListResultPayloadCopyWithImpl<
          $Res,
          _$CourseListResultPayloadImpl
        >
    implements _$$CourseListResultPayloadImplCopyWith<$Res> {
  __$$CourseListResultPayloadImplCopyWithImpl(
    _$CourseListResultPayloadImpl _value,
    $Res Function(_$CourseListResultPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseListResultPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? size = null,
    Object? total = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$CourseListResultPayloadImpl(
        list: null == list
            ? _value._list
            : list // ignore: cast_nullable_to_non_nullable
                  as List<CourseSummaryPayload>,
        page: null == page
            ? _value.page
            : page // ignore: cast_nullable_to_non_nullable
                  as int,
        size: null == size
            ? _value.size
            : size // ignore: cast_nullable_to_non_nullable
                  as int,
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
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
class _$CourseListResultPayloadImpl implements _CourseListResultPayload {
  const _$CourseListResultPayloadImpl({
    required final List<CourseSummaryPayload> list,
    required this.page,
    required this.size,
    required this.total,
    required this.hasNext,
  }) : _list = list;

  factory _$CourseListResultPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseListResultPayloadImplFromJson(json);

  final List<CourseSummaryPayload> _list;
  @override
  List<CourseSummaryPayload> get list {
    if (_list is EqualUnmodifiableListView) return _list;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_list);
  }

  @override
  final int page;
  @override
  final int size;
  @override
  final int total;
  @override
  final bool hasNext;

  @override
  String toString() {
    return 'CourseListResultPayload(list: $list, page: $page, size: $size, total: $total, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseListResultPayloadImpl &&
            const DeepCollectionEquality().equals(other._list, _list) &&
            (identical(other.page, page) || other.page == page) &&
            (identical(other.size, size) || other.size == size) &&
            (identical(other.total, total) || other.total == total) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_list),
    page,
    size,
    total,
    hasNext,
  );

  /// Create a copy of CourseListResultPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseListResultPayloadImplCopyWith<_$CourseListResultPayloadImpl>
  get copyWith =>
      __$$CourseListResultPayloadImplCopyWithImpl<
        _$CourseListResultPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseListResultPayloadImplToJson(this);
  }
}

abstract class _CourseListResultPayload implements CourseListResultPayload {
  const factory _CourseListResultPayload({
    required final List<CourseSummaryPayload> list,
    required final int page,
    required final int size,
    required final int total,
    required final bool hasNext,
  }) = _$CourseListResultPayloadImpl;

  factory _CourseListResultPayload.fromJson(Map<String, dynamic> json) =
      _$CourseListResultPayloadImpl.fromJson;

  @override
  List<CourseSummaryPayload> get list;
  @override
  int get page;
  @override
  int get size;
  @override
  int get total;
  @override
  bool get hasNext;

  /// Create a copy of CourseListResultPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseListResultPayloadImplCopyWith<_$CourseListResultPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}
