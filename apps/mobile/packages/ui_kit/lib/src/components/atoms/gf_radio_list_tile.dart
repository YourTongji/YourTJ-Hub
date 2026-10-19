import 'package:flutter/material.dart';

/// Material [RadioListTile] on every platform; the surrounding RadioGroup
/// owns selection.
class GfRadioListTile<T> extends StatelessWidget {
  const GfRadioListTile({
    super.key,
    required this.value,
    required this.title,
    this.contentPadding,
    this.controlAffinity = ListTileControlAffinity.platform,
  });
  final T value;
  final Widget title;
  final EdgeInsetsGeometry? contentPadding;
  final ListTileControlAffinity controlAffinity;

  @override
  Widget build(BuildContext context) => RadioListTile<T>(
    value: value,
    title: title,
    contentPadding: contentPadding,
    controlAffinity: controlAffinity,
  );
}
