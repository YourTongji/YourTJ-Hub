import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

Future<T?> showGfAlertDialog<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = false,
}) {
  return showDialog<T>(
    context: context,
    useRootNavigator: true,
    barrierColor: Theme.of(context).colorScheme.scrim,
    builder: (context) => Dialog(
      backgroundColor: GfTheme.colorsOf(context).base100,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      clipBehavior: Clip.antiAlias,
      insetPadding: const EdgeInsets.all(16),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: Builder(builder: builder),
      ),
    ),
    barrierDismissible: barrierDismissible,
  );
}

/// Alert content inside one native dialog surface. Content scrolls independently
/// of the action row; long or enlarged actions stack instead of being squeezed.
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
    final colors = GfTheme.colorsOf(context);
    return Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Flexible(
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (title != null) ...[
                    Semantics(
                      namesRoute: true,
                      child: DefaultTextStyle.merge(
                        style: TextStyle(
                          color: colors.baseContent,
                          fontSize: 20,
                          height: 1.3,
                          fontWeight: FontWeight.w700,
                        ),
                        child: title!,
                      ),
                    ),
                    if (content != null) const SizedBox(height: 12),
                  ],
                  if (content != null)
                    DefaultTextStyle.merge(
                      style: TextStyle(
                        color: colors.baseContent,
                        fontSize: 15,
                        height: 1.45,
                      ),
                      child: content!,
                    ),
                ],
              ),
            ),
          ),
          if (actions.isNotEmpty) ...[
            const SizedBox(height: 16),
            OverflowBar(
              alignment: MainAxisAlignment.end,
              overflowAlignment: OverflowBarAlignment.end,
              spacing: 8,
              overflowSpacing: 8,
              children: actions,
            ),
          ],
        ],
      ),
    );
  }
}
