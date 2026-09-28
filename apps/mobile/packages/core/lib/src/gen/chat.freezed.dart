// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'chat.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

ChatItemPayload _$ChatItemPayloadFromJson(Map<String, dynamic> json) {
  return _ChatItemPayload.fromJson(json);
}

/// @nodoc
mixin _$ChatItemPayload {
  int get id => throw _privateConstructorUsedError;
  int get peerId => throw _privateConstructorUsedError;
  String get peerUsername => throw _privateConstructorUsedError;
  String? get peerNickname => throw _privateConstructorUsedError;
  String get peerAvatar => throw _privateConstructorUsedError;
  String get lastMsg => throw _privateConstructorUsedError;
  String get lastMsgTime => throw _privateConstructorUsedError;
  int get unreadCount => throw _privateConstructorUsedError;
  int get convId => throw _privateConstructorUsedError;
  String get peerUrl => throw _privateConstructorUsedError;

  /// Serializes this ChatItemPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatItemPayloadCopyWith<ChatItemPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatItemPayloadCopyWith<$Res> {
  factory $ChatItemPayloadCopyWith(
    ChatItemPayload value,
    $Res Function(ChatItemPayload) then,
  ) = _$ChatItemPayloadCopyWithImpl<$Res, ChatItemPayload>;
  @useResult
  $Res call({
    int id,
    int peerId,
    String peerUsername,
    String? peerNickname,
    String peerAvatar,
    String lastMsg,
    String lastMsgTime,
    int unreadCount,
    int convId,
    String peerUrl,
  });
}

/// @nodoc
class _$ChatItemPayloadCopyWithImpl<$Res, $Val extends ChatItemPayload>
    implements $ChatItemPayloadCopyWith<$Res> {
  _$ChatItemPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? peerId = null,
    Object? peerUsername = null,
    Object? peerNickname = freezed,
    Object? peerAvatar = null,
    Object? lastMsg = null,
    Object? lastMsgTime = null,
    Object? unreadCount = null,
    Object? convId = null,
    Object? peerUrl = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            peerId: null == peerId
                ? _value.peerId
                : peerId // ignore: cast_nullable_to_non_nullable
                      as int,
            peerUsername: null == peerUsername
                ? _value.peerUsername
                : peerUsername // ignore: cast_nullable_to_non_nullable
                      as String,
            peerNickname: freezed == peerNickname
                ? _value.peerNickname
                : peerNickname // ignore: cast_nullable_to_non_nullable
                      as String?,
            peerAvatar: null == peerAvatar
                ? _value.peerAvatar
                : peerAvatar // ignore: cast_nullable_to_non_nullable
                      as String,
            lastMsg: null == lastMsg
                ? _value.lastMsg
                : lastMsg // ignore: cast_nullable_to_non_nullable
                      as String,
            lastMsgTime: null == lastMsgTime
                ? _value.lastMsgTime
                : lastMsgTime // ignore: cast_nullable_to_non_nullable
                      as String,
            unreadCount: null == unreadCount
                ? _value.unreadCount
                : unreadCount // ignore: cast_nullable_to_non_nullable
                      as int,
            convId: null == convId
                ? _value.convId
                : convId // ignore: cast_nullable_to_non_nullable
                      as int,
            peerUrl: null == peerUrl
                ? _value.peerUrl
                : peerUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatItemPayloadImplCopyWith<$Res>
    implements $ChatItemPayloadCopyWith<$Res> {
  factory _$$ChatItemPayloadImplCopyWith(
    _$ChatItemPayloadImpl value,
    $Res Function(_$ChatItemPayloadImpl) then,
  ) = __$$ChatItemPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int peerId,
    String peerUsername,
    String? peerNickname,
    String peerAvatar,
    String lastMsg,
    String lastMsgTime,
    int unreadCount,
    int convId,
    String peerUrl,
  });
}

