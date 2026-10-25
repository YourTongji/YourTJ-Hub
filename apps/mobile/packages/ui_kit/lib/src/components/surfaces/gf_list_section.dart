import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';

/// Inset group on the grouped canvas: optional header, a rounded card holding
/// rows separated by hairlines, and an optional footer.
///
/// Settings, account security, profile editing and other form-like pages use
/// it with `GfColorRoles.canvasGrouped` as the page background, so every
/// grouped list shares one radius, inset and divider alignment.
class GfListSection extends StatelessWidget {
  const GfListSection({
    super.key,
    required this.children,
    this.header,
    this.footer,
    this.dividerIndent = GfSpace.lg,
    this.margin = const EdgeInsets.symmetric(horizontal: GfSpace.gutter),
  });

  final List<Widget> children;

  /// Short label above the card.
  final String? header;

  /// Explanation below the card.
  final String? footer;

  /// Leading inset of the separators. Use [iconRowIndent] when rows carry a
  /// leading symbol so the hairline starts at the title.
  final double dividerIndent;

  final EdgeInsetsGeometry margin;

  /// Separator inset for rows with a 24px leading symbol.
  static const double iconRowIndent = GfSpace.lg + GfIconSize.lg + GfSpace.md;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final double hairline = gfHairline(context);
    final TextStyle noteStyle = typography.caption.copyWith(
      color: colors.textSecondary,
    );

    return Padding(
      padding: margin,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          if (header != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(
                GfSpace.lg,
                0,
                GfSpace.lg,
                GfSpace.sm,
              ),
              child: Semantics(
                header: true,
                child: GfText(
                  header!,
                  style: noteStyle.copyWith(fontWeight: FontWeight.w500),
                ),
              ),
            ),
          Material(
            color: colors.surfaceGrouped,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(GfRadius.lg),
            ),
            clipBehavior: Clip.antiAlias,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisSize: MainAxisSize.min,
              children: [
                for (int i = 0; i < children.length; i++) ...[
                  if (i > 0)
                    Padding(
                      padding: EdgeInsetsDirectional.only(start: dividerIndent),
                      child: Container(
                        height: hairline,
                        color: colors.separator,
                      ),
                    ),
                  children[i],
                ],
              ],
            ),
          ),
          if (footer != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(
                GfSpace.lg,
                GfSpace.sm,
                GfSpace.lg,
                0,
              ),
              child: GfText(footer!, style: noteStyle),
            ),
        ],
      ),
    );
  }
}

/// Labelled text entry for inset groups: a small label above a borderless
/// field, so a form reads as one card instead of a stack of boxes.
///
/// Wraps a native [TextFormField]; validation messages appear under the field
/// in the error colour and the label turns primary while focused.
class GfFieldRow extends StatefulWidget {
  const GfFieldRow({
    super.key,
    required this.label,
    this.controller,
    this.hintText,
    this.leading,
    this.trailing,
    this.minLines = 1,
    this.maxLines = 1,
    this.maxLength,
    this.keyboardType,
    this.textInputAction,
    this.enabled = true,
    this.autocorrect = true,
    this.validator,
    this.onChanged,
    this.style,
    this.inputFormatters,
    this.fieldKey,
  });

  final String label;

  /// Key for the inner [TextFormField], so tests and forms can address the
  /// native field directly.
  final Key? fieldKey;
  final TextEditingController? controller;
  final String? hintText;

  /// Leading mark beside the field (provider icon, prefix).
  final Widget? leading;

  /// Trailing accessory inside the row (clear, visibility, unit).
  final Widget? trailing;
  final int minLines;
  final int maxLines;

  /// Shows a live `n/max` counter beside the label.
  final int? maxLength;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final bool enabled;
  final bool autocorrect;
  final FormFieldValidator<String>? validator;
  final ValueChanged<String>? onChanged;

  /// Overrides the body text style (e.g. a larger display name).
  final TextStyle? style;
  final List<TextInputFormatter>? inputFormatters;

  @override
  State<GfFieldRow> createState() => _GfFieldRowState();
}

class _GfFieldRowState extends State<GfFieldRow> {
  final FocusNode _focus = FocusNode();
  TextEditingController? _owned;
  TextEditingController get _controller =>
      widget.controller ?? (_owned ??= TextEditingController());

  @override
  void initState() {
    super.initState();
    _focus.addListener(_onFocus);
  }

  void _onFocus() => setState(() {});

  @override
  void dispose() {
    _focus.removeListener(_onFocus);
    _focus.dispose();
    _owned?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final bool focused = _focus.hasFocus;
    final TextStyle labelStyle = typography.caption.copyWith(
      fontWeight: FontWeight.w500,
      color: focused ? colors.primary : colors.textSecondary,
    );

    return InkWell(
      onTap: widget.enabled ? _focus.requestFocus : null,
      splashFactory: NoSplash.splashFactory,
      highlightColor: Colors.transparent,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(
          GfSpace.lg,
          10,
          GfSpace.lg,
          GfSpace.sm,
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (widget.leading != null) ...[
              Padding(
                padding: const EdgeInsets.only(top: 14),
                child: SizedBox(
                  width: GfIconSize.lg,
                  child: Center(child: widget.leading),
                ),
              ),
              const SizedBox(width: GfSpace.md),
            ],
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Expanded(child: GfText(widget.label, style: labelStyle)),
                      if (widget.maxLength != null)
                        ValueListenableBuilder<TextEditingValue>(
                          valueListenable: _controller,
                          builder: (context, value, _) => GfText(
                            '${value.text.characters.length}/${widget.maxLength}',
                            style: typography.caption.copyWith(
                              color:
                                  value.text.characters.length >
                                      widget.maxLength!
                                  ? colors.error
                                  : colors.textPlaceholder,
                            ),
                          ),
                        ),
                    ],
                  ),
                  TextFormField(
                    key: widget.fieldKey,
                    controller: _controller,
                    focusNode: _focus,
                    enabled: widget.enabled,
                    minLines: widget.minLines,
                    maxLines: widget.maxLines,
                    keyboardType:
                        widget.keyboardType ??
                        (widget.maxLines > 1 ? TextInputType.multiline : null),
                    textInputAction: widget.textInputAction,
                    autocorrect: widget.autocorrect,
                    inputFormatters: widget.inputFormatters,
                    validator: widget.validator,
                    onChanged: widget.onChanged,
                    cursorColor: colors.primary,
                    style: (widget.style ?? typography.body).copyWith(
                      color: widget.enabled
                          ? colors.textPrimary
                          : colors.textDisabled,
                    ),
                    decoration: InputDecoration(
                      hintText: widget.hintText,
                      hintStyle: (widget.style ?? typography.body).copyWith(
                        color: colors.textPlaceholder,
                      ),
                      filled: false,
                      isDense: true,
                      constraints: const BoxConstraints(),
                      contentPadding: const EdgeInsets.symmetric(vertical: 4),
                      border: InputBorder.none,
                      enabledBorder: InputBorder.none,
                      focusedBorder: InputBorder.none,
                      disabledBorder: InputBorder.none,
                      errorBorder: InputBorder.none,
                      focusedErrorBorder: InputBorder.none,
                      errorStyle: typography.caption.copyWith(
                        color: colors.error,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            if (widget.trailing != null) ...[
              const SizedBox(width: GfSpace.sm),
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: widget.trailing,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
