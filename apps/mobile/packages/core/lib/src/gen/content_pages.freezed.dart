// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'content_pages.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

LinksPageProps _$LinksPagePropsFromJson(Map<String, dynamic> json) {
  return _LinksPageProps.fromJson(json);
}

/// @nodoc
mixin _$LinksPageProps {
  List<LinkGroupPayload> get groups => throw _privateConstructorUsedError;
  int get totalCount => throw _privateConstructorUsedError;

  /// Serializes this LinksPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LinksPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LinksPagePropsCopyWith<LinksPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LinksPagePropsCopyWith<$Res> {
  factory $LinksPagePropsCopyWith(
    LinksPageProps value,
    $Res Function(LinksPageProps) then,
  ) = _$LinksPagePropsCopyWithImpl<$Res, LinksPageProps>;
  @useResult
  $Res call({List<LinkGroupPayload> groups, int totalCount});
}

/// @nodoc
class _$LinksPagePropsCopyWithImpl<$Res, $Val extends LinksPageProps>
    implements $LinksPagePropsCopyWith<$Res> {
  _$LinksPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LinksPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? groups = null, Object? totalCount = null}) {
    return _then(
      _value.copyWith(
            groups: null == groups
                ? _value.groups
                : groups // ignore: cast_nullable_to_non_nullable
                      as List<LinkGroupPayload>,
            totalCount: null == totalCount
                ? _value.totalCount
                : totalCount // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$LinksPagePropsImplCopyWith<$Res>
    implements $LinksPagePropsCopyWith<$Res> {
  factory _$$LinksPagePropsImplCopyWith(
    _$LinksPagePropsImpl value,
    $Res Function(_$LinksPagePropsImpl) then,
  ) = __$$LinksPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<LinkGroupPayload> groups, int totalCount});
}

/// @nodoc
class __$$LinksPagePropsImplCopyWithImpl<$Res>
    extends _$LinksPagePropsCopyWithImpl<$Res, _$LinksPagePropsImpl>
    implements _$$LinksPagePropsImplCopyWith<$Res> {
  __$$LinksPagePropsImplCopyWithImpl(
    _$LinksPagePropsImpl _value,
    $Res Function(_$LinksPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LinksPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? groups = null, Object? totalCount = null}) {
    return _then(
      _$LinksPagePropsImpl(
        groups: null == groups
            ? _value._groups
            : groups // ignore: cast_nullable_to_non_nullable
                  as List<LinkGroupPayload>,
        totalCount: null == totalCount
            ? _value.totalCount
            : totalCount // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LinksPagePropsImpl implements _LinksPageProps {
  const _$LinksPagePropsImpl({
    required final List<LinkGroupPayload> groups,
    required this.totalCount,
  }) : _groups = groups;

  factory _$LinksPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$LinksPagePropsImplFromJson(json);

  final List<LinkGroupPayload> _groups;
  @override
  List<LinkGroupPayload> get groups {
    if (_groups is EqualUnmodifiableListView) return _groups;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_groups);
  }

  @override
  final int totalCount;

  @override
  String toString() {
    return 'LinksPageProps(groups: $groups, totalCount: $totalCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LinksPagePropsImpl &&
            const DeepCollectionEquality().equals(other._groups, _groups) &&
            (identical(other.totalCount, totalCount) ||
                other.totalCount == totalCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_groups),
    totalCount,
  );

  /// Create a copy of LinksPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LinksPagePropsImplCopyWith<_$LinksPagePropsImpl> get copyWith =>
      __$$LinksPagePropsImplCopyWithImpl<_$LinksPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$LinksPagePropsImplToJson(this);
  }
}

abstract class _LinksPageProps implements LinksPageProps {
  const factory _LinksPageProps({
    required final List<LinkGroupPayload> groups,
    required final int totalCount,
  }) = _$LinksPagePropsImpl;

  factory _LinksPageProps.fromJson(Map<String, dynamic> json) =
      _$LinksPagePropsImpl.fromJson;

  @override
  List<LinkGroupPayload> get groups;
  @override
  int get totalCount;

  /// Create a copy of LinksPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LinksPagePropsImplCopyWith<_$LinksPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

LinkGroupPayload _$LinkGroupPayloadFromJson(Map<String, dynamic> json) {
  return _LinkGroupPayload.fromJson(json);
}

/// @nodoc
mixin _$LinkGroupPayload {
  String get name => throw _privateConstructorUsedError;
  String get emoji => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;
  List<FriendLinkPayload> get links => throw _privateConstructorUsedError;

  /// Serializes this LinkGroupPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LinkGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LinkGroupPayloadCopyWith<LinkGroupPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LinkGroupPayloadCopyWith<$Res> {
  factory $LinkGroupPayloadCopyWith(
    LinkGroupPayload value,
    $Res Function(LinkGroupPayload) then,
  ) = _$LinkGroupPayloadCopyWithImpl<$Res, LinkGroupPayload>;
  @useResult
  $Res call({
    String name,
    String emoji,
    String color,
    List<FriendLinkPayload> links,
  });
}

/// @nodoc
class _$LinkGroupPayloadCopyWithImpl<$Res, $Val extends LinkGroupPayload>
    implements $LinkGroupPayloadCopyWith<$Res> {
  _$LinkGroupPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LinkGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? emoji = null,
    Object? color = null,
    Object? links = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            emoji: null == emoji
                ? _value.emoji
                : emoji // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
            links: null == links
                ? _value.links
                : links // ignore: cast_nullable_to_non_nullable
                      as List<FriendLinkPayload>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$LinkGroupPayloadImplCopyWith<$Res>
    implements $LinkGroupPayloadCopyWith<$Res> {
  factory _$$LinkGroupPayloadImplCopyWith(
    _$LinkGroupPayloadImpl value,
    $Res Function(_$LinkGroupPayloadImpl) then,
  ) = __$$LinkGroupPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String name,
    String emoji,
    String color,
    List<FriendLinkPayload> links,
  });
}

/// @nodoc
class __$$LinkGroupPayloadImplCopyWithImpl<$Res>
    extends _$LinkGroupPayloadCopyWithImpl<$Res, _$LinkGroupPayloadImpl>
    implements _$$LinkGroupPayloadImplCopyWith<$Res> {
  __$$LinkGroupPayloadImplCopyWithImpl(
    _$LinkGroupPayloadImpl _value,
    $Res Function(_$LinkGroupPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LinkGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? emoji = null,
    Object? color = null,
    Object? links = null,
  }) {
    return _then(
      _$LinkGroupPayloadImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        emoji: null == emoji
            ? _value.emoji
            : emoji // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
        links: null == links
            ? _value._links
            : links // ignore: cast_nullable_to_non_nullable
                  as List<FriendLinkPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LinkGroupPayloadImpl implements _LinkGroupPayload {
  const _$LinkGroupPayloadImpl({
    required this.name,
    required this.emoji,
    required this.color,
    required final List<FriendLinkPayload> links,
  }) : _links = links;

  factory _$LinkGroupPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$LinkGroupPayloadImplFromJson(json);

  @override
  final String name;
  @override
  final String emoji;
  @override
  final String color;
  final List<FriendLinkPayload> _links;
  @override
  List<FriendLinkPayload> get links {
    if (_links is EqualUnmodifiableListView) return _links;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_links);
  }

  @override
  String toString() {
    return 'LinkGroupPayload(name: $name, emoji: $emoji, color: $color, links: $links)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LinkGroupPayloadImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.emoji, emoji) || other.emoji == emoji) &&
            (identical(other.color, color) || other.color == color) &&
            const DeepCollectionEquality().equals(other._links, _links));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    name,
    emoji,
    color,
    const DeepCollectionEquality().hash(_links),
  );

  /// Create a copy of LinkGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LinkGroupPayloadImplCopyWith<_$LinkGroupPayloadImpl> get copyWith =>
      __$$LinkGroupPayloadImplCopyWithImpl<_$LinkGroupPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$LinkGroupPayloadImplToJson(this);
  }
}

abstract class _LinkGroupPayload implements LinkGroupPayload {
  const factory _LinkGroupPayload({
    required final String name,
    required final String emoji,
    required final String color,
    required final List<FriendLinkPayload> links,
  }) = _$LinkGroupPayloadImpl;

  factory _LinkGroupPayload.fromJson(Map<String, dynamic> json) =
      _$LinkGroupPayloadImpl.fromJson;

  @override
  String get name;
  @override
  String get emoji;
  @override
  String get color;
  @override
  List<FriendLinkPayload> get links;

  /// Create a copy of LinkGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LinkGroupPayloadImplCopyWith<_$LinkGroupPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

FriendLinkPayload _$FriendLinkPayloadFromJson(Map<String, dynamic> json) {
  return _FriendLinkPayload.fromJson(json);
}

/// @nodoc
mixin _$FriendLinkPayload {
  String get name => throw _privateConstructorUsedError;
  String get desc => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get logoUrl => throw _privateConstructorUsedError;

  /// Serializes this FriendLinkPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FriendLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FriendLinkPayloadCopyWith<FriendLinkPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FriendLinkPayloadCopyWith<$Res> {
  factory $FriendLinkPayloadCopyWith(
    FriendLinkPayload value,
    $Res Function(FriendLinkPayload) then,
  ) = _$FriendLinkPayloadCopyWithImpl<$Res, FriendLinkPayload>;
  @useResult
  $Res call({String name, String desc, String url, String logoUrl});
}

/// @nodoc
class _$FriendLinkPayloadCopyWithImpl<$Res, $Val extends FriendLinkPayload>
    implements $FriendLinkPayloadCopyWith<$Res> {
  _$FriendLinkPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FriendLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? desc = null,
    Object? url = null,
    Object? logoUrl = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            desc: null == desc
                ? _value.desc
                : desc // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            logoUrl: null == logoUrl
                ? _value.logoUrl
                : logoUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FriendLinkPayloadImplCopyWith<$Res>
    implements $FriendLinkPayloadCopyWith<$Res> {
  factory _$$FriendLinkPayloadImplCopyWith(
    _$FriendLinkPayloadImpl value,
    $Res Function(_$FriendLinkPayloadImpl) then,
  ) = __$$FriendLinkPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String name, String desc, String url, String logoUrl});
}

/// @nodoc
class __$$FriendLinkPayloadImplCopyWithImpl<$Res>
    extends _$FriendLinkPayloadCopyWithImpl<$Res, _$FriendLinkPayloadImpl>
    implements _$$FriendLinkPayloadImplCopyWith<$Res> {
  __$$FriendLinkPayloadImplCopyWithImpl(
    _$FriendLinkPayloadImpl _value,
    $Res Function(_$FriendLinkPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FriendLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? desc = null,
    Object? url = null,
    Object? logoUrl = null,
  }) {
    return _then(
      _$FriendLinkPayloadImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        desc: null == desc
            ? _value.desc
            : desc // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        logoUrl: null == logoUrl
            ? _value.logoUrl
            : logoUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FriendLinkPayloadImpl implements _FriendLinkPayload {
  const _$FriendLinkPayloadImpl({
    required this.name,
    required this.desc,
    required this.url,
    required this.logoUrl,
  });

  factory _$FriendLinkPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$FriendLinkPayloadImplFromJson(json);

  @override
  final String name;
  @override
  final String desc;
  @override
  final String url;
  @override
  final String logoUrl;

  @override
  String toString() {
    return 'FriendLinkPayload(name: $name, desc: $desc, url: $url, logoUrl: $logoUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FriendLinkPayloadImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.desc, desc) || other.desc == desc) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.logoUrl, logoUrl) || other.logoUrl == logoUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, name, desc, url, logoUrl);

  /// Create a copy of FriendLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FriendLinkPayloadImplCopyWith<_$FriendLinkPayloadImpl> get copyWith =>
      __$$FriendLinkPayloadImplCopyWithImpl<_$FriendLinkPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$FriendLinkPayloadImplToJson(this);
  }
}

abstract class _FriendLinkPayload implements FriendLinkPayload {
  const factory _FriendLinkPayload({
    required final String name,
    required final String desc,
    required final String url,
    required final String logoUrl,
  }) = _$FriendLinkPayloadImpl;

  factory _FriendLinkPayload.fromJson(Map<String, dynamic> json) =
      _$FriendLinkPayloadImpl.fromJson;

  @override
  String get name;
  @override
  String get desc;
  @override
  String get url;
  @override
  String get logoUrl;

  /// Create a copy of FriendLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FriendLinkPayloadImplCopyWith<_$FriendLinkPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SponsorsPageProps _$SponsorsPagePropsFromJson(Map<String, dynamic> json) {
  return _SponsorsPageProps.fromJson(json);
}

/// @nodoc
mixin _$SponsorsPageProps {
  List<SponsorSectionPayload> get sections =>
      throw _privateConstructorUsedError;
  int get totalCount => throw _privateConstructorUsedError;
  SponsorsPageIntroPayload get content => throw _privateConstructorUsedError;
  SponsorsContactPayload get contact => throw _privateConstructorUsedError;
  List<SponsorsRulePayload> get rules => throw _privateConstructorUsedError;

  /// Serializes this SponsorsPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorsPagePropsCopyWith<SponsorsPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorsPagePropsCopyWith<$Res> {
  factory $SponsorsPagePropsCopyWith(
    SponsorsPageProps value,
    $Res Function(SponsorsPageProps) then,
  ) = _$SponsorsPagePropsCopyWithImpl<$Res, SponsorsPageProps>;
  @useResult
  $Res call({
    List<SponsorSectionPayload> sections,
    int totalCount,
    SponsorsPageIntroPayload content,
    SponsorsContactPayload contact,
    List<SponsorsRulePayload> rules,
  });

  $SponsorsPageIntroPayloadCopyWith<$Res> get content;
  $SponsorsContactPayloadCopyWith<$Res> get contact;
}

/// @nodoc
class _$SponsorsPagePropsCopyWithImpl<$Res, $Val extends SponsorsPageProps>
    implements $SponsorsPagePropsCopyWith<$Res> {
  _$SponsorsPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? sections = null,
    Object? totalCount = null,
    Object? content = null,
    Object? contact = null,
    Object? rules = null,
  }) {
    return _then(
      _value.copyWith(
            sections: null == sections
                ? _value.sections
                : sections // ignore: cast_nullable_to_non_nullable
                      as List<SponsorSectionPayload>,
            totalCount: null == totalCount
                ? _value.totalCount
                : totalCount // ignore: cast_nullable_to_non_nullable
                      as int,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as SponsorsPageIntroPayload,
            contact: null == contact
                ? _value.contact
                : contact // ignore: cast_nullable_to_non_nullable
                      as SponsorsContactPayload,
            rules: null == rules
                ? _value.rules
                : rules // ignore: cast_nullable_to_non_nullable
                      as List<SponsorsRulePayload>,
          )
          as $Val,
    );
  }

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SponsorsPageIntroPayloadCopyWith<$Res> get content {
    return $SponsorsPageIntroPayloadCopyWith<$Res>(_value.content, (value) {
      return _then(_value.copyWith(content: value) as $Val);
    });
  }

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SponsorsContactPayloadCopyWith<$Res> get contact {
    return $SponsorsContactPayloadCopyWith<$Res>(_value.contact, (value) {
      return _then(_value.copyWith(contact: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$SponsorsPagePropsImplCopyWith<$Res>
    implements $SponsorsPagePropsCopyWith<$Res> {
  factory _$$SponsorsPagePropsImplCopyWith(
    _$SponsorsPagePropsImpl value,
    $Res Function(_$SponsorsPagePropsImpl) then,
  ) = __$$SponsorsPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<SponsorSectionPayload> sections,
    int totalCount,
    SponsorsPageIntroPayload content,
    SponsorsContactPayload contact,
    List<SponsorsRulePayload> rules,
  });

  @override
  $SponsorsPageIntroPayloadCopyWith<$Res> get content;
  @override
  $SponsorsContactPayloadCopyWith<$Res> get contact;
}

/// @nodoc
class __$$SponsorsPagePropsImplCopyWithImpl<$Res>
    extends _$SponsorsPagePropsCopyWithImpl<$Res, _$SponsorsPagePropsImpl>
    implements _$$SponsorsPagePropsImplCopyWith<$Res> {
  __$$SponsorsPagePropsImplCopyWithImpl(
    _$SponsorsPagePropsImpl _value,
    $Res Function(_$SponsorsPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? sections = null,
    Object? totalCount = null,
    Object? content = null,
    Object? contact = null,
    Object? rules = null,
  }) {
    return _then(
      _$SponsorsPagePropsImpl(
        sections: null == sections
            ? _value._sections
            : sections // ignore: cast_nullable_to_non_nullable
                  as List<SponsorSectionPayload>,
        totalCount: null == totalCount
            ? _value.totalCount
            : totalCount // ignore: cast_nullable_to_non_nullable
                  as int,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as SponsorsPageIntroPayload,
        contact: null == contact
            ? _value.contact
            : contact // ignore: cast_nullable_to_non_nullable
                  as SponsorsContactPayload,
        rules: null == rules
            ? _value._rules
            : rules // ignore: cast_nullable_to_non_nullable
                  as List<SponsorsRulePayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorsPagePropsImpl implements _SponsorsPageProps {
  const _$SponsorsPagePropsImpl({
    required final List<SponsorSectionPayload> sections,
    required this.totalCount,
    required this.content,
    required this.contact,
    required final List<SponsorsRulePayload> rules,
  }) : _sections = sections,
       _rules = rules;

  factory _$SponsorsPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorsPagePropsImplFromJson(json);

  final List<SponsorSectionPayload> _sections;
  @override
  List<SponsorSectionPayload> get sections {
    if (_sections is EqualUnmodifiableListView) return _sections;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_sections);
  }

  @override
  final int totalCount;
  @override
  final SponsorsPageIntroPayload content;
  @override
  final SponsorsContactPayload contact;
  final List<SponsorsRulePayload> _rules;
  @override
  List<SponsorsRulePayload> get rules {
    if (_rules is EqualUnmodifiableListView) return _rules;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_rules);
  }

  @override
  String toString() {
    return 'SponsorsPageProps(sections: $sections, totalCount: $totalCount, content: $content, contact: $contact, rules: $rules)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorsPagePropsImpl &&
            const DeepCollectionEquality().equals(other._sections, _sections) &&
            (identical(other.totalCount, totalCount) ||
                other.totalCount == totalCount) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.contact, contact) || other.contact == contact) &&
            const DeepCollectionEquality().equals(other._rules, _rules));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_sections),
    totalCount,
    content,
    contact,
    const DeepCollectionEquality().hash(_rules),
  );

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorsPagePropsImplCopyWith<_$SponsorsPagePropsImpl> get copyWith =>
      __$$SponsorsPagePropsImplCopyWithImpl<_$SponsorsPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorsPagePropsImplToJson(this);
  }
}

abstract class _SponsorsPageProps implements SponsorsPageProps {
  const factory _SponsorsPageProps({
    required final List<SponsorSectionPayload> sections,
    required final int totalCount,
    required final SponsorsPageIntroPayload content,
    required final SponsorsContactPayload contact,
    required final List<SponsorsRulePayload> rules,
  }) = _$SponsorsPagePropsImpl;

  factory _SponsorsPageProps.fromJson(Map<String, dynamic> json) =
      _$SponsorsPagePropsImpl.fromJson;

  @override
  List<SponsorSectionPayload> get sections;
  @override
  int get totalCount;
  @override
  SponsorsPageIntroPayload get content;
  @override
  SponsorsContactPayload get contact;
  @override
  List<SponsorsRulePayload> get rules;

  /// Create a copy of SponsorsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorsPagePropsImplCopyWith<_$SponsorsPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SponsorSectionPayload _$SponsorSectionPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _SponsorSectionPayload.fromJson(json);
}

/// @nodoc
mixin _$SponsorSectionPayload {
  String get key => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  String get tone => throw _privateConstructorUsedError;
  List<SponsorPayload> get sponsors => throw _privateConstructorUsedError;

  /// Serializes this SponsorSectionPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorSectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorSectionPayloadCopyWith<SponsorSectionPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorSectionPayloadCopyWith<$Res> {
  factory $SponsorSectionPayloadCopyWith(
    SponsorSectionPayload value,
    $Res Function(SponsorSectionPayload) then,
  ) = _$SponsorSectionPayloadCopyWithImpl<$Res, SponsorSectionPayload>;
  @useResult
  $Res call({
    String key,
    String label,
    String tone,
    List<SponsorPayload> sponsors,
  });
}

/// @nodoc
class _$SponsorSectionPayloadCopyWithImpl<
  $Res,
  $Val extends SponsorSectionPayload
>
    implements $SponsorSectionPayloadCopyWith<$Res> {
  _$SponsorSectionPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorSectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = null,
    Object? tone = null,
    Object? sponsors = null,
  }) {
    return _then(
      _value.copyWith(
            key: null == key
                ? _value.key
                : key // ignore: cast_nullable_to_non_nullable
                      as String,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            tone: null == tone
                ? _value.tone
                : tone // ignore: cast_nullable_to_non_nullable
                      as String,
            sponsors: null == sponsors
                ? _value.sponsors
                : sponsors // ignore: cast_nullable_to_non_nullable
                      as List<SponsorPayload>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SponsorSectionPayloadImplCopyWith<$Res>
    implements $SponsorSectionPayloadCopyWith<$Res> {
  factory _$$SponsorSectionPayloadImplCopyWith(
    _$SponsorSectionPayloadImpl value,
    $Res Function(_$SponsorSectionPayloadImpl) then,
  ) = __$$SponsorSectionPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String key,
    String label,
    String tone,
    List<SponsorPayload> sponsors,
  });
}

/// @nodoc
class __$$SponsorSectionPayloadImplCopyWithImpl<$Res>
    extends
        _$SponsorSectionPayloadCopyWithImpl<$Res, _$SponsorSectionPayloadImpl>
    implements _$$SponsorSectionPayloadImplCopyWith<$Res> {
  __$$SponsorSectionPayloadImplCopyWithImpl(
    _$SponsorSectionPayloadImpl _value,
    $Res Function(_$SponsorSectionPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorSectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = null,
    Object? tone = null,
    Object? sponsors = null,
  }) {
    return _then(
      _$SponsorSectionPayloadImpl(
        key: null == key
            ? _value.key
            : key // ignore: cast_nullable_to_non_nullable
                  as String,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
        tone: null == tone
            ? _value.tone
            : tone // ignore: cast_nullable_to_non_nullable
                  as String,
        sponsors: null == sponsors
            ? _value._sponsors
            : sponsors // ignore: cast_nullable_to_non_nullable
                  as List<SponsorPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorSectionPayloadImpl implements _SponsorSectionPayload {
  const _$SponsorSectionPayloadImpl({
    required this.key,
    required this.label,
    required this.tone,
    required final List<SponsorPayload> sponsors,
  }) : _sponsors = sponsors;

  factory _$SponsorSectionPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorSectionPayloadImplFromJson(json);

  @override
  final String key;
  @override
  final String label;
  @override
  final String tone;
  final List<SponsorPayload> _sponsors;
  @override
  List<SponsorPayload> get sponsors {
    if (_sponsors is EqualUnmodifiableListView) return _sponsors;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_sponsors);
  }

  @override
  String toString() {
    return 'SponsorSectionPayload(key: $key, label: $label, tone: $tone, sponsors: $sponsors)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorSectionPayloadImpl &&
            (identical(other.key, key) || other.key == key) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.tone, tone) || other.tone == tone) &&
            const DeepCollectionEquality().equals(other._sponsors, _sponsors));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    key,
    label,
    tone,
    const DeepCollectionEquality().hash(_sponsors),
  );

  /// Create a copy of SponsorSectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorSectionPayloadImplCopyWith<_$SponsorSectionPayloadImpl>
  get copyWith =>
      __$$SponsorSectionPayloadImplCopyWithImpl<_$SponsorSectionPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorSectionPayloadImplToJson(this);
  }
}

abstract class _SponsorSectionPayload implements SponsorSectionPayload {
  const factory _SponsorSectionPayload({
    required final String key,
    required final String label,
    required final String tone,
    required final List<SponsorPayload> sponsors,
  }) = _$SponsorSectionPayloadImpl;

  factory _SponsorSectionPayload.fromJson(Map<String, dynamic> json) =
      _$SponsorSectionPayloadImpl.fromJson;

  @override
  String get key;
  @override
  String get label;
  @override
  String get tone;
  @override
  List<SponsorPayload> get sponsors;

  /// Create a copy of SponsorSectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorSectionPayloadImplCopyWith<_$SponsorSectionPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

SponsorPayload _$SponsorPayloadFromJson(Map<String, dynamic> json) {
  return _SponsorPayload.fromJson(json);
}

/// @nodoc
mixin _$SponsorPayload {
  String get name => throw _privateConstructorUsedError;
  String get message => throw _privateConstructorUsedError;
  String get link => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;

  /// Serializes this SponsorPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorPayloadCopyWith<SponsorPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorPayloadCopyWith<$Res> {
  factory $SponsorPayloadCopyWith(
    SponsorPayload value,
    $Res Function(SponsorPayload) then,
  ) = _$SponsorPayloadCopyWithImpl<$Res, SponsorPayload>;
  @useResult
  $Res call({String name, String message, String link, String avatarUrl});
}

/// @nodoc
class _$SponsorPayloadCopyWithImpl<$Res, $Val extends SponsorPayload>
    implements $SponsorPayloadCopyWith<$Res> {
  _$SponsorPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? message = null,
    Object? link = null,
    Object? avatarUrl = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            message: null == message
                ? _value.message
                : message // ignore: cast_nullable_to_non_nullable
                      as String,
            link: null == link
                ? _value.link
                : link // ignore: cast_nullable_to_non_nullable
                      as String,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SponsorPayloadImplCopyWith<$Res>
    implements $SponsorPayloadCopyWith<$Res> {
  factory _$$SponsorPayloadImplCopyWith(
    _$SponsorPayloadImpl value,
    $Res Function(_$SponsorPayloadImpl) then,
  ) = __$$SponsorPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String name, String message, String link, String avatarUrl});
}

/// @nodoc
class __$$SponsorPayloadImplCopyWithImpl<$Res>
    extends _$SponsorPayloadCopyWithImpl<$Res, _$SponsorPayloadImpl>
    implements _$$SponsorPayloadImplCopyWith<$Res> {
  __$$SponsorPayloadImplCopyWithImpl(
    _$SponsorPayloadImpl _value,
    $Res Function(_$SponsorPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? message = null,
    Object? link = null,
    Object? avatarUrl = null,
  }) {
    return _then(
      _$SponsorPayloadImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        message: null == message
            ? _value.message
            : message // ignore: cast_nullable_to_non_nullable
                  as String,
        link: null == link
            ? _value.link
            : link // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorPayloadImpl implements _SponsorPayload {
  const _$SponsorPayloadImpl({
    required this.name,
    required this.message,
    required this.link,
    required this.avatarUrl,
  });

  factory _$SponsorPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorPayloadImplFromJson(json);

  @override
  final String name;
  @override
  final String message;
  @override
  final String link;
  @override
  final String avatarUrl;

  @override
  String toString() {
    return 'SponsorPayload(name: $name, message: $message, link: $link, avatarUrl: $avatarUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorPayloadImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.message, message) || other.message == message) &&
            (identical(other.link, link) || other.link == link) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, name, message, link, avatarUrl);

  /// Create a copy of SponsorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorPayloadImplCopyWith<_$SponsorPayloadImpl> get copyWith =>
      __$$SponsorPayloadImplCopyWithImpl<_$SponsorPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorPayloadImplToJson(this);
  }
}

abstract class _SponsorPayload implements SponsorPayload {
  const factory _SponsorPayload({
    required final String name,
    required final String message,
    required final String link,
    required final String avatarUrl,
  }) = _$SponsorPayloadImpl;

  factory _SponsorPayload.fromJson(Map<String, dynamic> json) =
      _$SponsorPayloadImpl.fromJson;

  @override
  String get name;
  @override
  String get message;
  @override
  String get link;
  @override
  String get avatarUrl;

  /// Create a copy of SponsorPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorPayloadImplCopyWith<_$SponsorPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SponsorsPageIntroPayload _$SponsorsPageIntroPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _SponsorsPageIntroPayload.fromJson(json);
}

/// @nodoc
mixin _$SponsorsPageIntroPayload {
  String get title => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;

  /// Serializes this SponsorsPageIntroPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorsPageIntroPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorsPageIntroPayloadCopyWith<SponsorsPageIntroPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorsPageIntroPayloadCopyWith<$Res> {
  factory $SponsorsPageIntroPayloadCopyWith(
    SponsorsPageIntroPayload value,
    $Res Function(SponsorsPageIntroPayload) then,
  ) = _$SponsorsPageIntroPayloadCopyWithImpl<$Res, SponsorsPageIntroPayload>;
  @useResult
  $Res call({String title, String description});
}

/// @nodoc
class _$SponsorsPageIntroPayloadCopyWithImpl<
  $Res,
  $Val extends SponsorsPageIntroPayload
>
    implements $SponsorsPageIntroPayloadCopyWith<$Res> {
  _$SponsorsPageIntroPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorsPageIntroPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? title = null, Object? description = null}) {
    return _then(
      _value.copyWith(
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SponsorsPageIntroPayloadImplCopyWith<$Res>
    implements $SponsorsPageIntroPayloadCopyWith<$Res> {
  factory _$$SponsorsPageIntroPayloadImplCopyWith(
    _$SponsorsPageIntroPayloadImpl value,
    $Res Function(_$SponsorsPageIntroPayloadImpl) then,
  ) = __$$SponsorsPageIntroPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String title, String description});
}

/// @nodoc
class __$$SponsorsPageIntroPayloadImplCopyWithImpl<$Res>
    extends
        _$SponsorsPageIntroPayloadCopyWithImpl<
          $Res,
          _$SponsorsPageIntroPayloadImpl
        >
    implements _$$SponsorsPageIntroPayloadImplCopyWith<$Res> {
  __$$SponsorsPageIntroPayloadImplCopyWithImpl(
    _$SponsorsPageIntroPayloadImpl _value,
    $Res Function(_$SponsorsPageIntroPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorsPageIntroPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? title = null, Object? description = null}) {
    return _then(
      _$SponsorsPageIntroPayloadImpl(
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorsPageIntroPayloadImpl implements _SponsorsPageIntroPayload {
  const _$SponsorsPageIntroPayloadImpl({
    required this.title,
    required this.description,
  });

  factory _$SponsorsPageIntroPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorsPageIntroPayloadImplFromJson(json);

  @override
  final String title;
  @override
  final String description;

  @override
  String toString() {
    return 'SponsorsPageIntroPayload(title: $title, description: $description)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorsPageIntroPayloadImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, title, description);

  /// Create a copy of SponsorsPageIntroPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorsPageIntroPayloadImplCopyWith<_$SponsorsPageIntroPayloadImpl>
  get copyWith =>
      __$$SponsorsPageIntroPayloadImplCopyWithImpl<
        _$SponsorsPageIntroPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorsPageIntroPayloadImplToJson(this);
  }
}

abstract class _SponsorsPageIntroPayload implements SponsorsPageIntroPayload {
  const factory _SponsorsPageIntroPayload({
    required final String title,
    required final String description,
  }) = _$SponsorsPageIntroPayloadImpl;

  factory _SponsorsPageIntroPayload.fromJson(Map<String, dynamic> json) =
      _$SponsorsPageIntroPayloadImpl.fromJson;

  @override
  String get title;
  @override
  String get description;

  /// Create a copy of SponsorsPageIntroPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorsPageIntroPayloadImplCopyWith<_$SponsorsPageIntroPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

SponsorsContactPayload _$SponsorsContactPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _SponsorsContactPayload.fromJson(json);
}

/// @nodoc
mixin _$SponsorsContactPayload {
  String get title => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get buttonText => throw _privateConstructorUsedError;
  String get buttonLink => throw _privateConstructorUsedError;

  /// Serializes this SponsorsContactPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorsContactPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorsContactPayloadCopyWith<SponsorsContactPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorsContactPayloadCopyWith<$Res> {
  factory $SponsorsContactPayloadCopyWith(
    SponsorsContactPayload value,
    $Res Function(SponsorsContactPayload) then,
  ) = _$SponsorsContactPayloadCopyWithImpl<$Res, SponsorsContactPayload>;
  @useResult
  $Res call({
    String title,
    String description,
    String buttonText,
    String buttonLink,
  });
}

/// @nodoc
class _$SponsorsContactPayloadCopyWithImpl<
  $Res,
  $Val extends SponsorsContactPayload
>
    implements $SponsorsContactPayloadCopyWith<$Res> {
  _$SponsorsContactPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorsContactPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? description = null,
    Object? buttonText = null,
    Object? buttonLink = null,
  }) {
    return _then(
      _value.copyWith(
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            buttonText: null == buttonText
                ? _value.buttonText
                : buttonText // ignore: cast_nullable_to_non_nullable
                      as String,
            buttonLink: null == buttonLink
                ? _value.buttonLink
                : buttonLink // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SponsorsContactPayloadImplCopyWith<$Res>
    implements $SponsorsContactPayloadCopyWith<$Res> {
  factory _$$SponsorsContactPayloadImplCopyWith(
    _$SponsorsContactPayloadImpl value,
    $Res Function(_$SponsorsContactPayloadImpl) then,
  ) = __$$SponsorsContactPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String title,
    String description,
    String buttonText,
    String buttonLink,
  });
}

/// @nodoc
class __$$SponsorsContactPayloadImplCopyWithImpl<$Res>
    extends
        _$SponsorsContactPayloadCopyWithImpl<$Res, _$SponsorsContactPayloadImpl>
    implements _$$SponsorsContactPayloadImplCopyWith<$Res> {
  __$$SponsorsContactPayloadImplCopyWithImpl(
    _$SponsorsContactPayloadImpl _value,
    $Res Function(_$SponsorsContactPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorsContactPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? description = null,
    Object? buttonText = null,
    Object? buttonLink = null,
  }) {
    return _then(
      _$SponsorsContactPayloadImpl(
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        buttonText: null == buttonText
            ? _value.buttonText
            : buttonText // ignore: cast_nullable_to_non_nullable
                  as String,
        buttonLink: null == buttonLink
            ? _value.buttonLink
            : buttonLink // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorsContactPayloadImpl implements _SponsorsContactPayload {
  const _$SponsorsContactPayloadImpl({
    required this.title,
    required this.description,
    required this.buttonText,
    required this.buttonLink,
  });

  factory _$SponsorsContactPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorsContactPayloadImplFromJson(json);

  @override
  final String title;
  @override
  final String description;
  @override
  final String buttonText;
  @override
  final String buttonLink;

  @override
  String toString() {
    return 'SponsorsContactPayload(title: $title, description: $description, buttonText: $buttonText, buttonLink: $buttonLink)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorsContactPayloadImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.buttonText, buttonText) ||
                other.buttonText == buttonText) &&
            (identical(other.buttonLink, buttonLink) ||
                other.buttonLink == buttonLink));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, title, description, buttonText, buttonLink);

  /// Create a copy of SponsorsContactPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorsContactPayloadImplCopyWith<_$SponsorsContactPayloadImpl>
  get copyWith =>
      __$$SponsorsContactPayloadImplCopyWithImpl<_$SponsorsContactPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorsContactPayloadImplToJson(this);
  }
}

abstract class _SponsorsContactPayload implements SponsorsContactPayload {
  const factory _SponsorsContactPayload({
    required final String title,
    required final String description,
    required final String buttonText,
    required final String buttonLink,
  }) = _$SponsorsContactPayloadImpl;

  factory _SponsorsContactPayload.fromJson(Map<String, dynamic> json) =
      _$SponsorsContactPayloadImpl.fromJson;

  @override
  String get title;
  @override
  String get description;
  @override
  String get buttonText;
  @override
  String get buttonLink;

  /// Create a copy of SponsorsContactPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorsContactPayloadImplCopyWith<_$SponsorsContactPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

SponsorsRulePayload _$SponsorsRulePayloadFromJson(Map<String, dynamic> json) {
  return _SponsorsRulePayload.fromJson(json);
}

/// @nodoc
mixin _$SponsorsRulePayload {
  String get content => throw _privateConstructorUsedError;

  /// Serializes this SponsorsRulePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SponsorsRulePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SponsorsRulePayloadCopyWith<SponsorsRulePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SponsorsRulePayloadCopyWith<$Res> {
  factory $SponsorsRulePayloadCopyWith(
    SponsorsRulePayload value,
    $Res Function(SponsorsRulePayload) then,
  ) = _$SponsorsRulePayloadCopyWithImpl<$Res, SponsorsRulePayload>;
  @useResult
  $Res call({String content});
}

/// @nodoc
class _$SponsorsRulePayloadCopyWithImpl<$Res, $Val extends SponsorsRulePayload>
    implements $SponsorsRulePayloadCopyWith<$Res> {
  _$SponsorsRulePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SponsorsRulePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? content = null}) {
    return _then(
      _value.copyWith(
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SponsorsRulePayloadImplCopyWith<$Res>
    implements $SponsorsRulePayloadCopyWith<$Res> {
  factory _$$SponsorsRulePayloadImplCopyWith(
    _$SponsorsRulePayloadImpl value,
    $Res Function(_$SponsorsRulePayloadImpl) then,
  ) = __$$SponsorsRulePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String content});
}

/// @nodoc
class __$$SponsorsRulePayloadImplCopyWithImpl<$Res>
    extends _$SponsorsRulePayloadCopyWithImpl<$Res, _$SponsorsRulePayloadImpl>
    implements _$$SponsorsRulePayloadImplCopyWith<$Res> {
  __$$SponsorsRulePayloadImplCopyWithImpl(
    _$SponsorsRulePayloadImpl _value,
    $Res Function(_$SponsorsRulePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SponsorsRulePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? content = null}) {
    return _then(
      _$SponsorsRulePayloadImpl(
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SponsorsRulePayloadImpl implements _SponsorsRulePayload {
  const _$SponsorsRulePayloadImpl({required this.content});

  factory _$SponsorsRulePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SponsorsRulePayloadImplFromJson(json);

  @override
  final String content;

  @override
  String toString() {
    return 'SponsorsRulePayload(content: $content)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SponsorsRulePayloadImpl &&
            (identical(other.content, content) || other.content == content));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, content);

  /// Create a copy of SponsorsRulePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SponsorsRulePayloadImplCopyWith<_$SponsorsRulePayloadImpl> get copyWith =>
      __$$SponsorsRulePayloadImplCopyWithImpl<_$SponsorsRulePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SponsorsRulePayloadImplToJson(this);
  }
}

abstract class _SponsorsRulePayload implements SponsorsRulePayload {
  const factory _SponsorsRulePayload({required final String content}) =
      _$SponsorsRulePayloadImpl;

  factory _SponsorsRulePayload.fromJson(Map<String, dynamic> json) =
      _$SponsorsRulePayloadImpl.fromJson;

  @override
  String get content;

  /// Create a copy of SponsorsRulePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SponsorsRulePayloadImplCopyWith<_$SponsorsRulePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TermsPageProps _$TermsPagePropsFromJson(Map<String, dynamic> json) {
  return _TermsPageProps.fromJson(json);
}

/// @nodoc
mixin _$TermsPageProps {
  bool get enabled => throw _privateConstructorUsedError;
  String get contentHtml => throw _privateConstructorUsedError;

  /// Serializes this TermsPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TermsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TermsPagePropsCopyWith<TermsPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TermsPagePropsCopyWith<$Res> {
  factory $TermsPagePropsCopyWith(
    TermsPageProps value,
    $Res Function(TermsPageProps) then,
  ) = _$TermsPagePropsCopyWithImpl<$Res, TermsPageProps>;
  @useResult
  $Res call({bool enabled, String contentHtml});
}

/// @nodoc
class _$TermsPagePropsCopyWithImpl<$Res, $Val extends TermsPageProps>
    implements $TermsPagePropsCopyWith<$Res> {
  _$TermsPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TermsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? enabled = null, Object? contentHtml = null}) {
    return _then(
      _value.copyWith(
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            contentHtml: null == contentHtml
                ? _value.contentHtml
                : contentHtml // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TermsPagePropsImplCopyWith<$Res>
    implements $TermsPagePropsCopyWith<$Res> {
  factory _$$TermsPagePropsImplCopyWith(
    _$TermsPagePropsImpl value,
    $Res Function(_$TermsPagePropsImpl) then,
  ) = __$$TermsPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool enabled, String contentHtml});
}

/// @nodoc
class __$$TermsPagePropsImplCopyWithImpl<$Res>
    extends _$TermsPagePropsCopyWithImpl<$Res, _$TermsPagePropsImpl>
    implements _$$TermsPagePropsImplCopyWith<$Res> {
  __$$TermsPagePropsImplCopyWithImpl(
    _$TermsPagePropsImpl _value,
    $Res Function(_$TermsPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TermsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? enabled = null, Object? contentHtml = null}) {
    return _then(
      _$TermsPagePropsImpl(
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        contentHtml: null == contentHtml
            ? _value.contentHtml
            : contentHtml // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TermsPagePropsImpl implements _TermsPageProps {
  const _$TermsPagePropsImpl({
    required this.enabled,
    required this.contentHtml,
  });

  factory _$TermsPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$TermsPagePropsImplFromJson(json);

  @override
  final bool enabled;
  @override
  final String contentHtml;

  @override
  String toString() {
    return 'TermsPageProps(enabled: $enabled, contentHtml: $contentHtml)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TermsPagePropsImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            (identical(other.contentHtml, contentHtml) ||
                other.contentHtml == contentHtml));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, enabled, contentHtml);

  /// Create a copy of TermsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TermsPagePropsImplCopyWith<_$TermsPagePropsImpl> get copyWith =>
      __$$TermsPagePropsImplCopyWithImpl<_$TermsPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TermsPagePropsImplToJson(this);
  }
}

abstract class _TermsPageProps implements TermsPageProps {
  const factory _TermsPageProps({
    required final bool enabled,
    required final String contentHtml,
  }) = _$TermsPagePropsImpl;

  factory _TermsPageProps.fromJson(Map<String, dynamic> json) =
      _$TermsPagePropsImpl.fromJson;

  @override
  bool get enabled;
  @override
  String get contentHtml;

  /// Create a copy of TermsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TermsPagePropsImplCopyWith<_$TermsPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
