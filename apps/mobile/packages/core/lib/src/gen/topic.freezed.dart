// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'topic.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

UserBriefPayload _$UserBriefPayloadFromJson(Map<String, dynamic> json) {
  return _UserBriefPayload.fromJson(json);
}

/// @nodoc
mixin _$UserBriefPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String? get nickname => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  UserBadgePayload? get wornBadge => throw _privateConstructorUsedError;

  /// Serializes this UserBriefPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserBriefPayloadCopyWith<UserBriefPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserBriefPayloadCopyWith<$Res> {
  factory $UserBriefPayloadCopyWith(
    UserBriefPayload value,
    $Res Function(UserBriefPayload) then,
  ) = _$UserBriefPayloadCopyWithImpl<$Res, UserBriefPayload>;
  @useResult
  $Res call({
    int id,
    String username,
    String? nickname,
    String avatarUrl,
    UserBadgePayload? wornBadge,
  });

  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class _$UserBriefPayloadCopyWithImpl<$Res, $Val extends UserBriefPayload>
    implements $UserBriefPayloadCopyWith<$Res> {
  _$UserBriefPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = freezed,
    Object? avatarUrl = null,
    Object? wornBadge = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            username: null == username
                ? _value.username
                : username // ignore: cast_nullable_to_non_nullable
                      as String,
            nickname: freezed == nickname
                ? _value.nickname
                : nickname // ignore: cast_nullable_to_non_nullable
                      as String?,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            wornBadge: freezed == wornBadge
                ? _value.wornBadge
                : wornBadge // ignore: cast_nullable_to_non_nullable
                      as UserBadgePayload?,
          )
          as $Val,
    );
  }

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBadgePayloadCopyWith<$Res>? get wornBadge {
    if (_value.wornBadge == null) {
      return null;
    }

    return $UserBadgePayloadCopyWith<$Res>(_value.wornBadge!, (value) {
      return _then(_value.copyWith(wornBadge: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$UserBriefPayloadImplCopyWith<$Res>
    implements $UserBriefPayloadCopyWith<$Res> {
  factory _$$UserBriefPayloadImplCopyWith(
    _$UserBriefPayloadImpl value,
    $Res Function(_$UserBriefPayloadImpl) then,
  ) = __$$UserBriefPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String username,
    String? nickname,
    String avatarUrl,
    UserBadgePayload? wornBadge,
  });

  @override
  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class __$$UserBriefPayloadImplCopyWithImpl<$Res>
    extends _$UserBriefPayloadCopyWithImpl<$Res, _$UserBriefPayloadImpl>
    implements _$$UserBriefPayloadImplCopyWith<$Res> {
  __$$UserBriefPayloadImplCopyWithImpl(
    _$UserBriefPayloadImpl _value,
    $Res Function(_$UserBriefPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = freezed,
    Object? avatarUrl = null,
    Object? wornBadge = freezed,
  }) {
    return _then(
      _$UserBriefPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String,
        nickname: freezed == nickname
            ? _value.nickname
            : nickname // ignore: cast_nullable_to_non_nullable
                  as String?,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        wornBadge: freezed == wornBadge
            ? _value.wornBadge
            : wornBadge // ignore: cast_nullable_to_non_nullable
                  as UserBadgePayload?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserBriefPayloadImpl implements _UserBriefPayload {
  const _$UserBriefPayloadImpl({
    required this.id,
    required this.username,
    this.nickname,
    required this.avatarUrl,
    this.wornBadge,
  });

  factory _$UserBriefPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserBriefPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String? nickname;
  @override
  final String avatarUrl;
  @override
  final UserBadgePayload? wornBadge;

  @override
  String toString() {
    return 'UserBriefPayload(id: $id, username: $username, nickname: $nickname, avatarUrl: $avatarUrl, wornBadge: $wornBadge)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserBriefPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.wornBadge, wornBadge) ||
                other.wornBadge == wornBadge));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, username, nickname, avatarUrl, wornBadge);

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserBriefPayloadImplCopyWith<_$UserBriefPayloadImpl> get copyWith =>
      __$$UserBriefPayloadImplCopyWithImpl<_$UserBriefPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserBriefPayloadImplToJson(this);
  }
}

abstract class _UserBriefPayload implements UserBriefPayload {
  const factory _UserBriefPayload({
    required final int id,
    required final String username,
    final String? nickname,
    required final String avatarUrl,
    final UserBadgePayload? wornBadge,
  }) = _$UserBriefPayloadImpl;

  factory _UserBriefPayload.fromJson(Map<String, dynamic> json) =
      _$UserBriefPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String? get nickname;
  @override
  String get avatarUrl;
  @override
  UserBadgePayload? get wornBadge;

  /// Create a copy of UserBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserBriefPayloadImplCopyWith<_$UserBriefPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

BadgePayload _$BadgePayloadFromJson(Map<String, dynamic> json) {
  return _BadgePayload.fromJson(json);
}

/// @nodoc
mixin _$BadgePayload {
  String get code => throw _privateConstructorUsedError;
  String get type => throw _privateConstructorUsedError;
  String get grantMode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get iconType => throw _privateConstructorUsedError;
  String get iconKey => throw _privateConstructorUsedError;
  String get iconUrl => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;
  String get level => throw _privateConstructorUsedError;
  bool get isEnabled => throw _privateConstructorUsedError;
  bool get isWearable => throw _privateConstructorUsedError;
  int get sortOrder => throw _privateConstructorUsedError;

  /// Serializes this BadgePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of BadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $BadgePayloadCopyWith<BadgePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $BadgePayloadCopyWith<$Res> {
  factory $BadgePayloadCopyWith(
    BadgePayload value,
    $Res Function(BadgePayload) then,
  ) = _$BadgePayloadCopyWithImpl<$Res, BadgePayload>;
  @useResult
  $Res call({
    String code,
    String type,
    String grantMode,
    String name,
    String description,
    String iconType,
    String iconKey,
    String iconUrl,
    String color,
    String level,
    bool isEnabled,
    bool isWearable,
    int sortOrder,
  });
}

/// @nodoc
class _$BadgePayloadCopyWithImpl<$Res, $Val extends BadgePayload>
    implements $BadgePayloadCopyWith<$Res> {
  _$BadgePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of BadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? type = null,
    Object? grantMode = null,
    Object? name = null,
    Object? description = null,
    Object? iconType = null,
    Object? iconKey = null,
    Object? iconUrl = null,
    Object? color = null,
    Object? level = null,
    Object? isEnabled = null,
    Object? isWearable = null,
    Object? sortOrder = null,
  }) {
    return _then(
      _value.copyWith(
            code: null == code
                ? _value.code
                : code // ignore: cast_nullable_to_non_nullable
                      as String,
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            grantMode: null == grantMode
                ? _value.grantMode
                : grantMode // ignore: cast_nullable_to_non_nullable
                      as String,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            iconType: null == iconType
                ? _value.iconType
                : iconType // ignore: cast_nullable_to_non_nullable
                      as String,
            iconKey: null == iconKey
                ? _value.iconKey
                : iconKey // ignore: cast_nullable_to_non_nullable
                      as String,
            iconUrl: null == iconUrl
                ? _value.iconUrl
                : iconUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
            level: null == level
                ? _value.level
                : level // ignore: cast_nullable_to_non_nullable
                      as String,
            isEnabled: null == isEnabled
                ? _value.isEnabled
                : isEnabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            isWearable: null == isWearable
                ? _value.isWearable
                : isWearable // ignore: cast_nullable_to_non_nullable
                      as bool,
            sortOrder: null == sortOrder
                ? _value.sortOrder
                : sortOrder // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$BadgePayloadImplCopyWith<$Res>
    implements $BadgePayloadCopyWith<$Res> {
  factory _$$BadgePayloadImplCopyWith(
    _$BadgePayloadImpl value,
    $Res Function(_$BadgePayloadImpl) then,
  ) = __$$BadgePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String code,
    String type,
    String grantMode,
    String name,
    String description,
    String iconType,
    String iconKey,
    String iconUrl,
    String color,
    String level,
    bool isEnabled,
    bool isWearable,
    int sortOrder,
  });
}

/// @nodoc
class __$$BadgePayloadImplCopyWithImpl<$Res>
    extends _$BadgePayloadCopyWithImpl<$Res, _$BadgePayloadImpl>
    implements _$$BadgePayloadImplCopyWith<$Res> {
  __$$BadgePayloadImplCopyWithImpl(
    _$BadgePayloadImpl _value,
    $Res Function(_$BadgePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of BadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? type = null,
    Object? grantMode = null,
    Object? name = null,
    Object? description = null,
    Object? iconType = null,
    Object? iconKey = null,
    Object? iconUrl = null,
    Object? color = null,
    Object? level = null,
    Object? isEnabled = null,
    Object? isWearable = null,
    Object? sortOrder = null,
  }) {
    return _then(
      _$BadgePayloadImpl(
        code: null == code
            ? _value.code
            : code // ignore: cast_nullable_to_non_nullable
                  as String,
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        grantMode: null == grantMode
            ? _value.grantMode
            : grantMode // ignore: cast_nullable_to_non_nullable
                  as String,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        iconType: null == iconType
            ? _value.iconType
            : iconType // ignore: cast_nullable_to_non_nullable
                  as String,
        iconKey: null == iconKey
            ? _value.iconKey
            : iconKey // ignore: cast_nullable_to_non_nullable
                  as String,
        iconUrl: null == iconUrl
            ? _value.iconUrl
            : iconUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
        level: null == level
            ? _value.level
            : level // ignore: cast_nullable_to_non_nullable
                  as String,
        isEnabled: null == isEnabled
            ? _value.isEnabled
            : isEnabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        isWearable: null == isWearable
            ? _value.isWearable
            : isWearable // ignore: cast_nullable_to_non_nullable
                  as bool,
        sortOrder: null == sortOrder
            ? _value.sortOrder
            : sortOrder // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$BadgePayloadImpl implements _BadgePayload {
  const _$BadgePayloadImpl({
    required this.code,
    required this.type,
    required this.grantMode,
    required this.name,
    required this.description,
    required this.iconType,
    required this.iconKey,
    required this.iconUrl,
    required this.color,
    required this.level,
    required this.isEnabled,
    required this.isWearable,
    required this.sortOrder,
  });

  factory _$BadgePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$BadgePayloadImplFromJson(json);

  @override
  final String code;
  @override
  final String type;
  @override
  final String grantMode;
  @override
  final String name;
  @override
  final String description;
  @override
  final String iconType;
  @override
  final String iconKey;
  @override
  final String iconUrl;
  @override
  final String color;
  @override
  final String level;
  @override
  final bool isEnabled;
  @override
  final bool isWearable;
  @override
  final int sortOrder;

  @override
  String toString() {
    return 'BadgePayload(code: $code, type: $type, grantMode: $grantMode, name: $name, description: $description, iconType: $iconType, iconKey: $iconKey, iconUrl: $iconUrl, color: $color, level: $level, isEnabled: $isEnabled, isWearable: $isWearable, sortOrder: $sortOrder)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$BadgePayloadImpl &&
            (identical(other.code, code) || other.code == code) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.grantMode, grantMode) ||
                other.grantMode == grantMode) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.iconType, iconType) ||
                other.iconType == iconType) &&
            (identical(other.iconKey, iconKey) || other.iconKey == iconKey) &&
            (identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl) &&
            (identical(other.color, color) || other.color == color) &&
            (identical(other.level, level) || other.level == level) &&
            (identical(other.isEnabled, isEnabled) ||
                other.isEnabled == isEnabled) &&
            (identical(other.isWearable, isWearable) ||
                other.isWearable == isWearable) &&
            (identical(other.sortOrder, sortOrder) ||
                other.sortOrder == sortOrder));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    code,
    type,
    grantMode,
    name,
    description,
    iconType,
    iconKey,
    iconUrl,
    color,
    level,
    isEnabled,
    isWearable,
    sortOrder,
  );

  /// Create a copy of BadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$BadgePayloadImplCopyWith<_$BadgePayloadImpl> get copyWith =>
      __$$BadgePayloadImplCopyWithImpl<_$BadgePayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$BadgePayloadImplToJson(this);
  }
}

abstract class _BadgePayload implements BadgePayload {
  const factory _BadgePayload({
    required final String code,
    required final String type,
    required final String grantMode,
    required final String name,
    required final String description,
    required final String iconType,
    required final String iconKey,
    required final String iconUrl,
    required final String color,
    required final String level,
    required final bool isEnabled,
    required final bool isWearable,
    required final int sortOrder,
  }) = _$BadgePayloadImpl;

  factory _BadgePayload.fromJson(Map<String, dynamic> json) =
      _$BadgePayloadImpl.fromJson;

  @override
  String get code;
  @override
  String get type;
  @override
  String get grantMode;
  @override
  String get name;
  @override
  String get description;
  @override
  String get iconType;
  @override
  String get iconKey;
  @override
  String get iconUrl;
  @override
  String get color;
  @override
  String get level;
  @override
  bool get isEnabled;
  @override
  bool get isWearable;
  @override
  int get sortOrder;

  /// Create a copy of BadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$BadgePayloadImplCopyWith<_$BadgePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserBadgePayload _$UserBadgePayloadFromJson(Map<String, dynamic> json) {
  return _UserBadgePayload.fromJson(json);
}

/// @nodoc
mixin _$UserBadgePayload {
  String get code => throw _privateConstructorUsedError;
  String get type => throw _privateConstructorUsedError;
  String get grantMode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get iconType => throw _privateConstructorUsedError;
  String get iconKey => throw _privateConstructorUsedError;
  String get iconUrl => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;
  String get level => throw _privateConstructorUsedError;
  bool get isEnabled => throw _privateConstructorUsedError;
  bool get isWearable => throw _privateConstructorUsedError;
  int get sortOrder => throw _privateConstructorUsedError;
  String get source => throw _privateConstructorUsedError;
  String get reason => throw _privateConstructorUsedError;
  String get grantedAt => throw _privateConstructorUsedError;

  /// Serializes this UserBadgePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserBadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserBadgePayloadCopyWith<UserBadgePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserBadgePayloadCopyWith<$Res> {
  factory $UserBadgePayloadCopyWith(
    UserBadgePayload value,
    $Res Function(UserBadgePayload) then,
  ) = _$UserBadgePayloadCopyWithImpl<$Res, UserBadgePayload>;
  @useResult
  $Res call({
    String code,
    String type,
    String grantMode,
    String name,
    String description,
    String iconType,
    String iconKey,
    String iconUrl,
    String color,
    String level,
    bool isEnabled,
    bool isWearable,
    int sortOrder,
    String source,
    String reason,
    String grantedAt,
  });
}

/// @nodoc
class _$UserBadgePayloadCopyWithImpl<$Res, $Val extends UserBadgePayload>
    implements $UserBadgePayloadCopyWith<$Res> {
  _$UserBadgePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserBadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? type = null,
    Object? grantMode = null,
    Object? name = null,
    Object? description = null,
    Object? iconType = null,
    Object? iconKey = null,
    Object? iconUrl = null,
    Object? color = null,
    Object? level = null,
    Object? isEnabled = null,
    Object? isWearable = null,
    Object? sortOrder = null,
    Object? source = null,
    Object? reason = null,
    Object? grantedAt = null,
  }) {
    return _then(
      _value.copyWith(
            code: null == code
                ? _value.code
                : code // ignore: cast_nullable_to_non_nullable
                      as String,
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            grantMode: null == grantMode
                ? _value.grantMode
                : grantMode // ignore: cast_nullable_to_non_nullable
                      as String,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            iconType: null == iconType
                ? _value.iconType
                : iconType // ignore: cast_nullable_to_non_nullable
                      as String,
            iconKey: null == iconKey
                ? _value.iconKey
                : iconKey // ignore: cast_nullable_to_non_nullable
                      as String,
            iconUrl: null == iconUrl
                ? _value.iconUrl
                : iconUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
            level: null == level
                ? _value.level
                : level // ignore: cast_nullable_to_non_nullable
                      as String,
            isEnabled: null == isEnabled
                ? _value.isEnabled
                : isEnabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            isWearable: null == isWearable
                ? _value.isWearable
                : isWearable // ignore: cast_nullable_to_non_nullable
                      as bool,
            sortOrder: null == sortOrder
                ? _value.sortOrder
                : sortOrder // ignore: cast_nullable_to_non_nullable
                      as int,
            source: null == source
                ? _value.source
                : source // ignore: cast_nullable_to_non_nullable
                      as String,
            reason: null == reason
                ? _value.reason
                : reason // ignore: cast_nullable_to_non_nullable
                      as String,
            grantedAt: null == grantedAt
                ? _value.grantedAt
                : grantedAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserBadgePayloadImplCopyWith<$Res>
    implements $UserBadgePayloadCopyWith<$Res> {
  factory _$$UserBadgePayloadImplCopyWith(
    _$UserBadgePayloadImpl value,
    $Res Function(_$UserBadgePayloadImpl) then,
  ) = __$$UserBadgePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String code,
    String type,
    String grantMode,
    String name,
    String description,
    String iconType,
    String iconKey,
    String iconUrl,
    String color,
    String level,
    bool isEnabled,
    bool isWearable,
    int sortOrder,
    String source,
    String reason,
    String grantedAt,
  });
}

/// @nodoc
class __$$UserBadgePayloadImplCopyWithImpl<$Res>
    extends _$UserBadgePayloadCopyWithImpl<$Res, _$UserBadgePayloadImpl>
    implements _$$UserBadgePayloadImplCopyWith<$Res> {
  __$$UserBadgePayloadImplCopyWithImpl(
    _$UserBadgePayloadImpl _value,
    $Res Function(_$UserBadgePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserBadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? type = null,
    Object? grantMode = null,
    Object? name = null,
    Object? description = null,
    Object? iconType = null,
    Object? iconKey = null,
    Object? iconUrl = null,
    Object? color = null,
    Object? level = null,
    Object? isEnabled = null,
    Object? isWearable = null,
    Object? sortOrder = null,
    Object? source = null,
    Object? reason = null,
    Object? grantedAt = null,
  }) {
    return _then(
      _$UserBadgePayloadImpl(
        code: null == code
            ? _value.code
            : code // ignore: cast_nullable_to_non_nullable
                  as String,
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        grantMode: null == grantMode
            ? _value.grantMode
            : grantMode // ignore: cast_nullable_to_non_nullable
                  as String,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        iconType: null == iconType
            ? _value.iconType
            : iconType // ignore: cast_nullable_to_non_nullable
                  as String,
        iconKey: null == iconKey
            ? _value.iconKey
            : iconKey // ignore: cast_nullable_to_non_nullable
                  as String,
        iconUrl: null == iconUrl
            ? _value.iconUrl
            : iconUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
        level: null == level
            ? _value.level
            : level // ignore: cast_nullable_to_non_nullable
                  as String,
        isEnabled: null == isEnabled
            ? _value.isEnabled
            : isEnabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        isWearable: null == isWearable
            ? _value.isWearable
            : isWearable // ignore: cast_nullable_to_non_nullable
                  as bool,
        sortOrder: null == sortOrder
            ? _value.sortOrder
            : sortOrder // ignore: cast_nullable_to_non_nullable
                  as int,
        source: null == source
            ? _value.source
            : source // ignore: cast_nullable_to_non_nullable
                  as String,
        reason: null == reason
            ? _value.reason
            : reason // ignore: cast_nullable_to_non_nullable
                  as String,
        grantedAt: null == grantedAt
            ? _value.grantedAt
            : grantedAt // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserBadgePayloadImpl implements _UserBadgePayload {
  const _$UserBadgePayloadImpl({
    required this.code,
    required this.type,
    required this.grantMode,
    required this.name,
    required this.description,
    required this.iconType,
    required this.iconKey,
    required this.iconUrl,
    required this.color,
    required this.level,
    required this.isEnabled,
    required this.isWearable,
    required this.sortOrder,
    required this.source,
    required this.reason,
    required this.grantedAt,
  });

  factory _$UserBadgePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserBadgePayloadImplFromJson(json);

  @override
  final String code;
  @override
  final String type;
  @override
  final String grantMode;
  @override
  final String name;
  @override
  final String description;
  @override
  final String iconType;
  @override
  final String iconKey;
  @override
  final String iconUrl;
  @override
  final String color;
  @override
  final String level;
  @override
  final bool isEnabled;
  @override
  final bool isWearable;
  @override
  final int sortOrder;
  @override
  final String source;
  @override
  final String reason;
  @override
  final String grantedAt;

  @override
  String toString() {
    return 'UserBadgePayload(code: $code, type: $type, grantMode: $grantMode, name: $name, description: $description, iconType: $iconType, iconKey: $iconKey, iconUrl: $iconUrl, color: $color, level: $level, isEnabled: $isEnabled, isWearable: $isWearable, sortOrder: $sortOrder, source: $source, reason: $reason, grantedAt: $grantedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserBadgePayloadImpl &&
            (identical(other.code, code) || other.code == code) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.grantMode, grantMode) ||
                other.grantMode == grantMode) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.iconType, iconType) ||
                other.iconType == iconType) &&
            (identical(other.iconKey, iconKey) || other.iconKey == iconKey) &&
            (identical(other.iconUrl, iconUrl) || other.iconUrl == iconUrl) &&
            (identical(other.color, color) || other.color == color) &&
            (identical(other.level, level) || other.level == level) &&
            (identical(other.isEnabled, isEnabled) ||
                other.isEnabled == isEnabled) &&
            (identical(other.isWearable, isWearable) ||
                other.isWearable == isWearable) &&
            (identical(other.sortOrder, sortOrder) ||
                other.sortOrder == sortOrder) &&
            (identical(other.source, source) || other.source == source) &&
            (identical(other.reason, reason) || other.reason == reason) &&
            (identical(other.grantedAt, grantedAt) ||
                other.grantedAt == grantedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    code,
    type,
    grantMode,
    name,
    description,
    iconType,
    iconKey,
    iconUrl,
    color,
    level,
    isEnabled,
    isWearable,
    sortOrder,
    source,
    reason,
    grantedAt,
  );

  /// Create a copy of UserBadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserBadgePayloadImplCopyWith<_$UserBadgePayloadImpl> get copyWith =>
      __$$UserBadgePayloadImplCopyWithImpl<_$UserBadgePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserBadgePayloadImplToJson(this);
  }
}

abstract class _UserBadgePayload implements UserBadgePayload {
  const factory _UserBadgePayload({
    required final String code,
    required final String type,
    required final String grantMode,
    required final String name,
    required final String description,
    required final String iconType,
    required final String iconKey,
    required final String iconUrl,
    required final String color,
    required final String level,
    required final bool isEnabled,
    required final bool isWearable,
    required final int sortOrder,
    required final String source,
    required final String reason,
    required final String grantedAt,
  }) = _$UserBadgePayloadImpl;

  factory _UserBadgePayload.fromJson(Map<String, dynamic> json) =
      _$UserBadgePayloadImpl.fromJson;

  @override
  String get code;
  @override
  String get type;
  @override
  String get grantMode;
  @override
  String get name;
  @override
  String get description;
  @override
  String get iconType;
  @override
  String get iconKey;
  @override
  String get iconUrl;
  @override
  String get color;
  @override
  String get level;
  @override
  bool get isEnabled;
  @override
  bool get isWearable;
  @override
  int get sortOrder;
  @override
  String get source;
  @override
  String get reason;
  @override
  String get grantedAt;

  /// Create a copy of UserBadgePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserBadgePayloadImplCopyWith<_$UserBadgePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CategoryBriefPayload _$CategoryBriefPayloadFromJson(Map<String, dynamic> json) {
  return _CategoryBriefPayload.fromJson(json);
}

/// @nodoc
mixin _$CategoryBriefPayload {
  int get id => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;

  /// Serializes this CategoryBriefPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CategoryBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CategoryBriefPayloadCopyWith<CategoryBriefPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CategoryBriefPayloadCopyWith<$Res> {
  factory $CategoryBriefPayloadCopyWith(
    CategoryBriefPayload value,
    $Res Function(CategoryBriefPayload) then,
  ) = _$CategoryBriefPayloadCopyWithImpl<$Res, CategoryBriefPayload>;
  @useResult
  $Res call({int id, String name, String url, String color});
}

/// @nodoc
class _$CategoryBriefPayloadCopyWithImpl<
  $Res,
  $Val extends CategoryBriefPayload
>
    implements $CategoryBriefPayloadCopyWith<$Res> {
  _$CategoryBriefPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CategoryBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? url = null,
    Object? color = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CategoryBriefPayloadImplCopyWith<$Res>
    implements $CategoryBriefPayloadCopyWith<$Res> {
  factory _$$CategoryBriefPayloadImplCopyWith(
    _$CategoryBriefPayloadImpl value,
    $Res Function(_$CategoryBriefPayloadImpl) then,
  ) = __$$CategoryBriefPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String name, String url, String color});
}

