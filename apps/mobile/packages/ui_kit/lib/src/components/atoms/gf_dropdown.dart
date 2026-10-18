import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';

import '../surfaces/gf_platform_menu.dart';

/// Retains Form validation on both platforms; iOS presents checked choices.
class GfDropdownButtonFormField<T> extends StatelessWidget {
  const GfDropdownButtonFormField({
    super.key,
    required this.items,
    required this.onChanged,
    this.initialValue,
    this.isExpanded = false,
    this.itemHeight,
    this.borderRadius,
    this.icon,
    this.decoration,
    this.validator,
    this.onSaved,
  });
  final List<DropdownMenuItem<T>> items;
  final ValueChanged<T?>? onChanged;
  final T? initialValue;
  final bool isExpanded;
  final double? itemHeight;
  final BorderRadius? borderRadius;
  final Widget? icon;
  final InputDecoration? decoration;
  final FormFieldValidator<T>? validator;
  final FormFieldSetter<T>? onSaved;

  @override
  Widget build(BuildContext context) {
    if (Theme.of(context).platform != TargetPlatform.iOS) {
      return DropdownButtonFormField<T>(
        initialValue: initialValue,
        items: items,
        onChanged: onChanged,
        isExpanded: isExpanded,
        itemHeight: itemHeight,
        borderRadius: borderRadius,
        icon: icon,
        decoration: decoration,
        validator: validator,
        onSaved: onSaved,
      );
    }
    return _GfChoiceFormField<T>(
      onChanged: onChanged,
      initialValue: initialValue,
      validator: validator,
      onSaved: onSaved,
      enabled: onChanged != null,
      builder: (field) => InputDecorator(
        decoration: (decoration ?? const InputDecoration())
            .applyDefaults(Theme.of(context).inputDecorationTheme)
            .copyWith(enabled: onChanged != null, errorText: field.errorText),
        isEmpty: !items.any((item) => item.value == field.value),
        child: GfDropdownButton<T>(
          value: field.value,
          items: items,
          isExpanded: true,
          icon: icon,
          onChanged: onChanged == null
              ? null
              : (value) {
                  field.didChange(value);
                  onChanged!(value);
                },
        ),
      ),
    );
  }
}

class GfDropdownButton<T> extends StatelessWidget {
  const GfDropdownButton({
    super.key,
    required this.items,
    required this.onChanged,
    this.value,
    this.isExpanded = false,
    this.isDense = true,
    this.dropdownColor,
    this.icon,
  });
  final List<DropdownMenuItem<T>> items;
  final ValueChanged<T?>? onChanged;
  final T? value;
  final bool isExpanded;
  final bool isDense;
  final Color? dropdownColor;
  final Widget? icon;

  @override
  Widget build(BuildContext context) {
    if (Theme.of(context).platform != TargetPlatform.iOS) {
      return DropdownButton<T>(
        value: value,
        items: items,
        onChanged: onChanged,
        isExpanded: isExpanded,
        isDense: isDense,
        dropdownColor: dropdownColor,
        icon: icon,
      );
    }
    final selected = items.indexWhere((item) => item.value == value);
    return Builder(
      builder: (anchor) => CupertinoButton(
        padding: EdgeInsets.zero,
        minimumSize: const Size(44, 44),
        alignment: AlignmentDirectional.centerStart,
        onPressed: onChanged == null || items.isEmpty
            ? null
            : () async {
                // Indices distinguish a nullable option from dismissing the route.
                final index = await showGfMenu<int>(
                  context: context,
                  anchorContext: anchor,
                  initialValue: selected,
                  items: [
                    for (var i = 0; i < items.length; i++)
                      GfPopupMenuItem(
                        value: i,
                        enabled: items[i].enabled,
                        child: items[i].child,
                      ),
                  ],
                );
                if (context.mounted && index != null) {
                  onChanged?.call(items[index].value);
                }
              },
        child: Row(
          mainAxisSize: isExpanded ? MainAxisSize.max : MainAxisSize.min,
          children: [
            Flexible(
              child: selected < 0 ? const SizedBox() : items[selected].child,
            ),
            const SizedBox(width: 8),
            icon ?? const Icon(CupertinoIcons.chevron_down, size: 18),
          ],
        ),
      ),
    );
  }
}

class _GfChoiceFormField<T> extends FormField<T> {
  const _GfChoiceFormField({
    required super.builder,
    required this.onChanged,
    super.initialValue,
    super.validator,
    super.onSaved,
    super.enabled,
  });

  final ValueChanged<T?>? onChanged;

  @override
  FormFieldState<T> createState() => _GfChoiceFormFieldState<T>();
}

class _GfChoiceFormFieldState<T> extends FormFieldState<T> {
  @override
  void reset() {
    super.reset();
    (widget as _GfChoiceFormField<T>).onChanged?.call(value);
  }

  @override
  void didUpdateWidget(covariant FormField<T> oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialValue != widget.initialValue) {
      setValue(widget.initialValue);
    }
  }
}
