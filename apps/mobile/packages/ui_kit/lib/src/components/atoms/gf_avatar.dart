import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';
import '../gf_network_svg.dart';
import '../gf_media_image.dart';

/// Circular user avatar matching the web app's avatar usage
/// (UserAvatar.vue). Any logical [size] is valid: decoding snaps up to a
/// shared ladder, so nearby display sizes reuse one decoded image (see
/// [_decodeSizes]). Renders the image via [GfMediaScope] with a muted
/// fallback while loading.
class GfAvatar extends StatelessWidget {
  const GfAvatar({
    super.key,
    required this.src,
    this.size = 40,
    this.ring = false,
    this.badge,
    this.frameColors = const [],
    this.frameAssetUrl = '',
    this.frameScale = 1.25,
  });

  /// Image URL (may be empty; falls back to a placeholder).
  final String src;
  final List<Color> frameColors;
  final String frameAssetUrl;
  final double frameScale;

  /// Edge length in logical pixels; decode size snaps up to the step that
  /// covers it, so every value is valid.
  final double size;

  /// Whether to draw a 2px base-100 ring around the avatar
  /// (web `ring-2 ring-base-100`, used in avatar stacks).
  final bool ring;

  /// Optional corner badge.
  final Widget? badge;

  /// Decode sizes avatars snap up to before the device pixel ratio is
  /// applied. Small inline avatars share the 24/40 steps (quote chips,
  /// stacks, mention, reply and conversation rows, chat bubbles, feed cards,
  /// detail headers); 48-96 covers connection rows, the account drawer, the
  /// settings avatar and user cards or profile editors. Keeping one entry per
  /// step means switching pages reuses the decoded picture instead of
  /// downloading and decoding it again per pixel size.
  static const List<double> _decodeSizes = <double>[24, 40, 48, 64, 96];

  /// Smallest decode step covering [size]; larger requests round up on a
  /// 16-pixel grid (100 to 112, 113 to 128) so an oversized avatar is never
  /// downscaled.
  static double _decodeSizeFor(double size) {
    for (final double candidate in _decodeSizes) {
      if (size <= candidate) return candidate;
    }
    return (size / 16).ceilToDouble() * 16;
  }

