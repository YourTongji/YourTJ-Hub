import '../../gen/external_link_policy.dart';
import '../api_error.dart';
import '../gf_api_client.dart';

/// 外链免确认白名单公开下发（`GET /api/external-link-policy`，公开只读）。
class ExternalLinkPolicyRepository {
  ExternalLinkPolicyRepository(this._client);

  final GfApiClient _client;

  /// 读取管理员配置的白名单；旧后端未部署该端点（404）时返回 null，
  /// 调用方保留当前清单（未命中白名单的外链仍走确认弹层）。
  Future<List<String>?> fetchTrustedDomains() async {
    try {
      final ExternalLinkPolicyPayload payload = await _client
          .get<ExternalLinkPolicyPayload>(
            '/api/external-link-policy',
            parser: (json) => ExternalLinkPolicyPayload.fromJson(
              Map<String, dynamic>.from(json as Map),
            ),
          );
      return payload.trustedDomains;
    } on ApiException {
      return null;
    }
  }
}
