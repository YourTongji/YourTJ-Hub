// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'user.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

UserCardPayload _$UserCardPayloadFromJson(Map<String, dynamic> json) {
  return _UserCardPayload.fromJson(json);
}

/// @nodoc
mixin _$UserCardPayload {
  int get userId => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get nickname => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get profileCoverUrl => throw _privateConstructorUsedError;
  String get bio => throw _privateConstructorUsedError;
  String get signature => throw _privateConstructorUsedError;
  String get websiteName => throw _privateConstructorUsedError;
  String get website => throw _privateConstructorUsedError;
  int get prestige => throw _privateConstructorUsedError;
  Map<String, ExternalLinkPayload> get externalInformation =>
      throw _privateConstructorUsedError;
  bool get isAdmin => throw _privateConstructorUsedError;
  int get topicCount => throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get likeReceivedCount => throw _privateConstructorUsedError;
  int get likeGivenCount => throw _privateConstructorUsedError;
  int get followerCount => throw _privateConstructorUsedError;
  int get followingCount => throw _privateConstructorUsedError;
  int get collectionCount => throw _privateConstructorUsedError;
  bool get isOnline => throw _privateConstructorUsedError;
  bool get isFollowing => throw _privateConstructorUsedError;
  bool get isSelf => throw _privateConstructorUsedError;
  List<UserBadgePayload> get badges => throw _privateConstructorUsedError;
  UserBadgePayload? get wornBadge => throw _privateConstructorUsedError;
  String get lastActiveTime => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;

  /// Serializes this UserCardPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserCardPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserCardPayloadCopyWith<UserCardPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserCardPayloadCopyWith<$Res> {
  factory $UserCardPayloadCopyWith(
    UserCardPayload value,
    $Res Function(UserCardPayload) then,
  ) = _$UserCardPayloadCopyWithImpl<$Res, UserCardPayload>;
  @useResult
  $Res call({
    int userId,
    String username,
    String nickname,
    String avatarUrl,
    String profileCoverUrl,
    String bio,
    String signature,
    String websiteName,
    String website,
    int prestige,
    Map<String, ExternalLinkPayload> externalInformation,
    bool isAdmin,
    int topicCount,
    int replyCount,
    int likeReceivedCount,
    int likeGivenCount,
    int followerCount,
    int followingCount,
    int collectionCount,
    bool isOnline,
    bool isFollowing,
    bool isSelf,
    List<UserBadgePayload> badges,
    UserBadgePayload? wornBadge,
    String lastActiveTime,
    String createdAt,
  });

  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class _$UserCardPayloadCopyWithImpl<$Res, $Val extends UserCardPayload>
    implements $UserCardPayloadCopyWith<$Res> {
  _$UserCardPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserCardPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? userId = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? profileCoverUrl = null,
    Object? bio = null,
    Object? signature = null,
    Object? websiteName = null,
    Object? website = null,
    Object? prestige = null,
    Object? externalInformation = null,
    Object? isAdmin = null,
    Object? topicCount = null,
    Object? replyCount = null,
    Object? likeReceivedCount = null,
    Object? likeGivenCount = null,
    Object? followerCount = null,
    Object? followingCount = null,
    Object? collectionCount = null,
    Object? isOnline = null,
    Object? isFollowing = null,
    Object? isSelf = null,
    Object? badges = null,
    Object? wornBadge = freezed,
    Object? lastActiveTime = null,
    Object? createdAt = null,
  }) {
    return _then(
      _value.copyWith(
            userId: null == userId
                ? _value.userId
                : userId // ignore: cast_nullable_to_non_nullable
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
            profileCoverUrl: null == profileCoverUrl
                ? _value.profileCoverUrl
                : profileCoverUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            bio: null == bio
                ? _value.bio
                : bio // ignore: cast_nullable_to_non_nullable
                      as String,
            signature: null == signature
                ? _value.signature
                : signature // ignore: cast_nullable_to_non_nullable
                      as String,
            websiteName: null == websiteName
                ? _value.websiteName
                : websiteName // ignore: cast_nullable_to_non_nullable
                      as String,
            website: null == website
                ? _value.website
                : website // ignore: cast_nullable_to_non_nullable
                      as String,
            prestige: null == prestige
                ? _value.prestige
                : prestige // ignore: cast_nullable_to_non_nullable
                      as int,
            externalInformation: null == externalInformation
                ? _value.externalInformation
                : externalInformation // ignore: cast_nullable_to_non_nullable
                      as Map<String, ExternalLinkPayload>,
            isAdmin: null == isAdmin
                ? _value.isAdmin
                : isAdmin // ignore: cast_nullable_to_non_nullable
                      as bool,
            topicCount: null == topicCount
                ? _value.topicCount
                : topicCount // ignore: cast_nullable_to_non_nullable
                      as int,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            likeReceivedCount: null == likeReceivedCount
                ? _value.likeReceivedCount
                : likeReceivedCount // ignore: cast_nullable_to_non_nullable
                      as int,
            likeGivenCount: null == likeGivenCount
                ? _value.likeGivenCount
                : likeGivenCount // ignore: cast_nullable_to_non_nullable
                      as int,
            followerCount: null == followerCount
                ? _value.followerCount
                : followerCount // ignore: cast_nullable_to_non_nullable
                      as int,
            followingCount: null == followingCount
                ? _value.followingCount
                : followingCount // ignore: cast_nullable_to_non_nullable
                      as int,
            collectionCount: null == collectionCount
                ? _value.collectionCount
                : collectionCount // ignore: cast_nullable_to_non_nullable
                      as int,
            isOnline: null == isOnline
                ? _value.isOnline
                : isOnline // ignore: cast_nullable_to_non_nullable
                      as bool,
            isFollowing: null == isFollowing
                ? _value.isFollowing
                : isFollowing // ignore: cast_nullable_to_non_nullable
                      as bool,
            isSelf: null == isSelf
                ? _value.isSelf
                : isSelf // ignore: cast_nullable_to_non_nullable
                      as bool,
            badges: null == badges
                ? _value.badges
                : badges // ignore: cast_nullable_to_non_nullable
                      as List<UserBadgePayload>,
            wornBadge: freezed == wornBadge
                ? _value.wornBadge
                : wornBadge // ignore: cast_nullable_to_non_nullable
                      as UserBadgePayload?,
            lastActiveTime: null == lastActiveTime
                ? _value.lastActiveTime
                : lastActiveTime // ignore: cast_nullable_to_non_nullable
                      as String,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }

  /// Create a copy of UserCardPayload
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
abstract class _$$UserCardPayloadImplCopyWith<$Res>
    implements $UserCardPayloadCopyWith<$Res> {
  factory _$$UserCardPayloadImplCopyWith(
    _$UserCardPayloadImpl value,
    $Res Function(_$UserCardPayloadImpl) then,
  ) = __$$UserCardPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int userId,
    String username,
    String nickname,
    String avatarUrl,
    String profileCoverUrl,
    String bio,
    String signature,
    String websiteName,
    String website,
    int prestige,
    Map<String, ExternalLinkPayload> externalInformation,
    bool isAdmin,
    int topicCount,
    int replyCount,
    int likeReceivedCount,
    int likeGivenCount,
    int followerCount,
    int followingCount,
    int collectionCount,
    bool isOnline,
    bool isFollowing,
    bool isSelf,
    List<UserBadgePayload> badges,
    UserBadgePayload? wornBadge,
    String lastActiveTime,
    String createdAt,
  });

  @override
  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class __$$UserCardPayloadImplCopyWithImpl<$Res>
    extends _$UserCardPayloadCopyWithImpl<$Res, _$UserCardPayloadImpl>
    implements _$$UserCardPayloadImplCopyWith<$Res> {
  __$$UserCardPayloadImplCopyWithImpl(
    _$UserCardPayloadImpl _value,
    $Res Function(_$UserCardPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserCardPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? userId = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? profileCoverUrl = null,
    Object? bio = null,
    Object? signature = null,
    Object? websiteName = null,
    Object? website = null,
    Object? prestige = null,
    Object? externalInformation = null,
    Object? isAdmin = null,
    Object? topicCount = null,
    Object? replyCount = null,
    Object? likeReceivedCount = null,
    Object? likeGivenCount = null,
    Object? followerCount = null,
    Object? followingCount = null,
    Object? collectionCount = null,
    Object? isOnline = null,
    Object? isFollowing = null,
    Object? isSelf = null,
    Object? badges = null,
    Object? wornBadge = freezed,
    Object? lastActiveTime = null,
    Object? createdAt = null,
  }) {
    return _then(
      _$UserCardPayloadImpl(
        userId: null == userId
            ? _value.userId
            : userId // ignore: cast_nullable_to_non_nullable
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
        profileCoverUrl: null == profileCoverUrl
            ? _value.profileCoverUrl
            : profileCoverUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        bio: null == bio
            ? _value.bio
            : bio // ignore: cast_nullable_to_non_nullable
                  as String,
        signature: null == signature
            ? _value.signature
            : signature // ignore: cast_nullable_to_non_nullable
                  as String,
        websiteName: null == websiteName
            ? _value.websiteName
            : websiteName // ignore: cast_nullable_to_non_nullable
                  as String,
        website: null == website
            ? _value.website
            : website // ignore: cast_nullable_to_non_nullable
                  as String,
        prestige: null == prestige
            ? _value.prestige
            : prestige // ignore: cast_nullable_to_non_nullable
                  as int,
        externalInformation: null == externalInformation
            ? _value._externalInformation
            : externalInformation // ignore: cast_nullable_to_non_nullable
                  as Map<String, ExternalLinkPayload>,
        isAdmin: null == isAdmin
            ? _value.isAdmin
            : isAdmin // ignore: cast_nullable_to_non_nullable
                  as bool,
        topicCount: null == topicCount
            ? _value.topicCount
            : topicCount // ignore: cast_nullable_to_non_nullable
                  as int,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        likeReceivedCount: null == likeReceivedCount
            ? _value.likeReceivedCount
            : likeReceivedCount // ignore: cast_nullable_to_non_nullable
                  as int,
        likeGivenCount: null == likeGivenCount
            ? _value.likeGivenCount
            : likeGivenCount // ignore: cast_nullable_to_non_nullable
                  as int,
        followerCount: null == followerCount
            ? _value.followerCount
            : followerCount // ignore: cast_nullable_to_non_nullable
                  as int,
        followingCount: null == followingCount
            ? _value.followingCount
            : followingCount // ignore: cast_nullable_to_non_nullable
                  as int,
        collectionCount: null == collectionCount
            ? _value.collectionCount
            : collectionCount // ignore: cast_nullable_to_non_nullable
                  as int,
        isOnline: null == isOnline
            ? _value.isOnline
            : isOnline // ignore: cast_nullable_to_non_nullable
                  as bool,
        isFollowing: null == isFollowing
            ? _value.isFollowing
            : isFollowing // ignore: cast_nullable_to_non_nullable
                  as bool,
        isSelf: null == isSelf
            ? _value.isSelf
            : isSelf // ignore: cast_nullable_to_non_nullable
                  as bool,
        badges: null == badges
            ? _value._badges
            : badges // ignore: cast_nullable_to_non_nullable
                  as List<UserBadgePayload>,
        wornBadge: freezed == wornBadge
            ? _value.wornBadge
            : wornBadge // ignore: cast_nullable_to_non_nullable
                  as UserBadgePayload?,
        lastActiveTime: null == lastActiveTime
            ? _value.lastActiveTime
            : lastActiveTime // ignore: cast_nullable_to_non_nullable
                  as String,
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
class _$UserCardPayloadImpl implements _UserCardPayload {
  const _$UserCardPayloadImpl({
    required this.userId,
    required this.username,
    required this.nickname,
    required this.avatarUrl,
    required this.profileCoverUrl,
    required this.bio,
    required this.signature,
    required this.websiteName,
    required this.website,
    required this.prestige,
    required final Map<String, ExternalLinkPayload> externalInformation,
    required this.isAdmin,
    required this.topicCount,
    required this.replyCount,
    required this.likeReceivedCount,
    required this.likeGivenCount,
    required this.followerCount,
    required this.followingCount,
    required this.collectionCount,
    required this.isOnline,
    required this.isFollowing,
    required this.isSelf,
    required final List<UserBadgePayload> badges,
    this.wornBadge,
    required this.lastActiveTime,
    required this.createdAt,
  }) : _externalInformation = externalInformation,
       _badges = badges;

  factory _$UserCardPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserCardPayloadImplFromJson(json);

  @override
  final int userId;
  @override
  final String username;
  @override
  final String nickname;
  @override
  final String avatarUrl;
  @override
  final String profileCoverUrl;
  @override
  final String bio;
  @override
  final String signature;
  @override
  final String websiteName;
  @override
  final String website;
  @override
  final int prestige;
  final Map<String, ExternalLinkPayload> _externalInformation;
  @override
  Map<String, ExternalLinkPayload> get externalInformation {
    if (_externalInformation is EqualUnmodifiableMapView)
      return _externalInformation;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(_externalInformation);
  }

  @override
  final bool isAdmin;
  @override
  final int topicCount;
  @override
  final int replyCount;
  @override
  final int likeReceivedCount;
  @override
  final int likeGivenCount;
  @override
  final int followerCount;
  @override
  final int followingCount;
  @override
  final int collectionCount;
  @override
  final bool isOnline;
  @override
  final bool isFollowing;
  @override
  final bool isSelf;
  final List<UserBadgePayload> _badges;
  @override
  List<UserBadgePayload> get badges {
    if (_badges is EqualUnmodifiableListView) return _badges;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_badges);
  }

  @override
  final UserBadgePayload? wornBadge;
  @override
  final String lastActiveTime;
  @override
  final String createdAt;

  @override
  String toString() {
    return 'UserCardPayload(userId: $userId, username: $username, nickname: $nickname, avatarUrl: $avatarUrl, profileCoverUrl: $profileCoverUrl, bio: $bio, signature: $signature, websiteName: $websiteName, website: $website, prestige: $prestige, externalInformation: $externalInformation, isAdmin: $isAdmin, topicCount: $topicCount, replyCount: $replyCount, likeReceivedCount: $likeReceivedCount, likeGivenCount: $likeGivenCount, followerCount: $followerCount, followingCount: $followingCount, collectionCount: $collectionCount, isOnline: $isOnline, isFollowing: $isFollowing, isSelf: $isSelf, badges: $badges, wornBadge: $wornBadge, lastActiveTime: $lastActiveTime, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserCardPayloadImpl &&
            (identical(other.userId, userId) || other.userId == userId) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.profileCoverUrl, profileCoverUrl) ||
                other.profileCoverUrl == profileCoverUrl) &&
            (identical(other.bio, bio) || other.bio == bio) &&
            (identical(other.signature, signature) ||
                other.signature == signature) &&
            (identical(other.websiteName, websiteName) ||
                other.websiteName == websiteName) &&
            (identical(other.website, website) || other.website == website) &&
            (identical(other.prestige, prestige) ||
                other.prestige == prestige) &&
            const DeepCollectionEquality().equals(
              other._externalInformation,
              _externalInformation,
            ) &&
            (identical(other.isAdmin, isAdmin) || other.isAdmin == isAdmin) &&
            (identical(other.topicCount, topicCount) ||
                other.topicCount == topicCount) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.likeReceivedCount, likeReceivedCount) ||
                other.likeReceivedCount == likeReceivedCount) &&
            (identical(other.likeGivenCount, likeGivenCount) ||
                other.likeGivenCount == likeGivenCount) &&
            (identical(other.followerCount, followerCount) ||
                other.followerCount == followerCount) &&
            (identical(other.followingCount, followingCount) ||
                other.followingCount == followingCount) &&
            (identical(other.collectionCount, collectionCount) ||
                other.collectionCount == collectionCount) &&
            (identical(other.isOnline, isOnline) ||
                other.isOnline == isOnline) &&
            (identical(other.isFollowing, isFollowing) ||
                other.isFollowing == isFollowing) &&
            (identical(other.isSelf, isSelf) || other.isSelf == isSelf) &&
            const DeepCollectionEquality().equals(other._badges, _badges) &&
            (identical(other.wornBadge, wornBadge) ||
                other.wornBadge == wornBadge) &&
            (identical(other.lastActiveTime, lastActiveTime) ||
                other.lastActiveTime == lastActiveTime) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hashAll([
    runtimeType,
    userId,
    username,
    nickname,
    avatarUrl,
    profileCoverUrl,
    bio,
    signature,
    websiteName,
    website,
    prestige,
    const DeepCollectionEquality().hash(_externalInformation),
    isAdmin,
    topicCount,
    replyCount,
    likeReceivedCount,
    likeGivenCount,
    followerCount,
    followingCount,
    collectionCount,
    isOnline,
    isFollowing,
    isSelf,
    const DeepCollectionEquality().hash(_badges),
    wornBadge,
    lastActiveTime,
    createdAt,
  ]);

  /// Create a copy of UserCardPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserCardPayloadImplCopyWith<_$UserCardPayloadImpl> get copyWith =>
      __$$UserCardPayloadImplCopyWithImpl<_$UserCardPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserCardPayloadImplToJson(this);
  }
}

