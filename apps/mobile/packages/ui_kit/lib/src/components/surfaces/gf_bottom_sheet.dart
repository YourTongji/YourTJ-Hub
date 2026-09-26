import 'package:flutter/material.dart';

/// Shows a themed bottom sheet and returns the value passed to `Navigator.pop`.
///
/// Short sheets fit their content; long sheets must provide a scrollable body.
/// [height] includes the optional handle and is constrained to the viewport.
/// This boundary owns safe areas and, when [keyboardAware], keyboard avoidance.
/// Builders must not add `viewInsets` again.
Future<T?> showGfBottomSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
  bool enableDrag = true,
  bool? showDragHandle,
  double? height,
  bool keyboardAware = false,
}) {
  final theme = Theme.of(context);
  final sheetTheme = theme.bottomSheetTheme;
  return showModalBottomSheet<T>(
    context: context,
    // The mobile shell paints its bottom navigation as a sibling overlay of
    // branch Navigators. Present sheets on the app Navigator so the sheet and
    // its scrim own the complete viewport, including the shell chrome.
    useRootNavigator: true,
    isDismissible: barrierDismissible,
    enableDrag: enableDrag,
    isScrollControlled: true,
    constraints: const BoxConstraints(maxWidth: 640),
    useSafeArea: true,
    // Paint inside the keyboard padding, including the home-indicator area.
    // Padding inside the route's painted Material leaves a blank surface behind
    // the keyboard and moves the rounded corners away from the content.
    backgroundColor: Colors.transparent,
    elevation: 0,
    showDragHandle: false,
    barrierColor: theme.colorScheme.scrim,
    builder: (sheetContext) => AnimatedPadding(
      duration: MediaQuery.disableAnimationsOf(sheetContext)
          ? Duration.zero
          : const Duration(milliseconds: 160),
      curve: Curves.easeOutCubic,
      padding: EdgeInsets.only(
        bottom: keyboardAware
            ? MediaQuery.viewInsetsOf(sheetContext).bottom
            : 0,
      ),
      child: Material(
        color: sheetTheme.backgroundColor ?? theme.colorScheme.surface,
        elevation: sheetTheme.elevation ?? 0,
        shape: sheetTheme.shape,
        clipBehavior: Clip.antiAlias,
        // The route consumes top/side insets; consume the bottom inset within
        // the painted surface. Nested SafeAreas and ListViews see zero remaining
        // padding, rather than the screen's notch height.
        child: SafeArea(
          top: false,
          child: SizedBox(
            width: double.infinity,
            height: height,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (showDragHandle ?? enableDrag)
                  ExcludeSemantics(
                    child: SizedBox(
                      height: 28,
                      child: Center(
                        child: Container(
                          width: 32,
                          height: 4,
                          decoration: BoxDecoration(
                            color: theme.colorScheme.onSurfaceVariant
                                .withValues(alpha: 0.3),
                            borderRadius: BorderRadius.circular(2),
                          ),
                        ),
                      ),
                    ),
                  ),
                if (height != null)
                  Expanded(child: Builder(builder: builder))
                else
                  Flexible(child: Builder(builder: builder)),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}
