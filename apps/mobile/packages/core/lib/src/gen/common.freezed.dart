// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'common.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

PageMeta _$PageMetaFromJson(Map<String, dynamic> json) {
  return _PageMeta.fromJson(json);
}

/// @nodoc
mixin _$PageMeta {
  String get title => throw _privateConstructorUsedError;
  String? get description => throw _privateConstructorUsedError;
  String? get canonical => throw _privateConstructorUsedError;
  String? get prevUrl => throw _privateConstructorUsedError;
  String? get nextUrl => throw _privateConstructorUsedError;
  String? get robots => throw _privateConstructorUsedError;
  OpenGraphMeta? get openGraph => throw _privateConstructorUsedError;
  TwitterMeta? get twitter => throw _privateConstructorUsedError;
  Map<String, dynamic>? get jsonLd => throw _privateConstructorUsedError;

  /// Serializes this PageMeta to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PageMetaCopyWith<PageMeta> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PageMetaCopyWith<$Res> {
  factory $PageMetaCopyWith(PageMeta value, $Res Function(PageMeta) then) =
      _$PageMetaCopyWithImpl<$Res, PageMeta>;
  @useResult
  $Res call({
    String title,
    String? description,
    String? canonical,
    String? prevUrl,
    String? nextUrl,
    String? robots,
    OpenGraphMeta? openGraph,
    TwitterMeta? twitter,
    Map<String, dynamic>? jsonLd,
  });

  $OpenGraphMetaCopyWith<$Res>? get openGraph;
  $TwitterMetaCopyWith<$Res>? get twitter;
}