abstract class _UserCardPayload implements UserCardPayload {
  const factory _UserCardPayload({
    required final int userId,
    required final String username,
    required final String nickname,
    required final String avatarUrl,
    required final String profileCoverUrl,
    required final String bio,
    required final String signature,
    required final String websiteName,
    required final String website,
    required final int prestige,
    required final Map<String, ExternalLinkPayload> externalInformation,
    required final bool isAdmin,
    required final int topicCount,
    required final int replyCount,
    required final int likeReceivedCount,
    required final int likeGivenCount,
    required final int followerCount,
    required final int followingCount,
    required final int collectionCount,
    required final bool isOnline,
    required final bool isFollowing,
    required final bool isSelf,
    required final List<UserBadgePayload> badges,
    final UserBadgePayload? wornBadge,
    required final String lastActiveTime,
    required final String createdAt,
  }) = _$UserCardPayloadImpl;

  factory _UserCardPayload.fromJson(Map<String, dynamic> json) =
      _$UserCardPayloadImpl.fromJson;

  @override
  int get userId;
  @override
  String get username;
  @override
  String get nickname;
  @override
  String get avatarUrl;
  @override
  String get profileCoverUrl;
  @override
  String get bio;
  @override
  String get signature;
  @override
  String get websiteName;
  @override
  String get website;
  @override
  int get prestige;
  @override
  Map<String, ExternalLinkPayload> get externalInformation;
  @override
  bool get isAdmin;
  @override
  int get topicCount;
  @override
  int get replyCount;
  @override
  int get likeReceivedCount;
  @override
  int get likeGivenCount;
  @override
  int get followerCount;
  @override
  int get followingCount;
  @override
  int get collectionCount;
  @override
  bool get isOnline;
  @override
  bool get isFollowing;
  @override
  bool get isSelf;
  @override
  List<UserBadgePayload> get badges;
  @override
  UserBadgePayload? get wornBadge;
  @override
  String get lastActiveTime;
  @override
  String get createdAt;

  /// Create a copy of UserCardPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserCardPayloadImplCopyWith<_$UserCardPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ExternalLinkPayload _$ExternalLinkPayloadFromJson(Map<String, dynamic> json) {
  return _ExternalLinkPayload.fromJson(json);
}

/// @nodoc
mixin _$ExternalLinkPayload {
  String? get link => throw _privateConstructorUsedError;

  /// Serializes this ExternalLinkPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ExternalLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ExternalLinkPayloadCopyWith<ExternalLinkPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ExternalLinkPayloadCopyWith<$Res> {
  factory $ExternalLinkPayloadCopyWith(
    ExternalLinkPayload value,
    $Res Function(ExternalLinkPayload) then,
  ) = _$ExternalLinkPayloadCopyWithImpl<$Res, ExternalLinkPayload>;
  @useResult
  $Res call({String? link});
}

