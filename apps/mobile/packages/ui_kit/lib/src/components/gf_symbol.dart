import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import '../theme/gf_theme.dart';

/// Original Lucide vectors shared with the Web and editable Figma library.
/// Provider marks retain their official colours.
class GfSymbol extends StatelessWidget {
  const GfSymbol(this.name, {super.key, this.size = 24, this.color});
  final String name;
  final double size;
  final Color? color;

  @override
  Widget build(BuildContext context) => SvgPicture.asset(
    'assets/icons/$name.svg',
    package: 'ui_kit',
    width: size,
    height: size,
    excludeFromSemantics: true,
    colorFilter: name == 'google'
        ? null
        : ColorFilter.mode(
            color ??
                IconTheme.of(context).color ??
                GfTheme.colorsOf(context).baseContent,
            BlendMode.srcIn,
          ),
  );
}
