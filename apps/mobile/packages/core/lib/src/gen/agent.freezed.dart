// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'agent.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

AgentMeResult _$AgentMeResultFromJson(Map<String, dynamic> json) {
  return _AgentMeResult.fromJson(json);
}

/// @nodoc
mixin _$AgentMeResult {
  int get agentId => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get nickname => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get tokenPrefix => throw _privateConstructorUsedError;
  int get enabled => throw _privateConstructorUsedError;
  int get createdAt => throw _privateConstructorUsedError;
  int get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this AgentMeResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentMeResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentMeResultCopyWith<AgentMeResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentMeResultCopyWith<$Res> {
  factory $AgentMeResultCopyWith(
    AgentMeResult value,
    $Res Function(AgentMeResult) then,
  ) = _$AgentMeResultCopyWithImpl<$Res, AgentMeResult>;
  @useResult
  $Res call({
    int agentId,
    String username,
    String nickname,
    String avatarUrl,
    String tokenPrefix,
    int enabled,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class _$AgentMeResultCopyWithImpl<$Res, $Val extends AgentMeResult>
    implements $AgentMeResultCopyWith<$Res> {
  _$AgentMeResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentMeResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? agentId = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? tokenPrefix = null,
    Object? enabled = null,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _value.copyWith(
            agentId: null == agentId
                ? _value.agentId
                : agentId // ignore: cast_nullable_to_non_nullable
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
            tokenPrefix: null == tokenPrefix
                ? _value.tokenPrefix
                : tokenPrefix // ignore: cast_nullable_to_non_nullable
                      as String,
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as int,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as int,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentMeResultImplCopyWith<$Res>
    implements $AgentMeResultCopyWith<$Res> {
  factory _$$AgentMeResultImplCopyWith(
    _$AgentMeResultImpl value,
    $Res Function(_$AgentMeResultImpl) then,
  ) = __$$AgentMeResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int agentId,
    String username,
    String nickname,
    String avatarUrl,
    String tokenPrefix,
    int enabled,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class __$$AgentMeResultImplCopyWithImpl<$Res>
    extends _$AgentMeResultCopyWithImpl<$Res, _$AgentMeResultImpl>
    implements _$$AgentMeResultImplCopyWith<$Res> {
  __$$AgentMeResultImplCopyWithImpl(
    _$AgentMeResultImpl _value,
    $Res Function(_$AgentMeResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentMeResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? agentId = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? tokenPrefix = null,
    Object? enabled = null,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _$AgentMeResultImpl(
        agentId: null == agentId
            ? _value.agentId
            : agentId // ignore: cast_nullable_to_non_nullable
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
        tokenPrefix: null == tokenPrefix
            ? _value.tokenPrefix
            : tokenPrefix // ignore: cast_nullable_to_non_nullable
                  as String,
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as int,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as int,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentMeResultImpl implements _AgentMeResult {
  const _$AgentMeResultImpl({
    required this.agentId,
    required this.username,
    required this.nickname,
    required this.avatarUrl,
    required this.tokenPrefix,
    required this.enabled,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _$AgentMeResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentMeResultImplFromJson(json);

  @override
  final int agentId;
  @override
  final String username;
  @override
  final String nickname;
  @override
  final String avatarUrl;
  @override
  final String tokenPrefix;
  @override
  final int enabled;
  @override
  final int createdAt;
  @override
  final int updatedAt;

  @override
  String toString() {
    return 'AgentMeResult(agentId: $agentId, username: $username, nickname: $nickname, avatarUrl: $avatarUrl, tokenPrefix: $tokenPrefix, enabled: $enabled, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentMeResultImpl &&
            (identical(other.agentId, agentId) || other.agentId == agentId) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.tokenPrefix, tokenPrefix) ||
                other.tokenPrefix == tokenPrefix) &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    agentId,
    username,
    nickname,
    avatarUrl,
    tokenPrefix,
    enabled,
    createdAt,
    updatedAt,
  );

  /// Create a copy of AgentMeResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentMeResultImplCopyWith<_$AgentMeResultImpl> get copyWith =>
      __$$AgentMeResultImplCopyWithImpl<_$AgentMeResultImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentMeResultImplToJson(this);
  }
}

abstract class _AgentMeResult implements AgentMeResult {
  const factory _AgentMeResult({
    required final int agentId,
    required final String username,
    required final String nickname,
    required final String avatarUrl,
    required final String tokenPrefix,
    required final int enabled,
    required final int createdAt,
    required final int updatedAt,
  }) = _$AgentMeResultImpl;

  factory _AgentMeResult.fromJson(Map<String, dynamic> json) =
      _$AgentMeResultImpl.fromJson;

  @override
  int get agentId;
  @override
  String get username;
  @override
  String get nickname;
  @override
  String get avatarUrl;
  @override
  String get tokenPrefix;
  @override
  int get enabled;
  @override
  int get createdAt;
  @override
  int get updatedAt;

  /// Create a copy of AgentMeResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentMeResultImplCopyWith<_$AgentMeResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

AgentTopicItem _$AgentTopicItemFromJson(Map<String, dynamic> json) {
  return _AgentTopicItem.fromJson(json);
}

/// @nodoc
mixin _$AgentTopicItem {
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get excerpt => throw _privateConstructorUsedError;
  List<int> get categoryIds => throw _privateConstructorUsedError;
  int get userId => throw _privateConstructorUsedError;
  int get status => throw _privateConstructorUsedError;
  int get processStatus => throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get viewCount => throw _privateConstructorUsedError;
  int get postCount => throw _privateConstructorUsedError;
  int? get lastPostedAt => throw _privateConstructorUsedError;
  int get createdAt => throw _privateConstructorUsedError;
  int get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this AgentTopicItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentTopicItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentTopicItemCopyWith<AgentTopicItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentTopicItemCopyWith<$Res> {
  factory $AgentTopicItemCopyWith(
    AgentTopicItem value,
    $Res Function(AgentTopicItem) then,
  ) = _$AgentTopicItemCopyWithImpl<$Res, AgentTopicItem>;
  @useResult
  $Res call({
    int id,
    String title,
    String excerpt,
    List<int> categoryIds,
    int userId,
    int status,
    int processStatus,
    int replyCount,
    int viewCount,
    int postCount,
    int? lastPostedAt,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class _$AgentTopicItemCopyWithImpl<$Res, $Val extends AgentTopicItem>
    implements $AgentTopicItemCopyWith<$Res> {
  _$AgentTopicItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentTopicItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? excerpt = null,
    Object? categoryIds = null,
    Object? userId = null,
    Object? status = null,
    Object? processStatus = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? postCount = null,
    Object? lastPostedAt = freezed,
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
            excerpt: null == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String,
            categoryIds: null == categoryIds
                ? _value.categoryIds
                : categoryIds // ignore: cast_nullable_to_non_nullable
                      as List<int>,
            userId: null == userId
                ? _value.userId
                : userId // ignore: cast_nullable_to_non_nullable
                      as int,
            status: null == status
                ? _value.status
                : status // ignore: cast_nullable_to_non_nullable
                      as int,
            processStatus: null == processStatus
                ? _value.processStatus
                : processStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            viewCount: null == viewCount
                ? _value.viewCount
                : viewCount // ignore: cast_nullable_to_non_nullable
                      as int,
            postCount: null == postCount
                ? _value.postCount
                : postCount // ignore: cast_nullable_to_non_nullable
                      as int,
            lastPostedAt: freezed == lastPostedAt
                ? _value.lastPostedAt
                : lastPostedAt // ignore: cast_nullable_to_non_nullable
                      as int?,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as int,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentTopicItemImplCopyWith<$Res>
    implements $AgentTopicItemCopyWith<$Res> {
  factory _$$AgentTopicItemImplCopyWith(
    _$AgentTopicItemImpl value,
    $Res Function(_$AgentTopicItemImpl) then,
  ) = __$$AgentTopicItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String title,
    String excerpt,
    List<int> categoryIds,
    int userId,
    int status,
    int processStatus,
    int replyCount,
    int viewCount,
    int postCount,
    int? lastPostedAt,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class __$$AgentTopicItemImplCopyWithImpl<$Res>
    extends _$AgentTopicItemCopyWithImpl<$Res, _$AgentTopicItemImpl>
    implements _$$AgentTopicItemImplCopyWith<$Res> {
  __$$AgentTopicItemImplCopyWithImpl(
    _$AgentTopicItemImpl _value,
    $Res Function(_$AgentTopicItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentTopicItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? excerpt = null,
    Object? categoryIds = null,
    Object? userId = null,
    Object? status = null,
    Object? processStatus = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? postCount = null,
    Object? lastPostedAt = freezed,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _$AgentTopicItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        excerpt: null == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String,
        categoryIds: null == categoryIds
            ? _value._categoryIds
            : categoryIds // ignore: cast_nullable_to_non_nullable
                  as List<int>,
        userId: null == userId
            ? _value.userId
            : userId // ignore: cast_nullable_to_non_nullable
                  as int,
        status: null == status
            ? _value.status
            : status // ignore: cast_nullable_to_non_nullable
                  as int,
        processStatus: null == processStatus
            ? _value.processStatus
            : processStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        viewCount: null == viewCount
            ? _value.viewCount
            : viewCount // ignore: cast_nullable_to_non_nullable
                  as int,
        postCount: null == postCount
            ? _value.postCount
            : postCount // ignore: cast_nullable_to_non_nullable
                  as int,
        lastPostedAt: freezed == lastPostedAt
            ? _value.lastPostedAt
            : lastPostedAt // ignore: cast_nullable_to_non_nullable
                  as int?,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as int,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentTopicItemImpl implements _AgentTopicItem {
  const _$AgentTopicItemImpl({
    required this.id,
    required this.title,
    required this.excerpt,
    required final List<int> categoryIds,
    required this.userId,
    required this.status,
    required this.processStatus,
    required this.replyCount,
    required this.viewCount,
    required this.postCount,
    this.lastPostedAt,
    required this.createdAt,
    required this.updatedAt,
  }) : _categoryIds = categoryIds;

  factory _$AgentTopicItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentTopicItemImplFromJson(json);

  @override
  final int id;
  @override
  final String title;
  @override
  final String excerpt;
  final List<int> _categoryIds;
  @override
  List<int> get categoryIds {
    if (_categoryIds is EqualUnmodifiableListView) return _categoryIds;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categoryIds);
  }

  @override
  final int userId;
  @override
  final int status;
  @override
  final int processStatus;
  @override
  final int replyCount;
  @override
  final int viewCount;
  @override
  final int postCount;
  @override
  final int? lastPostedAt;
  @override
  final int createdAt;
  @override
  final int updatedAt;

  @override
  String toString() {
    return 'AgentTopicItem(id: $id, title: $title, excerpt: $excerpt, categoryIds: $categoryIds, userId: $userId, status: $status, processStatus: $processStatus, replyCount: $replyCount, viewCount: $viewCount, postCount: $postCount, lastPostedAt: $lastPostedAt, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentTopicItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt) &&
            const DeepCollectionEquality().equals(
              other._categoryIds,
              _categoryIds,
            ) &&
            (identical(other.userId, userId) || other.userId == userId) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.processStatus, processStatus) ||
                other.processStatus == processStatus) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.viewCount, viewCount) ||
                other.viewCount == viewCount) &&
            (identical(other.postCount, postCount) ||
                other.postCount == postCount) &&
            (identical(other.lastPostedAt, lastPostedAt) ||
                other.lastPostedAt == lastPostedAt) &&
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
    excerpt,
    const DeepCollectionEquality().hash(_categoryIds),
    userId,
    status,
    processStatus,
    replyCount,
    viewCount,
    postCount,
    lastPostedAt,
    createdAt,
    updatedAt,
  );

  /// Create a copy of AgentTopicItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentTopicItemImplCopyWith<_$AgentTopicItemImpl> get copyWith =>
      __$$AgentTopicItemImplCopyWithImpl<_$AgentTopicItemImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentTopicItemImplToJson(this);
  }
}

abstract class _AgentTopicItem implements AgentTopicItem {
  const factory _AgentTopicItem({
    required final int id,
    required final String title,
    required final String excerpt,
    required final List<int> categoryIds,
    required final int userId,
    required final int status,
    required final int processStatus,
    required final int replyCount,
    required final int viewCount,
    required final int postCount,
    final int? lastPostedAt,
    required final int createdAt,
    required final int updatedAt,
  }) = _$AgentTopicItemImpl;

  factory _AgentTopicItem.fromJson(Map<String, dynamic> json) =
      _$AgentTopicItemImpl.fromJson;

  @override
  int get id;
  @override
  String get title;
  @override
  String get excerpt;
  @override
  List<int> get categoryIds;
  @override
  int get userId;
  @override
  int get status;
  @override
  int get processStatus;
  @override
  int get replyCount;
  @override
  int get viewCount;
  @override
  int get postCount;
  @override
  int? get lastPostedAt;
  @override
  int get createdAt;
  @override
  int get updatedAt;

  /// Create a copy of AgentTopicItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentTopicItemImplCopyWith<_$AgentTopicItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

AgentTopicListResult _$AgentTopicListResultFromJson(Map<String, dynamic> json) {
  return _AgentTopicListResult.fromJson(json);
}

/// @nodoc
mixin _$AgentTopicListResult {
  List<AgentTopicItem> get list => throw _privateConstructorUsedError;
  int get page => throw _privateConstructorUsedError;
  int get pageSize => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this AgentTopicListResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentTopicListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentTopicListResultCopyWith<AgentTopicListResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentTopicListResultCopyWith<$Res> {
  factory $AgentTopicListResultCopyWith(
    AgentTopicListResult value,
    $Res Function(AgentTopicListResult) then,
  ) = _$AgentTopicListResultCopyWithImpl<$Res, AgentTopicListResult>;
  @useResult
  $Res call({List<AgentTopicItem> list, int page, int pageSize, bool hasNext});
}

/// @nodoc
class _$AgentTopicListResultCopyWithImpl<
  $Res,
  $Val extends AgentTopicListResult
>
    implements $AgentTopicListResultCopyWith<$Res> {
  _$AgentTopicListResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentTopicListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? pageSize = null,
    Object? hasNext = null,
  }) {
    return _then(
      _value.copyWith(
            list: null == list
                ? _value.list
                : list // ignore: cast_nullable_to_non_nullable
                      as List<AgentTopicItem>,
            page: null == page
                ? _value.page
                : page // ignore: cast_nullable_to_non_nullable
                      as int,
            pageSize: null == pageSize
                ? _value.pageSize
                : pageSize // ignore: cast_nullable_to_non_nullable
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
abstract class _$$AgentTopicListResultImplCopyWith<$Res>
    implements $AgentTopicListResultCopyWith<$Res> {
  factory _$$AgentTopicListResultImplCopyWith(
    _$AgentTopicListResultImpl value,
    $Res Function(_$AgentTopicListResultImpl) then,
  ) = __$$AgentTopicListResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<AgentTopicItem> list, int page, int pageSize, bool hasNext});
}

/// @nodoc
class __$$AgentTopicListResultImplCopyWithImpl<$Res>
    extends _$AgentTopicListResultCopyWithImpl<$Res, _$AgentTopicListResultImpl>
    implements _$$AgentTopicListResultImplCopyWith<$Res> {
  __$$AgentTopicListResultImplCopyWithImpl(
    _$AgentTopicListResultImpl _value,
    $Res Function(_$AgentTopicListResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentTopicListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? pageSize = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$AgentTopicListResultImpl(
        list: null == list
            ? _value._list
            : list // ignore: cast_nullable_to_non_nullable
                  as List<AgentTopicItem>,
        page: null == page
            ? _value.page
            : page // ignore: cast_nullable_to_non_nullable
                  as int,
        pageSize: null == pageSize
            ? _value.pageSize
            : pageSize // ignore: cast_nullable_to_non_nullable
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
class _$AgentTopicListResultImpl implements _AgentTopicListResult {
  const _$AgentTopicListResultImpl({
    required final List<AgentTopicItem> list,
    required this.page,
    required this.pageSize,
    required this.hasNext,
  }) : _list = list;

  factory _$AgentTopicListResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentTopicListResultImplFromJson(json);

  final List<AgentTopicItem> _list;
  @override
  List<AgentTopicItem> get list {
    if (_list is EqualUnmodifiableListView) return _list;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_list);
  }

  @override
  final int page;
  @override
  final int pageSize;
  @override
  final bool hasNext;

  @override
  String toString() {
    return 'AgentTopicListResult(list: $list, page: $page, pageSize: $pageSize, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentTopicListResultImpl &&
            const DeepCollectionEquality().equals(other._list, _list) &&
            (identical(other.page, page) || other.page == page) &&
            (identical(other.pageSize, pageSize) ||
                other.pageSize == pageSize) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_list),
    page,
    pageSize,
    hasNext,
  );

  /// Create a copy of AgentTopicListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentTopicListResultImplCopyWith<_$AgentTopicListResultImpl>
  get copyWith =>
      __$$AgentTopicListResultImplCopyWithImpl<_$AgentTopicListResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentTopicListResultImplToJson(this);
  }
}

abstract class _AgentTopicListResult implements AgentTopicListResult {
  const factory _AgentTopicListResult({
    required final List<AgentTopicItem> list,
    required final int page,
    required final int pageSize,
    required final bool hasNext,
  }) = _$AgentTopicListResultImpl;

  factory _AgentTopicListResult.fromJson(Map<String, dynamic> json) =
      _$AgentTopicListResultImpl.fromJson;

  @override
  List<AgentTopicItem> get list;
  @override
  int get page;
  @override
  int get pageSize;
  @override
  bool get hasNext;

  /// Create a copy of AgentTopicListResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentTopicListResultImplCopyWith<_$AgentTopicListResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AgentWriteTopicRequest _$AgentWriteTopicRequestFromJson(
  Map<String, dynamic> json,
) {
  return _AgentWriteTopicRequest.fromJson(json);
}

/// @nodoc
mixin _$AgentWriteTopicRequest {
  String get title => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  List<int> get categoryId => throw _privateConstructorUsedError;

  /// Serializes this AgentWriteTopicRequest to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentWriteTopicRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentWriteTopicRequestCopyWith<AgentWriteTopicRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentWriteTopicRequestCopyWith<$Res> {
  factory $AgentWriteTopicRequestCopyWith(
    AgentWriteTopicRequest value,
    $Res Function(AgentWriteTopicRequest) then,
  ) = _$AgentWriteTopicRequestCopyWithImpl<$Res, AgentWriteTopicRequest>;
  @useResult
  $Res call({String title, String content, List<int> categoryId});
}

/// @nodoc
class _$AgentWriteTopicRequestCopyWithImpl<
  $Res,
  $Val extends AgentWriteTopicRequest
>
    implements $AgentWriteTopicRequestCopyWith<$Res> {
  _$AgentWriteTopicRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentWriteTopicRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? content = null,
    Object? categoryId = null,
  }) {
    return _then(
      _value.copyWith(
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            categoryId: null == categoryId
                ? _value.categoryId
                : categoryId // ignore: cast_nullable_to_non_nullable
                      as List<int>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentWriteTopicRequestImplCopyWith<$Res>
    implements $AgentWriteTopicRequestCopyWith<$Res> {
  factory _$$AgentWriteTopicRequestImplCopyWith(
    _$AgentWriteTopicRequestImpl value,
    $Res Function(_$AgentWriteTopicRequestImpl) then,
  ) = __$$AgentWriteTopicRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String title, String content, List<int> categoryId});
}

/// @nodoc
class __$$AgentWriteTopicRequestImplCopyWithImpl<$Res>
    extends
        _$AgentWriteTopicRequestCopyWithImpl<$Res, _$AgentWriteTopicRequestImpl>
    implements _$$AgentWriteTopicRequestImplCopyWith<$Res> {
  __$$AgentWriteTopicRequestImplCopyWithImpl(
    _$AgentWriteTopicRequestImpl _value,
    $Res Function(_$AgentWriteTopicRequestImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentWriteTopicRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? content = null,
    Object? categoryId = null,
  }) {
    return _then(
      _$AgentWriteTopicRequestImpl(
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        categoryId: null == categoryId
            ? _value._categoryId
            : categoryId // ignore: cast_nullable_to_non_nullable
                  as List<int>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentWriteTopicRequestImpl implements _AgentWriteTopicRequest {
  const _$AgentWriteTopicRequestImpl({
    required this.title,
    required this.content,
    required final List<int> categoryId,
  }) : _categoryId = categoryId;

  factory _$AgentWriteTopicRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentWriteTopicRequestImplFromJson(json);

  @override
  final String title;
  @override
  final String content;
  final List<int> _categoryId;
  @override
  List<int> get categoryId {
    if (_categoryId is EqualUnmodifiableListView) return _categoryId;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categoryId);
  }

  @override
  String toString() {
    return 'AgentWriteTopicRequest(title: $title, content: $content, categoryId: $categoryId)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentWriteTopicRequestImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.content, content) || other.content == content) &&
            const DeepCollectionEquality().equals(
              other._categoryId,
              _categoryId,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    title,
    content,
    const DeepCollectionEquality().hash(_categoryId),
  );

  /// Create a copy of AgentWriteTopicRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentWriteTopicRequestImplCopyWith<_$AgentWriteTopicRequestImpl>
  get copyWith =>
      __$$AgentWriteTopicRequestImplCopyWithImpl<_$AgentWriteTopicRequestImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentWriteTopicRequestImplToJson(this);
  }
}

abstract class _AgentWriteTopicRequest implements AgentWriteTopicRequest {
  const factory _AgentWriteTopicRequest({
    required final String title,
    required final String content,
    required final List<int> categoryId,
  }) = _$AgentWriteTopicRequestImpl;

  factory _AgentWriteTopicRequest.fromJson(Map<String, dynamic> json) =
      _$AgentWriteTopicRequestImpl.fromJson;

  @override
  String get title;
  @override
  String get content;
  @override
  List<int> get categoryId;

  /// Create a copy of AgentWriteTopicRequest
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentWriteTopicRequestImplCopyWith<_$AgentWriteTopicRequestImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AgentCreatePostRequest _$AgentCreatePostRequestFromJson(
  Map<String, dynamic> json,
) {
  return _AgentCreatePostRequest.fromJson(json);
}

/// @nodoc
mixin _$AgentCreatePostRequest {
  String get content => throw _privateConstructorUsedError;
  int? get replyToPostId => throw _privateConstructorUsedError;

  /// Serializes this AgentCreatePostRequest to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentCreatePostRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentCreatePostRequestCopyWith<AgentCreatePostRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentCreatePostRequestCopyWith<$Res> {
  factory $AgentCreatePostRequestCopyWith(
    AgentCreatePostRequest value,
    $Res Function(AgentCreatePostRequest) then,
  ) = _$AgentCreatePostRequestCopyWithImpl<$Res, AgentCreatePostRequest>;
  @useResult
  $Res call({String content, int? replyToPostId});
}

/// @nodoc
class _$AgentCreatePostRequestCopyWithImpl<
  $Res,
  $Val extends AgentCreatePostRequest
>
    implements $AgentCreatePostRequestCopyWith<$Res> {
  _$AgentCreatePostRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentCreatePostRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? content = null, Object? replyToPostId = freezed}) {
    return _then(
      _value.copyWith(
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            replyToPostId: freezed == replyToPostId
                ? _value.replyToPostId
                : replyToPostId // ignore: cast_nullable_to_non_nullable
                      as int?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentCreatePostRequestImplCopyWith<$Res>
    implements $AgentCreatePostRequestCopyWith<$Res> {
  factory _$$AgentCreatePostRequestImplCopyWith(
    _$AgentCreatePostRequestImpl value,
    $Res Function(_$AgentCreatePostRequestImpl) then,
  ) = __$$AgentCreatePostRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String content, int? replyToPostId});
}

/// @nodoc
class __$$AgentCreatePostRequestImplCopyWithImpl<$Res>
    extends
        _$AgentCreatePostRequestCopyWithImpl<$Res, _$AgentCreatePostRequestImpl>
    implements _$$AgentCreatePostRequestImplCopyWith<$Res> {
  __$$AgentCreatePostRequestImplCopyWithImpl(
    _$AgentCreatePostRequestImpl _value,
    $Res Function(_$AgentCreatePostRequestImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentCreatePostRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? content = null, Object? replyToPostId = freezed}) {
    return _then(
      _$AgentCreatePostRequestImpl(
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        replyToPostId: freezed == replyToPostId
            ? _value.replyToPostId
            : replyToPostId // ignore: cast_nullable_to_non_nullable
                  as int?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentCreatePostRequestImpl implements _AgentCreatePostRequest {
  const _$AgentCreatePostRequestImpl({
    required this.content,
    this.replyToPostId,
  });

  factory _$AgentCreatePostRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentCreatePostRequestImplFromJson(json);

  @override
  final String content;
  @override
  final int? replyToPostId;

  @override
  String toString() {
    return 'AgentCreatePostRequest(content: $content, replyToPostId: $replyToPostId)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentCreatePostRequestImpl &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.replyToPostId, replyToPostId) ||
                other.replyToPostId == replyToPostId));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, content, replyToPostId);

  /// Create a copy of AgentCreatePostRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentCreatePostRequestImplCopyWith<_$AgentCreatePostRequestImpl>
  get copyWith =>
      __$$AgentCreatePostRequestImplCopyWithImpl<_$AgentCreatePostRequestImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentCreatePostRequestImplToJson(this);
  }
}

abstract class _AgentCreatePostRequest implements AgentCreatePostRequest {
  const factory _AgentCreatePostRequest({
    required final String content,
    final int? replyToPostId,
  }) = _$AgentCreatePostRequestImpl;

  factory _AgentCreatePostRequest.fromJson(Map<String, dynamic> json) =
      _$AgentCreatePostRequestImpl.fromJson;

  @override
  String get content;
  @override
  int? get replyToPostId;

  /// Create a copy of AgentCreatePostRequest
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentCreatePostRequestImplCopyWith<_$AgentCreatePostRequestImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AgentCreatePostResult _$AgentCreatePostResultFromJson(
  Map<String, dynamic> json,
) {
  return _AgentCreatePostResult.fromJson(json);
}

/// @nodoc
mixin _$AgentCreatePostResult {
  int get id => throw _privateConstructorUsedError;
  int get postNo => throw _privateConstructorUsedError;
  String get renderedContent => throw _privateConstructorUsedError;

  /// Serializes this AgentCreatePostResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentCreatePostResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentCreatePostResultCopyWith<AgentCreatePostResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentCreatePostResultCopyWith<$Res> {
  factory $AgentCreatePostResultCopyWith(
    AgentCreatePostResult value,
    $Res Function(AgentCreatePostResult) then,
  ) = _$AgentCreatePostResultCopyWithImpl<$Res, AgentCreatePostResult>;
  @useResult
  $Res call({int id, int postNo, String renderedContent});
}

/// @nodoc
class _$AgentCreatePostResultCopyWithImpl<
  $Res,
  $Val extends AgentCreatePostResult
>
    implements $AgentCreatePostResultCopyWith<$Res> {
  _$AgentCreatePostResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentCreatePostResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? postNo = null,
    Object? renderedContent = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            postNo: null == postNo
                ? _value.postNo
                : postNo // ignore: cast_nullable_to_non_nullable
                      as int,
            renderedContent: null == renderedContent
                ? _value.renderedContent
                : renderedContent // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentCreatePostResultImplCopyWith<$Res>
    implements $AgentCreatePostResultCopyWith<$Res> {
  factory _$$AgentCreatePostResultImplCopyWith(
    _$AgentCreatePostResultImpl value,
    $Res Function(_$AgentCreatePostResultImpl) then,
  ) = __$$AgentCreatePostResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, int postNo, String renderedContent});
}

/// @nodoc
class __$$AgentCreatePostResultImplCopyWithImpl<$Res>
    extends
        _$AgentCreatePostResultCopyWithImpl<$Res, _$AgentCreatePostResultImpl>
    implements _$$AgentCreatePostResultImplCopyWith<$Res> {
  __$$AgentCreatePostResultImplCopyWithImpl(
    _$AgentCreatePostResultImpl _value,
    $Res Function(_$AgentCreatePostResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentCreatePostResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? postNo = null,
    Object? renderedContent = null,
  }) {
    return _then(
      _$AgentCreatePostResultImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        postNo: null == postNo
            ? _value.postNo
            : postNo // ignore: cast_nullable_to_non_nullable
                  as int,
        renderedContent: null == renderedContent
            ? _value.renderedContent
            : renderedContent // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentCreatePostResultImpl implements _AgentCreatePostResult {
  const _$AgentCreatePostResultImpl({
    required this.id,
    required this.postNo,
    required this.renderedContent,
  });

  factory _$AgentCreatePostResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentCreatePostResultImplFromJson(json);

  @override
  final int id;
  @override
  final int postNo;
  @override
  final String renderedContent;

  @override
  String toString() {
    return 'AgentCreatePostResult(id: $id, postNo: $postNo, renderedContent: $renderedContent)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentCreatePostResultImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.postNo, postNo) || other.postNo == postNo) &&
            (identical(other.renderedContent, renderedContent) ||
                other.renderedContent == renderedContent));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, postNo, renderedContent);

  /// Create a copy of AgentCreatePostResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentCreatePostResultImplCopyWith<_$AgentCreatePostResultImpl>
  get copyWith =>
      __$$AgentCreatePostResultImplCopyWithImpl<_$AgentCreatePostResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentCreatePostResultImplToJson(this);
  }
}

abstract class _AgentCreatePostResult implements AgentCreatePostResult {
  const factory _AgentCreatePostResult({
    required final int id,
    required final int postNo,
    required final String renderedContent,
  }) = _$AgentCreatePostResultImpl;

  factory _AgentCreatePostResult.fromJson(Map<String, dynamic> json) =
      _$AgentCreatePostResultImpl.fromJson;

  @override
  int get id;
  @override
  int get postNo;
  @override
  String get renderedContent;

  /// Create a copy of AgentCreatePostResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentCreatePostResultImplCopyWith<_$AgentCreatePostResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AgentInboxItem _$AgentInboxItemFromJson(Map<String, dynamic> json) {
  return _AgentInboxItem.fromJson(json);
}

/// @nodoc
mixin _$AgentInboxItem {
  int get id => throw _privateConstructorUsedError;
  int get topicId => throw _privateConstructorUsedError;
  int get postId => throw _privateConstructorUsedError;
  String get eventType => throw _privateConstructorUsedError;
  int get actorId => throw _privateConstructorUsedError;
  String get contentPreview => throw _privateConstructorUsedError;
  int get status => throw _privateConstructorUsedError;
  int get deliveryStatus => throw _privateConstructorUsedError;
  int get attempts => throw _privateConstructorUsedError;
  String get lastError => throw _privateConstructorUsedError;
  int? get readAt => throw _privateConstructorUsedError;
  int get createdAt => throw _privateConstructorUsedError;
  int get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this AgentInboxItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentInboxItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentInboxItemCopyWith<AgentInboxItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentInboxItemCopyWith<$Res> {
  factory $AgentInboxItemCopyWith(
    AgentInboxItem value,
    $Res Function(AgentInboxItem) then,
  ) = _$AgentInboxItemCopyWithImpl<$Res, AgentInboxItem>;
  @useResult
  $Res call({
    int id,
    int topicId,
    int postId,
    String eventType,
    int actorId,
    String contentPreview,
    int status,
    int deliveryStatus,
    int attempts,
    String lastError,
    int? readAt,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class _$AgentInboxItemCopyWithImpl<$Res, $Val extends AgentInboxItem>
    implements $AgentInboxItemCopyWith<$Res> {
  _$AgentInboxItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentInboxItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? postId = null,
    Object? eventType = null,
    Object? actorId = null,
    Object? contentPreview = null,
    Object? status = null,
    Object? deliveryStatus = null,
    Object? attempts = null,
    Object? lastError = null,
    Object? readAt = freezed,
    Object? createdAt = null,
    Object? updatedAt = null,
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
            postId: null == postId
                ? _value.postId
                : postId // ignore: cast_nullable_to_non_nullable
                      as int,
            eventType: null == eventType
                ? _value.eventType
                : eventType // ignore: cast_nullable_to_non_nullable
                      as String,
            actorId: null == actorId
                ? _value.actorId
                : actorId // ignore: cast_nullable_to_non_nullable
                      as int,
            contentPreview: null == contentPreview
                ? _value.contentPreview
                : contentPreview // ignore: cast_nullable_to_non_nullable
                      as String,
            status: null == status
                ? _value.status
                : status // ignore: cast_nullable_to_non_nullable
                      as int,
            deliveryStatus: null == deliveryStatus
                ? _value.deliveryStatus
                : deliveryStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            attempts: null == attempts
                ? _value.attempts
                : attempts // ignore: cast_nullable_to_non_nullable
                      as int,
            lastError: null == lastError
                ? _value.lastError
                : lastError // ignore: cast_nullable_to_non_nullable
                      as String,
            readAt: freezed == readAt
                ? _value.readAt
                : readAt // ignore: cast_nullable_to_non_nullable
                      as int?,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as int,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentInboxItemImplCopyWith<$Res>
    implements $AgentInboxItemCopyWith<$Res> {
  factory _$$AgentInboxItemImplCopyWith(
    _$AgentInboxItemImpl value,
    $Res Function(_$AgentInboxItemImpl) then,
  ) = __$$AgentInboxItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int topicId,
    int postId,
    String eventType,
    int actorId,
    String contentPreview,
    int status,
    int deliveryStatus,
    int attempts,
    String lastError,
    int? readAt,
    int createdAt,
    int updatedAt,
  });
}

/// @nodoc
class __$$AgentInboxItemImplCopyWithImpl<$Res>
    extends _$AgentInboxItemCopyWithImpl<$Res, _$AgentInboxItemImpl>
    implements _$$AgentInboxItemImplCopyWith<$Res> {
  __$$AgentInboxItemImplCopyWithImpl(
    _$AgentInboxItemImpl _value,
    $Res Function(_$AgentInboxItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentInboxItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? postId = null,
    Object? eventType = null,
    Object? actorId = null,
    Object? contentPreview = null,
    Object? status = null,
    Object? deliveryStatus = null,
    Object? attempts = null,
    Object? lastError = null,
    Object? readAt = freezed,
    Object? createdAt = null,
    Object? updatedAt = null,
  }) {
    return _then(
      _$AgentInboxItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        topicId: null == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int,
        postId: null == postId
            ? _value.postId
            : postId // ignore: cast_nullable_to_non_nullable
                  as int,
        eventType: null == eventType
            ? _value.eventType
            : eventType // ignore: cast_nullable_to_non_nullable
                  as String,
        actorId: null == actorId
            ? _value.actorId
            : actorId // ignore: cast_nullable_to_non_nullable
                  as int,
        contentPreview: null == contentPreview
            ? _value.contentPreview
            : contentPreview // ignore: cast_nullable_to_non_nullable
                  as String,
        status: null == status
            ? _value.status
            : status // ignore: cast_nullable_to_non_nullable
                  as int,
        deliveryStatus: null == deliveryStatus
            ? _value.deliveryStatus
            : deliveryStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        attempts: null == attempts
            ? _value.attempts
            : attempts // ignore: cast_nullable_to_non_nullable
                  as int,
        lastError: null == lastError
            ? _value.lastError
            : lastError // ignore: cast_nullable_to_non_nullable
                  as String,
        readAt: freezed == readAt
            ? _value.readAt
            : readAt // ignore: cast_nullable_to_non_nullable
                  as int?,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as int,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentInboxItemImpl implements _AgentInboxItem {
  const _$AgentInboxItemImpl({
    required this.id,
    required this.topicId,
    required this.postId,
    required this.eventType,
    required this.actorId,
    required this.contentPreview,
    required this.status,
    required this.deliveryStatus,
    required this.attempts,
    required this.lastError,
    this.readAt,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _$AgentInboxItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentInboxItemImplFromJson(json);

  @override
  final int id;
  @override
  final int topicId;
  @override
  final int postId;
  @override
  final String eventType;
  @override
  final int actorId;
  @override
  final String contentPreview;
  @override
  final int status;
  @override
  final int deliveryStatus;
  @override
  final int attempts;
  @override
  final String lastError;
  @override
  final int? readAt;
  @override
  final int createdAt;
  @override
  final int updatedAt;

  @override
  String toString() {
    return 'AgentInboxItem(id: $id, topicId: $topicId, postId: $postId, eventType: $eventType, actorId: $actorId, contentPreview: $contentPreview, status: $status, deliveryStatus: $deliveryStatus, attempts: $attempts, lastError: $lastError, readAt: $readAt, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentInboxItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.postId, postId) || other.postId == postId) &&
            (identical(other.eventType, eventType) ||
                other.eventType == eventType) &&
            (identical(other.actorId, actorId) || other.actorId == actorId) &&
            (identical(other.contentPreview, contentPreview) ||
                other.contentPreview == contentPreview) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.deliveryStatus, deliveryStatus) ||
                other.deliveryStatus == deliveryStatus) &&
            (identical(other.attempts, attempts) ||
                other.attempts == attempts) &&
            (identical(other.lastError, lastError) ||
                other.lastError == lastError) &&
            (identical(other.readAt, readAt) || other.readAt == readAt) &&
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
    topicId,
    postId,
    eventType,
    actorId,
    contentPreview,
    status,
    deliveryStatus,
    attempts,
    lastError,
    readAt,
    createdAt,
    updatedAt,
  );

  /// Create a copy of AgentInboxItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentInboxItemImplCopyWith<_$AgentInboxItemImpl> get copyWith =>
      __$$AgentInboxItemImplCopyWithImpl<_$AgentInboxItemImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentInboxItemImplToJson(this);
  }
}

abstract class _AgentInboxItem implements AgentInboxItem {
  const factory _AgentInboxItem({
    required final int id,
    required final int topicId,
    required final int postId,
    required final String eventType,
    required final int actorId,
    required final String contentPreview,
    required final int status,
    required final int deliveryStatus,
    required final int attempts,
    required final String lastError,
    final int? readAt,
    required final int createdAt,
    required final int updatedAt,
  }) = _$AgentInboxItemImpl;

  factory _AgentInboxItem.fromJson(Map<String, dynamic> json) =
      _$AgentInboxItemImpl.fromJson;

  @override
  int get id;
  @override
  int get topicId;
  @override
  int get postId;
  @override
  String get eventType;
  @override
  int get actorId;
  @override
  String get contentPreview;
  @override
  int get status;
  @override
  int get deliveryStatus;
  @override
  int get attempts;
  @override
  String get lastError;
  @override
  int? get readAt;
  @override
  int get createdAt;
  @override
  int get updatedAt;

  /// Create a copy of AgentInboxItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentInboxItemImplCopyWith<_$AgentInboxItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

AgentInboxListResult _$AgentInboxListResultFromJson(Map<String, dynamic> json) {
  return _AgentInboxListResult.fromJson(json);
}

/// @nodoc
mixin _$AgentInboxListResult {
  List<AgentInboxItem> get list => throw _privateConstructorUsedError;
  int get page => throw _privateConstructorUsedError;
  int get pageSize => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this AgentInboxListResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentInboxListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentInboxListResultCopyWith<AgentInboxListResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentInboxListResultCopyWith<$Res> {
  factory $AgentInboxListResultCopyWith(
    AgentInboxListResult value,
    $Res Function(AgentInboxListResult) then,
  ) = _$AgentInboxListResultCopyWithImpl<$Res, AgentInboxListResult>;
  @useResult
  $Res call({List<AgentInboxItem> list, int page, int pageSize, bool hasNext});
}

/// @nodoc
class _$AgentInboxListResultCopyWithImpl<
  $Res,
  $Val extends AgentInboxListResult
>
    implements $AgentInboxListResultCopyWith<$Res> {
  _$AgentInboxListResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentInboxListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? pageSize = null,
    Object? hasNext = null,
  }) {
    return _then(
      _value.copyWith(
            list: null == list
                ? _value.list
                : list // ignore: cast_nullable_to_non_nullable
                      as List<AgentInboxItem>,
            page: null == page
                ? _value.page
                : page // ignore: cast_nullable_to_non_nullable
                      as int,
            pageSize: null == pageSize
                ? _value.pageSize
                : pageSize // ignore: cast_nullable_to_non_nullable
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
abstract class _$$AgentInboxListResultImplCopyWith<$Res>
    implements $AgentInboxListResultCopyWith<$Res> {
  factory _$$AgentInboxListResultImplCopyWith(
    _$AgentInboxListResultImpl value,
    $Res Function(_$AgentInboxListResultImpl) then,
  ) = __$$AgentInboxListResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<AgentInboxItem> list, int page, int pageSize, bool hasNext});
}

/// @nodoc
class __$$AgentInboxListResultImplCopyWithImpl<$Res>
    extends _$AgentInboxListResultCopyWithImpl<$Res, _$AgentInboxListResultImpl>
    implements _$$AgentInboxListResultImplCopyWith<$Res> {
  __$$AgentInboxListResultImplCopyWithImpl(
    _$AgentInboxListResultImpl _value,
    $Res Function(_$AgentInboxListResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentInboxListResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? page = null,
    Object? pageSize = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$AgentInboxListResultImpl(
        list: null == list
            ? _value._list
            : list // ignore: cast_nullable_to_non_nullable
                  as List<AgentInboxItem>,
        page: null == page
            ? _value.page
            : page // ignore: cast_nullable_to_non_nullable
                  as int,
        pageSize: null == pageSize
            ? _value.pageSize
            : pageSize // ignore: cast_nullable_to_non_nullable
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
class _$AgentInboxListResultImpl implements _AgentInboxListResult {
  const _$AgentInboxListResultImpl({
    required final List<AgentInboxItem> list,
    required this.page,
    required this.pageSize,
    required this.hasNext,
  }) : _list = list;

  factory _$AgentInboxListResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentInboxListResultImplFromJson(json);

  final List<AgentInboxItem> _list;
  @override
  List<AgentInboxItem> get list {
    if (_list is EqualUnmodifiableListView) return _list;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_list);
  }

  @override
  final int page;
  @override
  final int pageSize;
  @override
  final bool hasNext;

  @override
  String toString() {
    return 'AgentInboxListResult(list: $list, page: $page, pageSize: $pageSize, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentInboxListResultImpl &&
            const DeepCollectionEquality().equals(other._list, _list) &&
            (identical(other.page, page) || other.page == page) &&
            (identical(other.pageSize, pageSize) ||
                other.pageSize == pageSize) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_list),
    page,
    pageSize,
    hasNext,
  );

  /// Create a copy of AgentInboxListResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentInboxListResultImplCopyWith<_$AgentInboxListResultImpl>
  get copyWith =>
      __$$AgentInboxListResultImplCopyWithImpl<_$AgentInboxListResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentInboxListResultImplToJson(this);
  }
}

abstract class _AgentInboxListResult implements AgentInboxListResult {
  const factory _AgentInboxListResult({
    required final List<AgentInboxItem> list,
    required final int page,
    required final int pageSize,
    required final bool hasNext,
  }) = _$AgentInboxListResultImpl;

  factory _AgentInboxListResult.fromJson(Map<String, dynamic> json) =
      _$AgentInboxListResultImpl.fromJson;

  @override
  List<AgentInboxItem> get list;
  @override
  int get page;
  @override
  int get pageSize;
  @override
  bool get hasNext;

  /// Create a copy of AgentInboxListResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentInboxListResultImplCopyWith<_$AgentInboxListResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AgentInboxSuccessResult _$AgentInboxSuccessResultFromJson(
  Map<String, dynamic> json,
) {
  return _AgentInboxSuccessResult.fromJson(json);
}

/// @nodoc
mixin _$AgentInboxSuccessResult {
  String get result => throw _privateConstructorUsedError;
  String get messageCode => throw _privateConstructorUsedError;

  /// Serializes this AgentInboxSuccessResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AgentInboxSuccessResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AgentInboxSuccessResultCopyWith<AgentInboxSuccessResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AgentInboxSuccessResultCopyWith<$Res> {
  factory $AgentInboxSuccessResultCopyWith(
    AgentInboxSuccessResult value,
    $Res Function(AgentInboxSuccessResult) then,
  ) = _$AgentInboxSuccessResultCopyWithImpl<$Res, AgentInboxSuccessResult>;
  @useResult
  $Res call({String result, String messageCode});
}

/// @nodoc
class _$AgentInboxSuccessResultCopyWithImpl<
  $Res,
  $Val extends AgentInboxSuccessResult
>
    implements $AgentInboxSuccessResultCopyWith<$Res> {
  _$AgentInboxSuccessResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AgentInboxSuccessResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? result = null, Object? messageCode = null}) {
    return _then(
      _value.copyWith(
            result: null == result
                ? _value.result
                : result // ignore: cast_nullable_to_non_nullable
                      as String,
            messageCode: null == messageCode
                ? _value.messageCode
                : messageCode // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AgentInboxSuccessResultImplCopyWith<$Res>
    implements $AgentInboxSuccessResultCopyWith<$Res> {
  factory _$$AgentInboxSuccessResultImplCopyWith(
    _$AgentInboxSuccessResultImpl value,
    $Res Function(_$AgentInboxSuccessResultImpl) then,
  ) = __$$AgentInboxSuccessResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String result, String messageCode});
}

/// @nodoc
class __$$AgentInboxSuccessResultImplCopyWithImpl<$Res>
    extends
        _$AgentInboxSuccessResultCopyWithImpl<
          $Res,
          _$AgentInboxSuccessResultImpl
        >
    implements _$$AgentInboxSuccessResultImplCopyWith<$Res> {
  __$$AgentInboxSuccessResultImplCopyWithImpl(
    _$AgentInboxSuccessResultImpl _value,
    $Res Function(_$AgentInboxSuccessResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AgentInboxSuccessResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? result = null, Object? messageCode = null}) {
    return _then(
      _$AgentInboxSuccessResultImpl(
        result: null == result
            ? _value.result
            : result // ignore: cast_nullable_to_non_nullable
                  as String,
        messageCode: null == messageCode
            ? _value.messageCode
            : messageCode // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AgentInboxSuccessResultImpl implements _AgentInboxSuccessResult {
  const _$AgentInboxSuccessResultImpl({
    required this.result,
    required this.messageCode,
  });

  factory _$AgentInboxSuccessResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$AgentInboxSuccessResultImplFromJson(json);

  @override
  final String result;
  @override
  final String messageCode;

  @override
  String toString() {
    return 'AgentInboxSuccessResult(result: $result, messageCode: $messageCode)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AgentInboxSuccessResultImpl &&
            (identical(other.result, result) || other.result == result) &&
            (identical(other.messageCode, messageCode) ||
                other.messageCode == messageCode));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, result, messageCode);

  /// Create a copy of AgentInboxSuccessResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AgentInboxSuccessResultImplCopyWith<_$AgentInboxSuccessResultImpl>
  get copyWith =>
      __$$AgentInboxSuccessResultImplCopyWithImpl<
        _$AgentInboxSuccessResultImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$AgentInboxSuccessResultImplToJson(this);
  }
}

abstract class _AgentInboxSuccessResult implements AgentInboxSuccessResult {
  const factory _AgentInboxSuccessResult({
    required final String result,
    required final String messageCode,
  }) = _$AgentInboxSuccessResultImpl;

  factory _AgentInboxSuccessResult.fromJson(Map<String, dynamic> json) =
      _$AgentInboxSuccessResultImpl.fromJson;

  @override
  String get result;
  @override
  String get messageCode;

  /// Create a copy of AgentInboxSuccessResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AgentInboxSuccessResultImplCopyWith<_$AgentInboxSuccessResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}
