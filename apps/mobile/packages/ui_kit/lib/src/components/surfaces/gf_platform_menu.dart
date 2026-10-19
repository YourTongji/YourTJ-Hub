import 'package:flutter/material.dart';

import 'gf_bottom_sheet.dart';

/// A [PopupMenuItem] that can mark a destructive action.
class GfPopupMenuItem<T> extends PopupMenuItem<T> {
  const GfPopupMenuItem({
    super.key,
    super.value,
    super.enabled,
    required super.child,
    this.destructive = false,
  });

  final bool destructive;
}

Future<T?> showGfMenu<T>({
  required BuildContext context,
  required List<PopupMenuEntry<T>> items,
  BuildContext? anchorContext,
  RelativeRect? position,
  T? initialValue,
  bool useRootNavigator = true,
}) {
  final navigator = Navigator.of(context, rootNavigator: useRootNavigator);
  final overlay = navigator.overlay!.context.findRenderObject()! as RenderBox;
  Rect anchor() {
    final box = (anchorContext ?? context).findRenderObject();
    if (box is RenderBox && box.attached && box.hasSize) {
      return box.localToGlobal(Offset.zero, ancestor: overlay) & box.size;
    }
    return Rect.fromCenter(
      center: overlay.size.center(Offset.zero),
      width: 0,
      height: 0,
    );
  }

  return showMenu<T>(
    context: context,
    useRootNavigator: useRootNavigator,
    position:
        position ?? RelativeRect.fromRect(anchor(), Offset.zero & overlay.size),
    initialValue: initialValue,
    items: items,
  );
}

class GfPopupMenuButton<T> extends StatelessWidget {
  const GfPopupMenuButton({
    super.key,
    required this.itemBuilder,
    this.onSelected,
    this.icon,
    this.child,
    this.tooltip,
    this.enabled = true,
    this.useRootNavigator = false,
    this.color,
    this.style,
  });
  final PopupMenuItemBuilder<T> itemBuilder;
  final PopupMenuItemSelected<T>? onSelected;
  final Widget? icon;
  final Widget? child;
  final String? tooltip;
  final bool enabled;
  final bool useRootNavigator;
  final Color? color;
  final ButtonStyle? style;

  @override
  Widget build(BuildContext context) => PopupMenuButton<T>(
    itemBuilder: itemBuilder,
    onSelected: onSelected,
    icon: icon,
    tooltip: tooltip,
    enabled: enabled,
    useRootNavigator: useRootNavigator,
    color: color,
    style: style,
    child: child,
  );
}

/// Short actions in a Material bottom sheet on every platform.
///
/// [anchorContext] is accepted for call-site compatibility and ignored.
Future<T?> showGfActionSheet<T>(
  BuildContext context, {
  required List<PopupMenuEntry<T>> items,
  Widget? title,
  T? initialValue,
  BuildContext? anchorContext,
  Widget Function(BuildContext, Widget)? surfaceBuilder,
}) {
  return showGfBottomSheet<T>(
    context,
    keyboardAware: true,
    builder: (context) {
      final content = ListView(
        shrinkWrap: true,
        padding: EdgeInsets.zero,
        children: [
          if (title != null)
            Padding(padding: const EdgeInsets.all(16), child: title),
          for (final item in items)
            if (item is PopupMenuItem<T>)
              ListTile(
                title: DefaultTextStyle.merge(
                  style: item is GfPopupMenuItem<T> && item.destructive
                      ? TextStyle(color: Theme.of(context).colorScheme.error)
                      : const TextStyle(),
                  child: item.child ?? const SizedBox(),
                ),
                enabled: item.enabled,
                selected: initialValue != null && item.value == initialValue,
                trailing: initialValue != null && item.value == initialValue
                    ? const Icon(Icons.check)
                    : null,
                onTap: () => Navigator.pop(context, item.value),
              ),
        ],
      );
      return surfaceBuilder?.call(context, content) ?? content;
    },
  );
}
