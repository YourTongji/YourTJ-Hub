// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'page.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

PagePayload _$PagePayloadFromJson(Map<String, dynamic> json) {
  return _PagePayload.fromJson(json);
}

/// @nodoc
mixin _$PagePayload {
  String get component => throw _privateConstructorUsedError;
  Map<String, dynamic> get props => throw _privateConstructorUsedError;
  PageMeta get meta => throw _privateConstructorUsedError;
  LayoutPayload get layout => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get version => throw _privateConstructorUsedError;

  /// Serializes this PagePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PagePayloadCopyWith<PagePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PagePayloadCopyWith<$Res> {
  factory $PagePayloadCopyWith(
    PagePayload value,
    $Res Function(PagePayload) then,
  ) = _$PagePayloadCopyWithImpl<$Res, PagePayload>;
  @useResult
  $Res call({
    String component,
    Map<String, dynamic> props,
    PageMeta meta,
    LayoutPayload layout,
    String url,
    String version,
  });

  $PageMetaCopyWith<$Res> get meta;
  $LayoutPayloadCopyWith<$Res> get layout;
}

/// @nodoc
class _$PagePayloadCopyWithImpl<$Res, $Val extends PagePayload>
    implements $PagePayloadCopyWith<$Res> {
  _$PagePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? component = null,
    Object? props = null,
    Object? meta = null,
    Object? layout = null,
    Object? url = null,
    Object? version = null,
  }) {
    return _then(
      _value.copyWith(
            component: null == component
                ? _value.component
                : component // ignore: cast_nullable_to_non_nullable
                      as String,
            props: null == props
                ? _value.props
                : props // ignore: cast_nullable_to_non_nullable
                      as Map<String, dynamic>,
            meta: null == meta
                ? _value.meta
                : meta // ignore: cast_nullable_to_non_nullable
                      as PageMeta,
            layout: null == layout
                ? _value.layout
                : layout // ignore: cast_nullable_to_non_nullable
                      as LayoutPayload,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            version: null == version
                ? _value.version
                : version // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PageMetaCopyWith<$Res> get meta {
    return $PageMetaCopyWith<$Res>(_value.meta, (value) {
      return _then(_value.copyWith(meta: value) as $Val);
    });
  }

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $LayoutPayloadCopyWith<$Res> get layout {
    return $LayoutPayloadCopyWith<$Res>(_value.layout, (value) {
      return _then(_value.copyWith(layout: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$PagePayloadImplCopyWith<$Res>
    implements $PagePayloadCopyWith<$Res> {
  factory _$$PagePayloadImplCopyWith(
    _$PagePayloadImpl value,
    $Res Function(_$PagePayloadImpl) then,
  ) = __$$PagePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String component,
    Map<String, dynamic> props,
    PageMeta meta,
    LayoutPayload layout,
    String url,
    String version,
  });

  @override
  $PageMetaCopyWith<$Res> get meta;
  @override
  $LayoutPayloadCopyWith<$Res> get layout;
}

/// @nodoc
class __$$PagePayloadImplCopyWithImpl<$Res>
    extends _$PagePayloadCopyWithImpl<$Res, _$PagePayloadImpl>
    implements _$$PagePayloadImplCopyWith<$Res> {
  __$$PagePayloadImplCopyWithImpl(
    _$PagePayloadImpl _value,
    $Res Function(_$PagePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? component = null,
    Object? props = null,
    Object? meta = null,
    Object? layout = null,
    Object? url = null,
    Object? version = null,
  }) {
    return _then(
      _$PagePayloadImpl(
        component: null == component
            ? _value.component
            : component // ignore: cast_nullable_to_non_nullable
                  as String,
        props: null == props
            ? _value._props
            : props // ignore: cast_nullable_to_non_nullable
                  as Map<String, dynamic>,
        meta: null == meta
            ? _value.meta
            : meta // ignore: cast_nullable_to_non_nullable
                  as PageMeta,
        layout: null == layout
            ? _value.layout
            : layout // ignore: cast_nullable_to_non_nullable
                  as LayoutPayload,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        version: null == version
            ? _value.version
            : version // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PagePayloadImpl implements _PagePayload {
  const _$PagePayloadImpl({
    required this.component,
    required final Map<String, dynamic> props,
    required this.meta,
    required this.layout,
    required this.url,
    required this.version,
  }) : _props = props;

  factory _$PagePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PagePayloadImplFromJson(json);

  @override
  final String component;
  final Map<String, dynamic> _props;
  @override
  Map<String, dynamic> get props {
    if (_props is EqualUnmodifiableMapView) return _props;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(_props);
  }

  @override
  final PageMeta meta;
  @override
  final LayoutPayload layout;
  @override
  final String url;
  @override
  final String version;

  @override
  String toString() {
    return 'PagePayload(component: $component, props: $props, meta: $meta, layout: $layout, url: $url, version: $version)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PagePayloadImpl &&
            (identical(other.component, component) ||
                other.component == component) &&
            const DeepCollectionEquality().equals(other._props, _props) &&
            (identical(other.meta, meta) || other.meta == meta) &&
            (identical(other.layout, layout) || other.layout == layout) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.version, version) || other.version == version));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    component,
    const DeepCollectionEquality().hash(_props),
    meta,
    layout,
    url,
    version,
  );

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PagePayloadImplCopyWith<_$PagePayloadImpl> get copyWith =>
      __$$PagePayloadImplCopyWithImpl<_$PagePayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PagePayloadImplToJson(this);
  }
}

abstract class _PagePayload implements PagePayload {
  const factory _PagePayload({
    required final String component,
    required final Map<String, dynamic> props,
    required final PageMeta meta,
    required final LayoutPayload layout,
    required final String url,
    required final String version,
  }) = _$PagePayloadImpl;

  factory _PagePayload.fromJson(Map<String, dynamic> json) =
      _$PagePayloadImpl.fromJson;

  @override
  String get component;
  @override
  Map<String, dynamic> get props;
  @override
  PageMeta get meta;
  @override
  LayoutPayload get layout;
  @override
  String get url;
  @override
  String get version;

  /// Create a copy of PagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PagePayloadImplCopyWith<_$PagePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
