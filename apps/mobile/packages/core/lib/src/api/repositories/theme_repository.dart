import '../api_error.dart';
import '../../gen/site_theme.dart';
import '../gf_api_client.dart';

/// 站点主题公开下发（`GET /api/site-theme/tokens`，公开只读）。
class ThemeRepository {
  ThemeRepository(this._client);

  final GfApiClient _client;

  /// 读取发布态主题 tokens；未启用/未发布返回 enabled=false，
  /// 旧后端未部署该端点（404）时返回 null（调用方回退内置主题）。
  Future<SiteThemePublicPayload?> fetchTokens() async {
    try {
      return await _client.get<SiteThemePublicPayload>(
        '/api/site-theme/tokens',
        parser: (json) => SiteThemePublicPayload.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );
    } on ApiException {
      return null;
    }
  }
}