/// @nodoc
class _$ExternalLinkPayloadCopyWithImpl<$Res, $Val extends ExternalLinkPayload>
    implements $ExternalLinkPayloadCopyWith<$Res> {
  _$ExternalLinkPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ExternalLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? link = freezed}) {
    return _then(
      _value.copyWith(
            link: freezed == link
                ? _value.link
                : link // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ExternalLinkPayloadImplCopyWith<$Res>
    implements $ExternalLinkPayloadCopyWith<$Res> {
  factory _$$ExternalLinkPayloadImplCopyWith(
    _$ExternalLinkPayloadImpl value,
    $Res Function(_$ExternalLinkPayloadImpl) then,
  ) = __$$ExternalLinkPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String? link});
}

/// @nodoc
class __$$ExternalLinkPayloadImplCopyWithImpl<$Res>
    extends _$ExternalLinkPayloadCopyWithImpl<$Res, _$ExternalLinkPayloadImpl>
    implements _$$ExternalLinkPayloadImplCopyWith<$Res> {
  __$$ExternalLinkPayloadImplCopyWithImpl(
    _$ExternalLinkPayloadImpl _value,
    $Res Function(_$ExternalLinkPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ExternalLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? link = freezed}) {
    return _then(
      _$ExternalLinkPayloadImpl(
        link: freezed == link
            ? _value.link
            : link // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ExternalLinkPayloadImpl implements _ExternalLinkPayload {
  const _$ExternalLinkPayloadImpl({this.link});

  factory _$ExternalLinkPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ExternalLinkPayloadImplFromJson(json);

  @override
  final String? link;

  @override
  String toString() {
    return 'ExternalLinkPayload(link: $link)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ExternalLinkPayloadImpl &&
            (identical(other.link, link) || other.link == link));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, link);

  /// Create a copy of ExternalLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ExternalLinkPayloadImplCopyWith<_$ExternalLinkPayloadImpl> get copyWith =>
      __$$ExternalLinkPayloadImplCopyWithImpl<_$ExternalLinkPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ExternalLinkPayloadImplToJson(this);
  }
}

abstract class _ExternalLinkPayload implements ExternalLinkPayload {
  const factory _ExternalLinkPayload({final String? link}) =
      _$ExternalLinkPayloadImpl;

  factory _ExternalLinkPayload.fromJson(Map<String, dynamic> json) =
      _$ExternalLinkPayloadImpl.fromJson;

  @override
  String? get link;

  /// Create a copy of ExternalLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ExternalLinkPayloadImplCopyWith<_$ExternalLinkPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserActivityPayload _$UserActivityPayloadFromJson(Map<String, dynamic> json) {
  return _UserActivityPayload.fromJson(json);
}

/// @nodoc
mixin _$UserActivityPayload {
  int get id => throw _privateConstructorUsedError;
  int get action => throw _privateConstructorUsedError;
  String get subjectType => throw _privateConstructorUsedError;
  int get subjectId => throw _privateConstructorUsedError;
  String get contentPreview => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;

  /// Serializes this UserActivityPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserActivityPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserActivityPayloadCopyWith<UserActivityPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserActivityPayloadCopyWith<$Res> {
  factory $UserActivityPayloadCopyWith(
    UserActivityPayload value,
    $Res Function(UserActivityPayload) then,
  ) = _$UserActivityPayloadCopyWithImpl<$Res, UserActivityPayload>;
  @useResult
  $Res call({
    int id,
    int action,
    String subjectType,
    int subjectId,
    String contentPreview,
    String url,
    String label,
    String createdAt,
  });
}

/// @nodoc
class _$UserActivityPayloadCopyWithImpl<$Res, $Val extends UserActivityPayload>
    implements $UserActivityPayloadCopyWith<$Res> {
  _$UserActivityPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserActivityPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? subjectType = null,
    Object? subjectId = null,
    Object? contentPreview = null,
    Object? url = null,
    Object? label = null,
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
                      as int,
            subjectType: null == subjectType
                ? _value.subjectType
                : subjectType // ignore: cast_nullable_to_non_nullable
                      as String,
            subjectId: null == subjectId
                ? _value.subjectId
                : subjectId // ignore: cast_nullable_to_non_nullable
                      as int,
            contentPreview: null == contentPreview
                ? _value.contentPreview
                : contentPreview // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserActivityPayloadImplCopyWith<$Res>
    implements $UserActivityPayloadCopyWith<$Res> {
  factory _$$UserActivityPayloadImplCopyWith(
    _$UserActivityPayloadImpl value,
    $Res Function(_$UserActivityPayloadImpl) then,
  ) = __$$UserActivityPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    int action,
    String subjectType,
    int subjectId,
    String contentPreview,
    String url,
    String label,
    String createdAt,
  });
}

/// @nodoc
class __$$UserActivityPayloadImplCopyWithImpl<$Res>
    extends _$UserActivityPayloadCopyWithImpl<$Res, _$UserActivityPayloadImpl>
    implements _$$UserActivityPayloadImplCopyWith<$Res> {
  __$$UserActivityPayloadImplCopyWithImpl(
    _$UserActivityPayloadImpl _value,
    $Res Function(_$UserActivityPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserActivityPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? action = null,
    Object? subjectType = null,
    Object? subjectId = null,
    Object? contentPreview = null,
    Object? url = null,
    Object? label = null,
    Object? createdAt = null,
  }) {
    return _then(
      _$UserActivityPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        action: null == action
            ? _value.action
            : action // ignore: cast_nullable_to_non_nullable
                  as int,
        subjectType: null == subjectType
            ? _value.subjectType
            : subjectType // ignore: cast_nullable_to_non_nullable
                  as String,
        subjectId: null == subjectId
            ? _value.subjectId
            : subjectId // ignore: cast_nullable_to_non_nullable
                  as int,
        contentPreview: null == contentPreview
            ? _value.contentPreview
            : contentPreview // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
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
class _$UserActivityPayloadImpl implements _UserActivityPayload {
  const _$UserActivityPayloadImpl({
    required this.id,
    required this.action,
    required this.subjectType,
    required this.subjectId,
    required this.contentPreview,
    required this.url,
    required this.label,
    required this.createdAt,
  });

  factory _$UserActivityPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserActivityPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int action;
  @override
  final String subjectType;
  @override
  final int subjectId;
  @override
  final String contentPreview;
  @override
  final String url;
  @override
  final String label;
  @override
  final String createdAt;

  @override
  String toString() {
    return 'UserActivityPayload(id: $id, action: $action, subjectType: $subjectType, subjectId: $subjectId, contentPreview: $contentPreview, url: $url, label: $label, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserActivityPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.action, action) || other.action == action) &&
            (identical(other.subjectType, subjectType) ||
                other.subjectType == subjectType) &&
            (identical(other.subjectId, subjectId) ||
                other.subjectId == subjectId) &&
            (identical(other.contentPreview, contentPreview) ||
                other.contentPreview == contentPreview) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    action,
    subjectType,
    subjectId,
    contentPreview,
    url,
    label,
    createdAt,
  );

  /// Create a copy of UserActivityPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserActivityPayloadImplCopyWith<_$UserActivityPayloadImpl> get copyWith =>
      __$$UserActivityPayloadImplCopyWithImpl<_$UserActivityPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserActivityPayloadImplToJson(this);
  }
}

abstract class _UserActivityPayload implements UserActivityPayload {
  const factory _UserActivityPayload({
    required final int id,
    required final int action,
    required final String subjectType,
    required final int subjectId,
    required final String contentPreview,
    required final String url,
    required final String label,
    required final String createdAt,
  }) = _$UserActivityPayloadImpl;

  factory _UserActivityPayload.fromJson(Map<String, dynamic> json) =
      _$UserActivityPayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get action;
  @override
  String get subjectType;
  @override
  int get subjectId;
  @override
  String get contentPreview;
  @override
  String get url;
  @override
  String get label;
  @override
  String get createdAt;

  /// Create a copy of UserActivityPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserActivityPayloadImplCopyWith<_$UserActivityPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserLikePayload _$UserLikePayloadFromJson(Map<String, dynamic> json) {
  return _UserLikePayload.fromJson(json);
}

/// @nodoc
mixin _$UserLikePayload {
  int get id => throw _privateConstructorUsedError;
  int get topicId => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get likedAt => throw _privateConstructorUsedError;

  /// Serializes this UserLikePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserLikePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserLikePayloadCopyWith<UserLikePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserLikePayloadCopyWith<$Res> {
  factory $UserLikePayloadCopyWith(
    UserLikePayload value,
    $Res Function(UserLikePayload) then,
  ) = _$UserLikePayloadCopyWithImpl<$Res, UserLikePayload>;
  @useResult
  $Res call({int id, int topicId, String title, String url, String likedAt});
}

/// @nodoc
class _$UserLikePayloadCopyWithImpl<$Res, $Val extends UserLikePayload>
    implements $UserLikePayloadCopyWith<$Res> {
  _$UserLikePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserLikePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? title = null,
    Object? url = null,
    Object? likedAt = null,
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
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            likedAt: null == likedAt
                ? _value.likedAt
                : likedAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserLikePayloadImplCopyWith<$Res>
    implements $UserLikePayloadCopyWith<$Res> {
  factory _$$UserLikePayloadImplCopyWith(
    _$UserLikePayloadImpl value,
    $Res Function(_$UserLikePayloadImpl) then,
  ) = __$$UserLikePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, int topicId, String title, String url, String likedAt});
}

/// @nodoc
class __$$UserLikePayloadImplCopyWithImpl<$Res>
    extends _$UserLikePayloadCopyWithImpl<$Res, _$UserLikePayloadImpl>
    implements _$$UserLikePayloadImplCopyWith<$Res> {
  __$$UserLikePayloadImplCopyWithImpl(
    _$UserLikePayloadImpl _value,
    $Res Function(_$UserLikePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserLikePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? topicId = null,
    Object? title = null,
    Object? url = null,
    Object? likedAt = null,
  }) {
    return _then(
      _$UserLikePayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        topicId: null == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        likedAt: null == likedAt
            ? _value.likedAt
            : likedAt // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserLikePayloadImpl implements _UserLikePayload {
  const _$UserLikePayloadImpl({
    required this.id,
    required this.topicId,
    required this.title,
    required this.url,
    required this.likedAt,
  });

  factory _$UserLikePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserLikePayloadImplFromJson(json);

  @override
  final int id;
  @override
  final int topicId;
  @override
  final String title;
  @override
  final String url;
  @override
  final String likedAt;

  @override
  String toString() {
    return 'UserLikePayload(id: $id, topicId: $topicId, title: $title, url: $url, likedAt: $likedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserLikePayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.likedAt, likedAt) || other.likedAt == likedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, topicId, title, url, likedAt);

  /// Create a copy of UserLikePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserLikePayloadImplCopyWith<_$UserLikePayloadImpl> get copyWith =>
      __$$UserLikePayloadImplCopyWithImpl<_$UserLikePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserLikePayloadImplToJson(this);
  }
}

abstract class _UserLikePayload implements UserLikePayload {
  const factory _UserLikePayload({
    required final int id,
    required final int topicId,
    required final String title,
    required final String url,
    required final String likedAt,
  }) = _$UserLikePayloadImpl;

  factory _UserLikePayload.fromJson(Map<String, dynamic> json) =
      _$UserLikePayloadImpl.fromJson;

  @override
  int get id;
  @override
  int get topicId;
  @override
  String get title;
  @override
  String get url;
  @override
  String get likedAt;

  /// Create a copy of UserLikePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserLikePayloadImplCopyWith<_$UserLikePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserBookmarkPayload _$UserBookmarkPayloadFromJson(Map<String, dynamic> json) {
  return _UserBookmarkPayload.fromJson(json);
}

/// @nodoc
mixin _$UserBookmarkPayload {
  int get id => throw _privateConstructorUsedError;
  String get type => throw _privateConstructorUsedError;
  int get topicId => throw _privateConstructorUsedError;
  int? get postId => throw _privateConstructorUsedError;
  int? get postNo => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String? get excerpt => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get bookmarkedAt => throw _privateConstructorUsedError;

  /// Serializes this UserBookmarkPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserBookmarkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserBookmarkPayloadCopyWith<UserBookmarkPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserBookmarkPayloadCopyWith<$Res> {
  factory $UserBookmarkPayloadCopyWith(
    UserBookmarkPayload value,
    $Res Function(UserBookmarkPayload) then,
  ) = _$UserBookmarkPayloadCopyWithImpl<$Res, UserBookmarkPayload>;
  @useResult
  $Res call({
    int id,
    String type,
    int topicId,
    int? postId,
    int? postNo,
    String title,
    String? excerpt,
    String url,
    String bookmarkedAt,
  });
}

/// @nodoc
class _$UserBookmarkPayloadCopyWithImpl<$Res, $Val extends UserBookmarkPayload>
    implements $UserBookmarkPayloadCopyWith<$Res> {
  _$UserBookmarkPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserBookmarkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? type = null,
    Object? topicId = null,
    Object? postId = freezed,
    Object? postNo = freezed,
    Object? title = null,
    Object? excerpt = freezed,
    Object? url = null,
    Object? bookmarkedAt = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            type: null == type
                ? _value.type
                : type // ignore: cast_nullable_to_non_nullable
                      as String,
            topicId: null == topicId
                ? _value.topicId
                : topicId // ignore: cast_nullable_to_non_nullable
                      as int,
            postId: freezed == postId
                ? _value.postId
                : postId // ignore: cast_nullable_to_non_nullable
                      as int?,
            postNo: freezed == postNo
                ? _value.postNo
                : postNo // ignore: cast_nullable_to_non_nullable
                      as int?,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            excerpt: freezed == excerpt
                ? _value.excerpt
                : excerpt // ignore: cast_nullable_to_non_nullable
                      as String?,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            bookmarkedAt: null == bookmarkedAt
                ? _value.bookmarkedAt
                : bookmarkedAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserBookmarkPayloadImplCopyWith<$Res>
    implements $UserBookmarkPayloadCopyWith<$Res> {
  factory _$$UserBookmarkPayloadImplCopyWith(
    _$UserBookmarkPayloadImpl value,
    $Res Function(_$UserBookmarkPayloadImpl) then,
  ) = __$$UserBookmarkPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String type,
    int topicId,
    int? postId,
    int? postNo,
    String title,
    String? excerpt,
    String url,
    String bookmarkedAt,
  });
}

/// @nodoc
class __$$UserBookmarkPayloadImplCopyWithImpl<$Res>
    extends _$UserBookmarkPayloadCopyWithImpl<$Res, _$UserBookmarkPayloadImpl>
    implements _$$UserBookmarkPayloadImplCopyWith<$Res> {
  __$$UserBookmarkPayloadImplCopyWithImpl(
    _$UserBookmarkPayloadImpl _value,
    $Res Function(_$UserBookmarkPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserBookmarkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? type = null,
    Object? topicId = null,
    Object? postId = freezed,
    Object? postNo = freezed,
    Object? title = null,
    Object? excerpt = freezed,
    Object? url = null,
    Object? bookmarkedAt = null,
  }) {
    return _then(
      _$UserBookmarkPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        type: null == type
            ? _value.type
            : type // ignore: cast_nullable_to_non_nullable
                  as String,
        topicId: null == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int,
        postId: freezed == postId
            ? _value.postId
            : postId // ignore: cast_nullable_to_non_nullable
                  as int?,
        postNo: freezed == postNo
            ? _value.postNo
            : postNo // ignore: cast_nullable_to_non_nullable
                  as int?,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        excerpt: freezed == excerpt
            ? _value.excerpt
            : excerpt // ignore: cast_nullable_to_non_nullable
                  as String?,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        bookmarkedAt: null == bookmarkedAt
            ? _value.bookmarkedAt
            : bookmarkedAt // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserBookmarkPayloadImpl implements _UserBookmarkPayload {
  const _$UserBookmarkPayloadImpl({
    required this.id,
    required this.type,
    required this.topicId,
    this.postId,
    this.postNo,
    required this.title,
    this.excerpt,
    required this.url,
    required this.bookmarkedAt,
  });

  factory _$UserBookmarkPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserBookmarkPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String type;
  @override
  final int topicId;
  @override
  final int? postId;
  @override
  final int? postNo;
  @override
  final String title;
  @override
  final String? excerpt;
  @override
  final String url;
  @override
  final String bookmarkedAt;

  @override
  String toString() {
    return 'UserBookmarkPayload(id: $id, type: $type, topicId: $topicId, postId: $postId, postNo: $postNo, title: $title, excerpt: $excerpt, url: $url, bookmarkedAt: $bookmarkedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserBookmarkPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.postId, postId) || other.postId == postId) &&
            (identical(other.postNo, postNo) || other.postNo == postNo) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.excerpt, excerpt) || other.excerpt == excerpt) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.bookmarkedAt, bookmarkedAt) ||
                other.bookmarkedAt == bookmarkedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    type,
    topicId,
    postId,
    postNo,
    title,
    excerpt,
    url,
    bookmarkedAt,
  );

  /// Create a copy of UserBookmarkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserBookmarkPayloadImplCopyWith<_$UserBookmarkPayloadImpl> get copyWith =>
      __$$UserBookmarkPayloadImplCopyWithImpl<_$UserBookmarkPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserBookmarkPayloadImplToJson(this);
  }
}

abstract class _UserBookmarkPayload implements UserBookmarkPayload {
  const factory _UserBookmarkPayload({
    required final int id,
    required final String type,
    required final int topicId,
    final int? postId,
    final int? postNo,
    required final String title,
    final String? excerpt,
    required final String url,
    required final String bookmarkedAt,
  }) = _$UserBookmarkPayloadImpl;

  factory _UserBookmarkPayload.fromJson(Map<String, dynamic> json) =
      _$UserBookmarkPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get type;
  @override
  int get topicId;
  @override
  int? get postId;
  @override
  int? get postNo;
  @override
  String get title;
  @override
  String? get excerpt;
  @override
  String get url;
  @override
  String get bookmarkedAt;

  /// Create a copy of UserBookmarkPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserBookmarkPayloadImplCopyWith<_$UserBookmarkPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserConnectionPayload _$UserConnectionPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _UserConnectionPayload.fromJson(json);
}

/// @nodoc
mixin _$UserConnectionPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get nickname => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get bio => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;

  /// Serializes this UserConnectionPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserConnectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserConnectionPayloadCopyWith<UserConnectionPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserConnectionPayloadCopyWith<$Res> {
  factory $UserConnectionPayloadCopyWith(
    UserConnectionPayload value,
    $Res Function(UserConnectionPayload) then,
  ) = _$UserConnectionPayloadCopyWithImpl<$Res, UserConnectionPayload>;
  @useResult
  $Res call({
    int id,
    String username,
    String nickname,
    String avatarUrl,
    String bio,
    String url,
  });
}

