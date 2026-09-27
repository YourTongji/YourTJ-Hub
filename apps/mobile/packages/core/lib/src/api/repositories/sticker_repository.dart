import 'package:flutter/foundation.dart';

import '../../gen/sticker.dart';
import '../../markdown/sticker_token.dart';
import '../api_error.dart';
import '../gf_api_client.dart';

List<StickerItemPayload> _items(Object? json) => (json as List? ?? const [])
    .map(
      (e) => StickerItemPayload.fromJson(Map<String, dynamic>.from(e as Map)),
    )
    .toList(growable: false);

/// Official assets, stable content resolution, and the signed-in user's library.
class StickerRepository {
  StickerRepository(this._client);
  final GfApiClient _client;

  Future<List<StickerItemPayload>> list() =>
      _client.get('/api/forum/stickers', parser: _items);

  Future<List<StickerItemPayload>> resolve(List<String> names) => _client.post(
    '/api/forum/stickers/resolve',
    body: {'names': names},
    parser: _items,
  );

  Future<List<StickerItemPayload>> mine() =>
      _client.get('/api/forum/my-stickers', parser: _items);

  Future<StickerItemPayload> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) => _client.post(
    '/api/forum/my-sticker-save',
    body: {
      'stickerName': ?stickerName,
      'fileName': ?fileName,
      'displayName': ?displayName,
    },
    parser: (json) =>
        StickerItemPayload.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  Future<void> remove(String name) => _client.post(
    '/api/forum/my-sticker-delete',
    body: {'name': name},
    parser: (_) {},
  );

  Future<void> reorder(List<String> names) => _client.post(
    '/api/forum/my-stickers-order',
    body: {'names': names},
    parser: (_) {},
  );
}

/// Cache belongs to a site/session provider; content references outlive membership.
/// Official lists expire, failed loads remain retryable, and personal assets are
/// resolved by stable token rather than exposed in the public library.
class StickerLibrary extends ChangeNotifier {
  StickerLibrary(this._repository);
  final StickerRepository _repository;
  List<StickerItemPayload>? _items;
  final Map<String, StickerItemPayload> _resolved = {};
  final Set<String> _queuedNames = {};
  Future<void>? _pendingResolution;
  Future<List<StickerItemPayload>>? _pending;
  DateTime? _loadedAt;
  bool _disposed = false;
  bool get isLoaded => _items != null;
  List<StickerItemPayload> get items => List.unmodifiable(_resolved.values);

  Future<List<StickerItemPayload>> load({bool refresh = false}) {
    if (_disposed) return Future.value(const []);
    if (!refresh &&
        _items != null &&
        DateTime.now().difference(_loadedAt!) < const Duration(minutes: 5)) {
      return Future.value(_items!);
    }
    return _pending ??= _loadAndCache();
  }

  Future<List<StickerItemPayload>> _loadAndCache() async {
    try {
      final items = await _repository.list();
      if (_disposed) return const [];
      final currentNames = items
          .where((item) => item.isEnabled)
          .map((item) => item.name)
          .toSet();
      for (final old in _items ?? <StickerItemPayload>[]) {
        if (!currentNames.contains(old.name)) _resolved.remove(old.name);
      }
      _items = items;
      _loadedAt = DateTime.now();
      remember(items);
      return items;
    } finally {
      _pending = null;
    }
  }

  void remember(Iterable<StickerItemPayload> items) {
    if (_disposed) return;
    for (final item in items) {
      if (!item.isEnabled) {
        _resolved.remove(item.name);
        continue;
      }
      if (item.name.isNotEmpty && item.url.isNotEmpty) {
        _resolved[item.name] = item;
      }
    }
    notifyListeners();
  }

  Future<void> resolveContent(String content) async {
    if (_disposed) return;
    final names =
        stickerTokenPattern
            .allMatches(content)
            .map((m) => m.group(1)!)
            .toSet()
            .toList()
          ..sort();
    if (names.isEmpty) return;
    await load();
    if (_disposed) return;
    final missing = names
        .where((name) => !_resolved.containsKey(name))
        .toList();
    if (missing.isEmpty) return;
    _queuedNames.addAll(missing);
    // Visible chat bubbles resolve together, rather than one HTTP request per
    // message consuming the site's rate limit. New arrivals join the same flush.
    await (_pendingResolution ??= Future<void>.microtask(_flushResolutions));
  }

  Future<void> _flushResolutions() async {
    try {
      while (!_disposed && _queuedNames.isNotEmpty) {
        final names =
            _queuedNames.where((name) => !_resolved.containsKey(name)).toList()
              ..sort();
        _queuedNames.clear();
        await _resolve(names);
      }
    } finally {
      // Every waiter receives the error; a user retry starts a fresh batch.
      _queuedNames.clear();
      _pendingResolution = null;
    }
  }

  Future<void> _resolve(List<String> names) async {
    try {
      // Bound reads for long threads to the server's resolver batch size.
      for (var i = 0; i < names.length; i += 100) {
        if (_disposed) return;
        remember(
          await _repository.resolve(
            names.sublist(i, (i + 100).clamp(0, names.length)),
          ),
        );
      }
    } on ApiException catch (error) {
      // Older servers still render official tokens. Personal-library UI reports
      // unsupported endpoints explicitly rather than fabricating empty success.
      if (error.statusCode != 404 && error.statusCode != 405) rethrow;
    }
  }

  Map<String, String> get urlByName => {
    for (final item in _resolved.values) item.name: item.url,
  };

  @override
  void dispose() {
    _disposed = true;
    _queuedNames.clear();
    _resolved.clear();
    _items = null;
    super.dispose();
  }
}
