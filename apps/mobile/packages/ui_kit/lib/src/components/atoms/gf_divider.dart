import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

import '../../theme/gf_theme.dart';

/// Hairline divider, mirroring web `border-line/70` hairlines used across
/// list rows (1px, `line` color). [inset] leaves a left/right gutter like
/// web `inset-x-4` dividers.
class GfDivider extends StatelessWidget {
  const GfDivider({super.key, this.inset = 0, this.color});

  /// Horizontal inset applied to both sides (web `inset-x-4` = 16).
  final double inset;

  /// Override color; defaults to `line` (web `border-line`).
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Theme(
      data: Theme.of(context).copyWith(
        dividerTheme: DividerThemeData(
          color: color ?? colors.line,
          thickness: 1,
          indent: inset,
          endIndent: inset,
        ),
      ),
      child: const td.TDivider(),
    );
  }
}
