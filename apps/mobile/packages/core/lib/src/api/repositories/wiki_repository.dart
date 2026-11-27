import 'package:dio/dio.dart';

import '../../gen/wiki.dart';
import '../../gen/wiki_search.dart';
import '../gf_api_client.dart';

/// Wiki 只读域（`/api/wiki/tree|namespaces|home|search`，公开只读）。
///
/// 页面正文走页面级数据通道（`PageRepository.fetch('/wiki/...')`，
/// X-Goose-Page 返回 wiki.detail 的 PagePayload），本 repository 只封
/// 纯 JSON 读取操作。
class WikiRepository {
  WikiRepository(
    this._client, {
    this.cacheTtl = Duration.zero,
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now;

  final GfApiClient _client;
  final Duration cacheTtl;
  final DateTime Function() _now;

  static const _base = '/api/wiki';

  ({WikiTreeResult data, DateTime fetchedAt})? _treeCache;
  Future<WikiTreeResult>? _treePending;
  ({WikiNamespaceList data, DateTime fetchedAt})? _namespacesCache;
  Future<WikiNamespaceList>? _namespacesPending;
  ({WikiHomeData data, DateTime fetchedAt})? _homeCache;
  Future<WikiHomeData>? _homePending;

  void clearCache() {
    _treeCache = null;
    _namespacesCache = null;
    _homeCache = null;
  }

  bool _isFresh(DateTime fetchedAt) =>
      cacheTtl > Duration.zero && _now().difference(fetchedAt) < cacheTtl;

  /// 公开层级树（namespace 分组的递归目录/页面树）。
  Future<WikiTreeResult> tree() {
    if (_treeCache != null && _isFresh(_treeCache!.fetchedAt)) {
      return Future.value(_treeCache!.data);
    }
    return _treePending ??= _client
        .get<WikiTreeResult>(
          '$_base/tree',
          parser: (json) =>
              WikiTreeResult.fromJson(Map<String, dynamic>.from(json as Map)),
        )
        .then((result) {
          if (cacheTtl > Duration.zero) {
            _treeCache = (data: result, fetchedAt: _now());
          }
          return result;
        })
        .whenComplete(() => _treePending = null);
  }

  /// 命名空间列表（按 sortOrder + name 排序）。
  Future<WikiNamespaceList> namespaces() {
    if (_namespacesCache != null && _isFresh(_namespacesCache!.fetchedAt)) {
      return Future.value(_namespacesCache!.data);
    }
    return _namespacesPending ??= _client
        .get<WikiNamespaceList>(
          '$_base/namespaces',
          parser: (json) => WikiNamespaceList.fromJson(
            Map<String, dynamic>.from(json as Map),
          ),
        )
        .then((result) {
          if (cacheTtl > Duration.zero) {
            _namespacesCache = (data: result, fetchedAt: _now());
          }
          return result;
        })
        .whenComplete(() => _namespacesPending = null);
  }

  /// 页面级搜索；[limit] 为页面数上限（服务端 1–20，缺省 12）。
  Future<WikiSearchResult> search(
    String query, {
    int? limit,
    CancelToken? cancelToken,
  }) => _client.get<WikiSearchResult>(
    '$_base/search',
    cancelToken: cancelToken,
    queryParameters: {'q': query, 'limit': ?limit},
    parser: (json) =>
        WikiSearchResult.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 首页 feed（命名空间概览 + 最近更新）。
  Future<WikiHomeData> home() {
    if (_homeCache != null && _isFresh(_homeCache!.fetchedAt)) {
      return Future.value(_homeCache!.data);
    }
    return _homePending ??= _client
        .get<WikiHomeData>(
          '$_base/home',
          parser: (json) =>
              WikiHomeData.fromJson(Map<String, dynamic>.from(json as Map)),
        )
        .then((result) {
          if (cacheTtl > Duration.zero) {
            _homeCache = (data: result, fetchedAt: _now());
          }
          return result;
        })
        .whenComplete(() => _homePending = null);
  }
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
