import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_symbol.dart';

import 'gf_bottom_sheet.dart';

/// A [PopupMenuItem] with an optional leading widget and destructive marker.
class GfPopupMenuItem<T> extends PopupMenuItem<T> {
  const GfPopupMenuItem({
    super.key,
    super.value,
    super.enabled,
    required super.child,
    this.destructive = false,
    this.leading,
  });

  final bool destructive;

  /// Optional icon before [child]. Keep [child] as the label when using this
  /// slot; existing custom rows without [leading] are rendered unchanged.
  final Widget? leading;

  Widget? _content() => leading == null
      ? child
      : Row(
          children: [
            leading!,
            const SizedBox(width: GfSpace.md),
            Expanded(child: child ?? const SizedBox()),
          ],
        );

  @override
  PopupMenuItemState<T, GfPopupMenuItem<T>> createState() =>
      _GfPopupMenuItemState<T>();
}

class _GfPopupMenuItemState<T>
    extends PopupMenuItemState<T, GfPopupMenuItem<T>> {
  @override
  Widget? buildChild() => widget._content();
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
Future<T?> showGfActionSheet<T>(
  BuildContext context, {
  required List<PopupMenuEntry<T>> items,
  Widget? title,
  T? initialValue,
  Widget Function(BuildContext, Widget)? surfaceBuilder,
}) {
  return showGfBottomSheet<T>(
    context,
    keyboardAware: true,
    builder: (context) {
      final colors = GfTheme.colorsOf(context);
      final content = ListView(
        shrinkWrap: true,
        padding: const EdgeInsets.fromLTRB(
          GfSpace.sm,
          0,
          GfSpace.sm,
          GfSpace.sm,
        ),
        children: [
          if (title != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(
                GfSpace.md,
                GfSpace.xs,
                GfSpace.md,
                GfSpace.md,
              ),
              child: DefaultTextStyle.merge(
                style: GfTheme.typographyOf(context).title3,
                child: title,
              ),
            ),
          for (final item in items)
            if (item is PopupMenuItem<T>)
              ListTile(
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(GfRadius.md),
                ),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: GfSpace.md,
                ),
                minTileHeight: GfControl.row,
                title: DefaultTextStyle.merge(
                  style: item is GfPopupMenuItem<T> && item.destructive
                      ? TextStyle(color: colors.error)
                      : const TextStyle(),
                  child:
                      (item is GfPopupMenuItem<T>
                          ? item._content()
                          : item.child) ??
                      const SizedBox(),
                ),
                enabled: item.enabled,
                selected: initialValue != null && item.value == initialValue,
                selectedColor: colors.primary,
                selectedTileColor: colors.primarySoft,
                trailing: initialValue != null && item.value == initialValue
                    ? GfSymbol(
                        'check',
                        size: GfIconSize.md,
                        color: colors.primary,
                      )
                    : null,
                onTap: () => Navigator.pop(context, item.value),
              ),
        ],
      );
      return surfaceBuilder?.call(context, content) ?? content;
    },
  );
}
