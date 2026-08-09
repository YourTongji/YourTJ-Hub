// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'notification.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

NotificationPayload _$NotificationPayloadFromJson(Map<String, dynamic> json) {
  return _NotificationPayload.fromJson(json);
}

/// @nodoc
mixin _$NotificationPayload {
  int get id => throw _privateConstructorUsedError;
  String get eventType => throw _privateConstructorUsedError;
  bool get isRead => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  NotificationActorPayload get actor => throw _privateConstructorUsedError;
  NotificationTopicPayload? get topic => throw _privateConstructorUsedError;
  NotificationInnerPayload get payload => throw _privateConstructorUsedError;

  /// Serializes this NotificationPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationPayloadCopyWith<NotificationPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationPayloadCopyWith<$Res> {
  factory $NotificationPayloadCopyWith(
    NotificationPayload value,
    $Res Function(NotificationPayload) then,
  ) = _$NotificationPayloadCopyWithImpl<$Res, NotificationPayload>;
  @useResult
  $Res call({
    int id,
    String eventType,
    bool isRead,
    String createdAt,
    String title,
    String content,
    NotificationActorPayload actor,
    NotificationTopicPayload? topic,
    NotificationInnerPayload payload,
  });

  $NotificationActorPayloadCopyWith<$Res> get actor;
  $NotificationTopicPayloadCopyWith<$Res>? get topic;
  $NotificationInnerPayloadCopyWith<$Res> get payload;
}

/// @nodoc
class _$NotificationPayloadCopyWithImpl<$Res, $Val extends NotificationPayload>
    implements $NotificationPayloadCopyWith<$Res> {
  _$NotificationPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? eventType = null,
    Object? isRead = null,
    Object? createdAt = null,
    Object? title = null,
    Object? content = null,
    Object? actor = null,
    Object? topic = freezed,
    Object? payload = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            eventType: null == eventType
                ? _value.eventType
                : eventType // ignore: cast_nullable_to_non_nullable
                      as String,
            isRead: null == isRead
                ? _value.isRead
                : isRead // ignore: cast_nullable_to_non_nullable
                      as bool,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            actor: null == actor
                ? _value.actor
                : actor // ignore: cast_nullable_to_non_nullable
                      as NotificationActorPayload,
            topic: freezed == topic
                ? _value.topic
                : topic // ignore: cast_nullable_to_non_nullable
                      as NotificationTopicPayload?,
            payload: null == payload
                ? _value.payload
                : payload // ignore: cast_nullable_to_non_nullable
                      as NotificationInnerPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $NotificationActorPayloadCopyWith<$Res> get actor {
    return $NotificationActorPayloadCopyWith<$Res>(_value.actor, (value) {
      return _then(_value.copyWith(actor: value) as $Val);
    });
  }

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $NotificationTopicPayloadCopyWith<$Res>? get topic {
    if (_value.topic == null) {
      return null;
    }

    return $NotificationTopicPayloadCopyWith<$Res>(_value.topic!, (value) {
      return _then(_value.copyWith(topic: value) as $Val);
    });
  }

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $NotificationInnerPayloadCopyWith<$Res> get payload {
    return $NotificationInnerPayloadCopyWith<$Res>(_value.payload, (value) {
      return _then(_value.copyWith(payload: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$NotificationPayloadImplCopyWith<$Res>
    implements $NotificationPayloadCopyWith<$Res> {
  factory _$$NotificationPayloadImplCopyWith(
    _$NotificationPayloadImpl value,
    $Res Function(_$NotificationPayloadImpl) then,
  ) = __$$NotificationPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String eventType,
    bool isRead,
    String createdAt,
    String title,
    String content,
    NotificationActorPayload actor,
    NotificationTopicPayload? topic,
    NotificationInnerPayload payload,
  });

  @override
  $NotificationActorPayloadCopyWith<$Res> get actor;
  @override
  $NotificationTopicPayloadCopyWith<$Res>? get topic;
  @override
  $NotificationInnerPayloadCopyWith<$Res> get payload;
}

/// @nodoc
class __$$NotificationPayloadImplCopyWithImpl<$Res>
    extends _$NotificationPayloadCopyWithImpl<$Res, _$NotificationPayloadImpl>
    implements _$$NotificationPayloadImplCopyWith<$Res> {
  __$$NotificationPayloadImplCopyWithImpl(
    _$NotificationPayloadImpl _value,
    $Res Function(_$NotificationPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? eventType = null,
    Object? isRead = null,
    Object? createdAt = null,
    Object? title = null,
    Object? content = null,
    Object? actor = null,
    Object? topic = freezed,
    Object? payload = null,
  }) {
    return _then(
      _$NotificationPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        eventType: null == eventType
            ? _value.eventType
            : eventType // ignore: cast_nullable_to_non_nullable
                  as String,
        isRead: null == isRead
            ? _value.isRead
            : isRead // ignore: cast_nullable_to_non_nullable
                  as bool,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        actor: null == actor
            ? _value.actor
            : actor // ignore: cast_nullable_to_non_nullable
                  as NotificationActorPayload,
        topic: freezed == topic
            ? _value.topic
            : topic // ignore: cast_nullable_to_non_nullable
                  as NotificationTopicPayload?,
        payload: null == payload
            ? _value.payload
            : payload // ignore: cast_nullable_to_non_nullable
                  as NotificationInnerPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationPayloadImpl implements _NotificationPayload {
  const _$NotificationPayloadImpl({
    required this.id,
    required this.eventType,
    required this.isRead,
    required this.createdAt,
    required this.title,
    required this.content,
    required this.actor,
    this.topic,
    required this.payload,
  });

  factory _$NotificationPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String eventType;
  @override
  final bool isRead;
  @override
  final String createdAt;
  @override
  final String title;
  @override
  final String content;
  @override
  final NotificationActorPayload actor;
  @override
  final NotificationTopicPayload? topic;
  @override
  final NotificationInnerPayload payload;

  @override
  String toString() {
    return 'NotificationPayload(id: $id, eventType: $eventType, isRead: $isRead, createdAt: $createdAt, title: $title, content: $content, actor: $actor, topic: $topic, payload: $payload)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.eventType, eventType) ||
                other.eventType == eventType) &&
            (identical(other.isRead, isRead) || other.isRead == isRead) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.actor, actor) || other.actor == actor) &&
            (identical(other.topic, topic) || other.topic == topic) &&
            (identical(other.payload, payload) || other.payload == payload));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    eventType,
    isRead,
    createdAt,
    title,
    content,
    actor,
    topic,
    payload,
  );

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationPayloadImplCopyWith<_$NotificationPayloadImpl> get copyWith =>
      __$$NotificationPayloadImplCopyWithImpl<_$NotificationPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationPayloadImplToJson(this);
  }
}

abstract class _NotificationPayload implements NotificationPayload {
  const factory _NotificationPayload({
    required final int id,
    required final String eventType,
    required final bool isRead,
    required final String createdAt,
    required final String title,
    required final String content,
    required final NotificationActorPayload actor,
    final NotificationTopicPayload? topic,
    required final NotificationInnerPayload payload,
  }) = _$NotificationPayloadImpl;

  factory _NotificationPayload.fromJson(Map<String, dynamic> json) =
      _$NotificationPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get eventType;
  @override
  bool get isRead;
  @override
  String get createdAt;
  @override
  String get title;
  @override
  String get content;
  @override
  NotificationActorPayload get actor;
  @override
  NotificationTopicPayload? get topic;
  @override
  NotificationInnerPayload get payload;

  /// Create a copy of NotificationPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationPayloadImplCopyWith<_$NotificationPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

NotificationActorPayload _$NotificationActorPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationActorPayload.fromJson(json);
}

/// @nodoc
mixin _$NotificationActorPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String? get avatarUrl => throw _privateConstructorUsedError;

  /// Serializes this NotificationActorPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationActorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationActorPayloadCopyWith<NotificationActorPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationActorPayloadCopyWith<$Res> {
  factory $NotificationActorPayloadCopyWith(
    NotificationActorPayload value,
    $Res Function(NotificationActorPayload) then,
  ) = _$NotificationActorPayloadCopyWithImpl<$Res, NotificationActorPayload>;
  @useResult
  $Res call({int id, String username, String? avatarUrl});
}

/// @nodoc
class _$NotificationActorPayloadCopyWithImpl<
  $Res,
  $Val extends NotificationActorPayload
>
    implements $NotificationActorPayloadCopyWith<$Res> {
  _$NotificationActorPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationActorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? avatarUrl = freezed,
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
            avatarUrl: freezed == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NotificationActorPayloadImplCopyWith<$Res>
    implements $NotificationActorPayloadCopyWith<$Res> {
  factory _$$NotificationActorPayloadImplCopyWith(
    _$NotificationActorPayloadImpl value,
    $Res Function(_$NotificationActorPayloadImpl) then,
  ) = __$$NotificationActorPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String username, String? avatarUrl});
}

