import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_symbol.dart';

/// Quiet, filled search surface shared by discovery, messages and courses.
/// Search is distinct from account/editor form fields. The clear action keeps
/// focus and notifies the owner once, including when it owns the controller.
class GfSearchField extends StatefulWidget {
  const GfSearchField({
    super.key,
    required this.hintText,
    required this.clearLabel,
    this.controller,
    this.maxLength,
    this.autofocus = false,
    this.onChanged,
    this.onSubmitted,
    this.onClear,
    this.readOnly = false,
    this.onTap,
  });

  final String hintText;
  final String clearLabel;
  final TextEditingController? controller;
  final int? maxLength;
  final bool autofocus;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final VoidCallback? onClear;
  final bool readOnly;
  final VoidCallback? onTap;

  @override
  State<GfSearchField> createState() => _GfSearchFieldState();
}

class _GfSearchFieldState extends State<GfSearchField> {
  final FocusNode _focus = FocusNode();
  TextEditingController? _owned;
  TextEditingController get _controller =>
      widget.controller ?? (_owned ??= TextEditingController());

  @override
  void dispose() {
    _owned?.dispose();
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final typography = GfTheme.typographyOf(context);
    const border = OutlineInputBorder(
      borderRadius: BorderRadius.all(Radius.circular(GfRadius.full)),
      borderSide: BorderSide.none,
    );
    return ValueListenableBuilder<TextEditingValue>(
      valueListenable: _controller,
      builder: (context, value, _) => TextField(
        controller: _controller,
        readOnly: widget.readOnly,
        onTap: widget.onTap,
        maxLength: widget.maxLength,
        focusNode: _focus,
        autofocus: widget.autofocus,
        autocorrect: false,
        textInputAction: TextInputAction.search,
        textAlignVertical: TextAlignVertical.center,
        onChanged: widget.onChanged,
        onSubmitted: widget.onSubmitted,
        style: typography.body.copyWith(color: colors.textPrimary),
        cursorColor: colors.primary,
        decoration: InputDecoration(
          hintText: widget.hintText,
          counterText: '',
          hintStyle: typography.body.copyWith(color: colors.textPlaceholder),
          filled: true,
          fillColor: colors.fill,
          isDense: true,
          constraints: const BoxConstraints(minHeight: GfControl.touch),
          contentPadding: const EdgeInsets.symmetric(
            horizontal: GfSpace.lg,
            vertical: 10,
          ),
          border: border,
          enabledBorder: border,
          focusedBorder: border,
          prefixIcon: Padding(
            padding: const EdgeInsets.only(left: 14, right: GfSpace.sm),
            child: GfSymbol(
              'search',
              size: GfIconSize.md - 2,
              color: colors.textSecondary,
            ),
          ),
          prefixIconConstraints: const BoxConstraints(
            minWidth: 40,
            minHeight: GfControl.touch,
          ),
          suffixIconConstraints: const BoxConstraints(
            minWidth: GfControl.touch,
            minHeight: GfControl.touch,
          ),
          suffixIcon: value.text.isEmpty
              ? null
              : IconButton(
                  tooltip: widget.clearLabel,
                  icon: Container(
                    width: 18,
                    height: 18,
                    decoration: BoxDecoration(
                      color: colors.textPlaceholder,
                      shape: BoxShape.circle,
                    ),
                    alignment: Alignment.center,
                    child: GfSymbol('x', size: 12, color: colors.canvas),
                  ),
                  onPressed: () {
                    _controller.clear();
                    _focus.requestFocus();
                    if (widget.onClear != null) {
                      widget.onClear!();
                    } else {
                      widget.onChanged?.call('');
                    }
                  },
                ),
        ),
      ),
    );
  }
}
