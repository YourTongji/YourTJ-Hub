import 'dart:async';

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
          GfLiveSvgLoader(snapshot.data!.bytes),
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