/// @nodoc
class __$$NotificationActorPayloadImplCopyWithImpl<$Res>
    extends
        _$NotificationActorPayloadCopyWithImpl<
          $Res,
          _$NotificationActorPayloadImpl
        >
    implements _$$NotificationActorPayloadImplCopyWith<$Res> {
  __$$NotificationActorPayloadImplCopyWithImpl(
    _$NotificationActorPayloadImpl _value,
    $Res Function(_$NotificationActorPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationActorPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? avatarUrl = freezed,
  }) {
    return _then(
      _$NotificationActorPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: freezed == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationActorPayloadImpl implements _NotificationActorPayload {
  const _$NotificationActorPayloadImpl({
    required this.id,
    required this.username,
    this.avatarUrl,
  });

  factory _$NotificationActorPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationActorPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String? avatarUrl;

  @override
  String toString() {
    return 'NotificationActorPayload(id: $id, username: $username, avatarUrl: $avatarUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationActorPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, username, avatarUrl);

  /// Create a copy of NotificationActorPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationActorPayloadImplCopyWith<_$NotificationActorPayloadImpl>
  get copyWith =>
      __$$NotificationActorPayloadImplCopyWithImpl<
        _$NotificationActorPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationActorPayloadImplToJson(this);
  }
}

abstract class _NotificationActorPayload implements NotificationActorPayload {
  const factory _NotificationActorPayload({
    required final int id,
    required final String username,
    final String? avatarUrl,
  }) = _$NotificationActorPayloadImpl;

  factory _NotificationActorPayload.fromJson(Map<String, dynamic> json) =
      _$NotificationActorPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String? get avatarUrl;

  /// Create a copy of NotificationActorPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationActorPayloadImplCopyWith<_$NotificationActorPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationTopicPayload _$NotificationTopicPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationTopicPayload.fromJson(json);
}

/// @nodoc
mixin _$NotificationTopicPayload {
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;

  /// Serializes this NotificationTopicPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationTopicPayloadCopyWith<NotificationTopicPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationTopicPayloadCopyWith<$Res> {
  factory $NotificationTopicPayloadCopyWith(
    NotificationTopicPayload value,
    $Res Function(NotificationTopicPayload) then,
  ) = _$NotificationTopicPayloadCopyWithImpl<$Res, NotificationTopicPayload>;
  @useResult
  $Res call({int id, String title, String url});
}

/// @nodoc
class _$NotificationTopicPayloadCopyWithImpl<
  $Res,
  $Val extends NotificationTopicPayload
>
    implements $NotificationTopicPayloadCopyWith<$Res> {
  _$NotificationTopicPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? title = null, Object? url = null}) {
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
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NotificationTopicPayloadImplCopyWith<$Res>
    implements $NotificationTopicPayloadCopyWith<$Res> {
  factory _$$NotificationTopicPayloadImplCopyWith(
    _$NotificationTopicPayloadImpl value,
    $Res Function(_$NotificationTopicPayloadImpl) then,
  ) = __$$NotificationTopicPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String title, String url});
}

/// @nodoc
class __$$NotificationTopicPayloadImplCopyWithImpl<$Res>
    extends
        _$NotificationTopicPayloadCopyWithImpl<
          $Res,
          _$NotificationTopicPayloadImpl
        >
    implements _$$NotificationTopicPayloadImplCopyWith<$Res> {
  __$$NotificationTopicPayloadImplCopyWithImpl(
    _$NotificationTopicPayloadImpl _value,
    $Res Function(_$NotificationTopicPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? title = null, Object? url = null}) {
    return _then(
      _$NotificationTopicPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationTopicPayloadImpl implements _NotificationTopicPayload {
  const _$NotificationTopicPayloadImpl({
    required this.id,
    required this.title,
    required this.url,
  });

  factory _$NotificationTopicPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationTopicPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String title;
  @override
  final String url;

  @override
  String toString() {
    return 'NotificationTopicPayload(id: $id, title: $title, url: $url)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationTopicPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.url, url) || other.url == url));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, title, url);

  /// Create a copy of NotificationTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationTopicPayloadImplCopyWith<_$NotificationTopicPayloadImpl>
  get copyWith =>
      __$$NotificationTopicPayloadImplCopyWithImpl<
        _$NotificationTopicPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationTopicPayloadImplToJson(this);
  }
}

abstract class _NotificationTopicPayload implements NotificationTopicPayload {
  const factory _NotificationTopicPayload({
    required final int id,
    required final String title,
    required final String url,
  }) = _$NotificationTopicPayloadImpl;

  factory _NotificationTopicPayload.fromJson(Map<String, dynamic> json) =
      _$NotificationTopicPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get title;
  @override
  String get url;

  /// Create a copy of NotificationTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationTopicPayloadImplCopyWith<_$NotificationTopicPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationInnerPayload _$NotificationInnerPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationInnerPayload.fromJson(json);
}

/// @nodoc
mixin _$NotificationInnerPayload {
  String? get title => throw _privateConstructorUsedError;
  String? get content => throw _privateConstructorUsedError;
  String? get templateKey => throw _privateConstructorUsedError;
  NotificationTemplateParams? get templateParams =>
      throw _privateConstructorUsedError;
  int get actorId => throw _privateConstructorUsedError;
  String? get actorName => throw _privateConstructorUsedError;
  int? get topicId => throw _privateConstructorUsedError;
  int? get postId => throw _privateConstructorUsedError;
  String? get topicTitle => throw _privateConstructorUsedError;
  NotificationMetadata? get metadata => throw _privateConstructorUsedError;

  /// Serializes this NotificationInnerPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationInnerPayloadCopyWith<NotificationInnerPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationInnerPayloadCopyWith<$Res> {
  factory $NotificationInnerPayloadCopyWith(
    NotificationInnerPayload value,
    $Res Function(NotificationInnerPayload) then,
  ) = _$NotificationInnerPayloadCopyWithImpl<$Res, NotificationInnerPayload>;
  @useResult
  $Res call({
    String? title,
    String? content,
    String? templateKey,
    NotificationTemplateParams? templateParams,
    int actorId,
    String? actorName,
    int? topicId,
    int? postId,
    String? topicTitle,
    NotificationMetadata? metadata,
  });

  $NotificationTemplateParamsCopyWith<$Res>? get templateParams;
  $NotificationMetadataCopyWith<$Res>? get metadata;
}

/// @nodoc
class _$NotificationInnerPayloadCopyWithImpl<
  $Res,
  $Val extends NotificationInnerPayload
>
    implements $NotificationInnerPayloadCopyWith<$Res> {
  _$NotificationInnerPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = freezed,
    Object? content = freezed,
    Object? templateKey = freezed,
    Object? templateParams = freezed,
    Object? actorId = null,
    Object? actorName = freezed,
    Object? topicId = freezed,
    Object? postId = freezed,
    Object? topicTitle = freezed,
    Object? metadata = freezed,
  }) {
    return _then(
      _value.copyWith(
            title: freezed == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String?,
            content: freezed == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String?,
            templateKey: freezed == templateKey
                ? _value.templateKey
                : templateKey // ignore: cast_nullable_to_non_nullable
                      as String?,
            templateParams: freezed == templateParams
                ? _value.templateParams
                : templateParams // ignore: cast_nullable_to_non_nullable
                      as NotificationTemplateParams?,
            actorId: null == actorId
                ? _value.actorId
                : actorId // ignore: cast_nullable_to_non_nullable
                      as int,
            actorName: freezed == actorName
                ? _value.actorName
                : actorName // ignore: cast_nullable_to_non_nullable
                      as String?,
            topicId: freezed == topicId
                ? _value.topicId
                : topicId // ignore: cast_nullable_to_non_nullable
                      as int?,
            postId: freezed == postId
                ? _value.postId
                : postId // ignore: cast_nullable_to_non_nullable
                      as int?,
            topicTitle: freezed == topicTitle
                ? _value.topicTitle
                : topicTitle // ignore: cast_nullable_to_non_nullable
                      as String?,
            metadata: freezed == metadata
                ? _value.metadata
                : metadata // ignore: cast_nullable_to_non_nullable
                      as NotificationMetadata?,
          )
          as $Val,
    );
  }

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $NotificationTemplateParamsCopyWith<$Res>? get templateParams {
    if (_value.templateParams == null) {
      return null;
    }

    return $NotificationTemplateParamsCopyWith<$Res>(_value.templateParams!, (
      value,
    ) {
      return _then(_value.copyWith(templateParams: value) as $Val);
    });
  }

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $NotificationMetadataCopyWith<$Res>? get metadata {
    if (_value.metadata == null) {
      return null;
    }

    return $NotificationMetadataCopyWith<$Res>(_value.metadata!, (value) {
      return _then(_value.copyWith(metadata: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$NotificationInnerPayloadImplCopyWith<$Res>
    implements $NotificationInnerPayloadCopyWith<$Res> {
  factory _$$NotificationInnerPayloadImplCopyWith(
    _$NotificationInnerPayloadImpl value,
    $Res Function(_$NotificationInnerPayloadImpl) then,
  ) = __$$NotificationInnerPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String? title,
    String? content,
    String? templateKey,
    NotificationTemplateParams? templateParams,
    int actorId,
    String? actorName,
    int? topicId,
    int? postId,
    String? topicTitle,
    NotificationMetadata? metadata,
  });

  @override
  $NotificationTemplateParamsCopyWith<$Res>? get templateParams;
  @override
  $NotificationMetadataCopyWith<$Res>? get metadata;
}

/// @nodoc
class __$$NotificationInnerPayloadImplCopyWithImpl<$Res>
    extends
        _$NotificationInnerPayloadCopyWithImpl<
          $Res,
          _$NotificationInnerPayloadImpl
        >
    implements _$$NotificationInnerPayloadImplCopyWith<$Res> {
  __$$NotificationInnerPayloadImplCopyWithImpl(
    _$NotificationInnerPayloadImpl _value,
    $Res Function(_$NotificationInnerPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = freezed,
    Object? content = freezed,
    Object? templateKey = freezed,
    Object? templateParams = freezed,
    Object? actorId = null,
    Object? actorName = freezed,
    Object? topicId = freezed,
    Object? postId = freezed,
    Object? topicTitle = freezed,
    Object? metadata = freezed,
  }) {
    return _then(
      _$NotificationInnerPayloadImpl(
        title: freezed == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String?,
        content: freezed == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String?,
        templateKey: freezed == templateKey
            ? _value.templateKey
            : templateKey // ignore: cast_nullable_to_non_nullable
                  as String?,
        templateParams: freezed == templateParams
            ? _value.templateParams
            : templateParams // ignore: cast_nullable_to_non_nullable
                  as NotificationTemplateParams?,
        actorId: null == actorId
            ? _value.actorId
            : actorId // ignore: cast_nullable_to_non_nullable
                  as int,
        actorName: freezed == actorName
            ? _value.actorName
            : actorName // ignore: cast_nullable_to_non_nullable
                  as String?,
        topicId: freezed == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int?,
        postId: freezed == postId
            ? _value.postId
            : postId // ignore: cast_nullable_to_non_nullable
                  as int?,
        topicTitle: freezed == topicTitle
            ? _value.topicTitle
            : topicTitle // ignore: cast_nullable_to_non_nullable
                  as String?,
        metadata: freezed == metadata
            ? _value.metadata
            : metadata // ignore: cast_nullable_to_non_nullable
                  as NotificationMetadata?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationInnerPayloadImpl implements _NotificationInnerPayload {
  const _$NotificationInnerPayloadImpl({
    this.title,
    this.content,
    this.templateKey,
    this.templateParams,
    required this.actorId,
    this.actorName,
    this.topicId,
    this.postId,
    this.topicTitle,
    this.metadata,
  });

  factory _$NotificationInnerPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationInnerPayloadImplFromJson(json);

  @override
  final String? title;
  @override
  final String? content;
  @override
  final String? templateKey;
  @override
  final NotificationTemplateParams? templateParams;
  @override
  final int actorId;
  @override
  final String? actorName;
  @override
  final int? topicId;
  @override
  final int? postId;
  @override
  final String? topicTitle;
  @override
  final NotificationMetadata? metadata;

  @override
  String toString() {
    return 'NotificationInnerPayload(title: $title, content: $content, templateKey: $templateKey, templateParams: $templateParams, actorId: $actorId, actorName: $actorName, topicId: $topicId, postId: $postId, topicTitle: $topicTitle, metadata: $metadata)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationInnerPayloadImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.templateKey, templateKey) ||
                other.templateKey == templateKey) &&
            (identical(other.templateParams, templateParams) ||
                other.templateParams == templateParams) &&
            (identical(other.actorId, actorId) || other.actorId == actorId) &&
            (identical(other.actorName, actorName) ||
                other.actorName == actorName) &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.postId, postId) || other.postId == postId) &&
            (identical(other.topicTitle, topicTitle) ||
                other.topicTitle == topicTitle) &&
            (identical(other.metadata, metadata) ||
                other.metadata == metadata));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    title,
    content,
    templateKey,
    templateParams,
    actorId,
    actorName,
    topicId,
    postId,
    topicTitle,
    metadata,
  );

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationInnerPayloadImplCopyWith<_$NotificationInnerPayloadImpl>
  get copyWith =>
      __$$NotificationInnerPayloadImplCopyWithImpl<
        _$NotificationInnerPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationInnerPayloadImplToJson(this);
  }
}

abstract class _NotificationInnerPayload implements NotificationInnerPayload {
  const factory _NotificationInnerPayload({
    final String? title,
    final String? content,
    final String? templateKey,
    final NotificationTemplateParams? templateParams,
    required final int actorId,
    final String? actorName,
    final int? topicId,
    final int? postId,
    final String? topicTitle,
    final NotificationMetadata? metadata,
  }) = _$NotificationInnerPayloadImpl;

  factory _NotificationInnerPayload.fromJson(Map<String, dynamic> json) =
      _$NotificationInnerPayloadImpl.fromJson;

  @override
  String? get title;
  @override
  String? get content;
  @override
  String? get templateKey;
  @override
  NotificationTemplateParams? get templateParams;
  @override
  int get actorId;
  @override
  String? get actorName;
  @override
  int? get topicId;
  @override
  int? get postId;
  @override
  String? get topicTitle;
  @override
  NotificationMetadata? get metadata;

  /// Create a copy of NotificationInnerPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationInnerPayloadImplCopyWith<_$NotificationInnerPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationTemplateParams _$NotificationTemplateParamsFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationTemplateParams.fromJson(json);
}

/// @nodoc
mixin _$NotificationTemplateParams {
  String? get preview => throw _privateConstructorUsedError;
  String? get followerName => throw _privateConstructorUsedError;
  String? get badgeCode => throw _privateConstructorUsedError;
  String? get badgeName => throw _privateConstructorUsedError;

  /// Serializes this NotificationTemplateParams to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationTemplateParams
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationTemplateParamsCopyWith<NotificationTemplateParams>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationTemplateParamsCopyWith<$Res> {
  factory $NotificationTemplateParamsCopyWith(
    NotificationTemplateParams value,
    $Res Function(NotificationTemplateParams) then,
  ) =
      _$NotificationTemplateParamsCopyWithImpl<
        $Res,
        NotificationTemplateParams
      >;
  @useResult
  $Res call({
    String? preview,
    String? followerName,
    String? badgeCode,
    String? badgeName,
  });
}

/// @nodoc
class _$NotificationTemplateParamsCopyWithImpl<
  $Res,
  $Val extends NotificationTemplateParams
>
    implements $NotificationTemplateParamsCopyWith<$Res> {
  _$NotificationTemplateParamsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationTemplateParams
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? preview = freezed,
    Object? followerName = freezed,
    Object? badgeCode = freezed,
    Object? badgeName = freezed,
  }) {
    return _then(
      _value.copyWith(
            preview: freezed == preview
                ? _value.preview
                : preview // ignore: cast_nullable_to_non_nullable
                      as String?,
            followerName: freezed == followerName
                ? _value.followerName
                : followerName // ignore: cast_nullable_to_non_nullable
                      as String?,
            badgeCode: freezed == badgeCode
                ? _value.badgeCode
                : badgeCode // ignore: cast_nullable_to_non_nullable
                      as String?,
            badgeName: freezed == badgeName
                ? _value.badgeName
                : badgeName // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NotificationTemplateParamsImplCopyWith<$Res>
    implements $NotificationTemplateParamsCopyWith<$Res> {
  factory _$$NotificationTemplateParamsImplCopyWith(
    _$NotificationTemplateParamsImpl value,
    $Res Function(_$NotificationTemplateParamsImpl) then,
  ) = __$$NotificationTemplateParamsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String? preview,
    String? followerName,
    String? badgeCode,
    String? badgeName,
  });
}

/// @nodoc
class __$$NotificationTemplateParamsImplCopyWithImpl<$Res>
    extends
        _$NotificationTemplateParamsCopyWithImpl<
          $Res,
          _$NotificationTemplateParamsImpl
        >
    implements _$$NotificationTemplateParamsImplCopyWith<$Res> {
  __$$NotificationTemplateParamsImplCopyWithImpl(
    _$NotificationTemplateParamsImpl _value,
    $Res Function(_$NotificationTemplateParamsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationTemplateParams
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? preview = freezed,
    Object? followerName = freezed,
    Object? badgeCode = freezed,
    Object? badgeName = freezed,
  }) {
    return _then(
      _$NotificationTemplateParamsImpl(
        preview: freezed == preview
            ? _value.preview
            : preview // ignore: cast_nullable_to_non_nullable
                  as String?,
        followerName: freezed == followerName
            ? _value.followerName
            : followerName // ignore: cast_nullable_to_non_nullable
                  as String?,
        badgeCode: freezed == badgeCode
            ? _value.badgeCode
            : badgeCode // ignore: cast_nullable_to_non_nullable
                  as String?,
        badgeName: freezed == badgeName
            ? _value.badgeName
            : badgeName // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationTemplateParamsImpl implements _NotificationTemplateParams {
  const _$NotificationTemplateParamsImpl({
    this.preview,
    this.followerName,
    this.badgeCode,
    this.badgeName,
  });

  factory _$NotificationTemplateParamsImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$NotificationTemplateParamsImplFromJson(json);

  @override
  final String? preview;
  @override
  final String? followerName;
  @override
  final String? badgeCode;
  @override
  final String? badgeName;

  @override
  String toString() {
    return 'NotificationTemplateParams(preview: $preview, followerName: $followerName, badgeCode: $badgeCode, badgeName: $badgeName)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationTemplateParamsImpl &&
            (identical(other.preview, preview) || other.preview == preview) &&
            (identical(other.followerName, followerName) ||
                other.followerName == followerName) &&
            (identical(other.badgeCode, badgeCode) ||
                other.badgeCode == badgeCode) &&
            (identical(other.badgeName, badgeName) ||
                other.badgeName == badgeName));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, preview, followerName, badgeCode, badgeName);

  /// Create a copy of NotificationTemplateParams
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationTemplateParamsImplCopyWith<_$NotificationTemplateParamsImpl>
  get copyWith =>
      __$$NotificationTemplateParamsImplCopyWithImpl<
        _$NotificationTemplateParamsImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationTemplateParamsImplToJson(this);
  }
}

abstract class _NotificationTemplateParams
    implements NotificationTemplateParams {
  const factory _NotificationTemplateParams({
    final String? preview,
    final String? followerName,
    final String? badgeCode,
    final String? badgeName,
  }) = _$NotificationTemplateParamsImpl;

  factory _NotificationTemplateParams.fromJson(Map<String, dynamic> json) =
      _$NotificationTemplateParamsImpl.fromJson;

  @override
  String? get preview;
  @override
  String? get followerName;
  @override
  String? get badgeCode;
  @override
  String? get badgeName;

  /// Create a copy of NotificationTemplateParams
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationTemplateParamsImplCopyWith<_$NotificationTemplateParamsImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationMetadata _$NotificationMetadataFromJson(Map<String, dynamic> json) {
  return _NotificationMetadata.fromJson(json);
}

/// @nodoc
mixin _$NotificationMetadata {
  String? get followerName => throw _privateConstructorUsedError;
  String? get badgeCode => throw _privateConstructorUsedError;
  String? get badgeName => throw _privateConstructorUsedError;
  String? get badgeIconUrl => throw _privateConstructorUsedError;
  String? get profileUrl => throw _privateConstructorUsedError;

  /// Serializes this NotificationMetadata to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationMetadata
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationMetadataCopyWith<NotificationMetadata> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationMetadataCopyWith<$Res> {
  factory $NotificationMetadataCopyWith(
    NotificationMetadata value,
    $Res Function(NotificationMetadata) then,
  ) = _$NotificationMetadataCopyWithImpl<$Res, NotificationMetadata>;
  @useResult
  $Res call({
    String? followerName,
    String? badgeCode,
    String? badgeName,
    String? badgeIconUrl,
    String? profileUrl,
  });
}

/// @nodoc
class _$NotificationMetadataCopyWithImpl<
  $Res,
  $Val extends NotificationMetadata
>
    implements $NotificationMetadataCopyWith<$Res> {
  _$NotificationMetadataCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationMetadata
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? followerName = freezed,
    Object? badgeCode = freezed,
    Object? badgeName = freezed,
    Object? badgeIconUrl = freezed,
    Object? profileUrl = freezed,
  }) {
    return _then(
      _value.copyWith(
            followerName: freezed == followerName
                ? _value.followerName
                : followerName // ignore: cast_nullable_to_non_nullable
                      as String?,
            badgeCode: freezed == badgeCode
                ? _value.badgeCode
                : badgeCode // ignore: cast_nullable_to_non_nullable
                      as String?,
            badgeName: freezed == badgeName
                ? _value.badgeName
                : badgeName // ignore: cast_nullable_to_non_nullable
                      as String?,
            badgeIconUrl: freezed == badgeIconUrl
                ? _value.badgeIconUrl
                : badgeIconUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
            profileUrl: freezed == profileUrl
                ? _value.profileUrl
                : profileUrl // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NotificationMetadataImplCopyWith<$Res>
    implements $NotificationMetadataCopyWith<$Res> {
  factory _$$NotificationMetadataImplCopyWith(
    _$NotificationMetadataImpl value,
    $Res Function(_$NotificationMetadataImpl) then,
  ) = __$$NotificationMetadataImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String? followerName,
    String? badgeCode,
    String? badgeName,
    String? badgeIconUrl,
    String? profileUrl,
  });
}

/// @nodoc
class __$$NotificationMetadataImplCopyWithImpl<$Res>
    extends _$NotificationMetadataCopyWithImpl<$Res, _$NotificationMetadataImpl>
    implements _$$NotificationMetadataImplCopyWith<$Res> {
  __$$NotificationMetadataImplCopyWithImpl(
    _$NotificationMetadataImpl _value,
    $Res Function(_$NotificationMetadataImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationMetadata
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? followerName = freezed,
    Object? badgeCode = freezed,
    Object? badgeName = freezed,
    Object? badgeIconUrl = freezed,
    Object? profileUrl = freezed,
  }) {
    return _then(
      _$NotificationMetadataImpl(
        followerName: freezed == followerName
            ? _value.followerName
            : followerName // ignore: cast_nullable_to_non_nullable
                  as String?,
        badgeCode: freezed == badgeCode
            ? _value.badgeCode
            : badgeCode // ignore: cast_nullable_to_non_nullable
                  as String?,
        badgeName: freezed == badgeName
            ? _value.badgeName
            : badgeName // ignore: cast_nullable_to_non_nullable
                  as String?,
        badgeIconUrl: freezed == badgeIconUrl
            ? _value.badgeIconUrl
            : badgeIconUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
        profileUrl: freezed == profileUrl
            ? _value.profileUrl
            : profileUrl // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationMetadataImpl implements _NotificationMetadata {
  const _$NotificationMetadataImpl({
    this.followerName,
    this.badgeCode,
    this.badgeName,
    this.badgeIconUrl,
    this.profileUrl,
  });

  factory _$NotificationMetadataImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationMetadataImplFromJson(json);

  @override
  final String? followerName;
  @override
  final String? badgeCode;
  @override
  final String? badgeName;
  @override
  final String? badgeIconUrl;
  @override
  final String? profileUrl;

  @override
  String toString() {
    return 'NotificationMetadata(followerName: $followerName, badgeCode: $badgeCode, badgeName: $badgeName, badgeIconUrl: $badgeIconUrl, profileUrl: $profileUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationMetadataImpl &&
            (identical(other.followerName, followerName) ||
                other.followerName == followerName) &&
            (identical(other.badgeCode, badgeCode) ||
                other.badgeCode == badgeCode) &&
            (identical(other.badgeName, badgeName) ||
                other.badgeName == badgeName) &&
            (identical(other.badgeIconUrl, badgeIconUrl) ||
                other.badgeIconUrl == badgeIconUrl) &&
            (identical(other.profileUrl, profileUrl) ||
                other.profileUrl == profileUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    followerName,
    badgeCode,
    badgeName,
    badgeIconUrl,
    profileUrl,
  );

  /// Create a copy of NotificationMetadata
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationMetadataImplCopyWith<_$NotificationMetadataImpl>
  get copyWith =>
      __$$NotificationMetadataImplCopyWithImpl<_$NotificationMetadataImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationMetadataImplToJson(this);
  }
}

abstract class _NotificationMetadata implements NotificationMetadata {
  const factory _NotificationMetadata({
    final String? followerName,
    final String? badgeCode,
    final String? badgeName,
    final String? badgeIconUrl,
    final String? profileUrl,
  }) = _$NotificationMetadataImpl;

  factory _NotificationMetadata.fromJson(Map<String, dynamic> json) =
      _$NotificationMetadataImpl.fromJson;

  @override
  String? get followerName;
  @override
  String? get badgeCode;
  @override
  String? get badgeName;
  @override
  String? get badgeIconUrl;
  @override
  String? get profileUrl;

  /// Create a copy of NotificationMetadata
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationMetadataImplCopyWith<_$NotificationMetadataImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationListResponse _$NotificationListResponseFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationListResponse.fromJson(json);
}

/// @nodoc
mixin _$NotificationListResponse {
  List<NotificationPayload> get items => throw _privateConstructorUsedError;
  int get nextCursor => throw _privateConstructorUsedError;
  bool get hasNext => throw _privateConstructorUsedError;
  int get unreadCount => throw _privateConstructorUsedError;

  /// Serializes this NotificationListResponse to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationListResponseCopyWith<NotificationListResponse> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationListResponseCopyWith<$Res> {
  factory $NotificationListResponseCopyWith(
    NotificationListResponse value,
    $Res Function(NotificationListResponse) then,
  ) = _$NotificationListResponseCopyWithImpl<$Res, NotificationListResponse>;
  @useResult
  $Res call({
    List<NotificationPayload> items,
    int nextCursor,
    bool hasNext,
    int unreadCount,
  });
}

/// @nodoc
class _$NotificationListResponseCopyWithImpl<
  $Res,
  $Val extends NotificationListResponse
>
    implements $NotificationListResponseCopyWith<$Res> {
  _$NotificationListResponseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
    Object? unreadCount = null,
  }) {
    return _then(
      _value.copyWith(
            items: null == items
                ? _value.items
                : items // ignore: cast_nullable_to_non_nullable
                      as List<NotificationPayload>,
            nextCursor: null == nextCursor
                ? _value.nextCursor
                : nextCursor // ignore: cast_nullable_to_non_nullable
                      as int,
            hasNext: null == hasNext
                ? _value.hasNext
                : hasNext // ignore: cast_nullable_to_non_nullable
                      as bool,
            unreadCount: null == unreadCount
                ? _value.unreadCount
                : unreadCount // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NotificationListResponseImplCopyWith<$Res>
    implements $NotificationListResponseCopyWith<$Res> {
  factory _$$NotificationListResponseImplCopyWith(
    _$NotificationListResponseImpl value,
    $Res Function(_$NotificationListResponseImpl) then,
  ) = __$$NotificationListResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<NotificationPayload> items,
    int nextCursor,
    bool hasNext,
    int unreadCount,
  });
}

/// @nodoc
class __$$NotificationListResponseImplCopyWithImpl<$Res>
    extends
        _$NotificationListResponseCopyWithImpl<
          $Res,
          _$NotificationListResponseImpl
        >
    implements _$$NotificationListResponseImplCopyWith<$Res> {
  __$$NotificationListResponseImplCopyWithImpl(
    _$NotificationListResponseImpl _value,
    $Res Function(_$NotificationListResponseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationListResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? items = null,
    Object? nextCursor = null,
    Object? hasNext = null,
    Object? unreadCount = null,
  }) {
    return _then(
      _$NotificationListResponseImpl(
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<NotificationPayload>,
        nextCursor: null == nextCursor
            ? _value.nextCursor
            : nextCursor // ignore: cast_nullable_to_non_nullable
                  as int,
        hasNext: null == hasNext
            ? _value.hasNext
            : hasNext // ignore: cast_nullable_to_non_nullable
                  as bool,
        unreadCount: null == unreadCount
            ? _value.unreadCount
            : unreadCount // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NotificationListResponseImpl implements _NotificationListResponse {
  const _$NotificationListResponseImpl({
    required final List<NotificationPayload> items,
    required this.nextCursor,
    required this.hasNext,
    required this.unreadCount,
  }) : _items = items;

  factory _$NotificationListResponseImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationListResponseImplFromJson(json);

  final List<NotificationPayload> _items;
  @override
  List<NotificationPayload> get items {
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_items);
  }

  @override
  final int nextCursor;
  @override
  final bool hasNext;
  @override
  final int unreadCount;

  @override
  String toString() {
    return 'NotificationListResponse(items: $items, nextCursor: $nextCursor, hasNext: $hasNext, unreadCount: $unreadCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationListResponseImpl &&
            const DeepCollectionEquality().equals(other._items, _items) &&
            (identical(other.nextCursor, nextCursor) ||
                other.nextCursor == nextCursor) &&
            (identical(other.hasNext, hasNext) || other.hasNext == hasNext) &&
            (identical(other.unreadCount, unreadCount) ||
                other.unreadCount == unreadCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_items),
    nextCursor,
    hasNext,
    unreadCount,
  );

  /// Create a copy of NotificationListResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationListResponseImplCopyWith<_$NotificationListResponseImpl>
  get copyWith =>
      __$$NotificationListResponseImplCopyWithImpl<
        _$NotificationListResponseImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationListResponseImplToJson(this);
  }
}

abstract class _NotificationListResponse implements NotificationListResponse {
  const factory _NotificationListResponse({
    required final List<NotificationPayload> items,
    required final int nextCursor,
    required final bool hasNext,
    required final int unreadCount,
  }) = _$NotificationListResponseImpl;

  factory _NotificationListResponse.fromJson(Map<String, dynamic> json) =
      _$NotificationListResponseImpl.fromJson;

  @override
  List<NotificationPayload> get items;
  @override
  int get nextCursor;
  @override
  bool get hasNext;
  @override
  int get unreadCount;

  /// Create a copy of NotificationListResponse
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationListResponseImplCopyWith<_$NotificationListResponseImpl>
  get copyWith => throw _privateConstructorUsedError;
}

NotificationsPageProps _$NotificationsPagePropsFromJson(
  Map<String, dynamic> json,
) {
  return _NotificationsPageProps.fromJson(json);
}

/// @nodoc
mixin _$NotificationsPageProps {
  int get total => throw _privateConstructorUsedError;
  int get unreadCount => throw _privateConstructorUsedError;
  List<NotificationPayload> get notifications =>
      throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;

  /// Serializes this NotificationsPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NotificationsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NotificationsPagePropsCopyWith<NotificationsPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NotificationsPagePropsCopyWith<$Res> {
  factory $NotificationsPagePropsCopyWith(
    NotificationsPageProps value,
    $Res Function(NotificationsPageProps) then,
  ) = _$NotificationsPagePropsCopyWithImpl<$Res, NotificationsPageProps>;
  @useResult
  $Res call({
    int total,
    int unreadCount,
    List<NotificationPayload> notifications,
    PaginationPayload pagination,
  });

  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$NotificationsPagePropsCopyWithImpl<
  $Res,
  $Val extends NotificationsPageProps
>
    implements $NotificationsPagePropsCopyWith<$Res> {
  _$NotificationsPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NotificationsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? total = null,
    Object? unreadCount = null,
    Object? notifications = null,
    Object? pagination = null,
  }) {
    return _then(
      _value.copyWith(
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
                      as int,
            unreadCount: null == unreadCount
                ? _value.unreadCount
                : unreadCount // ignore: cast_nullable_to_non_nullable
                      as int,
            notifications: null == notifications
                ? _value.notifications
                : notifications // ignore: cast_nullable_to_non_nullable
                      as List<NotificationPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of NotificationsPageProps
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
abstract class _$$NotificationsPagePropsImplCopyWith<$Res>
    implements $NotificationsPagePropsCopyWith<$Res> {
  factory _$$NotificationsPagePropsImplCopyWith(
    _$NotificationsPagePropsImpl value,
    $Res Function(_$NotificationsPagePropsImpl) then,
  ) = __$$NotificationsPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int total,
    int unreadCount,
    List<NotificationPayload> notifications,
    PaginationPayload pagination,
  });

  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$NotificationsPagePropsImplCopyWithImpl<$Res>
    extends
        _$NotificationsPagePropsCopyWithImpl<$Res, _$NotificationsPagePropsImpl>
    implements _$$NotificationsPagePropsImplCopyWith<$Res> {
  __$$NotificationsPagePropsImplCopyWithImpl(
    _$NotificationsPagePropsImpl _value,
    $Res Function(_$NotificationsPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NotificationsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? total = null,
    Object? unreadCount = null,
    Object? notifications = null,
    Object? pagination = null,
  }) {
    return _then(
      _$NotificationsPagePropsImpl(
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
                  as int,
        unreadCount: null == unreadCount
            ? _value.unreadCount
            : unreadCount // ignore: cast_nullable_to_non_nullable
                  as int,
        notifications: null == notifications
            ? _value._notifications
            : notifications // ignore: cast_nullable_to_non_nullable
                  as List<NotificationPayload>,
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
class _$NotificationsPagePropsImpl implements _NotificationsPageProps {
  const _$NotificationsPagePropsImpl({
    required this.total,
    required this.unreadCount,
    required final List<NotificationPayload> notifications,
    required this.pagination,
  }) : _notifications = notifications;

  factory _$NotificationsPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$NotificationsPagePropsImplFromJson(json);

  @override
  final int total;
  @override
  final int unreadCount;
  final List<NotificationPayload> _notifications;
  @override
  List<NotificationPayload> get notifications {
    if (_notifications is EqualUnmodifiableListView) return _notifications;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_notifications);
  }

  @override
  final PaginationPayload pagination;

  @override
  String toString() {
    return 'NotificationsPageProps(total: $total, unreadCount: $unreadCount, notifications: $notifications, pagination: $pagination)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NotificationsPagePropsImpl &&
            (identical(other.total, total) || other.total == total) &&
            (identical(other.unreadCount, unreadCount) ||
                other.unreadCount == unreadCount) &&
            const DeepCollectionEquality().equals(
              other._notifications,
              _notifications,
            ) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    total,
    unreadCount,
    const DeepCollectionEquality().hash(_notifications),
    pagination,
  );

  /// Create a copy of NotificationsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NotificationsPagePropsImplCopyWith<_$NotificationsPagePropsImpl>
  get copyWith =>
      __$$NotificationsPagePropsImplCopyWithImpl<_$NotificationsPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$NotificationsPagePropsImplToJson(this);
  }
}

abstract class _NotificationsPageProps implements NotificationsPageProps {
  const factory _NotificationsPageProps({
    required final int total,
    required final int unreadCount,
    required final List<NotificationPayload> notifications,
    required final PaginationPayload pagination,
  }) = _$NotificationsPagePropsImpl;

  factory _NotificationsPageProps.fromJson(Map<String, dynamic> json) =
      _$NotificationsPagePropsImpl.fromJson;

  @override
  int get total;
  @override
  int get unreadCount;
  @override
  List<NotificationPayload> get notifications;
  @override
  PaginationPayload get pagination;

  /// Create a copy of NotificationsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NotificationsPagePropsImplCopyWith<_$NotificationsPagePropsImpl>
  get copyWith => throw _privateConstructorUsedError;
}

DraftPayload _$DraftPayloadFromJson(Map<String, dynamic> json) {
  return _DraftPayload.fromJson(json);
}

/// @nodoc
mixin _$DraftPayload {
  int get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get editUrl => throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get viewCount => throw _privateConstructorUsedError;
  int get processStatus => throw _privateConstructorUsedError;
  String get updatedAt => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  List<CategoryBriefPayload> get categories =>
      throw _privateConstructorUsedError;

  /// Serializes this DraftPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of DraftPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $DraftPayloadCopyWith<DraftPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $DraftPayloadCopyWith<$Res> {
  factory $DraftPayloadCopyWith(
    DraftPayload value,
    $Res Function(DraftPayload) then,
  ) = _$DraftPayloadCopyWithImpl<$Res, DraftPayload>;
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String editUrl,
    int replyCount,
    int viewCount,
    int processStatus,
    String updatedAt,
    String createdAt,
    List<CategoryBriefPayload> categories,
  });
}

/// @nodoc
class _$DraftPayloadCopyWithImpl<$Res, $Val extends DraftPayload>
    implements $DraftPayloadCopyWith<$Res> {
  _$DraftPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of DraftPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? editUrl = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? processStatus = null,
    Object? updatedAt = null,
    Object? createdAt = null,
    Object? categories = null,
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
            editUrl: null == editUrl
                ? _value.editUrl
                : editUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            viewCount: null == viewCount
                ? _value.viewCount
                : viewCount // ignore: cast_nullable_to_non_nullable
                      as int,
            processStatus: null == processStatus
                ? _value.processStatus
                : processStatus // ignore: cast_nullable_to_non_nullable
                      as int,
            updatedAt: null == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryBriefPayload>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$DraftPayloadImplCopyWith<$Res>
    implements $DraftPayloadCopyWith<$Res> {
  factory _$$DraftPayloadImplCopyWith(
    _$DraftPayloadImpl value,
    $Res Function(_$DraftPayloadImpl) then,
  ) = __$$DraftPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String title,
    String description,
    String editUrl,
    int replyCount,
    int viewCount,
    int processStatus,
    String updatedAt,
    String createdAt,
    List<CategoryBriefPayload> categories,
  });
}

/// @nodoc
class __$$DraftPayloadImplCopyWithImpl<$Res>
    extends _$DraftPayloadCopyWithImpl<$Res, _$DraftPayloadImpl>
    implements _$$DraftPayloadImplCopyWith<$Res> {
  __$$DraftPayloadImplCopyWithImpl(
    _$DraftPayloadImpl _value,
    $Res Function(_$DraftPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of DraftPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? title = null,
    Object? description = null,
    Object? editUrl = null,
    Object? replyCount = null,
    Object? viewCount = null,
    Object? processStatus = null,
    Object? updatedAt = null,
    Object? createdAt = null,
    Object? categories = null,
  }) {
    return _then(
      _$DraftPayloadImpl(
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
        editUrl: null == editUrl
            ? _value.editUrl
            : editUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        viewCount: null == viewCount
            ? _value.viewCount
            : viewCount // ignore: cast_nullable_to_non_nullable
                  as int,
        processStatus: null == processStatus
            ? _value.processStatus
            : processStatus // ignore: cast_nullable_to_non_nullable
                  as int,
        updatedAt: null == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryBriefPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$DraftPayloadImpl implements _DraftPayload {
  const _$DraftPayloadImpl({
    required this.id,
    required this.title,
    required this.description,
    required this.editUrl,
    required this.replyCount,
    required this.viewCount,
    required this.processStatus,
    required this.updatedAt,
    required this.createdAt,
    required final List<CategoryBriefPayload> categories,
  }) : _categories = categories;

  factory _$DraftPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$DraftPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String title;
  @override
  final String description;
  @override
  final String editUrl;
  @override
  final int replyCount;
  @override
  final int viewCount;
  @override
  final int processStatus;
  @override
  final String updatedAt;
  @override
  final String createdAt;
  final List<CategoryBriefPayload> _categories;
  @override
  List<CategoryBriefPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  String toString() {
    return 'DraftPayload(id: $id, title: $title, description: $description, editUrl: $editUrl, replyCount: $replyCount, viewCount: $viewCount, processStatus: $processStatus, updatedAt: $updatedAt, createdAt: $createdAt, categories: $categories)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$DraftPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.editUrl, editUrl) || other.editUrl == editUrl) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.viewCount, viewCount) ||
                other.viewCount == viewCount) &&
            (identical(other.processStatus, processStatus) ||
                other.processStatus == processStatus) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    title,
    description,
    editUrl,
    replyCount,
    viewCount,
    processStatus,
    updatedAt,
    createdAt,
    const DeepCollectionEquality().hash(_categories),
  );

  /// Create a copy of DraftPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$DraftPayloadImplCopyWith<_$DraftPayloadImpl> get copyWith =>
      __$$DraftPayloadImplCopyWithImpl<_$DraftPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$DraftPayloadImplToJson(this);
  }
}

abstract class _DraftPayload implements DraftPayload {
  const factory _DraftPayload({
    required final int id,
    required final String title,
    required final String description,
    required final String editUrl,
    required final int replyCount,
    required final int viewCount,
    required final int processStatus,
    required final String updatedAt,
    required final String createdAt,
    required final List<CategoryBriefPayload> categories,
  }) = _$DraftPayloadImpl;

  factory _DraftPayload.fromJson(Map<String, dynamic> json) =
      _$DraftPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get title;
  @override
  String get description;
  @override
  String get editUrl;
  @override
  int get replyCount;
  @override
  int get viewCount;
  @override
  int get processStatus;
  @override
  String get updatedAt;
  @override
  String get createdAt;
  @override
  List<CategoryBriefPayload> get categories;

  /// Create a copy of DraftPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$DraftPayloadImplCopyWith<_$DraftPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

DraftsPageProps _$DraftsPagePropsFromJson(Map<String, dynamic> json) {
  return _DraftsPageProps.fromJson(json);
}

/// @nodoc
mixin _$DraftsPageProps {
  int get total => throw _privateConstructorUsedError;
  List<DraftPayload> get drafts => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;

  /// Serializes this DraftsPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of DraftsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $DraftsPagePropsCopyWith<DraftsPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $DraftsPagePropsCopyWith<$Res> {
  factory $DraftsPagePropsCopyWith(
    DraftsPageProps value,
    $Res Function(DraftsPageProps) then,
  ) = _$DraftsPagePropsCopyWithImpl<$Res, DraftsPageProps>;
  @useResult
  $Res call({
    int total,
    List<DraftPayload> drafts,
    PaginationPayload pagination,
  });

  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$DraftsPagePropsCopyWithImpl<$Res, $Val extends DraftsPageProps>
    implements $DraftsPagePropsCopyWith<$Res> {
  _$DraftsPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of DraftsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? total = null,
    Object? drafts = null,
    Object? pagination = null,
  }) {
    return _then(
      _value.copyWith(
            total: null == total
                ? _value.total
                : total // ignore: cast_nullable_to_non_nullable
                      as int,
            drafts: null == drafts
                ? _value.drafts
                : drafts // ignore: cast_nullable_to_non_nullable
                      as List<DraftPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of DraftsPageProps
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
abstract class _$$DraftsPagePropsImplCopyWith<$Res>
    implements $DraftsPagePropsCopyWith<$Res> {
  factory _$$DraftsPagePropsImplCopyWith(
    _$DraftsPagePropsImpl value,
    $Res Function(_$DraftsPagePropsImpl) then,
  ) = __$$DraftsPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int total,
    List<DraftPayload> drafts,
    PaginationPayload pagination,
  });

  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$DraftsPagePropsImplCopyWithImpl<$Res>
    extends _$DraftsPagePropsCopyWithImpl<$Res, _$DraftsPagePropsImpl>
    implements _$$DraftsPagePropsImplCopyWith<$Res> {
  __$$DraftsPagePropsImplCopyWithImpl(
    _$DraftsPagePropsImpl _value,
    $Res Function(_$DraftsPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of DraftsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? total = null,
    Object? drafts = null,
    Object? pagination = null,
  }) {
    return _then(
      _$DraftsPagePropsImpl(
        total: null == total
            ? _value.total
            : total // ignore: cast_nullable_to_non_nullable
                  as int,
        drafts: null == drafts
            ? _value._drafts
            : drafts // ignore: cast_nullable_to_non_nullable
                  as List<DraftPayload>,
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
class _$DraftsPagePropsImpl implements _DraftsPageProps {
  const _$DraftsPagePropsImpl({
    required this.total,
    required final List<DraftPayload> drafts,
    required this.pagination,
  }) : _drafts = drafts;

  factory _$DraftsPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$DraftsPagePropsImplFromJson(json);

  @override
  final int total;
  final List<DraftPayload> _drafts;
  @override
  List<DraftPayload> get drafts {
    if (_drafts is EqualUnmodifiableListView) return _drafts;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_drafts);
  }

  @override
  final PaginationPayload pagination;

  @override
  String toString() {
    return 'DraftsPageProps(total: $total, drafts: $drafts, pagination: $pagination)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$DraftsPagePropsImpl &&
            (identical(other.total, total) || other.total == total) &&
            const DeepCollectionEquality().equals(other._drafts, _drafts) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    total,
    const DeepCollectionEquality().hash(_drafts),
    pagination,
  );

  /// Create a copy of DraftsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$DraftsPagePropsImplCopyWith<_$DraftsPagePropsImpl> get copyWith =>
      __$$DraftsPagePropsImplCopyWithImpl<_$DraftsPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$DraftsPagePropsImplToJson(this);
  }
}

abstract class _DraftsPageProps implements DraftsPageProps {
  const factory _DraftsPageProps({
    required final int total,
    required final List<DraftPayload> drafts,
    required final PaginationPayload pagination,
  }) = _$DraftsPagePropsImpl;

  factory _DraftsPageProps.fromJson(Map<String, dynamic> json) =
      _$DraftsPagePropsImpl.fromJson;

  @override
  int get total;
  @override
  List<DraftPayload> get drafts;
  @override
  PaginationPayload get pagination;

  /// Create a copy of DraftsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$DraftsPagePropsImplCopyWith<_$DraftsPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
