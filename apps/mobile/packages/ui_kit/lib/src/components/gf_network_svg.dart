import 'dart:async';
import 'dart:collection';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import 'gf_media_image.dart';

/// Remote badges, avatar frames and anonymous SVG avatars share the media byte
/// owner. Cacheable public SVGs reuse a bounded, scope-cleared memory cache so
/// scrolling does not re-parse XML on every mount; private/unretained SVGs keep
/// using [GfLiveSvgLoader] without entering [svg.cache].
class GfNetworkSvg extends StatefulWidget {
  const GfNetworkSvg(this.url, {super.key, required this.fallback});
  final String url;
  final Widget fallback;

  static const int _maxEntries = 64;
  static final LinkedHashMap<Object, Uint8List> _cachedBytes =
      LinkedHashMap<Object, Uint8List>();
  static final LinkedHashMap<SvgCacheKey, Object> _trackedSvgKeys =
      LinkedHashMap<SvgCacheKey, Object>();

  /// Evicts every cached network SVG byte entry and its parsed vector entry
  /// from [svg.cache]. Called by the media host on invalidation, cache clear,
  /// session boundary and memory pressure.
  static void clearCache() {
    _cachedBytes.clear();
    for (final key in _trackedSvgKeys.keys.toList()) {
      svg.cache.evict(key);
    }
    _trackedSvgKeys.clear();
  }

  static Uint8List? _readCached(Object cacheKey) {
    final hit = _cachedBytes.remove(cacheKey);
    if (hit != null) {
      _cachedBytes[cacheKey] = hit;
    }
    return hit;
  }

  static void _storeCached(Object cacheKey, Uint8List bytes) {
    _cachedBytes.remove(cacheKey);
    if (_cachedBytes.length >= _maxEntries) {
      _cachedBytes.remove(_cachedBytes.keys.first);
    }
    _cachedBytes[cacheKey] = bytes;
  }

  static void _trackParsedKey(SvgCacheKey key) {
    _trackedSvgKeys.remove(key);
    if (_trackedSvgKeys.length >= _maxEntries) {
      final oldest = _trackedSvgKeys.keys.first;
      _trackedSvgKeys.remove(oldest);
      svg.cache.evict(oldest);
    }
    _trackedSvgKeys[key] = key;
  }

  @override
  State<GfNetworkSvg> createState() => _GfNetworkSvgState();
}

class _GfTrackedSvgLoader extends SvgBytesLoader {
  const _GfTrackedSvgLoader(super.bytes);

  @override
  Future<ByteData> loadBytes(BuildContext? context) async {
    final SvgCacheKey key = cacheKey(context);
    GfNetworkSvg._trackParsedKey(key);
    return super.loadBytes(context);
  }
}

class _GfNetworkSvgState extends State<GfNetworkSvg> {
  GfBytesImage? _provider;
  Future<GfMediaData>? _bytes;

  Object? _memoryKeyFor(GfBytesImage provider, Object? mediaIdentity) =>
      mediaIdentity == null ? null : (provider.identity, provider.url, mediaIdentity);

  @override
  Widget build(BuildContext context) {
    final provider = GfMediaScope.imageProvider(context, widget.url);
    if (provider is! GfBytesImage) {
      // Standalone UI-kit previews have no application storage owner.
      return SvgPicture.network(
        widget.url,
        fit: BoxFit.contain,
        placeholderBuilder: (_) => widget.fallback,
        errorBuilder: (_, _, _) => widget.fallback,
      );
    }
    final cachedMediaIdentity = provider.isCurrent()
        ? provider.cachedIdentity?.call()
        : null;
    final syncKey = _memoryKeyFor(provider, cachedMediaIdentity);
    final syncBytes = syncKey == null ? null : GfNetworkSvg._readCached(syncKey);
    if (syncBytes != null) {
      return _buildSvg(provider, syncBytes, retain: true);
    }
    if (provider != _provider) {
      _provider = provider;
      _bytes = provider.load().then((data) {
        if (!provider.isCurrent()) throw StateError('Media SVG invalidated');
        final key = _memoryKeyFor(provider, data.cacheIdentity);
        if (key != null) {
          GfNetworkSvg._storeCached(key, data.bytes);
        }
        return data;
      });
    }
    return FutureBuilder<GfMediaData>(
      future: _bytes,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done ||
            !snapshot.hasData ||
            !provider.isCurrent()) {
          return widget.fallback;
        }
        final data = snapshot.data!;
        return _buildSvg(
          provider,
          data.bytes,
          retain: data.cacheIdentity != null,
        );
      },
    );
  }

  Widget _buildSvg(
    GfBytesImage provider,
    Uint8List bytes, {
    required bool retain,
  }) => SvgPicture(
    retain ? _GfTrackedSvgLoader(bytes) : GfLiveSvgLoader(bytes),
    key: ValueKey((provider, retain)),
    fit: BoxFit.contain,
    placeholderBuilder: (_) => widget.fallback,
    errorBuilder: (_, _, _) {
      final discard = provider.onDecodeError;
      if (discard != null) unawaited(discard().catchError((Object _) {}));
      return widget.fallback;
    },
  );
}
