import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

Future<T?> showGfAlertDialog<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = false,
}) {
  return showDialog<T>(
    context: context,
    animationStyle: GfMotion.dialogStyle(context),
    useRootNavigator: true,
    barrierColor: Theme.of(context).colorScheme.scrim,
    builder: (context) => Dialog(
      backgroundColor: GfTheme.colorsOf(context).surfaceRaised,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(GfRadius.xl),
      ),
      clipBehavior: Clip.antiAlias,
      insetPadding: const EdgeInsets.symmetric(
        horizontal: GfSpace.xxl,
        vertical: GfSpace.xxl,
      ),
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
    final typography = GfTheme.typographyOf(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        GfSpace.xxl,
        GfSpace.xxl,
        GfSpace.xxl,
        GfSpace.xl,
      ),
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
                        style: typography.title2.copyWith(
                          color: colors.textPrimary,
                        ),
                        child: title!,
                      ),
                    ),
                    if (content != null) const SizedBox(height: GfSpace.sm),
                  ],
                  if (content != null)
                    DefaultTextStyle.merge(
                      style: typography.small.copyWith(
                        color: colors.textSecondary,
                      ),
                      child: content!,
                    ),
                ],
              ),
            ),
          ),
          if (actions.isNotEmpty) ...[
            const SizedBox(height: GfSpace.xxl),
            // One or two actions share the width; more stack vertically with
            // the confirming action first.
            if (actions.length <= 2)
              Row(
                children: [
                  for (int i = 0; i < actions.length; i++) ...[
                    if (i > 0) const SizedBox(width: GfSpace.md),
                    Expanded(child: actions[i]),
                  ],
                ],
              )
            else
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  for (int i = actions.length - 1; i >= 0; i--) ...[
                    if (i < actions.length - 1)
                      const SizedBox(height: GfSpace.sm),
                    actions[i],
                  ],
                ],
              ),
          ],
        ],
      ),
    );
  }
}

/// Material dialog action with explicit destructive/filled roles.
class GfDialogAction extends StatelessWidget {
  const GfDialogAction({
    super.key,
    required this.onPressed,
    required this.child,
    this.destructive = false,
    this.filled = false,
  });
  final VoidCallback? onPressed;
  final Widget child;
  final bool destructive;

  final bool filled;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final (Color background, Color foreground) = switch ((
      filled,
      destructive,
    )) {
      (true, true) => (colors.error, colors.errorContent),
      (true, false) => (colors.primary, colors.primaryContent),
      (false, true) => (colors.fill, colors.error),
      (false, false) => (colors.fill, colors.textPrimary),
    };
    return FilledButton(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        backgroundColor: background,
        foregroundColor: foreground,
        disabledBackgroundColor: colors.fill,
        disabledForegroundColor: colors.textDisabled,
        minimumSize: const Size(64, GfControl.large),
        padding: const EdgeInsets.symmetric(horizontal: GfSpace.lg),
      ),
      child: child,
    );
  }
}
