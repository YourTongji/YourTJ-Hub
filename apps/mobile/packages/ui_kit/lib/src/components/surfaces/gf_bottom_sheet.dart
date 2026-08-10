import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

/// Shows a TDesign bottom popup and returns the value passed to
/// `Navigator.pop` from [builder].
Future<T?> showGfBottomSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
  double? height,
  bool keyboardAware = false,
}) async {
  // TPopup keeps its panel pinned to the physical bottom of the viewport and
  // does not offset it for `viewInsets`. Search/composer sheets therefore need
  // Flutter's scroll-controlled route so the focused field remains above the
  // iOS keyboard. The visible panel still inherits the shared TDesign/Gf
  // bottom-sheet theme and hosts the same Gf components.
  if (keyboardAware) {
    return showModalBottomSheet<T>(
      context: context,
      isDismissible: barrierDismissible,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).bottomSheetTheme.backgroundColor,
      barrierColor: Theme.of(context).colorScheme.scrim,
      shape: Theme.of(context).bottomSheetTheme.shape,
      clipBehavior: Clip.antiAlias,
      builder: (BuildContext sheetContext) => AnimatedPadding(
        duration: const Duration(milliseconds: 160),
        curve: Curves.easeOutCubic,
        padding: EdgeInsets.only(
          bottom: MediaQuery.viewInsetsOf(sheetContext).bottom,
        ),
        child: SizedBox(
          height: height,
          child: Material(
            type: MaterialType.transparency,
            child: Builder(builder: builder),
          ),
        ),
      ),
    );
  }

  final td.TPopupHandle handle = td.TPopup.show(
    context,
    options: td.TPopupOptions.bottom(
      height: height,
      // TPopup renders its panel with a decorated Container rather than a
      // Material widget. Keep Material inputs, cells and ink effects usable
      // inside the popup route while preserving TDesign's panel background.
      child: Material(
        type: MaterialType.transparency,
        child: Builder(builder: builder),
      ),
      headerBuilder: null,
      cancelBuilder: null,
      confirmBuilder: null,
      closeOnOverlayClick: barrierDismissible,
      useSafeArea: true,
    ),
  );
  return (await handle.result) as T?;
}
