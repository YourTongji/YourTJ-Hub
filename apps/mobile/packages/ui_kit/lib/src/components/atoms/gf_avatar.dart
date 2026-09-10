import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

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

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final ThemeData theme = Theme.of(context);
    final List<ThemeExtension<dynamic>> extensions = theme.extensions.values
        .toList(growable: true);
    extensions
      ..removeWhere(
        (ThemeExtension<dynamic> item) => item is td.TAvatarThemeData,
      )
      ..add(
        td.TAvatarThemeData(
          dimension: size,
          iconSize: size * 0.6,
          backgroundColor: colors.base200,
          foregroundColor: colors.iconMuted,
        ),
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
      child: Theme(
        data: theme.copyWith(extensions: extensions),
        child: td.TAvatar(
          image: src.isEmpty ? null : NetworkImage(src),
          child: Icon(Icons.person, size: size * 0.6, color: colors.iconMuted),
        ),
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
