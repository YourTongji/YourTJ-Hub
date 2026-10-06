import 'dart:async';
import 'dart:convert';
import 'package:flutter/widgets.dart';
import '../gen/topic.dart';

class FeedContext {
  const FeedContext(this.trace, this.position, this.topic);
  final String trace;
  final int position;
  final int topic;
}

/// Bounded, account-scoped memory only. No URL, auth token or persisted queue.
class FeedTelemetry with WidgetsBindingObserver {
  FeedTelemetry();
  bool _observing = false;
  static final instance = FeedTelemetry();
  int _account = 0;
  FeedContext? selected;
  Future<void> Function(Map<String, dynamic>)? send;
  final Map<String, Map<String, dynamic>> _patches = {};
  Timer? _timer;
  bool _sending = false;
  bool _retried = false;
  bool foreground = true;
  bool _routeVisible = true;
  FeedContext? _detail;
  final Stopwatch _dwell = Stopwatch();
  int _reportedSeconds = 0;

  void bindAccount(int id) {
    if (id == _account) return;
    _account = id;
    _retried = false;
    _timer?.cancel();
    _timer = null;
    selected = null;
    endDetail();
    _patches.clear();
  }

  void select(TopicPayload topic) {
    final trace = topic.feedTrace;
    final position = topic.feedPosition;
    selected = trace != null && trace.isNotEmpty && position != null
        ? FeedContext(trace, position, topic.id)
        : null;
  }

  Map<String, String> headers(String path) {
    final c = selected;
    final uri = Uri.tryParse(path);
    if (c == null || uri == null || uri.hasScheme || uri.hasAuthority) {
      return {};
    }
    if (uri.path != '/p/post/${c.topic}' && !path.startsWith('/api/forum/')) {
      return {};
    }
    return {
      'X-Goose-Feed-Trace': c.trace,
      'X-Goose-Feed-Position': '${c.position}',
      'X-Goose-Feed-Topic': '${c.topic}',
    };
  }

  Map<String, dynamic>? _patch(FeedContext c) {
    if (!_patches.containsKey(c.trace) && _patches.length >= 50) return null;
    _start();
    return _patches.putIfAbsent(
      c.trace,
      () => {'trace': c.trace, 'visibleMask': 0, 'dwell': <String, int>{}},
    );
  }

  void visible(TopicPayload topic) {
    if (!foreground || topic.feedTrace == null || topic.feedPosition == null) {
      return;
    }
    final c = FeedContext(topic.feedTrace!, topic.feedPosition!, topic.id);
    final p = _patch(c);
    if (p != null) {
      p['visibleMask'] = (p['visibleMask'] as int) | (1 << c.position);
    }
  }

  void beginDetail(int topic) {
    final context = selected;
    endDetail();
    if (context?.topic != topic) return;
    selected = context;
    _detail = context;
    _reportedSeconds = 0;
    _dwell.reset();
    if (foreground && _routeVisible) _dwell.start();
    _start();
  }

  void _reportDwell() {
    final c = _detail;
    if (c == null) return;
    final seconds = (_dwell.elapsed.inSeconds ~/ 5 * 5).clamp(0, 600);
    if (seconds <= _reportedSeconds) return;
    _reportedSeconds = seconds;
    final p = _patch(c);
    if (p != null) (p['dwell'] as Map<String, int>)['${c.position}'] = seconds;
  }

  void routeVisible(bool visible) {
    _routeVisible = visible;
    _reportDwell();
    if (visible && foreground && _detail != null) {
      _dwell.start();
    } else {
      _dwell.stop();
    }
  }

  void endDetail() {
    _reportDwell();
    _detail = null;
    selected = null;
    _dwell.stop();
  }

  void _start() {
    if (!_observing) {
      WidgetsBinding.instance.addObserver(this);
      _observing = true;
    }
    _timer ??= Timer.periodic(const Duration(seconds: 5), (_) {
      _reportDwell();
      unawaited(flush());
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _reportDwell();
    foreground = state == AppLifecycleState.resumed;
    if (foreground && _routeVisible && _detail != null) {
      _dwell.start();
    } else {
      _dwell.stop();
      unawaited(flush());
    }
  }

  Future<void> flush() async {
    if (_sending || _patches.isEmpty || send == null) return;
    final currentAccount = _account;
    final rows = <Map<String, dynamic>>[];
    for (final row in _patches.values) {
      final next = [...rows, row];
      if (utf8.encode(jsonEncode({'patches': next})).length > 32768) break;
      rows.add(
        Map<String, dynamic>.from(row)
          ..['dwell'] = Map<String, int>.from(row['dwell'] as Map),
      );
    }
    if (rows.isEmpty) {
      _patches.clear();
      return;
    }
    for (final row in rows) {
      _patches.remove(row['trace']);
    }
    _sending = true;
    try {
      await send!({'patches': rows});
      _retried = false;
    } catch (_) {
      if (currentAccount == _account && !_retried) {
        _retried = true;
        for (final row in rows) {
          if (_patches.length < 50) {
            _patches.putIfAbsent(row['trace'] as String, () => row);
          }
        }
      }
    } finally {
      _sending = false;
    }
  }
}
