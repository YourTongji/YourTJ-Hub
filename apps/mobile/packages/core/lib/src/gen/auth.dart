import 'package:freezed_annotation/freezed_annotation.dart';

part 'auth.freezed.dart';
part 'auth.g.dart';

@freezed
abstract class LoginPageProps with _$LoginPageProps {
  const factory LoginPageProps({
    required String initialMode,
    required String redirectUrl,
    required String githubUrl,
    required bool googleReady,
    String? casdoorUrl,
  }) = _LoginPageProps;

  factory LoginPageProps.fromJson(Map<String, dynamic> json) =>
      _$LoginPagePropsFromJson(json);
}

@freezed
abstract class ResetPasswordPageProps with _$ResetPasswordPageProps {
  const factory ResetPasswordPageProps({required String token}) =
      _ResetPasswordPageProps;

  factory ResetPasswordPageProps.fromJson(Map<String, dynamic> json) =>
      _$ResetPasswordPagePropsFromJson(json);
}

@freezed
abstract class CaptchaPayload with _$CaptchaPayload {
  const factory CaptchaPayload({
    required String captchaId,
    required String captchaImg,
  }) = _CaptchaPayload;

  factory CaptchaPayload.fromJson(Map<String, dynamic> json) =>
      _$CaptchaPayloadFromJson(json);
}

@freezed
abstract class LoginPublicKeyPayload with _$LoginPublicKeyPayload {
  const factory LoginPublicKeyPayload({
    required String publicKey,
    required int serverTs,
    required String algorithm,
  }) = _LoginPublicKeyPayload;

  factory LoginPublicKeyPayload.fromJson(Map<String, dynamic> json) =>
      _$LoginPublicKeyPayloadFromJson(json);
}

@freezed
abstract class LoginResult with _$LoginResult {
  const factory LoginResult({
    required bool twoFactorRequired,
    String? message,
  }) = _LoginResult;

  factory LoginResult.fromJson(Map<String, dynamic> json) =>
      _$LoginResultFromJson(json);
}

@freezed
abstract class OidcExchangeRequest with _$OidcExchangeRequest {
  const factory OidcExchangeRequest({
    required String code,
    required String codeVerifier,
    required String nonce,
    required String redirectUri,
  }) = _OidcExchangeRequest;

  factory OidcExchangeRequest.fromJson(Map<String, dynamic> json) =>
      _$OidcExchangeRequestFromJson(json);
}

@freezed
abstract class OidcExchangeResult with _$OidcExchangeResult {
  const factory OidcExchangeResult({required String token}) =
      _OidcExchangeResult;

  factory OidcExchangeResult.fromJson(Map<String, dynamic> json) =>
      _$OidcExchangeResultFromJson(json);
}

@freezed
abstract class TotpSetupPayload with _$TotpSetupPayload {
  const factory TotpSetupPayload({
    required String secret,
    required String otpauthUrl,
  }) = _TotpSetupPayload;

  factory TotpSetupPayload.fromJson(Map<String, dynamic> json) =>
      _$TotpSetupPayloadFromJson(json);
}

@freezed
abstract class TotpEnablePayload with _$TotpEnablePayload {
  const factory TotpEnablePayload({required List<String> recoveryCodes}) =
      _TotpEnablePayload;

  factory TotpEnablePayload.fromJson(Map<String, dynamic> json) =>
      _$TotpEnablePayloadFromJson(json);
}

@freezed
abstract class TotpStatusPayload with _$TotpStatusPayload {
  const factory TotpStatusPayload({required bool enabled}) = _TotpStatusPayload;

  factory TotpStatusPayload.fromJson(Map<String, dynamic> json) =>
      _$TotpStatusPayloadFromJson(json);
}

@freezed
abstract class OAuthBindingPayload with _$OAuthBindingPayload {
  const factory OAuthBindingPayload({
    required bool bound,
    String? provider,
    String? createdAt,
    String? updatedAt,
  }) = _OAuthBindingPayload;

  factory OAuthBindingPayload.fromJson(Map<String, dynamic> json) =>
      _$OAuthBindingPayloadFromJson(json);
}

@freezed
abstract class UserSessionPayload with _$UserSessionPayload {
  const factory UserSessionPayload({
    required int id,
    required String ipMasked,
    required String userAgent,
    required int createdAt,
    required int expiresAt,
    required bool isCurrent,
  }) = _UserSessionPayload;

  factory UserSessionPayload.fromJson(Map<String, dynamic> json) =>
      _$UserSessionPayloadFromJson(json);
}

/// 登录/注册/找回密码等接口的成功消息,取自 component.MessageCode 文案。
@freezed
abstract class SuccessMessagePayload with _$SuccessMessagePayload {
  const factory SuccessMessagePayload({required String message}) =
      _SuccessMessagePayload;

  factory SuccessMessagePayload.fromJson(Map<String, dynamic> json) =>
      _$SuccessMessagePayloadFromJson(json);
}
