import 'package:flutter/foundation.dart';

/// 应用运行时配置(经 --dart-define 注入,提供安全默认)。
///
/// 用法示例:
/// ```bash
/// flutter run --dart-define=YOURTJ_OIDC_ISSUER=https://casdoor.example.com \
///             --dart-define=YOURTJ_OIDC_CLIENT_ID=forum-app
/// ```
abstract final class AppConfig {
  static const String _issuer = String.fromEnvironment(
    'YOURTJ_OIDC_ISSUER',
    defaultValue: 'http://localhost:8001',
  );

  static const String _clientId = String.fromEnvironment(
    'YOURTJ_OIDC_CLIENT_ID',
    defaultValue: 'forum-app',
  );

  static const String _apiBaseUrl = String.fromEnvironment(
    'YOURTJ_API_BASE_URL',
    defaultValue: '',
  );

  /// Casdoor/OIDC issuer(本地开发默认 localhost:8001)。
  static String get oidcIssuer => _issuer;

  /// Casdoor 应用 client_id。
  static String get oidcClientId => _clientId;

  /// API baseUrl;为空时由 GfApiClient 使用平台默认(Android 模拟器 10.0.2.2)。
  static String get apiBaseUrl => _apiBaseUrl;

  /// 是否为调试模式(允许非 HTTPS OIDC 端点)。
  static bool get allowInsecureOidc => kDebugMode;
}