/// @nodoc
class __$$CategoryBriefPayloadImplCopyWithImpl<$Res>
    extends _$CategoryBriefPayloadCopyWithImpl<$Res, _$CategoryBriefPayloadImpl>
    implements _$$CategoryBriefPayloadImplCopyWith<$Res> {
  __$$CategoryBriefPayloadImplCopyWithImpl(
    _$CategoryBriefPayloadImpl _value,
    $Res Function(_$CategoryBriefPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CategoryBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? url = null,
    Object? color = null,
  }) {
    return _then(
      _$CategoryBriefPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CategoryBriefPayloadImpl implements _CategoryBriefPayload {
  const _$CategoryBriefPayloadImpl({
    required this.id,
    required this.name,
    required this.url,
    required this.color,
  });

  factory _$CategoryBriefPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CategoryBriefPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String name;
  @override
  final String url;
  @override
  final String color;

  @override
  String toString() {
    return 'CategoryBriefPayload(id: $id, name: $name, url: $url, color: $color)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CategoryBriefPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.color, color) || other.color == color));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, name, url, color);

  /// Create a copy of CategoryBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CategoryBriefPayloadImplCopyWith<_$CategoryBriefPayloadImpl>
  get copyWith =>
      __$$CategoryBriefPayloadImplCopyWithImpl<_$CategoryBriefPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CategoryBriefPayloadImplToJson(this);
  }
}

abstract class _CategoryBriefPayload implements CategoryBriefPayload {
  const factory _CategoryBriefPayload({
    required final int id,
    required final String name,
    required final String url,
    required final String color,
  }) = _$CategoryBriefPayloadImpl;

  factory _CategoryBriefPayload.fromJson(Map<String, dynamic> json) =
      _$CategoryBriefPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get name;
  @override
  String get url;
  @override
  String get color;

  /// Create a copy of CategoryBriefPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CategoryBriefPayloadImplCopyWith<_$CategoryBriefPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

TopicPayload _$TopicPayloadFromJson(Map<String, dynamic> json) {
  return _TopicPayload.fromJson(json);
}

/// @nodoc
mixin _$TopicPayload {
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String? get firstImageUrl => throw _privateConstructorUsedError;
  List<String>? get images => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  UserBriefPayload get author => throw _privateConstructorUsedError;
  List<UserBriefPayload> get participants => throw _privateConstructorUsedError;
  List<CategoryBriefPayload> get categories =>
      throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get viewCount => throw _privateConstructorUsedError;
  int get pinWeight => throw _privateConstructorUsedError;
  int get processStatus => throw _privateConstructorUsedError;
  String get activityText => throw _privateConstructorUsedError;
  String get lastUpdateTime => throw _privateConstructorUsedError;
  bool? get unseen => throw _privateConstructorUsedError;

  /// Serializes this TopicPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TopicPayloadCopyWith<TopicPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TopicPayloadCopyWith<$Res> {
  factory $TopicPayloadCopyWith(
    TopicPayload value,
    $Res Function(TopicPayload) then,
  ) = _$TopicPayloadCopyWithImpl<$Res, TopicPayload>;
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String? firstImageUrl,
    List<String>? images,
    String url,
    UserBriefPayload author,
    List<UserBriefPayload> participants,
    List<CategoryBriefPayload> categories,
    int replyCount,
    int viewCount,
    int pinWeight,
    int processStatus,
    String activityText,
    String lastUpdateTime,
    bool? unseen,
  });

  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class _$TopicPayloadCopyWithImpl<$Res, $Val extends TopicPayload>
    implements $TopicPayloadCopyWith<$Res> {
  _$TopicPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? firstImageUrl = freezed,
    Object? images = freezed,
    Object? url = null,
    Object? author = null,
    Object? participants = null,
    Object? categories = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? pinWeight = null,
    Object? processStatus = null,
    Object? activityText = null,
    Object? lastUpdateTime = null,
    Object? unseen = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            firstImageUrl: freezed == firstImageUrl
                ? _value.firstImageUrl
                : firstImageUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
            images: freezed == images
                ? _value.images
                : images // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            author: null == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            participants: null == participants
                ? _value.participants
                : participants // ignore: cast_nullable_to_non_nullable
                      as List<UserBriefPayload>,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryBriefPayload>,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            viewCount: null == viewCount
                ? _value.viewCount
                : viewCount // ignore: cast_nullable_to_non_nullable
                      as int,
            pinWeight: null == pinWeight
                ? _value.pinWeight
                : pinWeight // ignore: cast_nullable_to_non_nullable
                      as int,
            processStatus: null == processStatus
                ? _value.processStatus
                : processStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            activityText: null == activityText
                ? _value.activityText
                : activityText // ignore: cast_nullable_to_non_nullable
                      as String,
            lastUpdateTime: null == lastUpdateTime
                ? _value.lastUpdateTime
                : lastUpdateTime // ignore: cast_nullable_to_non_nullable
                      as String,
            unseen: freezed == unseen
                ? _value.unseen
                : unseen // ignore: cast_nullable_to_non_nullable
                      as bool?,
          )
          as $Val,
    );
  }

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get author {
    return $UserBriefPayloadCopyWith<$Res>(_value.author, (value) {
      return _then(_value.copyWith(author: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$TopicPayloadImplCopyWith<$Res>
    implements $TopicPayloadCopyWith<$Res> {
  factory _$$TopicPayloadImplCopyWith(
    _$TopicPayloadImpl value,
    $Res Function(_$TopicPayloadImpl) then,
  ) = __$$TopicPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String? firstImageUrl,
    List<String>? images,
    String url,
    UserBriefPayload author,
    List<UserBriefPayload> participants,
    List<CategoryBriefPayload> categories,
    int replyCount,
    int viewCount,
    int pinWeight,
    int processStatus,
    String activityText,
    String lastUpdateTime,
    bool? unseen,
  });

  @override
  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class __$$TopicPayloadImplCopyWithImpl<$Res>
    extends _$TopicPayloadCopyWithImpl<$Res, _$TopicPayloadImpl>
    implements _$$TopicPayloadImplCopyWith<$Res> {
  __$$TopicPayloadImplCopyWithImpl(
    _$TopicPayloadImpl _value,
    $Res Function(_$TopicPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? firstImageUrl = freezed,
    Object? images = freezed,
    Object? url = null,
    Object? author = null,
    Object? participants = null,
    Object? categories = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? pinWeight = null,
    Object? processStatus = null,
    Object? activityText = null,
    Object? lastUpdateTime = null,
    Object? unseen = freezed,
  }) {
    return _then(
      _$TopicPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        firstImageUrl: freezed == firstImageUrl
            ? _value.firstImageUrl
            : firstImageUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
        images: freezed == images
            ? _value._images
            : images // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        author: null == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        participants: null == participants
            ? _value._participants
            : participants // ignore: cast_nullable_to_non_nullable
                  as List<UserBriefPayload>,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryBriefPayload>,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        viewCount: null == viewCount
            ? _value.viewCount
            : viewCount // ignore: cast_nullable_to_non_nullable
                  as int,
        pinWeight: null == pinWeight
            ? _value.pinWeight
            : pinWeight // ignore: cast_nullable_to_non_nullable
                  as int,
        processStatus: null == processStatus
            ? _value.processStatus
            : processStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        activityText: null == activityText
            ? _value.activityText
            : activityText // ignore: cast_nullable_to_non_nullable
                  as String,
        lastUpdateTime: null == lastUpdateTime
            ? _value.lastUpdateTime
            : lastUpdateTime // ignore: cast_nullable_to_non_nullable
                  as String,
        unseen: freezed == unseen
            ? _value.unseen
            : unseen // ignore: cast_nullable_to_non_nullable
                  as bool?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TopicPayloadImpl implements _TopicPayload {
  const _$TopicPayloadImpl({
    required this.id,
    required this.title,
    required this.description,
    this.firstImageUrl,
    final List<String>? images,
    required this.url,
    required this.author,
    required final List<UserBriefPayload> participants,
    required final List<CategoryBriefPayload> categories,
    required this.replyCount,
    required this.viewCount,
    required this.pinWeight,
    required this.processStatus,
    required this.activityText,
    required this.lastUpdateTime,
    this.unseen,
  }) : _images = images,
       _participants = participants,
       _categories = categories;

  factory _$TopicPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TopicPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String title;
  @override
  final String description;
  @override
  final String? firstImageUrl;
  final List<String>? _images;
  @override
  List<String>? get images {
    final value = _images;
    if (value == null) return null;
    if (_images is EqualUnmodifiableListView) return _images;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final String url;
  @override
  final UserBriefPayload author;
  final List<UserBriefPayload> _participants;
  @override
  List<UserBriefPayload> get participants {
    if (_participants is EqualUnmodifiableListView) return _participants;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_participants);
  }

  final List<CategoryBriefPayload> _categories;
  @override
  List<CategoryBriefPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final int replyCount;
  @override
  final int viewCount;
  @override
  final int pinWeight;
  @override
  final int processStatus;
  @override
  final String activityText;
  @override
  final String lastUpdateTime;
  @override
  final bool? unseen;

  @override
  String toString() {
    return 'TopicPayload(id: $id, title: $title, description: $description, firstImageUrl: $firstImageUrl, images: $images, url: $url, author: $author, participants: $participants, categories: $categories, replyCount: $replyCount, viewCount: $viewCount, pinWeight: $pinWeight, processStatus: $processStatus, activityText: $activityText, lastUpdateTime: $lastUpdateTime, unseen: $unseen)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TopicPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.firstImageUrl, firstImageUrl) ||
                other.firstImageUrl == firstImageUrl) &&
            const DeepCollectionEquality().equals(other._images, _images) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.author, author) || other.author == author) &&
            const DeepCollectionEquality().equals(
              other._participants,
              _participants,
            ) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.viewCount, viewCount) ||
                other.viewCount == viewCount) &&
            (identical(other.pinWeight, pinWeight) ||
                other.pinWeight == pinWeight) &&
            (identical(other.processStatus, processStatus) ||
                other.processStatus == processStatus) &&
            (identical(other.activityText, activityText) ||
                other.activityText == activityText) &&
            (identical(other.lastUpdateTime, lastUpdateTime) ||
                other.lastUpdateTime == lastUpdateTime) &&
            (identical(other.unseen, unseen) || other.unseen == unseen));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    title,
    description,
    firstImageUrl,
    const DeepCollectionEquality().hash(_images),
    url,
    author,
    const DeepCollectionEquality().hash(_participants),
    const DeepCollectionEquality().hash(_categories),
    replyCount,
    viewCount,
    pinWeight,
    processStatus,
    activityText,
    lastUpdateTime,
    unseen,
  );

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TopicPayloadImplCopyWith<_$TopicPayloadImpl> get copyWith =>
      __$$TopicPayloadImplCopyWithImpl<_$TopicPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TopicPayloadImplToJson(this);
  }
}

abstract class _TopicPayload implements TopicPayload {
  const factory _TopicPayload({
    required final int id,
    required final String title,
    required final String description,
    final String? firstImageUrl,
    final List<String>? images,
    required final String url,
    required final UserBriefPayload author,
    required final List<UserBriefPayload> participants,
    required final List<CategoryBriefPayload> categories,
    required final int replyCount,
    required final int viewCount,
    required final int pinWeight,
    required final int processStatus,
    required final String activityText,
    required final String lastUpdateTime,
    final bool? unseen,
  }) = _$TopicPayloadImpl;

  factory _TopicPayload.fromJson(Map<String, dynamic> json) =
      _$TopicPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get title;
  @override
  String get description;
  @override
  String? get firstImageUrl;
  @override
  List<String>? get images;
  @override
  String get url;
  @override
  UserBriefPayload get author;
  @override
  List<UserBriefPayload> get participants;
  @override
  List<CategoryBriefPayload> get categories;
  @override
  int get replyCount;
  @override
  int get viewCount;
  @override
  int get pinWeight;
  @override
  int get processStatus;
  @override
  String get activityText;
  @override
  String get lastUpdateTime;
  @override
  bool? get unseen;

  /// Create a copy of TopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TopicPayloadImplCopyWith<_$TopicPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TopicDetailPayload _$TopicDetailPayloadFromJson(Map<String, dynamic> json) {
  return _TopicDetailPayload.fromJson(json);
}

/// @nodoc
mixin _$TopicDetailPayload {
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  int get topicStatus => throw _privateConstructorUsedError;
  int get processStatus => throw _privateConstructorUsedError;
  UserBriefPayload get author => throw _privateConstructorUsedError;
  List<UserBriefPayload> get participants => throw _privateConstructorUsedError;
  List<CategoryBriefPayload> get categories =>
      throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get maxPostNo => throw _privateConstructorUsedError;
  int get viewCount => throw _privateConstructorUsedError;
  int get likeCount => throw _privateConstructorUsedError;
  bool get isLiked => throw _privateConstructorUsedError;
  bool get isBookmarked => throw _privateConstructorUsedError;
  bool get isWatched => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  String get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this TopicDetailPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TopicDetailPayloadCopyWith<TopicDetailPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TopicDetailPayloadCopyWith<$Res> {
  factory $TopicDetailPayloadCopyWith(
    TopicDetailPayload value,
    $Res Function(TopicDetailPayload) then,
  ) = _$TopicDetailPayloadCopyWithImpl<$Res, TopicDetailPayload>;
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String url,
    int topicStatus,
    int processStatus,
    UserBriefPayload author,
    List<UserBriefPayload> participants,
    List<CategoryBriefPayload> categories,
    int replyCount,
    int maxPostNo,
    int viewCount,
    int likeCount,
    bool isLiked,
    bool isBookmarked,
    bool isWatched,
    String createdAt,
    String updatedAt,
  });

  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class _$TopicDetailPayloadCopyWithImpl<$Res, $Val extends TopicDetailPayload>
    implements $TopicDetailPayloadCopyWith<$Res> {
  _$TopicDetailPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? url = null,
    Object? topicStatus = null,
    Object? processStatus = null,
    Object? author = null,
    Object? participants = null,
    Object? categories = null,
    Object? replyCount = null,
    Object? maxPostNo = null,
    Object? viewCount = null,
    Object? likeCount = null,
    Object? isLiked = null,
    Object? isBookmarked = null,
    Object? isWatched = null,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            topicStatus: null == topicStatus
                ? _value.topicStatus
                : topicStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            processStatus: null == processStatus
                ? _value.processStatus
                : processStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            author: null == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            participants: null == participants
                ? _value.participants
                : participants // ignore: cast_nullable_to_non_nullable
                      as List<UserBriefPayload>,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryBriefPayload>,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            maxPostNo: null == maxPostNo
                ? _value.maxPostNo
                : maxPostNo // ignore: cast_nullable_to_non_nullable
                      as int,
            viewCount: null == viewCount
                ? _value.viewCount
                : viewCount // ignore: cast_nullable_to_non_nullable
                      as int,
            likeCount: null == likeCount
                ? _value.likeCount
                : likeCount // ignore: cast_nullable_to_non_nullable
                      as int,
            isLiked: null == isLiked
                ? _value.isLiked
                : isLiked // ignore: cast_nullable_to_non_nullable
                      as bool,
            isBookmarked: null == isBookmarked
                ? _value.isBookmarked
                : isBookmarked // ignore: cast_nullable_to_non_nullable
                      as bool,
            isWatched: null == isWatched
                ? _value.isWatched
                : isWatched // ignore: cast_nullable_to_non_nullable
                      as bool,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get author {
    return $UserBriefPayloadCopyWith<$Res>(_value.author, (value) {
      return _then(_value.copyWith(author: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$TopicDetailPayloadImplCopyWith<$Res>
    implements $TopicDetailPayloadCopyWith<$Res> {
  factory _$$TopicDetailPayloadImplCopyWith(
    _$TopicDetailPayloadImpl value,
    $Res Function(_$TopicDetailPayloadImpl) then,
  ) = __$$TopicDetailPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String url,
    int topicStatus,
    int processStatus,
    UserBriefPayload author,
    List<UserBriefPayload> participants,
    List<CategoryBriefPayload> categories,
    int replyCount,
    int maxPostNo,
    int viewCount,
    int likeCount,
    bool isLiked,
    bool isBookmarked,
    bool isWatched,
    String createdAt,
    String updatedAt,
  });

  @override
  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class __$$TopicDetailPayloadImplCopyWithImpl<$Res>
    extends _$TopicDetailPayloadCopyWithImpl<$Res, _$TopicDetailPayloadImpl>
    implements _$$TopicDetailPayloadImplCopyWith<$Res> {
  __$$TopicDetailPayloadImplCopyWithImpl(
    _$TopicDetailPayloadImpl _value,
    $Res Function(_$TopicDetailPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? url = null,
    Object? topicStatus = null,
    Object? processStatus = null,
    Object? author = null,
    Object? participants = null,
    Object? categories = null,
    Object? replyCount = null,
    Object? maxPostNo = null,
    Object? viewCount = null,
    Object? likeCount = null,
    Object? isLiked = null,
    Object? isBookmarked = null,
    Object? isWatched = null,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _$TopicDetailPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        topicStatus: null == topicStatus
            ? _value.topicStatus
            : topicStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        processStatus: null == processStatus
            ? _value.processStatus
            : processStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        author: null == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        participants: null == participants
            ? _value._participants
            : participants // ignore: cast_nullable_to_non_nullable
                  as List<UserBriefPayload>,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryBriefPayload>,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        maxPostNo: null == maxPostNo
            ? _value.maxPostNo
            : maxPostNo // ignore: cast_nullable_to_non_nullable
                  as int,
        viewCount: null == viewCount
            ? _value.viewCount
            : viewCount // ignore: cast_nullable_to_non_nullable
                  as int,
        likeCount: null == likeCount
            ? _value.likeCount
            : likeCount // ignore: cast_nullable_to_non_nullable
                  as int,
        isLiked: null == isLiked
            ? _value.isLiked
            : isLiked // ignore: cast_nullable_to_non_nullable
                  as bool,
        isBookmarked: null == isBookmarked
            ? _value.isBookmarked
            : isBookmarked // ignore: cast_nullable_to_non_nullable
                  as bool,
        isWatched: null == isWatched
            ? _value.isWatched
            : isWatched // ignore: cast_nullable_to_non_nullable
                  as bool,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TopicDetailPayloadImpl implements _TopicDetailPayload {
  const _$TopicDetailPayloadImpl({
    required this.id,
    required this.title,
    required this.description,
    required this.url,
    required this.topicStatus,
    required this.processStatus,
    required this.author,
    required final List<UserBriefPayload> participants,
    required final List<CategoryBriefPayload> categories,
    required this.replyCount,
    required this.maxPostNo,
    required this.viewCount,
    required this.likeCount,
    required this.isLiked,
    required this.isBookmarked,
    required this.isWatched,
    required this.createdAt,
    required this.updatedAt,
  }) : _participants = participants,
       _categories = categories;

  factory _$TopicDetailPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TopicDetailPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String title;
  @override
  final String description;
  @override
  final String url;
  @override
  final int topicStatus;
  @override
  final int processStatus;
  @override
  final UserBriefPayload author;
  final List<UserBriefPayload> _participants;
  @override
  List<UserBriefPayload> get participants {
    if (_participants is EqualUnmodifiableListView) return _participants;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_participants);
  }

  final List<CategoryBriefPayload> _categories;
  @override
  List<CategoryBriefPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final int replyCount;
  @override
  final int maxPostNo;
  @override
  final int viewCount;
  @override
  final int likeCount;
  @override
  final bool isLiked;
  @override
  final bool isBookmarked;
  @override
  final bool isWatched;
  @override
  final String createdAt;
  @override
  final String updatedAt;

  @override
  String toString() {
    return 'TopicDetailPayload(id: $id, title: $title, description: $description, url: $url, topicStatus: $topicStatus, processStatus: $processStatus, author: $author, participants: $participants, categories: $categories, replyCount: $replyCount, maxPostNo: $maxPostNo, viewCount: $viewCount, likeCount: $likeCount, isLiked: $isLiked, isBookmarked: $isBookmarked, isWatched: $isWatched, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TopicDetailPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.topicStatus, topicStatus) ||
                other.topicStatus == topicStatus) &&
            (identical(other.processStatus, processStatus) ||
                other.processStatus == processStatus) &&
            (identical(other.author, author) || other.author == author) &&
            const DeepCollectionEquality().equals(
              other._participants,
              _participants,
            ) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.maxPostNo, maxPostNo) ||
                other.maxPostNo == maxPostNo) &&
            (identical(other.viewCount, viewCount) ||
                other.viewCount == viewCount) &&
            (identical(other.likeCount, likeCount) ||
                other.likeCount == likeCount) &&
            (identical(other.isLiked, isLiked) || other.isLiked == isLiked) &&
            (identical(other.isBookmarked, isBookmarked) ||
                other.isBookmarked == isBookmarked) &&
            (identical(other.isWatched, isWatched) ||
                other.isWatched == isWatched) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    title,
    description,
    url,
    topicStatus,
    processStatus,
    author,
    const DeepCollectionEquality().hash(_participants),
    const DeepCollectionEquality().hash(_categories),
    replyCount,
    maxPostNo,
    viewCount,
    likeCount,
    isLiked,
    isBookmarked,
    isWatched,
    createdAt,
    updatedAt,
  );

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TopicDetailPayloadImplCopyWith<_$TopicDetailPayloadImpl> get copyWith =>
      __$$TopicDetailPayloadImplCopyWithImpl<_$TopicDetailPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TopicDetailPayloadImplToJson(this);
  }
}

abstract class _TopicDetailPayload implements TopicDetailPayload {
  const factory _TopicDetailPayload({
    required final int id,
    required final String title,
    required final String description,
    required final String url,
    required final int topicStatus,
    required final int processStatus,
    required final UserBriefPayload author,
    required final List<UserBriefPayload> participants,
    required final List<CategoryBriefPayload> categories,
    required final int replyCount,
    required final int maxPostNo,
    required final int viewCount,
    required final int likeCount,
    required final bool isLiked,
    required final bool isBookmarked,
    required final bool isWatched,
    required final String createdAt,
    required final String updatedAt,
  }) = _$TopicDetailPayloadImpl;

  factory _TopicDetailPayload.fromJson(Map<String, dynamic> json) =
      _$TopicDetailPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get title;
  @override
  String get description;
  @override
  String get url;
  @override
  int get topicStatus;
  @override
  int get processStatus;
  @override
  UserBriefPayload get author;
  @override
  List<UserBriefPayload> get participants;
  @override
  List<CategoryBriefPayload> get categories;
  @override
  int get replyCount;
  @override
  int get maxPostNo;
  @override
  int get viewCount;
  @override
  int get likeCount;
  @override
  bool get isLiked;
  @override
  bool get isBookmarked;
  @override
  bool get isWatched;
  @override
  String get createdAt;
  @override
  String get updatedAt;

  /// Create a copy of TopicDetailPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TopicDetailPayloadImplCopyWith<_$TopicDetailPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PostPayload _$PostPayloadFromJson(Map<String, dynamic> json) {
  return _PostPayload.fromJson(json);
}

/// @nodoc
mixin _$PostPayload {
  int get id => throw _privateConstructorUsedError;
  int get topicId => throw _privateConstructorUsedError;
  int get postNo => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  String get renderedContent => throw _privateConstructorUsedError;
  int get processStatus => throw _privateConstructorUsedError;
  bool get isHidden => throw _privateConstructorUsedError;
  bool get canModerate => throw _privateConstructorUsedError;
  UserBriefPayload get author => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  int? get replyToPostId => throw _privateConstructorUsedError;
  int? get replyToUserId => throw _privateConstructorUsedError;
  String? get replyToUsername => throw _privateConstructorUsedError;
  bool get isOwnPost => throw _privateConstructorUsedError;
  String? get updatedAt => throw _privateConstructorUsedError;
  int get likeCount => throw _privateConstructorUsedError;
  bool get isLiked => throw _privateConstructorUsedError;
  bool get isBookmarked => throw _privateConstructorUsedError;

  /// Serializes this PostPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PostPayloadCopyWith<PostPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PostPayloadCopyWith<$Res> {
  factory $PostPayloadCopyWith(
    PostPayload value,
    $Res Function(PostPayload) then,
  ) = _$PostPayloadCopyWithImpl<$Res, PostPayload>;
  @useResult
  $Res call({
    int id,
    int topicId,
    int postNo,
    String content,
    String renderedContent,
    int processStatus,
    bool isHidden,
    bool canModerate,
    UserBriefPayload author,
    String createdAt,
    int? replyToPostId,
    int? replyToUserId,
    String? replyToUsername,
    bool isOwnPost,
    String? updatedAt,
    int likeCount,
    bool isLiked,
    bool isBookmarked,
  });

  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class _$PostPayloadCopyWithImpl<$Res, $Val extends PostPayload>
    implements $PostPayloadCopyWith<$Res> {
  _$PostPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? postNo = null,
    Object? content = null,
    Object? renderedContent = null,
    Object? processStatus = null,
    Object? isHidden = null,
    Object? canModerate = null,
    Object? author = null,
    Object? createdAt = null,
    Object? replyToPostId = freezed,
    Object? replyToUserId = freezed,
    Object? replyToUsername = freezed,
    Object? isOwnPost = null,
    Object? updatedAt = freezed,
    Object? likeCount = null,
    Object? isLiked = null,
    Object? isBookmarked = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            topicId: null == topicId
                ? _value.topicId
                : topicId // ignore: cast_nullable_to_non_nullable
                      as int,
            postNo: null == postNo
                ? _value.postNo
                : postNo // ignore: cast_nullable_to_non_nullable
                      as int,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            renderedContent: null == renderedContent
                ? _value.renderedContent
                : renderedContent // ignore: cast_nullable_to_non_nullable
                      as String,
            processStatus: null == processStatus
                ? _value.processStatus
                : processStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            isHidden: null == isHidden
                ? _value.isHidden
                : isHidden // ignore: cast_nullable_to_non_nullable
                      as bool,
            canModerate: null == canModerate
                ? _value.canModerate
                : canModerate // ignore: cast_nullable_to_non_nullable
                      as bool,
            author: null == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            replyToPostId: freezed == replyToPostId
                ? _value.replyToPostId
                : replyToPostId // ignore: cast_nullable_to_non_nullable
                      as int?,
            replyToUserId: freezed == replyToUserId
                ? _value.replyToUserId
                : replyToUserId // ignore: cast_nullable_to_non_nullable
                      as int?,
            replyToUsername: freezed == replyToUsername
                ? _value.replyToUsername
                : replyToUsername // ignore: cast_nullable_to_non_nullable
                      as String?,
            isOwnPost: null == isOwnPost
                ? _value.isOwnPost
                : isOwnPost // ignore: cast_nullable_to_non_nullable
                      as bool,
            updatedAt: freezed == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
            likeCount: null == likeCount
                ? _value.likeCount
                : likeCount // ignore: cast_nullable_to_non_nullable
                      as int,
            isLiked: null == isLiked
                ? _value.isLiked
                : isLiked // ignore: cast_nullable_to_non_nullable
                      as bool,
            isBookmarked: null == isBookmarked
                ? _value.isBookmarked
                : isBookmarked // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get author {
    return $UserBriefPayloadCopyWith<$Res>(_value.author, (value) {
      return _then(_value.copyWith(author: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$PostPayloadImplCopyWith<$Res>
    implements $PostPayloadCopyWith<$Res> {
  factory _$$PostPayloadImplCopyWith(
    _$PostPayloadImpl value,
    $Res Function(_$PostPayloadImpl) then,
  ) = __$$PostPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int topicId,
    int postNo,
    String content,
    String renderedContent,
    int processStatus,
    bool isHidden,
    bool canModerate,
    UserBriefPayload author,
    String createdAt,
    int? replyToPostId,
    int? replyToUserId,
    String? replyToUsername,
    bool isOwnPost,
    String? updatedAt,
    int likeCount,
    bool isLiked,
    bool isBookmarked,
  });

  @override
  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class __$$PostPayloadImplCopyWithImpl<$Res>
    extends _$PostPayloadCopyWithImpl<$Res, _$PostPayloadImpl>
    implements _$$PostPayloadImplCopyWith<$Res> {
  __$$PostPayloadImplCopyWithImpl(
    _$PostPayloadImpl _value,
    $Res Function(_$PostPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? postNo = null,
    Object? content = null,
    Object? renderedContent = null,
    Object? processStatus = null,
    Object? isHidden = null,
    Object? canModerate = null,
    Object? author = null,
    Object? createdAt = null,
    Object? replyToPostId = freezed,
    Object? replyToUserId = freezed,
    Object? replyToUsername = freezed,
    Object? isOwnPost = null,
    Object? updatedAt = freezed,
    Object? likeCount = null,
    Object? isLiked = null,
    Object? isBookmarked = null,
  }) {
    return _then(
      _$PostPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        topicId: null == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int,
        postNo: null == postNo
            ? _value.postNo
            : postNo // ignore: cast_nullable_to_non_nullable
                  as int,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        renderedContent: null == renderedContent
            ? _value.renderedContent
            : renderedContent // ignore: cast_nullable_to_non_nullable
                  as String,
        processStatus: null == processStatus
            ? _value.processStatus
            : processStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        isHidden: null == isHidden
            ? _value.isHidden
            : isHidden // ignore: cast_nullable_to_non_nullable
                  as bool,
        canModerate: null == canModerate
            ? _value.canModerate
            : canModerate // ignore: cast_nullable_to_non_nullable
                  as bool,
        author: null == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        replyToPostId: freezed == replyToPostId
            ? _value.replyToPostId
            : replyToPostId // ignore: cast_nullable_to_non_nullable
                  as int?,
        replyToUserId: freezed == replyToUserId
            ? _value.replyToUserId
            : replyToUserId // ignore: cast_nullable_to_non_nullable
                  as int?,
        replyToUsername: freezed == replyToUsername
            ? _value.replyToUsername
            : replyToUsername // ignore: cast_nullable_to_non_nullable
                  as String?,
        isOwnPost: null == isOwnPost
            ? _value.isOwnPost
            : isOwnPost // ignore: cast_nullable_to_non_nullable
                  as bool,
        updatedAt: freezed == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
        likeCount: null == likeCount
            ? _value.likeCount
            : likeCount // ignore: cast_nullable_to_non_nullable
                  as int,
        isLiked: null == isLiked
            ? _value.isLiked
            : isLiked // ignore: cast_nullable_to_non_nullable
                  as bool,
        isBookmarked: null == isBookmarked
            ? _value.isBookmarked
            : isBookmarked // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PostPayloadImpl implements _PostPayload {
  const _$PostPayloadImpl({
    required this.id,
    required this.topicId,
    required this.postNo,
    required this.content,
    required this.renderedContent,
    required this.processStatus,
    required this.isHidden,
    required this.canModerate,
    required this.author,
    required this.createdAt,
    this.replyToPostId,
    this.replyToUserId,
    this.replyToUsername,
    required this.isOwnPost,
    this.updatedAt,
    required this.likeCount,
    required this.isLiked,
    required this.isBookmarked,
  });

  factory _$PostPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PostPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int topicId;
  @override
  final int postNo;
  @override
  final String content;
  @override
  final String renderedContent;
  @override
  final int processStatus;
  @override
  final bool isHidden;
  @override
  final bool canModerate;
  @override
  final UserBriefPayload author;
  @override
  final String createdAt;
  @override
  final int? replyToPostId;
  @override
  final int? replyToUserId;
  @override
  final String? replyToUsername;
  @override
  final bool isOwnPost;
  @override
  final String? updatedAt;
  @override
  final int likeCount;
  @override
  final bool isLiked;
  @override
  final bool isBookmarked;

  @override
  String toString() {
    return 'PostPayload(id: $id, topicId: $topicId, postNo: $postNo, content: $content, renderedContent: $renderedContent, processStatus: $processStatus, isHidden: $isHidden, canModerate: $canModerate, author: $author, createdAt: $createdAt, replyToPostId: $replyToPostId, replyToUserId: $replyToUserId, replyToUsername: $replyToUsername, isOwnPost: $isOwnPost, updatedAt: $updatedAt, likeCount: $likeCount, isLiked: $isLiked, isBookmarked: $isBookmarked)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PostPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.postNo, postNo) || other.postNo == postNo) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.renderedContent, renderedContent) ||
                other.renderedContent == renderedContent) &&
            (identical(other.processStatus, processStatus) ||
                other.processStatus == processStatus) &&
            (identical(other.isHidden, isHidden) ||
                other.isHidden == isHidden) &&
            (identical(other.canModerate, canModerate) ||
                other.canModerate == canModerate) &&
            (identical(other.author, author) || other.author == author) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.replyToPostId, replyToPostId) ||
                other.replyToPostId == replyToPostId) &&
            (identical(other.replyToUserId, replyToUserId) ||
                other.replyToUserId == replyToUserId) &&
            (identical(other.replyToUsername, replyToUsername) ||
                other.replyToUsername == replyToUsername) &&
            (identical(other.isOwnPost, isOwnPost) ||
                other.isOwnPost == isOwnPost) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.likeCount, likeCount) ||
                other.likeCount == likeCount) &&
            (identical(other.isLiked, isLiked) || other.isLiked == isLiked) &&
            (identical(other.isBookmarked, isBookmarked) ||
                other.isBookmarked == isBookmarked));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    topicId,
    postNo,
    content,
    renderedContent,
    processStatus,
    isHidden,
    canModerate,
    author,
    createdAt,
    replyToPostId,
    replyToUserId,
    replyToUsername,
    isOwnPost,
    updatedAt,
    likeCount,
    isLiked,
    isBookmarked,
  );

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PostPayloadImplCopyWith<_$PostPayloadImpl> get copyWith =>
      __$$PostPayloadImplCopyWithImpl<_$PostPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$PostPayloadImplToJson(this);
  }
}

abstract class _PostPayload implements PostPayload {
  const factory _PostPayload({
    required final int id,
    required final int topicId,
    required final int postNo,
    required final String content,
    required final String renderedContent,
    required final int processStatus,
    required final bool isHidden,
    required final bool canModerate,
    required final UserBriefPayload author,
    required final String createdAt,
    final int? replyToPostId,
    final int? replyToUserId,
    final String? replyToUsername,
    required final bool isOwnPost,
    final String? updatedAt,
    required final int likeCount,
    required final bool isLiked,
    required final bool isBookmarked,
  }) = _$PostPayloadImpl;

  factory _PostPayload.fromJson(Map<String, dynamic> json) =
      _$PostPayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get topicId;
  @override
  int get postNo;
  @override
  String get content;
  @override
  String get renderedContent;
  @override
  int get processStatus;
  @override
  bool get isHidden;
  @override
  bool get canModerate;
  @override
  UserBriefPayload get author;
  @override
  String get createdAt;
  @override
  int? get replyToPostId;
  @override
  int? get replyToUserId;
  @override
  String? get replyToUsername;
  @override
  bool get isOwnPost;
  @override
  String? get updatedAt;
  @override
  int get likeCount;
  @override
  bool get isLiked;
  @override
  bool get isBookmarked;

  /// Create a copy of PostPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PostPayloadImplCopyWith<_$PostPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ReplyTargetPayload _$ReplyTargetPayloadFromJson(Map<String, dynamic> json) {
  return _ReplyTargetPayload.fromJson(json);
}

/// @nodoc
mixin _$ReplyTargetPayload {
  int get id => throw _privateConstructorUsedError;
  int? get postNo => throw _privateConstructorUsedError;
  UserBriefPayload get author => throw _privateConstructorUsedError;
  String? get renderedContent => throw _privateConstructorUsedError;
  bool? get unavailable => throw _privateConstructorUsedError;

  /// Serializes this ReplyTargetPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ReplyTargetPayloadCopyWith<ReplyTargetPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ReplyTargetPayloadCopyWith<$Res> {
  factory $ReplyTargetPayloadCopyWith(
    ReplyTargetPayload value,
    $Res Function(ReplyTargetPayload) then,
  ) = _$ReplyTargetPayloadCopyWithImpl<$Res, ReplyTargetPayload>;
  @useResult
  $Res call({
    int id,
    int? postNo,
    UserBriefPayload author,
    String? renderedContent,
    bool? unavailable,
  });

  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class _$ReplyTargetPayloadCopyWithImpl<$Res, $Val extends ReplyTargetPayload>
    implements $ReplyTargetPayloadCopyWith<$Res> {
  _$ReplyTargetPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? postNo = freezed,
    Object? author = null,
    Object? renderedContent = freezed,
    Object? unavailable = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            postNo: freezed == postNo
                ? _value.postNo
                : postNo // ignore: cast_nullable_to_non_nullable
                      as int?,
            author: null == author
                ? _value.author
                : author // ignore: cast_nullable_to_non_nullable
                      as UserBriefPayload,
            renderedContent: freezed == renderedContent
                ? _value.renderedContent
                : renderedContent // ignore: cast_nullable_to_non_nullable
                      as String?,
            unavailable: freezed == unavailable
                ? _value.unavailable
                : unavailable // ignore: cast_nullable_to_non_nullable
                      as bool?,
          )
          as $Val,
    );
  }

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserBriefPayloadCopyWith<$Res> get author {
    return $UserBriefPayloadCopyWith<$Res>(_value.author, (value) {
      return _then(_value.copyWith(author: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ReplyTargetPayloadImplCopyWith<$Res>
    implements $ReplyTargetPayloadCopyWith<$Res> {
  factory _$$ReplyTargetPayloadImplCopyWith(
    _$ReplyTargetPayloadImpl value,
    $Res Function(_$ReplyTargetPayloadImpl) then,
  ) = __$$ReplyTargetPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int? postNo,
    UserBriefPayload author,
    String? renderedContent,
    bool? unavailable,
  });

  @override
  $UserBriefPayloadCopyWith<$Res> get author;
}

/// @nodoc
class __$$ReplyTargetPayloadImplCopyWithImpl<$Res>
    extends _$ReplyTargetPayloadCopyWithImpl<$Res, _$ReplyTargetPayloadImpl>
    implements _$$ReplyTargetPayloadImplCopyWith<$Res> {
  __$$ReplyTargetPayloadImplCopyWithImpl(
    _$ReplyTargetPayloadImpl _value,
    $Res Function(_$ReplyTargetPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? postNo = freezed,
    Object? author = null,
    Object? renderedContent = freezed,
    Object? unavailable = freezed,
  }) {
    return _then(
      _$ReplyTargetPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        postNo: freezed == postNo
            ? _value.postNo
            : postNo // ignore: cast_nullable_to_non_nullable
                  as int?,
        author: null == author
            ? _value.author
            : author // ignore: cast_nullable_to_non_nullable
                  as UserBriefPayload,
        renderedContent: freezed == renderedContent
            ? _value.renderedContent
            : renderedContent // ignore: cast_nullable_to_non_nullable
                  as String?,
        unavailable: freezed == unavailable
            ? _value.unavailable
            : unavailable // ignore: cast_nullable_to_non_nullable
                  as bool?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ReplyTargetPayloadImpl implements _ReplyTargetPayload {
  const _$ReplyTargetPayloadImpl({
    required this.id,
    this.postNo,
    required this.author,
    this.renderedContent,
    this.unavailable,
  });

  factory _$ReplyTargetPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ReplyTargetPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int? postNo;
  @override
  final UserBriefPayload author;
  @override
  final String? renderedContent;
  @override
  final bool? unavailable;

  @override
  String toString() {
    return 'ReplyTargetPayload(id: $id, postNo: $postNo, author: $author, renderedContent: $renderedContent, unavailable: $unavailable)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ReplyTargetPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.postNo, postNo) || other.postNo == postNo) &&
            (identical(other.author, author) || other.author == author) &&
            (identical(other.renderedContent, renderedContent) ||
                other.renderedContent == renderedContent) &&
            (identical(other.unavailable, unavailable) ||
                other.unavailable == unavailable));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    postNo,
    author,
    renderedContent,
    unavailable,
  );

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ReplyTargetPayloadImplCopyWith<_$ReplyTargetPayloadImpl> get copyWith =>
      __$$ReplyTargetPayloadImplCopyWithImpl<_$ReplyTargetPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ReplyTargetPayloadImplToJson(this);
  }
}

abstract class _ReplyTargetPayload implements ReplyTargetPayload {
  const factory _ReplyTargetPayload({
    required final int id,
    final int? postNo,
    required final UserBriefPayload author,
    final String? renderedContent,
    final bool? unavailable,
  }) = _$ReplyTargetPayloadImpl;

  factory _ReplyTargetPayload.fromJson(Map<String, dynamic> json) =
      _$ReplyTargetPayloadImpl.fromJson;

  @override
  int get id;
  @override
  int? get postNo;
  @override
  UserBriefPayload get author;
  @override
  String? get renderedContent;
  @override
  bool? get unavailable;

  /// Create a copy of ReplyTargetPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ReplyTargetPayloadImplCopyWith<_$ReplyTargetPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PostWindowPayload _$PostWindowPayloadFromJson(Map<String, dynamic> json) {
  return _PostWindowPayload.fromJson(json);
}

/// @nodoc
mixin _$PostWindowPayload {
  List<PostPayload> get posts => throw _privateConstructorUsedError;
  List<ReplyTargetPayload> get replyTargets =>
      throw _privateConstructorUsedError;
  int? get anchorPostId => throw _privateConstructorUsedError;
  int? get beforePostNo => throw _privateConstructorUsedError;
  int? get afterPostNo => throw _privateConstructorUsedError;
  bool get hasBefore => throw _privateConstructorUsedError;
  bool get hasAfter => throw _privateConstructorUsedError;
  int get total => throw _privateConstructorUsedError;
  int get maxPostNo => throw _privateConstructorUsedError;

  /// Serializes this PostWindowPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PostWindowPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PostWindowPayloadCopyWith<PostWindowPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PostWindowPayloadCopyWith<$Res> {
  factory $PostWindowPayloadCopyWith(
    PostWindowPayload value,
    $Res Function(PostWindowPayload) then,
  ) = _$PostWindowPayloadCopyWithImpl<$Res, PostWindowPayload>;
  @useResult
  $Res call({
    List<PostPayload> posts,
    List<ReplyTargetPayload> replyTargets,
    int? anchorPostId,
    int? beforePostNo,
    int? afterPostNo,
    bool hasBefore,
    bool hasAfter,
    int total,
    int maxPostNo,
  });
}

/// @nodoc
class _$PostWindowPayloadCopyWithImpl<$Res, $Val extends PostWindowPayload>
    implements $PostWindowPayloadCopyWith<$Res> {
  _$PostWindowPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PostWindowPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? posts = null,
    Object? replyTargets = null,
    Object? anchorPostId = freezed,
    Object? beforePostNo = freezed,
    Object? afterPostNo = freezed,
    Object? hasBefore = null,
    Object? hasAfter = null,
    Object? total = null,
    Object? maxPostNo = null,
  }) {
    return _then(
      _value.copyWith(
            posts: null == posts
                ? _value.posts
                : posts // ignore: cast_nullable_to_non_nullable
                      as List<PostPayload>,
            replyTargets: null == replyTargets
                ? _value.replyTargets
                : replyTargets // ignore: cast_nullable_to_non_nullable
                      as List<ReplyTargetPayload>,
            anchorPostId: freezed == anchorPostId
                ? _value.anchorPostId
                : anchorPostId // ignore: cast_nullable_to_non_nullable
                      as int?,
            beforePostNo: freezed == beforePostNo
                ? _value.beforePostNo
                : beforePostNo // ignore: cast_nullable_to_non_nullable
                      as int?,
            afterPostNo: freezed == afterPostNo
                ? _value.afterPostNo
                : afterPostNo // ignore: cast_nullable_to_non_nullable
                      as int?,
            hasBefore: null == hasBefore
                ? _value.hasBefore
                : hasBefore // ignore: cast_nullable_to_non_nullable
                      as bool,
            hasAfter: null == hasAfter
                ? _value.hasAfter
                : hasAfter // ignore: cast_nullable_to_non_nullable
                      as bool,
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
                      as int,
            maxPostNo: null == maxPostNo
                ? _value.maxPostNo
                : maxPostNo // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$PostWindowPayloadImplCopyWith<$Res>
    implements $PostWindowPayloadCopyWith<$Res> {
  factory _$$PostWindowPayloadImplCopyWith(
    _$PostWindowPayloadImpl value,
    $Res Function(_$PostWindowPayloadImpl) then,
  ) = __$$PostWindowPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<PostPayload> posts,
    List<ReplyTargetPayload> replyTargets,
    int? anchorPostId,
    int? beforePostNo,
    int? afterPostNo,
    bool hasBefore,
    bool hasAfter,
    int total,
    int maxPostNo,
  });
}

/// @nodoc
class __$$PostWindowPayloadImplCopyWithImpl<$Res>
    extends _$PostWindowPayloadCopyWithImpl<$Res, _$PostWindowPayloadImpl>
    implements _$$PostWindowPayloadImplCopyWith<$Res> {
  __$$PostWindowPayloadImplCopyWithImpl(
    _$PostWindowPayloadImpl _value,
    $Res Function(_$PostWindowPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PostWindowPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? posts = null,
    Object? replyTargets = null,
    Object? anchorPostId = freezed,
    Object? beforePostNo = freezed,
    Object? afterPostNo = freezed,
    Object? hasBefore = null,
    Object? hasAfter = null,
    Object? total = null,
    Object? maxPostNo = null,
  }) {
    return _then(
      _$PostWindowPayloadImpl(
        posts: null == posts
            ? _value._posts
            : posts // ignore: cast_nullable_to_non_nullable
                  as List<PostPayload>,
        replyTargets: null == replyTargets
            ? _value._replyTargets
            : replyTargets // ignore: cast_nullable_to_non_nullable
                  as List<ReplyTargetPayload>,
        anchorPostId: freezed == anchorPostId
            ? _value.anchorPostId
            : anchorPostId // ignore: cast_nullable_to_non_nullable
                  as int?,
        beforePostNo: freezed == beforePostNo
            ? _value.beforePostNo
            : beforePostNo // ignore: cast_nullable_to_non_nullable
                  as int?,
        afterPostNo: freezed == afterPostNo
            ? _value.afterPostNo
            : afterPostNo // ignore: cast_nullable_to_non_nullable
                  as int?,
        hasBefore: null == hasBefore
            ? _value.hasBefore
            : hasBefore // ignore: cast_nullable_to_non_nullable
                  as bool,
        hasAfter: null == hasAfter
            ? _value.hasAfter
            : hasAfter // ignore: cast_nullable_to_non_nullable
                  as bool,
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
                  as int,
        maxPostNo: null == maxPostNo
            ? _value.maxPostNo
            : maxPostNo // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PostWindowPayloadImpl implements _PostWindowPayload {
  const _$PostWindowPayloadImpl({
    required final List<PostPayload> posts,
    required final List<ReplyTargetPayload> replyTargets,
    this.anchorPostId,
    this.beforePostNo,
    this.afterPostNo,
    required this.hasBefore,
    required this.hasAfter,
    required this.total,
    required this.maxPostNo,
  }) : _posts = posts,
       _replyTargets = replyTargets;

  factory _$PostWindowPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PostWindowPayloadImplFromJson(json);

  final List<PostPayload> _posts;
  @override
  List<PostPayload> get posts {
    if (_posts is EqualUnmodifiableListView) return _posts;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_posts);
  }

  final List<ReplyTargetPayload> _replyTargets;
  @override
  List<ReplyTargetPayload> get replyTargets {
    if (_replyTargets is EqualUnmodifiableListView) return _replyTargets;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_replyTargets);
  }

  @override
  final int? anchorPostId;
  @override
  final int? beforePostNo;
  @override
  final int? afterPostNo;
  @override
  final bool hasBefore;
  @override
  final bool hasAfter;
  @override
  final int total;
  @override
  final int maxPostNo;

  @override
  String toString() {
    return 'PostWindowPayload(posts: $posts, replyTargets: $replyTargets, anchorPostId: $anchorPostId, beforePostNo: $beforePostNo, afterPostNo: $afterPostNo, hasBefore: $hasBefore, hasAfter: $hasAfter, total: $total, maxPostNo: $maxPostNo)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PostWindowPayloadImpl &&
            const DeepCollectionEquality().equals(other._posts, _posts) &&
            const DeepCollectionEquality().equals(
              other._replyTargets,
              _replyTargets,
            ) &&
            (identical(other.anchorPostId, anchorPostId) ||
                other.anchorPostId == anchorPostId) &&
            (identical(other.beforePostNo, beforePostNo) ||
                other.beforePostNo == beforePostNo) &&
            (identical(other.afterPostNo, afterPostNo) ||
                other.afterPostNo == afterPostNo) &&
            (identical(other.hasBefore, hasBefore) ||
                other.hasBefore == hasBefore) &&
            (identical(other.hasAfter, hasAfter) ||
                other.hasAfter == hasAfter) &&
            (identical(other.total, total) || other.total == total) &&
            (identical(other.maxPostNo, maxPostNo) ||
                other.maxPostNo == maxPostNo));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_posts),
    const DeepCollectionEquality().hash(_replyTargets),
    anchorPostId,
    beforePostNo,
    afterPostNo,
    hasBefore,
    hasAfter,
    total,
    maxPostNo,
  );

  /// Create a copy of PostWindowPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PostWindowPayloadImplCopyWith<_$PostWindowPayloadImpl> get copyWith =>
      __$$PostWindowPayloadImplCopyWithImpl<_$PostWindowPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$PostWindowPayloadImplToJson(this);
  }
}

abstract class _PostWindowPayload implements PostWindowPayload {
  const factory _PostWindowPayload({
    required final List<PostPayload> posts,
    required final List<ReplyTargetPayload> replyTargets,
    final int? anchorPostId,
    final int? beforePostNo,
    final int? afterPostNo,
    required final bool hasBefore,
    required final bool hasAfter,
    required final int total,
    required final int maxPostNo,
  }) = _$PostWindowPayloadImpl;

  factory _PostWindowPayload.fromJson(Map<String, dynamic> json) =
      _$PostWindowPayloadImpl.fromJson;

  @override
  List<PostPayload> get posts;
  @override
  List<ReplyTargetPayload> get replyTargets;
  @override
  int? get anchorPostId;
  @override
  int? get beforePostNo;
  @override
  int? get afterPostNo;
  @override
  bool get hasBefore;
  @override
  bool get hasAfter;
  @override
  int get total;
  @override
  int get maxPostNo;

  /// Create a copy of PostWindowPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PostWindowPayloadImplCopyWith<_$PostWindowPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TopicDetailProps _$TopicDetailPropsFromJson(Map<String, dynamic> json) {
  return _TopicDetailProps.fromJson(json);
}

/// @nodoc
mixin _$TopicDetailProps {
  TopicDetailPayload get topic => throw _privateConstructorUsedError;
  PostWindowPayload get postStream => throw _privateConstructorUsedError;
  List<TopicPayload> get hotTopics => throw _privateConstructorUsedError;
  TopicDetailPermissions get permissions => throw _privateConstructorUsedError;

  /// Serializes this TopicDetailProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TopicDetailPropsCopyWith<TopicDetailProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TopicDetailPropsCopyWith<$Res> {
  factory $TopicDetailPropsCopyWith(
    TopicDetailProps value,
    $Res Function(TopicDetailProps) then,
  ) = _$TopicDetailPropsCopyWithImpl<$Res, TopicDetailProps>;
  @useResult
  $Res call({
    TopicDetailPayload topic,
    PostWindowPayload postStream,
    List<TopicPayload> hotTopics,
    TopicDetailPermissions permissions,
  });

  $TopicDetailPayloadCopyWith<$Res> get topic;
  $PostWindowPayloadCopyWith<$Res> get postStream;
  $TopicDetailPermissionsCopyWith<$Res> get permissions;
}

/// @nodoc
class _$TopicDetailPropsCopyWithImpl<$Res, $Val extends TopicDetailProps>
    implements $TopicDetailPropsCopyWith<$Res> {
  _$TopicDetailPropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topic = null,
    Object? postStream = null,
    Object? hotTopics = null,
    Object? permissions = null,
  }) {
    return _then(
      _value.copyWith(
            topic: null == topic
                ? _value.topic
                : topic // ignore: cast_nullable_to_non_nullable
                      as TopicDetailPayload,
            postStream: null == postStream
                ? _value.postStream
                : postStream // ignore: cast_nullable_to_non_nullable
                      as PostWindowPayload,
            hotTopics: null == hotTopics
                ? _value.hotTopics
                : hotTopics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            permissions: null == permissions
                ? _value.permissions
                : permissions // ignore: cast_nullable_to_non_nullable
                      as TopicDetailPermissions,
          )
          as $Val,
    );
  }

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $TopicDetailPayloadCopyWith<$Res> get topic {
    return $TopicDetailPayloadCopyWith<$Res>(_value.topic, (value) {
      return _then(_value.copyWith(topic: value) as $Val);
    });
  }

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PostWindowPayloadCopyWith<$Res> get postStream {
    return $PostWindowPayloadCopyWith<$Res>(_value.postStream, (value) {
      return _then(_value.copyWith(postStream: value) as $Val);
    });
  }

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $TopicDetailPermissionsCopyWith<$Res> get permissions {
    return $TopicDetailPermissionsCopyWith<$Res>(_value.permissions, (value) {
      return _then(_value.copyWith(permissions: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$TopicDetailPropsImplCopyWith<$Res>
    implements $TopicDetailPropsCopyWith<$Res> {
  factory _$$TopicDetailPropsImplCopyWith(
    _$TopicDetailPropsImpl value,
    $Res Function(_$TopicDetailPropsImpl) then,
  ) = __$$TopicDetailPropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    TopicDetailPayload topic,
    PostWindowPayload postStream,
    List<TopicPayload> hotTopics,
    TopicDetailPermissions permissions,
  });

  @override
  $TopicDetailPayloadCopyWith<$Res> get topic;
  @override
  $PostWindowPayloadCopyWith<$Res> get postStream;
  @override
  $TopicDetailPermissionsCopyWith<$Res> get permissions;
}

/// @nodoc
class __$$TopicDetailPropsImplCopyWithImpl<$Res>
    extends _$TopicDetailPropsCopyWithImpl<$Res, _$TopicDetailPropsImpl>
    implements _$$TopicDetailPropsImplCopyWith<$Res> {
  __$$TopicDetailPropsImplCopyWithImpl(
    _$TopicDetailPropsImpl _value,
    $Res Function(_$TopicDetailPropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topic = null,
    Object? postStream = null,
    Object? hotTopics = null,
    Object? permissions = null,
  }) {
    return _then(
      _$TopicDetailPropsImpl(
        topic: null == topic
            ? _value.topic
            : topic // ignore: cast_nullable_to_non_nullable
                  as TopicDetailPayload,
        postStream: null == postStream
            ? _value.postStream
            : postStream // ignore: cast_nullable_to_non_nullable
                  as PostWindowPayload,
        hotTopics: null == hotTopics
            ? _value._hotTopics
            : hotTopics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        permissions: null == permissions
            ? _value.permissions
            : permissions // ignore: cast_nullable_to_non_nullable
                  as TopicDetailPermissions,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TopicDetailPropsImpl implements _TopicDetailProps {
  const _$TopicDetailPropsImpl({
    required this.topic,
    required this.postStream,
    required final List<TopicPayload> hotTopics,
    required this.permissions,
  }) : _hotTopics = hotTopics;

  factory _$TopicDetailPropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$TopicDetailPropsImplFromJson(json);

  @override
  final TopicDetailPayload topic;
  @override
  final PostWindowPayload postStream;
  final List<TopicPayload> _hotTopics;
  @override
  List<TopicPayload> get hotTopics {
    if (_hotTopics is EqualUnmodifiableListView) return _hotTopics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_hotTopics);
  }

  @override
  final TopicDetailPermissions permissions;

  @override
  String toString() {
    return 'TopicDetailProps(topic: $topic, postStream: $postStream, hotTopics: $hotTopics, permissions: $permissions)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TopicDetailPropsImpl &&
            (identical(other.topic, topic) || other.topic == topic) &&
            (identical(other.postStream, postStream) ||
                other.postStream == postStream) &&
            const DeepCollectionEquality().equals(
              other._hotTopics,
              _hotTopics,
            ) &&
            (identical(other.permissions, permissions) ||
                other.permissions == permissions));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    topic,
    postStream,
    const DeepCollectionEquality().hash(_hotTopics),
    permissions,
  );

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TopicDetailPropsImplCopyWith<_$TopicDetailPropsImpl> get copyWith =>
      __$$TopicDetailPropsImplCopyWithImpl<_$TopicDetailPropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TopicDetailPropsImplToJson(this);
  }
}

abstract class _TopicDetailProps implements TopicDetailProps {
  const factory _TopicDetailProps({
    required final TopicDetailPayload topic,
    required final PostWindowPayload postStream,
    required final List<TopicPayload> hotTopics,
    required final TopicDetailPermissions permissions,
  }) = _$TopicDetailPropsImpl;

  factory _TopicDetailProps.fromJson(Map<String, dynamic> json) =
      _$TopicDetailPropsImpl.fromJson;

  @override
  TopicDetailPayload get topic;
  @override
  PostWindowPayload get postStream;
  @override
  List<TopicPayload> get hotTopics;
  @override
  TopicDetailPermissions get permissions;

  /// Create a copy of TopicDetailProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TopicDetailPropsImplCopyWith<_$TopicDetailPropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TopicDetailPermissions _$TopicDetailPermissionsFromJson(
  Map<String, dynamic> json,
) {
  return _TopicDetailPermissions.fromJson(json);
}

/// @nodoc
mixin _$TopicDetailPermissions {
  bool get isOwnTopic => throw _privateConstructorUsedError;
  bool get canPost => throw _privateConstructorUsedError;
  bool get canModerateTopic => throw _privateConstructorUsedError;

  /// Serializes this TopicDetailPermissions to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TopicDetailPermissions
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TopicDetailPermissionsCopyWith<TopicDetailPermissions> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TopicDetailPermissionsCopyWith<$Res> {
  factory $TopicDetailPermissionsCopyWith(
    TopicDetailPermissions value,
    $Res Function(TopicDetailPermissions) then,
  ) = _$TopicDetailPermissionsCopyWithImpl<$Res, TopicDetailPermissions>;
  @useResult
  $Res call({bool isOwnTopic, bool canPost, bool canModerateTopic});
}

/// @nodoc
class _$TopicDetailPermissionsCopyWithImpl<
  $Res,
  $Val extends TopicDetailPermissions
>
    implements $TopicDetailPermissionsCopyWith<$Res> {
  _$TopicDetailPermissionsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TopicDetailPermissions
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? isOwnTopic = null,
    Object? canPost = null,
    Object? canModerateTopic = null,
  }) {
    return _then(
      _value.copyWith(
            isOwnTopic: null == isOwnTopic
                ? _value.isOwnTopic
                : isOwnTopic // ignore: cast_nullable_to_non_nullable
                      as bool,
            canPost: null == canPost
                ? _value.canPost
                : canPost // ignore: cast_nullable_to_non_nullable
                      as bool,
            canModerateTopic: null == canModerateTopic
                ? _value.canModerateTopic
                : canModerateTopic // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TopicDetailPermissionsImplCopyWith<$Res>
    implements $TopicDetailPermissionsCopyWith<$Res> {
  factory _$$TopicDetailPermissionsImplCopyWith(
    _$TopicDetailPermissionsImpl value,
    $Res Function(_$TopicDetailPermissionsImpl) then,
  ) = __$$TopicDetailPermissionsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool isOwnTopic, bool canPost, bool canModerateTopic});
}

/// @nodoc
class __$$TopicDetailPermissionsImplCopyWithImpl<$Res>
    extends
        _$TopicDetailPermissionsCopyWithImpl<$Res, _$TopicDetailPermissionsImpl>
    implements _$$TopicDetailPermissionsImplCopyWith<$Res> {
  __$$TopicDetailPermissionsImplCopyWithImpl(
    _$TopicDetailPermissionsImpl _value,
    $Res Function(_$TopicDetailPermissionsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TopicDetailPermissions
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? isOwnTopic = null,
    Object? canPost = null,
    Object? canModerateTopic = null,
  }) {
    return _then(
      _$TopicDetailPermissionsImpl(
        isOwnTopic: null == isOwnTopic
            ? _value.isOwnTopic
            : isOwnTopic // ignore: cast_nullable_to_non_nullable
                  as bool,
        canPost: null == canPost
            ? _value.canPost
            : canPost // ignore: cast_nullable_to_non_nullable
                  as bool,
        canModerateTopic: null == canModerateTopic
            ? _value.canModerateTopic
            : canModerateTopic // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TopicDetailPermissionsImpl implements _TopicDetailPermissions {
  const _$TopicDetailPermissionsImpl({
    required this.isOwnTopic,
    required this.canPost,
    required this.canModerateTopic,
  });

  factory _$TopicDetailPermissionsImpl.fromJson(Map<String, dynamic> json) =>
      _$$TopicDetailPermissionsImplFromJson(json);

  @override
  final bool isOwnTopic;
  @override
  final bool canPost;
  @override
  final bool canModerateTopic;

  @override
  String toString() {
    return 'TopicDetailPermissions(isOwnTopic: $isOwnTopic, canPost: $canPost, canModerateTopic: $canModerateTopic)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TopicDetailPermissionsImpl &&
            (identical(other.isOwnTopic, isOwnTopic) ||
                other.isOwnTopic == isOwnTopic) &&
            (identical(other.canPost, canPost) || other.canPost == canPost) &&
            (identical(other.canModerateTopic, canModerateTopic) ||
                other.canModerateTopic == canModerateTopic));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, isOwnTopic, canPost, canModerateTopic);

  /// Create a copy of TopicDetailPermissions
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TopicDetailPermissionsImplCopyWith<_$TopicDetailPermissionsImpl>
  get copyWith =>
      __$$TopicDetailPermissionsImplCopyWithImpl<_$TopicDetailPermissionsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TopicDetailPermissionsImplToJson(this);
  }
}

abstract class _TopicDetailPermissions implements TopicDetailPermissions {
  const factory _TopicDetailPermissions({
    required final bool isOwnTopic,
    required final bool canPost,
    required final bool canModerateTopic,
  }) = _$TopicDetailPermissionsImpl;

  factory _TopicDetailPermissions.fromJson(Map<String, dynamic> json) =
      _$TopicDetailPermissionsImpl.fromJson;

  @override
  bool get isOwnTopic;
  @override
  bool get canPost;
  @override
  bool get canModerateTopic;

  /// Create a copy of TopicDetailPermissions
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TopicDetailPermissionsImplCopyWith<_$TopicDetailPermissionsImpl>
  get copyWith => throw _privateConstructorUsedError;
}
