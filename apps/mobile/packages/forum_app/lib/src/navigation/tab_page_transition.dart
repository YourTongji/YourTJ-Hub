import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

class TabPageTransition extends StatefulWidget {
  const TabPageTransition({
    super.key,
    required this.index,
    required this.length,
    required this.pageBuilder,
    this.chromeHidden = false,
    this.retainInactivePages = true,
    this.pageKey,
  });

  final int index;
  final int length;
  final Widget Function(int index, bool chromeHidden) pageBuilder;
  final bool chromeHidden;
  final bool retainInactivePages;
  final Object Function(int index)? pageKey;

  @override
  State<TabPageTransition> createState() => _TabPageTransitionState();
}

class _TabPageTransitionState extends State<TabPageTransition> {
  ValueListenable<GfTabSwipeProgress>? _progress;
  final Map<int, Widget> _pages = {};
  final Map<int, GlobalKey> _pageKeys = {};
  final Map<int, Object> _identities = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final progress = GfTabSwipeProgressScope.maybeOf(context);
    if (progress == _progress) return;
    _progress?.removeListener(_onProgressChanged);
    _progress = progress;
    _progress?.addListener(_onProgressChanged);
    _refreshVisiblePages();
  }

  @override
  void didUpdateWidget(covariant TabPageTransition oldWidget) {
    super.didUpdateWidget(oldWidget);
    _pruneChangedPages();
    if (oldWidget.chromeHidden != widget.chromeHidden) {
      _refreshAllPages();
    } else {
      _refreshVisiblePages();
    }
    _pruneInactivePages();
  }

  void _onProgressChanged() {
    if (!mounted) return;
    setState(() {
      _ensureVisiblePages();
      _pruneInactivePages();
    });
  }

  void _pruneInactivePages() {
    if (widget.retainInactivePages) return;
    final progress = _progress?.value;
    final target = progress == null ? null : _targetIndex(progress);
    final keep = <int>{
      if (widget.index >= 0 && widget.index < widget.length) widget.index,
      ?target,
      if (progress?.originIndex case final origin?
          when target != null && origin != widget.index)
        origin,
    };
    for (final index in _pages.keys.toList()) {
      if (keep.contains(index)) continue;
      _pages.remove(index);
      _pageKeys.remove(index);
      _identities.remove(index);
    }
  }

  int? _targetIndex(GfTabSwipeProgress progress) {
    final target =
        progress.targetIndex ??
        (progress.offset == 0
            ? null
            : progress.originIndex + progress.offset.sign.toInt());
    if (target == null ||
        progress.originIndex < 0 ||
        progress.originIndex >= widget.length ||
        target < 0 ||
        target >= widget.length ||
        target == progress.originIndex) {
      return null;
    }
    return target;
  }

  void _ensureIdentity(int index) {
    final identity = widget.pageKey?.call(index) ?? index;
    final previous = _identities[index];
    if (previous != null && previous != identity) {
      _pages.remove(index);
      _pageKeys.remove(index);
    }
    _identities[index] = identity;
  }

  Widget _page(int index) {
    _ensureIdentity(index);
    return _pages.putIfAbsent(
      index,
      () => widget.pageBuilder(index, widget.chromeHidden),
    );
  }

  GlobalKey _pageKey(int index) => _pageKeys.putIfAbsent(
    index,
    () => GlobalKey(debugLabel: 'root-tab-page-$index'),
  );

  void _pruneChangedPages() {
    for (final index in _pages.keys.toList()) {
      if (index < 0 || index >= widget.length) {
        _pages.remove(index);
        _pageKeys.remove(index);
        _identities.remove(index);
        continue;
      }
      _ensureIdentity(index);
    }
  }

  void _ensureVisiblePages() {
    if (widget.index >= 0 && widget.index < widget.length) {
      _page(widget.index);
    }
    final progress = _progress?.value;
    if (progress == null) return;
    final target = GfMotion.reducedOf(context) ? null : _targetIndex(progress);
    if (target == null) return;
    _page(progress.originIndex);
    _page(target);
  }

  void _refreshVisiblePages() {
    final progress = _progress?.value;
    final target = progress == null ? null : _targetIndex(progress);
    final current = widget.index >= 0 && widget.index < widget.length
        ? widget.index
        : null;
    final visible = <int>{
      ?current,
      ?target,
      if (progress != null &&
          target != null &&
          progress.originIndex != widget.index)
        progress.originIndex,
    };
    for (final index in visible) {
      if (progress != null &&
          target != null &&
          widget.index != progress.originIndex &&
          index == progress.originIndex) {
        continue;
      }
      _ensureIdentity(index);
      _pages[index] = widget.pageBuilder(index, widget.chromeHidden);
    }
  }

  void _refreshAllPages() {
    for (final index in _pages.keys.toList()) {
      _ensureIdentity(index);
      _pages[index] = widget.pageBuilder(index, widget.chromeHidden);
    }
  }

  @override
  void dispose() {
    _progress?.removeListener(_onProgressChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _ensureVisiblePages();
    final progress =
        _progress?.value ??
        GfTabSwipeProgress(originIndex: widget.index, offset: 0);
    // Reduced motion must match _ensureVisiblePages and TabSwipeSurface's
    // settle path: drag progress keeps pages in place and the pane swaps
    // instantly on index change instead of translating.
    final target = GfMotion.reducedOf(context) ? null : _targetIndex(progress);
    final transitioning = target != null;
    final origin = transitioning ? progress.originIndex : widget.index;
    final direction = target != null && target > origin ? 1.0 : -1.0;
    final offset = transitioning
        ? progress.offset.clamp(-1.0, 1.0).toDouble()
        : 0.0;
    final tickersEnabled = TickerMode.valuesOf(context).enabled;

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        return ClipRect(
          child: Stack(
            fit: StackFit.expand,
            children: [
              for (final index in _pages.keys.toList()..sort())
                Positioned.fill(
                  child: KeyedSubtree(
                    key: _pageKey(index),
                    child: index == origin && transitioning
                        ? TickerMode(
                            enabled: tickersEnabled,
                            child: ExcludeSemantics(
                              excluding: widget.index != origin,
                              child: IgnorePointer(
                                ignoring: true,
                                child: Transform.translate(
                                  offset: Offset(-offset * width, 0),
                                  child: _pages[index]!,
                                ),
                              ),
                            ),
                          )
                        : index == target && transitioning
                        ? TickerMode(
                            enabled: tickersEnabled,
                            child: ExcludeSemantics(
                              excluding: widget.index != target,
                              child: IgnorePointer(
                                ignoring: true,
                                child: Transform.translate(
                                  offset: Offset(
                                    (direction - offset) * width,
                                    0,
                                  ),
                                  child: _pages[index]!,
                                ),
                              ),
                            ),
                          )
                        : index == widget.index
                        ? TickerMode(
                            enabled: tickersEnabled,
                            child: _pages[index]!,
                          )
                        : Offstage(
                            offstage: true,
                            child: TickerMode(
                              enabled: false,
                              child: _pages[index]!,
                            ),
                          ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
