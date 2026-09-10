import 'package:flutter/material.dart';
import '../../theme/gf_theme.dart';
import '../gf_icon_tile.dart';

/// A compact activity entry: action mark, readable preview, then timestamp.
class GfActivityCard extends StatelessWidget {
  const GfActivityCard({
    super.key,
    required this.symbol,
    required this.title,
    required this.time,
    this.color,
    this.onTap,
  });
  final String symbol;
  final String title;
  final String time;
  final Color? color;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 0),
      child: Material(
        color: colors.base100,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(color: colors.line),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                GfIconTile(symbol, color: color, size: 34),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: GfTheme.typographyOf(
                          context,
                        ).body.copyWith(fontWeight: FontWeight.w600),
                      ),
                      const SizedBox(height: 5),
                      Text(
                        time,
                        style: GfTheme.typographyOf(
                          context,
                        ).caption.copyWith(color: colors.iconMuted),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
