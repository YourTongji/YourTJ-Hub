import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import '../../theme/gf_theme.dart';

/// A server-defined badge, including the SVG assets used by Web UserAvatar.
class GfBadgeIcon extends StatelessWidget {
  const GfBadgeIcon({
    super.key,
    required this.url,
    required this.label,
    this.size = 28,
    this.framed = true,
  });
  final String url;
  final String label;
  final double size;
  final bool framed;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final fallback = Icon(
      Icons.workspace_premium_outlined,
      size: size - 6,
      color: colors.primary,
    );
    final isSvg =
        Uri.tryParse(url)?.path.toLowerCase().endsWith('.svg') ?? false;
    return Tooltip(
      message: label,
      child: Semantics(
        label: label,
        image: true,
        child: Container(
          width: size,
          height: size,
          padding: const EdgeInsets.all(2),
          decoration: !framed
              ? null
              : BoxDecoration(
                  color: colors.base100,
                  shape: BoxShape.circle,
                  border: Border.all(color: colors.line),
                ),
          child: ExcludeSemantics(
            child: url.isEmpty
                ? fallback
                : isSvg
                ? SvgPicture.network(
                    url,
                    fit: BoxFit.contain,
                    placeholderBuilder: (_) => fallback,
                    errorBuilder: (_, _, _) => fallback,
                  )
                : Image.network(
                    url,
                    fit: BoxFit.contain,
                    errorBuilder: (_, _, _) => fallback,
                  ),
          ),
        ),
      ),
    );
  }
}
