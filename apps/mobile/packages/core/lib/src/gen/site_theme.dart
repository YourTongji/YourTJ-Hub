/// 站点主题公开下发契约镜像（`GET /api/site-theme/tokens` 的 data）。
///
/// 手写维护；服务端未启用/未发布时返回 `{enabled:false}`。
/// tokens 键集合与 web `tokens.css` / mobile `tokens.json` 的
/// siteThemeTokenKeys 一致（27 键），逐键 hex 校验由 GfRuntimeTheme 负责。
library;

/// 单个模式的主题 token 集（mode: "light" | "dark"）。
class SiteThemeTokensPayload {
  const SiteThemeTokensPayload({required this.mode, required this.tokens});

  final String mode;
  final Map<String, String> tokens;

  factory SiteThemeTokensPayload.fromJson(Map<String, dynamic> json) =>
      SiteThemeTokensPayload(
        mode: json['mode'] as String? ?? '',
        tokens: (json['tokens'] as Map<String, dynamic>? ?? const {}).map(
          (k, v) => MapEntry(k, v?.toString() ?? ''),
        ),
      );
}

/// GET /api/site-theme/tokens 响应 data。
class SiteThemePublicPayload {
  const SiteThemePublicPayload({
    required this.enabled,
    this.version = 0,
    this.publishedAt,
    this.themes = const [],
  });

  final bool enabled;
  final int version;
  final String? publishedAt; // RFC3339
  final List<SiteThemeTokensPayload> themes;

  factory SiteThemePublicPayload.fromJson(Map<String, dynamic> json) =>
      SiteThemePublicPayload(
        enabled: json['enabled'] as bool? ?? false,
        version: (json['version'] as num?)?.toInt() ?? 0,
        publishedAt: json['publishedAt'] as String?,
        themes: (json['themes'] as List<dynamic>? ?? const [])
            .map(
              (e) => SiteThemeTokensPayload.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      );
}
