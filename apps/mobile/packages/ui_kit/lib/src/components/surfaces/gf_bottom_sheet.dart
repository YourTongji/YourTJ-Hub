import 'package:flutter/material.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;

/// Shows a TDesign bottom popup and returns the value passed to
/// `Navigator.pop` from [builder].
Future<T?> showGfBottomSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  bool barrierDismissible = true,
}) async {
  final td.TPopupHandle handle = td.TPopup.show(
    context,
    options: td.TPopupOptions.bottom(
      child: Builder(builder: builder),
      headerBuilder: null,
      cancelBuilder: null,
      confirmBuilder: null,
      closeOnOverlayClick: barrierDismissible,
      useSafeArea: true,
    ),
  );
  return (await handle.result) as T?;
}