/// @nodoc
class _$UserConnectionPayloadCopyWithImpl<
  $Res,
  $Val extends UserConnectionPayload
>
    implements $UserConnectionPayloadCopyWith<$Res> {
  _$UserConnectionPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserConnectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? bio = null,
    Object? url = null,
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
abstract class _$$UserConnectionPayloadImplCopyWith<$Res>
    implements $UserConnectionPayloadCopyWith<$Res> {
  factory _$$UserConnectionPayloadImplCopyWith(
    _$UserConnectionPayloadImpl value,
    $Res Function(_$UserConnectionPayloadImpl) then,
  ) = __$$UserConnectionPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String username,
    String nickname,
    String avatarUrl,
    String bio,
    String url,
  });
}

/// @nodoc
class __$$UserConnectionPayloadImplCopyWithImpl<$Res>
    extends
        _$UserConnectionPayloadCopyWithImpl<$Res, _$UserConnectionPayloadImpl>
    implements _$$UserConnectionPayloadImplCopyWith<$Res> {
  __$$UserConnectionPayloadImplCopyWithImpl(
    _$UserConnectionPayloadImpl _value,
    $Res Function(_$UserConnectionPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserConnectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? nickname = null,
    Object? avatarUrl = null,
    Object? bio = null,
    Object? url = null,
  }) {
    return _then(
      _$UserConnectionPayloadImpl(
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
class _$UserConnectionPayloadImpl implements _UserConnectionPayload {
  const _$UserConnectionPayloadImpl({
    required this.id,
    required this.username,
    required this.nickname,
    required this.avatarUrl,
    required this.bio,
    required this.url,
  });

  factory _$UserConnectionPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserConnectionPayloadImplFromJson(json);

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
  final String url;

  @override
  String toString() {
    return 'UserConnectionPayload(id: $id, username: $username, nickname: $nickname, avatarUrl: $avatarUrl, bio: $bio, url: $url)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserConnectionPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.bio, bio) || other.bio == bio) &&
            (identical(other.url, url) || other.url == url));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, username, nickname, avatarUrl, bio, url);

  /// Create a copy of UserConnectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserConnectionPayloadImplCopyWith<_$UserConnectionPayloadImpl>
  get copyWith =>
      __$$UserConnectionPayloadImplCopyWithImpl<_$UserConnectionPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserConnectionPayloadImplToJson(this);
  }
}

abstract class _UserConnectionPayload implements UserConnectionPayload {
  const factory _UserConnectionPayload({
    required final int id,
    required final String username,
    required final String nickname,
    required final String avatarUrl,
    required final String bio,
    required final String url,
  }) = _$UserConnectionPayloadImpl;

  factory _UserConnectionPayload.fromJson(Map<String, dynamic> json) =
      _$UserConnectionPayloadImpl.fromJson;

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
  @override
  String get url;

  /// Create a copy of UserConnectionPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserConnectionPayloadImplCopyWith<_$UserConnectionPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

UserProfileProps _$UserProfilePropsFromJson(Map<String, dynamic> json) {
  return _UserProfileProps.fromJson(json);
}

/// @nodoc
mixin _$UserProfileProps {
  UserCardPayload get user => throw _privateConstructorUsedError;
  String get section => throw _privateConstructorUsedError;
  String get activityTab => throw _privateConstructorUsedError;
  List<TabItemPayload> get tabs => throw _privateConstructorUsedError;
  List<TabItemPayload> get activityTabs => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;
  List<UserBadgePayload> get badges => throw _privateConstructorUsedError;
  List<TopicPayload> get topics => throw _privateConstructorUsedError;
  List<UserActivityPayload> get activities =>
      throw _privateConstructorUsedError;
  List<UserLikePayload> get likes => throw _privateConstructorUsedError;
  List<UserBookmarkPayload> get bookmarks => throw _privateConstructorUsedError;
  List<UserConnectionPayload> get following =>
      throw _privateConstructorUsedError;
  List<UserConnectionPayload> get followers =>
      throw _privateConstructorUsedError;
  bool get isOwnProfile => throw _privateConstructorUsedError;
  bool get canMessage => throw _privateConstructorUsedError;
  bool get canFollow => throw _privateConstructorUsedError;
  String get messageUrl => throw _privateConstructorUsedError;
  String get settingsUrl => throw _privateConstructorUsedError;

  /// Serializes this UserProfileProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserProfilePropsCopyWith<UserProfileProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserProfilePropsCopyWith<$Res> {
  factory $UserProfilePropsCopyWith(
    UserProfileProps value,
    $Res Function(UserProfileProps) then,
  ) = _$UserProfilePropsCopyWithImpl<$Res, UserProfileProps>;
  @useResult
  $Res call({
    UserCardPayload user,
    String section,
    String activityTab,
    List<TabItemPayload> tabs,
    List<TabItemPayload> activityTabs,
    PaginationPayload pagination,
    List<UserBadgePayload> badges,
    List<TopicPayload> topics,
    List<UserActivityPayload> activities,
    List<UserLikePayload> likes,
    List<UserBookmarkPayload> bookmarks,
    List<UserConnectionPayload> following,
    List<UserConnectionPayload> followers,
    bool isOwnProfile,
    bool canMessage,
    bool canFollow,
    String messageUrl,
    String settingsUrl,
  });

  $UserCardPayloadCopyWith<$Res> get user;
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$UserProfilePropsCopyWithImpl<$Res, $Val extends UserProfileProps>
    implements $UserProfilePropsCopyWith<$Res> {
  _$UserProfilePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? user = null,
    Object? section = null,
    Object? activityTab = null,
    Object? tabs = null,
    Object? activityTabs = null,
    Object? pagination = null,
    Object? badges = null,
    Object? topics = null,
    Object? activities = null,
    Object? likes = null,
    Object? bookmarks = null,
    Object? following = null,
    Object? followers = null,
    Object? isOwnProfile = null,
    Object? canMessage = null,
    Object? canFollow = null,
    Object? messageUrl = null,
    Object? settingsUrl = null,
  }) {
    return _then(
      _value.copyWith(
            user: null == user
                ? _value.user
                : user // ignore: cast_nullable_to_non_nullable
                      as UserCardPayload,
            section: null == section
                ? _value.section
                : section // ignore: cast_nullable_to_non_nullable
                      as String,
            activityTab: null == activityTab
                ? _value.activityTab
                : activityTab // ignore: cast_nullable_to_non_nullable
                      as String,
            tabs: null == tabs
                ? _value.tabs
                : tabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
            activityTabs: null == activityTabs
                ? _value.activityTabs
                : activityTabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
            badges: null == badges
                ? _value.badges
                : badges // ignore: cast_nullable_to_non_nullable
                      as List<UserBadgePayload>,
            topics: null == topics
                ? _value.topics
                : topics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            activities: null == activities
                ? _value.activities
                : activities // ignore: cast_nullable_to_non_nullable
                      as List<UserActivityPayload>,
            likes: null == likes
                ? _value.likes
                : likes // ignore: cast_nullable_to_non_nullable
                      as List<UserLikePayload>,
            bookmarks: null == bookmarks
                ? _value.bookmarks
                : bookmarks // ignore: cast_nullable_to_non_nullable
                      as List<UserBookmarkPayload>,
            following: null == following
                ? _value.following
                : following // ignore: cast_nullable_to_non_nullable
                      as List<UserConnectionPayload>,
            followers: null == followers
                ? _value.followers
                : followers // ignore: cast_nullable_to_non_nullable
                      as List<UserConnectionPayload>,
            isOwnProfile: null == isOwnProfile
                ? _value.isOwnProfile
                : isOwnProfile // ignore: cast_nullable_to_non_nullable
                      as bool,
            canMessage: null == canMessage
                ? _value.canMessage
                : canMessage // ignore: cast_nullable_to_non_nullable
                      as bool,
            canFollow: null == canFollow
                ? _value.canFollow
                : canFollow // ignore: cast_nullable_to_non_nullable
                      as bool,
            messageUrl: null == messageUrl
                ? _value.messageUrl
                : messageUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            settingsUrl: null == settingsUrl
                ? _value.settingsUrl
                : settingsUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UserCardPayloadCopyWith<$Res> get user {
    return $UserCardPayloadCopyWith<$Res>(_value.user, (value) {
      return _then(_value.copyWith(user: value) as $Val);
    });
  }

  /// Create a copy of UserProfileProps
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
abstract class _$$UserProfilePropsImplCopyWith<$Res>
    implements $UserProfilePropsCopyWith<$Res> {
  factory _$$UserProfilePropsImplCopyWith(
    _$UserProfilePropsImpl value,
    $Res Function(_$UserProfilePropsImpl) then,
  ) = __$$UserProfilePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    UserCardPayload user,
    String section,
    String activityTab,
    List<TabItemPayload> tabs,
    List<TabItemPayload> activityTabs,
    PaginationPayload pagination,
    List<UserBadgePayload> badges,
    List<TopicPayload> topics,
    List<UserActivityPayload> activities,
    List<UserLikePayload> likes,
    List<UserBookmarkPayload> bookmarks,
    List<UserConnectionPayload> following,
    List<UserConnectionPayload> followers,
    bool isOwnProfile,
    bool canMessage,
    bool canFollow,
    String messageUrl,
    String settingsUrl,
  });

  @override
  $UserCardPayloadCopyWith<$Res> get user;
  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$UserProfilePropsImplCopyWithImpl<$Res>
    extends _$UserProfilePropsCopyWithImpl<$Res, _$UserProfilePropsImpl>
    implements _$$UserProfilePropsImplCopyWith<$Res> {
  __$$UserProfilePropsImplCopyWithImpl(
    _$UserProfilePropsImpl _value,
    $Res Function(_$UserProfilePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? user = null,
    Object? section = null,
    Object? activityTab = null,
    Object? tabs = null,
    Object? activityTabs = null,
    Object? pagination = null,
    Object? badges = null,
    Object? topics = null,
    Object? activities = null,
    Object? likes = null,
    Object? bookmarks = null,
    Object? following = null,
    Object? followers = null,
    Object? isOwnProfile = null,
    Object? canMessage = null,
    Object? canFollow = null,
    Object? messageUrl = null,
    Object? settingsUrl = null,
  }) {
    return _then(
      _$UserProfilePropsImpl(
        user: null == user
            ? _value.user
            : user // ignore: cast_nullable_to_non_nullable
                  as UserCardPayload,
        section: null == section
            ? _value.section
            : section // ignore: cast_nullable_to_non_nullable
                  as String,
        activityTab: null == activityTab
            ? _value.activityTab
            : activityTab // ignore: cast_nullable_to_non_nullable
                  as String,
        tabs: null == tabs
            ? _value._tabs
            : tabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
        activityTabs: null == activityTabs
            ? _value._activityTabs
            : activityTabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
        badges: null == badges
            ? _value._badges
            : badges // ignore: cast_nullable_to_non_nullable
                  as List<UserBadgePayload>,
        topics: null == topics
            ? _value._topics
            : topics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        activities: null == activities
            ? _value._activities
            : activities // ignore: cast_nullable_to_non_nullable
                  as List<UserActivityPayload>,
        likes: null == likes
            ? _value._likes
            : likes // ignore: cast_nullable_to_non_nullable
                  as List<UserLikePayload>,
        bookmarks: null == bookmarks
            ? _value._bookmarks
            : bookmarks // ignore: cast_nullable_to_non_nullable
                  as List<UserBookmarkPayload>,
        following: null == following
            ? _value._following
            : following // ignore: cast_nullable_to_non_nullable
                  as List<UserConnectionPayload>,
        followers: null == followers
            ? _value._followers
            : followers // ignore: cast_nullable_to_non_nullable
                  as List<UserConnectionPayload>,
        isOwnProfile: null == isOwnProfile
            ? _value.isOwnProfile
            : isOwnProfile // ignore: cast_nullable_to_non_nullable
                  as bool,
        canMessage: null == canMessage
            ? _value.canMessage
            : canMessage // ignore: cast_nullable_to_non_nullable
                  as bool,
        canFollow: null == canFollow
            ? _value.canFollow
            : canFollow // ignore: cast_nullable_to_non_nullable
                  as bool,
        messageUrl: null == messageUrl
            ? _value.messageUrl
            : messageUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        settingsUrl: null == settingsUrl
            ? _value.settingsUrl
            : settingsUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserProfilePropsImpl implements _UserProfileProps {
  const _$UserProfilePropsImpl({
    required this.user,
    required this.section,
    required this.activityTab,
    required final List<TabItemPayload> tabs,
    required final List<TabItemPayload> activityTabs,
    required this.pagination,
    required final List<UserBadgePayload> badges,
    required final List<TopicPayload> topics,
    required final List<UserActivityPayload> activities,
    required final List<UserLikePayload> likes,
    required final List<UserBookmarkPayload> bookmarks,
    required final List<UserConnectionPayload> following,
    required final List<UserConnectionPayload> followers,
    required this.isOwnProfile,
    required this.canMessage,
    required this.canFollow,
    required this.messageUrl,
    required this.settingsUrl,
  }) : _tabs = tabs,
       _activityTabs = activityTabs,
       _badges = badges,
       _topics = topics,
       _activities = activities,
       _likes = likes,
       _bookmarks = bookmarks,
       _following = following,
       _followers = followers;

  factory _$UserProfilePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserProfilePropsImplFromJson(json);

  @override
  final UserCardPayload user;
  @override
  final String section;
  @override
  final String activityTab;
  final List<TabItemPayload> _tabs;
  @override
  List<TabItemPayload> get tabs {
    if (_tabs is EqualUnmodifiableListView) return _tabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tabs);
  }

  final List<TabItemPayload> _activityTabs;
  @override
  List<TabItemPayload> get activityTabs {
    if (_activityTabs is EqualUnmodifiableListView) return _activityTabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_activityTabs);
  }

  @override
  final PaginationPayload pagination;
  final List<UserBadgePayload> _badges;
  @override
  List<UserBadgePayload> get badges {
    if (_badges is EqualUnmodifiableListView) return _badges;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_badges);
  }

  final List<TopicPayload> _topics;
  @override
  List<TopicPayload> get topics {
    if (_topics is EqualUnmodifiableListView) return _topics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_topics);
  }

  final List<UserActivityPayload> _activities;
  @override
  List<UserActivityPayload> get activities {
    if (_activities is EqualUnmodifiableListView) return _activities;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_activities);
  }

  final List<UserLikePayload> _likes;
  @override
  List<UserLikePayload> get likes {
    if (_likes is EqualUnmodifiableListView) return _likes;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_likes);
  }

  final List<UserBookmarkPayload> _bookmarks;
  @override
  List<UserBookmarkPayload> get bookmarks {
    if (_bookmarks is EqualUnmodifiableListView) return _bookmarks;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_bookmarks);
  }

  final List<UserConnectionPayload> _following;
  @override
  List<UserConnectionPayload> get following {
    if (_following is EqualUnmodifiableListView) return _following;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_following);
  }

  final List<UserConnectionPayload> _followers;
  @override
  List<UserConnectionPayload> get followers {
    if (_followers is EqualUnmodifiableListView) return _followers;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_followers);
  }

  @override
  final bool isOwnProfile;
  @override
  final bool canMessage;
  @override
  final bool canFollow;
  @override
  final String messageUrl;
  @override
  final String settingsUrl;

  @override
  String toString() {
    return 'UserProfileProps(user: $user, section: $section, activityTab: $activityTab, tabs: $tabs, activityTabs: $activityTabs, pagination: $pagination, badges: $badges, topics: $topics, activities: $activities, likes: $likes, bookmarks: $bookmarks, following: $following, followers: $followers, isOwnProfile: $isOwnProfile, canMessage: $canMessage, canFollow: $canFollow, messageUrl: $messageUrl, settingsUrl: $settingsUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserProfilePropsImpl &&
            (identical(other.user, user) || other.user == user) &&
            (identical(other.section, section) || other.section == section) &&
            (identical(other.activityTab, activityTab) ||
                other.activityTab == activityTab) &&
            const DeepCollectionEquality().equals(other._tabs, _tabs) &&
            const DeepCollectionEquality().equals(
              other._activityTabs,
              _activityTabs,
            ) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination) &&
            const DeepCollectionEquality().equals(other._badges, _badges) &&
            const DeepCollectionEquality().equals(other._topics, _topics) &&
            const DeepCollectionEquality().equals(
              other._activities,
              _activities,
            ) &&
            const DeepCollectionEquality().equals(other._likes, _likes) &&
            const DeepCollectionEquality().equals(
              other._bookmarks,
              _bookmarks,
            ) &&
            const DeepCollectionEquality().equals(
              other._following,
              _following,
            ) &&
            const DeepCollectionEquality().equals(
              other._followers,
              _followers,
            ) &&
            (identical(other.isOwnProfile, isOwnProfile) ||
                other.isOwnProfile == isOwnProfile) &&
            (identical(other.canMessage, canMessage) ||
                other.canMessage == canMessage) &&
            (identical(other.canFollow, canFollow) ||
                other.canFollow == canFollow) &&
            (identical(other.messageUrl, messageUrl) ||
                other.messageUrl == messageUrl) &&
            (identical(other.settingsUrl, settingsUrl) ||
                other.settingsUrl == settingsUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    user,
    section,
    activityTab,
    const DeepCollectionEquality().hash(_tabs),
    const DeepCollectionEquality().hash(_activityTabs),
    pagination,
    const DeepCollectionEquality().hash(_badges),
    const DeepCollectionEquality().hash(_topics),
    const DeepCollectionEquality().hash(_activities),
    const DeepCollectionEquality().hash(_likes),
    const DeepCollectionEquality().hash(_bookmarks),
    const DeepCollectionEquality().hash(_following),
    const DeepCollectionEquality().hash(_followers),
    isOwnProfile,
    canMessage,
    canFollow,
    messageUrl,
    settingsUrl,
  );

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserProfilePropsImplCopyWith<_$UserProfilePropsImpl> get copyWith =>
      __$$UserProfilePropsImplCopyWithImpl<_$UserProfilePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserProfilePropsImplToJson(this);
  }
}

