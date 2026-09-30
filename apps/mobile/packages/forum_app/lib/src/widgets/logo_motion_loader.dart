import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

class LogoMotionLoader extends StatelessWidget {
  const LogoMotionLoader({super.key, this.showMessage = false});

  final bool showMessage;

  static const double size = 64;
  static const String asset = 'assets/splash/logo_motion.webp';

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final message = showMessage ? l10n.tabPageLoading : l10n.commonLoading;
    final colors = GfTheme.colorsOf(context);
    return Center(
      child: Semantics(
        label: message,
        liveRegion: true,
        child: ExcludeSemantics(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Image.asset(
                asset,
                width: size,
                height: size,
                fit: BoxFit.contain,
                gaplessPlayback: true,
                errorBuilder: (_, _, _) => const GfLogo(size: 48),
              ),
              if (showMessage) ...[
                const SizedBox(height: 8),
                Text(
                  message,
                  style: GfTheme.typographyOf(
                    context,
                  ).caption.copyWith(color: colors.iconMuted),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
