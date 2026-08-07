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
  int get total => throw _privateConstructorUsedError;
  int get usersTotal => throw _privateConstructorUsedError;
  int get categoriesTotal => throw _privateConstructorUsedError;
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
    int total,
    int usersTotal,
    int categoriesTotal,
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
    Object? total = null,
    Object? usersTotal = null,
    Object? categoriesTotal = null,
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
    int total,
    int usersTotal,
    int categoriesTotal,
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
    Object? total = null,
    Object? usersTotal = null,
    Object? categoriesTotal = null,
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
    required this.total,
    required this.usersTotal,
    required this.categoriesTotal,
    required this.totalPages,
    required this.pagination,
    final List<String>? failedScopes,
    this.searchUnavailable,
  }) : _topics = topics,
       _users = users,
       _categories = categories,
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

  @override
  final int total;
  @override
  final int usersTotal;
  @override
  final int categoriesTotal;
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
    return 'SearchPageProps(query: $query, scope: $scope, topics: $topics, users: $users, categories: $categories, total: $total, usersTotal: $usersTotal, categoriesTotal: $categoriesTotal, totalPages: $totalPages, pagination: $pagination, failedScopes: $failedScopes, searchUnavailable: $searchUnavailable)';
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
            (identical(other.total, total) || other.total == total) &&
            (identical(other.usersTotal, usersTotal) ||
                other.usersTotal == usersTotal) &&
            (identical(other.categoriesTotal, categoriesTotal) ||
                other.categoriesTotal == categoriesTotal) &&
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
    total,
    usersTotal,
    categoriesTotal,
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
    required final int total,
    required final int usersTotal,
    required final int categoriesTotal,
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
  int get total;
  @override
  int get usersTotal;
  @override
  int get categoriesTotal;
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
