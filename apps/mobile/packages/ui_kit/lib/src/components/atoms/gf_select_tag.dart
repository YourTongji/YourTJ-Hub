import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

/// Selectable chip backed by TDesign's controlled tag component.
class GfSelectTag extends StatelessWidget {
  const GfSelectTag({
    super.key,
    required this.label,
    required this.selected,
    required this.onChanged,
    this.icon,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool>? onChanged;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return td.TSelectTag(
      label,
      value: selected,
      onChanged: onChanged,
      icon: icon,
      size: td.TTagSize.large,
      colorScheme: td.TTagColorScheme.primary,
    );
  }
}