abstract class _UserProfileProps implements UserProfileProps {
  const factory _UserProfileProps({
    required final UserCardPayload user,
    required final String section,
    required final String activityTab,
    required final List<TabItemPayload> tabs,
    required final List<TabItemPayload> activityTabs,
    required final PaginationPayload pagination,
    required final List<UserBadgePayload> badges,
    required final List<TopicPayload> topics,
    required final List<UserActivityPayload> activities,
    required final List<UserLikePayload> likes,
    required final List<UserBookmarkPayload> bookmarks,
    required final List<UserConnectionPayload> following,
    required final List<UserConnectionPayload> followers,
    required final bool isOwnProfile,
    required final bool canMessage,
    required final bool canFollow,
    required final String messageUrl,
    required final String settingsUrl,
  }) = _$UserProfilePropsImpl;

  factory _UserProfileProps.fromJson(Map<String, dynamic> json) =
      _$UserProfilePropsImpl.fromJson;

  @override
  UserCardPayload get user;
  @override
  String get section;
  @override
  String get activityTab;
  @override
  List<TabItemPayload> get tabs;
  @override
  List<TabItemPayload> get activityTabs;
  @override
  PaginationPayload get pagination;
  @override
  List<UserBadgePayload> get badges;
  @override
  List<TopicPayload> get topics;
  @override
  List<UserActivityPayload> get activities;
  @override
  List<UserLikePayload> get likes;
  @override
  List<UserBookmarkPayload> get bookmarks;
  @override
  List<UserConnectionPayload> get following;
  @override
  List<UserConnectionPayload> get followers;
  @override
  bool get isOwnProfile;
  @override
  bool get canMessage;
  @override
  bool get canFollow;
  @override
  String get messageUrl;
  @override
  String get settingsUrl;

