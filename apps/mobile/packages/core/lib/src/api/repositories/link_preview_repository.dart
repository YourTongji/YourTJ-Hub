import 'dart:collection';

import '../../gen/link_preview.dart';
import '../../markdown/link_preview_candidate.dart';
import '../gf_api_client.dart';

class LinkPreviewRepository {
  LinkPreviewRepository(
    this._client, {
    this.cacheTtl = const Duration(minutes: 10),
    this.maxCacheEntries = 200,
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now;

  final GfApiClient _client;
  final Duration cacheTtl;
  final int maxCacheEntries;
  final DateTime Function() _now;
  final LinkedHashMap<
    String,
    ({LinkPreviewPayload payload, DateTime fetchedAt})
  >
  _cache =
      LinkedHashMap<
        String,
        ({LinkPreviewPayload payload, DateTime fetchedAt})
      >();

  void clearCache() => _cache.clear();

  LinkPreviewPayload? _readFresh(String url) {
    if (cacheTtl <= Duration.zero) return null;
    final entry = _cache.remove(url);
    if (entry == null) return null;
    if (_now().difference(entry.fetchedAt) >= cacheTtl) {
      return null;
    }
    _cache[url] = entry;
    return entry.payload;
  }

  void _store(String url, LinkPreviewPayload payload) {
    if (cacheTtl <= Duration.zero || url.isEmpty) return;
    _cache.remove(url);
    if (_cache.length >= maxCacheEntries) {
      _cache.remove(_cache.keys.first);
    }
    _cache[url] = (payload: payload, fetchedAt: _now());
  }

  Future<List<LinkPreviewPayload>> resolve(List<String> urls) async {
    final requested = urls
        .take(maxLinkPreviewCandidates)
        .toList(growable: false);
    if (requested.isEmpty) return const <LinkPreviewPayload>[];
    final hits = <String, LinkPreviewPayload>{};
    final missing = <String>[];
    final seenMissing = <String>{};
    for (final url in requested) {
      final cached = _readFresh(url);
      if (cached != null) {
        hits[url] = cached;
      } else if (seenMissing.add(url)) {
        missing.add(url);
      }
    }
    if (missing.isEmpty) {
      return [for (final url in requested) hits[url]!];
    }
    final fetched = await _client.post<List<LinkPreviewPayload>>(
      '/api/link-previews/resolve',
      body: <String, Object>{'urls': missing},
      parser: (Object? json) => (json as List<dynamic>? ?? const <dynamic>[])
          .map(
            (dynamic item) => LinkPreviewPayload.fromJson(
              Map<String, dynamic>.from(item as Map<dynamic, dynamic>),
            ),
          )
          .toList(growable: false),
    );
    for (final item in fetched) {
      if (item.requestedUrl.isNotEmpty) {
        _store(item.requestedUrl, item);
        hits[item.requestedUrl] = item;
      }
      final resolvedUrl = item.url;
      if (resolvedUrl != null &&
          resolvedUrl.isNotEmpty &&
          resolvedUrl != item.requestedUrl) {
        _store(resolvedUrl, item);
        hits.putIfAbsent(resolvedUrl, () => item);
      }
    }
    if (hits.length == fetched.length && missing.length == requested.length) {
      return fetched;
    }
    final ordered = <LinkPreviewPayload>[];
    final emitted = <LinkPreviewPayload>{};
    for (final url in requested) {
      final item = hits[url];
      if (item != null && emitted.add(item)) {
        ordered.add(item);
      }
    }
    for (final item in fetched) {
      if (emitted.add(item)) {
        ordered.add(item);
      }
    }
    return ordered;
  }
}
