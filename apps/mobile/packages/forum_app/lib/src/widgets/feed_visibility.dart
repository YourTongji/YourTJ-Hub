import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/material.dart';

/// Viewport-based exposure for recycled rows; background time cannot complete
/// the continuous one-second, at-least-half-visible threshold.
class FeedVisibility extends StatefulWidget {
  const FeedVisibility({
    super.key,
    required this.topic,
    required this.child,
    this.reason,
  });
  final TopicPayload topic;
  final Widget child;
  final String? reason;
  @override
  State<FeedVisibility> createState() => _FeedVisibilityState();
}

class _FeedVisibilityState extends State<FeedVisibility>
    with WidgetsBindingObserver {
  ScrollPosition? _scroll;
  Timer? _check;
  Timer? _qualify;
  String? _reported;
  bool _foreground = true;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _scroll?.removeListener(_schedule);
    _scroll = Scrollable.maybeOf(context)?.position;
    _scroll?.addListener(_schedule);
    _schedule();
  }

  @override
  void didUpdateWidget(FeedVisibility old) {
    super.didUpdateWidget(old);
    if (old.topic.id != widget.topic.id ||
        old.topic.feedTrace != widget.topic.feedTrace ||
        old.topic.feedPosition != widget.topic.feedPosition) {
      _qualify?.cancel();
      _qualify = null;
      _reported = null;
      _schedule();
    }
  }

  void _schedule() {
    _check?.cancel();
    _check = Timer(const Duration(milliseconds: 100), _measure);
  }

  bool _visible() {
    if (!_foreground ||
        !mounted ||
        widget.topic.feedTrace == null ||
        ModalRoute.of(context)?.isCurrent == false) {
      return false;
    }
    final box = context.findRenderObject();
    if (box is! RenderBox || !box.hasSize) return false;
    final viewportObject = RenderAbstractViewport.maybeOf(box);
    if (viewportObject == null || viewportObject is! RenderBox) return false;
    final viewport = viewportObject as RenderBox;
    if (!viewport.hasSize) return false;
    final rect = box.localToGlobal(Offset.zero) & box.size;
    final bounds = viewport.localToGlobal(Offset.zero) & viewport.size;
    final visible = rect.intersect(bounds);
    return !visible.isEmpty &&
        rect.width * rect.height > 0 &&
        visible.width * visible.height / (rect.width * rect.height) >= .5;
  }

  void _measure() {
    final key = '${widget.topic.feedTrace}:${widget.topic.feedPosition}';
    if (!_visible()) {
      _qualify?.cancel();
      _qualify = null;
      return;
    }
    if (_reported == key || _qualify != null) return;
    _qualify = Timer(const Duration(seconds: 1), () {
      _qualify = null;
      if (_visible()) {
        _reported = key;
        FeedTelemetry.instance.visible(widget.topic);
      }
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    if (!_foreground) {
      _qualify?.cancel();
      _qualify = null;
    } else {
      _schedule();
    }
  }

  @override
  void dispose() {
    _scroll?.removeListener(_schedule);
    _check?.cancel();
    _qualify?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.reason == null
      ? widget.child
      : Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
              child: Text(
                widget.reason!,
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
            ),
            widget.child,
          ],
        );
}
