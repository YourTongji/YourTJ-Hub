import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../gf_icon_button.dart';
import '../gf_motion.dart';

/// Shows a centered tablet form or compact bottom sheet and returns the value passed to `Navigator.pop`.
///
/// Short sheets fit their content; long sheets must provide a scrollable body.
/// [height] includes the optional handle and is constrained to the viewport.
/// This boundary owns safe areas and, when [keyboardAware], keyboard avoidance:
/// the panel is clamped to the space above the keyboard, so an explicit height
/// can never lift the sheet's own top off-screen. Builders must not add
/// `viewInsets` again.
Future<T?> showGfBottomSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
  bool enableDrag = true,
  bool? showDragHandle,
  double? height,
  bool keyboardAware = false,
}) {
  final navigator = Navigator.of(context, rootNavigator: true);
  final localizations = MaterialLocalizations.of(context);
  if (MediaQuery.sizeOf(navigator.context).width >= 760) {
    return navigator.push<T>(
      DialogRoute<T>(
        context: context,
        barrierDismissible: barrierDismissible,
        barrierColor: GfTheme.colorsOf(context).scrim,
        animationStyle: GfMotion.dialogStyle(context),
        builder: (panelContext) => AnimatedPadding(
          duration: GfMotion.duration(panelContext, GfMotion.content),
          curve: GfMotion.enterCurve,
          padding: EdgeInsets.fromLTRB(
            24,
            24,
            24,
            24 +
                (keyboardAware
                    ? MediaQuery.viewInsetsOf(panelContext).bottom
                    : 0),
          ),
          child: LayoutBuilder(
            builder: (context, constraints) {
              final theme = Theme.of(context);
              return Center(
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    maxWidth: 640,
                    maxHeight: constraints.maxHeight,
                  ),
                  child: Material(
                    color:
                        theme.bottomSheetTheme.backgroundColor ??
                        theme.colorScheme.surface,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(GfRadius.xl),
                    ),
                    clipBehavior: Clip.antiAlias,
                    child: SizedBox(
                      width: double.infinity,
                      height: height == null
                          ? null
                          : math.min(height, constraints.maxHeight),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          if (height != null)
                            Expanded(child: Builder(builder: builder))
                          else
                            Flexible(child: Builder(builder: builder)),
                        ],
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
      ),
    );
  }
  // The mobile shell paints its bottom navigation as a sibling overlay of
  // branch Navigators. Present sheets on the app Navigator so the sheet and
  // its scrim own the complete viewport, including the shell chrome.
  return navigator.push<T>(
    ModalBottomSheetRoute<T>(
      sheetAnimationStyle: GfMotion.sheetStyle(context),
      // Inherit the navigator's live theme instead of a snapshot of the caller.
      // Appearance controls can therefore repaint their still-open sheet.
      capturedThemes: null,
      isDismissible: barrierDismissible,
      enableDrag: enableDrag,
      isScrollControlled: true,
      constraints: const BoxConstraints(maxWidth: 640),
      useSafeArea: true,
      backgroundColor: Colors.transparent,
      elevation: 0,
      showDragHandle: false,
      modalBarrierColor: GfTheme.colorsOf(context).scrim,
      barrierLabel: localizations.scrimLabel,
      barrierOnTapHint: localizations.scrimOnTapHint(
        localizations.bottomSheetLabel,
      ),
      builder: (sheetContext) {
        // Keyboard avoidance: pad by the keyboard inset and, when the caller
        // asked for an explicit height, never let the panel be taller than the
        // space that is left above it. Without the clamp a 600px panel plus a
        // 300px keyboard asks for more height than the viewport has, and the
        // top of the sheet (title, first controls) leaves the screen.
        final double insets = keyboardAware
            ? MediaQuery.viewInsetsOf(sheetContext).bottom
            : 0;
        final double? panelHeight = height == null
            ? null
            : math.min(
                height,
                math.max(0.0, MediaQuery.sizeOf(sheetContext).height - insets),
              );
        return AnimatedPadding(
          duration: GfMotion.duration(sheetContext, GfMotion.content),
          curve: GfMotion.enterCurve,
          padding: EdgeInsets.only(bottom: insets),
          child: Builder(
            builder: (surfaceContext) {
              final theme = Theme.of(surfaceContext);
              final sheetTheme = theme.bottomSheetTheme;
              // Paint inside the keyboard padding, including the home indicator.
              // The route consumes top/side insets; nested builders see no
              // remaining bottom inset and must not add keyboard padding again.
              return Material(
                color: sheetTheme.backgroundColor ?? theme.colorScheme.surface,
                elevation: sheetTheme.elevation ?? 0,
                shape: sheetTheme.shape,
                clipBehavior: Clip.antiAlias,
                child: SafeArea(
                  top: false,
                  child: SizedBox(
                    width: double.infinity,
                    height: panelHeight,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        if (showDragHandle ?? enableDrag)
                          ExcludeSemantics(
                            child: SizedBox(
                              height: 28,
                              child: Center(
                                child: Container(
                                  width: 36,
                                  height: 5,
                                  decoration: BoxDecoration(
                                    color: GfTheme.colorsOf(
                                      surfaceContext,
                                    ).fillStrong,
                                    borderRadius: BorderRadius.circular(
                                      GfRadius.full,
                                    ),
                                  ),
                                ),
                              ),
                            ),
                          ),
                        if (panelHeight != null)
                          Expanded(child: Builder(builder: builder))
                        else
                          Flexible(child: Builder(builder: builder)),
                      ],
                    ),
                  ),
                ),
              );
            },
          ),
        );
      },
    ),
  );
}

/// Title row for bottom sheets: a 20px title with optional description on
/// the leading side and a tonal close action, matching dialog titles.
class GfSheetHeader extends StatelessWidget {
  const GfSheetHeader({
    super.key,
    required this.title,
    this.description,
    this.onClose,
    this.closeLabel,
    this.trailing,
  });

  final String title;
  final String? description;

  /// Shows a tonal close button when set.
  final VoidCallback? onClose;
  final String? closeLabel;

  /// Replaces the close button (e.g. a Done text action).
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final Widget? action =
        trailing ??
        (onClose == null
            ? null
            : GfIconButton(
                symbol: 'x',
                tooltip:
                    closeLabel ??
                    MaterialLocalizations.of(context).closeButtonTooltip,
                onPressed: onClose,
                iconSize: 18,
                variant: GfIconButtonVariant.tonal,
              ));
    return Padding(
      padding: EdgeInsets.fromLTRB(
        GfSpace.xl,
        0,
        action == null ? GfSpace.xl : GfSpace.md,
        GfSpace.md,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Semantics(
                    header: true,
                    child: GfText(title, style: typography.title2),
                  ),
                  if (description != null) ...[
                    const SizedBox(height: GfSpace.xxs),
                    GfText(
                      description!,
                      style: typography.subhead.copyWith(
                        color: colors.textSecondary,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
          if (action != null) ...[const SizedBox(width: GfSpace.sm), action],
        ],
      ),
    );
  }
}
