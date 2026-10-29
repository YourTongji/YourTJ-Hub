import 'package:flutter/foundation.dart';

/// 应用运行时配置(经 --dart-define 注入,提供安全默认)。
///
/// 用法示例:
/// ```bash
/// flutter run --dart-define=YOURTJ_OIDC_ISSUER=https://forum.example.com/api/oauth \
///             --dart-define=YOURTJ_OIDC_CLIENT_ID=yourtj-mobile
/// ```
abstract final class AppConfig {
  static const environment = String.fromEnvironment(
    'YOURTJ_ENVIRONMENT',
    defaultValue: 'production',
  );
  static const sourceSha = String.fromEnvironment('YOURTJ_SOURCE_SHA');
  static const preview = bool.fromEnvironment('YOURTJ_PREVIEW');
  static bool get isDev => environment == 'dev';
  static String get sessionNamespace => isDev ? 'https://dev.yourtj.de' : '';
  static String get previewLabel =>
      '${isDev ? 'DEV' : 'PROD TEST'} ${sourceSha.length >= 7 ? sourceSha.substring(0, 7) : 'local'}';

  static void validate() {
    validateValues(
      environment: environment,
      preview: preview,
      apiBaseUrl: apiBaseUrl,
      oidcIssuer: oidcIssuer,
    );
  }

  @visibleForTesting
  static void validateValues({
    required String environment,
    required bool preview,
    required String apiBaseUrl,
    required String oidcIssuer,
  }) {
    if (environment != 'dev' && environment != 'production') {
      throw StateError('Unknown build environment');
    }
    final isDevBuild = environment == 'dev';
    if (isDevBuild &&
        (!preview ||
            apiBaseUrl != 'https://dev.yourtj.de' ||
            oidcIssuer != 'https://dev.yourtj.de/api/oauth')) {
      throw StateError(
        'Dev builds require dev API, OIDC and preview distribution',
      );
    }
    if (!isDevBuild && apiBaseUrl == 'https://dev.yourtj.de') {
      throw StateError('Dev API requires the dev environment');
    }
  }

  static const String _issuer = String.fromEnvironment(
    'YOURTJ_OIDC_ISSUER',
    defaultValue: 'http://localhost:5234/api/oauth',
  );

  static const String _clientId = String.fromEnvironment(
    'YOURTJ_OIDC_CLIENT_ID',
    defaultValue: 'yourtj-mobile',
  );

  static const String _apiBaseUrl = String.fromEnvironment(
    'YOURTJ_API_BASE_URL',
    defaultValue: '',
  );

  /// 论坛内建 OIDC Provider issuer（本地开发默认 localhost:5234/api/oauth）。
  static String get oidcIssuer => _issuer;

  /// 论坛移动端 OIDC client_id。
  static String get oidcClientId => _clientId;

  /// API baseUrl;为空时由 GfApiClient 使用平台默认(Android 模拟器 10.0.2.2)。
  static String get apiBaseUrl => _apiBaseUrl;

  /// Comma-separated, exact asset origins trusted for automatic chat images.
  /// The API origin is always trusted; never infer trust from received messages.
  static const String chatImageOrigins = String.fromEnvironment(
    'YOURTJ_CHAT_IMAGE_ORIGINS',
  );

  /// 是否为调试模式(允许非 HTTPS OIDC 端点)。
  static bool get allowInsecureOidc => kDebugMode;
}
