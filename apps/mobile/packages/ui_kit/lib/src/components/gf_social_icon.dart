import 'package:flutter/material.dart';
import '../theme/gf_theme.dart';
import 'gf_symbol.dart';

/// The same six official provider marks and colours as Web social-icons.ts.
class GfSocialIcon extends StatelessWidget {
  const GfSocialIcon(this.provider, {super.key, this.size = 20});
  final String? provider;
  final double size;
  @override
  Widget build(BuildContext context) {
    final (symbol, color) = switch (provider) {
      'github' => ('github', GfTheme.colorsOf(context).baseContent),
      'twitter' => ('twitter', GfTheme.colorsOf(context).baseContent),
      'linkedIn' => ('linkedin', const Color(0xFF0A66C2)),
      'weibo' => ('weibo', const Color(0xFFE6162D)),
      'bilibili' => ('bilibili', const Color(0xFF00A1D6)),
      'zhihu' => ('zhihu', const Color(0xFF0084FF)),
      _ => ('link', GfTheme.colorsOf(context).iconMuted),
    };
    return GfSymbol(symbol, size: size, color: color);
  }
}
