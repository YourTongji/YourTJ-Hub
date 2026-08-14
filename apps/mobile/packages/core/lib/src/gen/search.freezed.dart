// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'search.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

UserSearchPayload _$UserSearchPayloadFromJson(Map<String, dynamic> json) {
  return _UserSearchPayload.fromJson(json);
}

/// @nodoc
mixin _$UserSearchPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get nickname => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get bio => throw _privateConstructorUsedError;

  /// Serializes this UserSearchPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserSearchPayloadCopyWith<UserSearchPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserSearchPayloadCopyWith<$Res> {
  factory $UserSearchPayloadCopyWith(
    UserSearchPayload value,
    $Res Function(UserSearchPayload) then,
  ) = _$UserSearchPayloadCopyWithImpl<$Res, UserSearchPayload>;
  @useResult
  $Res call({
    int id,
    String username,
    String nickname,
    String avatarUrl,
    String bio,
  });
}

/// @nodoc
class _$UserSearchPayloadCopyWithImpl<$Res, $Val extends UserSearchPayload>
    implements $UserSearchPayloadCopyWith<$Res> {
  _$UserSearchPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? bio = null,
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
            nickname: null == nickname
                ? _value.nickname
                : nickname // ignore: cast_nullable_to_non_nullable
                      as String,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            bio: null == bio
                ? _value.bio
                : bio // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserSearchPayloadImplCopyWith<$Res>
    implements $UserSearchPayloadCopyWith<$Res> {
  factory _$$UserSearchPayloadImplCopyWith(
    _$UserSearchPayloadImpl value,
    $Res Function(_$UserSearchPayloadImpl) then,
  ) = __$$UserSearchPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String username,
    String nickname,
    String avatarUrl,
    String bio,
  });
}

