import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// External handle for a [GfScrollToTop] surface.
///
/// It lets a navigation shell implement the familiar "tap the active tab to
/// return to the beginning" behaviour without owning the page's scroll view.
class GfScrollToTopController {
  Future<void> Function()? _scrollToTop;

  bool get isAttached => _scrollToTop != null;

  Future<void> scrollToTop() async {
    await _scrollToTop?.call();
  }

  void _attach(Future<void> Function() callback) {
    _scrollToTop = callback;
  }

  void _detach() {
    _scrollToTop = null;
  }
}

typedef GfScrollViewBuilder =
    Widget Function(BuildContext context, ScrollController controller);

/// Owns a page scroll controller and reveals a quiet return-to-top action.
///
/// The button only appears after [threshold], never changes scroll position
/// without an explicit user action, and avoids animated motion when the
/// platform requests reduced motion.
class GfScrollToTop extends StatefulWidget {
  const GfScrollToTop({
    super.key,
    required this.builder,
    required this.semanticLabel,
    this.controller,
    this.showButton = true,
    this.threshold = 300,
    this.bottomInset = 16,
    this.rightInset = 16,
  });

  final GfScrollViewBuilder builder;
  final GfScrollToTopController? controller;
  final bool showButton;
  final double threshold;
  final double bottomInset;
  final double rightInset;
  final String semanticLabel;

  @override
  State<GfScrollToTop> createState() => _GfScrollToTopState();
}

class _GfScrollToTopState extends State<GfScrollToTop> {
  late final ScrollController _scrollController;
  bool _visible = false;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController()..addListener(_handleScroll);
    widget.controller?._attach(_scrollToTop);
  }

  @override
  void didUpdateWidget(covariant GfScrollToTop oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller?._detach();
      widget.controller?._attach(_scrollToTop);
    }
    if (!widget.showButton && _visible) {
      _visible = false;
    }
  }

  @override
  void dispose() {
    widget.controller?._detach();
    _scrollController
      ..removeListener(_handleScroll)
      ..dispose();
    super.dispose();
  }

  void _handleScroll() {
    if (!_scrollController.hasClients) return;
    final bool visible =
        widget.showButton && _scrollController.offset > widget.threshold;
    if (visible != _visible && mounted) {
      setState(() => _visible = visible);
    }
  }

  Future<void> _scrollToTop() async {
    if (!_scrollController.hasClients) return;
    final double top = _scrollController.position.minScrollExtent;
    final bool disableAnimations =
        MediaQuery.maybeOf(context)?.disableAnimations ?? false;
    if (disableAnimations) {
      _scrollController.jumpTo(top);
      return;
    }
    await _scrollController.animateTo(
      top,
      duration: GfMotion.standard,
      curve: GfMotion.standardEase,
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfShadows shadows = GfTheme.shadowsOf(context);

    return Stack(
      fit: StackFit.expand,
      children: <Widget>[
        widget.builder(context, _scrollController),
        Positioned(
          right: widget.rightInset,
          bottom: widget.bottomInset,
          child: AnimatedSwitcher(
            duration: GfMotion.fast,
            switchInCurve: GfMotion.standardEase,
            switchOutCurve: GfMotion.standardEase,
            transitionBuilder: (Widget child, Animation<double> animation) {
              return FadeTransition(opacity: animation, child: child);
            },
            child: !_visible
                ? const SizedBox.shrink(key: ValueKey<String>('hidden'))
                : DecoratedBox(
                    key: const ValueKey<String>('visible'),
                    decoration: BoxDecoration(
                      color: colors.base100,
                      shape: BoxShape.circle,
                      border: Border.all(color: colors.line),
                      boxShadow: shadows.floating,
                    ),
                    child: IconButton(
                      onPressed: _scrollToTop,
                      tooltip: widget.semanticLabel,
                      icon: Icon(
                        Icons.arrow_upward_rounded,
                        color: colors.primary,
                      ),
                      iconSize: 21,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints.tightFor(
                        width: 44,
                        height: 44,
                      ),
                    ),
                  ),
          ),
        ),
      ],
    );
  }
}
