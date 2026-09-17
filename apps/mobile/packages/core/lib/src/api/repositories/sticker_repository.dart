import '../../gen/sticker.dart';
import '../gf_api_client.dart';

/// 表情包公开只读域（`GET /api/forum/stickers`，启用列表，sortOrder 排序）。
///
/// url 为后端相对路径（如 `/file/img/stickers/xxx.png`），展示前经上层
/// resolveApiAssetUrl 转绝对 URL。
class StickerRepository {
  StickerRepository(this._client);

  final GfApiClient _client;

  /// 公开启用表情包列表（token 展开/分段渲染的数据源）。
  Future<List<StickerItemPayload>> list() =>
      _client.get<List<StickerItemPayload>>(
        '/api/forum/stickers',
        parser: (json) => (json as List<dynamic>? ?? const [])
            .map(
              (e) => StickerItemPayload.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(growable: false),
      );
}

/// 表情包库会话级缓存：一次 app 运行拉取一次，失败不缓存（下次重试）。
class StickerLibrary {
  StickerLibrary(this._repository);

  final StickerRepository _repository;

  List<StickerItemPayload>? _items;
  Future<List<StickerItemPayload>>? _pending;

  /// 库是否已成功加载过。
  bool get isLoaded => _items != null;

  /// 返回缓存列表；未加载时发起拉取（并发调用共享同一在途请求）。
  Future<List<StickerItemPayload>> load() {
    final List<StickerItemPayload>? cached = _items;
    if (cached != null) return Future<List<StickerItemPayload>>.value(cached);
    return _pending ??= _loadAndCache();
  }

  Future<List<StickerItemPayload>> _loadAndCache() async {
    try {
      final List<StickerItemPayload> items = await _repository.list();
      _items = items;
      return items;
    } on Object {
      _pending = null; // 失败不缓存,下次 load 重试。
      rethrow;
    }
  }

  /// token → url 映射；未加载时返回空映射（渲染端把 token 保持原文）。
  Map<String, String> get urlByName {
    final List<StickerItemPayload>? items = _items;
    if (items == null) return const <String, String>{};
    return <String, String>{
      for (final StickerItemPayload item in items)
        if (item.name.isNotEmpty && item.url.isNotEmpty) item.name: item.url,
    };
  }
}
