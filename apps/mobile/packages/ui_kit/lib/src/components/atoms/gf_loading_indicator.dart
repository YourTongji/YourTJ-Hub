import 'dart:ui' show SemanticsRole;

import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// Quiet native progress with optional supporting text.
class GfLoadingIndicator extends StatelessWidget {
  const GfLoadingIndicator({super.key, this.message, this.small = false});

  final String? message;
  final bool small;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      liveRegion: message != null,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CupertinoActivityIndicator(
            radius: small ? 8 : 11,
            color: colors.iconMuted,
            animating: !MediaQuery.disableAnimationsOf(context),
          ),
          if (message != null) ...[
            const SizedBox(height: 10),
            Text(
              message!,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: colors.iconMuted,
                fontSize: 14,
                height: 1.4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Native progress with an honest, static indeterminate state for reduced motion.
class GfProgressIndicator extends StatelessWidget {
  const GfProgressIndicator({
    super.key,
    this.value,
    this.color,
    this.strokeWidth = 4,
  });
  final double? value;
  final Color? color;
  final double strokeWidth;

  @override
  Widget build(BuildContext context) {
    final stationary = value == null && GfMotion.reducedOf(context);
    final indicator = CircularProgressIndicator(
      value: stationary ? .75 : value,
      color: color,
      strokeWidth: strokeWidth,
    );
    return stationary
        ? Semantics(
            role: SemanticsRole.loadingSpinner,
            child: ExcludeSemantics(child: indicator),
          )
        : indicator;
  }
}
