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

CourseSummaryPayload _$CourseSummaryPayloadFromJson(Map<String, dynamic> json) {
  return _CourseSummaryPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseSummaryPayload {
  int get id => throw _privateConstructorUsedError;
  String get primaryCode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get department => throw _privateConstructorUsedError;
  int get creditX10 => throw _privateConstructorUsedError;
  List<String>? get aliases => throw _privateConstructorUsedError;
  List<String>? get instructors => throw _privateConstructorUsedError;
  List<String>? get recentTerms =>
      throw _privateConstructorUsedError; // B1 统计投影（PRD §5.1）：非 NULL 评分均分 / 可见评价数；无评分时省略。
  double? get ratingAvg => throw _privateConstructorUsedError;
  int? get reviewCount => throw _privateConstructorUsedError;

  /// Serializes this CourseSummaryPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseSummaryPayloadCopyWith<CourseSummaryPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseSummaryPayloadCopyWith<$Res> {
  factory $CourseSummaryPayloadCopyWith(
    CourseSummaryPayload value,
    $Res Function(CourseSummaryPayload) then,
  ) = _$CourseSummaryPayloadCopyWithImpl<$Res, CourseSummaryPayload>;
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    int creditX10,
    List<String>? aliases,
    List<String>? instructors,
    List<String>? recentTerms,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class _$CourseSummaryPayloadCopyWithImpl<
  $Res,
  $Val extends CourseSummaryPayload
>
    implements $CourseSummaryPayloadCopyWith<$Res> {
  _$CourseSummaryPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? creditX10 = null,
    Object? aliases = freezed,
    Object? instructors = freezed,
    Object? recentTerms = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
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
            creditX10: null == creditX10
                ? _value.creditX10
                : creditX10 // ignore: cast_nullable_to_non_nullable
                      as int,
            aliases: freezed == aliases
                ? _value.aliases
                : aliases // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            instructors: freezed == instructors
                ? _value.instructors
                : instructors // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            recentTerms: freezed == recentTerms
                ? _value.recentTerms
                : recentTerms // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            ratingAvg: freezed == ratingAvg
                ? _value.ratingAvg
                : ratingAvg // ignore: cast_nullable_to_non_nullable
                      as double?,
            reviewCount: freezed == reviewCount
                ? _value.reviewCount
                : reviewCount // ignore: cast_nullable_to_non_nullable
                      as int?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseSummaryPayloadImplCopyWith<$Res>
    implements $CourseSummaryPayloadCopyWith<$Res> {
  factory _$$CourseSummaryPayloadImplCopyWith(
    _$CourseSummaryPayloadImpl value,
    $Res Function(_$CourseSummaryPayloadImpl) then,
  ) = __$$CourseSummaryPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    int creditX10,
    List<String>? aliases,
    List<String>? instructors,
    List<String>? recentTerms,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class __$$CourseSummaryPayloadImplCopyWithImpl<$Res>
    extends _$CourseSummaryPayloadCopyWithImpl<$Res, _$CourseSummaryPayloadImpl>
    implements _$$CourseSummaryPayloadImplCopyWith<$Res> {
  __$$CourseSummaryPayloadImplCopyWithImpl(
    _$CourseSummaryPayloadImpl _value,
    $Res Function(_$CourseSummaryPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? creditX10 = null,
    Object? aliases = freezed,
    Object? instructors = freezed,
    Object? recentTerms = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
  }) {
    return _then(
      _$CourseSummaryPayloadImpl(
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
        creditX10: null == creditX10
            ? _value.creditX10
            : creditX10 // ignore: cast_nullable_to_non_nullable
                  as int,
        aliases: freezed == aliases
            ? _value._aliases
            : aliases // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        instructors: freezed == instructors
            ? _value._instructors
            : instructors // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        recentTerms: freezed == recentTerms
            ? _value._recentTerms
            : recentTerms // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        ratingAvg: freezed == ratingAvg
            ? _value.ratingAvg
            : ratingAvg // ignore: cast_nullable_to_non_nullable
                  as double?,
        reviewCount: freezed == reviewCount
            ? _value.reviewCount
            : reviewCount // ignore: cast_nullable_to_non_nullable
                  as int?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseSummaryPayloadImpl implements _CourseSummaryPayload {
  const _$CourseSummaryPayloadImpl({
    required this.id,
    required this.primaryCode,
    required this.name,
    required this.department,
    required this.creditX10,
    final List<String>? aliases,
    final List<String>? instructors,
    final List<String>? recentTerms,
    this.ratingAvg,
    this.reviewCount,
  }) : _aliases = aliases,
       _instructors = instructors,
       _recentTerms = recentTerms;

  factory _$CourseSummaryPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseSummaryPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String primaryCode;
  @override
  final String name;
  @override
  final String department;
  @override
  final int creditX10;
  final List<String>? _aliases;
  @override
  List<String>? get aliases {
    final value = _aliases;
    if (value == null) return null;
    if (_aliases is EqualUnmodifiableListView) return _aliases;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<String>? _instructors;
  @override
  List<String>? get instructors {
    final value = _instructors;
    if (value == null) return null;
    if (_instructors is EqualUnmodifiableListView) return _instructors;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<String>? _recentTerms;
  @override
  List<String>? get recentTerms {
    final value = _recentTerms;
    if (value == null) return null;
    if (_recentTerms is EqualUnmodifiableListView) return _recentTerms;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  // B1 统计投影（PRD §5.1）：非 NULL 评分均分 / 可见评价数；无评分时省略。
  @override
  final double? ratingAvg;
  @override
  final int? reviewCount;

  @override
  String toString() {
    return 'CourseSummaryPayload(id: $id, primaryCode: $primaryCode, name: $name, department: $department, creditX10: $creditX10, aliases: $aliases, instructors: $instructors, recentTerms: $recentTerms, ratingAvg: $ratingAvg, reviewCount: $reviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseSummaryPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.primaryCode, primaryCode) ||
                other.primaryCode == primaryCode) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.department, department) ||
                other.department == department) &&
            (identical(other.creditX10, creditX10) ||
                other.creditX10 == creditX10) &&
            const DeepCollectionEquality().equals(other._aliases, _aliases) &&
            const DeepCollectionEquality().equals(
              other._instructors,
              _instructors,
            ) &&
            const DeepCollectionEquality().equals(
              other._recentTerms,
              _recentTerms,
            ) &&
            (identical(other.ratingAvg, ratingAvg) ||
                other.ratingAvg == ratingAvg) &&
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
    creditX10,
    const DeepCollectionEquality().hash(_aliases),
    const DeepCollectionEquality().hash(_instructors),
    const DeepCollectionEquality().hash(_recentTerms),
    ratingAvg,
    reviewCount,
  );

  /// Create a copy of CourseSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseSummaryPayloadImplCopyWith<_$CourseSummaryPayloadImpl>
  get copyWith =>
      __$$CourseSummaryPayloadImplCopyWithImpl<_$CourseSummaryPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseSummaryPayloadImplToJson(this);
  }
}

abstract class _CourseSummaryPayload implements CourseSummaryPayload {
  const factory _CourseSummaryPayload({
    required final int id,
    required final String primaryCode,
    required final String name,
    required final String department,
    required final int creditX10,
    final List<String>? aliases,
    final List<String>? instructors,
    final List<String>? recentTerms,
    final double? ratingAvg,
    final int? reviewCount,
  }) = _$CourseSummaryPayloadImpl;

  factory _CourseSummaryPayload.fromJson(Map<String, dynamic> json) =
      _$CourseSummaryPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get primaryCode;
  @override
  String get name;
  @override
  String get department;
  @override
  int get creditX10;
  @override
  List<String>? get aliases;
  @override
  List<String>? get instructors;
  @override
  List<String>? get recentTerms; // B1 统计投影（PRD §5.1）：非 NULL 评分均分 / 可见评价数；无评分时省略。
  @override
  double? get ratingAvg;
  @override
  int? get reviewCount;

  /// Create a copy of CourseSummaryPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseSummaryPayloadImplCopyWith<_$CourseSummaryPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseCatalogPageProps _$CourseCatalogPagePropsFromJson(
  Map<String, dynamic> json,
) {
  return _CourseCatalogPageProps.fromJson(json);
}

/// @nodoc
mixin _$CourseCatalogPageProps {
  CourseCatalogQueryPayload get query => throw _privateConstructorUsedError;
  List<CourseSummaryPayload> get courses => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;
  List<String> get departments => throw _privateConstructorUsedError;

  /// Serializes this CourseCatalogPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseCatalogPagePropsCopyWith<CourseCatalogPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseCatalogPagePropsCopyWith<$Res> {
  factory $CourseCatalogPagePropsCopyWith(
    CourseCatalogPageProps value,
    $Res Function(CourseCatalogPageProps) then,
  ) = _$CourseCatalogPagePropsCopyWithImpl<$Res, CourseCatalogPageProps>;
  @useResult
  $Res call({
    CourseCatalogQueryPayload query,
    List<CourseSummaryPayload> courses,
    PaginationPayload pagination,
    List<String> departments,
  });

  $CourseCatalogQueryPayloadCopyWith<$Res> get query;
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$CourseCatalogPagePropsCopyWithImpl<
  $Res,
  $Val extends CourseCatalogPageProps
>
    implements $CourseCatalogPagePropsCopyWith<$Res> {
  _$CourseCatalogPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? query = null,
    Object? courses = null,
    Object? pagination = null,
    Object? departments = null,
  }) {
    return _then(
      _value.copyWith(
            query: null == query
                ? _value.query
                : query // ignore: cast_nullable_to_non_nullable
                      as CourseCatalogQueryPayload,
            courses: null == courses
                ? _value.courses
                : courses // ignore: cast_nullable_to_non_nullable
                      as List<CourseSummaryPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
            departments: null == departments
                ? _value.departments
                : departments // ignore: cast_nullable_to_non_nullable
                      as List<String>,
          )
          as $Val,
    );
  }

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $CourseCatalogQueryPayloadCopyWith<$Res> get query {
    return $CourseCatalogQueryPayloadCopyWith<$Res>(_value.query, (value) {
      return _then(_value.copyWith(query: value) as $Val);
    });
  }

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PaginationPayloadCopyWith<$Res> get pagination {
    return $PaginationPayloadCopyWith<$Res>(_value.pagination, (value) {
      return _then(_value.copyWith(pagination: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$CourseCatalogPagePropsImplCopyWith<$Res>
    implements $CourseCatalogPagePropsCopyWith<$Res> {
  factory _$$CourseCatalogPagePropsImplCopyWith(
    _$CourseCatalogPagePropsImpl value,
    $Res Function(_$CourseCatalogPagePropsImpl) then,
  ) = __$$CourseCatalogPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    CourseCatalogQueryPayload query,
    List<CourseSummaryPayload> courses,
    PaginationPayload pagination,
    List<String> departments,
  });

  @override
  $CourseCatalogQueryPayloadCopyWith<$Res> get query;
  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$CourseCatalogPagePropsImplCopyWithImpl<$Res>
    extends
        _$CourseCatalogPagePropsCopyWithImpl<$Res, _$CourseCatalogPagePropsImpl>
    implements _$$CourseCatalogPagePropsImplCopyWith<$Res> {
  __$$CourseCatalogPagePropsImplCopyWithImpl(
    _$CourseCatalogPagePropsImpl _value,
    $Res Function(_$CourseCatalogPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? query = null,
    Object? courses = null,
    Object? pagination = null,
    Object? departments = null,
  }) {
    return _then(
      _$CourseCatalogPagePropsImpl(
        query: null == query
            ? _value.query
            : query // ignore: cast_nullable_to_non_nullable
                  as CourseCatalogQueryPayload,
        courses: null == courses
            ? _value._courses
            : courses // ignore: cast_nullable_to_non_nullable
                  as List<CourseSummaryPayload>,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
        departments: null == departments
            ? _value._departments
            : departments // ignore: cast_nullable_to_non_nullable
                  as List<String>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseCatalogPagePropsImpl implements _CourseCatalogPageProps {
  const _$CourseCatalogPagePropsImpl({
    required this.query,
    required final List<CourseSummaryPayload> courses,
    required this.pagination,
    required final List<String> departments,
  }) : _courses = courses,
       _departments = departments;

  factory _$CourseCatalogPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseCatalogPagePropsImplFromJson(json);

  @override
  final CourseCatalogQueryPayload query;
  final List<CourseSummaryPayload> _courses;
  @override
  List<CourseSummaryPayload> get courses {
    if (_courses is EqualUnmodifiableListView) return _courses;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_courses);
  }

  @override
  final PaginationPayload pagination;
  final List<String> _departments;
  @override
  List<String> get departments {
    if (_departments is EqualUnmodifiableListView) return _departments;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_departments);
  }

  @override
  String toString() {
    return 'CourseCatalogPageProps(query: $query, courses: $courses, pagination: $pagination, departments: $departments)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseCatalogPagePropsImpl &&
            (identical(other.query, query) || other.query == query) &&
            const DeepCollectionEquality().equals(other._courses, _courses) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination) &&
            const DeepCollectionEquality().equals(
              other._departments,
              _departments,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    query,
    const DeepCollectionEquality().hash(_courses),
    pagination,
    const DeepCollectionEquality().hash(_departments),
  );

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseCatalogPagePropsImplCopyWith<_$CourseCatalogPagePropsImpl>
  get copyWith =>
      __$$CourseCatalogPagePropsImplCopyWithImpl<_$CourseCatalogPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseCatalogPagePropsImplToJson(this);
  }
}

abstract class _CourseCatalogPageProps implements CourseCatalogPageProps {
  const factory _CourseCatalogPageProps({
    required final CourseCatalogQueryPayload query,
    required final List<CourseSummaryPayload> courses,
    required final PaginationPayload pagination,
    required final List<String> departments,
  }) = _$CourseCatalogPagePropsImpl;

  factory _CourseCatalogPageProps.fromJson(Map<String, dynamic> json) =
      _$CourseCatalogPagePropsImpl.fromJson;

  @override
  CourseCatalogQueryPayload get query;
  @override
  List<CourseSummaryPayload> get courses;
  @override
  PaginationPayload get pagination;
  @override
  List<String> get departments;

  /// Create a copy of CourseCatalogPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseCatalogPagePropsImplCopyWith<_$CourseCatalogPagePropsImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseCatalogQueryPayload _$CourseCatalogQueryPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CourseCatalogQueryPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseCatalogQueryPayload {
  String? get keyword => throw _privateConstructorUsedError;
  String? get department => throw _privateConstructorUsedError;
  String? get term => throw _privateConstructorUsedError;
  String? get campus => throw _privateConstructorUsedError;
  String? get instructor => throw _privateConstructorUsedError;
  bool? get onlyWithReviews => throw _privateConstructorUsedError;
  String? get sortBy => throw _privateConstructorUsedError;
  int get page => throw _privateConstructorUsedError;
  int get size => throw _privateConstructorUsedError;

  /// Serializes this CourseCatalogQueryPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseCatalogQueryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseCatalogQueryPayloadCopyWith<CourseCatalogQueryPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseCatalogQueryPayloadCopyWith<$Res> {
  factory $CourseCatalogQueryPayloadCopyWith(
    CourseCatalogQueryPayload value,
    $Res Function(CourseCatalogQueryPayload) then,
  ) = _$CourseCatalogQueryPayloadCopyWithImpl<$Res, CourseCatalogQueryPayload>;
  @useResult
  $Res call({
    String? keyword,
    String? department,
    String? term,
    String? campus,
    String? instructor,
    bool? onlyWithReviews,
    String? sortBy,
    int page,
    int size,
  });
}

/// @nodoc
class _$CourseCatalogQueryPayloadCopyWithImpl<
  $Res,
  $Val extends CourseCatalogQueryPayload
>
    implements $CourseCatalogQueryPayloadCopyWith<$Res> {
  _$CourseCatalogQueryPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseCatalogQueryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? keyword = freezed,
    Object? department = freezed,
    Object? term = freezed,
    Object? campus = freezed,
    Object? instructor = freezed,
    Object? onlyWithReviews = freezed,
    Object? sortBy = freezed,
    Object? page = null,
    Object? size = null,
  }) {
    return _then(
      _value.copyWith(
            keyword: freezed == keyword
                ? _value.keyword
                : keyword // ignore: cast_nullable_to_non_nullable
                      as String?,
            department: freezed == department
                ? _value.department
                : department // ignore: cast_nullable_to_non_nullable
                      as String?,
            term: freezed == term
                ? _value.term
                : term // ignore: cast_nullable_to_non_nullable
                      as String?,
            campus: freezed == campus
                ? _value.campus
                : campus // ignore: cast_nullable_to_non_nullable
                      as String?,
            instructor: freezed == instructor
                ? _value.instructor
                : instructor // ignore: cast_nullable_to_non_nullable
                      as String?,
            onlyWithReviews: freezed == onlyWithReviews
                ? _value.onlyWithReviews
                : onlyWithReviews // ignore: cast_nullable_to_non_nullable
                      as bool?,
            sortBy: freezed == sortBy
                ? _value.sortBy
                : sortBy // ignore: cast_nullable_to_non_nullable
                      as String?,
            page: null == page
                ? _value.page
                : page // ignore: cast_nullable_to_non_nullable
                      as int,
            size: null == size
                ? _value.size
                : size // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseCatalogQueryPayloadImplCopyWith<$Res>
    implements $CourseCatalogQueryPayloadCopyWith<$Res> {
  factory _$$CourseCatalogQueryPayloadImplCopyWith(
    _$CourseCatalogQueryPayloadImpl value,
    $Res Function(_$CourseCatalogQueryPayloadImpl) then,
  ) = __$$CourseCatalogQueryPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String? keyword,
    String? department,
    String? term,
    String? campus,
    String? instructor,
    bool? onlyWithReviews,
    String? sortBy,
    int page,
    int size,
  });
}

/// @nodoc
class __$$CourseCatalogQueryPayloadImplCopyWithImpl<$Res>
    extends
        _$CourseCatalogQueryPayloadCopyWithImpl<
          $Res,
          _$CourseCatalogQueryPayloadImpl
        >
    implements _$$CourseCatalogQueryPayloadImplCopyWith<$Res> {
  __$$CourseCatalogQueryPayloadImplCopyWithImpl(
    _$CourseCatalogQueryPayloadImpl _value,
    $Res Function(_$CourseCatalogQueryPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseCatalogQueryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? keyword = freezed,
    Object? department = freezed,
    Object? term = freezed,
    Object? campus = freezed,
    Object? instructor = freezed,
    Object? onlyWithReviews = freezed,
    Object? sortBy = freezed,
    Object? page = null,
    Object? size = null,
  }) {
    return _then(
      _$CourseCatalogQueryPayloadImpl(
        keyword: freezed == keyword
            ? _value.keyword
            : keyword // ignore: cast_nullable_to_non_nullable
                  as String?,
        department: freezed == department
            ? _value.department
            : department // ignore: cast_nullable_to_non_nullable
                  as String?,
        term: freezed == term
            ? _value.term
            : term // ignore: cast_nullable_to_non_nullable
                  as String?,
        campus: freezed == campus
            ? _value.campus
            : campus // ignore: cast_nullable_to_non_nullable
                  as String?,
        instructor: freezed == instructor
            ? _value.instructor
            : instructor // ignore: cast_nullable_to_non_nullable
                  as String?,
        onlyWithReviews: freezed == onlyWithReviews
            ? _value.onlyWithReviews
            : onlyWithReviews // ignore: cast_nullable_to_non_nullable
                  as bool?,
        sortBy: freezed == sortBy
            ? _value.sortBy
            : sortBy // ignore: cast_nullable_to_non_nullable
                  as String?,
        page: null == page
            ? _value.page
            : page // ignore: cast_nullable_to_non_nullable
                  as int,
        size: null == size
            ? _value.size
            : size // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseCatalogQueryPayloadImpl implements _CourseCatalogQueryPayload {
  const _$CourseCatalogQueryPayloadImpl({
    this.keyword,
    this.department,
    this.term,
    this.campus,
    this.instructor,
    this.onlyWithReviews,
    this.sortBy,
    required this.page,
    required this.size,
  });

  factory _$CourseCatalogQueryPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseCatalogQueryPayloadImplFromJson(json);

  @override
  final String? keyword;
  @override
  final String? department;
  @override
  final String? term;
  @override
  final String? campus;
  @override
  final String? instructor;
  @override
  final bool? onlyWithReviews;
  @override
  final String? sortBy;
  @override
  final int page;
  @override
  final int size;

  @override
  String toString() {
    return 'CourseCatalogQueryPayload(keyword: $keyword, department: $department, term: $term, campus: $campus, instructor: $instructor, onlyWithReviews: $onlyWithReviews, sortBy: $sortBy, page: $page, size: $size)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseCatalogQueryPayloadImpl &&
            (identical(other.keyword, keyword) || other.keyword == keyword) &&
            (identical(other.department, department) ||
                other.department == department) &&
            (identical(other.term, term) || other.term == term) &&
            (identical(other.campus, campus) || other.campus == campus) &&
            (identical(other.instructor, instructor) ||
                other.instructor == instructor) &&
            (identical(other.onlyWithReviews, onlyWithReviews) ||
                other.onlyWithReviews == onlyWithReviews) &&
            (identical(other.sortBy, sortBy) || other.sortBy == sortBy) &&
            (identical(other.page, page) || other.page == page) &&
            (identical(other.size, size) || other.size == size));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(
        runtimeType,
        keyword,
        department,
        term,
        campus,
        instructor,
        onlyWithReviews,
        sortBy,
        page,
        size,
      );

  /// Create a copy of CourseCatalogQueryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseCatalogQueryPayloadImplCopyWith<_$CourseCatalogQueryPayloadImpl>
  get copyWith =>
      __$$CourseCatalogQueryPayloadImplCopyWithImpl<
        _$CourseCatalogQueryPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseCatalogQueryPayloadImplToJson(this);
  }
}

abstract class _CourseCatalogQueryPayload implements CourseCatalogQueryPayload {
  const factory _CourseCatalogQueryPayload({
    final String? keyword,
    final String? department,
    final String? term,
    final String? campus,
    final String? instructor,
    final bool? onlyWithReviews,
    final String? sortBy,
    required final int page,
    required final int size,
  }) = _$CourseCatalogQueryPayloadImpl;

  factory _CourseCatalogQueryPayload.fromJson(Map<String, dynamic> json) =
      _$CourseCatalogQueryPayloadImpl.fromJson;

  @override
  String? get keyword;
  @override
  String? get department;
  @override
  String? get term;
  @override
  String? get campus;
  @override
  String? get instructor;
  @override
  bool? get onlyWithReviews;
  @override
  String? get sortBy;
  @override
  int get page;
  @override
  int get size;

  /// Create a copy of CourseCatalogQueryPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseCatalogQueryPayloadImplCopyWith<_$CourseCatalogQueryPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseOfferingPayload _$CourseOfferingPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CourseOfferingPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseOfferingPayload {
  int get id => throw _privateConstructorUsedError;
  String get termCode => throw _privateConstructorUsedError;
  String? get termName => throw _privateConstructorUsedError;
  String? get campus => throw _privateConstructorUsedError;
  String? get faculty => throw _privateConstructorUsedError;
  List<String>? get instructors => throw _privateConstructorUsedError;
  double? get ratingAvg => throw _privateConstructorUsedError;
  int? get reviewCount => throw _privateConstructorUsedError;

  /// Serializes this CourseOfferingPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseOfferingPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseOfferingPayloadCopyWith<CourseOfferingPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseOfferingPayloadCopyWith<$Res> {
  factory $CourseOfferingPayloadCopyWith(
    CourseOfferingPayload value,
    $Res Function(CourseOfferingPayload) then,
  ) = _$CourseOfferingPayloadCopyWithImpl<$Res, CourseOfferingPayload>;
  @useResult
  $Res call({
    int id,
    String termCode,
    String? termName,
    String? campus,
    String? faculty,
    List<String>? instructors,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class _$CourseOfferingPayloadCopyWithImpl<
  $Res,
  $Val extends CourseOfferingPayload
>
    implements $CourseOfferingPayloadCopyWith<$Res> {
  _$CourseOfferingPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseOfferingPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? termCode = null,
    Object? termName = freezed,
    Object? campus = freezed,
    Object? faculty = freezed,
    Object? instructors = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            termCode: null == termCode
                ? _value.termCode
                : termCode // ignore: cast_nullable_to_non_nullable
                      as String,
            termName: freezed == termName
                ? _value.termName
                : termName // ignore: cast_nullable_to_non_nullable
                      as String?,
            campus: freezed == campus
                ? _value.campus
                : campus // ignore: cast_nullable_to_non_nullable
                      as String?,
            faculty: freezed == faculty
                ? _value.faculty
                : faculty // ignore: cast_nullable_to_non_nullable
                      as String?,
            instructors: freezed == instructors
                ? _value.instructors
                : instructors // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            ratingAvg: freezed == ratingAvg
                ? _value.ratingAvg
                : ratingAvg // ignore: cast_nullable_to_non_nullable
                      as double?,
            reviewCount: freezed == reviewCount
                ? _value.reviewCount
                : reviewCount // ignore: cast_nullable_to_non_nullable
                      as int?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseOfferingPayloadImplCopyWith<$Res>
    implements $CourseOfferingPayloadCopyWith<$Res> {
  factory _$$CourseOfferingPayloadImplCopyWith(
    _$CourseOfferingPayloadImpl value,
    $Res Function(_$CourseOfferingPayloadImpl) then,
  ) = __$$CourseOfferingPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String termCode,
    String? termName,
    String? campus,
    String? faculty,
    List<String>? instructors,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class __$$CourseOfferingPayloadImplCopyWithImpl<$Res>
    extends
        _$CourseOfferingPayloadCopyWithImpl<$Res, _$CourseOfferingPayloadImpl>
    implements _$$CourseOfferingPayloadImplCopyWith<$Res> {
  __$$CourseOfferingPayloadImplCopyWithImpl(
    _$CourseOfferingPayloadImpl _value,
    $Res Function(_$CourseOfferingPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseOfferingPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? termCode = null,
    Object? termName = freezed,
    Object? campus = freezed,
    Object? faculty = freezed,
    Object? instructors = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
  }) {
    return _then(
      _$CourseOfferingPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        termCode: null == termCode
            ? _value.termCode
            : termCode // ignore: cast_nullable_to_non_nullable
                  as String,
        termName: freezed == termName
            ? _value.termName
            : termName // ignore: cast_nullable_to_non_nullable
                  as String?,
        campus: freezed == campus
            ? _value.campus
            : campus // ignore: cast_nullable_to_non_nullable
                  as String?,
        faculty: freezed == faculty
            ? _value.faculty
            : faculty // ignore: cast_nullable_to_non_nullable
                  as String?,
        instructors: freezed == instructors
            ? _value._instructors
            : instructors // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        ratingAvg: freezed == ratingAvg
            ? _value.ratingAvg
            : ratingAvg // ignore: cast_nullable_to_non_nullable
                  as double?,
        reviewCount: freezed == reviewCount
            ? _value.reviewCount
            : reviewCount // ignore: cast_nullable_to_non_nullable
                  as int?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseOfferingPayloadImpl implements _CourseOfferingPayload {
  const _$CourseOfferingPayloadImpl({
    required this.id,
    required this.termCode,
    this.termName,
    this.campus,
    this.faculty,
    final List<String>? instructors,
    this.ratingAvg,
    this.reviewCount,
  }) : _instructors = instructors;

  factory _$CourseOfferingPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseOfferingPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String termCode;
  @override
  final String? termName;
  @override
  final String? campus;
  @override
  final String? faculty;
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
  final double? ratingAvg;
  @override
  final int? reviewCount;

  @override
  String toString() {
    return 'CourseOfferingPayload(id: $id, termCode: $termCode, termName: $termName, campus: $campus, faculty: $faculty, instructors: $instructors, ratingAvg: $ratingAvg, reviewCount: $reviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseOfferingPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.termCode, termCode) ||
                other.termCode == termCode) &&
            (identical(other.termName, termName) ||
                other.termName == termName) &&
            (identical(other.campus, campus) || other.campus == campus) &&
            (identical(other.faculty, faculty) || other.faculty == faculty) &&
            const DeepCollectionEquality().equals(
              other._instructors,
              _instructors,
            ) &&
            (identical(other.ratingAvg, ratingAvg) ||
                other.ratingAvg == ratingAvg) &&
            (identical(other.reviewCount, reviewCount) ||
                other.reviewCount == reviewCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    termCode,
    termName,
    campus,
    faculty,
    const DeepCollectionEquality().hash(_instructors),
    ratingAvg,
    reviewCount,
  );

  /// Create a copy of CourseOfferingPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseOfferingPayloadImplCopyWith<_$CourseOfferingPayloadImpl>
  get copyWith =>
      __$$CourseOfferingPayloadImplCopyWithImpl<_$CourseOfferingPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseOfferingPayloadImplToJson(this);
  }
}

abstract class _CourseOfferingPayload implements CourseOfferingPayload {
  const factory _CourseOfferingPayload({
    required final int id,
    required final String termCode,
    final String? termName,
    final String? campus,
    final String? faculty,
    final List<String>? instructors,
    final double? ratingAvg,
    final int? reviewCount,
  }) = _$CourseOfferingPayloadImpl;

  factory _CourseOfferingPayload.fromJson(Map<String, dynamic> json) =
      _$CourseOfferingPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get termCode;
  @override
  String? get termName;
  @override
  String? get campus;
  @override
  String? get faculty;
  @override
  List<String>? get instructors;
  @override
  double? get ratingAvg;
  @override
  int? get reviewCount;

  /// Create a copy of CourseOfferingPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseOfferingPayloadImplCopyWith<_$CourseOfferingPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseDetailPageProps _$CourseDetailPagePropsFromJson(
  Map<String, dynamic> json,
) {
  return _CourseDetailPageProps.fromJson(json);
}

/// @nodoc
mixin _$CourseDetailPageProps {
  CourseDetailPayload get course => throw _privateConstructorUsedError;

  /// Serializes this CourseDetailPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseDetailPagePropsCopyWith<CourseDetailPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseDetailPagePropsCopyWith<$Res> {
  factory $CourseDetailPagePropsCopyWith(
    CourseDetailPageProps value,
    $Res Function(CourseDetailPageProps) then,
  ) = _$CourseDetailPagePropsCopyWithImpl<$Res, CourseDetailPageProps>;
  @useResult
  $Res call({CourseDetailPayload course});

  $CourseDetailPayloadCopyWith<$Res> get course;
}

/// @nodoc
class _$CourseDetailPagePropsCopyWithImpl<
  $Res,
  $Val extends CourseDetailPageProps
>
    implements $CourseDetailPagePropsCopyWith<$Res> {
  _$CourseDetailPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? course = null}) {
    return _then(
      _value.copyWith(
            course: null == course
                ? _value.course
                : course // ignore: cast_nullable_to_non_nullable
                      as CourseDetailPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $CourseDetailPayloadCopyWith<$Res> get course {
    return $CourseDetailPayloadCopyWith<$Res>(_value.course, (value) {
      return _then(_value.copyWith(course: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$CourseDetailPagePropsImplCopyWith<$Res>
    implements $CourseDetailPagePropsCopyWith<$Res> {
  factory _$$CourseDetailPagePropsImplCopyWith(
    _$CourseDetailPagePropsImpl value,
    $Res Function(_$CourseDetailPagePropsImpl) then,
  ) = __$$CourseDetailPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({CourseDetailPayload course});

  @override
  $CourseDetailPayloadCopyWith<$Res> get course;
}

/// @nodoc
class __$$CourseDetailPagePropsImplCopyWithImpl<$Res>
    extends
        _$CourseDetailPagePropsCopyWithImpl<$Res, _$CourseDetailPagePropsImpl>
    implements _$$CourseDetailPagePropsImplCopyWith<$Res> {
  __$$CourseDetailPagePropsImplCopyWithImpl(
    _$CourseDetailPagePropsImpl _value,
    $Res Function(_$CourseDetailPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? course = null}) {
    return _then(
      _$CourseDetailPagePropsImpl(
        course: null == course
            ? _value.course
            : course // ignore: cast_nullable_to_non_nullable
                  as CourseDetailPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseDetailPagePropsImpl implements _CourseDetailPageProps {
  const _$CourseDetailPagePropsImpl({required this.course});

  factory _$CourseDetailPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseDetailPagePropsImplFromJson(json);

  @override
  final CourseDetailPayload course;

  @override
  String toString() {
    return 'CourseDetailPageProps(course: $course)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseDetailPagePropsImpl &&
            (identical(other.course, course) || other.course == course));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, course);

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseDetailPagePropsImplCopyWith<_$CourseDetailPagePropsImpl>
  get copyWith =>
      __$$CourseDetailPagePropsImplCopyWithImpl<_$CourseDetailPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseDetailPagePropsImplToJson(this);
  }
}

abstract class _CourseDetailPageProps implements CourseDetailPageProps {
  const factory _CourseDetailPageProps({
    required final CourseDetailPayload course,
  }) = _$CourseDetailPagePropsImpl;

  factory _CourseDetailPageProps.fromJson(Map<String, dynamic> json) =
      _$CourseDetailPagePropsImpl.fromJson;

  @override
  CourseDetailPayload get course;

  /// Create a copy of CourseDetailPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseDetailPagePropsImplCopyWith<_$CourseDetailPagePropsImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseDetailPayload _$CourseDetailPayloadFromJson(Map<String, dynamic> json) {
  return _CourseDetailPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseDetailPayload {
  int get id => throw _privateConstructorUsedError;
  String get primaryCode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get department => throw _privateConstructorUsedError;
  int get creditX10 => throw _privateConstructorUsedError;
  List<String>? get aliases => throw _privateConstructorUsedError;
  List<CourseOfferingPayload>? get offerings =>
      throw _privateConstructorUsedError;
  double? get ratingAvg => throw _privateConstructorUsedError;
  int? get reviewCount => throw _privateConstructorUsedError;
  List<int>? get ratingDistribution => throw _privateConstructorUsedError;

  /// Serializes this CourseDetailPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseDetailPayloadCopyWith<CourseDetailPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseDetailPayloadCopyWith<$Res> {
  factory $CourseDetailPayloadCopyWith(
    CourseDetailPayload value,
    $Res Function(CourseDetailPayload) then,
  ) = _$CourseDetailPayloadCopyWithImpl<$Res, CourseDetailPayload>;
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    int creditX10,
    List<String>? aliases,
    List<CourseOfferingPayload>? offerings,
    double? ratingAvg,
    int? reviewCount,
    List<int>? ratingDistribution,
  });
}

/// @nodoc
class _$CourseDetailPayloadCopyWithImpl<$Res, $Val extends CourseDetailPayload>
    implements $CourseDetailPayloadCopyWith<$Res> {
  _$CourseDetailPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? creditX10 = null,
    Object? aliases = freezed,
    Object? offerings = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
    Object? ratingDistribution = freezed,
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
            creditX10: null == creditX10
                ? _value.creditX10
                : creditX10 // ignore: cast_nullable_to_non_nullable
                      as int,
            aliases: freezed == aliases
                ? _value.aliases
                : aliases // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            offerings: freezed == offerings
                ? _value.offerings
                : offerings // ignore: cast_nullable_to_non_nullable
                      as List<CourseOfferingPayload>?,
            ratingAvg: freezed == ratingAvg
                ? _value.ratingAvg
                : ratingAvg // ignore: cast_nullable_to_non_nullable
                      as double?,
            reviewCount: freezed == reviewCount
                ? _value.reviewCount
                : reviewCount // ignore: cast_nullable_to_non_nullable
                      as int?,
            ratingDistribution: freezed == ratingDistribution
                ? _value.ratingDistribution
                : ratingDistribution // ignore: cast_nullable_to_non_nullable
                      as List<int>?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CourseDetailPayloadImplCopyWith<$Res>
    implements $CourseDetailPayloadCopyWith<$Res> {
  factory _$$CourseDetailPayloadImplCopyWith(
    _$CourseDetailPayloadImpl value,
    $Res Function(_$CourseDetailPayloadImpl) then,
  ) = __$$CourseDetailPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    int creditX10,
    List<String>? aliases,
    List<CourseOfferingPayload>? offerings,
    double? ratingAvg,
    int? reviewCount,
    List<int>? ratingDistribution,
  });
}

/// @nodoc
class __$$CourseDetailPayloadImplCopyWithImpl<$Res>
    extends _$CourseDetailPayloadCopyWithImpl<$Res, _$CourseDetailPayloadImpl>
    implements _$$CourseDetailPayloadImplCopyWith<$Res> {
  __$$CourseDetailPayloadImplCopyWithImpl(
    _$CourseDetailPayloadImpl _value,
    $Res Function(_$CourseDetailPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? primaryCode = null,
    Object? name = null,
    Object? department = null,
    Object? creditX10 = null,
    Object? aliases = freezed,
    Object? offerings = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
    Object? ratingDistribution = freezed,
  }) {
    return _then(
      _$CourseDetailPayloadImpl(
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
        creditX10: null == creditX10
            ? _value.creditX10
            : creditX10 // ignore: cast_nullable_to_non_nullable
                  as int,
        aliases: freezed == aliases
            ? _value._aliases
            : aliases // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        offerings: freezed == offerings
            ? _value._offerings
            : offerings // ignore: cast_nullable_to_non_nullable
                  as List<CourseOfferingPayload>?,
        ratingAvg: freezed == ratingAvg
            ? _value.ratingAvg
            : ratingAvg // ignore: cast_nullable_to_non_nullable
                  as double?,
        reviewCount: freezed == reviewCount
            ? _value.reviewCount
            : reviewCount // ignore: cast_nullable_to_non_nullable
                  as int?,
        ratingDistribution: freezed == ratingDistribution
            ? _value._ratingDistribution
            : ratingDistribution // ignore: cast_nullable_to_non_nullable
                  as List<int>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CourseDetailPayloadImpl implements _CourseDetailPayload {
  const _$CourseDetailPayloadImpl({
    required this.id,
    required this.primaryCode,
    required this.name,
    required this.department,
    required this.creditX10,
    final List<String>? aliases,
    final List<CourseOfferingPayload>? offerings,
    this.ratingAvg,
    this.reviewCount,
    final List<int>? ratingDistribution,
  }) : _aliases = aliases,
       _offerings = offerings,
       _ratingDistribution = ratingDistribution;

  factory _$CourseDetailPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseDetailPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String primaryCode;
  @override
  final String name;
  @override
  final String department;
  @override
  final int creditX10;
  final List<String>? _aliases;
  @override
  List<String>? get aliases {
    final value = _aliases;
    if (value == null) return null;
    if (_aliases is EqualUnmodifiableListView) return _aliases;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<CourseOfferingPayload>? _offerings;
  @override
  List<CourseOfferingPayload>? get offerings {
    final value = _offerings;
    if (value == null) return null;
    if (_offerings is EqualUnmodifiableListView) return _offerings;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final double? ratingAvg;
  @override
  final int? reviewCount;
  final List<int>? _ratingDistribution;
  @override
  List<int>? get ratingDistribution {
    final value = _ratingDistribution;
    if (value == null) return null;
    if (_ratingDistribution is EqualUnmodifiableListView)
      return _ratingDistribution;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'CourseDetailPayload(id: $id, primaryCode: $primaryCode, name: $name, department: $department, creditX10: $creditX10, aliases: $aliases, offerings: $offerings, ratingAvg: $ratingAvg, reviewCount: $reviewCount, ratingDistribution: $ratingDistribution)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseDetailPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.primaryCode, primaryCode) ||
                other.primaryCode == primaryCode) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.department, department) ||
                other.department == department) &&
            (identical(other.creditX10, creditX10) ||
                other.creditX10 == creditX10) &&
            const DeepCollectionEquality().equals(other._aliases, _aliases) &&
            const DeepCollectionEquality().equals(
              other._offerings,
              _offerings,
            ) &&
            (identical(other.ratingAvg, ratingAvg) ||
                other.ratingAvg == ratingAvg) &&
            (identical(other.reviewCount, reviewCount) ||
                other.reviewCount == reviewCount) &&
            const DeepCollectionEquality().equals(
              other._ratingDistribution,
              _ratingDistribution,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    primaryCode,
    name,
    department,
    creditX10,
    const DeepCollectionEquality().hash(_aliases),
    const DeepCollectionEquality().hash(_offerings),
    ratingAvg,
    reviewCount,
    const DeepCollectionEquality().hash(_ratingDistribution),
  );

  /// Create a copy of CourseDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseDetailPayloadImplCopyWith<_$CourseDetailPayloadImpl> get copyWith =>
      __$$CourseDetailPayloadImplCopyWithImpl<_$CourseDetailPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseDetailPayloadImplToJson(this);
  }
}

abstract class _CourseDetailPayload implements CourseDetailPayload {
  const factory _CourseDetailPayload({
    required final int id,
    required final String primaryCode,
    required final String name,
    required final String department,
    required final int creditX10,
    final List<String>? aliases,
    final List<CourseOfferingPayload>? offerings,
    final double? ratingAvg,
    final int? reviewCount,
    final List<int>? ratingDistribution,
  }) = _$CourseDetailPayloadImpl;

  factory _CourseDetailPayload.fromJson(Map<String, dynamic> json) =
      _$CourseDetailPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get primaryCode;
  @override
  String get name;
  @override
  String get department;
  @override
  int get creditX10;
  @override
  List<String>? get aliases;
  @override
  List<CourseOfferingPayload>? get offerings;
  @override
  double? get ratingAvg;
  @override
  int? get reviewCount;
  @override
  List<int>? get ratingDistribution;

  /// Create a copy of CourseDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseDetailPayloadImplCopyWith<_$CourseDetailPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
