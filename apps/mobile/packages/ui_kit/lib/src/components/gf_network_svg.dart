import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import 'gf_media_image.dart';

/// Remote badges share the media byte owner; local SVG icon assets keep their
/// existing parser cache. Network SVG parsing is kept only by the live picture.
class GfNetworkSvg extends StatefulWidget {
  const GfNetworkSvg(this.url, {super.key, required this.fallback});
  final String url;
  final Widget fallback;
  @override
  State<GfNetworkSvg> createState() => _GfNetworkSvgState();
}

class _GfNetworkSvgState extends State<GfNetworkSvg> {
  GfBytesImage? _provider;
  Future<GfMediaData>? _bytes;

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
    if (provider != _provider) {
      _provider = provider;
      _bytes = provider.load().then((data) {
        if (!provider.isCurrent()) throw StateError('Media SVG invalidated');
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
        return SvgPicture(
          _LiveSvgLoader(snapshot.data!.bytes),
          key: ValueKey(provider),
          fit: BoxFit.contain,
          placeholderBuilder: (_) => widget.fallback,
          errorBuilder: (_, _, _) {
            final discard = provider.onDecodeError;
            if (discard != null) unawaited(discard().catchError((Object _) {}));
            return widget.fallback;
          },
        );
      },
    );
  }
}

class _LiveSvgLoader extends SvgBytesLoader {
  const _LiveSvgLoader(super.bytes);
  @override
  Future<ByteData> loadBytes(BuildContext? context) async {
    final key = cacheKey(context);
    try {
      return await super.loadBytes(context);
    } finally {
      // Never leave a private/no-store parsed response in flutter_svg's global
      // cache, nor let a late parse refill that cache after clear/account switch.
      svg.cache.evict(key);
    }
  }
}
