// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'auth.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

LoginPageProps _$LoginPagePropsFromJson(Map<String, dynamic> json) {
  return _LoginPageProps.fromJson(json);
}

/// @nodoc
mixin _$LoginPageProps {
  String get initialMode => throw _privateConstructorUsedError;
  String get redirectUrl => throw _privateConstructorUsedError;
  String get githubUrl => throw _privateConstructorUsedError;
  bool get googleReady => throw _privateConstructorUsedError;

  /// Serializes this LoginPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LoginPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LoginPagePropsCopyWith<LoginPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LoginPagePropsCopyWith<$Res> {
  factory $LoginPagePropsCopyWith(
    LoginPageProps value,
    $Res Function(LoginPageProps) then,
  ) = _$LoginPagePropsCopyWithImpl<$Res, LoginPageProps>;
  @useResult
  $Res call({
    String initialMode,
    String redirectUrl,
    String githubUrl,
    bool googleReady,
  });
}

/// @nodoc
class _$LoginPagePropsCopyWithImpl<$Res, $Val extends LoginPageProps>
    implements $LoginPagePropsCopyWith<$Res> {
  _$LoginPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LoginPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? initialMode = null,
    Object? redirectUrl = null,
    Object? githubUrl = null,
    Object? googleReady = null,
  }) {
    return _then(
      _value.copyWith(
            initialMode: null == initialMode
                ? _value.initialMode
                : initialMode // ignore: cast_nullable_to_non_nullable
                      as String,
            redirectUrl: null == redirectUrl
                ? _value.redirectUrl
                : redirectUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            githubUrl: null == githubUrl
                ? _value.githubUrl
                : githubUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            googleReady: null == googleReady
                ? _value.googleReady
                : googleReady // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$LoginPagePropsImplCopyWith<$Res>
    implements $LoginPagePropsCopyWith<$Res> {
  factory _$$LoginPagePropsImplCopyWith(
    _$LoginPagePropsImpl value,
    $Res Function(_$LoginPagePropsImpl) then,
  ) = __$$LoginPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String initialMode,
    String redirectUrl,
    String githubUrl,
    bool googleReady,
  });
}

/// @nodoc
class __$$LoginPagePropsImplCopyWithImpl<$Res>
    extends _$LoginPagePropsCopyWithImpl<$Res, _$LoginPagePropsImpl>
    implements _$$LoginPagePropsImplCopyWith<$Res> {
  __$$LoginPagePropsImplCopyWithImpl(
    _$LoginPagePropsImpl _value,
    $Res Function(_$LoginPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LoginPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? initialMode = null,
    Object? redirectUrl = null,
    Object? githubUrl = null,
    Object? googleReady = null,
  }) {
    return _then(
      _$LoginPagePropsImpl(
        initialMode: null == initialMode
            ? _value.initialMode
            : initialMode // ignore: cast_nullable_to_non_nullable
                  as String,
        redirectUrl: null == redirectUrl
            ? _value.redirectUrl
            : redirectUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        githubUrl: null == githubUrl
            ? _value.githubUrl
            : githubUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        googleReady: null == googleReady
            ? _value.googleReady
            : googleReady // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LoginPagePropsImpl implements _LoginPageProps {
  const _$LoginPagePropsImpl({
    required this.initialMode,
    required this.redirectUrl,
    required this.githubUrl,
    required this.googleReady,
  });

  factory _$LoginPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$LoginPagePropsImplFromJson(json);

  @override
  final String initialMode;
  @override
  final String redirectUrl;
  @override
  final String githubUrl;
  @override
  final bool googleReady;

  @override
  String toString() {
    return 'LoginPageProps(initialMode: $initialMode, redirectUrl: $redirectUrl, githubUrl: $githubUrl, googleReady: $googleReady)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LoginPagePropsImpl &&
            (identical(other.initialMode, initialMode) ||
                other.initialMode == initialMode) &&
            (identical(other.redirectUrl, redirectUrl) ||
                other.redirectUrl == redirectUrl) &&
            (identical(other.githubUrl, githubUrl) ||
                other.githubUrl == githubUrl) &&
            (identical(other.googleReady, googleReady) ||
                other.googleReady == googleReady));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    initialMode,
    redirectUrl,
    githubUrl,
    googleReady,
  );

  /// Create a copy of LoginPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LoginPagePropsImplCopyWith<_$LoginPagePropsImpl> get copyWith =>
      __$$LoginPagePropsImplCopyWithImpl<_$LoginPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$LoginPagePropsImplToJson(this);
  }
}

abstract class _LoginPageProps implements LoginPageProps {
  const factory _LoginPageProps({
    required final String initialMode,
    required final String redirectUrl,
    required final String githubUrl,
    required final bool googleReady,
  }) = _$LoginPagePropsImpl;

  factory _LoginPageProps.fromJson(Map<String, dynamic> json) =
      _$LoginPagePropsImpl.fromJson;

  @override
  String get initialMode;
  @override
  String get redirectUrl;
  @override
  String get githubUrl;
  @override
  bool get googleReady;

  /// Create a copy of LoginPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LoginPagePropsImplCopyWith<_$LoginPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ResetPasswordPageProps _$ResetPasswordPagePropsFromJson(
  Map<String, dynamic> json,
) {
  return _ResetPasswordPageProps.fromJson(json);
}

/// @nodoc
mixin _$ResetPasswordPageProps {
  String get token => throw _privateConstructorUsedError;

  /// Serializes this ResetPasswordPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ResetPasswordPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ResetPasswordPagePropsCopyWith<ResetPasswordPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ResetPasswordPagePropsCopyWith<$Res> {
  factory $ResetPasswordPagePropsCopyWith(
    ResetPasswordPageProps value,
    $Res Function(ResetPasswordPageProps) then,
  ) = _$ResetPasswordPagePropsCopyWithImpl<$Res, ResetPasswordPageProps>;
  @useResult
  $Res call({String token});
}

/// @nodoc
class _$ResetPasswordPagePropsCopyWithImpl<
  $Res,
  $Val extends ResetPasswordPageProps
>
    implements $ResetPasswordPagePropsCopyWith<$Res> {
  _$ResetPasswordPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ResetPasswordPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? token = null}) {
    return _then(
      _value.copyWith(
            token: null == token
                ? _value.token
                : token // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ResetPasswordPagePropsImplCopyWith<$Res>
    implements $ResetPasswordPagePropsCopyWith<$Res> {
  factory _$$ResetPasswordPagePropsImplCopyWith(
    _$ResetPasswordPagePropsImpl value,
    $Res Function(_$ResetPasswordPagePropsImpl) then,
  ) = __$$ResetPasswordPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String token});
}

/// @nodoc
class __$$ResetPasswordPagePropsImplCopyWithImpl<$Res>
    extends
        _$ResetPasswordPagePropsCopyWithImpl<$Res, _$ResetPasswordPagePropsImpl>
    implements _$$ResetPasswordPagePropsImplCopyWith<$Res> {
  __$$ResetPasswordPagePropsImplCopyWithImpl(
    _$ResetPasswordPagePropsImpl _value,
    $Res Function(_$ResetPasswordPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ResetPasswordPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? token = null}) {
    return _then(
      _$ResetPasswordPagePropsImpl(
        token: null == token
            ? _value.token
            : token // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ResetPasswordPagePropsImpl implements _ResetPasswordPageProps {
  const _$ResetPasswordPagePropsImpl({required this.token});

  factory _$ResetPasswordPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$ResetPasswordPagePropsImplFromJson(json);

  @override
  final String token;

  @override
  String toString() {
    return 'ResetPasswordPageProps(token: $token)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ResetPasswordPagePropsImpl &&
            (identical(other.token, token) || other.token == token));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, token);

  /// Create a copy of ResetPasswordPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ResetPasswordPagePropsImplCopyWith<_$ResetPasswordPagePropsImpl>
  get copyWith =>
      __$$ResetPasswordPagePropsImplCopyWithImpl<_$ResetPasswordPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ResetPasswordPagePropsImplToJson(this);
  }
}

abstract class _ResetPasswordPageProps implements ResetPasswordPageProps {
  const factory _ResetPasswordPageProps({required final String token}) =
      _$ResetPasswordPagePropsImpl;

  factory _ResetPasswordPageProps.fromJson(Map<String, dynamic> json) =
      _$ResetPasswordPagePropsImpl.fromJson;

  @override
  String get token;

  /// Create a copy of ResetPasswordPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ResetPasswordPagePropsImplCopyWith<_$ResetPasswordPagePropsImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CaptchaPayload _$CaptchaPayloadFromJson(Map<String, dynamic> json) {
  return _CaptchaPayload.fromJson(json);
}

/// @nodoc
mixin _$CaptchaPayload {
  String get captchaId => throw _privateConstructorUsedError;
  String get captchaImg => throw _privateConstructorUsedError;

  /// Serializes this CaptchaPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CaptchaPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CaptchaPayloadCopyWith<CaptchaPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CaptchaPayloadCopyWith<$Res> {
  factory $CaptchaPayloadCopyWith(
    CaptchaPayload value,
    $Res Function(CaptchaPayload) then,
  ) = _$CaptchaPayloadCopyWithImpl<$Res, CaptchaPayload>;
  @useResult
  $Res call({String captchaId, String captchaImg});
}

/// @nodoc
class _$CaptchaPayloadCopyWithImpl<$Res, $Val extends CaptchaPayload>
    implements $CaptchaPayloadCopyWith<$Res> {
  _$CaptchaPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CaptchaPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? captchaId = null, Object? captchaImg = null}) {
    return _then(
      _value.copyWith(
            captchaId: null == captchaId
                ? _value.captchaId
                : captchaId // ignore: cast_nullable_to_non_nullable
                      as String,
            captchaImg: null == captchaImg
                ? _value.captchaImg
                : captchaImg // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CaptchaPayloadImplCopyWith<$Res>
    implements $CaptchaPayloadCopyWith<$Res> {
  factory _$$CaptchaPayloadImplCopyWith(
    _$CaptchaPayloadImpl value,
    $Res Function(_$CaptchaPayloadImpl) then,
  ) = __$$CaptchaPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String captchaId, String captchaImg});
}

/// @nodoc
class __$$CaptchaPayloadImplCopyWithImpl<$Res>
    extends _$CaptchaPayloadCopyWithImpl<$Res, _$CaptchaPayloadImpl>
    implements _$$CaptchaPayloadImplCopyWith<$Res> {
  __$$CaptchaPayloadImplCopyWithImpl(
    _$CaptchaPayloadImpl _value,
    $Res Function(_$CaptchaPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CaptchaPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? captchaId = null, Object? captchaImg = null}) {
    return _then(
      _$CaptchaPayloadImpl(
        captchaId: null == captchaId
            ? _value.captchaId
            : captchaId // ignore: cast_nullable_to_non_nullable
                  as String,
        captchaImg: null == captchaImg
            ? _value.captchaImg
            : captchaImg // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CaptchaPayloadImpl implements _CaptchaPayload {
  const _$CaptchaPayloadImpl({
    required this.captchaId,
    required this.captchaImg,
  });

  factory _$CaptchaPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CaptchaPayloadImplFromJson(json);

  @override
  final String captchaId;
  @override
  final String captchaImg;

  @override
  String toString() {
    return 'CaptchaPayload(captchaId: $captchaId, captchaImg: $captchaImg)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CaptchaPayloadImpl &&
            (identical(other.captchaId, captchaId) ||
                other.captchaId == captchaId) &&
            (identical(other.captchaImg, captchaImg) ||
                other.captchaImg == captchaImg));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, captchaId, captchaImg);

  /// Create a copy of CaptchaPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CaptchaPayloadImplCopyWith<_$CaptchaPayloadImpl> get copyWith =>
      __$$CaptchaPayloadImplCopyWithImpl<_$CaptchaPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CaptchaPayloadImplToJson(this);
  }
}

abstract class _CaptchaPayload implements CaptchaPayload {
  const factory _CaptchaPayload({
    required final String captchaId,
    required final String captchaImg,
  }) = _$CaptchaPayloadImpl;

  factory _CaptchaPayload.fromJson(Map<String, dynamic> json) =
      _$CaptchaPayloadImpl.fromJson;

  @override
  String get captchaId;
  @override
  String get captchaImg;

  /// Create a copy of CaptchaPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CaptchaPayloadImplCopyWith<_$CaptchaPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

LoginPublicKeyPayload _$LoginPublicKeyPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _LoginPublicKeyPayload.fromJson(json);
}

/// @nodoc
mixin _$LoginPublicKeyPayload {
  String get publicKey => throw _privateConstructorUsedError;
  int get serverTs => throw _privateConstructorUsedError;
  String get algorithm => throw _privateConstructorUsedError;

  /// Serializes this LoginPublicKeyPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LoginPublicKeyPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LoginPublicKeyPayloadCopyWith<LoginPublicKeyPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LoginPublicKeyPayloadCopyWith<$Res> {
  factory $LoginPublicKeyPayloadCopyWith(
    LoginPublicKeyPayload value,
    $Res Function(LoginPublicKeyPayload) then,
  ) = _$LoginPublicKeyPayloadCopyWithImpl<$Res, LoginPublicKeyPayload>;
  @useResult
  $Res call({String publicKey, int serverTs, String algorithm});
}

/// @nodoc
class _$LoginPublicKeyPayloadCopyWithImpl<
  $Res,
  $Val extends LoginPublicKeyPayload
>
    implements $LoginPublicKeyPayloadCopyWith<$Res> {
  _$LoginPublicKeyPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LoginPublicKeyPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? publicKey = null,
    Object? serverTs = null,
    Object? algorithm = null,
  }) {
    return _then(
      _value.copyWith(
            publicKey: null == publicKey
                ? _value.publicKey
                : publicKey // ignore: cast_nullable_to_non_nullable
                      as String,
            serverTs: null == serverTs
                ? _value.serverTs
                : serverTs // ignore: cast_nullable_to_non_nullable
                      as int,
            algorithm: null == algorithm
                ? _value.algorithm
                : algorithm // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$LoginPublicKeyPayloadImplCopyWith<$Res>
    implements $LoginPublicKeyPayloadCopyWith<$Res> {
  factory _$$LoginPublicKeyPayloadImplCopyWith(
    _$LoginPublicKeyPayloadImpl value,
    $Res Function(_$LoginPublicKeyPayloadImpl) then,
  ) = __$$LoginPublicKeyPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String publicKey, int serverTs, String algorithm});
}

/// @nodoc
class __$$LoginPublicKeyPayloadImplCopyWithImpl<$Res>
    extends
        _$LoginPublicKeyPayloadCopyWithImpl<$Res, _$LoginPublicKeyPayloadImpl>
    implements _$$LoginPublicKeyPayloadImplCopyWith<$Res> {
  __$$LoginPublicKeyPayloadImplCopyWithImpl(
    _$LoginPublicKeyPayloadImpl _value,
    $Res Function(_$LoginPublicKeyPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LoginPublicKeyPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? publicKey = null,
    Object? serverTs = null,
    Object? algorithm = null,
  }) {
    return _then(
      _$LoginPublicKeyPayloadImpl(
        publicKey: null == publicKey
            ? _value.publicKey
            : publicKey // ignore: cast_nullable_to_non_nullable
                  as String,
        serverTs: null == serverTs
            ? _value.serverTs
            : serverTs // ignore: cast_nullable_to_non_nullable
                  as int,
        algorithm: null == algorithm
            ? _value.algorithm
            : algorithm // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LoginPublicKeyPayloadImpl implements _LoginPublicKeyPayload {
  const _$LoginPublicKeyPayloadImpl({
    required this.publicKey,
    required this.serverTs,
    required this.algorithm,
  });

  factory _$LoginPublicKeyPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$LoginPublicKeyPayloadImplFromJson(json);

  @override
  final String publicKey;
  @override
  final int serverTs;
  @override
  final String algorithm;

  @override
  String toString() {
    return 'LoginPublicKeyPayload(publicKey: $publicKey, serverTs: $serverTs, algorithm: $algorithm)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LoginPublicKeyPayloadImpl &&
            (identical(other.publicKey, publicKey) ||
                other.publicKey == publicKey) &&
            (identical(other.serverTs, serverTs) ||
                other.serverTs == serverTs) &&
            (identical(other.algorithm, algorithm) ||
                other.algorithm == algorithm));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, publicKey, serverTs, algorithm);

  /// Create a copy of LoginPublicKeyPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LoginPublicKeyPayloadImplCopyWith<_$LoginPublicKeyPayloadImpl>
  get copyWith =>
      __$$LoginPublicKeyPayloadImplCopyWithImpl<_$LoginPublicKeyPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$LoginPublicKeyPayloadImplToJson(this);
  }
}

abstract class _LoginPublicKeyPayload implements LoginPublicKeyPayload {
  const factory _LoginPublicKeyPayload({
    required final String publicKey,
    required final int serverTs,
    required final String algorithm,
  }) = _$LoginPublicKeyPayloadImpl;

  factory _LoginPublicKeyPayload.fromJson(Map<String, dynamic> json) =
      _$LoginPublicKeyPayloadImpl.fromJson;

  @override
  String get publicKey;
  @override
  int get serverTs;
  @override
  String get algorithm;

  /// Create a copy of LoginPublicKeyPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LoginPublicKeyPayloadImplCopyWith<_$LoginPublicKeyPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

LoginResult _$LoginResultFromJson(Map<String, dynamic> json) {
  return _LoginResult.fromJson(json);
}

/// @nodoc
mixin _$LoginResult {
  bool get twoFactorRequired => throw _privateConstructorUsedError;
  String? get message => throw _privateConstructorUsedError;

  /// Serializes this LoginResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LoginResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LoginResultCopyWith<LoginResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LoginResultCopyWith<$Res> {
  factory $LoginResultCopyWith(
    LoginResult value,
    $Res Function(LoginResult) then,
  ) = _$LoginResultCopyWithImpl<$Res, LoginResult>;
  @useResult
  $Res call({bool twoFactorRequired, String? message});
}

/// @nodoc
class _$LoginResultCopyWithImpl<$Res, $Val extends LoginResult>
    implements $LoginResultCopyWith<$Res> {
  _$LoginResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LoginResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? twoFactorRequired = null, Object? message = freezed}) {
    return _then(
      _value.copyWith(
            twoFactorRequired: null == twoFactorRequired
                ? _value.twoFactorRequired
                : twoFactorRequired // ignore: cast_nullable_to_non_nullable
                      as bool,
            message: freezed == message
                ? _value.message
                : message // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$LoginResultImplCopyWith<$Res>
    implements $LoginResultCopyWith<$Res> {
  factory _$$LoginResultImplCopyWith(
    _$LoginResultImpl value,
    $Res Function(_$LoginResultImpl) then,
  ) = __$$LoginResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool twoFactorRequired, String? message});
}

/// @nodoc
class __$$LoginResultImplCopyWithImpl<$Res>
    extends _$LoginResultCopyWithImpl<$Res, _$LoginResultImpl>
    implements _$$LoginResultImplCopyWith<$Res> {
  __$$LoginResultImplCopyWithImpl(
    _$LoginResultImpl _value,
    $Res Function(_$LoginResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LoginResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? twoFactorRequired = null, Object? message = freezed}) {
    return _then(
      _$LoginResultImpl(
        twoFactorRequired: null == twoFactorRequired
            ? _value.twoFactorRequired
            : twoFactorRequired // ignore: cast_nullable_to_non_nullable
                  as bool,
        message: freezed == message
            ? _value.message
            : message // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LoginResultImpl implements _LoginResult {
  const _$LoginResultImpl({required this.twoFactorRequired, this.message});

  factory _$LoginResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$LoginResultImplFromJson(json);

  @override
  final bool twoFactorRequired;
  @override
  final String? message;

  @override
  String toString() {
    return 'LoginResult(twoFactorRequired: $twoFactorRequired, message: $message)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LoginResultImpl &&
            (identical(other.twoFactorRequired, twoFactorRequired) ||
                other.twoFactorRequired == twoFactorRequired) &&
            (identical(other.message, message) || other.message == message));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, twoFactorRequired, message);

  /// Create a copy of LoginResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LoginResultImplCopyWith<_$LoginResultImpl> get copyWith =>
      __$$LoginResultImplCopyWithImpl<_$LoginResultImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$LoginResultImplToJson(this);
  }
}

abstract class _LoginResult implements LoginResult {
  const factory _LoginResult({
    required final bool twoFactorRequired,
    final String? message,
  }) = _$LoginResultImpl;

  factory _LoginResult.fromJson(Map<String, dynamic> json) =
      _$LoginResultImpl.fromJson;

  @override
  bool get twoFactorRequired;
  @override
  String? get message;

  /// Create a copy of LoginResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LoginResultImplCopyWith<_$LoginResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OidcExchangeRequest _$OidcExchangeRequestFromJson(Map<String, dynamic> json) {
  return _OidcExchangeRequest.fromJson(json);
}

/// @nodoc
mixin _$OidcExchangeRequest {
  String get code => throw _privateConstructorUsedError;
  String get codeVerifier => throw _privateConstructorUsedError;
  String get nonce => throw _privateConstructorUsedError;
  String get redirectUri => throw _privateConstructorUsedError;

  /// Serializes this OidcExchangeRequest to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of OidcExchangeRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $OidcExchangeRequestCopyWith<OidcExchangeRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OidcExchangeRequestCopyWith<$Res> {
  factory $OidcExchangeRequestCopyWith(
    OidcExchangeRequest value,
    $Res Function(OidcExchangeRequest) then,
  ) = _$OidcExchangeRequestCopyWithImpl<$Res, OidcExchangeRequest>;
  @useResult
  $Res call({
    String code,
    String codeVerifier,
    String nonce,
    String redirectUri,
  });
}

/// @nodoc
class _$OidcExchangeRequestCopyWithImpl<$Res, $Val extends OidcExchangeRequest>
    implements $OidcExchangeRequestCopyWith<$Res> {
  _$OidcExchangeRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of OidcExchangeRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? codeVerifier = null,
    Object? nonce = null,
    Object? redirectUri = null,
  }) {
    return _then(
      _value.copyWith(
            code: null == code
                ? _value.code
                : code // ignore: cast_nullable_to_non_nullable
                      as String,
            codeVerifier: null == codeVerifier
                ? _value.codeVerifier
                : codeVerifier // ignore: cast_nullable_to_non_nullable
                      as String,
            nonce: null == nonce
                ? _value.nonce
                : nonce // ignore: cast_nullable_to_non_nullable
                      as String,
            redirectUri: null == redirectUri
                ? _value.redirectUri
                : redirectUri // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$OidcExchangeRequestImplCopyWith<$Res>
    implements $OidcExchangeRequestCopyWith<$Res> {
  factory _$$OidcExchangeRequestImplCopyWith(
    _$OidcExchangeRequestImpl value,
    $Res Function(_$OidcExchangeRequestImpl) then,
  ) = __$$OidcExchangeRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String code,
    String codeVerifier,
    String nonce,
    String redirectUri,
  });
}

/// @nodoc
class __$$OidcExchangeRequestImplCopyWithImpl<$Res>
    extends _$OidcExchangeRequestCopyWithImpl<$Res, _$OidcExchangeRequestImpl>
    implements _$$OidcExchangeRequestImplCopyWith<$Res> {
  __$$OidcExchangeRequestImplCopyWithImpl(
    _$OidcExchangeRequestImpl _value,
    $Res Function(_$OidcExchangeRequestImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of OidcExchangeRequest
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? code = null,
    Object? codeVerifier = null,
    Object? nonce = null,
    Object? redirectUri = null,
  }) {
    return _then(
      _$OidcExchangeRequestImpl(
        code: null == code
            ? _value.code
            : code // ignore: cast_nullable_to_non_nullable
                  as String,
        codeVerifier: null == codeVerifier
            ? _value.codeVerifier
            : codeVerifier // ignore: cast_nullable_to_non_nullable
                  as String,
        nonce: null == nonce
            ? _value.nonce
            : nonce // ignore: cast_nullable_to_non_nullable
                  as String,
        redirectUri: null == redirectUri
            ? _value.redirectUri
            : redirectUri // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$OidcExchangeRequestImpl implements _OidcExchangeRequest {
  const _$OidcExchangeRequestImpl({
    required this.code,
    required this.codeVerifier,
    required this.nonce,
    required this.redirectUri,
  });

  factory _$OidcExchangeRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$OidcExchangeRequestImplFromJson(json);

  @override
  final String code;
  @override
  final String codeVerifier;
  @override
  final String nonce;
  @override
  final String redirectUri;

  @override
  String toString() {
    return 'OidcExchangeRequest(code: $code, codeVerifier: $codeVerifier, nonce: $nonce, redirectUri: $redirectUri)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OidcExchangeRequestImpl &&
            (identical(other.code, code) || other.code == code) &&
            (identical(other.codeVerifier, codeVerifier) ||
                other.codeVerifier == codeVerifier) &&
            (identical(other.nonce, nonce) || other.nonce == nonce) &&
            (identical(other.redirectUri, redirectUri) ||
                other.redirectUri == redirectUri));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, code, codeVerifier, nonce, redirectUri);

  /// Create a copy of OidcExchangeRequest
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$OidcExchangeRequestImplCopyWith<_$OidcExchangeRequestImpl> get copyWith =>
      __$$OidcExchangeRequestImplCopyWithImpl<_$OidcExchangeRequestImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$OidcExchangeRequestImplToJson(this);
  }
}

abstract class _OidcExchangeRequest implements OidcExchangeRequest {
  const factory _OidcExchangeRequest({
    required final String code,
    required final String codeVerifier,
    required final String nonce,
    required final String redirectUri,
  }) = _$OidcExchangeRequestImpl;

  factory _OidcExchangeRequest.fromJson(Map<String, dynamic> json) =
      _$OidcExchangeRequestImpl.fromJson;

  @override
  String get code;
  @override
  String get codeVerifier;
  @override
  String get nonce;
  @override
  String get redirectUri;

  /// Create a copy of OidcExchangeRequest
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$OidcExchangeRequestImplCopyWith<_$OidcExchangeRequestImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OidcExchangeResult _$OidcExchangeResultFromJson(Map<String, dynamic> json) {
  return _OidcExchangeResult.fromJson(json);
}

/// @nodoc
mixin _$OidcExchangeResult {
  String get token => throw _privateConstructorUsedError;

  /// Serializes this OidcExchangeResult to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of OidcExchangeResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $OidcExchangeResultCopyWith<OidcExchangeResult> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OidcExchangeResultCopyWith<$Res> {
  factory $OidcExchangeResultCopyWith(
    OidcExchangeResult value,
    $Res Function(OidcExchangeResult) then,
  ) = _$OidcExchangeResultCopyWithImpl<$Res, OidcExchangeResult>;
  @useResult
  $Res call({String token});
}

/// @nodoc
class _$OidcExchangeResultCopyWithImpl<$Res, $Val extends OidcExchangeResult>
    implements $OidcExchangeResultCopyWith<$Res> {
  _$OidcExchangeResultCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of OidcExchangeResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? token = null}) {
    return _then(
      _value.copyWith(
            token: null == token
                ? _value.token
                : token // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$OidcExchangeResultImplCopyWith<$Res>
    implements $OidcExchangeResultCopyWith<$Res> {
  factory _$$OidcExchangeResultImplCopyWith(
    _$OidcExchangeResultImpl value,
    $Res Function(_$OidcExchangeResultImpl) then,
  ) = __$$OidcExchangeResultImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String token});
}

/// @nodoc
class __$$OidcExchangeResultImplCopyWithImpl<$Res>
    extends _$OidcExchangeResultCopyWithImpl<$Res, _$OidcExchangeResultImpl>
    implements _$$OidcExchangeResultImplCopyWith<$Res> {
  __$$OidcExchangeResultImplCopyWithImpl(
    _$OidcExchangeResultImpl _value,
    $Res Function(_$OidcExchangeResultImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of OidcExchangeResult
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? token = null}) {
    return _then(
      _$OidcExchangeResultImpl(
        token: null == token
            ? _value.token
            : token // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$OidcExchangeResultImpl implements _OidcExchangeResult {
  const _$OidcExchangeResultImpl({required this.token});

  factory _$OidcExchangeResultImpl.fromJson(Map<String, dynamic> json) =>
      _$$OidcExchangeResultImplFromJson(json);

  @override
  final String token;

  @override
  String toString() {
    return 'OidcExchangeResult(token: $token)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OidcExchangeResultImpl &&
            (identical(other.token, token) || other.token == token));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, token);

  /// Create a copy of OidcExchangeResult
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$OidcExchangeResultImplCopyWith<_$OidcExchangeResultImpl> get copyWith =>
      __$$OidcExchangeResultImplCopyWithImpl<_$OidcExchangeResultImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$OidcExchangeResultImplToJson(this);
  }
}

abstract class _OidcExchangeResult implements OidcExchangeResult {
  const factory _OidcExchangeResult({required final String token}) =
      _$OidcExchangeResultImpl;

  factory _OidcExchangeResult.fromJson(Map<String, dynamic> json) =
      _$OidcExchangeResultImpl.fromJson;

  @override
  String get token;

  /// Create a copy of OidcExchangeResult
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$OidcExchangeResultImplCopyWith<_$OidcExchangeResultImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TotpSetupPayload _$TotpSetupPayloadFromJson(Map<String, dynamic> json) {
  return _TotpSetupPayload.fromJson(json);
}

/// @nodoc
mixin _$TotpSetupPayload {
  String get secret => throw _privateConstructorUsedError;
  String get otpauthUrl => throw _privateConstructorUsedError;

  /// Serializes this TotpSetupPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TotpSetupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TotpSetupPayloadCopyWith<TotpSetupPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TotpSetupPayloadCopyWith<$Res> {
  factory $TotpSetupPayloadCopyWith(
    TotpSetupPayload value,
    $Res Function(TotpSetupPayload) then,
  ) = _$TotpSetupPayloadCopyWithImpl<$Res, TotpSetupPayload>;
  @useResult
  $Res call({String secret, String otpauthUrl});
}

/// @nodoc
class _$TotpSetupPayloadCopyWithImpl<$Res, $Val extends TotpSetupPayload>
    implements $TotpSetupPayloadCopyWith<$Res> {
  _$TotpSetupPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TotpSetupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? secret = null, Object? otpauthUrl = null}) {
    return _then(
      _value.copyWith(
            secret: null == secret
                ? _value.secret
                : secret // ignore: cast_nullable_to_non_nullable
                      as String,
            otpauthUrl: null == otpauthUrl
                ? _value.otpauthUrl
                : otpauthUrl // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TotpSetupPayloadImplCopyWith<$Res>
    implements $TotpSetupPayloadCopyWith<$Res> {
  factory _$$TotpSetupPayloadImplCopyWith(
    _$TotpSetupPayloadImpl value,
    $Res Function(_$TotpSetupPayloadImpl) then,
  ) = __$$TotpSetupPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String secret, String otpauthUrl});
}

/// @nodoc
class __$$TotpSetupPayloadImplCopyWithImpl<$Res>
    extends _$TotpSetupPayloadCopyWithImpl<$Res, _$TotpSetupPayloadImpl>
    implements _$$TotpSetupPayloadImplCopyWith<$Res> {
  __$$TotpSetupPayloadImplCopyWithImpl(
    _$TotpSetupPayloadImpl _value,
    $Res Function(_$TotpSetupPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TotpSetupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? secret = null, Object? otpauthUrl = null}) {
    return _then(
      _$TotpSetupPayloadImpl(
        secret: null == secret
            ? _value.secret
            : secret // ignore: cast_nullable_to_non_nullable
                  as String,
        otpauthUrl: null == otpauthUrl
            ? _value.otpauthUrl
            : otpauthUrl // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TotpSetupPayloadImpl implements _TotpSetupPayload {
  const _$TotpSetupPayloadImpl({
    required this.secret,
    required this.otpauthUrl,
  });

  factory _$TotpSetupPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TotpSetupPayloadImplFromJson(json);

  @override
  final String secret;
  @override
  final String otpauthUrl;

  @override
  String toString() {
    return 'TotpSetupPayload(secret: $secret, otpauthUrl: $otpauthUrl)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TotpSetupPayloadImpl &&
            (identical(other.secret, secret) || other.secret == secret) &&
            (identical(other.otpauthUrl, otpauthUrl) ||
                other.otpauthUrl == otpauthUrl));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, secret, otpauthUrl);

  /// Create a copy of TotpSetupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TotpSetupPayloadImplCopyWith<_$TotpSetupPayloadImpl> get copyWith =>
      __$$TotpSetupPayloadImplCopyWithImpl<_$TotpSetupPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TotpSetupPayloadImplToJson(this);
  }
}

abstract class _TotpSetupPayload implements TotpSetupPayload {
  const factory _TotpSetupPayload({
    required final String secret,
    required final String otpauthUrl,
  }) = _$TotpSetupPayloadImpl;

  factory _TotpSetupPayload.fromJson(Map<String, dynamic> json) =
      _$TotpSetupPayloadImpl.fromJson;

  @override
  String get secret;
  @override
  String get otpauthUrl;

  /// Create a copy of TotpSetupPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TotpSetupPayloadImplCopyWith<_$TotpSetupPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TotpEnablePayload _$TotpEnablePayloadFromJson(Map<String, dynamic> json) {
  return _TotpEnablePayload.fromJson(json);
}

/// @nodoc
mixin _$TotpEnablePayload {
  List<String> get recoveryCodes => throw _privateConstructorUsedError;

  /// Serializes this TotpEnablePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TotpEnablePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TotpEnablePayloadCopyWith<TotpEnablePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TotpEnablePayloadCopyWith<$Res> {
  factory $TotpEnablePayloadCopyWith(
    TotpEnablePayload value,
    $Res Function(TotpEnablePayload) then,
  ) = _$TotpEnablePayloadCopyWithImpl<$Res, TotpEnablePayload>;
  @useResult
  $Res call({List<String> recoveryCodes});
}

/// @nodoc
class _$TotpEnablePayloadCopyWithImpl<$Res, $Val extends TotpEnablePayload>
    implements $TotpEnablePayloadCopyWith<$Res> {
  _$TotpEnablePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TotpEnablePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? recoveryCodes = null}) {
    return _then(
      _value.copyWith(
            recoveryCodes: null == recoveryCodes
                ? _value.recoveryCodes
                : recoveryCodes // ignore: cast_nullable_to_non_nullable
                      as List<String>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TotpEnablePayloadImplCopyWith<$Res>
    implements $TotpEnablePayloadCopyWith<$Res> {
  factory _$$TotpEnablePayloadImplCopyWith(
    _$TotpEnablePayloadImpl value,
    $Res Function(_$TotpEnablePayloadImpl) then,
  ) = __$$TotpEnablePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<String> recoveryCodes});
}

/// @nodoc
class __$$TotpEnablePayloadImplCopyWithImpl<$Res>
    extends _$TotpEnablePayloadCopyWithImpl<$Res, _$TotpEnablePayloadImpl>
    implements _$$TotpEnablePayloadImplCopyWith<$Res> {
  __$$TotpEnablePayloadImplCopyWithImpl(
    _$TotpEnablePayloadImpl _value,
    $Res Function(_$TotpEnablePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TotpEnablePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? recoveryCodes = null}) {
    return _then(
      _$TotpEnablePayloadImpl(
        recoveryCodes: null == recoveryCodes
            ? _value._recoveryCodes
            : recoveryCodes // ignore: cast_nullable_to_non_nullable
                  as List<String>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TotpEnablePayloadImpl implements _TotpEnablePayload {
  const _$TotpEnablePayloadImpl({required final List<String> recoveryCodes})
    : _recoveryCodes = recoveryCodes;

  factory _$TotpEnablePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TotpEnablePayloadImplFromJson(json);

  final List<String> _recoveryCodes;
  @override
  List<String> get recoveryCodes {
    if (_recoveryCodes is EqualUnmodifiableListView) return _recoveryCodes;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_recoveryCodes);
  }

  @override
  String toString() {
    return 'TotpEnablePayload(recoveryCodes: $recoveryCodes)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TotpEnablePayloadImpl &&
            const DeepCollectionEquality().equals(
              other._recoveryCodes,
              _recoveryCodes,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_recoveryCodes),
  );

  /// Create a copy of TotpEnablePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TotpEnablePayloadImplCopyWith<_$TotpEnablePayloadImpl> get copyWith =>
      __$$TotpEnablePayloadImplCopyWithImpl<_$TotpEnablePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TotpEnablePayloadImplToJson(this);
  }
}

abstract class _TotpEnablePayload implements TotpEnablePayload {
  const factory _TotpEnablePayload({
    required final List<String> recoveryCodes,
  }) = _$TotpEnablePayloadImpl;

  factory _TotpEnablePayload.fromJson(Map<String, dynamic> json) =
      _$TotpEnablePayloadImpl.fromJson;

  @override
  List<String> get recoveryCodes;

  /// Create a copy of TotpEnablePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TotpEnablePayloadImplCopyWith<_$TotpEnablePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TotpStatusPayload _$TotpStatusPayloadFromJson(Map<String, dynamic> json) {
  return _TotpStatusPayload.fromJson(json);
}

/// @nodoc
mixin _$TotpStatusPayload {
  bool get enabled => throw _privateConstructorUsedError;

  /// Serializes this TotpStatusPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of TotpStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $TotpStatusPayloadCopyWith<TotpStatusPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TotpStatusPayloadCopyWith<$Res> {
  factory $TotpStatusPayloadCopyWith(
    TotpStatusPayload value,
    $Res Function(TotpStatusPayload) then,
  ) = _$TotpStatusPayloadCopyWithImpl<$Res, TotpStatusPayload>;
  @useResult
  $Res call({bool enabled});
}

/// @nodoc
class _$TotpStatusPayloadCopyWithImpl<$Res, $Val extends TotpStatusPayload>
    implements $TotpStatusPayloadCopyWith<$Res> {
  _$TotpStatusPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of TotpStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? enabled = null}) {
    return _then(
      _value.copyWith(
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$TotpStatusPayloadImplCopyWith<$Res>
    implements $TotpStatusPayloadCopyWith<$Res> {
  factory _$$TotpStatusPayloadImplCopyWith(
    _$TotpStatusPayloadImpl value,
    $Res Function(_$TotpStatusPayloadImpl) then,
  ) = __$$TotpStatusPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({bool enabled});
}

/// @nodoc
class __$$TotpStatusPayloadImplCopyWithImpl<$Res>
    extends _$TotpStatusPayloadCopyWithImpl<$Res, _$TotpStatusPayloadImpl>
    implements _$$TotpStatusPayloadImplCopyWith<$Res> {
  __$$TotpStatusPayloadImplCopyWithImpl(
    _$TotpStatusPayloadImpl _value,
    $Res Function(_$TotpStatusPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of TotpStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? enabled = null}) {
    return _then(
      _$TotpStatusPayloadImpl(
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$TotpStatusPayloadImpl implements _TotpStatusPayload {
  const _$TotpStatusPayloadImpl({required this.enabled});

  factory _$TotpStatusPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$TotpStatusPayloadImplFromJson(json);

  @override
  final bool enabled;

  @override
  String toString() {
    return 'TotpStatusPayload(enabled: $enabled)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TotpStatusPayloadImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, enabled);

  /// Create a copy of TotpStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$TotpStatusPayloadImplCopyWith<_$TotpStatusPayloadImpl> get copyWith =>
      __$$TotpStatusPayloadImplCopyWithImpl<_$TotpStatusPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$TotpStatusPayloadImplToJson(this);
  }
}

abstract class _TotpStatusPayload implements TotpStatusPayload {
  const factory _TotpStatusPayload({required final bool enabled}) =
      _$TotpStatusPayloadImpl;

  factory _TotpStatusPayload.fromJson(Map<String, dynamic> json) =
      _$TotpStatusPayloadImpl.fromJson;

  @override
  bool get enabled;

  /// Create a copy of TotpStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$TotpStatusPayloadImplCopyWith<_$TotpStatusPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OAuthBindingPayload _$OAuthBindingPayloadFromJson(Map<String, dynamic> json) {
  return _OAuthBindingPayload.fromJson(json);
}

/// @nodoc
mixin _$OAuthBindingPayload {
  bool get bound => throw _privateConstructorUsedError;
  String? get provider => throw _privateConstructorUsedError;
  String? get createdAt => throw _privateConstructorUsedError;
  String? get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this OAuthBindingPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of OAuthBindingPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $OAuthBindingPayloadCopyWith<OAuthBindingPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OAuthBindingPayloadCopyWith<$Res> {
  factory $OAuthBindingPayloadCopyWith(
    OAuthBindingPayload value,
    $Res Function(OAuthBindingPayload) then,
  ) = _$OAuthBindingPayloadCopyWithImpl<$Res, OAuthBindingPayload>;
  @useResult
  $Res call({
    bool bound,
    String? provider,
    String? createdAt,
    String? updatedAt,
  });
}

/// @nodoc
class _$OAuthBindingPayloadCopyWithImpl<$Res, $Val extends OAuthBindingPayload>
    implements $OAuthBindingPayloadCopyWith<$Res> {
  _$OAuthBindingPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of OAuthBindingPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? bound = null,
    Object? provider = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _value.copyWith(
            bound: null == bound
                ? _value.bound
                : bound // ignore: cast_nullable_to_non_nullable
                      as bool,
            provider: freezed == provider
                ? _value.provider
                : provider // ignore: cast_nullable_to_non_nullable
                      as String?,
            createdAt: freezed == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as String?,
            updatedAt: freezed == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$OAuthBindingPayloadImplCopyWith<$Res>
    implements $OAuthBindingPayloadCopyWith<$Res> {
  factory _$$OAuthBindingPayloadImplCopyWith(
    _$OAuthBindingPayloadImpl value,
    $Res Function(_$OAuthBindingPayloadImpl) then,
  ) = __$$OAuthBindingPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    bool bound,
    String? provider,
    String? createdAt,
    String? updatedAt,
  });
}

/// @nodoc
class __$$OAuthBindingPayloadImplCopyWithImpl<$Res>
    extends _$OAuthBindingPayloadCopyWithImpl<$Res, _$OAuthBindingPayloadImpl>
    implements _$$OAuthBindingPayloadImplCopyWith<$Res> {
  __$$OAuthBindingPayloadImplCopyWithImpl(
    _$OAuthBindingPayloadImpl _value,
    $Res Function(_$OAuthBindingPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of OAuthBindingPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? bound = null,
    Object? provider = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _$OAuthBindingPayloadImpl(
        bound: null == bound
            ? _value.bound
            : bound // ignore: cast_nullable_to_non_nullable
                  as bool,
        provider: freezed == provider
            ? _value.provider
            : provider // ignore: cast_nullable_to_non_nullable
                  as String?,
        createdAt: freezed == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as String?,
        updatedAt: freezed == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$OAuthBindingPayloadImpl implements _OAuthBindingPayload {
  const _$OAuthBindingPayloadImpl({
    required this.bound,
    this.provider,
    this.createdAt,
    this.updatedAt,
  });

  factory _$OAuthBindingPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$OAuthBindingPayloadImplFromJson(json);

  @override
  final bool bound;
  @override
  final String? provider;
  @override
  final String? createdAt;
  @override
  final String? updatedAt;

  @override
  String toString() {
    return 'OAuthBindingPayload(bound: $bound, provider: $provider, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OAuthBindingPayloadImpl &&
            (identical(other.bound, bound) || other.bound == bound) &&
            (identical(other.provider, provider) ||
                other.provider == provider) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, bound, provider, createdAt, updatedAt);

  /// Create a copy of OAuthBindingPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$OAuthBindingPayloadImplCopyWith<_$OAuthBindingPayloadImpl> get copyWith =>
      __$$OAuthBindingPayloadImplCopyWithImpl<_$OAuthBindingPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$OAuthBindingPayloadImplToJson(this);
  }
}

abstract class _OAuthBindingPayload implements OAuthBindingPayload {
  const factory _OAuthBindingPayload({
    required final bool bound,
    final String? provider,
    final String? createdAt,
    final String? updatedAt,
  }) = _$OAuthBindingPayloadImpl;

  factory _OAuthBindingPayload.fromJson(Map<String, dynamic> json) =
      _$OAuthBindingPayloadImpl.fromJson;

  @override
  bool get bound;
  @override
  String? get provider;
  @override
  String? get createdAt;
  @override
  String? get updatedAt;

  /// Create a copy of OAuthBindingPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$OAuthBindingPayloadImplCopyWith<_$OAuthBindingPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UserSessionPayload _$UserSessionPayloadFromJson(Map<String, dynamic> json) {
  return _UserSessionPayload.fromJson(json);
}

/// @nodoc
mixin _$UserSessionPayload {
  int get id => throw _privateConstructorUsedError;
  String get ipMasked => throw _privateConstructorUsedError;
  String get userAgent => throw _privateConstructorUsedError;
  int get createdAt => throw _privateConstructorUsedError;
  int get expiresAt => throw _privateConstructorUsedError;
  bool get isCurrent => throw _privateConstructorUsedError;

  /// Serializes this UserSessionPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UserSessionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UserSessionPayloadCopyWith<UserSessionPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserSessionPayloadCopyWith<$Res> {
  factory $UserSessionPayloadCopyWith(
    UserSessionPayload value,
    $Res Function(UserSessionPayload) then,
  ) = _$UserSessionPayloadCopyWithImpl<$Res, UserSessionPayload>;
  @useResult
  $Res call({
    int id,
    String ipMasked,
    String userAgent,
    int createdAt,
    int expiresAt,
    bool isCurrent,
  });
}

/// @nodoc
class _$UserSessionPayloadCopyWithImpl<$Res, $Val extends UserSessionPayload>
    implements $UserSessionPayloadCopyWith<$Res> {
  _$UserSessionPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UserSessionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? ipMasked = null,
    Object? userAgent = null,
    Object? createdAt = null,
    Object? expiresAt = null,
    Object? isCurrent = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            ipMasked: null == ipMasked
                ? _value.ipMasked
                : ipMasked // ignore: cast_nullable_to_non_nullable
                      as String,
            userAgent: null == userAgent
                ? _value.userAgent
                : userAgent // ignore: cast_nullable_to_non_nullable
                      as String,
            createdAt: null == createdAt
                ? _value.createdAt
                : createdAt // ignore: cast_nullable_to_non_nullable
                      as int,
            expiresAt: null == expiresAt
                ? _value.expiresAt
                : expiresAt // ignore: cast_nullable_to_non_nullable
                      as int,
            isCurrent: null == isCurrent
                ? _value.isCurrent
                : isCurrent // ignore: cast_nullable_to_non_nullable
                      as bool,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UserSessionPayloadImplCopyWith<$Res>
    implements $UserSessionPayloadCopyWith<$Res> {
  factory _$$UserSessionPayloadImplCopyWith(
    _$UserSessionPayloadImpl value,
    $Res Function(_$UserSessionPayloadImpl) then,
  ) = __$$UserSessionPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String ipMasked,
    String userAgent,
    int createdAt,
    int expiresAt,
    bool isCurrent,
  });
}

/// @nodoc
class __$$UserSessionPayloadImplCopyWithImpl<$Res>
    extends _$UserSessionPayloadCopyWithImpl<$Res, _$UserSessionPayloadImpl>
    implements _$$UserSessionPayloadImplCopyWith<$Res> {
  __$$UserSessionPayloadImplCopyWithImpl(
    _$UserSessionPayloadImpl _value,
    $Res Function(_$UserSessionPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UserSessionPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? ipMasked = null,
    Object? userAgent = null,
    Object? createdAt = null,
    Object? expiresAt = null,
    Object? isCurrent = null,
  }) {
    return _then(
      _$UserSessionPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        ipMasked: null == ipMasked
            ? _value.ipMasked
            : ipMasked // ignore: cast_nullable_to_non_nullable
                  as String,
        userAgent: null == userAgent
            ? _value.userAgent
            : userAgent // ignore: cast_nullable_to_non_nullable
                  as String,
        createdAt: null == createdAt
            ? _value.createdAt
            : createdAt // ignore: cast_nullable_to_non_nullable
                  as int,
        expiresAt: null == expiresAt
            ? _value.expiresAt
            : expiresAt // ignore: cast_nullable_to_non_nullable
                  as int,
        isCurrent: null == isCurrent
            ? _value.isCurrent
            : isCurrent // ignore: cast_nullable_to_non_nullable
                  as bool,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UserSessionPayloadImpl implements _UserSessionPayload {
  const _$UserSessionPayloadImpl({
    required this.id,
    required this.ipMasked,
    required this.userAgent,
    required this.createdAt,
    required this.expiresAt,
    required this.isCurrent,
  });

  factory _$UserSessionPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserSessionPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String ipMasked;
  @override
  final String userAgent;
  @override
  final int createdAt;
  @override
  final int expiresAt;
  @override
  final bool isCurrent;

  @override
  String toString() {
    return 'UserSessionPayload(id: $id, ipMasked: $ipMasked, userAgent: $userAgent, createdAt: $createdAt, expiresAt: $expiresAt, isCurrent: $isCurrent)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserSessionPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.ipMasked, ipMasked) ||
                other.ipMasked == ipMasked) &&
            (identical(other.userAgent, userAgent) ||
                other.userAgent == userAgent) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.expiresAt, expiresAt) ||
                other.expiresAt == expiresAt) &&
            (identical(other.isCurrent, isCurrent) ||
                other.isCurrent == isCurrent));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    ipMasked,
    userAgent,
    createdAt,
    expiresAt,
    isCurrent,
  );

  /// Create a copy of UserSessionPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UserSessionPayloadImplCopyWith<_$UserSessionPayloadImpl> get copyWith =>
      __$$UserSessionPayloadImplCopyWithImpl<_$UserSessionPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UserSessionPayloadImplToJson(this);
  }
}

abstract class _UserSessionPayload implements UserSessionPayload {
  const factory _UserSessionPayload({
    required final int id,
    required final String ipMasked,
    required final String userAgent,
    required final int createdAt,
    required final int expiresAt,
    required final bool isCurrent,
  }) = _$UserSessionPayloadImpl;

  factory _UserSessionPayload.fromJson(Map<String, dynamic> json) =
      _$UserSessionPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get ipMasked;
  @override
  String get userAgent;
  @override
  int get createdAt;
  @override
  int get expiresAt;
  @override
  bool get isCurrent;

  /// Create a copy of UserSessionPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UserSessionPayloadImplCopyWith<_$UserSessionPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SuccessMessagePayload _$SuccessMessagePayloadFromJson(
  Map<String, dynamic> json,
) {
  return _SuccessMessagePayload.fromJson(json);
}

/// @nodoc
mixin _$SuccessMessagePayload {
  String get message => throw _privateConstructorUsedError;

  /// Serializes this SuccessMessagePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SuccessMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SuccessMessagePayloadCopyWith<SuccessMessagePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SuccessMessagePayloadCopyWith<$Res> {
  factory $SuccessMessagePayloadCopyWith(
    SuccessMessagePayload value,
    $Res Function(SuccessMessagePayload) then,
  ) = _$SuccessMessagePayloadCopyWithImpl<$Res, SuccessMessagePayload>;
  @useResult
  $Res call({String message});
}

/// @nodoc
class _$SuccessMessagePayloadCopyWithImpl<
  $Res,
  $Val extends SuccessMessagePayload
>
    implements $SuccessMessagePayloadCopyWith<$Res> {
  _$SuccessMessagePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SuccessMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? message = null}) {
    return _then(
      _value.copyWith(
            message: null == message
                ? _value.message
                : message // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SuccessMessagePayloadImplCopyWith<$Res>
    implements $SuccessMessagePayloadCopyWith<$Res> {
  factory _$$SuccessMessagePayloadImplCopyWith(
    _$SuccessMessagePayloadImpl value,
    $Res Function(_$SuccessMessagePayloadImpl) then,
  ) = __$$SuccessMessagePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String message});
}

/// @nodoc
class __$$SuccessMessagePayloadImplCopyWithImpl<$Res>
    extends
        _$SuccessMessagePayloadCopyWithImpl<$Res, _$SuccessMessagePayloadImpl>
    implements _$$SuccessMessagePayloadImplCopyWith<$Res> {
  __$$SuccessMessagePayloadImplCopyWithImpl(
    _$SuccessMessagePayloadImpl _value,
    $Res Function(_$SuccessMessagePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SuccessMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? message = null}) {
    return _then(
      _$SuccessMessagePayloadImpl(
        message: null == message
            ? _value.message
            : message // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SuccessMessagePayloadImpl implements _SuccessMessagePayload {
  const _$SuccessMessagePayloadImpl({required this.message});

  factory _$SuccessMessagePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SuccessMessagePayloadImplFromJson(json);

  @override
  final String message;

  @override
  String toString() {
    return 'SuccessMessagePayload(message: $message)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SuccessMessagePayloadImpl &&
            (identical(other.message, message) || other.message == message));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, message);

  /// Create a copy of SuccessMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SuccessMessagePayloadImplCopyWith<_$SuccessMessagePayloadImpl>
  get copyWith =>
      __$$SuccessMessagePayloadImplCopyWithImpl<_$SuccessMessagePayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SuccessMessagePayloadImplToJson(this);
  }
}

abstract class _SuccessMessagePayload implements SuccessMessagePayload {
  const factory _SuccessMessagePayload({required final String message}) =
      _$SuccessMessagePayloadImpl;

  factory _SuccessMessagePayload.fromJson(Map<String, dynamic> json) =
      _$SuccessMessagePayloadImpl.fromJson;

  @override
  String get message;

  /// Create a copy of SuccessMessagePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SuccessMessagePayloadImplCopyWith<_$SuccessMessagePayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}
