import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_button.dart';
import '../gf_symbol.dart';

/// Compact reply surface with a borderless growing editor and one action row.
/// Optional verification content appears only when supplied by the caller.
class GfPostComposer extends StatefulWidget {
  const GfPostComposer({
    super.key,
    required this.controller,
    required this.onPublish,
    required this.publishLabel,
    required this.hintText,
    this.focusNode,
    this.hideKeyboardLabel,
    this.onCollapse,
    this.collapseLabel,
    this.targetName,
    this.targetLabel,
    this.onCloseTarget,
    this.onPickImage,
    this.imageTooltip,
    this.onPickSticker,
    this.stickerTooltip,
    this.imageUrl,
    this.onRemoveImage,
    this.removeImageTooltip,
    this.uploading = false,
    this.publishing = false,
    this.canPublish = true,
    this.toolbar,
  }) : assert(onPickImage == null || imageTooltip != null),
       assert(onRemoveImage == null || removeImageTooltip != null);

  final TextEditingController controller;
  final FocusNode? focusNode;
  final String? hideKeyboardLabel;
  final VoidCallback? onCollapse;
  final String? collapseLabel;
  final VoidCallback onPublish;

  /// When replying to a user, the target name and localized label shown in the
  /// reference bar.
  final String? targetName;
  final String? targetLabel;
  final VoidCallback? onCloseTarget;

  /// Image upload action, localized accessibility labels, and optional preview.
  final VoidCallback? onPickImage;
  final String? imageTooltip;
  final VoidCallback? onPickSticker;
  final String? stickerTooltip;
  final String? imageUrl;
  final VoidCallback? onRemoveImage;
  final String? removeImageTooltip;
  final bool uploading;

  final bool publishing;
  final bool canPublish;
  final String publishLabel;
  final String hintText;

  /// Optional extra toolbar widgets (bold/italic/link/quote/code…).
  final Widget? toolbar;

  @override
  State<GfPostComposer> createState() => _GfPostComposerState();
}

class _GfPostComposerState extends State<GfPostComposer> {
  final FocusNode _ownedFocus = FocusNode();
  FocusNode get _focus => widget.focusNode ?? _ownedFocus;

  @override
  void initState() {
    super.initState();
    _focus.addListener(_focusChanged);
  }

  void _focusChanged() {
    if (mounted) setState(() {});
  }

