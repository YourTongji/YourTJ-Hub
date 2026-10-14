import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';

/// Vertical breathing room kept between the caret line and the viewport edges
/// when the follower pulls the caret back into view; the line being typed
/// should never sit flush against an edge.
const double _caretFollowInset = 12;

/// Keeps a `scrollable: false` Quill editor's caret inside the host page's
/// scroll view while the editor has focus.
///
/// flutter_quill only reveals the caret on its own when the editor owns a
/// scroll view (`scrollable: true` or an attached `scrollController`). Editors
/// that hand scrolling to an enclosing [Scrollable] — the publish page and any
/// non-fill `RichMarkdownEditor` host — get no following at all: upstream's
/// `_showCaretOnScreen` bails out in that shape, and `showOnScreen` propagation
/// is platform-dependent. This widget bridges the gap: after selection or
/// document changes it measures the caret line through
/// [EditorState.renderEditor] and nudges the nearest vertical scrollable so the
/// line stays visible.
class EditorCaretFollower extends StatefulWidget {
  const EditorCaretFollower({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.editorKey,
    required this.child,
  });

  final QuillController controller;
  final FocusNode focusNode;

  /// Key of the [QuillEditorState] whose render editor measures the caret.
  final GlobalKey<EditorState> editorKey;

  final Widget child;

  @override
  State<EditorCaretFollower> createState() => _EditorCaretFollowerState();
}

class _EditorCaretFollowerState extends State<EditorCaretFollower>
    with WidgetsBindingObserver {
  /// One reveal per frame: controller notifications arrive mid-gesture and
  /// mid-IME update, and the caret must be measured after the new line paints.
  bool _scheduled = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    widget.controller.addListener(_scheduleReveal);
  }

  @override
  void didUpdateWidget(EditorCaretFollower oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_scheduleReveal);
      widget.controller.addListener(_scheduleReveal);
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.controller.removeListener(_scheduleReveal);
    super.dispose();
  }

  /// Keyboard open/close and rotation resize the viewport under a stationary
  /// caret, exactly like editable text handles viewport metrics.
  @override
  void didChangeMetrics() => _scheduleReveal();

  void _scheduleReveal() {
    if (_scheduled) return;
    _scheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _scheduled = false;
      if (mounted) _revealCaret();
    });
  }

  void _revealCaret() {
    if (!widget.focusNode.hasFocus) return;
    final EditorState? editorState = widget.editorKey.currentState;
    final ScrollableState? scrollable = Scrollable.maybeOf(context);
    if (editorState == null || scrollable == null) return;
    final ScrollPosition position = scrollable.position;
    if (!position.hasPixels ||
        !position.hasViewportDimension ||
        position.axisDirection != AxisDirection.down) {
      return;
    }
    final RenderObject? viewport = scrollable.context.findRenderObject();
    if (viewport is! RenderBox || !viewport.attached || !viewport.hasSize) {
      return;
    }
    final RenderEditor renderEditor = editorState.renderEditor;
    if (!renderEditor.attached || !renderEditor.hasSize) return;
    final TextSelection selection = widget.controller.selection;
    if (!selection.isValid) return;

    // The caret line's rect in viewport coordinates; the viewport's own scroll
    // offset is part of the transform, so the rect is relative to what the
    // user currently sees.
    final Rect caret = MatrixUtils.transformRect(
      renderEditor.getTransformTo(viewport),
      renderEditor.getLocalRectForCaret(selection.extent),
    );

    final double viewportHeight = position.viewportDimension;
    final double lowerBound = viewportHeight - _caretFollowInset;
    double target = position.pixels;
    if (caret.bottom > lowerBound) {
      target += caret.bottom - lowerBound;
    } else if (caret.top < _caretFollowInset) {
      target += caret.top - _caretFollowInset;
    } else {
      return;
    }
    target = target.clamp(position.minScrollExtent, position.maxScrollExtent);
    if ((target - position.pixels).abs() < 0.5) return;
    // Mirrors flutter_quill's own caret follow: short, quick, interruptible.
    unawaited(_animateTo(position, target));
  }

  Future<void> _animateTo(ScrollPosition position, double target) async {
    await position.animateTo(
      target,
      duration: const Duration(milliseconds: 120),
      curve: Curves.fastOutSlowIn,
    );
    if (!mounted) return;
    // An interrupted (user drag) or clamped scroll lands away from the target;
    // only a completed one is worth re-measuring. Content above the caret can
    // still grow while the animation runs — the publish page's draft-status
    // row appears one frame after the edit that triggered this reveal — so
    // check once more on the settled layout instead of trusting the old one.
    if ((position.pixels - target).abs() > 0.5) return;
    _scheduleReveal();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