  /// Create a copy of UserProfileProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserProfilePropsImplCopyWith<_$UserProfilePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SettingsUserPayload _$SettingsUserPayloadFromJson(Map<String, dynamic> json) {
  return _SettingsUserPayload.fromJson(json);
}

/// @nodoc
mixin _$SettingsUserPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get email => throw _privateConstructorUsedError;
  String get nickname => throw _privateConstructorUsedError;
  String get locale => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  String get profileCoverUrl => throw _privateConstructorUsedError;
  String get bio => throw _privateConstructorUsedError;
  String get signature => throw _privateConstructorUsedError;
  String get websiteName => throw _privateConstructorUsedError;
  String get website => throw _privateConstructorUsedError;
  int get prestige => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;
  Map<String, ExternalLinkPayload> get externalInformation =>
      throw _privateConstructorUsedError;
  String get wornBadgeCode => throw _privateConstructorUsedError;
  List<UserBadgePayload> get badges => throw _privateConstructorUsedError;
  List<UserBadgePayload> get wearableBadges =>
      throw _privateConstructorUsedError;
  UserBadgePayload? get wornBadge => throw _privateConstructorUsedError;

  /// Serializes this SettingsUserPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SettingsUserPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SettingsUserPayloadCopyWith<SettingsUserPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SettingsUserPayloadCopyWith<$Res> {
  factory $SettingsUserPayloadCopyWith(
    SettingsUserPayload value,
    $Res Function(SettingsUserPayload) then,
  ) = _$SettingsUserPayloadCopyWithImpl<$Res, SettingsUserPayload>;
  @useResult
  $Res call({
    int id,
    String username,
    String email,
    String nickname,
    String locale,
    String avatarUrl,
    String profileCoverUrl,
    String bio,
    String signature,
    String websiteName,
    String website,
    int prestige,
    String createdAt,
    Map<String, ExternalLinkPayload> externalInformation,
    String wornBadgeCode,
    List<UserBadgePayload> badges,
    List<UserBadgePayload> wearableBadges,
    UserBadgePayload? wornBadge,
  });

  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class _$SettingsUserPayloadCopyWithImpl<$Res, $Val extends SettingsUserPayload>
    implements $SettingsUserPayloadCopyWith<$Res> {
  _$SettingsUserPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SettingsUserPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? email = null,
    Object? nickname = null,
    Object? locale = null,
    Object? avatarUrl = null,
    Object? profileCoverUrl = null,
    Object? bio = null,
    Object? signature = null,
    Object? websiteName = null,
    Object? website = null,
    Object? prestige = null,
    Object? createdAt = null,
    Object? externalInformation = null,
    Object? wornBadgeCode = null,
    Object? badges = null,
    Object? wearableBadges = null,
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
            email: null == email
                ? _value.email
                : email // ignore: cast_nullable_to_non_nullable
                      as String,
            nickname: null == nickname
                ? _value.nickname
                : nickname // ignore: cast_nullable_to_non_nullable
                      as String,
            locale: null == locale
                ? _value.locale
                : locale // ignore: cast_nullable_to_non_nullable
                      as String,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            profileCoverUrl: null == profileCoverUrl
                ? _value.profileCoverUrl
                : profileCoverUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            bio: null == bio
                ? _value.bio
                : bio // ignore: cast_nullable_to_non_nullable
                      as String,
            signature: null == signature
                ? _value.signature
                : signature // ignore: cast_nullable_to_non_nullable
                      as String,
            websiteName: null == websiteName
                ? _value.websiteName
                : websiteName // ignore: cast_nullable_to_non_nullable
                      as String,
            website: null == website
                ? _value.website
                : website // ignore: cast_nullable_to_non_nullable
                      as String,
            prestige: null == prestige
                ? _value.prestige
                : prestige // ignore: cast_nullable_to_non_nullable
                      as int,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
            externalInformation: null == externalInformation
                ? _value.externalInformation
                : externalInformation // ignore: cast_nullable_to_non_nullable
                      as Map<String, ExternalLinkPayload>,
            wornBadgeCode: null == wornBadgeCode
                ? _value.wornBadgeCode
                : wornBadgeCode // ignore: cast_nullable_to_non_nullable
                      as String,
            badges: null == badges
                ? _value.badges
                : badges // ignore: cast_nullable_to_non_nullable
                      as List<UserBadgePayload>,
            wearableBadges: null == wearableBadges
                ? _value.wearableBadges
                : wearableBadges // ignore: cast_nullable_to_non_nullable
                      as List<UserBadgePayload>,
            wornBadge: freezed == wornBadge
                ? _value.wornBadge
                : wornBadge // ignore: cast_nullable_to_non_nullable
                      as UserBadgePayload?,
          )
          as $Val,
    );
  }

  /// Create a copy of SettingsUserPayload
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
abstract class _$$SettingsUserPayloadImplCopyWith<$Res>
    implements $SettingsUserPayloadCopyWith<$Res> {
  factory _$$SettingsUserPayloadImplCopyWith(
    _$SettingsUserPayloadImpl value,
    $Res Function(_$SettingsUserPayloadImpl) then,
  ) = __$$SettingsUserPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String username,
    String email,
    String nickname,
    String locale,
    String avatarUrl,
    String profileCoverUrl,
    String bio,
    String signature,
    String websiteName,
    String website,
    int prestige,
    String createdAt,
    Map<String, ExternalLinkPayload> externalInformation,
    String wornBadgeCode,
    List<UserBadgePayload> badges,
    List<UserBadgePayload> wearableBadges,
    UserBadgePayload? wornBadge,
  });

  @override
  $UserBadgePayloadCopyWith<$Res>? get wornBadge;
}

