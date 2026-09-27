import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';

/// Native text input with the shared filled form surface. Focus, validation,
/// selection and autofill stay with TextField; Gf owns only the presentation.
class GfInput extends StatelessWidget {
  const GfInput({
    super.key,
    this.controller,
    this.focusNode,
    this.hintText,
    this.labelText,
    this.prefixIcon,
    this.suffixIcon,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
    this.autofillHints,
    this.autocorrect = true,
    this.enableSuggestions = true,
    this.textCapitalization = TextCapitalization.none,
    this.maxLength,
    this.enabled = true,
    this.readOnly = false,
    this.onChanged,
    this.onSubmitted,
    this.onEditingComplete,
    this.onTap,
    this.autofocus = false,
    this.textAlignVertical,
    this.textAlign = TextAlign.start,
    this.inputFormatters,
    this.decoration,
    this.style,
    this.cursorColor,
    this.minLines,
    this.maxLines = 1,
  });

  final TextEditingController? controller;
  final FocusNode? focusNode;
  final String? hintText;
  final String? labelText;
  final Widget? prefixIcon;
  final Widget? suffixIcon;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final Iterable<String>? autofillHints;
  final bool autocorrect;
  final bool enableSuggestions;
  final TextCapitalization textCapitalization;
  final int? maxLength;
  final bool enabled;
  final bool readOnly;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final VoidCallback? onEditingComplete;
  final VoidCallback? onTap;
  final bool autofocus;
  final TextAlignVertical? textAlignVertical;
  final TextAlign textAlign;
  final List<TextInputFormatter>? inputFormatters;
  final InputDecoration? decoration;
  final TextStyle? style;
  final Color? cursorColor;
  final int? minLines;
  final int maxLines;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = GfTheme.colorsOf(context);
    return TextField(
      onTap: onTap,
      controller: controller,
      focusNode: focusNode,
      enabled: enabled,
      readOnly: readOnly,
      obscureText: obscureText,
      keyboardType: keyboardType,
      textInputAction: textInputAction,
      autofillHints: autofillHints,
      autocorrect: autocorrect,
      enableSuggestions: enableSuggestions,
      textCapitalization: textCapitalization,
      textAlignVertical: textAlignVertical,
      maxLength: maxLength,
      autofocus: autofocus,
      textAlign: textAlign,
      minLines: minLines,
      maxLines: maxLines,
      onChanged: onChanged,
      onSubmitted: onSubmitted,
      onEditingComplete: onEditingComplete,
      inputFormatters: inputFormatters,
      style:
          style ??
          TextStyle(
            fontSize: 16,
            height: 1.4,
            color: enabled
                ? colors.baseContent
                : colors.baseContent.withValues(alpha: 0.38),
          ),
      cursorColor: cursorColor ?? theme.colorScheme.primary,
      decoration: (decoration ?? const InputDecoration()).copyWith(
        hintText: hintText ?? decoration?.hintText,
        labelText: labelText ?? decoration?.labelText,
        prefixIcon: prefixIcon ?? decoration?.prefixIcon,
        suffixIcon: suffixIcon ?? decoration?.suffixIcon,
        counterText: decoration?.counterText ?? (maxLength != null ? '' : null),
      ),
    );
  }
}
