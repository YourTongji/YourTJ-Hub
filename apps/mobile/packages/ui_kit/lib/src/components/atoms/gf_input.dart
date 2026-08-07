import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';

/// Text field aligned with web `.gf-input` (components.css): 1px line border,
/// radius field (8), base-100 fill, 14px text; on focus the border turns
/// primary and a 4px primary/20 ring appears (web `ring-4 ring-primary/20`).
class GfInput extends StatefulWidget {
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
    this.maxLength,
    this.enabled = true,
    this.onChanged,
    this.onSubmitted,
    this.autofocus = false,
    this.textAlignVertical,
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
  final int? maxLength;
  final bool enabled;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final bool autofocus;
  final TextAlignVertical? textAlignVertical;
  final int? minLines;
  final int maxLines;

  @override
  State<GfInput> createState() => _GfInputState();
}

class _GfInputState extends State<GfInput> {
  FocusNode? _internalFocusNode;
  bool _focused = false;

  FocusNode get _effectiveFocusNode =>
      widget.focusNode ?? (_internalFocusNode ??= FocusNode());

  @override
  void initState() {
    super.initState();
    _effectiveFocusNode.addListener(_onFocusChange);
  }

  @override
  void didUpdateWidget(GfInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      oldWidget.focusNode?.removeListener(_onFocusChange);
      _effectiveFocusNode.addListener(_onFocusChange);
    }
  }

  @override
  void dispose() {
    widget.focusNode?.removeListener(_onFocusChange);
    _internalFocusNode?.dispose();
    super.dispose();
  }

  void _onFocusChange() {
    if (!mounted) return;
    final bool focused = _effectiveFocusNode.hasFocus;
    if (focused != _focused) {
      setState(() => _focused = focused);
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);

    return AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      curve: Curves.easeOut,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(radii.field),
        boxShadow: _focused
            ? <BoxShadow>[
                BoxShadow(
                  color: colors.primary.withValues(alpha: 0.20),
                  blurRadius: 0,
                  spreadRadius: 4,
                ),
              ]
            : null,
      ),
      child: TextField(
        controller: widget.controller,
        focusNode: _effectiveFocusNode,
        enabled: widget.enabled,
        obscureText: widget.obscureText,
        keyboardType: widget.keyboardType,
        textInputAction: widget.textInputAction,
        maxLength: widget.maxLength,
        autofocus: widget.autofocus,
        textAlignVertical: widget.textAlignVertical,
        minLines: widget.minLines,
        maxLines: widget.maxLines,
        onChanged: widget.onChanged,
        onSubmitted: widget.onSubmitted,
        style: const TextStyle(fontSize: 14),
        onTapOutside: (_) => FocusManager.instance.primaryFocus?.unfocus(),
        decoration: InputDecoration(
          hintText: widget.hintText,
          labelText: widget.labelText,
          prefixIcon: widget.prefixIcon,
          suffixIcon: widget.suffixIcon,
          filled: true,
          fillColor: colors.base100,
          isDense: true,
          contentPadding: const EdgeInsets.symmetric(horizontal: 12),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(radii.field),
            borderSide: BorderSide(color: colors.line, width: borders.width),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(radii.field),
            borderSide: BorderSide(color: colors.line, width: borders.width),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(radii.field),
            borderSide: BorderSide(color: colors.primary, width: 1.5),
          ),
          disabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(radii.field),
            borderSide: BorderSide(
              color: colors.line.withValues(alpha: 0.5),
              width: borders.width,
            ),
          ),
          counterText: widget.maxLength != null ? '' : null,
        ),
      ),
    );
  }
}
