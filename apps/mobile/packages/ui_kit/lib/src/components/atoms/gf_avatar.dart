import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

/// Circular user avatar with the sizes used across the web app
/// (UserAvatar.vue): 24 (sm stack) / 32 (md stack) / 40 (rows, chat) /
/// 48 / 56 / 64 (profile, settings). Renders the image via [NetworkImage]
/// with a muted fallback while loading.
class GfAvatar extends StatelessWidget {
  const GfAvatar({
    super.key,
    required this.src,
    this.size = 40,
    this.ring = false,
    this.badge,
  });

  /// Image URL (may be empty; falls back to a placeholder).
  final String src;

  /// Edge length; one of the web size steps (24/32/40/48/56/64).
  final double size;

  /// Whether to draw a 2px base-100 ring around the avatar
  /// (web `ring-2 ring-base-100`, used in avatar stacks).
  final bool ring;

  /// Optional corner badge (e.g. online dot, badge icon).
  final Widget? badge;

  /// Image key shared by visible avatars and their startup prefetch.
  static ImageProvider<Object>? imageProviderFor(
    String src, {
    required double size,
    required double devicePixelRatio,
  }) {
    if (src.isEmpty) return null;
    final pixels = (size * devicePixelRatio).round();
    return ResizeImage(
      NetworkImage(src),
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
      size: size,
      devicePixelRatio: MediaQuery.devicePixelRatioOf(context),
    );

    final Widget avatar = Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: colors.base200,
        border: ring
            ? Border.all(color: colors.base100, width: 2 * borders.width)
            : null,
      ),
      clipBehavior: Clip.antiAlias,
      child: provider == null
          ? fallback
          : Image(
              image: provider,
              fit: BoxFit.cover,
              excludeFromSemantics: true,
              frameBuilder: (_, child, frame, wasSynchronouslyLoaded) =>
                  wasSynchronouslyLoaded || frame != null ? child : fallback,
              errorBuilder: (_, _, _) => fallback,
            ),
    );

    if (badge == null) return avatar;
    return Stack(
      clipBehavior: Clip.none,
      children: <Widget>[
        avatar,
        Positioned(right: 0, bottom: 0, child: badge!),
      ],
    );
  }
}
