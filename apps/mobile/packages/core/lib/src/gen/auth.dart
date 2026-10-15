import 'package:freezed_annotation/freezed_annotation.dart';

part 'auth.freezed.dart';
part 'auth.g.dart';

@freezed
abstract class LoginPageProps with _$LoginPageProps {
  const factory LoginPageProps({
    @Default(false) bool tongjiRegistration,
    required String initialMode,
    required String redirectUrl,
    required String githubUrl,
    required bool googleReady,
    @Default(false) bool appleReady,
    @Default(false) bool tongjiReady,
    @Default('') String tongjiUrl,
    @Default('') String tongjiNotice,
    @Default(false) bool termsOfServiceEnabled,
    @Default(false) bool privacyPolicyEnabled,
    @Default(<String>[]) List<String> allowedDomains,

    /// issue #531：OAuth 回调无本地账号时跳转注册页的提示标记。
    @Default(false) bool oauthNotice,
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

/// Dart mirror of the passkey API schemas; these remain intentionally simple
/// so updating the contract does not require regenerating unrelated Freezed code.
class PasskeyStatusPayload {
  const PasskeyStatusPayload({required this.enabled});
  final bool enabled;

  factory PasskeyStatusPayload.fromJson(Map<String, dynamic> json) =>
      PasskeyStatusPayload(enabled: json['enabled'] == true);
}

class PasskeyBeginPayload {
  const PasskeyBeginPayload({required this.ceremonyId, required this.publicKey});
  final String ceremonyId;
  final Map<String, dynamic> publicKey;

  factory PasskeyBeginPayload.fromJson(Map<String, dynamic> json) =>
      PasskeyBeginPayload(
        ceremonyId: json['ceremonyId'] as String,
        publicKey: Map<String, dynamic>.from(json['publicKey'] as Map),
      );
}

class PasskeyReauthPayload {
  const PasskeyReauthPayload({required this.proof, required this.expiresInSeconds});
  final String proof;
  final int expiresInSeconds;

  factory PasskeyReauthPayload.fromJson(Map<String, dynamic> json) =>
      PasskeyReauthPayload(
        proof: json['proof'] as String,
        expiresInSeconds: (json['expiresInSeconds'] as num).toInt(),
      );
}

class PasskeyCredentialSummary {
  const PasskeyCredentialSummary({
    required this.id,
    required this.name,
    required this.createdAt,
    required this.backupEligible,
    required this.backupState,
    required this.transports,
    this.lastUsedAt,
  });
  final String id;
  final String name;
  final DateTime createdAt;
  final DateTime? lastUsedAt;
  final bool backupEligible;
  final bool backupState;
  final List<String> transports;

  factory PasskeyCredentialSummary.fromJson(Map<String, dynamic> json) =>
      PasskeyCredentialSummary(
        id: json['id'] as String,
        name: json['name'] as String,
        createdAt: DateTime.parse(json['createdAt'] as String),
        lastUsedAt: json['lastUsedAt'] == null
            ? null
            : DateTime.parse(json['lastUsedAt'] as String),
        backupEligible: json['backupEligible'] == true,
        backupState: json['backupState'] == true,
        transports: (json['transports'] as List<dynamic>? ?? const <dynamic>[])
            .map((value) => value as String)
            .toList(growable: false),
      );

  Map<String, dynamic> toJson() => <String, dynamic>{
    'id': id,
    'name': name,
    'createdAt': createdAt.toIso8601String(),
    if (lastUsedAt != null) 'lastUsedAt': lastUsedAt!.toIso8601String(),
    'backupEligible': backupEligible,
    'backupState': backupState,
    'transports': transports,
  };
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

/// Query mirror for GET /api/auth/mobile-web-session (fixed server targets).
enum MobileWebTarget {
  admin,
  moderation,
  courseManagement,
  courseReviews,
  campus,
}

@freezed
abstract class TongjiRegistrationStatusPayload
    with _$TongjiRegistrationStatusPayload {
  const factory TongjiRegistrationStatusPayload({
    required String csrfToken,
    required String email,
    required String expiresAt,
  }) = _TongjiRegistrationStatusPayload;
  factory TongjiRegistrationStatusPayload.fromJson(Map<String, dynamic> json) =>
      _$TongjiRegistrationStatusPayloadFromJson(json);
}

@freezed
abstract class TongjiRegistrationRequest with _$TongjiRegistrationRequest {
  const factory TongjiRegistrationRequest({
    required String username,
    required String password,
    required String csrfToken,
  }) = _TongjiRegistrationRequest;
  factory TongjiRegistrationRequest.fromJson(Map<String, dynamic> json) =>
      _$TongjiRegistrationRequestFromJson(json);
}

@freezed
abstract class TongjiRegistrationResult with _$TongjiRegistrationResult {
  const factory TongjiRegistrationResult({required String redirect}) =
      _TongjiRegistrationResult;
  factory TongjiRegistrationResult.fromJson(Map<String, dynamic> json) =>
      _$TongjiRegistrationResultFromJson(json);
}

@freezed
abstract class AppleCredentialRequest with _$AppleCredentialRequest {
  const factory AppleCredentialRequest({
    required String authorizationCode,
    required String identityToken,
    required String nonce,
  }) = _AppleCredentialRequest;
  factory AppleCredentialRequest.fromJson(Map<String, dynamic> json) => _$AppleCredentialRequestFromJson(json);
}
