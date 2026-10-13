import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

/// The surrounding RadioGroup owns selection; iOS exposes a checked row.
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
  Widget build(BuildContext context) {
    if (Theme.of(context).platform != TargetPlatform.iOS) {
      return RadioListTile<T>(
        value: value,
        title: title,
        contentPadding: contentPadding,
        controlAffinity: controlAffinity,
      );
    }
    final group = RadioGroup.maybeOf<T>(context)!;
    final selected = group.groupValue == value;
    return Semantics(
      checked: selected,
      inMutuallyExclusiveGroup: true,
      child: ListTile(
        contentPadding: contentPadding,
        title: title,
        trailing: selected
            ? Icon(
                CupertinoIcons.check_mark,
                color: Theme.of(context).colorScheme.primary,
              )
            : null,
        onTap: () => group.onChanged(value),
      ),
    );
  }
}
