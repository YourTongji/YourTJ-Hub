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
  final navigator = Navigator.of(context, rootNavigator: true);
  final localizations = MaterialLocalizations.of(context);
  // The mobile shell paints its bottom navigation as a sibling overlay of
  // branch Navigators. Present sheets on the app Navigator so the sheet and
  // its scrim own the complete viewport, including the shell chrome.
  return navigator.push<T>(
    ModalBottomSheetRoute<T>(
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
      modalBarrierColor: const Color(0x66000000),
      barrierLabel: localizations.scrimLabel,
      barrierOnTapHint: localizations.scrimOnTapHint(
        localizations.bottomSheetLabel,
      ),
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
            );
          },
        ),
      ),
    ),
  );
}
