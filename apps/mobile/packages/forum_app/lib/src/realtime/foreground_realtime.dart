import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';

typedef RealtimeConnect =
    Future<ForumRealtimeConnection> Function(
      String token,
      CancelToken cancelToken,
      void Function() onActivity,
    );

/// A single foreground connection. SSE is an invalidation channel: `hello`
/// and reconnection reconcile REST state, while failed/old servers use a
/// foreground-only timer. A generation fences every pending connect/event.
class ForegroundRealtimeCoordinator {
  ForegroundRealtimeCoordinator({
    required this.readToken,
    required this.connect,
    required this.onResync,
    required this.onEvent,
    required this.onFallbackTick,
    required this.onHealthChanged,
  });

  final Future<String?> Function() readToken;
  final RealtimeConnect connect;
  final void Function() onResync;
  final void Function(ForumSseFrame) onEvent;
  final void Function() onFallbackTick;
  final void Function(bool) onHealthChanged;

  bool _running = false;
  bool _healthy = false;
  bool _unsupported = false;
  int _attempt = 0;
  int _failures = 0;
  int _heartbeatSeconds = 15;
  DateTime _lastActivity = DateTime.now();
  CancelToken? _pendingCancel;
  ForumRealtimeConnection? _connection;
  StreamSubscription<ForumSseFrame>? _subscription;
  Timer? _retryTimer;
  Timer? _fallbackTimer;
  Timer? _watchdog;
  Timer? _handshakeTimer;

  bool get healthy => _healthy;

  void start() {
    if (_running) return;
    _running = true;
    _unsupported = false;
    _failures = 0;
    _open();
  }

  void stop() {
    if (!_running) return;
    _running = false;
    _attempt++;
    _retryTimer?.cancel();
    _fallbackTimer?.cancel();
    _cancelConnection();
    _setHealthy(false);
  }

  void _cancelConnection() {
    _handshakeTimer?.cancel();
    _watchdog?.cancel();
    unawaited(_subscription?.cancel());
    _subscription = null;
    _connection?.close();
    _connection = null;
    if (_pendingCancel case final cancel?) {
      if (!cancel.isCancelled) cancel.cancel('SSE attempt ended');
    }
    _pendingCancel = null;
  }

  void _setHealthy(bool value) {
    if (_healthy == value) return;
    _healthy = value;
    onHealthChanged(value);
    if (value) {
      _fallbackTimer?.cancel();
      _fallbackTimer = null;
    }
  }

  void _open() {
    if (!_running || _unsupported) return;
    final attempt = ++_attempt;
    final cancel = CancelToken();
    _pendingCancel = cancel;
    unawaited(_connect(attempt, cancel));
  }

  Future<void> _connect(int attempt, CancelToken cancel) async {
    try {
      final token = await readToken();
      if (!_current(attempt)) return;
      if (token == null || token.isEmpty) {
        _pendingCancel = null;
        _running = false;
        return;
      }
      final connection = await connect(token, cancel, () {
        if (_current(attempt)) _lastActivity = DateTime.now();
      });
      if (!_current(attempt)) {
        connection.close();
        return;
      }
      _connection = connection;
      _lastActivity = DateTime.now();
      _handshakeTimer = Timer(const Duration(seconds: 10), () {
        if (_current(attempt) && !_healthy) _disconnect(attempt);
      });
      _watchdog = Timer.periodic(const Duration(seconds: 5), (_) {
        if (!_current(attempt)) return;
        final limit = Duration(seconds: _heartbeatSeconds * 3);
        if (DateTime.now().difference(_lastActivity) > limit) {
          _disconnect(attempt);
        }
      });
      _subscription = connection.events.listen(
        (frame) => _handleFrame(attempt, frame),
        onError: (Object error, StackTrace stack) => _disconnect(attempt),
        onDone: () => _disconnect(attempt),
        cancelOnError: true,
      );
    } on ForumRealtimeHttpException catch (error) {
      if (!_current(attempt)) return;
      if (error.statusCode == 401 || error.statusCode == 403) {
        stop();
        return;
      }
      if (error.statusCode == 404 || error.statusCode == 405) {
        _unsupported = true;
      }
      _disconnect(attempt, retryAfter: error.retryAfter);
    } on DioException catch (error) {
      if (!_current(attempt) || CancelToken.isCancel(error)) return;
      _disconnect(attempt);
    } catch (_) {
      if (_current(attempt)) _disconnect(attempt);
    }
  }

  bool _current(int attempt) => _running && attempt == _attempt;

  void _handleFrame(int attempt, ForumSseFrame frame) {
    if (!_current(attempt)) return;
    try {
      if (frame.event == ForumRealtimeEventType.hello.wireName) {
        final hello = ForumRealtimeHello.fromJson(
          jsonDecode(frame.data) as Map<String, dynamic>,
        );
        if (hello.version != 1 || !hello.resync) {
          throw const FormatException('Unsupported SSE hello');
        }
        _heartbeatSeconds = hello.heartbeatSeconds.clamp(5, 60);
        _failures = 0;
        _handshakeTimer?.cancel();
        _setHealthy(true);
        onResync();
      } else if (_healthy) {
        onEvent(frame);
      }
    } catch (_) {
      _disconnect(attempt);
    }
  }

  void _disconnect(int attempt, {Duration? retryAfter}) {
    if (!_current(attempt)) return;
    // Retire this attempt before cancelling its subscription: a queued onDone
    // must not schedule a second reconnect or duplicate fallback tick.
    _attempt++;
    _cancelConnection();
    _setHealthy(false);
    _fallbackTimer ??= Timer.periodic(
      const Duration(seconds: 30),
      (_) => onFallbackTick(),
    );
    onFallbackTick();
    if (_unsupported) return;
    _failures++;
    final seconds = (1 << (_failures.clamp(1, 5))) + (attempt % 3);
    final backoff = Duration(seconds: seconds);
    final delay = retryAfter != null && retryAfter > backoff
        ? retryAfter
        : backoff;
    _retryTimer = Timer(delay, _open);
  }
}
