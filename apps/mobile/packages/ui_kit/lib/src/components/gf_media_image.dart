import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

/// The host owns HTTP policy and bytes. Null [cacheIdentity] means that decoded
/// frames may be displayed by the current listener but must not enter ImageCache.
class GfMediaData {
  const GfMediaData(this.bytes, {this.cacheIdentity});
  final Uint8List bytes;
  final Object? cacheIdentity;
}

typedef GfMediaProviderFactory =
    ImageProvider<Object> Function(
      String url, {
      int? width,
      int? height,
      ResizeImagePolicy policy,
      Set<String>? allowedOrigins,
    });

/// Optional request policy for untrusted user-supplied media, including redirects.
class GfMediaOriginPolicy extends InheritedWidget {
  const GfMediaOriginPolicy({
    super.key,
    required this.origins,
    required super.child,
  });
  final Set<String> origins;

  @override
  bool updateShouldNotify(GfMediaOriginPolicy oldWidget) =>
      !setEquals(origins, oldWidget.origins);
}

/// Inverts the dependency: reusable UI components know no forum/storage APIs.
class GfMediaScope extends InheritedWidget {
  const GfMediaScope({
    super.key,
    required this.identity,
    required this.factory,
    required super.child,
  });
  final Object identity;
  final GfMediaProviderFactory factory;

  static ImageProvider<Object> imageProvider(
    BuildContext? context,
    String url, {
    int? width,
    int? height,
    ResizeImagePolicy policy = ResizeImagePolicy.exact,
  }) {
    final scope = context?.dependOnInheritedWidgetOfExactType<GfMediaScope>();
    if (scope != null) {
      return scope.factory(
        url,
        width: width,
        height: height,
        policy: policy,
        allowedOrigins: context
            ?.dependOnInheritedWidgetOfExactType<GfMediaOriginPolicy>()
            ?.origins,
      );
    }
    // Standalone design-system previews/tests have no application host.
    final image = NetworkImage(url);
    return width == null && height == null
        ? image
        : ResizeImage(image, width: width, height: height, policy: policy);
  }

  @override
  bool updateShouldNotify(GfMediaScope oldWidget) =>
      identity != oldWidget.identity;
}

/// Fetches/revalidates before consulting Flutter's decoded cache. Consequently a
/// fresh decoded frame cannot bypass no-cache or reuse bytes past HTTP expiry.
class GfBytesImage extends ImageProvider<GfMediaImageKey> {
  const GfBytesImage({
    required this.identity,
    required this.url,
    required this.load,
    required this.isCurrent,
    this.onDecodeError,
    this.width,
    this.height,
    this.policy = ResizeImagePolicy.exact,
  });
  final Object identity;
  final String url;
  final Future<GfMediaData> Function() load;
  final bool Function() isCurrent;
  final Future<void> Function()? onDecodeError;
  final int? width;
  final int? height;
  final ResizeImagePolicy policy;

  void _guard() {
    if (!isCurrent()) throw StateError('Media image invalidated');
  }

  @override
  Future<GfMediaImageKey> obtainKey(ImageConfiguration configuration) async {
    if (!isCurrent()) return GfMediaImageKey(Object(), null, false);
    try {
      final data = await load();
      if (!isCurrent()) return GfMediaImageKey(Object(), null, false);
      return GfMediaImageKey(
        (identity, url, data.cacheIdentity ?? Object(), width, height, policy),
        data.bytes,
        data.cacheIdentity != null,
      );
    } catch (_) {
      if (!isCurrent()) return GfMediaImageKey(Object(), null, false);
      rethrow;
    }
  }

  @override
  void resolveStreamForKey(
    ImageConfiguration configuration,
    ImageStream stream,
    GfMediaImageKey key,
    ImageErrorListener handleError,
  ) {
    if (!isCurrent()) {
      key.bytes = null;
      stream.setCompleter(_CancelledImageCompleter());
      return;
    }
    if (key.retain) {
      super.resolveStreamForKey(configuration, stream, key, handleError);
    } else if (stream.completer == null) {
      stream.setCompleter(
        loadImage(key, PaintingBinding.instance.instantiateImageCodecWithSize),
      );
    }
    key.bytes = null; // Never retain compressed bytes as a decoded-cache key.
  }

