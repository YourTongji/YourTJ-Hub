import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import 'gf_motion.dart';

/// Official community logo mark supporting dark and light themes.
///
/// Under dark mode, it displays the light (white) logo vector mark;
/// under light mode, it displays the dark (black) logo vector mark.
class GfLogo extends StatelessWidget {
  const GfLogo({super.key, this.size = 32, this.semanticLabel = 'YourTJ'});

  /// Visual size in logical pixels (width and height).
  final double size;

  /// Accessibility semantic label.
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    final bool isDark = Theme.of(context).brightness == Brightness.dark;
    final String assetName = isDark
        ? 'assets/logos/logo-light.svg'
        : 'assets/logos/logo-dark.svg';

    return Semantics(
      label: semanticLabel,
      image: true,
      child: AnimatedSwitcher(
        duration: GfMotion.duration(context, GfMotion.layout),
        switchInCurve: GfMotion.enterCurve,
        switchOutCurve: GfMotion.enterCurve,
        child: SvgPicture.asset(
          assetName,
          key: ValueKey<String>(assetName),
          package: 'ui_kit',
          width: size,
          height: size,
          fit: BoxFit.contain,
        ),
      ),
    );
  }
}
