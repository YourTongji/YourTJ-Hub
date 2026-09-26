import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Theme fallback for routes without an AppBar or their own status-bar region.
///
/// Descendant regions, such as the immersive profile cover, take precedence
/// while visible. Returning to a plain route restores this style automatically.
class AppSystemUiOverlay extends StatelessWidget {
  const AppSystemUiOverlay({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final brightness = Theme.of(context).brightness;
    final iconBrightness = brightness == Brightness.dark
        ? Brightness.light
        : Brightness.dark;
    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle(
        statusBarColor: Colors.transparent,
        statusBarBrightness: brightness,
        statusBarIconBrightness: iconBrightness,
        // Explicitly clear the launch screen's Android navigation-bar colors.
        // Null values leave the previous platform style in place.
        systemNavigationBarColor: Colors.transparent,
        systemNavigationBarDividerColor: Colors.transparent,
        systemNavigationBarIconBrightness: iconBrightness,
      ),
      child: child,
    );
  }
}
