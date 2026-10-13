import 'dart:math' as math;

import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../gf_motion.dart';
import 'gf_bottom_sheet.dart';

/// Menu values and permissions stay shared; only presentation follows iOS.
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
  Widget? title,
  bool useRootNavigator = true,
  Widget Function(BuildContext, Widget)? surfaceBuilder,
}) {
  final navigator = Navigator.of(context, rootNavigator: useRootNavigator);
  Rect anchor() {
    final overlay = navigator.overlay!.context.findRenderObject()! as RenderBox;
    final box = (anchorContext ?? context).findRenderObject();
    if (anchorContext == null && position != null) {
      return position.toRect(Offset.zero & overlay.size);
    }
    if (box is RenderBox && box.attached && box.hasSize) {
      return box.localToGlobal(Offset.zero, ancestor: overlay) & box.size;
    }
    return Rect.fromCenter(
      center: overlay.size.center(Offset.zero),
      width: 0,
      height: 0,
    );
  }

  if (Theme.of(context).platform != TargetPlatform.iOS) {
    final overlay = navigator.overlay!.context.findRenderObject()! as RenderBox;
    return showMenu<T>(
      context: context,
      useRootNavigator: useRootNavigator,
      position:
          position ??
          RelativeRect.fromRect(anchor(), Offset.zero & overlay.size),
      initialValue: initialValue,
      items: items,
    );
  }

  Widget sheet(BuildContext routeContext) {
    final content = CupertinoActionSheet(
      title: title,
      actions: [
        for (final item in items)
          if (item is PopupMenuItem<T>)
            if (!item.enabled)
              Semantics(
                enabled: false,
                button: true,
                child: ExcludeFocus(
                  child: AbsorbPointer(
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 18,
                        vertical: 16,
                      ),
                      child: DefaultTextStyle.merge(
                        style: TextStyle(
                          color: CupertinoColors.inactiveGray.resolveFrom(
                            routeContext,
                          ),
                        ),
                        child: item.child ?? const SizedBox(),
                      ),
                    ),
                  ),
                ),
              )
            else
              Semantics(
                selected: initialValue != null && item.value == initialValue,
                child: CupertinoActionSheetAction(
                  isDefaultAction:
                      initialValue != null && item.value == initialValue,
                  isDestructiveAction:
                      item is GfPopupMenuItem<T> && item.destructive,
                  onPressed: () => Navigator.pop(routeContext, item.value),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Flexible(child: item.child ?? const SizedBox()),
                      if (initialValue != null &&
                          item.value == initialValue) ...[
                        const SizedBox(width: 8),
                        const Icon(CupertinoIcons.check_mark, size: 18),
                      ],
                    ],
                  ),
                ),
              ),
      ],
      cancelButton: CupertinoActionSheetAction(
        onPressed: () => Navigator.pop(routeContext),
        child: Text(MaterialLocalizations.of(routeContext).cancelButtonLabel),
      ),
    );
    return surfaceBuilder?.call(routeContext, content) ?? content;
  }

  return showGeneralDialog<T>(
    context: context,
    useRootNavigator: useRootNavigator,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: const Color(0x33000000),
    transitionDuration: GfMotion.duration(context, GfMotion.layout),
    pageBuilder: (context, _, _) {
      final media = MediaQuery.of(context);
      return Shortcuts(
        shortcuts: const {
          SingleActivator(LogicalKeyboardKey.escape): DismissIntent(),
        },
        child: Actions(
          actions: {
            DismissIntent: CallbackAction<DismissIntent>(
              onInvoke: (_) {
                Navigator.pop(context);
                return null;
              },
            ),
          },
          child: Focus(
            autofocus: true,
            child: media.size.width < 600
                ? Align(
                    alignment: Alignment.bottomCenter,
                    child: Padding(
                      padding: EdgeInsets.only(bottom: media.viewInsets.bottom),
                      child: sheet(context),
                    ),
                  )
                : CustomSingleChildLayout(
                    delegate: _MenuPosition(
                      anchor(),
                      media.padding,
                      media.viewInsets.bottom,
                    ),
                    child: MediaQuery(
                      data: media.copyWith(
                        padding: EdgeInsets.zero,
                        viewPadding: EdgeInsets.zero,
                        viewInsets: EdgeInsets.zero,
                      ),
                      child: sheet(context),
                    ),
                  ),
          ),
        ),
      );
    },
  );
}

class _MenuPosition extends SingleChildLayoutDelegate {
  const _MenuPosition(this.anchor, this.safe, this.keyboard);
  final Rect anchor;
  final EdgeInsets safe;
  final double keyboard;

  @override
  BoxConstraints getConstraintsForChild(BoxConstraints constraints) =>
      BoxConstraints(
        maxWidth: math.min(
          380,
          math.max(0, constraints.maxWidth - safe.horizontal - 24),
        ),
        maxHeight: math.max(
          0,
          constraints.maxHeight -
              safe.top -
              math.max(safe.bottom, keyboard) -
              24,
        ),
      );

  @override
  Offset getPositionForChild(Size size, Size childSize) {
    final left = safe.left + 12;
    final top = safe.top + 12;
    final bottom = size.height - math.max(safe.bottom, keyboard) - 12;
    final y = anchor.bottom + childSize.height <= bottom
        ? anchor.bottom
        : anchor.top - childSize.height;
    return Offset(
      anchor.left.clamp(
        left,
        math.max(left, size.width - safe.right - childSize.width - 12),
      ),
      y.clamp(top, math.max(top, bottom - childSize.height)),
    );
  }

  @override
  bool shouldRelayout(_MenuPosition oldDelegate) =>
      anchor != oldDelegate.anchor ||
      safe != oldDelegate.safe ||
      keyboard != oldDelegate.keyboard;
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
  Widget build(BuildContext context) {
    if (Theme.of(context).platform != TargetPlatform.iOS) {
      return PopupMenuButton<T>(
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
    return Builder(
      builder: (anchor) {
        Future<void> open() async {
          final items = itemBuilder(context);
          if (items.isEmpty) return;
          final value = await showGfMenu<T>(
            context: context,
            anchorContext: anchor,
            items: items,
            useRootNavigator: true,
          );
          if (context.mounted && value != null) onSelected?.call(value);
        }

        if (child != null) {
          return Tooltip(
            message:
                tooltip ?? MaterialLocalizations.of(context).showMenuTooltip,
            child: InkWell(onTap: enabled ? open : null, child: child),
          );
        }
        return IconButton(
          tooltip: tooltip ?? MaterialLocalizations.of(context).showMenuTooltip,
          style: style,
          onPressed: enabled ? open : null,
          icon: icon ?? const Icon(CupertinoIcons.ellipsis),
        );
      },
    );
  }
}

/// Short actions share iOS anchored presentation and Android bottom sheets.
Future<T?> showGfActionSheet<T>(
  BuildContext context, {
  required List<PopupMenuEntry<T>> items,
  Widget? title,
  T? initialValue,
  BuildContext? anchorContext,
  Widget Function(BuildContext, Widget)? surfaceBuilder,
}) {
  if (Theme.of(context).platform == TargetPlatform.iOS) {
    return showGfMenu<T>(
      context: context,
      items: items,
      title: title,
      initialValue: initialValue,
      anchorContext: anchorContext,
      surfaceBuilder: surfaceBuilder,
    );
  }
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