  @override
  void didUpdateWidget(covariant GfPostComposer oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.focusNode != widget.focusNode) {
      (oldWidget.focusNode ?? _ownedFocus).removeListener(_focusChanged);
      _focus.addListener(_focusChanged);
    }
  }

  @override
  void dispose() {
    _focus.removeListener(_focusChanged);
    _ownedFocus.dispose();
    super.dispose();
  }

  Widget _tool({
    required String symbol,
    required String? tooltip,
    required VoidCallback? onPressed,
  }) {
    final colors = GfTheme.colorsOf(context);
    return IconButton(
      tooltip: tooltip,
      onPressed: onPressed,
      icon: GfSymbol(symbol, size: 23),
      style: IconButton.styleFrom(
        fixedSize: const Size.square(44),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        foregroundColor: colors.primary,
        disabledForegroundColor: colors.iconMuted.withValues(alpha: .4),
        padding: EdgeInsets.zero,
        shape: const CircleBorder(),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Material(
      key: const Key('reply-composer-surface'),
      color: colors.base100,
      child: DecoratedBox(
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: colors.line)),
        ),
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (widget.targetName != null)
                Row(
                  children: [
                    GfSymbol(
                      'corner-down-left',
                      size: 17,
                      color: colors.iconMuted,
                    ),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(
                        widget.targetLabel ?? widget.targetName!,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: GfTheme.typographyOf(
                          context,
                        ).caption.copyWith(color: colors.iconMuted),
                      ),
                    ),
                    if (widget.onCloseTarget != null)
                      _tool(
                        symbol: 'x',
                        tooltip: MaterialLocalizations.of(
                          context,
                        ).closeButtonTooltip,
                        onPressed: widget.publishing
                            ? null
                            : widget.onCloseTarget,
                      ),
                  ],
                ),
              if (widget.imageUrl != null && widget.imageUrl!.isNotEmpty) ...[
                Align(
                  alignment: Alignment.centerLeft,
                  child: Stack(
                    children: [
                      ClipRRect(
                        borderRadius: BorderRadius.circular(16),
                        child: Image(
                          image: ResizeImage(
                            NetworkImage(widget.imageUrl!),
                            policy: ResizeImagePolicy.fit,
                            width:
                                (176 * MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                            height:
                                (144 * MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                          ),
                          width: 112,
                          height: 88,
                          fit: BoxFit.cover,
                          errorBuilder: (_, _, _) => Container(
                            width: 112,
                            height: 88,
                            color: colors.base200,
                            alignment: Alignment.center,
                            child: GfSymbol(
                              'image-off',
                              color: colors.iconMuted,
                            ),
                          ),
                        ),
                      ),
                      if (widget.onRemoveImage != null)
                        Positioned(
                          right: 0,
                          top: 0,
                          child: IconButton(
                            tooltip: widget.removeImageTooltip,
                            onPressed: widget.publishing
                                ? null
                                : widget.onRemoveImage,
                            icon: const GfSymbol('x', size: 18),
                            style: IconButton.styleFrom(
                              fixedSize: const Size.square(44),
                              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                              foregroundColor: colors.base100,
                              backgroundColor: colors.baseContent.withValues(
                                alpha: .7,
                              ),
                              shape: const CircleBorder(),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
                const SizedBox(height: 8),
              ],
              AnimatedContainer(
                key: const Key('reply-input-surface'),
                duration: const Duration(milliseconds: 140),
                decoration: BoxDecoration(
                  color: colors.base200,
                  borderRadius: BorderRadius.circular(24),
                  border: Border.all(
                    color: _focus.hasFocus
                        ? colors.primary.withValues(alpha: .32)
                        : Colors.transparent,
                  ),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: TextField(
                        controller: widget.controller,
                        focusNode: _focus,
                        readOnly: widget.publishing,
                        keyboardType: TextInputType.multiline,
                        textInputAction: TextInputAction.newline,
                        maxLines: 4,
                        minLines: 1,
                        style: GfTheme.typographyOf(
                          context,
                        ).body.copyWith(fontSize: 16, height: 1.4),
                        cursorColor: colors.primary,
                        decoration: InputDecoration(
                          hintText: widget.hintText,
                          hintStyle: TextStyle(color: colors.iconMuted),
                          border: InputBorder.none,
                          enabledBorder: InputBorder.none,
                          focusedBorder: InputBorder.none,
                          disabledBorder: InputBorder.none,
                          errorBorder: InputBorder.none,
                          focusedErrorBorder: InputBorder.none,
                          filled: false,
                          isDense: true,
                          constraints: const BoxConstraints(minHeight: 44),
                          contentPadding: EdgeInsets.fromLTRB(
                            16,
                            11,
                            widget.onCollapse == null ? 16 : 4,
                            11,
                          ),
                        ),
                      ),
                    ),
                    if (widget.onCollapse != null)
                      IconButton(
                        key: const Key('reply-collapse'),
                        onPressed: widget.publishing ? null : widget.onCollapse,
                        tooltip:
                            widget.collapseLabel ??
                            MaterialLocalizations.of(
                              context,
                            ).closeButtonTooltip,
                        icon: const GfSymbol('chevron-down', size: 22),
                        style: IconButton.styleFrom(
                          fixedSize: const Size.square(44),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                          foregroundColor: colors.iconMuted,
                          padding: EdgeInsets.zero,
                          shape: const CircleBorder(),
                        ),
                      ),
                  ],
                ),
              ),
              if (widget.toolbar != null)
                Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: widget.toolbar!,
                ),
              const SizedBox(height: 4),
              Row(
                children: [
                  if (widget.onPickImage != null)
                    IconButton(
                      tooltip: widget.imageTooltip,
                      onPressed: widget.uploading || widget.publishing
                          ? null
                          : widget.onPickImage,
                      icon: widget.uploading
                          ? const SizedBox.square(
                              dimension: 20,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const GfSymbol('gallery', size: 23),
                      style: IconButton.styleFrom(
                        fixedSize: const Size.square(44),
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        foregroundColor: colors.primary,
                        disabledForegroundColor: colors.iconMuted.withValues(
                          alpha: .4,
                        ),
                        padding: EdgeInsets.zero,
                        shape: const CircleBorder(),
                      ),
                    ),
                  if (widget.onPickSticker != null)
                    _tool(
                      symbol: 'emoji-circle',
                      tooltip: widget.stickerTooltip,
                      onPressed: widget.publishing
                          ? null
                          : widget.onPickSticker,
                    ),
                  _tool(
                    symbol: 'keyboard-hide',
                    tooltip:
                        widget.hideKeyboardLabel ??
                        MaterialLocalizations.of(context).closeButtonTooltip,
                    onPressed: () =>
                        FocusManager.instance.primaryFocus?.unfocus(),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Align(
                      alignment: Alignment.centerRight,
                      child: GfButton(
                        label: widget.publishLabel,
                        size: GfButtonSize.medium,
                        loading: widget.publishing,
                        onPressed:
                            widget.publishing ||
                                widget.uploading ||
                                !widget.canPublish
                            ? null
                            : widget.onPublish,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
