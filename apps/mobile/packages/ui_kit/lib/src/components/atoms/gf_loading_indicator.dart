import 'package:flutter/cupertino.dart';

import '../../theme/gf_theme.dart';

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