  @override
  ImageStreamCompleter loadImage(
    GfMediaImageKey key,
    ImageDecoderCallback decode,
  ) {
    final bytes = key.bytes!;
    return _MediaCompleter(
      isCurrent: isCurrent,
      codec: _decode(bytes, decode).catchError((
        Object error,
        StackTrace stack,
      ) async {
        if (isCurrent() && onDecodeError != null) {
          try {
            await onDecodeError!();
          } catch (_) {
            /* Preserve the original decode error. */
          }
        }
        Error.throwWithStackTrace(error, stack);
      }),
      scale: 1,
      debugLabel: url,
    );
  }

  Future<ui.Codec> _decode(Uint8List bytes, ImageDecoderCallback decode) async {
    _guard();
    final buffer = await ui.ImmutableBuffer.fromUint8List(bytes);
    if (!isCurrent()) {
      buffer.dispose();
      _guard();
    }
    final codec = await decode(
      buffer,
      getTargetSize: (w, h) {
        if (policy == ResizeImagePolicy.fit) {
          final ratio = math.min(
            1.0,
            math.min((width ?? w) / w, (height ?? h) / h),
          );
          return ui.TargetImageSize(
            width: math.max(1, (w * ratio).floor()),
            height: math.max(1, (h * ratio).floor()),
          );
        }
        return ui.TargetImageSize(
          width: width == null ? null : math.min(w, width!),
          height: height == null ? null : math.min(h, height!),
        );
      },
    );
    if (!isCurrent()) {
      codec.dispose();
      _guard();
    }
    return codec;
  }

  @override
  bool operator ==(Object other) =>
      other is GfBytesImage &&
      identity == other.identity &&
      url == other.url &&
      width == other.width &&
      height == other.height &&
      policy == other.policy;
  @override
  int get hashCode => Object.hash(identity, url, width, height, policy);
}

// Cancellation after a scope change is expected and has no pixels to emit.
// Preserve regular image errors while avoiding detached-listener error reports.
class _CancelledImageCompleter extends ImageStreamCompleter {}

class _MediaCompleter extends MultiFrameImageStreamCompleter {
  _MediaCompleter({
    required this.isCurrent,
    required super.codec,
    required super.scale,
    super.debugLabel,
  });
  final bool Function() isCurrent;
  @override
  void reportError({
    DiagnosticsNode? context,
    required Object exception,
    StackTrace? stack,
    InformationCollector? informationCollector,
    bool silent = false,
  }) {
    if (!isCurrent()) return;
    super.reportError(
      context: context,
      exception: exception,
      stack: stack,
      informationCollector: informationCollector,
      silent: silent,
    );
  }
}

class GfMediaImageKey {
  GfMediaImageKey(this.identity, this.bytes, this.retain);
  final Object identity;
  // Cleared immediately after the completer takes ownership of the bytes.
  Uint8List? bytes;
  final bool retain;
  @override
  bool operator ==(Object other) =>
      other is GfMediaImageKey && identity == other.identity;
  @override
  int get hashCode => identity.hashCode;
}

/// Image.network's visual API, routed through the host's one media repository.
class GfNetworkImage extends StatelessWidget {
  const GfNetworkImage(
    this.url, {
    super.key,
    this.width,
    this.height,
    this.fit,
    this.cacheWidth,
    this.cacheHeight,
    this.cacheResizePolicy = ResizeImagePolicy.exact,
    this.semanticLabel,
    this.excludeFromSemantics = false,
    this.errorBuilder,
    this.loadingBuilder,
    this.frameBuilder,
    this.filterQuality = FilterQuality.medium,
  });
  final String url;
  final double? width;
  final double? height;
  final BoxFit? fit;
  final int? cacheWidth;
  final int? cacheHeight;
  final ResizeImagePolicy cacheResizePolicy;
  final String? semanticLabel;
  final bool excludeFromSemantics;
  final ImageErrorWidgetBuilder? errorBuilder;
  final ImageLoadingBuilder? loadingBuilder;
  final ImageFrameBuilder? frameBuilder;
  final FilterQuality filterQuality;

  @override
  Widget build(BuildContext context) => Image(
    image: GfMediaScope.imageProvider(
      context,
      url,
      width: cacheWidth,
      height: cacheHeight,
      policy: cacheResizePolicy,
    ),
    width: width,
    height: height,
    fit: fit,
    semanticLabel: semanticLabel,
    excludeFromSemantics: excludeFromSemantics,
    errorBuilder: errorBuilder,
    loadingBuilder: loadingBuilder,
    frameBuilder: frameBuilder,
    filterQuality: filterQuality,
    // No stale frames across account, language or clear-generation boundaries.
    gaplessPlayback: false,
  );
}
