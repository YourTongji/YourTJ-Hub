import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'gf_scroll_to_top.dart';

/// Supplies a page's main scroll controller to Flutter's iOS status-bar
/// scroll-to-top handling in [Scaffold].
///
/// The wrapper belongs above the page's [Scaffold], because Scaffold reads its
/// [PrimaryScrollController] from its own context. A [GfScrollToTopController]
/// keeps the selected controller in sync when a tab changes or its scroll view
/// mounts. The explicit scroll view still owns its controller and offset.
class GfStatusBarScrollToTop extends StatefulWidget {
  const GfStatusBarScrollToTop({
    super.key,
    required this.child,
    this.controller,
    this.scrollController,
    this.onStatusBarTap,
  }) : assert(controller == null || scrollController == null);

  final Widget child;
  final GfScrollToTopController? controller;

  /// Use for a page whose primary scroll view already owns its controller.
  final ScrollController? scrollController;

  /// Optional bookkeeping callback, invoked before [Scaffold] scrolls.
  ///
  /// Keep this for side effects that a programmatic status-bar scroll must
  /// suppress, such as edge-triggered pagination. The framework Scaffold still
  /// owns the actual scroll-to-top behavior.
  final VoidCallback? onStatusBarTap;

  @override
  State<GfStatusBarScrollToTop> createState() =>
      _GfStatusBarScrollToTopState();
}

class _GfStatusBarScrollToTopState extends State<GfStatusBarScrollToTop>
    with WidgetsBindingObserver {
  ScrollController? _unattachedController;
  bool _observingStatusBar = false;

  @override
  void initState() {
    super.initState();
    _syncStatusBarObserver();
  }

  @override
  void didUpdateWidget(covariant GfStatusBarScrollToTop oldWidget) {
    super.didUpdateWidget(oldWidget);
    _syncStatusBarObserver();
  }

  @override
  void dispose() {
    if (_observingStatusBar) {
      WidgetsBinding.instance.removeObserver(this);
    }
    _unattachedController?.dispose();
    super.dispose();
  }

  bool get _shouldObserveStatusBar =>
      widget.onStatusBarTap != null &&
      !kIsWeb &&
      defaultTargetPlatform == TargetPlatform.iOS;

  void _syncStatusBarObserver() {
    final bool shouldObserve = _shouldObserveStatusBar;
    if (shouldObserve == _observingStatusBar) return;
    _observingStatusBar = shouldObserve;
    if (shouldObserve) {
      // This wrapper is above Scaffold, so its bookkeeping runs before the
      // Scaffold's built-in handler receives the same iOS event.
      WidgetsBinding.instance.addObserver(this);
    } else {
      WidgetsBinding.instance.removeObserver(this);
    }
  }

  @override
  void handleStatusBarTap() {
    if (!_shouldObserveStatusBar ||
        !TickerMode.valuesOf(context).enabled ||
        !(ModalRoute.of(context)?.isCurrent ?? true)) {
      return;
    }
    final ScrollController? controller =
        widget.scrollController ?? widget.controller?.scrollController;
    if (controller?.hasClients == true &&
        controller!.position.pixels != 0) {
      widget.onStatusBarTap?.call();
    }
  }

  @override
  Widget build(BuildContext context) {
    if (kIsWeb ||
        defaultTargetPlatform != TargetPlatform.iOS ||
        (widget.controller == null && widget.scrollController == null)) {
      return widget.child;
    }

    final ScrollController? directController = widget.scrollController;
    if (directController != null) {
      return _withPrimaryController(directController, widget.child);
    }

    final GfScrollToTopController controller = widget.controller!;
    return AnimatedBuilder(
      animation: controller,
      child: widget.child,
      builder: (BuildContext context, Widget? child) =>
          _withPrimaryController(
            controller.scrollController ??
                (_unattachedController ??= ScrollController()),
            child!,
          ),
    );
  }

  Widget _withPrimaryController(ScrollController controller, Widget child) =>
      PrimaryScrollController(
        controller: controller,
        // GfScrollToTop's primary list already has an explicit controller.
        // Avoid making unrelated vertical scroll views in the page share it.
        automaticallyInheritForPlatforms: const <TargetPlatform>{},
        child: child,
      );
}
