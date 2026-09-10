import 'package:flutter/foundation.dart';

/// 应用运行时配置(经 --dart-define 注入,提供安全默认)。
///
/// 用法示例:
/// ```bash
/// flutter run --dart-define=YOURTJ_OIDC_ISSUER=https://forum.example.com/api/oauth \
///             --dart-define=YOURTJ_OIDC_CLIENT_ID=yourtj-mobile
/// ```
abstract final class AppConfig {
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

  /// 是否为调试模式(允许非 HTTPS OIDC 端点)。
  static bool get allowInsecureOidc => kDebugMode;
}
