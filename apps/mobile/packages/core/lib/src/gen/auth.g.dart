// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$LoginPagePropsImpl _$$LoginPagePropsImplFromJson(Map<String, dynamic> json) =>
    _$LoginPagePropsImpl(
      initialMode: json['initialMode'] as String,
      redirectUrl: json['redirectUrl'] as String,
      githubUrl: json['githubUrl'] as String,
      googleReady: json['googleReady'] as bool,
    );

Map<String, dynamic> _$$LoginPagePropsImplToJson(
  _$LoginPagePropsImpl instance,
) => <String, dynamic>{
  'initialMode': instance.initialMode,
  'redirectUrl': instance.redirectUrl,
  'githubUrl': instance.githubUrl,
  'googleReady': instance.googleReady,
};

_$ResetPasswordPagePropsImpl _$$ResetPasswordPagePropsImplFromJson(
  Map<String, dynamic> json,
) => _$ResetPasswordPagePropsImpl(token: json['token'] as String);

Map<String, dynamic> _$$ResetPasswordPagePropsImplToJson(
  _$ResetPasswordPagePropsImpl instance,
) => <String, dynamic>{'token': instance.token};

_$CaptchaPayloadImpl _$$CaptchaPayloadImplFromJson(Map<String, dynamic> json) =>
    _$CaptchaPayloadImpl(
      captchaId: json['captchaId'] as String,
      captchaImg: json['captchaImg'] as String,
    );

Map<String, dynamic> _$$CaptchaPayloadImplToJson(
  _$CaptchaPayloadImpl instance,
) => <String, dynamic>{
  'captchaId': instance.captchaId,
  'captchaImg': instance.captchaImg,
};

_$LoginPublicKeyPayloadImpl _$$LoginPublicKeyPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$LoginPublicKeyPayloadImpl(
  publicKey: json['publicKey'] as String,
  serverTs: (json['serverTs'] as num).toInt(),
  algorithm: json['algorithm'] as String,
);

Map<String, dynamic> _$$LoginPublicKeyPayloadImplToJson(
  _$LoginPublicKeyPayloadImpl instance,
) => <String, dynamic>{
  'publicKey': instance.publicKey,
  'serverTs': instance.serverTs,
  'algorithm': instance.algorithm,
};

_$LoginResultImpl _$$LoginResultImplFromJson(Map<String, dynamic> json) =>
    _$LoginResultImpl(
      twoFactorRequired: json['twoFactorRequired'] as bool,
      message: json['message'] as String?,
    );

Map<String, dynamic> _$$LoginResultImplToJson(_$LoginResultImpl instance) =>
    <String, dynamic>{
      'twoFactorRequired': instance.twoFactorRequired,
      'message': instance.message,
    };

_$OidcExchangeRequestImpl _$$OidcExchangeRequestImplFromJson(
  Map<String, dynamic> json,
) => _$OidcExchangeRequestImpl(
  code: json['code'] as String,
  codeVerifier: json['codeVerifier'] as String,
  nonce: json['nonce'] as String,
  redirectUri: json['redirectUri'] as String,
);

Map<String, dynamic> _$$OidcExchangeRequestImplToJson(
  _$OidcExchangeRequestImpl instance,
) => <String, dynamic>{
  'code': instance.code,
  'codeVerifier': instance.codeVerifier,
  'nonce': instance.nonce,
  'redirectUri': instance.redirectUri,
};

_$OidcExchangeResultImpl _$$OidcExchangeResultImplFromJson(
  Map<String, dynamic> json,
) => _$OidcExchangeResultImpl(token: json['token'] as String);

Map<String, dynamic> _$$OidcExchangeResultImplToJson(
  _$OidcExchangeResultImpl instance,
) => <String, dynamic>{'token': instance.token};

_$TotpSetupPayloadImpl _$$TotpSetupPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$TotpSetupPayloadImpl(
  secret: json['secret'] as String,
  otpauthUrl: json['otpauthUrl'] as String,
);

Map<String, dynamic> _$$TotpSetupPayloadImplToJson(
  _$TotpSetupPayloadImpl instance,
) => <String, dynamic>{
  'secret': instance.secret,
  'otpauthUrl': instance.otpauthUrl,
};

_$TotpEnablePayloadImpl _$$TotpEnablePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$TotpEnablePayloadImpl(
  recoveryCodes: (json['recoveryCodes'] as List<dynamic>)
      .map((e) => e as String)
      .toList(),
);

Map<String, dynamic> _$$TotpEnablePayloadImplToJson(
  _$TotpEnablePayloadImpl instance,
) => <String, dynamic>{'recoveryCodes': instance.recoveryCodes};

_$TotpStatusPayloadImpl _$$TotpStatusPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$TotpStatusPayloadImpl(enabled: json['enabled'] as bool);

Map<String, dynamic> _$$TotpStatusPayloadImplToJson(
  _$TotpStatusPayloadImpl instance,
) => <String, dynamic>{'enabled': instance.enabled};

_$OAuthBindingPayloadImpl _$$OAuthBindingPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$OAuthBindingPayloadImpl(
  bound: json['bound'] as bool,
  provider: json['provider'] as String?,
  createdAt: json['createdAt'] as String?,
  updatedAt: json['updatedAt'] as String?,
);

Map<String, dynamic> _$$OAuthBindingPayloadImplToJson(
  _$OAuthBindingPayloadImpl instance,
) => <String, dynamic>{
  'bound': instance.bound,
  'provider': instance.provider,
  'createdAt': instance.createdAt,
  'updatedAt': instance.updatedAt,
};

_$UserSessionPayloadImpl _$$UserSessionPayloadImplFromJson(
  Map<String, dynamic> json,
) => _$UserSessionPayloadImpl(
  id: (json['id'] as num).toInt(),
  ipMasked: json['ipMasked'] as String,
  userAgent: json['userAgent'] as String,
  createdAt: (json['createdAt'] as num).toInt(),
  expiresAt: (json['expiresAt'] as num).toInt(),
  isCurrent: json['isCurrent'] as bool,
);

Map<String, dynamic> _$$UserSessionPayloadImplToJson(
  _$UserSessionPayloadImpl instance,
) => <String, dynamic>{
  'id': instance.id,
  'ipMasked': instance.ipMasked,
  'userAgent': instance.userAgent,
  'createdAt': instance.createdAt,
  'expiresAt': instance.expiresAt,
  'isCurrent': instance.isCurrent,
};

_$SuccessMessagePayloadImpl _$$SuccessMessagePayloadImplFromJson(
  Map<String, dynamic> json,
) => _$SuccessMessagePayloadImpl(message: json['message'] as String);

Map<String, dynamic> _$$SuccessMessagePayloadImplToJson(
  _$SuccessMessagePayloadImpl instance,
) => <String, dynamic>{'message': instance.message};
