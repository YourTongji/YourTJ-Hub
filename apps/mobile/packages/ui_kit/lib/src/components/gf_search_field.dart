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
  });

  final String hintText;
  final String clearLabel;
  final TextEditingController? controller;
  final int? maxLength;
  final bool autofocus;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final VoidCallback? onClear;

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
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(28),
      borderSide: BorderSide.none,
    );
    return ValueListenableBuilder<TextEditingValue>(
      valueListenable: _controller,
      builder: (context, value, _) => TextField(
        controller: _controller,
        maxLength: widget.maxLength,
        focusNode: _focus,
        autofocus: widget.autofocus,
        autocorrect: false,
        textInputAction: TextInputAction.search,
        textAlignVertical: TextAlignVertical.center,
        onChanged: widget.onChanged,
        onSubmitted: widget.onSubmitted,
        style: TextStyle(fontSize: 16, color: colors.baseContent),
        cursorColor: colors.primary,
        decoration: InputDecoration(
          hintText: widget.hintText,
          counterText: '',
          hintStyle: TextStyle(fontSize: 16, color: colors.iconMuted),
          filled: true,
          fillColor: colors.base300,
          isDense: true,
          constraints: const BoxConstraints(minHeight: 48),
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 16,
            vertical: 12,
          ),
          border: border,
          enabledBorder: border,
          focusedBorder: border.copyWith(
            borderSide: BorderSide(color: colors.primary),
          ),
          prefixIcon: Padding(
            padding: const EdgeInsets.only(left: 16, right: 10),
            child: GfSymbol('search', size: 20, color: colors.iconMuted),
          ),
          prefixIconConstraints: const BoxConstraints(
            minWidth: 46,
            minHeight: 48,
          ),
          suffixIconConstraints: const BoxConstraints(
            minWidth: 44,
            minHeight: 44,
          ),
          suffixIcon: value.text.isEmpty
              ? null
              : IconButton(
                  tooltip: widget.clearLabel,
                  icon: GfSymbol('x', size: 18, color: colors.iconMuted),
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