  /// Image key shared by visible avatars and their startup prefetch.
  ///
  /// A non-positive [size] snaps to the smallest decode step (24 logical
  /// pixels), so a caller cannot force a one-pixel decode.
  static ImageProvider<Object>? imageProviderFor(
    String src, {
    BuildContext? context,
    required double size,
    required double devicePixelRatio,
  }) {
    if (src.isEmpty) return null;
    final pixels = (_decodeSizeFor(size) * devicePixelRatio).round();
    return GfMediaScope.imageProvider(
      context,
      src,
      policy: ResizeImagePolicy.fit,
      width: pixels < 1 ? 1 : pixels,
      height: pixels < 1 ? 1 : pixels,
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final fallback = Center(
      child: GfSymbol('user-round', size: size * 0.56, color: colors.iconMuted),
    );
    final provider = imageProviderFor(
      src,
      context: context,
      size: size,
      devicePixelRatio: MediaQuery.devicePixelRatioOf(context),
    );
    final ringWidth = ring ? 2 * borders.width : 0.0;
    final Widget image = Uri.tryParse(src)?.path.endsWith('/avatar.svg') == true
        ? SizedBox(
            width: size,
            height: size,
            child: GfNetworkSvg(src, fallback: fallback),
          )
        : provider == null
        ? fallback
        : Image(
            image: provider,
            fit: BoxFit.cover,
            excludeFromSemantics: true,
            frameBuilder: (_, child, frame, wasSynchronouslyLoaded) =>
                wasSynchronouslyLoaded || frame != null ? child : fallback,
            errorBuilder: (_, _, _) => fallback,
          );

    final Widget avatar = Container(
      width: size,
      height: size,
      decoration: BoxDecoration(shape: BoxShape.circle, color: colors.base200),
      // Reserve the ring outside the image: overlaying it on a full-size
      // portrait hides its edges and makes small chat avatars look zoomed in.
      // The inset image needs its own circle, otherwise clipping the smaller
      // square only by the outer circle leaves flat sides (issue #877).
      foregroundDecoration: ring
          ? BoxDecoration(
              shape: BoxShape.circle,
              border: Border.all(color: colors.base100, width: ringWidth),
            )
          : null,
      clipBehavior: Clip.antiAlias,
      child: ring
          ? Padding(
              padding: EdgeInsets.all(ringWidth),
              child: ClipOval(child: image),
            )
          : image,
    );

    final framed = GfAvatarFrame(
      size: size,
      colors: frameColors,
      assetUrl: frameAssetUrl,
      assetScale: frameScale,
      child: avatar,
    );
    if (badge == null) return framed;
    return Stack(
      clipBehavior: Clip.none,
      children: <Widget>[
        framed,
        Positioned(right: -size * .06, bottom: -size * .06, child: badge!),
      ],
    );
  }
}

/// Static frame artwork paints beyond the avatar without changing its layout.
class GfAvatarFrame extends StatelessWidget {
  const GfAvatarFrame({
    super.key,
    required this.size,
    required this.child,
    this.colors = const [],
    this.assetUrl = '',
    this.assetScale = 1.25,
  });
  final double size;
  final Widget child;
  final List<Color> colors;
  final String assetUrl;
  final double assetScale;
  @override
  Widget build(BuildContext context) {
    final uri = Uri.tryParse(assetUrl);
    final hasFrameAsset =
        uri != null &&
        RegExp(
          r'^/static/decorations/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(png|webp|gif|svg)$',
        ).hasMatch(uri.path) &&
        !uri.hasQuery &&
        !uri.hasFragment &&
        uri.userInfo.isEmpty &&
        (!uri.hasScheme || uri.scheme == 'https' || uri.scheme == 'http');
    if (!hasFrameAsset) {
      if (colors.length < 2) return child;
      // Web `.gf-avatar-frame` 是 3px 环 + `inset: 7.5%`，即环画在头像盒子之外、
      // 头像尺寸不变，且环厚随头像直径等比缩放（40px 头像 = 3px，20px = 1.5px，
      // 96px = 7.2px）。固定 6px 外扩会让列表小头像上的环显得过粗、大头像上过细。
      final double grow = size * 0.075;
      return SizedBox(
        width: size,
        height: size,
        child: Stack(
          clipBehavior: Clip.none,
          children: <Widget>[
            Positioned(
              left: -grow,
              top: -grow,
              width: size + grow * 2,
              height: size + grow * 2,
              child: Container(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  // CSS `linear-gradient(120deg, …)` runs toward
                  // (sin 120°, -cos 120°) = (0.866, 0.5) — right and down — over
                  // the corner-to-corner span (0.866 + 0.5) of a square box:
                  // ±0.866*1.366 / ±0.5*1.366 in Flutter's -1..1 alignment.
                  gradient: LinearGradient(
                    begin: const Alignment(-1.183, -0.683),
                    end: const Alignment(1.183, 0.683),
                    colors: colors,
                  ),
                ),
              ),
            ),
            child,
          ],
        ),
      );
    }
    final scale = assetScale.isFinite && assetScale >= 0.5 && assetScale <= 2
        ? assetScale
        : 1.25;
    final edge = size * scale;
    final pixels = (edge * MediaQuery.devicePixelRatioOf(context)).ceil();
    // A safe frame asset owns the frame: nothing gradient is built underneath,
    // so loading, decode failure and load failure can never leak a colour ring.
    return SizedBox(
      width: size,
      height: size,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          child,
          Positioned(
            left: -(edge - size) / 2,
            top: -(edge - size) / 2,
            width: edge,
            height: edge,
            child: IgnorePointer(
              child: uri.path.endsWith('.svg')
                  ? GfNetworkSvg(assetUrl, fallback: const SizedBox.shrink())
                  : Image(
                      image: GfMediaScope.imageProvider(
                        context,
                        assetUrl,
                        width: pixels,
                        height: pixels,
                        policy: ResizeImagePolicy.fit,
                      ),
                      fit: BoxFit.contain,
                      excludeFromSemantics: true,
                      frameBuilder: (_, child, frame, loaded) =>
                          loaded || frame != null
                          ? child
                          : const SizedBox.shrink(),
                      errorBuilder: (_, _, _) => const SizedBox.shrink(),
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
