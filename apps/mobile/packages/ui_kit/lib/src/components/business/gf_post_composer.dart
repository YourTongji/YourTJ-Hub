import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';

import '../../theme/gf_theme.dart';
import '../../twemoji/twemoji_text.dart';
import '../atoms/gf_loading_indicator.dart';
import '../gf_button.dart';
import '../gf_media_image.dart';
import '../gf_motion.dart';
import '../gf_symbol.dart';

/// Compact reply surface with a borderless growing editor and one action row.
/// Optional verification content appears only when supplied by the caller.
/// Input accessories (keyboard or sticker panel) live below it in a
/// [GfInputAccessorySlot]; switching them never changes this surface.
class GfPostComposer extends StatefulWidget {
  const GfPostComposer({
    super.key,
    required this.controller,
    required this.onPublish,
    required this.publishLabel,
    required this.hintText,
    this.focusNode,
    this.leading,
    this.status,
    this.hideKeyboardLabel,
    this.hidePanelLabel,
    this.onHideAccessory,
    this.onCollapse,
    this.collapseLabel,
    this.targetName,
    this.targetLabel,
    this.onCloseTarget,
    this.onPickImage,
    this.imageTooltip,
    this.onPickSticker,
    this.stickerTooltip,
    this.stickerOpen = false,
    this.compactEditor = false,
    this.expandEditor = false,
    this.preview,
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

  /// Docks into the input's leading edge: a ring in the surface color carves
  /// a socket around it, and it glides with the input's centre as lines are
  /// added, e.g. the publishing identity. A 44px square whose content fits a
  /// 32px circle.
  final Widget? leading;

  /// Short state beside the send action, e.g. draft saving; it shares the
  /// action row, so it never adds height or gets clipped by the editor.
  final Widget? status;
  final String? hideKeyboardLabel;

  /// Names the same hide action while the sticker panel is open.
  final String? hidePanelLabel;

  /// Dismisses the open keyboard or panel; defaults to removing focus.
  final VoidCallback? onHideAccessory;
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
  final bool stickerOpen;

  /// Bounds the rendered editor to one line, e.g. while a panel field and the
  /// keyboard share the space below. The field stays multiline: return keeps
  /// inserting newlines and longer drafts scroll inside it.
  final bool compactEditor;

  /// Fills the height the composer is given: the input grows into the space
  /// above the action row instead of stopping at four lines, for full-screen
  /// editing of longer text.
  final bool expandEditor;
  final Widget? preview;
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
  /// Where the input starts behind a docked [GfPostComposer.leading]; with
  /// the 38px socket centred in its 44px square, 3px of ring overlaps the
  /// input by 13px.
  static const double _dockInset = 28;
  static const double _dockSocket = 38;

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
    required Widget icon,
    required String? tooltip,
    required VoidCallback? onPressed,
  }) {
    final colors = GfTheme.colorsOf(context);
    return IconButton(
      tooltip: tooltip,
      onPressed: onPressed,
      icon: icon,
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

  /// Grouped content scrolls above a capped input; an expanded input fills
  /// the space instead.
  Widget _scrollUnless(bool expand, Widget child) =>
      expand ? child : SingleChildScrollView(reverse: true, child: child);

  Widget _fillUnless(bool capped, Widget child) =>
      capped ? child : Expanded(child: child);

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final bool expand = widget.expandEditor;
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
              Flexible(
                fit: expand ? FlexFit.tight : FlexFit.loose,
                child: _scrollUnless(
                  expand,
                  Column(
                    mainAxisSize: expand ? MainAxisSize.max : MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (widget.preview != null) widget.preview!,
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
                              child: GfText(
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
                                icon: const GfSymbol('x', size: 23),
                                tooltip: MaterialLocalizations.of(
                                  context,
                                ).closeButtonTooltip,
                                onPressed: widget.publishing
                                    ? null
                                    : widget.onCloseTarget,
                              ),
                          ],
                        ),
                      if (widget.imageUrl != null &&
                          widget.imageUrl!.isNotEmpty) ...[
                        Align(
                          alignment: Alignment.centerLeft,
                          child: Stack(
                            children: [
                              ClipRRect(
                                borderRadius: BorderRadius.circular(16),
                                child: Image(
                                  image: ResizeImage(
                                    GfMediaScope.imageProvider(
                                      context,
                                      widget.imageUrl!,
                                    ),
                                    policy: ResizeImagePolicy.fit,
                                    width:
                                        (176 *
                                                MediaQuery.devicePixelRatioOf(
                                                  context,
                                                ))
                                            .round(),
                                    height:
                                        (144 *
                                                MediaQuery.devicePixelRatioOf(
                                                  context,
                                                ))
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
                                      tapTargetSize:
                                          MaterialTapTargetSize.shrinkWrap,
                                      foregroundColor: colors.base100,
                                      backgroundColor: colors.baseContent
                                          .withValues(alpha: .7),
                                      shape: const CircleBorder(),
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                        const SizedBox(height: 8),
                      ],
                      if (widget.toolbar != null)
                        Padding(
                          padding: const EdgeInsets.only(top: 6),
                          child: widget.toolbar!,
                        ),
                      _fillUnless(
                        !expand,
                        Stack(
                          fit: expand ? StackFit.expand : StackFit.loose,
                          children: [
                            Padding(
                              padding: EdgeInsetsDirectional.only(
                                start: widget.leading == null ? 0 : _dockInset,
                              ),
                              child: AnimatedContainer(
                                key: const Key('reply-input-surface'),
                                duration: GfMotion.duration(
                                  context,
                                  GfMotion.press,
                                ),
                                curve: GfMotion.enterCurve,
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
                                  crossAxisAlignment: expand
                                      ? CrossAxisAlignment.stretch
                                      : CrossAxisAlignment.start,
                                  children: [
                                    Expanded(
                                      child: ConstrainedBox(
                                        constraints: widget.compactEditor
                                            ? BoxConstraints(
                                                maxHeight:
                                                    MediaQuery.textScalerOf(
                                                          context,
                                                        ).scale(16) *
                                                        1.4 +
                                                    22,
                                              )
                                            : const BoxConstraints(),
                                        child: TextField(
                                          controller: widget.controller,
                                          focusNode: _focus,
                                          readOnly: widget.publishing,
                                          keyboardType: TextInputType.multiline,
                                          textInputAction:
                                              TextInputAction.newline,
                                          expands: expand,
                                          maxLines: expand ? null : 4,
                                          minLines: expand ? null : 1,
                                          textAlignVertical: expand
                                              ? TextAlignVertical.top
                                              : null,
                                          style: GfTheme.typographyOf(context)
                                              .body
                                              .copyWith(
                                                fontSize: 16,
                                                height: 1.4,
                                              ),
                                          cursorColor: colors.primary,
                                          decoration: InputDecoration(
                                            hintText: widget.hintText,
                                            hintStyle: TextStyle(
                                              color: colors.iconMuted,
                                            ),
                                            border: InputBorder.none,
                                            enabledBorder: InputBorder.none,
                                            focusedBorder: InputBorder.none,
                                            disabledBorder: InputBorder.none,
                                            errorBorder: InputBorder.none,
                                            focusedErrorBorder:
                                                InputBorder.none,
                                            filled: false,
                                            isDense: true,
                                            constraints: const BoxConstraints(
                                              minHeight: 44,
                                            ),
                                            contentPadding:
                                                EdgeInsetsDirectional.fromSTEB(
                                                  widget.leading == null
                                                      ? 16
                                                      : 24,
                                                  11,
                                                  widget.onCollapse == null
                                                      ? 16
                                                      : 4,
                                                  11,
                                                ),
                                          ),
                                        ),
                                      ),
                                    ),
                                    if (widget.onCollapse != null)
                                      IconButton(
                                        key: const Key('reply-collapse'),
                                        onPressed: widget.publishing
                                            ? null
                                            : widget.onCollapse,
                                        tooltip:
                                            widget.collapseLabel ??
                                            MaterialLocalizations.of(
                                              context,
                                            ).closeButtonTooltip,
                                        icon: const GfSymbol(
                                          'chevron-down',
                                          size: 22,
                                        ),
                                        style: IconButton.styleFrom(
                                          fixedSize: const Size.square(44),
                                          tapTargetSize:
                                              MaterialTapTargetSize.shrinkWrap,
                                          foregroundColor: colors.iconMuted,
                                          padding: EdgeInsets.zero,
                                          shape: const CircleBorder(),
                                        ),
                                      ),
                                  ],
                                ),
                              ),
                            ),
                            if (widget.leading != null)
                              PositionedDirectional(
                                start: 0,
                                top: 0,
                                bottom: 0,
                                width: 44,
                                child: Center(
                                  child: _LayoutGlide(
                                    child: Stack(
                                      alignment: Alignment.center,
                                      children: [
                                        Container(
                                          width: _dockSocket,
                                          height: _dockSocket,
                                          decoration: ShapeDecoration(
                                            color: colors.base100,
                                            shape: const CircleBorder(),
                                          ),
                                        ),
                                        widget.leading!,
                                      ],
                                    ),
                                  ),
                                ),
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
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
                              child: GfProgressIndicator(strokeWidth: 2),
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
                      icon: AnimatedSwitcher(
                        duration: GfMotion.duration(
                          context,
                          GfMotion.selection,
                        ),
                        switchInCurve: GfMotion.enterCurve,
                        switchOutCurve: GfMotion.enterCurve,
                        transitionBuilder: (child, animation) => FadeTransition(
                          opacity: animation,
                          child: ScaleTransition(
                            scale: Tween<double>(
                              begin: .25,
                              end: 1,
                            ).animate(animation),
                            child: child,
                          ),
                        ),
                        child: GfSymbol(
                          widget.stickerOpen ? 'keyboard' : 'emoji-circle',
                          key: ValueKey<bool>(widget.stickerOpen),
                          size: 23,
                        ),
                      ),
                      tooltip: widget.stickerTooltip,
                      onPressed: widget.publishing
                          ? null
                          : widget.onPickSticker,
                    ),
                  // Always present so the row never reflows on a switch. The
                  // hide glyph (keyboard over a down chevron) never repeats the
                  // plain keyboard glyph of the switch beside it.
                  _tool(
                    icon: const GfSymbol('keyboard-hide', size: 23),
                    tooltip:
                        (widget.stickerOpen ? widget.hidePanelLabel : null) ??
                        widget.hideKeyboardLabel ??
                        MaterialLocalizations.of(context).closeButtonTooltip,
                    onPressed: _focus.hasFocus || widget.stickerOpen
                        ? widget.onHideAccessory ??
                              () =>
                                  FocusManager.instance.primaryFocus?.unfocus()
                        : null,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    // Send keeps its width; the status takes what is left.
                    child: LayoutBuilder(
                      builder: (context, box) => Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          if (widget.status != null) ...[
                            Flexible(child: widget.status!),
                            const SizedBox(width: 8),
                          ],
                          ConstrainedBox(
                            constraints: BoxConstraints(
                              maxWidth: math.max(
                                0,
                                box.maxWidth - (widget.status == null ? 0 : 8),
                              ),
                            ),
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
                        ],
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

/// Glides its child to each new layout position instead of jumping (FLIP):
/// a move is first painted as an offset back to the old place, which then
/// eases out, so a control centred on a line-by-line growing input rides its
/// centre. Positions are measured from the parent's bottom edge, where the
/// composer is anchored, so moving the whole composer (e.g. with the
/// keyboard) never glides. Instant with reduced motion.
class _LayoutGlide extends StatefulWidget {
  const _LayoutGlide({required this.child});

  final Widget child;

  @override
  State<_LayoutGlide> createState() => _LayoutGlideState();
}

class _LayoutGlideState extends State<_LayoutGlide>
    with SingleTickerProviderStateMixin {
  late final AnimationController _glide = AnimationController(
    vsync: this,
    value: 1,
  );

  @override
  void dispose() {
    _glide.dispose();
    super.dispose();
  }

  void _moved() {
    if (!mounted) return;
    _glide.duration = GfMotion.duration(context, GfMotion.layout);
    _glide.forward(from: 0);
  }

  @override
  Widget build(BuildContext context) => _GlideBox(
    progress: _glide,
    enabled: !GfMotion.reducedOf(context),
    onMoved: _moved,
    child: widget.child,
  );
}

class _GlideBox extends SingleChildRenderObjectWidget {
  const _GlideBox({
    required this.progress,
    required this.enabled,
    required this.onMoved,
    super.child,
  });

  final Animation<double> progress;
  final bool enabled;
  final VoidCallback onMoved;

  @override
  _RenderGlideBox createRenderObject(BuildContext context) =>
      _RenderGlideBox(progress, enabled, onMoved);

  @override
  void updateRenderObject(BuildContext context, _RenderGlideBox renderObject) {
    renderObject
      ..progress = progress
      ..enabled = enabled
      ..onMoved = onMoved;
  }
}

class _RenderGlideBox extends RenderProxyBox {
  _RenderGlideBox(this._progress, this.enabled, this.onMoved);

  Animation<double> _progress;
  set progress(Animation<double> value) {
    if (value == _progress) return;
    if (attached) {
      _progress.removeListener(markNeedsPaint);
      value.addListener(markNeedsPaint);
    }
    _progress = value;
  }

  bool enabled;
  VoidCallback onMoved;
  double? _lastY;
  double _from = 0;
  bool _restartPending = false;

  /// Remaining offset back towards the previous position.
  double get _shift => _restartPending
      ? _from
      : _from * (1 - GfMotion.enterCurve.transform(_progress.value));

  @override
  void attach(PipelineOwner owner) {
    super.attach(owner);
    _progress.addListener(markNeedsPaint);
  }

  @override
  void detach() {
    _progress.removeListener(markNeedsPaint);
    super.detach();
  }

  @override
  void paint(PaintingContext context, Offset offset) {
    final ParentData? data = parentData;
    final RenderObject? row = parent;
    final double y = data is BoxParentData && row is RenderBox
        ? data.offset.dy - row.size.height
        : 0;
    final double? last = _lastY;
    _lastY = y;
    if (!enabled) {
      _from = 0;
    } else if (last != null && (y - last).abs() >= .5) {
      _from = _shift + last - y;
      if (!_restartPending) {
        _restartPending = true;
        SchedulerBinding.instance.addPostFrameCallback((_) {
          _restartPending = false;
          onMoved();
        });
      }
    }
    super.paint(context, offset.translate(0, _shift));
  }

  @override
  bool hitTestChildren(BoxHitTestResult result, {required Offset position}) =>
      child != null &&
      result.addWithPaintOffset(
        offset: Offset(0, _shift),
        position: position,
        hitTest: (result, transformed) =>
            child!.hitTest(result, position: transformed),
      );

  @override
  void applyPaintTransform(RenderBox child, Matrix4 transform) {
    transform.translateByDouble(0, _shift, 0, 1);
  }
}