/// @nodoc
class __$$SettingsUserPayloadImplCopyWithImpl<$Res>
    extends _$SettingsUserPayloadCopyWithImpl<$Res, _$SettingsUserPayloadImpl>
    implements _$$SettingsUserPayloadImplCopyWith<$Res> {
  __$$SettingsUserPayloadImplCopyWithImpl(
    _$SettingsUserPayloadImpl _value,
    $Res Function(_$SettingsUserPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SettingsUserPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? email = null,
    Object? nickname = null,
    Object? locale = null,
    Object? avatarUrl = null,
    Object? profileCoverUrl = null,
    Object? bio = null,
    Object? signature = null,
    Object? websiteName = null,
    Object? website = null,
    Object? prestige = null,
    Object? createdAt = null,
    Object? externalInformation = null,
    Object? wornBadgeCode = null,
    Object? badges = null,
    Object? wearableBadges = null,
    Object? wornBadge = freezed,
  }) {
    return _then(
      _$SettingsUserPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String,
        email: null == email
            ? _value.email
            : email // ignore: cast_nullable_to_non_nullable
                  as String,
        nickname: null == nickname
            ? _value.nickname
            : nickname // ignore: cast_nullable_to_non_nullable
                  as String,
        locale: null == locale
            ? _value.locale
            : locale // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        profileCoverUrl: null == profileCoverUrl
            ? _value.profileCoverUrl
            : profileCoverUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        bio: null == bio
            ? _value.bio
            : bio // ignore: cast_nullable_to_non_nullable
                  as String,
        signature: null == signature
            ? _value.signature
            : signature // ignore: cast_nullable_to_non_nullable
                  as String,
        websiteName: null == websiteName
            ? _value.websiteName
            : websiteName // ignore: cast_nullable_to_non_nullable
                  as String,
        website: null == website
            ? _value.website
            : website // ignore: cast_nullable_to_non_nullable
                  as String,
        prestige: null == prestige
            ? _value.prestige
            : prestige // ignore: cast_nullable_to_non_nullable
                  as int,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String,
        externalInformation: null == externalInformation
            ? _value._externalInformation
            : externalInformation // ignore: cast_nullable_to_non_nullable
                  as Map<String, ExternalLinkPayload>,
        wornBadgeCode: null == wornBadgeCode
            ? _value.wornBadgeCode
            : wornBadgeCode // ignore: cast_nullable_to_non_nullable
                  as String,
        badges: null == badges
            ? _value._badges
            : badges // ignore: cast_nullable_to_non_nullable
                  as List<UserBadgePayload>,
        wearableBadges: null == wearableBadges
            ? _value._wearableBadges
            : wearableBadges // ignore: cast_nullable_to_non_nullable
                  as List<UserBadgePayload>,
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
class _$SettingsUserPayloadImpl implements _SettingsUserPayload {
  const _$SettingsUserPayloadImpl({
    required this.id,
    required this.username,
    required this.email,
    required this.nickname,
    required this.locale,
    required this.avatarUrl,
    required this.profileCoverUrl,
    required this.bio,
    required this.signature,
    required this.websiteName,
    required this.website,
    required this.prestige,
    required this.createdAt,
    required final Map<String, ExternalLinkPayload> externalInformation,
    required this.wornBadgeCode,
    required final List<UserBadgePayload> badges,
    required final List<UserBadgePayload> wearableBadges,
    this.wornBadge,
  }) : _externalInformation = externalInformation,
       _badges = badges,
       _wearableBadges = wearableBadges;

  factory _$SettingsUserPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SettingsUserPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String email;
  @override
  final String nickname;
  @override
  final String locale;
  @override
  final String avatarUrl;
  @override
  final String profileCoverUrl;
  @override
  final String bio;
  @override
  final String signature;
  @override
  final String websiteName;
  @override
  final String website;
  @override
  final int prestige;
  @override
  final String createdAt;
  final Map<String, ExternalLinkPayload> _externalInformation;
  @override
  Map<String, ExternalLinkPayload> get externalInformation {
    if (_externalInformation is EqualUnmodifiableMapView)
      return _externalInformation;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(_externalInformation);
  }

  @override
  final String wornBadgeCode;
  final List<UserBadgePayload> _badges;
  @override
  List<UserBadgePayload> get badges {
    if (_badges is EqualUnmodifiableListView) return _badges;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_badges);
  }

  final List<UserBadgePayload> _wearableBadges;
  @override
  List<UserBadgePayload> get wearableBadges {
    if (_wearableBadges is EqualUnmodifiableListView) return _wearableBadges;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_wearableBadges);
  }

  @override
  final UserBadgePayload? wornBadge;

  @override
  String toString() {
    return 'SettingsUserPayload(id: $id, username: $username, email: $email, nickname: $nickname, locale: $locale, avatarUrl: $avatarUrl, profileCoverUrl: $profileCoverUrl, bio: $bio, signature: $signature, websiteName: $websiteName, website: $website, prestige: $prestige, createdAt: $createdAt, externalInformation: $externalInformation, wornBadgeCode: $wornBadgeCode, badges: $badges, wearableBadges: $wearableBadges, wornBadge: $wornBadge)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SettingsUserPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.email, email) || other.email == email) &&
            (identical(other.nickname, nickname) ||
                other.nickname == nickname) &&
            (identical(other.locale, locale) || other.locale == locale) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.profileCoverUrl, profileCoverUrl) ||
                other.profileCoverUrl == profileCoverUrl) &&
            (identical(other.bio, bio) || other.bio == bio) &&
            (identical(other.signature, signature) ||
                other.signature == signature) &&
            (identical(other.websiteName, websiteName) ||
                other.websiteName == websiteName) &&
            (identical(other.website, website) || other.website == website) &&
            (identical(other.prestige, prestige) ||
                other.prestige == prestige) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            const DeepCollectionEquality().equals(
              other._externalInformation,
              _externalInformation,
            ) &&
            (identical(other.wornBadgeCode, wornBadgeCode) ||
                other.wornBadgeCode == wornBadgeCode) &&
            const DeepCollectionEquality().equals(other._badges, _badges) &&
            const DeepCollectionEquality().equals(
              other._wearableBadges,
              _wearableBadges,
            ) &&
            (identical(other.wornBadge, wornBadge) ||
                other.wornBadge == wornBadge));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    username,
    email,
    nickname,
    locale,
    avatarUrl,
    profileCoverUrl,
    bio,
    signature,
    websiteName,
    website,
    prestige,
    createdAt,
    const DeepCollectionEquality().hash(_externalInformation),
    wornBadgeCode,
    const DeepCollectionEquality().hash(_badges),
    const DeepCollectionEquality().hash(_wearableBadges),
    wornBadge,
  );

  /// Create a copy of SettingsUserPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SettingsUserPayloadImplCopyWith<_$SettingsUserPayloadImpl> get copyWith =>
      __$$SettingsUserPayloadImplCopyWithImpl<_$SettingsUserPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SettingsUserPayloadImplToJson(this);
  }
}

abstract class _SettingsUserPayload implements SettingsUserPayload {
  const factory _SettingsUserPayload({
    required final int id,
    required final String username,
    required final String email,
    required final String nickname,
    required final String locale,
    required final String avatarUrl,
    required final String profileCoverUrl,
    required final String bio,
    required final String signature,
    required final String websiteName,
    required final String website,
    required final int prestige,
    required final String createdAt,
    required final Map<String, ExternalLinkPayload> externalInformation,
    required final String wornBadgeCode,
    required final List<UserBadgePayload> badges,
    required final List<UserBadgePayload> wearableBadges,
    final UserBadgePayload? wornBadge,
  }) = _$SettingsUserPayloadImpl;

  factory _SettingsUserPayload.fromJson(Map<String, dynamic> json) =
      _$SettingsUserPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String get email;
  @override
  String get nickname;
  @override
  String get locale;
  @override
  String get avatarUrl;
  @override
  String get profileCoverUrl;
  @override
  String get bio;
  @override
  String get signature;
  @override
  String get websiteName;
  @override
  String get website;
  @override
  int get prestige;
  @override
  String get createdAt;
  @override
  Map<String, ExternalLinkPayload> get externalInformation;
  @override
  String get wornBadgeCode;
  @override
  List<UserBadgePayload> get badges;
  @override
  List<UserBadgePayload> get wearableBadges;
  @override
  UserBadgePayload? get wornBadge;

  /// Create a copy of SettingsUserPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SettingsUserPayloadImplCopyWith<_$SettingsUserPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SettingsStatsPayload _$SettingsStatsPayloadFromJson(Map<String, dynamic> json) {
  return _SettingsStatsPayload.fromJson(json);
}

/// @nodoc
mixin _$SettingsStatsPayload {
  int get topicCount => throw _privateConstructorUsedError;
  int get replyCount => throw _privateConstructorUsedError;
  int get followerCount => throw _privateConstructorUsedError;
  int get followingCount => throw _privateConstructorUsedError;
  int get likeReceivedCount => throw _privateConstructorUsedError;
  int get likeGivenCount => throw _privateConstructorUsedError;
  int get collectionCount => throw _privateConstructorUsedError;
  String get createdAt => throw _privateConstructorUsedError;

  /// Serializes this SettingsStatsPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SettingsStatsPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SettingsStatsPayloadCopyWith<SettingsStatsPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SettingsStatsPayloadCopyWith<$Res> {
  factory $SettingsStatsPayloadCopyWith(
    SettingsStatsPayload value,
    $Res Function(SettingsStatsPayload) then,
  ) = _$SettingsStatsPayloadCopyWithImpl<$Res, SettingsStatsPayload>;
  @useResult
  $Res call({
    int topicCount,
    int replyCount,
    int followerCount,
    int followingCount,
    int likeReceivedCount,
    int likeGivenCount,
    int collectionCount,
    String createdAt,
  });
}

/// @nodoc
class _$SettingsStatsPayloadCopyWithImpl<
  $Res,
  $Val extends SettingsStatsPayload
