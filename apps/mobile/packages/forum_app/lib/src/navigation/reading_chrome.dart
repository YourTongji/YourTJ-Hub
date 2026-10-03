import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

/// How much of the reading chrome is on screen, from 0 (hidden) to 1.
///
/// [animate] marks settling changes; finger-tracked changes apply directly.
@immutable
class ChromeReveal {
  const ChromeReveal(this.value, {this.animate = false});

  final double value;
  final bool animate;
}

/// Chrome follows the finger like X's feed: it slides out over [travel]
/// pixels of downward scroll, any upward scroll pulls it back, and a partial
/// state settles in the last direction once scrolling stops. Bounce and
/// horizontal gestures are ignored. Moving chrome never changes a scroll
/// view's viewport or pixel offset.
class ReadingChrome extends ChangeNotifier {
  static const double travel = 64;

  final ValueNotifier<ChromeReveal> _reveal = ValueNotifier(
    const ChromeReveal(1),
  );
  bool _hidden = false;
  int _direction = 0;

  /// Continuous position for widgets that move with the scroll.
  ValueListenable<ChromeReveal> get reveal => _reveal;

  /// Whether chrome is mostly off screen; listeners hear only its flips.
  bool get hidden => _hidden;

  void show() {
    _direction = 0;
    _set(1, animate: true);
  }

  void update(
    double delta,
    double pixels, {
    double maxScrollExtent = double.infinity,
    bool locked = false,
  }) {
    if (locked || pixels <= 0) {
      show();
      return;
    }
    if (delta == 0 || pixels > maxScrollExtent) return;
    _direction = delta > 0 ? 1 : -1;
    // Near the top the header stays tied to the content it covers.
    final double floor = (1 - pixels / travel).clamp(0.0, 1.0);
    _set(
      (_reveal.value.value - delta / travel).clamp(floor, 1.0),
      animate: false,
    );
  }

  /// Finishes a partial reveal in the direction the reader last moved.
  void settle(double pixels) {
    final double value = _reveal.value.value;
    if (value == 0 || value == 1) return;
    _set(_direction < 0 || pixels < travel ? 1 : 0, animate: true);
  }

  void _set(double value, {required bool animate}) {
    final ChromeReveal current = _reveal.value;
    if (current.value == value && current.animate == animate) return;
    _reveal.value = ChromeReveal(value, animate: animate);
    final bool hidden = value < .5;
    if (hidden == _hidden) return;
    _hidden = hidden;
    notifyListeners();
  }

  @override
  void dispose() {
    _reveal.dispose();
    super.dispose();
  }
}

final readingChromeProvider = ChangeNotifierProvider<ReadingChrome>(
  (ref) => ReadingChrome(),
);

/// Settles chrome quickly with the shared enter curve; tracking is instant.
Duration readingChromeDuration(BuildContext context, ChromeReveal reveal) =>
    reveal.animate
    ? GfMotion.duration(context, GfMotion.content)
    : Duration.zero;

/// Slides [child] out by its own height in [direction] as chrome hides.
class ReadingChromeSlide extends ConsumerWidget {
  const ReadingChromeSlide({
    super.key,
    required this.direction,
    required this.child,
  });

  /// -1 slides up (headers), 1 slides down (bottom bars).
  final double direction;
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) =>
      ValueListenableBuilder<ChromeReveal>(
        valueListenable: ref.watch(readingChromeProvider).reveal,
        child: child,
        builder: (context, reveal, child) => AnimatedSlide(
          offset: Offset(0, direction * (1 - reveal.value)),
          duration: readingChromeDuration(context, reveal),
          curve: GfMotion.enterCurve,
          child: child,
        ),
      );
}

/// Lines a retained tab page up with hidden chrome, like a collapsed app bar.
///
/// Page insets never change, so while chrome is hidden an off-screen page
/// scrolls its reserved header band away; swiping to it then shows content
/// instead of a blank gap. The page on screen never moves, and returning
/// chrome restores the offsets this widget changed. Off-screen and outgoing
/// pages also keep their scroll events from driving the shared chrome.
class ChromeAlignedPage extends StatefulWidget {
  const ChromeAlignedPage({
    super.key,
    required this.topInset,
    required this.chromeHidden,
    required this.current,
    required this.child,
  });

  final double topInset;
  final bool chromeHidden;
  final bool current;
  final Widget child;

  @override
  State<ChromeAlignedPage> createState() => _ChromeAlignedPageState();
}

class _ChromeAlignedPageState extends State<ChromeAlignedPage> {
  ScrollPosition? _position;
  double? _restoreTo;

  // TabPageTransition and the shell disable tickers for off-screen pages.
  bool get _offstage => !TickerMode.valuesOf(context).enabled;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(covariant ChromeAlignedPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    _sync();
  }

  void _sync() {
    if (!_offstage) return;
    widget.chromeHidden ? _align() : _restore();
  }

  double _alignedOffset(ScrollPosition position) =>
      math.min(widget.topInset, position.maxScrollExtent);

  void _align() {
    final ScrollPosition? position = _position;
    if (position == null || !position.hasContentDimensions) return;
    if (_restoreTo != null) return;
    final double target = _alignedOffset(position);
    if (position.pixels >= target) return;
    _restoreTo = position.pixels;
    position.jumpTo(target);
  }

  void _restore() {
    final ScrollPosition? position = _position;
    final double? from = _restoreTo;
    _restoreTo = null;
    if (position == null || from == null || !position.hasContentDimensions) {
      return;
    }
    // Only undo this widget's own move; a page the reader scrolled stays.
    if (position.pixels == _alignedOffset(position)) position.jumpTo(from);
  }

  void _track(BuildContext? context, int depth, Axis axis) {
    if (context == null || depth != 0 || axis != Axis.vertical) return;
    final ScrollPosition? position = Scrollable.maybeOf(context)?.position;
    if (position == null || identical(position, _position)) return;
    _position = position;
    _restoreTo = null;
    // A freshly laid out page has not been seen yet, so it starts aligned
    // whether it enters mid-swipe or becomes current first.
    if (widget.chromeHidden) _align();
  }

  @override
  Widget build(BuildContext context) =>
      NotificationListener<ScrollMetricsNotification>(
        onNotification: (notification) {
          _track(
            notification.context,
            notification.depth,
            notification.metrics.axis,
          );
          return false;
        },
        child: NotificationListener<ScrollNotification>(
          onNotification: (notification) {
            _track(
              notification.context,
              notification.depth,
              notification.metrics.axis,
            );
            if (notification is ScrollStartNotification &&
                notification.dragDetails != null) {
              _restoreTo = null;
            }
            return !widget.current || _offstage;
          },
          child: widget.child,
        ),
      );
}
