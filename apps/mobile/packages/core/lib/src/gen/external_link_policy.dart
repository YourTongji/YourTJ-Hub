/// 外链免确认白名单契约镜像（`GET /api/external-link-policy` 的 data，issue #1200）。
///
/// OpenAPI 仍是事实源（packages/api-contract/paths/external-link-policy.yaml）；
/// Dart 生成尚未启用，本文件按其他手写镜像的约定维护：条目为小写主机名
/// （IDN 为 punycode），条目覆盖其全部子域。
library;

/// GET /api/external-link-policy 响应 data。
class ExternalLinkPolicyPayload {
  const ExternalLinkPolicyPayload({this.trustedDomains = const []});

  /// 管理员配置的免确认白名单；空列表表示不豁免任何域名。
  final List<String> trustedDomains;

  factory ExternalLinkPolicyPayload.fromJson(Map<String, dynamic> json) =>
      ExternalLinkPolicyPayload(
        trustedDomains: (json['trustedDomains'] as List<dynamic>? ?? const [])
            .map((dynamic item) => item?.toString() ?? '')
            .where((String item) => item.isNotEmpty)
            .toList(growable: false),
      );
}