>
    implements $SettingsStatsPayloadCopyWith<$Res> {
  _$SettingsStatsPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SettingsStatsPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topicCount = null,
    Object? replyCount = null,
    Object? followerCount = null,
    Object? followingCount = null,
    Object? likeReceivedCount = null,
    Object? likeGivenCount = null,
    Object? collectionCount = null,
    Object? createdAt = null,
  }) {
    return _then(
      _value.copyWith(
            topicCount: null == topicCount
                ? _value.topicCount
                : topicCount // ignore: cast_nullable_to_non_nullable
                      as int,
            replyCount: null == replyCount
                ? _value.replyCount
                : replyCount // ignore: cast_nullable_to_non_nullable
                      as int,
            followerCount: null == followerCount
                ? _value.followerCount
                : followerCount // ignore: cast_nullable_to_non_nullable
                      as int,
            followingCount: null == followingCount
                ? _value.followingCount
                : followingCount // ignore: cast_nullable_to_non_nullable
                      as int,
            likeReceivedCount: null == likeReceivedCount
                ? _value.likeReceivedCount
                : likeReceivedCount // ignore: cast_nullable_to_non_nullable
                      as int,
            likeGivenCount: null == likeGivenCount
                ? _value.likeGivenCount
                : likeGivenCount // ignore: cast_nullable_to_non_nullable
                      as int,
            collectionCount: null == collectionCount
                ? _value.collectionCount
                : collectionCount // ignore: cast_nullable_to_non_nullable
                      as int,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SettingsStatsPayloadImplCopyWith<$Res>
    implements $SettingsStatsPayloadCopyWith<$Res> {
  factory _$$SettingsStatsPayloadImplCopyWith(
    _$SettingsStatsPayloadImpl value,
    $Res Function(_$SettingsStatsPayloadImpl) then,
  ) = __$$SettingsStatsPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int topicCount,
    int replyCount,
    int followerCount,
    int followingCount,
    int likeReceivedCount,
    int likeGivenCount,
    int collectionCount,
    String createdAt,
  });
}

/// @nodoc
class __$$SettingsStatsPayloadImplCopyWithImpl<$Res>
    extends _$SettingsStatsPayloadCopyWithImpl<$Res, _$SettingsStatsPayloadImpl>
    implements _$$SettingsStatsPayloadImplCopyWith<$Res> {
  __$$SettingsStatsPayloadImplCopyWithImpl(
    _$SettingsStatsPayloadImpl _value,
    $Res Function(_$SettingsStatsPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SettingsStatsPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topicCount = null,
    Object? replyCount = null,
    Object? followerCount = null,
    Object? followingCount = null,
    Object? likeReceivedCount = null,
    Object? likeGivenCount = null,
    Object? collectionCount = null,
    Object? createdAt = null,
  }) {
    return _then(
      _$SettingsStatsPayloadImpl(
        topicCount: null == topicCount
            ? _value.topicCount
            : topicCount // ignore: cast_nullable_to_non_nullable
                  as int,
        replyCount: null == replyCount
            ? _value.replyCount
            : replyCount // ignore: cast_nullable_to_non_nullable
                  as int,
        followerCount: null == followerCount
            ? _value.followerCount
            : followerCount // ignore: cast_nullable_to_non_nullable
                  as int,
        followingCount: null == followingCount
            ? _value.followingCount
            : followingCount // ignore: cast_nullable_to_non_nullable
                  as int,
        likeReceivedCount: null == likeReceivedCount
            ? _value.likeReceivedCount
            : likeReceivedCount // ignore: cast_nullable_to_non_nullable
                  as int,
        likeGivenCount: null == likeGivenCount
            ? _value.likeGivenCount
            : likeGivenCount // ignore: cast_nullable_to_non_nullable
                  as int,
        collectionCount: null == collectionCount
            ? _value.collectionCount
            : collectionCount // ignore: cast_nullable_to_non_nullable
                  as int,
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
class _$SettingsStatsPayloadImpl implements _SettingsStatsPayload {
  const _$SettingsStatsPayloadImpl({
    required this.topicCount,
    required this.replyCount,
    required this.followerCount,
    required this.followingCount,
    required this.likeReceivedCount,
    required this.likeGivenCount,
    required this.collectionCount,
    required this.createdAt,
  });

  factory _$SettingsStatsPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SettingsStatsPayloadImplFromJson(json);

  @override
  final int topicCount;
  @override
  final int replyCount;
  @override
  final int followerCount;
  @override
  final int followingCount;
  @override
  final int likeReceivedCount;
  @override
  final int likeGivenCount;
  @override
  final int collectionCount;
  @override
  final String createdAt;

  @override
  String toString() {
    return 'SettingsStatsPayload(topicCount: $topicCount, replyCount: $replyCount, followerCount: $followerCount, followingCount: $followingCount, likeReceivedCount: $likeReceivedCount, likeGivenCount: $likeGivenCount, collectionCount: $collectionCount, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SettingsStatsPayloadImpl &&
            (identical(other.topicCount, topicCount) ||
                other.topicCount == topicCount) &&
            (identical(other.replyCount, replyCount) ||
                other.replyCount == replyCount) &&
            (identical(other.followerCount, followerCount) ||
                other.followerCount == followerCount) &&
            (identical(other.followingCount, followingCount) ||
                other.followingCount == followingCount) &&
            (identical(other.likeReceivedCount, likeReceivedCount) ||
                other.likeReceivedCount == likeReceivedCount) &&
            (identical(other.likeGivenCount, likeGivenCount) ||
                other.likeGivenCount == likeGivenCount) &&
            (identical(other.collectionCount, collectionCount) ||
                other.collectionCount == collectionCount) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    topicCount,
    replyCount,
    followerCount,
    followingCount,
    likeReceivedCount,
    likeGivenCount,
    collectionCount,
    createdAt,
  );

  /// Create a copy of SettingsStatsPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SettingsStatsPayloadImplCopyWith<_$SettingsStatsPayloadImpl>
  get copyWith =>
      __$$SettingsStatsPayloadImplCopyWithImpl<_$SettingsStatsPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SettingsStatsPayloadImplToJson(this);
  }
}

abstract class _SettingsStatsPayload implements SettingsStatsPayload {
  const factory _SettingsStatsPayload({
    required final int topicCount,
    required final int replyCount,
    required final int followerCount,
    required final int followingCount,
    required final int likeReceivedCount,
    required final int likeGivenCount,
    required final int collectionCount,
    required final String createdAt,
  }) = _$SettingsStatsPayloadImpl;

  factory _SettingsStatsPayload.fromJson(Map<String, dynamic> json) =
      _$SettingsStatsPayloadImpl.fromJson;

  @override
  int get topicCount;
  @override
  int get replyCount;
  @override
  int get followerCount;
  @override
  int get followingCount;
  @override
  int get likeReceivedCount;
  @override
  int get likeGivenCount;
  @override
  int get collectionCount;
  @override
  String get createdAt;

  /// Create a copy of SettingsStatsPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SettingsStatsPayloadImplCopyWith<_$SettingsStatsPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

SettingsPageProps _$SettingsPagePropsFromJson(Map<String, dynamic> json) {
  return _SettingsPageProps.fromJson(json);
}

/// @nodoc
mixin _$SettingsPageProps {
  SettingsUserPayload get user => throw _privateConstructorUsedError;
  SettingsStatsPayload get stats => throw _privateConstructorUsedError;
  List<TabItemPayload> get tabs => throw _privateConstructorUsedError;

  /// Serializes this SettingsPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SettingsPagePropsCopyWith<SettingsPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SettingsPagePropsCopyWith<$Res> {
  factory $SettingsPagePropsCopyWith(
    SettingsPageProps value,
    $Res Function(SettingsPageProps) then,
  ) = _$SettingsPagePropsCopyWithImpl<$Res, SettingsPageProps>;
  @useResult
  $Res call({
    SettingsUserPayload user,
    SettingsStatsPayload stats,
    List<TabItemPayload> tabs,
  });

  $SettingsUserPayloadCopyWith<$Res> get user;
  $SettingsStatsPayloadCopyWith<$Res> get stats;
}

/// @nodoc
class _$SettingsPagePropsCopyWithImpl<$Res, $Val extends SettingsPageProps>
    implements $SettingsPagePropsCopyWith<$Res> {
  _$SettingsPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? user = null, Object? stats = null, Object? tabs = null}) {
    return _then(
      _value.copyWith(
            user: null == user
                ? _value.user
                : user // ignore: cast_nullable_to_non_nullable
                      as SettingsUserPayload,
            stats: null == stats
                ? _value.stats
                : stats // ignore: cast_nullable_to_non_nullable
                      as SettingsStatsPayload,
            tabs: null == tabs
                ? _value.tabs
                : tabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
          )
          as $Val,
    );
  }

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SettingsUserPayloadCopyWith<$Res> get user {
    return $SettingsUserPayloadCopyWith<$Res>(_value.user, (value) {
      return _then(_value.copyWith(user: value) as $Val);
    });
  }

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SettingsStatsPayloadCopyWith<$Res> get stats {
    return $SettingsStatsPayloadCopyWith<$Res>(_value.stats, (value) {
      return _then(_value.copyWith(stats: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$SettingsPagePropsImplCopyWith<$Res>
    implements $SettingsPagePropsCopyWith<$Res> {
  factory _$$SettingsPagePropsImplCopyWith(
    _$SettingsPagePropsImpl value,
    $Res Function(_$SettingsPagePropsImpl) then,
  ) = __$$SettingsPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    SettingsUserPayload user,
    SettingsStatsPayload stats,
    List<TabItemPayload> tabs,
  });

  @override
  $SettingsUserPayloadCopyWith<$Res> get user;
  @override
  $SettingsStatsPayloadCopyWith<$Res> get stats;
}

/// @nodoc
class __$$SettingsPagePropsImplCopyWithImpl<$Res>
    extends _$SettingsPagePropsCopyWithImpl<$Res, _$SettingsPagePropsImpl>
    implements _$$SettingsPagePropsImplCopyWith<$Res> {
  __$$SettingsPagePropsImplCopyWithImpl(
    _$SettingsPagePropsImpl _value,
    $Res Function(_$SettingsPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? user = null, Object? stats = null, Object? tabs = null}) {
    return _then(
      _$SettingsPagePropsImpl(
        user: null == user
            ? _value.user
            : user // ignore: cast_nullable_to_non_nullable
                  as SettingsUserPayload,
        stats: null == stats
            ? _value.stats
            : stats // ignore: cast_nullable_to_non_nullable
                  as SettingsStatsPayload,
        tabs: null == tabs
            ? _value._tabs
            : tabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SettingsPagePropsImpl implements _SettingsPageProps {
  const _$SettingsPagePropsImpl({
    required this.user,
    required this.stats,
    required final List<TabItemPayload> tabs,
  }) : _tabs = tabs;

  factory _$SettingsPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$SettingsPagePropsImplFromJson(json);

  @override
  final SettingsUserPayload user;
  @override
  final SettingsStatsPayload stats;
  final List<TabItemPayload> _tabs;
  @override
  List<TabItemPayload> get tabs {
    if (_tabs is EqualUnmodifiableListView) return _tabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tabs);
  }

  @override
  String toString() {
    return 'SettingsPageProps(user: $user, stats: $stats, tabs: $tabs)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SettingsPagePropsImpl &&
            (identical(other.user, user) || other.user == user) &&
            (identical(other.stats, stats) || other.stats == stats) &&
            const DeepCollectionEquality().equals(other._tabs, _tabs));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    user,
    stats,
    const DeepCollectionEquality().hash(_tabs),
  );

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SettingsPagePropsImplCopyWith<_$SettingsPagePropsImpl> get copyWith =>
      __$$SettingsPagePropsImplCopyWithImpl<_$SettingsPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SettingsPagePropsImplToJson(this);
  }
}

abstract class _SettingsPageProps implements SettingsPageProps {
  const factory _SettingsPageProps({
    required final SettingsUserPayload user,
    required final SettingsStatsPayload stats,
    required final List<TabItemPayload> tabs,
  }) = _$SettingsPagePropsImpl;

  factory _SettingsPageProps.fromJson(Map<String, dynamic> json) =
      _$SettingsPagePropsImpl.fromJson;

  @override
  SettingsUserPayload get user;
  @override
  SettingsStatsPayload get stats;
  @override
  List<TabItemPayload> get tabs;

  /// Create a copy of SettingsPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SettingsPagePropsImplCopyWith<_$SettingsPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
