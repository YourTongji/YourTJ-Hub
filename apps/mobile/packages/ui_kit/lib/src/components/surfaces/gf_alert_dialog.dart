import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

Future<T?> showGfAlertDialog<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = false,
}) {
  return td.TDialog.show<T>(
    context,
    dialog: Builder(builder: builder),
    barrierDismissible: barrierDismissible,
  );
}

/// Alert-style dialog backed by TDesign while keeping a familiar page API.
class GfAlertDialog extends StatelessWidget {
  const GfAlertDialog({
    super.key,
    this.title,
    this.content,
    this.actions = const <Widget>[],
  });

  final Widget? title;
  final Widget? content;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    return td.TDialog(
      title: title,
      content: content,
      actionsWidget: actions.isEmpty
          ? null
          : Padding(
              padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: <Widget>[
                  for (
                    int index = 0;
                    index < actions.length;
                    index++
                  ) ...<Widget>[
                    if (index > 0) const SizedBox(width: 8),
                    Flexible(child: actions[index]),
                  ],
                ],
              ),
            ),
    );
  }
}