/// @nodoc
class __$$ChatItemPayloadImplCopyWithImpl<$Res>
    extends _$ChatItemPayloadCopyWithImpl<$Res, _$ChatItemPayloadImpl>
    implements _$$ChatItemPayloadImplCopyWith<$Res> {
  __$$ChatItemPayloadImplCopyWithImpl(
    _$ChatItemPayloadImpl _value,
    $Res Function(_$ChatItemPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? peerId = null,
    Object? peerUsername = null,
    Object? peerNickname = freezed,
    Object? peerAvatar = null,
    Object? lastMsg = null,
    Object? lastMsgTime = null,
    Object? unreadCount = null,
    Object? convId = null,
    Object? peerUrl = null,
  }) {
    return _then(
      _$ChatItemPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        peerId: null == peerId
            ? _value.peerId
            : peerId // ignore: cast_nullable_to_non_nullable
                  as int,
        peerUsername: null == peerUsername
            ? _value.peerUsername
            : peerUsername // ignore: cast_nullable_to_non_nullable
                  as String,
        peerNickname: freezed == peerNickname
            ? _value.peerNickname
            : peerNickname // ignore: cast_nullable_to_non_nullable
                  as String?,
        peerAvatar: null == peerAvatar
            ? _value.peerAvatar
            : peerAvatar // ignore: cast_nullable_to_non_nullable
                  as String,
        lastMsg: null == lastMsg
            ? _value.lastMsg
            : lastMsg // ignore: cast_nullable_to_non_nullable
                  as String,
        lastMsgTime: null == lastMsgTime
            ? _value.lastMsgTime
            : lastMsgTime // ignore: cast_nullable_to_non_nullable
                  as String,
        unreadCount: null == unreadCount
            ? _value.unreadCount
            : unreadCount // ignore: cast_nullable_to_non_nullable
                  as int,
        convId: null == convId
            ? _value.convId
            : convId // ignore: cast_nullable_to_non_nullable
                  as int,
        peerUrl: null == peerUrl
            ? _value.peerUrl
            : peerUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatItemPayloadImpl implements _ChatItemPayload {
  const _$ChatItemPayloadImpl({
    required this.id,
    required this.peerId,
    required this.peerUsername,
    this.peerNickname,
    required this.peerAvatar,
    required this.lastMsg,
    required this.lastMsgTime,
    required this.unreadCount,
    required this.convId,
    required this.peerUrl,
  });

  factory _$ChatItemPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatItemPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int peerId;
  @override
  final String peerUsername;
  @override
  final String? peerNickname;
  @override
  final String peerAvatar;
  @override
  final String lastMsg;
  @override
  final String lastMsgTime;
  @override
  final int unreadCount;
  @override
  final int convId;
  @override
  final String peerUrl;

  @override
  String toString() {
    return 'ChatItemPayload(id: $id, peerId: $peerId, peerUsername: $peerUsername, peerNickname: $peerNickname, peerAvatar: $peerAvatar, lastMsg: $lastMsg, lastMsgTime: $lastMsgTime, unreadCount: $unreadCount, convId: $convId, peerUrl: $peerUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatItemPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.peerId, peerId) || other.peerId == peerId) &&
            (identical(other.peerUsername, peerUsername) ||
                other.peerUsername == peerUsername) &&
            (identical(other.peerNickname, peerNickname) ||
                other.peerNickname == peerNickname) &&
            (identical(other.peerAvatar, peerAvatar) ||
                other.peerAvatar == peerAvatar) &&
            (identical(other.lastMsg, lastMsg) || other.lastMsg == lastMsg) &&
            (identical(other.lastMsgTime, lastMsgTime) ||
                other.lastMsgTime == lastMsgTime) &&
            (identical(other.unreadCount, unreadCount) ||
                other.unreadCount == unreadCount) &&
            (identical(other.convId, convId) || other.convId == convId) &&
            (identical(other.peerUrl, peerUrl) || other.peerUrl == peerUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    peerId,
    peerUsername,
    peerNickname,
    peerAvatar,
    lastMsg,
    lastMsgTime,
    unreadCount,
    convId,
    peerUrl,
  );

  /// Create a copy of ChatItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatItemPayloadImplCopyWith<_$ChatItemPayloadImpl> get copyWith =>
      __$$ChatItemPayloadImplCopyWithImpl<_$ChatItemPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatItemPayloadImplToJson(this);
  }
}

abstract class _ChatItemPayload implements ChatItemPayload {
  const factory _ChatItemPayload({
    required final int id,
    required final int peerId,
    required final String peerUsername,
    final String? peerNickname,
    required final String peerAvatar,
    required final String lastMsg,
    required final String lastMsgTime,
    required final int unreadCount,
    required final int convId,
    required final String peerUrl,
  }) = _$ChatItemPayloadImpl;

  factory _ChatItemPayload.fromJson(Map<String, dynamic> json) =
      _$ChatItemPayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get peerId;
  @override
  String get peerUsername;
  @override
  String? get peerNickname;
  @override
  String get peerAvatar;
  @override
  String get lastMsg;
  @override
  String get lastMsgTime;
  @override
  int get unreadCount;
  @override
  int get convId;
  @override
  String get peerUrl;

  /// Create a copy of ChatItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatItemPayloadImplCopyWith<_$ChatItemPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

MessagesPageProps _$MessagesPagePropsFromJson(Map<String, dynamic> json) {
  return _MessagesPageProps.fromJson(json);
}

/// @nodoc
mixin _$MessagesPageProps {
  List<ChatItemPayload> get conversations => throw _privateConstructorUsedError;
  List<UserConnectionPayload> get suggestedUsers =>
      throw _privateConstructorUsedError;

  /// Serializes this MessagesPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of MessagesPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $MessagesPagePropsCopyWith<MessagesPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $MessagesPagePropsCopyWith<$Res> {
  factory $MessagesPagePropsCopyWith(
    MessagesPageProps value,
    $Res Function(MessagesPageProps) then,
  ) = _$MessagesPagePropsCopyWithImpl<$Res, MessagesPageProps>;
  @useResult
  $Res call({
    List<ChatItemPayload> conversations,
    List<UserConnectionPayload> suggestedUsers,
  });
}

/// @nodoc
class _$MessagesPagePropsCopyWithImpl<$Res, $Val extends MessagesPageProps>
    implements $MessagesPagePropsCopyWith<$Res> {
  _$MessagesPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of MessagesPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? conversations = null, Object? suggestedUsers = null}) {
    return _then(
      _value.copyWith(
            conversations: null == conversations
                ? _value.conversations
                : conversations // ignore: cast_nullable_to_non_nullable
                      as List<ChatItemPayload>,
            suggestedUsers: null == suggestedUsers
                ? _value.suggestedUsers
                : suggestedUsers // ignore: cast_nullable_to_non_nullable
                      as List<UserConnectionPayload>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$MessagesPagePropsImplCopyWith<$Res>
    implements $MessagesPagePropsCopyWith<$Res> {
  factory _$$MessagesPagePropsImplCopyWith(
    _$MessagesPagePropsImpl value,
    $Res Function(_$MessagesPagePropsImpl) then,
  ) = __$$MessagesPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<ChatItemPayload> conversations,
    List<UserConnectionPayload> suggestedUsers,
  });
}

/// @nodoc
class __$$MessagesPagePropsImplCopyWithImpl<$Res>
    extends _$MessagesPagePropsCopyWithImpl<$Res, _$MessagesPagePropsImpl>
    implements _$$MessagesPagePropsImplCopyWith<$Res> {
  __$$MessagesPagePropsImplCopyWithImpl(
    _$MessagesPagePropsImpl _value,
    $Res Function(_$MessagesPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of MessagesPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? conversations = null, Object? suggestedUsers = null}) {
    return _then(
      _$MessagesPagePropsImpl(
        conversations: null == conversations
            ? _value._conversations
            : conversations // ignore: cast_nullable_to_non_nullable
                  as List<ChatItemPayload>,
        suggestedUsers: null == suggestedUsers
            ? _value._suggestedUsers
            : suggestedUsers // ignore: cast_nullable_to_non_nullable
                  as List<UserConnectionPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$MessagesPagePropsImpl implements _MessagesPageProps {
  const _$MessagesPagePropsImpl({
    required final List<ChatItemPayload> conversations,
    required final List<UserConnectionPayload> suggestedUsers,
  }) : _conversations = conversations,
       _suggestedUsers = suggestedUsers;

  factory _$MessagesPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$MessagesPagePropsImplFromJson(json);

  final List<ChatItemPayload> _conversations;
  @override
  List<ChatItemPayload> get conversations {
    if (_conversations is EqualUnmodifiableListView) return _conversations;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_conversations);
  }

  final List<UserConnectionPayload> _suggestedUsers;
  @override
  List<UserConnectionPayload> get suggestedUsers {
    if (_suggestedUsers is EqualUnmodifiableListView) return _suggestedUsers;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_suggestedUsers);
  }

  @override
  String toString() {
    return 'MessagesPageProps(conversations: $conversations, suggestedUsers: $suggestedUsers)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$MessagesPagePropsImpl &&
            const DeepCollectionEquality().equals(
              other._conversations,
              _conversations,
            ) &&
            const DeepCollectionEquality().equals(
              other._suggestedUsers,
              _suggestedUsers,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_conversations),
    const DeepCollectionEquality().hash(_suggestedUsers),
  );

  /// Create a copy of MessagesPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$MessagesPagePropsImplCopyWith<_$MessagesPagePropsImpl> get copyWith =>
      __$$MessagesPagePropsImplCopyWithImpl<_$MessagesPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$MessagesPagePropsImplToJson(this);
  }
}

abstract class _MessagesPageProps implements MessagesPageProps {
  const factory _MessagesPageProps({
    required final List<ChatItemPayload> conversations,
    required final List<UserConnectionPayload> suggestedUsers,
  }) = _$MessagesPagePropsImpl;

  factory _MessagesPageProps.fromJson(Map<String, dynamic> json) =
      _$MessagesPagePropsImpl.fromJson;

  @override
  List<ChatItemPayload> get conversations;
  @override
  List<UserConnectionPayload> get suggestedUsers;

  /// Create a copy of MessagesPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$MessagesPagePropsImplCopyWith<_$MessagesPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChatMessagePayload _$ChatMessagePayloadFromJson(Map<String, dynamic> json) {
  return _ChatMessagePayload.fromJson(json);
}

/// @nodoc
mixin _$ChatMessagePayload {
  int get id => throw _privateConstructorUsedError;
  int get senderId => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  int get msgType => throw _privateConstructorUsedError;
  int get isRead => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  bool get isSelf => throw _privateConstructorUsedError;
  ChatForwardBundle? get forwarded => throw _privateConstructorUsedError;

  /// Serializes this ChatMessagePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatMessagePayloadCopyWith<ChatMessagePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatMessagePayloadCopyWith<$Res> {
  factory $ChatMessagePayloadCopyWith(
    ChatMessagePayload value,
    $Res Function(ChatMessagePayload) then,
  ) = _$ChatMessagePayloadCopyWithImpl<$Res, ChatMessagePayload>;
  @useResult
  $Res call({
    int id,
    int senderId,
    String content,
    int msgType,
    int isRead,
    String createdAt,
    bool isSelf,
    ChatForwardBundle? forwarded,
  });

  $ChatForwardBundleCopyWith<$Res>? get forwarded;
}

/// @nodoc
class _$ChatMessagePayloadCopyWithImpl<$Res, $Val extends ChatMessagePayload>
    implements $ChatMessagePayloadCopyWith<$Res> {
  _$ChatMessagePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? senderId = null,
    Object? content = null,
    Object? msgType = null,
    Object? isRead = null,
    Object? createdAt = null,
    Object? isSelf = null,
    Object? forwarded = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            senderId: null == senderId
                ? _value.senderId
                : senderId // ignore: cast_nullable_to_non_nullable
                      as int,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            msgType: null == msgType
                ? _value.msgType
                : msgType // ignore: cast_nullable_to_non_nullable
                      as int,
            isRead: null == isRead
                ? _value.isRead
                : isRead // ignore: cast_nullable_to_non_nullable
                      as int,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            isSelf: null == isSelf
                ? _value.isSelf
                : isSelf // ignore: cast_nullable_to_non_nullable
                      as bool,
            forwarded: freezed == forwarded
                ? _value.forwarded
                : forwarded // ignore: cast_nullable_to_non_nullable
                      as ChatForwardBundle?,
          )
          as $Val,
    );
  }

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ChatForwardBundleCopyWith<$Res>? get forwarded {
    if (_value.forwarded == null) {
      return null;
    }

    return $ChatForwardBundleCopyWith<$Res>(_value.forwarded!, (value) {
      return _then(_value.copyWith(forwarded: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ChatMessagePayloadImplCopyWith<$Res>
    implements $ChatMessagePayloadCopyWith<$Res> {
  factory _$$ChatMessagePayloadImplCopyWith(
    _$ChatMessagePayloadImpl value,
    $Res Function(_$ChatMessagePayloadImpl) then,
  ) = __$$ChatMessagePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int senderId,
    String content,
    int msgType,
    int isRead,
    String createdAt,
    bool isSelf,
    ChatForwardBundle? forwarded,
  });

  @override
  $ChatForwardBundleCopyWith<$Res>? get forwarded;
}

/// @nodoc
class __$$ChatMessagePayloadImplCopyWithImpl<$Res>
    extends _$ChatMessagePayloadCopyWithImpl<$Res, _$ChatMessagePayloadImpl>
    implements _$$ChatMessagePayloadImplCopyWith<$Res> {
  __$$ChatMessagePayloadImplCopyWithImpl(
    _$ChatMessagePayloadImpl _value,
    $Res Function(_$ChatMessagePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? senderId = null,
    Object? content = null,
    Object? msgType = null,
    Object? isRead = null,
    Object? createdAt = null,
    Object? isSelf = null,
    Object? forwarded = freezed,
  }) {
    return _then(
      _$ChatMessagePayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        senderId: null == senderId
            ? _value.senderId
            : senderId // ignore: cast_nullable_to_non_nullable
                  as int,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        msgType: null == msgType
            ? _value.msgType
            : msgType // ignore: cast_nullable_to_non_nullable
                  as int,
        isRead: null == isRead
            ? _value.isRead
            : isRead // ignore: cast_nullable_to_non_nullable
                  as int,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        isSelf: null == isSelf
            ? _value.isSelf
            : isSelf // ignore: cast_nullable_to_non_nullable
                  as bool,
        forwarded: freezed == forwarded
            ? _value.forwarded
            : forwarded // ignore: cast_nullable_to_non_nullable
                  as ChatForwardBundle?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatMessagePayloadImpl implements _ChatMessagePayload {
  const _$ChatMessagePayloadImpl({
    required this.id,
    required this.senderId,
    required this.content,
    required this.msgType,
    required this.isRead,
    required this.createdAt,
    required this.isSelf,
    this.forwarded,
  });

  factory _$ChatMessagePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatMessagePayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int senderId;
  @override
  final String content;
  @override
  final int msgType;
  @override
  final int isRead;
  @override
  final String createdAt;
  @override
  final bool isSelf;
  @override
  final ChatForwardBundle? forwarded;

  @override
  String toString() {
    return 'ChatMessagePayload(id: $id, senderId: $senderId, content: $content, msgType: $msgType, isRead: $isRead, createdAt: $createdAt, isSelf: $isSelf, forwarded: $forwarded)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatMessagePayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.senderId, senderId) ||
                other.senderId == senderId) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.msgType, msgType) || other.msgType == msgType) &&
            (identical(other.isRead, isRead) || other.isRead == isRead) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.isSelf, isSelf) || other.isSelf == isSelf) &&
            (identical(other.forwarded, forwarded) ||
                other.forwarded == forwarded));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    senderId,
    content,
    msgType,
    isRead,
    createdAt,
    isSelf,
    forwarded,
  );

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatMessagePayloadImplCopyWith<_$ChatMessagePayloadImpl> get copyWith =>
      __$$ChatMessagePayloadImplCopyWithImpl<_$ChatMessagePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatMessagePayloadImplToJson(this);
  }
}

abstract class _ChatMessagePayload implements ChatMessagePayload {
  const factory _ChatMessagePayload({
    required final int id,
    required final int senderId,
    required final String content,
    required final int msgType,
    required final int isRead,
    required final String createdAt,
    required final bool isSelf,
    final ChatForwardBundle? forwarded,
  }) = _$ChatMessagePayloadImpl;

  factory _ChatMessagePayload.fromJson(Map<String, dynamic> json) =
      _$ChatMessagePayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get senderId;
  @override
  String get content;
  @override
  int get msgType;
  @override
  int get isRead;
  @override
  String get createdAt;
  @override
  bool get isSelf;
  @override
  ChatForwardBundle? get forwarded;

  /// Create a copy of ChatMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatMessagePayloadImplCopyWith<_$ChatMessagePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChatMessagesResponse _$ChatMessagesResponseFromJson(Map<String, dynamic> json) {
  return _ChatMessagesResponse.fromJson(json);
}

/// @nodoc
mixin _$ChatMessagesResponse {
  List<ChatMessagePayload> get list => throw _privateConstructorUsedError;
  bool get hasMoreBefore => throw _privateConstructorUsedError;
  bool get hasMoreAfter => throw _privateConstructorUsedError;
  int get nextBeforeId => throw _privateConstructorUsedError;
  int get latestId => throw _privateConstructorUsedError;

  /// Serializes this ChatMessagesResponse to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatMessagesResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatMessagesResponseCopyWith<ChatMessagesResponse> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatMessagesResponseCopyWith<$Res> {
  factory $ChatMessagesResponseCopyWith(
    ChatMessagesResponse value,
    $Res Function(ChatMessagesResponse) then,
  ) = _$ChatMessagesResponseCopyWithImpl<$Res, ChatMessagesResponse>;
  @useResult
  $Res call({
    List<ChatMessagePayload> list,
    bool hasMoreBefore,
    bool hasMoreAfter,
    int nextBeforeId,
    int latestId,
  });
}

/// @nodoc
class _$ChatMessagesResponseCopyWithImpl<
  $Res,
  $Val extends ChatMessagesResponse
>
    implements $ChatMessagesResponseCopyWith<$Res> {
  _$ChatMessagesResponseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatMessagesResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? hasMoreBefore = null,
    Object? hasMoreAfter = null,
    Object? nextBeforeId = null,
    Object? latestId = null,
  }) {
    return _then(
      _value.copyWith(
            list: null == list
                ? _value.list
                : list // ignore: cast_nullable_to_non_nullable
                      as List<ChatMessagePayload>,
            hasMoreBefore: null == hasMoreBefore
                ? _value.hasMoreBefore
                : hasMoreBefore // ignore: cast_nullable_to_non_nullable
                      as bool,
            hasMoreAfter: null == hasMoreAfter
                ? _value.hasMoreAfter
                : hasMoreAfter // ignore: cast_nullable_to_non_nullable
                      as bool,
            nextBeforeId: null == nextBeforeId
                ? _value.nextBeforeId
                : nextBeforeId // ignore: cast_nullable_to_non_nullable
                      as int,
            latestId: null == latestId
                ? _value.latestId
                : latestId // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatMessagesResponseImplCopyWith<$Res>
    implements $ChatMessagesResponseCopyWith<$Res> {
  factory _$$ChatMessagesResponseImplCopyWith(
    _$ChatMessagesResponseImpl value,
    $Res Function(_$ChatMessagesResponseImpl) then,
  ) = __$$ChatMessagesResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<ChatMessagePayload> list,
    bool hasMoreBefore,
    bool hasMoreAfter,
    int nextBeforeId,
    int latestId,
  });
}

/// @nodoc
class __$$ChatMessagesResponseImplCopyWithImpl<$Res>
    extends _$ChatMessagesResponseCopyWithImpl<$Res, _$ChatMessagesResponseImpl>
    implements _$$ChatMessagesResponseImplCopyWith<$Res> {
  __$$ChatMessagesResponseImplCopyWithImpl(
    _$ChatMessagesResponseImpl _value,
    $Res Function(_$ChatMessagesResponseImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatMessagesResponse
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? list = null,
    Object? hasMoreBefore = null,
    Object? hasMoreAfter = null,
    Object? nextBeforeId = null,
    Object? latestId = null,
  }) {
    return _then(
      _$ChatMessagesResponseImpl(
        list: null == list
            ? _value._list
            : list // ignore: cast_nullable_to_non_nullable
                  as List<ChatMessagePayload>,
        hasMoreBefore: null == hasMoreBefore
            ? _value.hasMoreBefore
            : hasMoreBefore // ignore: cast_nullable_to_non_nullable
                  as bool,
        hasMoreAfter: null == hasMoreAfter
            ? _value.hasMoreAfter
            : hasMoreAfter // ignore: cast_nullable_to_non_nullable
                  as bool,
        nextBeforeId: null == nextBeforeId
            ? _value.nextBeforeId
            : nextBeforeId // ignore: cast_nullable_to_non_nullable
                  as int,
        latestId: null == latestId
            ? _value.latestId
            : latestId // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatMessagesResponseImpl implements _ChatMessagesResponse {
  const _$ChatMessagesResponseImpl({
    required final List<ChatMessagePayload> list,
    required this.hasMoreBefore,
    required this.hasMoreAfter,
    required this.nextBeforeId,
    required this.latestId,
  }) : _list = list;

  factory _$ChatMessagesResponseImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatMessagesResponseImplFromJson(json);

  final List<ChatMessagePayload> _list;
  @override
  List<ChatMessagePayload> get list {
    if (_list is EqualUnmodifiableListView) return _list;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_list);
  }

  @override
  final bool hasMoreBefore;
  @override
  final bool hasMoreAfter;
  @override
  final int nextBeforeId;
  @override
  final int latestId;

  @override
  String toString() {
    return 'ChatMessagesResponse(list: $list, hasMoreBefore: $hasMoreBefore, hasMoreAfter: $hasMoreAfter, nextBeforeId: $nextBeforeId, latestId: $latestId)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatMessagesResponseImpl &&
            const DeepCollectionEquality().equals(other._list, _list) &&
            (identical(other.hasMoreBefore, hasMoreBefore) ||
                other.hasMoreBefore == hasMoreBefore) &&
            (identical(other.hasMoreAfter, hasMoreAfter) ||
                other.hasMoreAfter == hasMoreAfter) &&
            (identical(other.nextBeforeId, nextBeforeId) ||
                other.nextBeforeId == nextBeforeId) &&
            (identical(other.latestId, latestId) ||
                other.latestId == latestId));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_list),
    hasMoreBefore,
    hasMoreAfter,
    nextBeforeId,
    latestId,
  );

  /// Create a copy of ChatMessagesResponse
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatMessagesResponseImplCopyWith<_$ChatMessagesResponseImpl>
  get copyWith =>
      __$$ChatMessagesResponseImplCopyWithImpl<_$ChatMessagesResponseImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatMessagesResponseImplToJson(this);
  }
}

abstract class _ChatMessagesResponse implements ChatMessagesResponse {
  const factory _ChatMessagesResponse({
    required final List<ChatMessagePayload> list,
    required final bool hasMoreBefore,
    required final bool hasMoreAfter,
    required final int nextBeforeId,
    required final int latestId,
  }) = _$ChatMessagesResponseImpl;

  factory _ChatMessagesResponse.fromJson(Map<String, dynamic> json) =
      _$ChatMessagesResponseImpl.fromJson;

  @override
  List<ChatMessagePayload> get list;
  @override
  bool get hasMoreBefore;
  @override
  bool get hasMoreAfter;
  @override
  int get nextBeforeId;
  @override
  int get latestId;

  /// Create a copy of ChatMessagesResponse
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatMessagesResponseImplCopyWith<_$ChatMessagesResponseImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ChatVisibleReadResult _$ChatVisibleReadResultFromJson(
  Map<String, dynamic> json,
) {
  return _ChatVisibleReadResult.fromJson(json);
}

/// @nodoc
mixin _$ChatVisibleReadResult {
  int get convId => throw _privateConstructorUsedError;
  List<int> get acknowledgedMessageIds => throw _privateConstructorUsedError;
  int get unreadCount => throw _privateConstructorUsedError;

  /// Serializes this ChatVisibleReadResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatVisibleReadResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatVisibleReadResultCopyWith<ChatVisibleReadResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatVisibleReadResultCopyWith<$Res> {
  factory $ChatVisibleReadResultCopyWith(
    ChatVisibleReadResult value,
    $Res Function(ChatVisibleReadResult) then,
  ) = _$ChatVisibleReadResultCopyWithImpl<$Res, ChatVisibleReadResult>;
  @useResult
  $Res call({int convId, List<int> acknowledgedMessageIds, int unreadCount});
}

/// @nodoc
class _$ChatVisibleReadResultCopyWithImpl<
  $Res,
  $Val extends ChatVisibleReadResult
>
    implements $ChatVisibleReadResultCopyWith<$Res> {
  _$ChatVisibleReadResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatVisibleReadResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? convId = null,
    Object? acknowledgedMessageIds = null,
    Object? unreadCount = null,
  }) {
    return _then(
      _value.copyWith(
            convId: null == convId
                ? _value.convId
                : convId // ignore: cast_nullable_to_non_nullable
                      as int,
            acknowledgedMessageIds: null == acknowledgedMessageIds
                ? _value.acknowledgedMessageIds
                : acknowledgedMessageIds // ignore: cast_nullable_to_non_nullable
                      as List<int>,
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
abstract class _$$ChatVisibleReadResultImplCopyWith<$Res>
    implements $ChatVisibleReadResultCopyWith<$Res> {
  factory _$$ChatVisibleReadResultImplCopyWith(
    _$ChatVisibleReadResultImpl value,
    $Res Function(_$ChatVisibleReadResultImpl) then,
  ) = __$$ChatVisibleReadResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int convId, List<int> acknowledgedMessageIds, int unreadCount});
}

/// @nodoc
class __$$ChatVisibleReadResultImplCopyWithImpl<$Res>
    extends
        _$ChatVisibleReadResultCopyWithImpl<$Res, _$ChatVisibleReadResultImpl>
    implements _$$ChatVisibleReadResultImplCopyWith<$Res> {
  __$$ChatVisibleReadResultImplCopyWithImpl(
    _$ChatVisibleReadResultImpl _value,
    $Res Function(_$ChatVisibleReadResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatVisibleReadResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? convId = null,
    Object? acknowledgedMessageIds = null,
    Object? unreadCount = null,
  }) {
    return _then(
      _$ChatVisibleReadResultImpl(
        convId: null == convId
            ? _value.convId
            : convId // ignore: cast_nullable_to_non_nullable
                  as int,
        acknowledgedMessageIds: null == acknowledgedMessageIds
            ? _value._acknowledgedMessageIds
            : acknowledgedMessageIds // ignore: cast_nullable_to_non_nullable
                  as List<int>,
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
class _$ChatVisibleReadResultImpl implements _ChatVisibleReadResult {
  const _$ChatVisibleReadResultImpl({
    required this.convId,
    required final List<int> acknowledgedMessageIds,
    required this.unreadCount,
  }) : _acknowledgedMessageIds = acknowledgedMessageIds;

  factory _$ChatVisibleReadResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatVisibleReadResultImplFromJson(json);

  @override
  final int convId;
  final List<int> _acknowledgedMessageIds;
  @override
  List<int> get acknowledgedMessageIds {
    if (_acknowledgedMessageIds is EqualUnmodifiableListView)
      return _acknowledgedMessageIds;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_acknowledgedMessageIds);
  }

  @override
  final int unreadCount;

  @override
  String toString() {
    return 'ChatVisibleReadResult(convId: $convId, acknowledgedMessageIds: $acknowledgedMessageIds, unreadCount: $unreadCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatVisibleReadResultImpl &&
            (identical(other.convId, convId) || other.convId == convId) &&
            const DeepCollectionEquality().equals(
              other._acknowledgedMessageIds,
              _acknowledgedMessageIds,
            ) &&
            (identical(other.unreadCount, unreadCount) ||
                other.unreadCount == unreadCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    convId,
    const DeepCollectionEquality().hash(_acknowledgedMessageIds),
    unreadCount,
  );

  /// Create a copy of ChatVisibleReadResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatVisibleReadResultImplCopyWith<_$ChatVisibleReadResultImpl>
  get copyWith =>
      __$$ChatVisibleReadResultImplCopyWithImpl<_$ChatVisibleReadResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatVisibleReadResultImplToJson(this);
  }
}

abstract class _ChatVisibleReadResult implements ChatVisibleReadResult {
  const factory _ChatVisibleReadResult({
    required final int convId,
    required final List<int> acknowledgedMessageIds,
    required final int unreadCount,
  }) = _$ChatVisibleReadResultImpl;

  factory _ChatVisibleReadResult.fromJson(Map<String, dynamic> json) =
      _$ChatVisibleReadResultImpl.fromJson;

  @override
  int get convId;
  @override
  List<int> get acknowledgedMessageIds;
  @override
  int get unreadCount;

  /// Create a copy of ChatVisibleReadResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatVisibleReadResultImplCopyWith<_$ChatVisibleReadResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ChatMessageReadState _$ChatMessageReadStateFromJson(Map<String, dynamic> json) {
  return _ChatMessageReadState.fromJson(json);
}

/// @nodoc
mixin _$ChatMessageReadState {
  int get id => throw _privateConstructorUsedError;
  int get isRead => throw _privateConstructorUsedError;

  /// Serializes this ChatMessageReadState to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatMessageReadState
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatMessageReadStateCopyWith<ChatMessageReadState> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatMessageReadStateCopyWith<$Res> {
  factory $ChatMessageReadStateCopyWith(
    ChatMessageReadState value,
    $Res Function(ChatMessageReadState) then,
  ) = _$ChatMessageReadStateCopyWithImpl<$Res, ChatMessageReadState>;
  @useResult
  $Res call({int id, int isRead});
}

/// @nodoc
class _$ChatMessageReadStateCopyWithImpl<
  $Res,
  $Val extends ChatMessageReadState
>
    implements $ChatMessageReadStateCopyWith<$Res> {
  _$ChatMessageReadStateCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatMessageReadState
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? isRead = null}) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            isRead: null == isRead
                ? _value.isRead
                : isRead // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatMessageReadStateImplCopyWith<$Res>
    implements $ChatMessageReadStateCopyWith<$Res> {
  factory _$$ChatMessageReadStateImplCopyWith(
    _$ChatMessageReadStateImpl value,
    $Res Function(_$ChatMessageReadStateImpl) then,
  ) = __$$ChatMessageReadStateImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, int isRead});
}

/// @nodoc
class __$$ChatMessageReadStateImplCopyWithImpl<$Res>
    extends _$ChatMessageReadStateCopyWithImpl<$Res, _$ChatMessageReadStateImpl>
    implements _$$ChatMessageReadStateImplCopyWith<$Res> {
  __$$ChatMessageReadStateImplCopyWithImpl(
    _$ChatMessageReadStateImpl _value,
    $Res Function(_$ChatMessageReadStateImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatMessageReadState
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? isRead = null}) {
    return _then(
      _$ChatMessageReadStateImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        isRead: null == isRead
            ? _value.isRead
            : isRead // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatMessageReadStateImpl implements _ChatMessageReadState {
  const _$ChatMessageReadStateImpl({required this.id, required this.isRead});

  factory _$ChatMessageReadStateImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatMessageReadStateImplFromJson(json);

  @override
  final int id;
  @override
  final int isRead;

  @override
  String toString() {
    return 'ChatMessageReadState(id: $id, isRead: $isRead)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatMessageReadStateImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.isRead, isRead) || other.isRead == isRead));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, isRead);

  /// Create a copy of ChatMessageReadState
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatMessageReadStateImplCopyWith<_$ChatMessageReadStateImpl>
  get copyWith =>
      __$$ChatMessageReadStateImplCopyWithImpl<_$ChatMessageReadStateImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatMessageReadStateImplToJson(this);
  }
}

abstract class _ChatMessageReadState implements ChatMessageReadState {
  const factory _ChatMessageReadState({
    required final int id,
    required final int isRead,
  }) = _$ChatMessageReadStateImpl;

  factory _ChatMessageReadState.fromJson(Map<String, dynamic> json) =
      _$ChatMessageReadStateImpl.fromJson;

  @override
  int get id;
  @override
  int get isRead;

  /// Create a copy of ChatMessageReadState
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatMessageReadStateImplCopyWith<_$ChatMessageReadStateImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ChatMessageReadStatesResult _$ChatMessageReadStatesResultFromJson(
  Map<String, dynamic> json,
) {
  return _ChatMessageReadStatesResult.fromJson(json);
}

/// @nodoc
mixin _$ChatMessageReadStatesResult {
  List<ChatMessageReadState> get items => throw _privateConstructorUsedError;
  int get unreadCount => throw _privateConstructorUsedError;

  /// Serializes this ChatMessageReadStatesResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatMessageReadStatesResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatMessageReadStatesResultCopyWith<ChatMessageReadStatesResult>
  get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatMessageReadStatesResultCopyWith<$Res> {
  factory $ChatMessageReadStatesResultCopyWith(
    ChatMessageReadStatesResult value,
    $Res Function(ChatMessageReadStatesResult) then,
  ) =
      _$ChatMessageReadStatesResultCopyWithImpl<
        $Res,
        ChatMessageReadStatesResult
      >;
  @useResult
  $Res call({List<ChatMessageReadState> items, int unreadCount});
}

/// @nodoc
class _$ChatMessageReadStatesResultCopyWithImpl<
  $Res,
  $Val extends ChatMessageReadStatesResult
>
    implements $ChatMessageReadStatesResultCopyWith<$Res> {
  _$ChatMessageReadStatesResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatMessageReadStatesResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? items = null, Object? unreadCount = null}) {
    return _then(
      _value.copyWith(
            items: null == items
                ? _value.items
                : items // ignore: cast_nullable_to_non_nullable
                      as List<ChatMessageReadState>,
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
abstract class _$$ChatMessageReadStatesResultImplCopyWith<$Res>
    implements $ChatMessageReadStatesResultCopyWith<$Res> {
  factory _$$ChatMessageReadStatesResultImplCopyWith(
    _$ChatMessageReadStatesResultImpl value,
    $Res Function(_$ChatMessageReadStatesResultImpl) then,
  ) = __$$ChatMessageReadStatesResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<ChatMessageReadState> items, int unreadCount});
}

/// @nodoc
class __$$ChatMessageReadStatesResultImplCopyWithImpl<$Res>
    extends
        _$ChatMessageReadStatesResultCopyWithImpl<
          $Res,
          _$ChatMessageReadStatesResultImpl
        >
    implements _$$ChatMessageReadStatesResultImplCopyWith<$Res> {
  __$$ChatMessageReadStatesResultImplCopyWithImpl(
    _$ChatMessageReadStatesResultImpl _value,
    $Res Function(_$ChatMessageReadStatesResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatMessageReadStatesResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? items = null, Object? unreadCount = null}) {
    return _then(
      _$ChatMessageReadStatesResultImpl(
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<ChatMessageReadState>,
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
class _$ChatMessageReadStatesResultImpl
    implements _ChatMessageReadStatesResult {
  const _$ChatMessageReadStatesResultImpl({
    required final List<ChatMessageReadState> items,
    required this.unreadCount,
  }) : _items = items;

  factory _$ChatMessageReadStatesResultImpl.fromJson(
    Map<String, dynamic> json,
  ) => _$$ChatMessageReadStatesResultImplFromJson(json);

  final List<ChatMessageReadState> _items;
  @override
  List<ChatMessageReadState> get items {
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_items);
  }

  @override
  final int unreadCount;

  @override
  String toString() {
    return 'ChatMessageReadStatesResult(items: $items, unreadCount: $unreadCount)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatMessageReadStatesResultImpl &&
            const DeepCollectionEquality().equals(other._items, _items) &&
            (identical(other.unreadCount, unreadCount) ||
                other.unreadCount == unreadCount));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_items),
    unreadCount,
  );

  /// Create a copy of ChatMessageReadStatesResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatMessageReadStatesResultImplCopyWith<_$ChatMessageReadStatesResultImpl>
  get copyWith =>
      __$$ChatMessageReadStatesResultImplCopyWithImpl<
        _$ChatMessageReadStatesResultImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatMessageReadStatesResultImplToJson(this);
  }
}

abstract class _ChatMessageReadStatesResult
    implements ChatMessageReadStatesResult {
  const factory _ChatMessageReadStatesResult({
    required final List<ChatMessageReadState> items,
    required final int unreadCount,
  }) = _$ChatMessageReadStatesResultImpl;

  factory _ChatMessageReadStatesResult.fromJson(Map<String, dynamic> json) =
      _$ChatMessageReadStatesResultImpl.fromJson;

  @override
  List<ChatMessageReadState> get items;
  @override
  int get unreadCount;

  /// Create a copy of ChatMessageReadStatesResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatMessageReadStatesResultImplCopyWith<_$ChatMessageReadStatesResultImpl>
  get copyWith => throw _privateConstructorUsedError;
}

ChatForwardBundle _$ChatForwardBundleFromJson(Map<String, dynamic> json) {
  return _ChatForwardBundle.fromJson(json);
}

/// @nodoc
mixin _$ChatForwardBundle {
  int get version => throw _privateConstructorUsedError;
  List<ChatForwardEntry> get messages => throw _privateConstructorUsedError;

  /// Serializes this ChatForwardBundle to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatForwardBundle
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatForwardBundleCopyWith<ChatForwardBundle> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatForwardBundleCopyWith<$Res> {
  factory $ChatForwardBundleCopyWith(
    ChatForwardBundle value,
    $Res Function(ChatForwardBundle) then,
  ) = _$ChatForwardBundleCopyWithImpl<$Res, ChatForwardBundle>;
  @useResult
  $Res call({int version, List<ChatForwardEntry> messages});
}

/// @nodoc
class _$ChatForwardBundleCopyWithImpl<$Res, $Val extends ChatForwardBundle>
    implements $ChatForwardBundleCopyWith<$Res> {
  _$ChatForwardBundleCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatForwardBundle
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? version = null, Object? messages = null}) {
    return _then(
      _value.copyWith(
            version: null == version
                ? _value.version
                : version // ignore: cast_nullable_to_non_nullable
                      as int,
            messages: null == messages
                ? _value.messages
                : messages // ignore: cast_nullable_to_non_nullable
                      as List<ChatForwardEntry>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatForwardBundleImplCopyWith<$Res>
    implements $ChatForwardBundleCopyWith<$Res> {
  factory _$$ChatForwardBundleImplCopyWith(
    _$ChatForwardBundleImpl value,
    $Res Function(_$ChatForwardBundleImpl) then,
  ) = __$$ChatForwardBundleImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int version, List<ChatForwardEntry> messages});
}

/// @nodoc
class __$$ChatForwardBundleImplCopyWithImpl<$Res>
    extends _$ChatForwardBundleCopyWithImpl<$Res, _$ChatForwardBundleImpl>
    implements _$$ChatForwardBundleImplCopyWith<$Res> {
  __$$ChatForwardBundleImplCopyWithImpl(
    _$ChatForwardBundleImpl _value,
    $Res Function(_$ChatForwardBundleImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatForwardBundle
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? version = null, Object? messages = null}) {
    return _then(
      _$ChatForwardBundleImpl(
        version: null == version
            ? _value.version
            : version // ignore: cast_nullable_to_non_nullable
                  as int,
        messages: null == messages
            ? _value._messages
            : messages // ignore: cast_nullable_to_non_nullable
                  as List<ChatForwardEntry>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatForwardBundleImpl implements _ChatForwardBundle {
  const _$ChatForwardBundleImpl({
    required this.version,
    required final List<ChatForwardEntry> messages,
  }) : _messages = messages;

  factory _$ChatForwardBundleImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatForwardBundleImplFromJson(json);

  @override
  final int version;
  final List<ChatForwardEntry> _messages;
  @override
  List<ChatForwardEntry> get messages {
    if (_messages is EqualUnmodifiableListView) return _messages;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_messages);
  }

  @override
  String toString() {
    return 'ChatForwardBundle(version: $version, messages: $messages)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatForwardBundleImpl &&
            (identical(other.version, version) || other.version == version) &&
            const DeepCollectionEquality().equals(other._messages, _messages));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    version,
    const DeepCollectionEquality().hash(_messages),
  );

  /// Create a copy of ChatForwardBundle
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatForwardBundleImplCopyWith<_$ChatForwardBundleImpl> get copyWith =>
      __$$ChatForwardBundleImplCopyWithImpl<_$ChatForwardBundleImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatForwardBundleImplToJson(this);
  }
}

abstract class _ChatForwardBundle implements ChatForwardBundle {
  const factory _ChatForwardBundle({
    required final int version,
    required final List<ChatForwardEntry> messages,
  }) = _$ChatForwardBundleImpl;

  factory _ChatForwardBundle.fromJson(Map<String, dynamic> json) =
      _$ChatForwardBundleImpl.fromJson;

  @override
  int get version;
  @override
  List<ChatForwardEntry> get messages;

  /// Create a copy of ChatForwardBundle
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatForwardBundleImplCopyWith<_$ChatForwardBundleImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChatForwardEntry _$ChatForwardEntryFromJson(Map<String, dynamic> json) {
  return _ChatForwardEntry.fromJson(json);
}

/// @nodoc
mixin _$ChatForwardEntry {
  String get senderName => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  int get msgType => throw _privateConstructorUsedError;
  ChatForwardBundle? get forwarded => throw _privateConstructorUsedError;

  /// Serializes this ChatForwardEntry to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatForwardEntryCopyWith<ChatForwardEntry> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatForwardEntryCopyWith<$Res> {
  factory $ChatForwardEntryCopyWith(
    ChatForwardEntry value,
    $Res Function(ChatForwardEntry) then,
  ) = _$ChatForwardEntryCopyWithImpl<$Res, ChatForwardEntry>;
  @useResult
  $Res call({
    String senderName,
    String avatarUrl,
    String content,
    String createdAt,
    int msgType,
    ChatForwardBundle? forwarded,
  });

  $ChatForwardBundleCopyWith<$Res>? get forwarded;
}

/// @nodoc
class _$ChatForwardEntryCopyWithImpl<$Res, $Val extends ChatForwardEntry>
    implements $ChatForwardEntryCopyWith<$Res> {
  _$ChatForwardEntryCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? senderName = null,
    Object? avatarUrl = null,
    Object? content = null,
    Object? createdAt = null,
    Object? msgType = null,
    Object? forwarded = freezed,
  }) {
    return _then(
      _value.copyWith(
            senderName: null == senderName
                ? _value.senderName
                : senderName // ignore: cast_nullable_to_non_nullable
                      as String,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            msgType: null == msgType
                ? _value.msgType
                : msgType // ignore: cast_nullable_to_non_nullable
                      as int,
            forwarded: freezed == forwarded
                ? _value.forwarded
                : forwarded // ignore: cast_nullable_to_non_nullable
                      as ChatForwardBundle?,
          )
          as $Val,
    );
  }

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ChatForwardBundleCopyWith<$Res>? get forwarded {
    if (_value.forwarded == null) {
      return null;
    }

    return $ChatForwardBundleCopyWith<$Res>(_value.forwarded!, (value) {
      return _then(_value.copyWith(forwarded: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ChatForwardEntryImplCopyWith<$Res>
    implements $ChatForwardEntryCopyWith<$Res> {
  factory _$$ChatForwardEntryImplCopyWith(
    _$ChatForwardEntryImpl value,
    $Res Function(_$ChatForwardEntryImpl) then,
  ) = __$$ChatForwardEntryImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String senderName,
    String avatarUrl,
    String content,
    String createdAt,
    int msgType,
    ChatForwardBundle? forwarded,
  });

  @override
  $ChatForwardBundleCopyWith<$Res>? get forwarded;
}

/// @nodoc
class __$$ChatForwardEntryImplCopyWithImpl<$Res>
    extends _$ChatForwardEntryCopyWithImpl<$Res, _$ChatForwardEntryImpl>
    implements _$$ChatForwardEntryImplCopyWith<$Res> {
  __$$ChatForwardEntryImplCopyWithImpl(
    _$ChatForwardEntryImpl _value,
    $Res Function(_$ChatForwardEntryImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? senderName = null,
    Object? avatarUrl = null,
    Object? content = null,
    Object? createdAt = null,
    Object? msgType = null,
    Object? forwarded = freezed,
  }) {
    return _then(
      _$ChatForwardEntryImpl(
        senderName: null == senderName
            ? _value.senderName
            : senderName // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        msgType: null == msgType
            ? _value.msgType
            : msgType // ignore: cast_nullable_to_non_nullable
                  as int,
        forwarded: freezed == forwarded
            ? _value.forwarded
            : forwarded // ignore: cast_nullable_to_non_nullable
                  as ChatForwardBundle?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatForwardEntryImpl implements _ChatForwardEntry {
  const _$ChatForwardEntryImpl({
    required this.senderName,
    this.avatarUrl = '',
    required this.content,
    required this.createdAt,
    required this.msgType,
    this.forwarded,
  });

  factory _$ChatForwardEntryImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatForwardEntryImplFromJson(json);

  @override
  final String senderName;
  @override
  @JsonKey()
  final String avatarUrl;
  @override
  final String content;
  @override
  final String createdAt;
  @override
  final int msgType;
  @override
  final ChatForwardBundle? forwarded;

  @override
  String toString() {
    return 'ChatForwardEntry(senderName: $senderName, avatarUrl: $avatarUrl, content: $content, createdAt: $createdAt, msgType: $msgType, forwarded: $forwarded)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatForwardEntryImpl &&
            (identical(other.senderName, senderName) ||
                other.senderName == senderName) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.msgType, msgType) || other.msgType == msgType) &&
            (identical(other.forwarded, forwarded) ||
                other.forwarded == forwarded));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    senderName,
    avatarUrl,
    content,
    createdAt,
    msgType,
    forwarded,
  );

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatForwardEntryImplCopyWith<_$ChatForwardEntryImpl> get copyWith =>
      __$$ChatForwardEntryImplCopyWithImpl<_$ChatForwardEntryImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatForwardEntryImplToJson(this);
  }
}

abstract class _ChatForwardEntry implements ChatForwardEntry {
  const factory _ChatForwardEntry({
    required final String senderName,
    final String avatarUrl,
    required final String content,
    required final String createdAt,
    required final int msgType,
    final ChatForwardBundle? forwarded,
  }) = _$ChatForwardEntryImpl;

  factory _ChatForwardEntry.fromJson(Map<String, dynamic> json) =
      _$ChatForwardEntryImpl.fromJson;

  @override
  String get senderName;
  @override
  String get avatarUrl;
  @override
  String get content;
  @override
  String get createdAt;
  @override
  int get msgType;
  @override
  ChatForwardBundle? get forwarded;

  /// Create a copy of ChatForwardEntry
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatForwardEntryImplCopyWith<_$ChatForwardEntryImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ChatForwardResult _$ChatForwardResultFromJson(Map<String, dynamic> json) {
  return _ChatForwardResult.fromJson(json);
}

/// @nodoc
mixin _$ChatForwardResult {
  int get convId => throw _privateConstructorUsedError;
  List<int> get messageIds => throw _privateConstructorUsedError;

  /// Serializes this ChatForwardResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ChatForwardResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ChatForwardResultCopyWith<ChatForwardResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ChatForwardResultCopyWith<$Res> {
  factory $ChatForwardResultCopyWith(
    ChatForwardResult value,
    $Res Function(ChatForwardResult) then,
  ) = _$ChatForwardResultCopyWithImpl<$Res, ChatForwardResult>;
  @useResult
  $Res call({int convId, List<int> messageIds});
}

/// @nodoc
class _$ChatForwardResultCopyWithImpl<$Res, $Val extends ChatForwardResult>
    implements $ChatForwardResultCopyWith<$Res> {
  _$ChatForwardResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ChatForwardResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? convId = null, Object? messageIds = null}) {
    return _then(
      _value.copyWith(
            convId: null == convId
                ? _value.convId
                : convId // ignore: cast_nullable_to_non_nullable
                      as int,
            messageIds: null == messageIds
                ? _value.messageIds
                : messageIds // ignore: cast_nullable_to_non_nullable
                      as List<int>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ChatForwardResultImplCopyWith<$Res>
    implements $ChatForwardResultCopyWith<$Res> {
  factory _$$ChatForwardResultImplCopyWith(
    _$ChatForwardResultImpl value,
    $Res Function(_$ChatForwardResultImpl) then,
  ) = __$$ChatForwardResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int convId, List<int> messageIds});
}

/// @nodoc
class __$$ChatForwardResultImplCopyWithImpl<$Res>
    extends _$ChatForwardResultCopyWithImpl<$Res, _$ChatForwardResultImpl>
    implements _$$ChatForwardResultImplCopyWith<$Res> {
  __$$ChatForwardResultImplCopyWithImpl(
    _$ChatForwardResultImpl _value,
    $Res Function(_$ChatForwardResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ChatForwardResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? convId = null, Object? messageIds = null}) {
    return _then(
      _$ChatForwardResultImpl(
        convId: null == convId
            ? _value.convId
            : convId // ignore: cast_nullable_to_non_nullable
                  as int,
        messageIds: null == messageIds
            ? _value._messageIds
            : messageIds // ignore: cast_nullable_to_non_nullable
                  as List<int>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ChatForwardResultImpl implements _ChatForwardResult {
  const _$ChatForwardResultImpl({
    required this.convId,
    required final List<int> messageIds,
  }) : _messageIds = messageIds;

  factory _$ChatForwardResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$ChatForwardResultImplFromJson(json);

  @override
  final int convId;
  final List<int> _messageIds;
  @override
  List<int> get messageIds {
    if (_messageIds is EqualUnmodifiableListView) return _messageIds;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_messageIds);
  }

  @override
  String toString() {
    return 'ChatForwardResult(convId: $convId, messageIds: $messageIds)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ChatForwardResultImpl &&
            (identical(other.convId, convId) || other.convId == convId) &&
            const DeepCollectionEquality().equals(
              other._messageIds,
              _messageIds,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    convId,
    const DeepCollectionEquality().hash(_messageIds),
  );

  /// Create a copy of ChatForwardResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ChatForwardResultImplCopyWith<_$ChatForwardResultImpl> get copyWith =>
      __$$ChatForwardResultImplCopyWithImpl<_$ChatForwardResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ChatForwardResultImplToJson(this);
  }
}

abstract class _ChatForwardResult implements ChatForwardResult {
  const factory _ChatForwardResult({
    required final int convId,
    required final List<int> messageIds,
  }) = _$ChatForwardResultImpl;

  factory _ChatForwardResult.fromJson(Map<String, dynamic> json) =
      _$ChatForwardResultImpl.fromJson;

  @override
  int get convId;
  @override
  List<int> get messageIds;

  /// Create a copy of ChatForwardResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ChatForwardResultImplCopyWith<_$ChatForwardResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
