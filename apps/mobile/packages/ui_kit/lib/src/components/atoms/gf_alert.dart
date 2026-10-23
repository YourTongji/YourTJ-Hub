import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

/// Quiet inline feedback, distinct from modal confirmation surfaces.
class GfAlert extends StatelessWidget {
  const GfAlert({
    super.key,
    required this.child,
    this.icon,
    this.symbol,
    this.tone = GfTone.neutral,
    this.padding = const EdgeInsets.symmetric(
      horizontal: 14,
      vertical: GfSpace.md,
    ),
  });

  final Widget child;
  final Widget? icon;

  /// ReIcon name drawn in the tone colour when [icon] is not supplied.
  final String? symbol;

  /// Soft background tone shared with badges and icon tiles.
  final GfTone tone;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color accent = colors.toneOnSoft(tone);
    final Widget? leading =
        icon ??
        (symbol == null
            ? null
            : GfSymbol(symbol!, size: GfIconSize.md, color: accent));

    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: colors.toneSoft(tone),
        borderRadius: BorderRadius.circular(GfRadius.md),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          if (leading != null) ...<Widget>[
            IconTheme.merge(
              data: IconThemeData(color: accent, size: GfIconSize.md),
              child: leading,
            ),
            const SizedBox(width: 10),
          ],
          Expanded(
            child: DefaultTextStyle.merge(
              style: GfTheme.typographyOf(
                context,
              ).subhead.copyWith(color: colors.textPrimary),
              child: child,
            ),
          ),
        ],
      ),
    );
  }
}
