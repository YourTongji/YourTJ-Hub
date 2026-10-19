import 'package:flutter/material.dart';

/// Material [DropdownButtonFormField] on every platform.
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
  Widget build(BuildContext context) => DropdownButtonFormField<T>(
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

/// Material [DropdownButton] on every platform.
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
  Widget build(BuildContext context) => DropdownButton<T>(
    value: value,
    items: items,
    onChanged: onChanged,
    isExpanded: isExpanded,
    isDense: isDense,
    dropdownColor: dropdownColor,
    icon: icon,
  );
}
