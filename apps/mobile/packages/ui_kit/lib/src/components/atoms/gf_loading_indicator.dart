import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

/// Compact loading primitive backed by TDesign.
class GfLoadingIndicator extends StatelessWidget {
  const GfLoadingIndicator({super.key, this.message, this.small = false});

  final String? message;
  final bool small;

  @override
  Widget build(BuildContext context) {
    return td.TLoading(
      size: small ? td.TLoadingSize.small : td.TLoadingSize.large,
      text: message,
    );
  }
}
