import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

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

/// Host-provided actionable failure state for shared media components.
///
/// [retry] drops the failed decoded entry and rebuilds the image. The host owns
/// the copy and actions, so ui_kit keeps no API or localization dependency.
typedef GfImageErrorBuilder =
    Widget Function(
      BuildContext context,
      Object error,
      VoidCallback retry,
      String url,
    );

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
    this.imageErrorBuilder,
    required super.child,
  });
  final Object identity;
  final GfMediaProviderFactory factory;

  /// Optional host fallback for failed images. A caller's own `errorBuilder`
  /// still wins; without this hook the previous silent failure is kept.
  final GfImageErrorBuilder? imageErrorBuilder;

  static GfImageErrorBuilder? imageErrorBuilderOf(BuildContext? context) =>
      context
          ?.dependOnInheritedWidgetOfExactType<GfMediaScope>()
          ?.imageErrorBuilder;

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
    if (isSvgDocument(bytes)) return _decodeVectorImage(bytes, decode);
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

  /// Rasterizes vector bytes through the bitmap codec contract so `Image`,
  /// loading/error builders and generation fencing stay unchanged.
  ///
  /// ponytail: rasterized once here, so a full-screen 4x zoom softens; switch to
  /// `SvgPicture` or pass per-surface sizes if lossless zoom is required.
  Future<ui.Codec> _decodeVectorImage(
    Uint8List bytes,
    ImageDecoderCallback decode,
  ) async {
    final PictureInfo info = await vg.loadPicture(GfLiveSvgLoader(bytes), null);
    try {
      _guard();
      final ui.Size target = _vectorRasterSize(info.size);
      final ui.Image image = await info.picture.toImage(
        target.width.toInt(),
        target.height.toInt(),
      );
      try {
        _guard();
        // Encode the rasterized pixels and hand them to the host decoder: the
        // descriptor/buffer ownership then stays exactly like every bitmap here
        // (a raw descriptor disposed after instantiateCodec yields a codec whose
        // first frame fails, so this path deliberately skips that API).
        final ByteData? encoded = await image.toByteData(
          format: ui.ImageByteFormat.png,
        );
        if (encoded == null) {
          throw StateError('Vector image produced no pixels');
        }
        final ui.ImmutableBuffer buffer = await ui.ImmutableBuffer.fromUint8List(
          encoded.buffer.asUint8List(encoded.offsetInBytes, encoded.lengthInBytes),
        );
        return await decode(buffer);
      } finally {
        image.dispose();
      }
    } finally {
      info.picture.dispose();
    }
  }

  /// Caller cache size when given, otherwise 2x the intrinsic size, uniformly
  /// scaled within [0.05, 4.0] and never above 2048x2048.
  ui.Size _vectorRasterSize(ui.Size intrinsic) {
    final double intrinsicWidth =
        intrinsic.width.isFinite && intrinsic.width > 0
        ? intrinsic.width
        : 256.0;
    final double intrinsicHeight =
        intrinsic.height.isFinite && intrinsic.height > 0
        ? intrinsic.height
        : 256.0;
    double scale = 2.0;
    final int? requestedWidth = width;
    final int? requestedHeight = height;
    if (requestedWidth != null || requestedHeight != null) {
      final double scaleX = requestedWidth == null
          ? double.infinity
          : requestedWidth / intrinsicWidth;
      final double scaleY = requestedHeight == null
          ? double.infinity
          : requestedHeight / intrinsicHeight;
      scale = math.min(scaleX, scaleY);
    }
    if (!scale.isFinite || scale <= 0) scale = 2.0;
    scale = scale.clamp(0.05, 4.0).toDouble();
    double rasterWidth = intrinsicWidth * scale;
    double rasterHeight = intrinsicHeight * scale;
    final double longest = math.max(rasterWidth, rasterHeight);
    if (longest > 2048) {
      final double limit = 2048 / longest;
      rasterWidth *= limit;
      rasterHeight *= limit;
    }
    return ui.Size(
      math.max(1, rasterWidth.roundToDouble()),
      math.max(1, rasterHeight.roundToDouble()),
    );
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
class GfNetworkImage extends StatefulWidget {
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
    this.onImageError,
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

  /// Called when loading fails, before the host fallback is built: a caller
  /// keeps its own bookkeeping (e.g. screenshot readiness) without owning the
  /// error UI. Ignored when [errorBuilder] is supplied.
  final void Function(Object error)? onImageError;
  final ImageLoadingBuilder? loadingBuilder;
  final ImageFrameBuilder? frameBuilder;
  final FilterQuality filterQuality;

  @override
  State<GfNetworkImage> createState() => _GfNetworkImageState();
}

class _GfNetworkImageState extends State<GfNetworkImage> {
  int _attempt = 0;
  ImageProvider<Object>? _provider;

  void _retry() {
    final provider = _provider;
    if (provider != null) {
      // Drop the failed decoded entry, then rebuild under a fresh key: Image
      // resolves again only when its provider instance changes.
      unawaited(provider.evict().catchError((Object _) => false));
    }
    setState(() => _attempt++);
  }

  @override
  Widget build(BuildContext context) {
    final GfImageErrorBuilder? host = GfMediaScope.imageErrorBuilderOf(context);
    final provider = GfMediaScope.imageProvider(
      context,
      widget.url,
      width: widget.cacheWidth,
      height: widget.cacheHeight,
      policy: widget.cacheResizePolicy,
    );
    _provider = provider;
    return Image(
      key: ValueKey<int>(_attempt),
      image: provider,
      width: widget.width,
      height: widget.height,
      fit: widget.fit,
      semanticLabel: widget.semanticLabel,
      excludeFromSemantics: widget.excludeFromSemantics,
      errorBuilder:
          widget.errorBuilder ??
          (host == null
              ? (widget.onImageError == null
                    ? null
                    : (_, error, _) {
                        widget.onImageError!(error);
                        return const SizedBox.shrink();
                      })
              : (errorContext, error, _) {
                  widget.onImageError?.call(error);
                  return host(errorContext, error, _retry, widget.url);
                }),
      loadingBuilder: widget.loadingBuilder,
      frameBuilder: widget.frameBuilder,
      filterQuality: widget.filterQuality,
      // No stale frames across account, language or clear-generation boundaries.
      gaplessPlayback: false,
    );
  }
}

/// True when the bytes begin with an SVG document (case-insensitive, after an
/// optional UTF-8 BOM and leading whitespace). Bitmap bytes never qualify.
bool isSvgDocument(Uint8List bytes) {
  var start = 0;
  if (bytes.length >= 3 &&
      bytes[0] == 0xEF &&
      bytes[1] == 0xBB &&
      bytes[2] == 0xBF) {
    start = 3;
  }
  while (start < bytes.length && bytes[start] <= 0x20) {
    start++;
  }
  final int end = math.min(bytes.length, start + 1024);
  final String head = latin1
      .decode(bytes.sublist(start, end))
      .toLowerCase();
  return head.startsWith('<svg') ||
      (head.startsWith('<?xml') && head.contains('<svg'));
}

/// Parses SVG bytes without leaving the parsed result in flutter_svg's global
/// cache: private/no-store media must not outlive the scope that fetched it.
class GfLiveSvgLoader extends SvgBytesLoader {
  const GfLiveSvgLoader(super.bytes);
  @override
  Future<ByteData> loadBytes(BuildContext? context) async {
    final SvgCacheKey key = cacheKey(context);
    try {
      return await super.loadBytes(context);
    } finally {
      svg.cache.evict(key);
    }
  }
}
