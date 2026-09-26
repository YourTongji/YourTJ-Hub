import 'package:flutter/material.dart';

/// Shows a themed bottom sheet and returns the value passed to `Navigator.pop`.
///
/// Short sheets fit their content; long sheets must provide a scrollable body.
/// [height] is a preferred content height, constrained to the available viewport.
/// This boundary owns safe areas and, when [keyboardAware], keyboard avoidance.
/// Builders must not add `viewInsets` again.
Future<T?> showGfBottomSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
  bool enableDrag = true,
  double? height,
  bool keyboardAware = false,
}) {
  final navigator = Navigator.of(context, rootNavigator: true);
  final localizations = MaterialLocalizations.of(context);
  // The mobile shell paints its bottom navigation as a sibling overlay of
  // branch Navigators. Present sheets on the app Navigator so the sheet and
  // its scrim own the complete viewport, including the shell chrome.
  return navigator.push<T>(
    ModalBottomSheetRoute<T>(
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
        child: Builder(
          builder: (surfaceContext) {
            final theme = Theme.of(surfaceContext);
            final sheetTheme = theme.bottomSheetTheme;
            return Material(
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
                  child: Builder(builder: builder),
                ),
              ),
            );
          },
        ),
      ),
      capturedThemes: null,
      isScrollControlled: true,
      barrierLabel: localizations.scrimLabel,
      barrierOnTapHint: localizations.scrimOnTapHint(
        localizations.bottomSheetLabel,
      ),
      backgroundColor: Colors.transparent,
      elevation: 0,
      modalBarrierColor: const Color(0x66000000),
      isDismissible: barrierDismissible,
      enableDrag: enableDrag,
      showDragHandle: false,
      useSafeArea: true,
    ),
  );
}
