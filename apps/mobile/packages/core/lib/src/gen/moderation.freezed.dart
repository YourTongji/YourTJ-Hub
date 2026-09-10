// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'moderation.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

ModerationPageProps _$ModerationPagePropsFromJson(Map<String, dynamic> json) {
  return _ModerationPageProps.fromJson(json);
}

/// @nodoc
mixin _$ModerationPageProps {
  List<TabItemPayload> get categoryTabs => throw _privateConstructorUsedError;
  List<TopicPayload> get topics => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;

  /// Serializes this ModerationPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationPagePropsCopyWith<ModerationPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationPagePropsCopyWith<$Res> {
  factory $ModerationPagePropsCopyWith(
    ModerationPageProps value,
    $Res Function(ModerationPageProps) then,
  ) = _$ModerationPagePropsCopyWithImpl<$Res, ModerationPageProps>;
  @useResult
  $Res call({
    List<TabItemPayload> categoryTabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
  });

  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$ModerationPagePropsCopyWithImpl<$Res, $Val extends ModerationPageProps>
    implements $ModerationPagePropsCopyWith<$Res> {
  _$ModerationPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? categoryTabs = null,
    Object? topics = null,
    Object? pagination = null,
  }) {
    return _then(
      _value.copyWith(
            categoryTabs: null == categoryTabs
                ? _value.categoryTabs
                : categoryTabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
            topics: null == topics
                ? _value.topics
                : topics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of ModerationPageProps
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
abstract class _$$ModerationPagePropsImplCopyWith<$Res>
    implements $ModerationPagePropsCopyWith<$Res> {
  factory _$$ModerationPagePropsImplCopyWith(
    _$ModerationPagePropsImpl value,
    $Res Function(_$ModerationPagePropsImpl) then,
  ) = __$$ModerationPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<TabItemPayload> categoryTabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
  });

  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$ModerationPagePropsImplCopyWithImpl<$Res>
    extends _$ModerationPagePropsCopyWithImpl<$Res, _$ModerationPagePropsImpl>
    implements _$$ModerationPagePropsImplCopyWith<$Res> {
  __$$ModerationPagePropsImplCopyWithImpl(
    _$ModerationPagePropsImpl _value,
    $Res Function(_$ModerationPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? categoryTabs = null,
    Object? topics = null,
    Object? pagination = null,
  }) {
    return _then(
      _$ModerationPagePropsImpl(
        categoryTabs: null == categoryTabs
            ? _value._categoryTabs
            : categoryTabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
        topics: null == topics
            ? _value._topics
            : topics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationPagePropsImpl implements _ModerationPageProps {
  const _$ModerationPagePropsImpl({
    required final List<TabItemPayload> categoryTabs,
    required final List<TopicPayload> topics,
    required this.pagination,
  }) : _categoryTabs = categoryTabs,
       _topics = topics;

  factory _$ModerationPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationPagePropsImplFromJson(json);

  final List<TabItemPayload> _categoryTabs;
  @override
  List<TabItemPayload> get categoryTabs {
    if (_categoryTabs is EqualUnmodifiableListView) return _categoryTabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categoryTabs);
  }

  final List<TopicPayload> _topics;
  @override
  List<TopicPayload> get topics {
    if (_topics is EqualUnmodifiableListView) return _topics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_topics);
  }

  @override
  final PaginationPayload pagination;

  @override
  String toString() {
    return 'ModerationPageProps(categoryTabs: $categoryTabs, topics: $topics, pagination: $pagination)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationPagePropsImpl &&
            const DeepCollectionEquality().equals(
              other._categoryTabs,
              _categoryTabs,
            ) &&
            const DeepCollectionEquality().equals(other._topics, _topics) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_categoryTabs),
    const DeepCollectionEquality().hash(_topics),
    pagination,
  );

  /// Create a copy of ModerationPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationPagePropsImplCopyWith<_$ModerationPagePropsImpl> get copyWith =>
      __$$ModerationPagePropsImplCopyWithImpl<_$ModerationPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationPagePropsImplToJson(this);
  }
}

abstract class _ModerationPageProps implements ModerationPageProps {
  const factory _ModerationPageProps({
    required final List<TabItemPayload> categoryTabs,
    required final List<TopicPayload> topics,
    required final PaginationPayload pagination,
  }) = _$ModerationPagePropsImpl;

  factory _ModerationPageProps.fromJson(Map<String, dynamic> json) =
      _$ModerationPagePropsImpl.fromJson;

  @override
  List<TabItemPayload> get categoryTabs;
  @override
  List<TopicPayload> get topics;
  @override
  PaginationPayload get pagination;

  /// Create a copy of ModerationPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationPagePropsImplCopyWith<_$ModerationPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ModerationLogSubject _$ModerationLogSubjectFromJson(Map<String, dynamic> json) {
  return _ModerationLogSubject.fromJson(json);
}

/// @nodoc
mixin _$ModerationLogSubject {
  String get type => throw _privateConstructorUsedError;
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String? get url => throw _privateConstructorUsedError;
  String? get excerpt => throw _privateConstructorUsedError;

  /// Serializes this ModerationLogSubject to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationLogSubject
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationLogSubjectCopyWith<ModerationLogSubject> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationLogSubjectCopyWith<$Res> {
  factory $ModerationLogSubjectCopyWith(
    ModerationLogSubject value,
    $Res Function(ModerationLogSubject) then,
  ) = _$ModerationLogSubjectCopyWithImpl<$Res, ModerationLogSubject>;
  @useResult
  $Res call({String type, int id, String title, String? url, String? excerpt});
}

/// @nodoc
class _$ModerationLogSubjectCopyWithImpl<
  $Res,
  $Val extends ModerationLogSubject
>
    implements $ModerationLogSubjectCopyWith<$Res> {
  _$ModerationLogSubjectCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationLogSubject
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? type = null,
    Object? id = null,
    Object? title = null,
    Object? url = freezed,
    Object? excerpt = freezed,
  }) {
    return _then(
      _value.copyWith(
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            url: freezed == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String?,
            excerpt: freezed == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ModerationLogSubjectImplCopyWith<$Res>
    implements $ModerationLogSubjectCopyWith<$Res> {
  factory _$$ModerationLogSubjectImplCopyWith(
    _$ModerationLogSubjectImpl value,
    $Res Function(_$ModerationLogSubjectImpl) then,
  ) = __$$ModerationLogSubjectImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String type, int id, String title, String? url, String? excerpt});
}

/// @nodoc
class __$$ModerationLogSubjectImplCopyWithImpl<$Res>
    extends _$ModerationLogSubjectCopyWithImpl<$Res, _$ModerationLogSubjectImpl>
    implements _$$ModerationLogSubjectImplCopyWith<$Res> {
  __$$ModerationLogSubjectImplCopyWithImpl(
    _$ModerationLogSubjectImpl _value,
    $Res Function(_$ModerationLogSubjectImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationLogSubject
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? type = null,
    Object? id = null,
    Object? title = null,
    Object? url = freezed,
    Object? excerpt = freezed,
  }) {
    return _then(
      _$ModerationLogSubjectImpl(
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        url: freezed == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String?,
        excerpt: freezed == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationLogSubjectImpl implements _ModerationLogSubject {
  const _$ModerationLogSubjectImpl({
    required this.type,
    required this.id,
    required this.title,
    this.url,
    this.excerpt,
  });

  factory _$ModerationLogSubjectImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationLogSubjectImplFromJson(json);

  @override
  final String type;
  @override
  final int id;
  @override
  final String title;
  @override
  final String? url;
  @override
  final String? excerpt;

  @override
  String toString() {
    return 'ModerationLogSubject(type: $type, id: $id, title: $title, url: $url, excerpt: $excerpt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationLogSubjectImpl &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, type, id, title, url, excerpt);

  /// Create a copy of ModerationLogSubject
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationLogSubjectImplCopyWith<_$ModerationLogSubjectImpl>
  get copyWith =>
      __$$ModerationLogSubjectImplCopyWithImpl<_$ModerationLogSubjectImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationLogSubjectImplToJson(this);
  }
}

abstract class _ModerationLogSubject implements ModerationLogSubject {
  const factory _ModerationLogSubject({
    required final String type,
    required final int id,
    required final String title,
    final String? url,
    final String? excerpt,
  }) = _$ModerationLogSubjectImpl;

  factory _ModerationLogSubject.fromJson(Map<String, dynamic> json) =
      _$ModerationLogSubjectImpl.fromJson;

  @override
  String get type;
  @override
  int get id;
  @override
  String get title;
  @override
  String? get url;
  @override
  String? get excerpt;

  /// Create a copy of ModerationLogSubject
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationLogSubjectImplCopyWith<_$ModerationLogSubjectImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ModerationLogActor _$ModerationLogActorFromJson(Map<String, dynamic> json) {
  return _ModerationLogActor.fromJson(json);
}

/// @nodoc
mixin _$ModerationLogActor {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;

  /// Serializes this ModerationLogActor to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationLogActor
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationLogActorCopyWith<ModerationLogActor> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationLogActorCopyWith<$Res> {
  factory $ModerationLogActorCopyWith(
    ModerationLogActor value,
    $Res Function(ModerationLogActor) then,
  ) = _$ModerationLogActorCopyWithImpl<$Res, ModerationLogActor>;
  @useResult
  $Res call({int id, String username, String avatarUrl});
}

/// @nodoc
class _$ModerationLogActorCopyWithImpl<$Res, $Val extends ModerationLogActor>
    implements $ModerationLogActorCopyWith<$Res> {
  _$ModerationLogActorCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationLogActor
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? avatarUrl = null,
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
abstract class _$$ModerationLogActorImplCopyWith<$Res>
    implements $ModerationLogActorCopyWith<$Res> {
  factory _$$ModerationLogActorImplCopyWith(
    _$ModerationLogActorImpl value,
    $Res Function(_$ModerationLogActorImpl) then,
  ) = __$$ModerationLogActorImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String username, String avatarUrl});
}

/// @nodoc
class __$$ModerationLogActorImplCopyWithImpl<$Res>
    extends _$ModerationLogActorCopyWithImpl<$Res, _$ModerationLogActorImpl>
    implements _$$ModerationLogActorImplCopyWith<$Res> {
  __$$ModerationLogActorImplCopyWithImpl(
    _$ModerationLogActorImpl _value,
    $Res Function(_$ModerationLogActorImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationLogActor
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? avatarUrl = null,
  }) {
    return _then(
      _$ModerationLogActorImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
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
class _$ModerationLogActorImpl implements _ModerationLogActor {
  const _$ModerationLogActorImpl({
    required this.id,
    required this.username,
    required this.avatarUrl,
  });

  factory _$ModerationLogActorImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationLogActorImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String avatarUrl;

  @override
  String toString() {
    return 'ModerationLogActor(id: $id, username: $username, avatarUrl: $avatarUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationLogActorImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, username, avatarUrl);

  /// Create a copy of ModerationLogActor
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationLogActorImplCopyWith<_$ModerationLogActorImpl> get copyWith =>
      __$$ModerationLogActorImplCopyWithImpl<_$ModerationLogActorImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationLogActorImplToJson(this);
  }
}

abstract class _ModerationLogActor implements ModerationLogActor {
  const factory _ModerationLogActor({
    required final int id,
    required final String username,
    required final String avatarUrl,
  }) = _$ModerationLogActorImpl;

  factory _ModerationLogActor.fromJson(Map<String, dynamic> json) =
      _$ModerationLogActorImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String get avatarUrl;

  /// Create a copy of ModerationLogActor
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationLogActorImplCopyWith<_$ModerationLogActorImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ModerationLogItem _$ModerationLogItemFromJson(Map<String, dynamic> json) {
  return _ModerationLogItem.fromJson(json);
}

/// @nodoc
mixin _$ModerationLogItem {
  int get id => throw _privateConstructorUsedError;
  String get action => throw _privateConstructorUsedError;
  ModerationLogActor get actor => throw _privateConstructorUsedError;
  ModerationLogSubject get subject => throw _privateConstructorUsedError;
  List<CategoryBriefPayload> get categories =>
      throw _privateConstructorUsedError;
  String get messageCode => throw _privateConstructorUsedError;
  Map<String, dynamic> get params => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;

  /// Serializes this ModerationLogItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationLogItemCopyWith<ModerationLogItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationLogItemCopyWith<$Res> {
  factory $ModerationLogItemCopyWith(
    ModerationLogItem value,
    $Res Function(ModerationLogItem) then,
  ) = _$ModerationLogItemCopyWithImpl<$Res, ModerationLogItem>;
  @useResult
  $Res call({
    int id,
    String action,
    ModerationLogActor actor,
    ModerationLogSubject subject,
    List<CategoryBriefPayload> categories,
    String messageCode,
    Map<String, dynamic> params,
    String createdAt,
  });

  $ModerationLogActorCopyWith<$Res> get actor;
  $ModerationLogSubjectCopyWith<$Res> get subject;
}

/// @nodoc
class _$ModerationLogItemCopyWithImpl<$Res, $Val extends ModerationLogItem>
    implements $ModerationLogItemCopyWith<$Res> {
  _$ModerationLogItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? actor = null,
    Object? subject = null,
    Object? categories = null,
    Object? messageCode = null,
    Object? params = null,
    Object? createdAt = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            action: null == action
                ? _value.action
                : action // ignore: cast_nullable_to_non_nullable
                      as String,
            actor: null == actor
                ? _value.actor
                : actor // ignore: cast_nullable_to_non_nullable
                      as ModerationLogActor,
            subject: null == subject
                ? _value.subject
                : subject // ignore: cast_nullable_to_non_nullable
                      as ModerationLogSubject,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryBriefPayload>,
            messageCode: null == messageCode
                ? _value.messageCode
                : messageCode // ignore: cast_nullable_to_non_nullable
                      as String,
            params: null == params
                ? _value.params
                : params // ignore: cast_nullable_to_non_nullable
                      as Map<String, dynamic>,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ModerationLogActorCopyWith<$Res> get actor {
    return $ModerationLogActorCopyWith<$Res>(_value.actor, (value) {
      return _then(_value.copyWith(actor: value) as $Val);
    });
  }

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ModerationLogSubjectCopyWith<$Res> get subject {
    return $ModerationLogSubjectCopyWith<$Res>(_value.subject, (value) {
      return _then(_value.copyWith(subject: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ModerationLogItemImplCopyWith<$Res>
    implements $ModerationLogItemCopyWith<$Res> {
  factory _$$ModerationLogItemImplCopyWith(
    _$ModerationLogItemImpl value,
    $Res Function(_$ModerationLogItemImpl) then,
  ) = __$$ModerationLogItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String action,
    ModerationLogActor actor,
    ModerationLogSubject subject,
    List<CategoryBriefPayload> categories,
    String messageCode,
    Map<String, dynamic> params,
    String createdAt,
  });

  @override
  $ModerationLogActorCopyWith<$Res> get actor;
  @override
  $ModerationLogSubjectCopyWith<$Res> get subject;
}

/// @nodoc
class __$$ModerationLogItemImplCopyWithImpl<$Res>
    extends _$ModerationLogItemCopyWithImpl<$Res, _$ModerationLogItemImpl>
    implements _$$ModerationLogItemImplCopyWith<$Res> {
  __$$ModerationLogItemImplCopyWithImpl(
    _$ModerationLogItemImpl _value,
    $Res Function(_$ModerationLogItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? actor = null,
    Object? subject = null,
    Object? categories = null,
    Object? messageCode = null,
    Object? params = null,
    Object? createdAt = null,
  }) {
    return _then(
      _$ModerationLogItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        action: null == action
            ? _value.action
            : action // ignore: cast_nullable_to_non_nullable
                  as String,
        actor: null == actor
            ? _value.actor
            : actor // ignore: cast_nullable_to_non_nullable
                  as ModerationLogActor,
        subject: null == subject
            ? _value.subject
            : subject // ignore: cast_nullable_to_non_nullable
                  as ModerationLogSubject,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryBriefPayload>,
        messageCode: null == messageCode
            ? _value.messageCode
            : messageCode // ignore: cast_nullable_to_non_nullable
                  as String,
        params: null == params
            ? _value._params
            : params // ignore: cast_nullable_to_non_nullable
                  as Map<String, dynamic>,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationLogItemImpl implements _ModerationLogItem {
  const _$ModerationLogItemImpl({
    required this.id,
    required this.action,
    required this.actor,
    required this.subject,
    required final List<CategoryBriefPayload> categories,
    required this.messageCode,
    required final Map<String, dynamic> params,
    required this.createdAt,
  }) : _categories = categories,
       _params = params;

  factory _$ModerationLogItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationLogItemImplFromJson(json);

  @override
  final int id;
  @override
  final String action;
  @override
  final ModerationLogActor actor;
  @override
  final ModerationLogSubject subject;
  final List<CategoryBriefPayload> _categories;
  @override
  List<CategoryBriefPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final String messageCode;
  final Map<String, dynamic> _params;
  @override
  Map<String, dynamic> get params {
    if (_params is EqualUnmodifiableMapView) return _params;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(_params);
  }

  @override
  final String createdAt;

  @override
  String toString() {
    return 'ModerationLogItem(id: $id, action: $action, actor: $actor, subject: $subject, categories: $categories, messageCode: $messageCode, params: $params, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationLogItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.action, action) || other.action == action) &&
            (identical(other.actor, actor) || other.actor == actor) &&
            (identical(other.subject, subject) || other.subject == subject) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.messageCode, messageCode) ||
                other.messageCode == messageCode) &&
            const DeepCollectionEquality().equals(other._params, _params) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    action,
    actor,
    subject,
    const DeepCollectionEquality().hash(_categories),
    messageCode,
    const DeepCollectionEquality().hash(_params),
    createdAt,
  );

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationLogItemImplCopyWith<_$ModerationLogItemImpl> get copyWith =>
      __$$ModerationLogItemImplCopyWithImpl<_$ModerationLogItemImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationLogItemImplToJson(this);
  }
}

abstract class _ModerationLogItem implements ModerationLogItem {
  const factory _ModerationLogItem({
    required final int id,
    required final String action,
    required final ModerationLogActor actor,
    required final ModerationLogSubject subject,
    required final List<CategoryBriefPayload> categories,
    required final String messageCode,
    required final Map<String, dynamic> params,
    required final String createdAt,
  }) = _$ModerationLogItemImpl;

  factory _ModerationLogItem.fromJson(Map<String, dynamic> json) =
      _$ModerationLogItemImpl.fromJson;

  @override
  int get id;
  @override
  String get action;
  @override
  ModerationLogActor get actor;
  @override
  ModerationLogSubject get subject;
  @override
  List<CategoryBriefPayload> get categories;
  @override
  String get messageCode;
  @override
  Map<String, dynamic> get params;
  @override
  String get createdAt;

  /// Create a copy of ModerationLogItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationLogItemImplCopyWith<_$ModerationLogItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ModerationLogListResponse _$ModerationLogListResponseFromJson(
  Map<String, dynamic> json,
) {
  return _ModerationLogListResponse.fromJson(json);
}

/// @nodoc
mixin _$ModerationLogListResponse {
  List<ModerationLogItem> get items => throw _privateConstructorUsedError;
  int get nextCursor => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this ModerationLogListResponse to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationLogListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationLogListResponseCopyWith<ModerationLogListResponse> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationLogListResponseCopyWith<$Res> {
  factory $ModerationLogListResponseCopyWith(
    ModerationLogListResponse value,
    $Res Function(ModerationLogListResponse) then,
  ) = _$ModerationLogListResponseCopyWithImpl<$Res, ModerationLogListResponse>;
  @useResult
  $Res call({List<ModerationLogItem> items, int nextCursor, bool hasNext});
}

/// @nodoc
class _$ModerationLogListResponseCopyWithImpl<
  $Res,
  $Val extends ModerationLogListResponse
>
    implements $ModerationLogListResponseCopyWith<$Res> {
  _$ModerationLogListResponseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationLogListResponse
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
                      as List<ModerationLogItem>,
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
abstract class _$$ModerationLogListResponseImplCopyWith<$Res>
    implements $ModerationLogListResponseCopyWith<$Res> {
  factory _$$ModerationLogListResponseImplCopyWith(
    _$ModerationLogListResponseImpl value,
    $Res Function(_$ModerationLogListResponseImpl) then,
  ) = __$$ModerationLogListResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<ModerationLogItem> items, int nextCursor, bool hasNext});
}

/// @nodoc
class __$$ModerationLogListResponseImplCopyWithImpl<$Res>
    extends
        _$ModerationLogListResponseCopyWithImpl<
          $Res,
          _$ModerationLogListResponseImpl
        >
    implements _$$ModerationLogListResponseImplCopyWith<$Res> {
  __$$ModerationLogListResponseImplCopyWithImpl(
    _$ModerationLogListResponseImpl _value,
    $Res Function(_$ModerationLogListResponseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationLogListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$ModerationLogListResponseImpl(
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<ModerationLogItem>,
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
class _$ModerationLogListResponseImpl implements _ModerationLogListResponse {
  const _$ModerationLogListResponseImpl({
    required final List<ModerationLogItem> items,
    required this.nextCursor,
    required this.hasNext,
  }) : _items = items;

  factory _$ModerationLogListResponseImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationLogListResponseImplFromJson(json);

  final List<ModerationLogItem> _items;
  @override
  List<ModerationLogItem> get items {
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
    return 'ModerationLogListResponse(items: $items, nextCursor: $nextCursor, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationLogListResponseImpl &&
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

  /// Create a copy of ModerationLogListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationLogListResponseImplCopyWith<_$ModerationLogListResponseImpl>
  get copyWith =>
      __$$ModerationLogListResponseImplCopyWithImpl<
        _$ModerationLogListResponseImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationLogListResponseImplToJson(this);
  }
}

abstract class _ModerationLogListResponse implements ModerationLogListResponse {
  const factory _ModerationLogListResponse({
    required final List<ModerationLogItem> items,
    required final int nextCursor,
    required final bool hasNext,
  }) = _$ModerationLogListResponseImpl;

  factory _ModerationLogListResponse.fromJson(Map<String, dynamic> json) =
      _$ModerationLogListResponseImpl.fromJson;

  @override
  List<ModerationLogItem> get items;
  @override
  int get nextCursor;
  @override
  bool get hasNext;

  /// Create a copy of ModerationLogListResponse
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationLogListResponseImplCopyWith<_$ModerationLogListResponseImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ModerationReportItem _$ModerationReportItemFromJson(Map<String, dynamic> json) {
  return _ModerationReportItem.fromJson(json);
}

/// @nodoc
mixin _$ModerationReportItem {
  int get id => throw _privateConstructorUsedError;
  String get targetType => throw _privateConstructorUsedError;
  int get targetId => throw _privateConstructorUsedError;
  String get targetUrl => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get excerpt => throw _privateConstructorUsedError;
  String get reason => throw _privateConstructorUsedError;
  String get note => throw _privateConstructorUsedError;
  String get status => throw _privateConstructorUsedError;
  String get resolution => throw _privateConstructorUsedError;
  ModerationLogActor get reporter => throw _privateConstructorUsedError;
  ModerationLogActor get handler => throw _privateConstructorUsedError;
  List<CategoryBriefPayload> get categories =>
      throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  String? get handledAt => throw _privateConstructorUsedError;

  /// Serializes this ModerationReportItem to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationReportItemCopyWith<ModerationReportItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationReportItemCopyWith<$Res> {
  factory $ModerationReportItemCopyWith(
    ModerationReportItem value,
    $Res Function(ModerationReportItem) then,
  ) = _$ModerationReportItemCopyWithImpl<$Res, ModerationReportItem>;
  @useResult
  $Res call({
    int id,
    String targetType,
    int targetId,
    String targetUrl,
    String title,
    String excerpt,
    String reason,
    String note,
    String status,
    String resolution,
    ModerationLogActor reporter,
    ModerationLogActor handler,
    List<CategoryBriefPayload> categories,
    String createdAt,
    String? handledAt,
  });

  $ModerationLogActorCopyWith<$Res> get reporter;
  $ModerationLogActorCopyWith<$Res> get handler;
}

/// @nodoc
class _$ModerationReportItemCopyWithImpl<
  $Res,
  $Val extends ModerationReportItem
>
    implements $ModerationReportItemCopyWith<$Res> {
  _$ModerationReportItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? targetType = null,
    Object? targetId = null,
    Object? targetUrl = null,
    Object? title = null,
    Object? excerpt = null,
    Object? reason = null,
    Object? note = null,
    Object? status = null,
    Object? resolution = null,
    Object? reporter = null,
    Object? handler = null,
    Object? categories = null,
    Object? createdAt = null,
    Object? handledAt = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            targetType: null == targetType
                ? _value.targetType
                : targetType // ignore: cast_nullable_to_non_nullable
                      as String,
            targetId: null == targetId
                ? _value.targetId
                : targetId // ignore: cast_nullable_to_non_nullable
                      as int,
            targetUrl: null == targetUrl
                ? _value.targetUrl
                : targetUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            excerpt: null == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String,
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
            reporter: null == reporter
                ? _value.reporter
                : reporter // ignore: cast_nullable_to_non_nullable
                      as ModerationLogActor,
            handler: null == handler
                ? _value.handler
                : handler // ignore: cast_nullable_to_non_nullable
                      as ModerationLogActor,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryBriefPayload>,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            handledAt: freezed == handledAt
                ? _value.handledAt
                : handledAt // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ModerationLogActorCopyWith<$Res> get reporter {
    return $ModerationLogActorCopyWith<$Res>(_value.reporter, (value) {
      return _then(_value.copyWith(reporter: value) as $Val);
    });
  }

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ModerationLogActorCopyWith<$Res> get handler {
    return $ModerationLogActorCopyWith<$Res>(_value.handler, (value) {
      return _then(_value.copyWith(handler: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ModerationReportItemImplCopyWith<$Res>
    implements $ModerationReportItemCopyWith<$Res> {
  factory _$$ModerationReportItemImplCopyWith(
    _$ModerationReportItemImpl value,
    $Res Function(_$ModerationReportItemImpl) then,
  ) = __$$ModerationReportItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String targetType,
    int targetId,
    String targetUrl,
    String title,
    String excerpt,
    String reason,
    String note,
    String status,
    String resolution,
    ModerationLogActor reporter,
    ModerationLogActor handler,
    List<CategoryBriefPayload> categories,
    String createdAt,
    String? handledAt,
  });

  @override
  $ModerationLogActorCopyWith<$Res> get reporter;
  @override
  $ModerationLogActorCopyWith<$Res> get handler;
}

/// @nodoc
class __$$ModerationReportItemImplCopyWithImpl<$Res>
    extends _$ModerationReportItemCopyWithImpl<$Res, _$ModerationReportItemImpl>
    implements _$$ModerationReportItemImplCopyWith<$Res> {
  __$$ModerationReportItemImplCopyWithImpl(
    _$ModerationReportItemImpl _value,
    $Res Function(_$ModerationReportItemImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? targetType = null,
    Object? targetId = null,
    Object? targetUrl = null,
    Object? title = null,
    Object? excerpt = null,
    Object? reason = null,
    Object? note = null,
    Object? status = null,
    Object? resolution = null,
    Object? reporter = null,
    Object? handler = null,
    Object? categories = null,
    Object? createdAt = null,
    Object? handledAt = freezed,
  }) {
    return _then(
      _$ModerationReportItemImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        targetType: null == targetType
            ? _value.targetType
            : targetType // ignore: cast_nullable_to_non_nullable
                  as String,
        targetId: null == targetId
            ? _value.targetId
            : targetId // ignore: cast_nullable_to_non_nullable
                  as int,
        targetUrl: null == targetUrl
            ? _value.targetUrl
            : targetUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        excerpt: null == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String,
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
        reporter: null == reporter
            ? _value.reporter
            : reporter // ignore: cast_nullable_to_non_nullable
                  as ModerationLogActor,
        handler: null == handler
            ? _value.handler
            : handler // ignore: cast_nullable_to_non_nullable
                  as ModerationLogActor,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryBriefPayload>,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        handledAt: freezed == handledAt
            ? _value.handledAt
            : handledAt // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ModerationReportItemImpl implements _ModerationReportItem {
  const _$ModerationReportItemImpl({
    required this.id,
    required this.targetType,
    required this.targetId,
    required this.targetUrl,
    required this.title,
    required this.excerpt,
    required this.reason,
    required this.note,
    required this.status,
    required this.resolution,
    required this.reporter,
    required this.handler,
    required final List<CategoryBriefPayload> categories,
    required this.createdAt,
    this.handledAt,
  }) : _categories = categories;

  factory _$ModerationReportItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$ModerationReportItemImplFromJson(json);

  @override
  final int id;
  @override
  final String targetType;
  @override
  final int targetId;
  @override
  final String targetUrl;
  @override
  final String title;
  @override
  final String excerpt;
  @override
  final String reason;
  @override
  final String note;
  @override
  final String status;
  @override
  final String resolution;
  @override
  final ModerationLogActor reporter;
  @override
  final ModerationLogActor handler;
  final List<CategoryBriefPayload> _categories;
  @override
  List<CategoryBriefPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final String createdAt;
  @override
  final String? handledAt;

  @override
  String toString() {
    return 'ModerationReportItem(id: $id, targetType: $targetType, targetId: $targetId, targetUrl: $targetUrl, title: $title, excerpt: $excerpt, reason: $reason, note: $note, status: $status, resolution: $resolution, reporter: $reporter, handler: $handler, categories: $categories, createdAt: $createdAt, handledAt: $handledAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationReportItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.targetType, targetType) ||
                other.targetType == targetType) &&
            (identical(other.targetId, targetId) ||
                other.targetId == targetId) &&
            (identical(other.targetUrl, targetUrl) ||
                other.targetUrl == targetUrl) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt) &&
            (identical(other.reason, reason) || other.reason == reason) &&
            (identical(other.note, note) || other.note == note) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.resolution, resolution) ||
                other.resolution == resolution) &&
            (identical(other.reporter, reporter) ||
                other.reporter == reporter) &&
            (identical(other.handler, handler) || other.handler == handler) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.handledAt, handledAt) ||
                other.handledAt == handledAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    targetType,
    targetId,
    targetUrl,
    title,
    excerpt,
    reason,
    note,
    status,
    resolution,
    reporter,
    handler,
    const DeepCollectionEquality().hash(_categories),
    createdAt,
    handledAt,
  );

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationReportItemImplCopyWith<_$ModerationReportItemImpl>
  get copyWith =>
      __$$ModerationReportItemImplCopyWithImpl<_$ModerationReportItemImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationReportItemImplToJson(this);
  }
}

abstract class _ModerationReportItem implements ModerationReportItem {
  const factory _ModerationReportItem({
    required final int id,
    required final String targetType,
    required final int targetId,
    required final String targetUrl,
    required final String title,
    required final String excerpt,
    required final String reason,
    required final String note,
    required final String status,
    required final String resolution,
    required final ModerationLogActor reporter,
    required final ModerationLogActor handler,
    required final List<CategoryBriefPayload> categories,
    required final String createdAt,
    final String? handledAt,
  }) = _$ModerationReportItemImpl;

  factory _ModerationReportItem.fromJson(Map<String, dynamic> json) =
      _$ModerationReportItemImpl.fromJson;

  @override
  int get id;
  @override
  String get targetType;
  @override
  int get targetId;
  @override
  String get targetUrl;
  @override
  String get title;
  @override
  String get excerpt;
  @override
  String get reason;
  @override
  String get note;
  @override
  String get status;
  @override
  String get resolution;
  @override
  ModerationLogActor get reporter;
  @override
  ModerationLogActor get handler;
  @override
  List<CategoryBriefPayload> get categories;
  @override
  String get createdAt;
  @override
  String? get handledAt;

  /// Create a copy of ModerationReportItem
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationReportItemImplCopyWith<_$ModerationReportItemImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ModerationReportListResponse _$ModerationReportListResponseFromJson(
  Map<String, dynamic> json,
) {
  return _ModerationReportListResponse.fromJson(json);
}

/// @nodoc
mixin _$ModerationReportListResponse {
  List<ModerationReportItem> get items => throw _privateConstructorUsedError;
  int get nextCursor => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;

  /// Serializes this ModerationReportListResponse to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ModerationReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ModerationReportListResponseCopyWith<ModerationReportListResponse>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ModerationReportListResponseCopyWith<$Res> {
  factory $ModerationReportListResponseCopyWith(
    ModerationReportListResponse value,
    $Res Function(ModerationReportListResponse) then,
  ) =
      _$ModerationReportListResponseCopyWithImpl<
        $Res,
        ModerationReportListResponse
      >;
  @useResult
  $Res call({List<ModerationReportItem> items, int nextCursor, bool hasNext});
}

/// @nodoc
class _$ModerationReportListResponseCopyWithImpl<
  $Res,
  $Val extends ModerationReportListResponse
>
    implements $ModerationReportListResponseCopyWith<$Res> {
  _$ModerationReportListResponseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ModerationReportListResponse
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
                      as List<ModerationReportItem>,
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
abstract class _$$ModerationReportListResponseImplCopyWith<$Res>
    implements $ModerationReportListResponseCopyWith<$Res> {
  factory _$$ModerationReportListResponseImplCopyWith(
    _$ModerationReportListResponseImpl value,
    $Res Function(_$ModerationReportListResponseImpl) then,
  ) = __$$ModerationReportListResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<ModerationReportItem> items, int nextCursor, bool hasNext});
}

/// @nodoc
class __$$ModerationReportListResponseImplCopyWithImpl<$Res>
    extends
        _$ModerationReportListResponseCopyWithImpl<
          $Res,
          _$ModerationReportListResponseImpl
        >
    implements _$$ModerationReportListResponseImplCopyWith<$Res> {
  __$$ModerationReportListResponseImplCopyWithImpl(
    _$ModerationReportListResponseImpl _value,
    $Res Function(_$ModerationReportListResponseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ModerationReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
  }) {
    return _then(
      _$ModerationReportListResponseImpl(
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<ModerationReportItem>,
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
class _$ModerationReportListResponseImpl
    implements _ModerationReportListResponse {
  const _$ModerationReportListResponseImpl({
    required final List<ModerationReportItem> items,
    required this.nextCursor,
    required this.hasNext,
  }) : _items = items;

  factory _$ModerationReportListResponseImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$ModerationReportListResponseImplFromJson(json);

  final List<ModerationReportItem> _items;
  @override
  List<ModerationReportItem> get items {
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
    return 'ModerationReportListResponse(items: $items, nextCursor: $nextCursor, hasNext: $hasNext)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ModerationReportListResponseImpl &&
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

  /// Create a copy of ModerationReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ModerationReportListResponseImplCopyWith<
    _$ModerationReportListResponseImpl
  >
  get copyWith =>
      __$$ModerationReportListResponseImplCopyWithImpl<
        _$ModerationReportListResponseImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ModerationReportListResponseImplToJson(this);
  }
}

abstract class _ModerationReportListResponse
    implements ModerationReportListResponse {
  const factory _ModerationReportListResponse({
    required final List<ModerationReportItem> items,
    required final int nextCursor,
    required final bool hasNext,
  }) = _$ModerationReportListResponseImpl;

  factory _ModerationReportListResponse.fromJson(Map<String, dynamic> json) =
      _$ModerationReportListResponseImpl.fromJson;

  @override
  List<ModerationReportItem> get items;
  @override
  int get nextCursor;
  @override
  bool get hasNext;

  /// Create a copy of ModerationReportListResponse
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ModerationReportListResponseImplCopyWith<
    _$ModerationReportListResponseImpl
  >
  get copyWith => throw _privateConstructorUsedError;
}