/// @nodoc
class __$$UserSearchPayloadImplCopyWithImpl<$Res>
    extends _$UserSearchPayloadCopyWithImpl<$Res, _$UserSearchPayloadImpl>
    implements _$$UserSearchPayloadImplCopyWith<$Res> {
  __$$UserSearchPayloadImplCopyWithImpl(
    _$UserSearchPayloadImpl _value,
    $Res Function(_$UserSearchPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? bio = null,
  }) {
    return _then(
      _$UserSearchPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String,
        nickname: null == nickname
            ? _value.nickname
            : nickname // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        bio: null == bio
            ? _value.bio
            : bio // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserSearchPayloadImpl implements _UserSearchPayload {
  const _$UserSearchPayloadImpl({
    required this.id,
    required this.username,
    required this.nickname,
    required this.avatarUrl,
    required this.bio,
  });

  factory _$UserSearchPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserSearchPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String nickname;
  @override
  final String avatarUrl;
  @override
  final String bio;

  @override
  String toString() {
    return 'UserSearchPayload(id: $id, username: $username, nickname: $nickname, avatarUrl: $avatarUrl, bio: $bio)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserSearchPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.bio, bio) || other.bio == bio));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, username, nickname, avatarUrl, bio);

  /// Create a copy of UserSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserSearchPayloadImplCopyWith<_$UserSearchPayloadImpl> get copyWith =>
      __$$UserSearchPayloadImplCopyWithImpl<_$UserSearchPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserSearchPayloadImplToJson(this);
  }
}

abstract class _UserSearchPayload implements UserSearchPayload {
  const factory _UserSearchPayload({
    required final int id,
    required final String username,
    required final String nickname,
    required final String avatarUrl,
    required final String bio,
  }) = _$UserSearchPayloadImpl;

  factory _UserSearchPayload.fromJson(Map<String, dynamic> json) =
      _$UserSearchPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String get nickname;
  @override
  String get avatarUrl;
  @override
  String get bio;

  /// Create a copy of UserSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserSearchPayloadImplCopyWith<_$UserSearchPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CategorySearchPayload _$CategorySearchPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CategorySearchPayload.fromJson(json);
}

/// @nodoc
mixin _$CategorySearchPayload {
  int get id => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get slug => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;
  String get desc => throw _privateConstructorUsedError;

  /// Serializes this CategorySearchPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CategorySearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CategorySearchPayloadCopyWith<CategorySearchPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CategorySearchPayloadCopyWith<$Res> {
  factory $CategorySearchPayloadCopyWith(
    CategorySearchPayload value,
    $Res Function(CategorySearchPayload) then,
  ) = _$CategorySearchPayloadCopyWithImpl<$Res, CategorySearchPayload>;
  @useResult
  $Res call({
    int id,
    String name,
    String slug,
    String icon,
    String color,
    String desc,
  });
}

/// @nodoc
class _$CategorySearchPayloadCopyWithImpl<
  $Res,
  $Val extends CategorySearchPayload
>
    implements $CategorySearchPayloadCopyWith<$Res> {
  _$CategorySearchPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CategorySearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? slug = null,
    Object? icon = null,
    Object? color = null,
    Object? desc = null,
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
            slug: null == slug
                ? _value.slug
                : slug // ignore: cast_nullable_to_non_nullable
                      as String,
            icon: null == icon
                ? _value.icon
                : icon // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
            desc: null == desc
                ? _value.desc
                : desc // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CategorySearchPayloadImplCopyWith<$Res>
    implements $CategorySearchPayloadCopyWith<$Res> {
  factory _$$CategorySearchPayloadImplCopyWith(
    _$CategorySearchPayloadImpl value,
    $Res Function(_$CategorySearchPayloadImpl) then,
  ) = __$$CategorySearchPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String name,
    String slug,
    String icon,
    String color,
    String desc,
  });
}

/// @nodoc
class __$$CategorySearchPayloadImplCopyWithImpl<$Res>
    extends
        _$CategorySearchPayloadCopyWithImpl<$Res, _$CategorySearchPayloadImpl>
    implements _$$CategorySearchPayloadImplCopyWith<$Res> {
  __$$CategorySearchPayloadImplCopyWithImpl(
    _$CategorySearchPayloadImpl _value,
    $Res Function(_$CategorySearchPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CategorySearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? slug = null,
    Object? icon = null,
    Object? color = null,
    Object? desc = null,
  }) {
    return _then(
      _$CategorySearchPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        slug: null == slug
            ? _value.slug
            : slug // ignore: cast_nullable_to_non_nullable
                  as String,
        icon: null == icon
            ? _value.icon
            : icon // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
        desc: null == desc
            ? _value.desc
            : desc // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CategorySearchPayloadImpl implements _CategorySearchPayload {
  const _$CategorySearchPayloadImpl({
    required this.id,
    required this.name,
    required this.slug,
    required this.icon,
    required this.color,
    required this.desc,
  });

  factory _$CategorySearchPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CategorySearchPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String name;
  @override
  final String slug;
  @override
  final String icon;
  @override
  final String color;
  @override
  final String desc;

  @override
  String toString() {
    return 'CategorySearchPayload(id: $id, name: $name, slug: $slug, icon: $icon, color: $color, desc: $desc)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CategorySearchPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.slug, slug) || other.slug == slug) &&
            (identical(other.icon, icon) || other.icon == icon) &&
            (identical(other.color, color) || other.color == color) &&
            (identical(other.desc, desc) || other.desc == desc));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, name, slug, icon, color, desc);

  /// Create a copy of CategorySearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CategorySearchPayloadImplCopyWith<_$CategorySearchPayloadImpl>
  get copyWith =>
      __$$CategorySearchPayloadImplCopyWithImpl<_$CategorySearchPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CategorySearchPayloadImplToJson(this);
  }
}

abstract class _CategorySearchPayload implements CategorySearchPayload {
  const factory _CategorySearchPayload({
    required final int id,
    required final String name,
    required final String slug,
    required final String icon,
    required final String color,
    required final String desc,
  }) = _$CategorySearchPayloadImpl;

  factory _CategorySearchPayload.fromJson(Map<String, dynamic> json) =
      _$CategorySearchPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get name;
  @override
  String get slug;
  @override
  String get icon;
  @override
  String get color;
  @override
  String get desc;

  /// Create a copy of CategorySearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CategorySearchPayloadImplCopyWith<_$CategorySearchPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CourseSearchPayload _$CourseSearchPayloadFromJson(Map<String, dynamic> json) {
  return _CourseSearchPayload.fromJson(json);
}

/// @nodoc
mixin _$CourseSearchPayload {
  int get id => throw _privateConstructorUsedError;
  String get primaryCode => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get department => throw _privateConstructorUsedError;
  int get creditX10 => throw _privateConstructorUsedError;
  List<String>? get aliases => throw _privateConstructorUsedError;
  List<String>? get instructors => throw _privateConstructorUsedError;
  List<String>? get terms => throw _privateConstructorUsedError;
  List<String>? get campus => throw _privateConstructorUsedError;
  double? get ratingAvg => throw _privateConstructorUsedError;
  int? get reviewCount => throw _privateConstructorUsedError;

  /// Serializes this CourseSearchPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CourseSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CourseSearchPayloadCopyWith<CourseSearchPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CourseSearchPayloadCopyWith<$Res> {
  factory $CourseSearchPayloadCopyWith(
    CourseSearchPayload value,
    $Res Function(CourseSearchPayload) then,
  ) = _$CourseSearchPayloadCopyWithImpl<$Res, CourseSearchPayload>;
  @useResult
  $Res call({
    int id,
    String primaryCode,
    String name,
    String department,
    int creditX10,
    List<String>? aliases,
    List<String>? instructors,
    List<String>? terms,
    List<String>? campus,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class _$CourseSearchPayloadCopyWithImpl<$Res, $Val extends CourseSearchPayload>
    implements $CourseSearchPayloadCopyWith<$Res> {
  _$CourseSearchPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CourseSearchPayload
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
    Object? terms = freezed,
    Object? campus = freezed,
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
            terms: freezed == terms
                ? _value.terms
                : terms // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            campus: freezed == campus
                ? _value.campus
                : campus // ignore: cast_nullable_to_non_nullable
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
abstract class _$$CourseSearchPayloadImplCopyWith<$Res>
    implements $CourseSearchPayloadCopyWith<$Res> {
  factory _$$CourseSearchPayloadImplCopyWith(
    _$CourseSearchPayloadImpl value,
    $Res Function(_$CourseSearchPayloadImpl) then,
  ) = __$$CourseSearchPayloadImplCopyWithImpl<$Res>;
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
    List<String>? terms,
    List<String>? campus,
    double? ratingAvg,
    int? reviewCount,
  });
}

/// @nodoc
class __$$CourseSearchPayloadImplCopyWithImpl<$Res>
    extends _$CourseSearchPayloadCopyWithImpl<$Res, _$CourseSearchPayloadImpl>
    implements _$$CourseSearchPayloadImplCopyWith<$Res> {
  __$$CourseSearchPayloadImplCopyWithImpl(
    _$CourseSearchPayloadImpl _value,
    $Res Function(_$CourseSearchPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CourseSearchPayload
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
    Object? terms = freezed,
    Object? campus = freezed,
    Object? ratingAvg = freezed,
    Object? reviewCount = freezed,
  }) {
    return _then(
      _$CourseSearchPayloadImpl(
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
        terms: freezed == terms
            ? _value._terms
            : terms // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        campus: freezed == campus
            ? _value._campus
            : campus // ignore: cast_nullable_to_non_nullable
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
class _$CourseSearchPayloadImpl implements _CourseSearchPayload {
  const _$CourseSearchPayloadImpl({
    required this.id,
    required this.primaryCode,
    required this.name,
    required this.department,
    required this.creditX10,
    final List<String>? aliases,
    final List<String>? instructors,
    final List<String>? terms,
    final List<String>? campus,
    this.ratingAvg,
    this.reviewCount,
  }) : _aliases = aliases,
       _instructors = instructors,
       _terms = terms,
       _campus = campus;

  factory _$CourseSearchPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CourseSearchPayloadImplFromJson(json);

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

  final List<String>? _terms;
  @override
  List<String>? get terms {
    final value = _terms;
    if (value == null) return null;
    if (_terms is EqualUnmodifiableListView) return _terms;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<String>? _campus;
  @override
  List<String>? get campus {
    final value = _campus;
    if (value == null) return null;
    if (_campus is EqualUnmodifiableListView) return _campus;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final double? ratingAvg;
  @override
  final int? reviewCount;

  @override
  String toString() {
    return 'CourseSearchPayload(id: $id, primaryCode: $primaryCode, name: $name, department: $department, creditX10: $creditX10, aliases: $aliases, instructors: $instructors, terms: $terms, campus: $campus, ratingAvg: $ratingAvg, reviewCount: $reviewCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CourseSearchPayloadImpl &&
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
            const DeepCollectionEquality().equals(other._terms, _terms) &&
            const DeepCollectionEquality().equals(other._campus, _campus) &&
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
    const DeepCollectionEquality().hash(_terms),
    const DeepCollectionEquality().hash(_campus),
    ratingAvg,
    reviewCount,
  );

  /// Create a copy of CourseSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CourseSearchPayloadImplCopyWith<_$CourseSearchPayloadImpl> get copyWith =>
      __$$CourseSearchPayloadImplCopyWithImpl<_$CourseSearchPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CourseSearchPayloadImplToJson(this);
  }
}

abstract class _CourseSearchPayload implements CourseSearchPayload {
  const factory _CourseSearchPayload({
    required final int id,
    required final String primaryCode,
    required final String name,
    required final String department,
    required final int creditX10,
    final List<String>? aliases,
    final List<String>? instructors,
    final List<String>? terms,
    final List<String>? campus,
    final double? ratingAvg,
    final int? reviewCount,
  }) = _$CourseSearchPayloadImpl;

  factory _CourseSearchPayload.fromJson(Map<String, dynamic> json) =
      _$CourseSearchPayloadImpl.fromJson;

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
  List<String>? get terms;
  @override
  List<String>? get campus;
  @override
  double? get ratingAvg;
  @override
  int? get reviewCount;

  /// Create a copy of CourseSearchPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CourseSearchPayloadImplCopyWith<_$CourseSearchPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SearchPageProps _$SearchPagePropsFromJson(Map<String, dynamic> json) {
  return _SearchPageProps.fromJson(json);
}

/// @nodoc
mixin _$SearchPageProps {
  String get query => throw _privateConstructorUsedError;
  String get scope => throw _privateConstructorUsedError;
  List<TopicPayload> get topics => throw _privateConstructorUsedError;
  List<UserSearchPayload> get users => throw _privateConstructorUsedError;
  List<CategorySearchPayload> get categories =>
      throw _privateConstructorUsedError;
  List<CourseSearchPayload> get courses => throw _privateConstructorUsedError;
  int get total => throw _privateConstructorUsedError;
  int get usersTotal => throw _privateConstructorUsedError;
  int get categoriesTotal => throw _privateConstructorUsedError;
  int get coursesTotal => throw _privateConstructorUsedError;
  int get totalPages => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;
  List<String>? get failedScopes => throw _privateConstructorUsedError;
  bool? get searchUnavailable => throw _privateConstructorUsedError;

  /// Serializes this SearchPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SearchPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SearchPagePropsCopyWith<SearchPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SearchPagePropsCopyWith<$Res> {
  factory $SearchPagePropsCopyWith(
    SearchPageProps value,
    $Res Function(SearchPageProps) then,
  ) = _$SearchPagePropsCopyWithImpl<$Res, SearchPageProps>;
  @useResult
  $Res call({
    String query,
    String scope,
    List<TopicPayload> topics,
    List<UserSearchPayload> users,
    List<CategorySearchPayload> categories,
    List<CourseSearchPayload> courses,
    int total,
    int usersTotal,
    int categoriesTotal,
    int coursesTotal,
    int totalPages,
    PaginationPayload pagination,
    List<String>? failedScopes,
    bool? searchUnavailable,
  });

  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$SearchPagePropsCopyWithImpl<$Res, $Val extends SearchPageProps>
    implements $SearchPagePropsCopyWith<$Res> {
  _$SearchPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SearchPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? query = null,
    Object? scope = null,
    Object? topics = null,
    Object? users = null,
    Object? categories = null,
    Object? courses = null,
    Object? total = null,
    Object? usersTotal = null,
    Object? categoriesTotal = null,
    Object? coursesTotal = null,
    Object? totalPages = null,
    Object? pagination = null,
    Object? failedScopes = freezed,
    Object? searchUnavailable = freezed,
  }) {
    return _then(
      _value.copyWith(
            query: null == query
                ? _value.query
                : query // ignore: cast_nullable_to_non_nullable
                      as String,
            scope: null == scope
                ? _value.scope
                : scope // ignore: cast_nullable_to_non_nullable
                      as String,
            topics: null == topics
                ? _value.topics
                : topics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            users: null == users
                ? _value.users
                : users // ignore: cast_nullable_to_non_nullable
                      as List<UserSearchPayload>,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategorySearchPayload>,
            courses: null == courses
                ? _value.courses
                : courses // ignore: cast_nullable_to_non_nullable
                      as List<CourseSearchPayload>,
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
                      as int,
            usersTotal: null == usersTotal
                ? _value.usersTotal
                : usersTotal // ignore: cast_nullable_to_non_nullable
                      as int,
            categoriesTotal: null == categoriesTotal
                ? _value.categoriesTotal
                : categoriesTotal // ignore: cast_nullable_to_non_nullable
                      as int,
            coursesTotal: null == coursesTotal
                ? _value.coursesTotal
                : coursesTotal // ignore: cast_nullable_to_non_nullable
                      as int,
            totalPages: null == totalPages
                ? _value.totalPages
                : totalPages // ignore: cast_nullable_to_non_nullable
                      as int,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
            failedScopes: freezed == failedScopes
                ? _value.failedScopes
                : failedScopes // ignore: cast_nullable_to_non_nullable
                      as List<String>?,
            searchUnavailable: freezed == searchUnavailable
                ? _value.searchUnavailable
                : searchUnavailable // ignore: cast_nullable_to_non_nullable
                      as bool?,
          )
          as $Val,
    );
  }

  /// Create a copy of SearchPageProps
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
abstract class _$$SearchPagePropsImplCopyWith<$Res>
    implements $SearchPagePropsCopyWith<$Res> {
  factory _$$SearchPagePropsImplCopyWith(
    _$SearchPagePropsImpl value,
    $Res Function(_$SearchPagePropsImpl) then,
  ) = __$$SearchPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String query,
    String scope,
    List<TopicPayload> topics,
    List<UserSearchPayload> users,
    List<CategorySearchPayload> categories,
    List<CourseSearchPayload> courses,
    int total,
    int usersTotal,
    int categoriesTotal,
    int coursesTotal,
    int totalPages,
    PaginationPayload pagination,
    List<String>? failedScopes,
    bool? searchUnavailable,
  });

  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$SearchPagePropsImplCopyWithImpl<$Res>
    extends _$SearchPagePropsCopyWithImpl<$Res, _$SearchPagePropsImpl>
    implements _$$SearchPagePropsImplCopyWith<$Res> {
  __$$SearchPagePropsImplCopyWithImpl(
    _$SearchPagePropsImpl _value,
    $Res Function(_$SearchPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SearchPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? query = null,
    Object? scope = null,
    Object? topics = null,
    Object? users = null,
    Object? categories = null,
    Object? courses = null,
    Object? total = null,
    Object? usersTotal = null,
    Object? categoriesTotal = null,
    Object? coursesTotal = null,
    Object? totalPages = null,
    Object? pagination = null,
    Object? failedScopes = freezed,
    Object? searchUnavailable = freezed,
  }) {
    return _then(
      _$SearchPagePropsImpl(
        query: null == query
            ? _value.query
            : query // ignore: cast_nullable_to_non_nullable
                  as String,
        scope: null == scope
            ? _value.scope
            : scope // ignore: cast_nullable_to_non_nullable
                  as String,
        topics: null == topics
            ? _value._topics
            : topics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        users: null == users
            ? _value._users
            : users // ignore: cast_nullable_to_non_nullable
                  as List<UserSearchPayload>,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategorySearchPayload>,
        courses: null == courses
            ? _value._courses
            : courses // ignore: cast_nullable_to_non_nullable
                  as List<CourseSearchPayload>,
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
                  as int,
        usersTotal: null == usersTotal
            ? _value.usersTotal
            : usersTotal // ignore: cast_nullable_to_non_nullable
                  as int,
        categoriesTotal: null == categoriesTotal
            ? _value.categoriesTotal
            : categoriesTotal // ignore: cast_nullable_to_non_nullable
                  as int,
        coursesTotal: null == coursesTotal
            ? _value.coursesTotal
            : coursesTotal // ignore: cast_nullable_to_non_nullable
                  as int,
        totalPages: null == totalPages
            ? _value.totalPages
            : totalPages // ignore: cast_nullable_to_non_nullable
                  as int,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
        failedScopes: freezed == failedScopes
            ? _value._failedScopes
            : failedScopes // ignore: cast_nullable_to_non_nullable
                  as List<String>?,
        searchUnavailable: freezed == searchUnavailable
            ? _value.searchUnavailable
            : searchUnavailable // ignore: cast_nullable_to_non_nullable
                  as bool?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SearchPagePropsImpl implements _SearchPageProps {
  const _$SearchPagePropsImpl({
    required this.query,
    required this.scope,
    required final List<TopicPayload> topics,
    required final List<UserSearchPayload> users,
    required final List<CategorySearchPayload> categories,
    required final List<CourseSearchPayload> courses,
    required this.total,
    required this.usersTotal,
    required this.categoriesTotal,
    required this.coursesTotal,
    required this.totalPages,
    required this.pagination,
    final List<String>? failedScopes,
    this.searchUnavailable,
  }) : _topics = topics,
       _users = users,
       _categories = categories,
       _courses = courses,
       _failedScopes = failedScopes;

  factory _$SearchPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$SearchPagePropsImplFromJson(json);

  @override
  final String query;
  @override
  final String scope;
  final List<TopicPayload> _topics;
  @override
  List<TopicPayload> get topics {
    if (_topics is EqualUnmodifiableListView) return _topics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_topics);
  }

  final List<UserSearchPayload> _users;
  @override
  List<UserSearchPayload> get users {
    if (_users is EqualUnmodifiableListView) return _users;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_users);
  }

  final List<CategorySearchPayload> _categories;
  @override
  List<CategorySearchPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  final List<CourseSearchPayload> _courses;
  @override
  List<CourseSearchPayload> get courses {
    if (_courses is EqualUnmodifiableListView) return _courses;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_courses);
  }

  @override
  final int total;
  @override
  final int usersTotal;
  @override
  final int categoriesTotal;
  @override
  final int coursesTotal;
  @override
  final int totalPages;
  @override
  final PaginationPayload pagination;
  final List<String>? _failedScopes;
  @override
  List<String>? get failedScopes {
    final value = _failedScopes;
    if (value == null) return null;
    if (_failedScopes is EqualUnmodifiableListView) return _failedScopes;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final bool? searchUnavailable;

  @override
  String toString() {
    return 'SearchPageProps(query: $query, scope: $scope, topics: $topics, users: $users, categories: $categories, courses: $courses, total: $total, usersTotal: $usersTotal, categoriesTotal: $categoriesTotal, coursesTotal: $coursesTotal, totalPages: $totalPages, pagination: $pagination, failedScopes: $failedScopes, searchUnavailable: $searchUnavailable)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SearchPagePropsImpl &&
            (identical(other.query, query) || other.query == query) &&
            (identical(other.scope, scope) || other.scope == scope) &&
            const DeepCollectionEquality().equals(other._topics, _topics) &&
            const DeepCollectionEquality().equals(other._users, _users) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            const DeepCollectionEquality().equals(other._courses, _courses) &&
            (identical(other.total, total) || other.total == total) &&
            (identical(other.usersTotal, usersTotal) ||
                other.usersTotal == usersTotal) &&
            (identical(other.categoriesTotal, categoriesTotal) ||
                other.categoriesTotal == categoriesTotal) &&
            (identical(other.coursesTotal, coursesTotal) ||
                other.coursesTotal == coursesTotal) &&
            (identical(other.totalPages, totalPages) ||
                other.totalPages == totalPages) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination) &&
            const DeepCollectionEquality().equals(
              other._failedScopes,
              _failedScopes,
            ) &&
            (identical(other.searchUnavailable, searchUnavailable) ||
                other.searchUnavailable == searchUnavailable));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    query,
    scope,
    const DeepCollectionEquality().hash(_topics),
    const DeepCollectionEquality().hash(_users),
    const DeepCollectionEquality().hash(_categories),
    const DeepCollectionEquality().hash(_courses),
    total,
    usersTotal,
    categoriesTotal,
    coursesTotal,
    totalPages,
    pagination,
    const DeepCollectionEquality().hash(_failedScopes),
    searchUnavailable,
  );

  /// Create a copy of SearchPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SearchPagePropsImplCopyWith<_$SearchPagePropsImpl> get copyWith =>
      __$$SearchPagePropsImplCopyWithImpl<_$SearchPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SearchPagePropsImplToJson(this);
  }
}

abstract class _SearchPageProps implements SearchPageProps {
  const factory _SearchPageProps({
    required final String query,
    required final String scope,
    required final List<TopicPayload> topics,
    required final List<UserSearchPayload> users,
    required final List<CategorySearchPayload> categories,
    required final List<CourseSearchPayload> courses,
    required final int total,
    required final int usersTotal,
    required final int categoriesTotal,
    required final int coursesTotal,
    required final int totalPages,
    required final PaginationPayload pagination,
    final List<String>? failedScopes,
    final bool? searchUnavailable,
  }) = _$SearchPagePropsImpl;

  factory _SearchPageProps.fromJson(Map<String, dynamic> json) =
      _$SearchPagePropsImpl.fromJson;

  @override
  String get query;
  @override
  String get scope;
  @override
  List<TopicPayload> get topics;
  @override
  List<UserSearchPayload> get users;
  @override
  List<CategorySearchPayload> get categories;
  @override
  List<CourseSearchPayload> get courses;
  @override
  int get total;
  @override
  int get usersTotal;
  @override
  int get categoriesTotal;
  @override
  int get coursesTotal;
  @override
  int get totalPages;
  @override
  PaginationPayload get pagination;
  @override
  List<String>? get failedScopes;
  @override
  bool? get searchUnavailable;

  /// Create a copy of SearchPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SearchPagePropsImplCopyWith<_$SearchPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
