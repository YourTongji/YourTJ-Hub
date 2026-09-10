import '../../gen/wiki.dart';
import '../../gen/wiki_search.dart';
import '../gf_api_client.dart';

/// Wiki 只读域（`/api/wiki/tree|namespaces|home|search`，公开只读）。
///
/// 页面正文走页面级数据通道（`PageRepository.fetch('/wiki/...')`，
/// X-Goose-Page 返回 wiki.detail 的 PagePayload），本 repository 只封
/// 纯 JSON 读取操作。
class WikiRepository {
  WikiRepository(this._client);

  final GfApiClient _client;

  static const _base = '/api/wiki';

  /// 公开层级树（namespace 分组的递归目录/页面树）。
  Future<WikiTreeResult> tree() => _client.get<WikiTreeResult>(
    '$_base/tree',
    parser: (json) =>
        WikiTreeResult.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 命名空间列表（按 sortOrder + name 排序）。
  Future<WikiNamespaceList> namespaces() => _client.get<WikiNamespaceList>(
    '$_base/namespaces',
    parser: (json) =>
        WikiNamespaceList.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  Future<WikiSearchResult> search(String query) =>
      _client.get<WikiSearchResult>(
        '$_base/search',
        queryParameters: {'q': query},
        parser: (json) =>
            WikiSearchResult.fromJson(Map<String, dynamic>.from(json as Map)),
      );

  /// 首页 feed（命名空间概览 + 最近更新）。
  Future<WikiHomeData> home() => _client.get<WikiHomeData>(
    '$_base/home',
    parser: (json) =>
        WikiHomeData.fromJson(Map<String, dynamic>.from(json as Map)),
  );
}

/// GET /api/wiki/tree 的 result。
class WikiTreeResult {
  const WikiTreeResult({required this.namespaces});

  final List<WikiTreeNamespace> namespaces;

  factory WikiTreeResult.fromJson(Map<String, dynamic> json) => WikiTreeResult(
    namespaces: (json['namespaces'] as List<dynamic>? ?? const [])
        .map(
          (e) =>
              WikiTreeNamespace.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );
}

/// GET /api/wiki/namespaces 的 result。
class WikiNamespaceList {
  const WikiNamespaceList({required this.namespaces});

  final List<WikiNamespace> namespaces;

  factory WikiNamespaceList.fromJson(Map<String, dynamic> json) =>
      WikiNamespaceList(
        namespaces: (json['namespaces'] as List<dynamic>? ?? const [])
            .map(
              (e) =>
                  WikiNamespace.fromJson(Map<String, dynamic>.from(e as Map)),
            )
            .toList(),
      );
}