/// @nodoc
class _$PageMetaCopyWithImpl<$Res, $Val extends PageMeta>
    implements $PageMetaCopyWith<$Res> {
  _$PageMetaCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? description = freezed,
    Object? canonical = freezed,
    Object? prevUrl = freezed,
    Object? nextUrl = freezed,
    Object? robots = freezed,
    Object? openGraph = freezed,
    Object? twitter = freezed,
    Object? jsonLd = freezed,
  }) {
    return _then(
      _value.copyWith(
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            description: freezed == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String?,
            canonical: freezed == canonical
                ? _value.canonical
                : canonical // ignore: cast_nullable_to_non_nullable
                      as String?,
            prevUrl: freezed == prevUrl
                ? _value.prevUrl
                : prevUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
            nextUrl: freezed == nextUrl
                ? _value.nextUrl
                : nextUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
            robots: freezed == robots
                ? _value.robots
                : robots // ignore: cast_nullable_to_non_nullable
                      as String?,
            openGraph: freezed == openGraph
                ? _value.openGraph
                : openGraph // ignore: cast_nullable_to_non_nullable
                      as OpenGraphMeta?,
            twitter: freezed == twitter
                ? _value.twitter
                : twitter // ignore: cast_nullable_to_non_nullable
                      as TwitterMeta?,
            jsonLd: freezed == jsonLd
                ? _value.jsonLd
                : jsonLd // ignore: cast_nullable_to_non_nullable
                      as Map<String, dynamic>?,
          )
          as $Val,
    );
  }

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $OpenGraphMetaCopyWith<$Res>? get openGraph {
    if (_value.openGraph == null) {
      return null;
    }

    return $OpenGraphMetaCopyWith<$Res>(_value.openGraph!, (value) {
      return _then(_value.copyWith(openGraph: value) as $Val);
    });
  }

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $TwitterMetaCopyWith<$Res>? get twitter {
    if (_value.twitter == null) {
      return null;
    }

    return $TwitterMetaCopyWith<$Res>(_value.twitter!, (value) {
      return _then(_value.copyWith(twitter: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$PageMetaImplCopyWith<$Res>
    implements $PageMetaCopyWith<$Res> {
  factory _$$PageMetaImplCopyWith(
    _$PageMetaImpl value,
    $Res Function(_$PageMetaImpl) then,
  ) = __$$PageMetaImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String title,
    String? description,
    String? canonical,
    String? prevUrl,
    String? nextUrl,
    String? robots,
    OpenGraphMeta? openGraph,
    TwitterMeta? twitter,
    Map<String, dynamic>? jsonLd,
  });

  @override
  $OpenGraphMetaCopyWith<$Res>? get openGraph;
  @override
  $TwitterMetaCopyWith<$Res>? get twitter;
}

/// @nodoc
class __$$PageMetaImplCopyWithImpl<$Res>
    extends _$PageMetaCopyWithImpl<$Res, _$PageMetaImpl>
    implements _$$PageMetaImplCopyWith<$Res> {
  __$$PageMetaImplCopyWithImpl(
    _$PageMetaImpl _value,
    $Res Function(_$PageMetaImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? description = freezed,
    Object? canonical = freezed,
    Object? prevUrl = freezed,
    Object? nextUrl = freezed,
    Object? robots = freezed,
    Object? openGraph = freezed,
    Object? twitter = freezed,
    Object? jsonLd = freezed,
  }) {
    return _then(
      _$PageMetaImpl(
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        description: freezed == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String?,
        canonical: freezed == canonical
            ? _value.canonical
            : canonical // ignore: cast_nullable_to_non_nullable
                  as String?,
        prevUrl: freezed == prevUrl
            ? _value.prevUrl
            : prevUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
        nextUrl: freezed == nextUrl
            ? _value.nextUrl
            : nextUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
        robots: freezed == robots
            ? _value.robots
            : robots // ignore: cast_nullable_to_non_nullable
                  as String?,
        openGraph: freezed == openGraph
            ? _value.openGraph
            : openGraph // ignore: cast_nullable_to_non_nullable
                  as OpenGraphMeta?,
        twitter: freezed == twitter
            ? _value.twitter
            : twitter // ignore: cast_nullable_to_non_nullable
                  as TwitterMeta?,
        jsonLd: freezed == jsonLd
            ? _value._jsonLd
            : jsonLd // ignore: cast_nullable_to_non_nullable
                  as Map<String, dynamic>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PageMetaImpl implements _PageMeta {
  const _$PageMetaImpl({
    required this.title,
    this.description,
    this.canonical,
    this.prevUrl,
    this.nextUrl,
    this.robots,
    this.openGraph,
    this.twitter,
    final Map<String, dynamic>? jsonLd,
  }) : _jsonLd = jsonLd;

  factory _$PageMetaImpl.fromJson(Map<String, dynamic> json) =>
      _$$PageMetaImplFromJson(json);

  @override
  final String title;
  @override
  final String? description;
  @override
  final String? canonical;
  @override
  final String? prevUrl;
  @override
  final String? nextUrl;
  @override
  final String? robots;
  @override
  final OpenGraphMeta? openGraph;
  @override
  final TwitterMeta? twitter;
  final Map<String, dynamic>? _jsonLd;
  @override
  Map<String, dynamic>? get jsonLd {
    final value = _jsonLd;
    if (value == null) return null;
    if (_jsonLd is EqualUnmodifiableMapView) return _jsonLd;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(value);
  }

  @override
  String toString() {
    return 'PageMeta(title: $title, description: $description, canonical: $canonical, prevUrl: $prevUrl, nextUrl: $nextUrl, robots: $robots, openGraph: $openGraph, twitter: $twitter, jsonLd: $jsonLd)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PageMetaImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.canonical, canonical) ||
                other.canonical == canonical) &&
            (identical(other.prevUrl, prevUrl) || other.prevUrl == prevUrl) &&
            (identical(other.nextUrl, nextUrl) || other.nextUrl == nextUrl) &&
            (identical(other.robots, robots) || other.robots == robots) &&
            (identical(other.openGraph, openGraph) ||
                other.openGraph == openGraph) &&
            (identical(other.twitter, twitter) || other.twitter == twitter) &&
            const DeepCollectionEquality().equals(other._jsonLd, _jsonLd));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    title,
    description,
    canonical,
    prevUrl,
    nextUrl,
    robots,
    openGraph,
    twitter,
    const DeepCollectionEquality().hash(_jsonLd),
  );

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PageMetaImplCopyWith<_$PageMetaImpl> get copyWith =>
      __$$PageMetaImplCopyWithImpl<_$PageMetaImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PageMetaImplToJson(this);
  }
}

abstract class _PageMeta implements PageMeta {
  const factory _PageMeta({
    required final String title,
    final String? description,
    final String? canonical,
    final String? prevUrl,
    final String? nextUrl,
    final String? robots,
    final OpenGraphMeta? openGraph,
    final TwitterMeta? twitter,
    final Map<String, dynamic>? jsonLd,
  }) = _$PageMetaImpl;

  factory _PageMeta.fromJson(Map<String, dynamic> json) =
      _$PageMetaImpl.fromJson;

  @override
  String get title;
  @override
  String? get description;
  @override
  String? get canonical;
  @override
  String? get prevUrl;
  @override
  String? get nextUrl;
  @override
  String? get robots;
  @override
  OpenGraphMeta? get openGraph;
  @override
  TwitterMeta? get twitter;
  @override
  Map<String, dynamic>? get jsonLd;

  /// Create a copy of PageMeta
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PageMetaImplCopyWith<_$PageMetaImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OpenGraphMeta _$OpenGraphMetaFromJson(Map<String, dynamic> json) {
  return _OpenGraphMeta.fromJson(json);
}

/// @nodoc
mixin _$OpenGraphMeta {
  String? get title => throw _privateConstructorUsedError;
  String? get description => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  String? get url => throw _privateConstructorUsedError;
  String? get siteName => throw _privateConstructorUsedError;
  String? get image => throw _privateConstructorUsedError;
  String? get publishedTime => throw _privateConstructorUsedError;
  String? get modifiedTime => throw _privateConstructorUsedError;
  String? get author => throw _privateConstructorUsedError;
  String? get section => throw _privateConstructorUsedError;
  List<String>? get tags => throw _privateConstructorUsedError;

  /// Serializes this OpenGraphMeta to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of OpenGraphMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $OpenGraphMetaCopyWith<OpenGraphMeta> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OpenGraphMetaCopyWith<$Res> {
  factory $OpenGraphMetaCopyWith(
    OpenGraphMeta value,
    $Res Function(OpenGraphMeta) then,
  ) = _$OpenGraphMetaCopyWithImpl<$Res, OpenGraphMeta>;
  @useResult
  $Res call({
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
  });
}

/// @nodoc
class _$OpenGraphMetaCopyWithImpl<$Res, $Val extends OpenGraphMeta>
    implements $OpenGraphMetaCopyWith<$Res> {
  _$OpenGraphMetaCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of OpenGraphMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = freezed,
    Object? description = freezed,
    Object? type = freezed,
    Object? url = freezed,
    Object? siteName = freezed,
    Object? image = freezed,
    Object? publishedTime = freezed,
    Object? modifiedTime = freezed,
    Object? author = freezed,
    Object? section = freezed,
    Object? tags = freezed,
  }) {
    return _then(
      _value.copyWith(
            title: freezed == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String?,
            description: freezed == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String?,
            type: freezed == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String?,
            url: freezed == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String?,
            siteName: freezed == siteName
                ? _value.siteName
                : siteName // ignore: cast_nullable_to_non_nullable
                      as String?,
            image: freezed == image
                ? _value.image
                : image // ignore: cast_nullable_to_non_nullable
                      as String?,
            publishedTime: freezed == publishedTime
                ? _value.publishedTime
                : publishedTime // ignore: cast_nullable_to_non_nullable
                      as String?,
            modifiedTime: freezed == modifiedTime
                ? _value.modifiedTime
                : modifiedTime // ignore: cast_nullable_to_non_nullable
                      as String?,
            author: freezed == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as String?,
            section: freezed == section
                ? _value.section
                : section // ignore: cast_nullable_to_non_nullable
                      as String?,
            tags: freezed == tags
                ? _value.tags
                : tags // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$OpenGraphMetaImplCopyWith<$Res>
    implements $OpenGraphMetaCopyWith<$Res> {
  factory _$$OpenGraphMetaImplCopyWith(
    _$OpenGraphMetaImpl value,
    $Res Function(_$OpenGraphMetaImpl) then,
  ) = __$$OpenGraphMetaImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
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
  });
}

/// @nodoc
class __$$OpenGraphMetaImplCopyWithImpl<$Res>
    extends _$OpenGraphMetaCopyWithImpl<$Res, _$OpenGraphMetaImpl>
    implements _$$OpenGraphMetaImplCopyWith<$Res> {
  __$$OpenGraphMetaImplCopyWithImpl(
    _$OpenGraphMetaImpl _value,
    $Res Function(_$OpenGraphMetaImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of OpenGraphMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = freezed,
    Object? description = freezed,
    Object? type = freezed,
    Object? url = freezed,
    Object? siteName = freezed,
    Object? image = freezed,
    Object? publishedTime = freezed,
    Object? modifiedTime = freezed,
    Object? author = freezed,
    Object? section = freezed,
    Object? tags = freezed,
  }) {
    return _then(
      _$OpenGraphMetaImpl(
        title: freezed == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String?,
        description: freezed == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String?,
        type: freezed == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String?,
        url: freezed == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String?,
        siteName: freezed == siteName
            ? _value.siteName
            : siteName // ignore: cast_nullable_to_non_nullable
                  as String?,
        image: freezed == image
            ? _value.image
            : image // ignore: cast_nullable_to_non_nullable
                  as String?,
        publishedTime: freezed == publishedTime
            ? _value.publishedTime
            : publishedTime // ignore: cast_nullable_to_non_nullable
                  as String?,
        modifiedTime: freezed == modifiedTime
            ? _value.modifiedTime
            : modifiedTime // ignore: cast_nullable_to_non_nullable
                  as String?,
        author: freezed == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as String?,
        section: freezed == section
            ? _value.section
            : section // ignore: cast_nullable_to_non_nullable
                  as String?,
        tags: freezed == tags
            ? _value._tags
            : tags // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$OpenGraphMetaImpl implements _OpenGraphMeta {
  const _$OpenGraphMetaImpl({
    this.title,
    this.description,
    this.type,
    this.url,
    this.siteName,
    this.image,
    this.publishedTime,
    this.modifiedTime,
    this.author,
    this.section,
    final List<String>? tags,
  }) : _tags = tags;

  factory _$OpenGraphMetaImpl.fromJson(Map<String, dynamic> json) =>
      _$$OpenGraphMetaImplFromJson(json);

  @override
  final String? title;
  @override
  final String? description;
  @override
  final String? type;
  @override
  final String? url;
  @override
  final String? siteName;
  @override
  final String? image;
  @override
  final String? publishedTime;
  @override
  final String? modifiedTime;
  @override
  final String? author;
  @override
  final String? section;
  final List<String>? _tags;
  @override
  List<String>? get tags {
    final value = _tags;
    if (value == null) return null;
    if (_tags is EqualUnmodifiableListView) return _tags;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'OpenGraphMeta(title: $title, description: $description, type: $type, url: $url, siteName: $siteName, image: $image, publishedTime: $publishedTime, modifiedTime: $modifiedTime, author: $author, section: $section, tags: $tags)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OpenGraphMetaImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.siteName, siteName) ||
                other.siteName == siteName) &&
            (identical(other.image, image) || other.image == image) &&
            (identical(other.publishedTime, publishedTime) ||
                other.publishedTime == publishedTime) &&
            (identical(other.modifiedTime, modifiedTime) ||
                other.modifiedTime == modifiedTime) &&
            (identical(other.author, author) || other.author == author) &&
            (identical(other.section, section) || other.section == section) &&
            const DeepCollectionEquality().equals(other._tags, _tags));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    title,
    description,
    type,
    url,
    siteName,
    image,
    publishedTime,
    modifiedTime,
    author,
    section,
    const DeepCollectionEquality().hash(_tags),
  );

  /// Create a copy of OpenGraphMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$OpenGraphMetaImplCopyWith<_$OpenGraphMetaImpl> get copyWith =>
      __$$OpenGraphMetaImplCopyWithImpl<_$OpenGraphMetaImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$OpenGraphMetaImplToJson(this);
  }
}

abstract class _OpenGraphMeta implements OpenGraphMeta {
  const factory _OpenGraphMeta({
    final String? title,
    final String? description,
    final String? type,
    final String? url,
    final String? siteName,
    final String? image,
    final String? publishedTime,
    final String? modifiedTime,
    final String? author,
    final String? section,
    final List<String>? tags,
  }) = _$OpenGraphMetaImpl;

  factory _OpenGraphMeta.fromJson(Map<String, dynamic> json) =
      _$OpenGraphMetaImpl.fromJson;

  @override
  String? get title;
  @override
  String? get description;
  @override
  String? get type;
  @override
  String? get url;
  @override
  String? get siteName;
  @override
  String? get image;
  @override
  String? get publishedTime;
  @override
  String? get modifiedTime;
  @override
  String? get author;
  @override
  String? get section;
  @override
  List<String>? get tags;

  /// Create a copy of OpenGraphMeta
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$OpenGraphMetaImplCopyWith<_$OpenGraphMetaImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TwitterMeta _$TwitterMetaFromJson(Map<String, dynamic> json) {
  return _TwitterMeta.fromJson(json);
}

/// @nodoc
mixin _$TwitterMeta {
  String? get card => throw _privateConstructorUsedError;
  String? get title => throw _privateConstructorUsedError;
  String? get description => throw _privateConstructorUsedError;
  String? get image => throw _privateConstructorUsedError;

  /// Serializes this TwitterMeta to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TwitterMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TwitterMetaCopyWith<TwitterMeta> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TwitterMetaCopyWith<$Res> {
  factory $TwitterMetaCopyWith(
    TwitterMeta value,
    $Res Function(TwitterMeta) then,
  ) = _$TwitterMetaCopyWithImpl<$Res, TwitterMeta>;
  @useResult
  $Res call({String? card, String? title, String? description, String? image});
}

/// @nodoc
class _$TwitterMetaCopyWithImpl<$Res, $Val extends TwitterMeta>
    implements $TwitterMetaCopyWith<$Res> {
  _$TwitterMetaCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TwitterMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? card = freezed,
    Object? title = freezed,
    Object? description = freezed,
    Object? image = freezed,
  }) {
    return _then(
      _value.copyWith(
            card: freezed == card
                ? _value.card
                : card // ignore: cast_nullable_to_non_nullable
                      as String?,
            title: freezed == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String?,
            description: freezed == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String?,
            image: freezed == image
                ? _value.image
                : image // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TwitterMetaImplCopyWith<$Res>
    implements $TwitterMetaCopyWith<$Res> {
  factory _$$TwitterMetaImplCopyWith(
    _$TwitterMetaImpl value,
    $Res Function(_$TwitterMetaImpl) then,
  ) = __$$TwitterMetaImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String? card, String? title, String? description, String? image});
}

/// @nodoc
class __$$TwitterMetaImplCopyWithImpl<$Res>
    extends _$TwitterMetaCopyWithImpl<$Res, _$TwitterMetaImpl>
    implements _$$TwitterMetaImplCopyWith<$Res> {
  __$$TwitterMetaImplCopyWithImpl(
    _$TwitterMetaImpl _value,
    $Res Function(_$TwitterMetaImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TwitterMeta
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? card = freezed,
    Object? title = freezed,
    Object? description = freezed,
    Object? image = freezed,
  }) {
    return _then(
      _$TwitterMetaImpl(
        card: freezed == card
            ? _value.card
            : card // ignore: cast_nullable_to_non_nullable
                  as String?,
        title: freezed == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String?,
        description: freezed == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String?,
        image: freezed == image
            ? _value.image
            : image // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TwitterMetaImpl implements _TwitterMeta {
  const _$TwitterMetaImpl({
    this.card,
    this.title,
    this.description,
    this.image,
  });

  factory _$TwitterMetaImpl.fromJson(Map<String, dynamic> json) =>
      _$$TwitterMetaImplFromJson(json);

  @override
  final String? card;
  @override
  final String? title;
  @override
  final String? description;
  @override
  final String? image;

  @override
  String toString() {
    return 'TwitterMeta(card: $card, title: $title, description: $description, image: $image)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TwitterMetaImpl &&
            (identical(other.card, card) || other.card == card) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.image, image) || other.image == image));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, card, title, description, image);

  /// Create a copy of TwitterMeta
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TwitterMetaImplCopyWith<_$TwitterMetaImpl> get copyWith =>
      __$$TwitterMetaImplCopyWithImpl<_$TwitterMetaImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TwitterMetaImplToJson(this);
  }
}

abstract class _TwitterMeta implements TwitterMeta {
  const factory _TwitterMeta({
    final String? card,
    final String? title,
    final String? description,
    final String? image,
  }) = _$TwitterMetaImpl;

  factory _TwitterMeta.fromJson(Map<String, dynamic> json) =
      _$TwitterMetaImpl.fromJson;

  @override
  String? get card;
  @override
  String? get title;
  @override
  String? get description;
  @override
  String? get image;

  /// Create a copy of TwitterMeta
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TwitterMetaImplCopyWith<_$TwitterMetaImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PaginationPayload _$PaginationPayloadFromJson(Map<String, dynamic> json) {
  return _PaginationPayload.fromJson(json);
}

/// @nodoc
mixin _$PaginationPayload {
  int get page => throw _privateConstructorUsedError;
  int get nextPage => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;
  String get nextUrl => throw _privateConstructorUsedError;

  /// Serializes this PaginationPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PaginationPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PaginationPayloadCopyWith<PaginationPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PaginationPayloadCopyWith<$Res> {
  factory $PaginationPayloadCopyWith(
    PaginationPayload value,
    $Res Function(PaginationPayload) then,
  ) = _$PaginationPayloadCopyWithImpl<$Res, PaginationPayload>;
  @useResult
  $Res call({int page, int nextPage, bool hasNext, String nextUrl});
}

/// @nodoc
class _$PaginationPayloadCopyWithImpl<$Res, $Val extends PaginationPayload>
    implements $PaginationPayloadCopyWith<$Res> {
  _$PaginationPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PaginationPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? page = null,
    Object? nextPage = null,
    Object? hasNext = null,
    Object? nextUrl = null,
  }) {
    return _then(
      _value.copyWith(
            page: null == page
                ? _value.page
                : page // ignore: cast_nullable_to_non_nullable
                      as int,
            nextPage: null == nextPage
                ? _value.nextPage
                : nextPage // ignore: cast_nullable_to_non_nullable
                      as int,
            hasNext: null == hasNext
                ? _value.hasNext
                : hasNext // ignore: cast_nullable_to_non_nullable
                      as bool,
            nextUrl: null == nextUrl
                ? _value.nextUrl
                : nextUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$PaginationPayloadImplCopyWith<$Res>
    implements $PaginationPayloadCopyWith<$Res> {
  factory _$$PaginationPayloadImplCopyWith(
    _$PaginationPayloadImpl value,
    $Res Function(_$PaginationPayloadImpl) then,
  ) = __$$PaginationPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int page, int nextPage, bool hasNext, String nextUrl});
}

/// @nodoc
class __$$PaginationPayloadImplCopyWithImpl<$Res>
    extends _$PaginationPayloadCopyWithImpl<$Res, _$PaginationPayloadImpl>
    implements _$$PaginationPayloadImplCopyWith<$Res> {
  __$$PaginationPayloadImplCopyWithImpl(
    _$PaginationPayloadImpl _value,
    $Res Function(_$PaginationPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PaginationPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? page = null,
    Object? nextPage = null,
    Object? hasNext = null,
    Object? nextUrl = null,
  }) {
    return _then(
      _$PaginationPayloadImpl(
        page: null == page
            ? _value.page
            : page // ignore: cast_nullable_to_non_nullable
                  as int,
        nextPage: null == nextPage
            ? _value.nextPage
            : nextPage // ignore: cast_nullable_to_non_nullable
                  as int,
        hasNext: null == hasNext
            ? _value.hasNext
            : hasNext // ignore: cast_nullable_to_non_nullable
                  as bool,
        nextUrl: null == nextUrl
            ? _value.nextUrl
            : nextUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PaginationPayloadImpl implements _PaginationPayload {
  const _$PaginationPayloadImpl({
    required this.page,
    required this.nextPage,
    required this.hasNext,
    required this.nextUrl,
  });

  factory _$PaginationPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PaginationPayloadImplFromJson(json);

  @override
  final int page;
  @override
  final int nextPage;
  @override
  final bool hasNext;
  @override
  final String nextUrl;

  @override
  String toString() {
    return 'PaginationPayload(page: $page, nextPage: $nextPage, hasNext: $hasNext, nextUrl: $nextUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PaginationPayloadImpl &&
            (identical(other.page, page) || other.page == page) &&
            (identical(other.nextPage, nextPage) ||
                other.nextPage == nextPage) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext) &&
            (identical(other.nextUrl, nextUrl) || other.nextUrl == nextUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, page, nextPage, hasNext, nextUrl);

  /// Create a copy of PaginationPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PaginationPayloadImplCopyWith<_$PaginationPayloadImpl> get copyWith =>
      __$$PaginationPayloadImplCopyWithImpl<_$PaginationPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$PaginationPayloadImplToJson(this);
  }
}

abstract class _PaginationPayload implements PaginationPayload {
  const factory _PaginationPayload({
    required final int page,
    required final int nextPage,
    required final bool hasNext,
    required final String nextUrl,
  }) = _$PaginationPayloadImpl;

  factory _PaginationPayload.fromJson(Map<String, dynamic> json) =
      _$PaginationPayloadImpl.fromJson;

  @override
  int get page;
  @override
  int get nextPage;
  @override
  bool get hasNext;
  @override
  String get nextUrl;

  /// Create a copy of PaginationPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PaginationPayloadImplCopyWith<_$PaginationPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TabItemPayload _$TabItemPayloadFromJson(Map<String, dynamic> json) {
  return _TabItemPayload.fromJson(json);
}

/// @nodoc
mixin _$TabItemPayload {
  String get key => throw _privateConstructorUsedError;
  String? get label => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  bool get active => throw _privateConstructorUsedError;

  /// Serializes this TabItemPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TabItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TabItemPayloadCopyWith<TabItemPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TabItemPayloadCopyWith<$Res> {
  factory $TabItemPayloadCopyWith(
    TabItemPayload value,
    $Res Function(TabItemPayload) then,
  ) = _$TabItemPayloadCopyWithImpl<$Res, TabItemPayload>;
  @useResult
  $Res call({String key, String? label, String url, bool active});
}

/// @nodoc
class _$TabItemPayloadCopyWithImpl<$Res, $Val extends TabItemPayload>
    implements $TabItemPayloadCopyWith<$Res> {
  _$TabItemPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TabItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = freezed,
    Object? url = null,
    Object? active = null,
  }) {
    return _then(
      _value.copyWith(
            key: null == key
                ? _value.key
                : key // ignore: cast_nullable_to_non_nullable
                      as String,
            label: freezed == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String?,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            active: null == active
                ? _value.active
                : active // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TabItemPayloadImplCopyWith<$Res>
    implements $TabItemPayloadCopyWith<$Res> {
  factory _$$TabItemPayloadImplCopyWith(
    _$TabItemPayloadImpl value,
    $Res Function(_$TabItemPayloadImpl) then,
  ) = __$$TabItemPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String key, String? label, String url, bool active});
}

/// @nodoc
class __$$TabItemPayloadImplCopyWithImpl<$Res>
    extends _$TabItemPayloadCopyWithImpl<$Res, _$TabItemPayloadImpl>
    implements _$$TabItemPayloadImplCopyWith<$Res> {
  __$$TabItemPayloadImplCopyWithImpl(
    _$TabItemPayloadImpl _value,
    $Res Function(_$TabItemPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TabItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = freezed,
    Object? url = null,
    Object? active = null,
  }) {
    return _then(
      _$TabItemPayloadImpl(
        key: null == key
            ? _value.key
            : key // ignore: cast_nullable_to_non_nullable
                  as String,
        label: freezed == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String?,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        active: null == active
            ? _value.active
            : active // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TabItemPayloadImpl implements _TabItemPayload {
  const _$TabItemPayloadImpl({
    required this.key,
    this.label,
    required this.url,
    required this.active,
  });

  factory _$TabItemPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TabItemPayloadImplFromJson(json);

  @override
  final String key;
  @override
  final String? label;
  @override
  final String url;
  @override
  final bool active;

  @override
  String toString() {
    return 'TabItemPayload(key: $key, label: $label, url: $url, active: $active)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TabItemPayloadImpl &&
            (identical(other.key, key) || other.key == key) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.active, active) || other.active == active));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, key, label, url, active);

  /// Create a copy of TabItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TabItemPayloadImplCopyWith<_$TabItemPayloadImpl> get copyWith =>
      __$$TabItemPayloadImplCopyWithImpl<_$TabItemPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TabItemPayloadImplToJson(this);
  }
}

abstract class _TabItemPayload implements TabItemPayload {
  const factory _TabItemPayload({
    required final String key,
    final String? label,
    required final String url,
    required final bool active,
  }) = _$TabItemPayloadImpl;

  factory _TabItemPayload.fromJson(Map<String, dynamic> json) =
      _$TabItemPayloadImpl.fromJson;

  @override
  String get key;
  @override
  String? get label;
  @override
  String get url;
  @override
  bool get active;

  /// Create a copy of TabItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TabItemPayloadImplCopyWith<_$TabItemPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ErrorPageProps _$ErrorPagePropsFromJson(Map<String, dynamic> json) {
  return _ErrorPageProps.fromJson(json);
}

/// @nodoc
mixin _$ErrorPageProps {
  String get code => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String? get messageCode => throw _privateConstructorUsedError;
  Map<String, dynamic>? get params => throw _privateConstructorUsedError;

  /// Serializes this ErrorPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ErrorPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ErrorPagePropsCopyWith<ErrorPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ErrorPagePropsCopyWith<$Res> {
  factory $ErrorPagePropsCopyWith(
    ErrorPageProps value,
    $Res Function(ErrorPageProps) then,
  ) = _$ErrorPagePropsCopyWithImpl<$Res, ErrorPageProps>;
  @useResult
  $Res call({
    String code,
    String title,
    String? messageCode,
    Map<String, dynamic>? params,
  });
}

/// @nodoc
class _$ErrorPagePropsCopyWithImpl<$Res, $Val extends ErrorPageProps>
    implements $ErrorPagePropsCopyWith<$Res> {
  _$ErrorPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ErrorPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? title = null,
    Object? messageCode = freezed,
    Object? params = freezed,
  }) {
    return _then(
      _value.copyWith(
            code: null == code
                ? _value.code
                : code // ignore: cast_nullable_to_non_nullable
                      as String,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            messageCode: freezed == messageCode
                ? _value.messageCode
                : messageCode // ignore: cast_nullable_to_non_nullable
                      as String?,
            params: freezed == params
                ? _value.params
                : params // ignore: cast_nullable_to_non_nullable
                      as Map<String, dynamic>?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ErrorPagePropsImplCopyWith<$Res>
    implements $ErrorPagePropsCopyWith<$Res> {
  factory _$$ErrorPagePropsImplCopyWith(
    _$ErrorPagePropsImpl value,
    $Res Function(_$ErrorPagePropsImpl) then,
  ) = __$$ErrorPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String code,
    String title,
    String? messageCode,
    Map<String, dynamic>? params,
  });
}

/// @nodoc
class __$$ErrorPagePropsImplCopyWithImpl<$Res>
    extends _$ErrorPagePropsCopyWithImpl<$Res, _$ErrorPagePropsImpl>
    implements _$$ErrorPagePropsImplCopyWith<$Res> {
  __$$ErrorPagePropsImplCopyWithImpl(
    _$ErrorPagePropsImpl _value,
    $Res Function(_$ErrorPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ErrorPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? title = null,
    Object? messageCode = freezed,
    Object? params = freezed,
  }) {
    return _then(
      _$ErrorPagePropsImpl(
        code: null == code
            ? _value.code
            : code // ignore: cast_nullable_to_non_nullable
                  as String,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        messageCode: freezed == messageCode
            ? _value.messageCode
            : messageCode // ignore: cast_nullable_to_non_nullable
                  as String?,
        params: freezed == params
            ? _value._params
            : params // ignore: cast_nullable_to_non_nullable
                  as Map<String, dynamic>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ErrorPagePropsImpl implements _ErrorPageProps {
  const _$ErrorPagePropsImpl({
    required this.code,
    required this.title,
    this.messageCode,
    final Map<String, dynamic>? params,
  }) : _params = params;

  factory _$ErrorPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$ErrorPagePropsImplFromJson(json);

  @override
  final String code;
  @override
  final String title;
  @override
  final String? messageCode;
  final Map<String, dynamic>? _params;
  @override
  Map<String, dynamic>? get params {
    final value = _params;
    if (value == null) return null;
    if (_params is EqualUnmodifiableMapView) return _params;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(value);
  }

  @override
  String toString() {
    return 'ErrorPageProps(code: $code, title: $title, messageCode: $messageCode, params: $params)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ErrorPagePropsImpl &&
            (identical(other.code, code) || other.code == code) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.messageCode, messageCode) ||
                other.messageCode == messageCode) &&
            const DeepCollectionEquality().equals(other._params, _params));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    code,
    title,
    messageCode,
    const DeepCollectionEquality().hash(_params),
  );

  /// Create a copy of ErrorPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ErrorPagePropsImplCopyWith<_$ErrorPagePropsImpl> get copyWith =>
      __$$ErrorPagePropsImplCopyWithImpl<_$ErrorPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ErrorPagePropsImplToJson(this);
  }
}

abstract class _ErrorPageProps implements ErrorPageProps {
  const factory _ErrorPageProps({
    required final String code,
    required final String title,
    final String? messageCode,
    final Map<String, dynamic>? params,
  }) = _$ErrorPagePropsImpl;

  factory _ErrorPageProps.fromJson(Map<String, dynamic> json) =
      _$ErrorPagePropsImpl.fromJson;

  @override
  String get code;
  @override
  String get title;
  @override
  String? get messageCode;
  @override
  Map<String, dynamic>? get params;

  /// Create a copy of ErrorPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ErrorPagePropsImplCopyWith<_$ErrorPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
