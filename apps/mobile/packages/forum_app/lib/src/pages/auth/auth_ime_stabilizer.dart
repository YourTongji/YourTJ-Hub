import 'package:flutter/foundation.dart';

/// A cancellable delayed callback used by [AuthImeStabilizer].
abstract interface class AuthImeTimer {
  /// Prevents the callback from running when the scheduler supports canceling.
  void cancel();
}

/// Schedules a callback and returns a handle that can cancel it.
typedef AuthImeSchedule =
    AuthImeTimer Function(Duration delay, VoidCallback callback);

/// Supplies elapsed time to the stabilizer.
typedef AuthImeNow = Duration Function();

/// Bounded Android recovery for auth text fields.
///
/// The stabilizer deliberately has no knowledge of Android brands, passwords,
/// or credentials. The page supplies the target identity, focus state, actual
/// IME visibility, and the platform show/request-focus operations. Only an
/// explicit pointer or keyboard-next intent authorizes focus handoff recovery;
/// ordinary focus changes never fight the system by themselves.
class AuthImeStabilizer<T> {
  AuthImeStabilizer({
    required this.enabled,
    required this.isFocused,
    required this.isImeVisible,
    required this.showIme,
    required this.releaseFocus,
    required this.requestFocus,
    required this.now,
    required this.schedule,
    this.recoverySource,
    this.onRecoverySourceDismissed,
    this.onLog,
    this.targetName,
  });

  /// Whether this instance is active. The page sets this only for Android.
  final bool enabled;

  static const Duration tokenLifetime = Duration(milliseconds: 800);
  static const Duration handoffObservationDelay = Duration(milliseconds: 80);
  static const Duration handoffRetryDelay = Duration(milliseconds: 80);
  static const int maxRecoveryAttempts = 2;

  // First attempt is post-frame; the following attempts land at 80, 180, 380
  // and 700 ms from the interaction. This is intentionally finite.
  static const List<Duration> _watchdogDelays = <Duration>[
    Duration.zero,
    Duration(milliseconds: 80),
    Duration(milliseconds: 100),
    Duration(milliseconds: 200),
    Duration(milliseconds: 320),
  ];

  static const int _maxWatchdogActivations = 2;

  final bool Function(T target) isFocused;
  final bool Function() isImeVisible;
  final VoidCallback showIme;
  final void Function(T target) releaseFocus;
  final void Function(T target) requestFocus;
  final AuthImeNow now;
  final AuthImeSchedule schedule;
  final T? recoverySource;
  final void Function(T target)? onRecoverySourceDismissed;
  final void Function(String message)? onLog;
  final String Function(T target)? targetName;

  _AuthImeIntent<T>? _intent;
  AuthImeTimer? _intentExpiry;
  AuthImeTimer? _handoffObservation;
  AuthImeTimer? _recoveryTimer;
  AuthImeTimer? _watchdogTimer;
  bool _disposed = false;
  bool? _lastImeVisible;
  int _generation = 0;

  /// Records a user pointer-down before the normal TextField gesture handling.
  void pointerDown(T target) => focusRequested(target);

  /// Records an explicit field target before moving focus, including IME Next.
  void focusRequested(T target) {
    if (!_active) return;

    _cancelIntentTimers();
    final int generation = ++_generation;
    // Moving across a secure field can briefly hide either keyboard. A tap on
    // the already-focused field does not authorize reopening a dismissed IME.
    final bool targetWasFocused = isFocused(target);
    final T? source = recoverySource;
    final _AuthImeIntent<T> intent = _AuthImeIntent<T>(
      target: target,
      generation: generation,
      expiresAt: now() + tokenLifetime,
      suppressImeDismissal:
          source != null &&
          (target == source || isFocused(source)) &&
          !targetWasFocused &&
          isImeVisible(),
    );
    _intent = intent;
    _log('token create target=${_name(target)}');
    _intentExpiry = schedule(tokenLifetime, () => _expireIntent(intent));

    // A second tap after hiding the keyboard may not emit a FocusNode change.
    if (targetWasFocused) _startWatchdog(intent);
    if (source != null && source != target && isFocused(source)) {
      _handoffObservation = schedule(handoffObservationDelay, () {
        if (_isCurrent(intent)) _maybeRecover(intent);
      });
    }
  }

  /// Reports a FocusNode transition for a field covered by the page.
  void focusChanged(T target, bool focused) {
    if (!_active) return;
    _log('focus ${_name(target)}=${focused ? 'true' : 'false'}');

    final _AuthImeIntent<T>? intent = _intent;
    if (intent == null) return;

    if (target == intent.target) {
      if (focused) {
        _cancelHandoffTimers();
        _startWatchdog(intent);
      } else {
        _cancelWatchdog();
      }
      return;
    }

    if (focused && target == recoverySource && !isFocused(intent.target)) {
      _scheduleRecoveryObservation(intent);
    }
  }

  /// Re-checks actual viewInsets after a metrics update.
  void metricsChanged() {
    if (!_active) return;
    final bool visible = isImeVisible();
    final bool wasVisible = _lastImeVisible == true;
    final bool becameVisible = visible && _lastImeVisible != true;
    if (_lastImeVisible != visible) {
      _lastImeVisible = visible;
      _log('ime visible=${visible ? 'true' : 'false'}');
    }

    if (wasVisible && !visible) {
      final T? dismissedTarget = _dismissedTarget();
      if (dismissedTarget != null) {
        _releaseDismissedFocus(dismissedTarget);
        return;
      }
    }

    final _AuthImeIntent<T>? intent = _intent;
    if (visible) {
      if (intent != null &&
          intent.suppressImeDismissal &&
          isFocused(intent.target) &&
          becameVisible) {
        intent.suppressImeDismissal = false;
        _log('target IME visible; dismissal suppression consumed');
      }
      _cancelWatchdog();
    } else if (intent != null && isFocused(intent.target)) {
      _startWatchdog(intent);
    }
  }

  /// Cancels all current recovery and watchdog work.
  void cancel() {
    if (!_active) return;
    _generation++;
    _cancelIntentTimers();
    _intent = null;
  }

  /// Releases timers when the owning page is disposed.
  void dispose() {
    if (_disposed) return;
    _disposed = true;
    _generation++;
    _cancelIntentTimers();
    _intent = null;
  }

  bool get _active => enabled && !_disposed;

  T? _dismissedTarget() {
    final T? source = recoverySource;
    final _AuthImeIntent<T>? intent = _intent;
    if (source != null && isFocused(source)) {
      if (intent != null && intent.target != source && _isCurrent(intent)) {
        _log('user dismiss suppressed handoff target=${_name(intent.target)}');
        return null;
      }
      if (intent != null && intent.suppressImeDismissal && _isCurrent(intent)) {
        _log('user dismiss suppressed secure field transition');
        return null;
      }
      return source;
    }
    if (intent == null || !isFocused(intent.target)) return null;
    if (intent.suppressImeDismissal && _isCurrent(intent)) {
      _log('user dismiss suppressed secure field transition');
      return null;
    }
    return intent.target;
  }

  void _releaseDismissedFocus(T target) {
    _log('user dismiss releaseFocus target=${_name(target)}');
    cancel();
    if (target == recoverySource) onRecoverySourceDismissed?.call(target);
    releaseFocus(target);
  }

  bool _isCurrent(_AuthImeIntent<T> intent) {
    return _active &&
        _generation == intent.generation &&
        identical(_intent, intent) &&
        now().compareTo(intent.expiresAt) < 0;
  }

  void _expireIntent(_AuthImeIntent<T> intent) {
    if (!_active ||
        _generation != intent.generation ||
        !identical(_intent, intent)) {
      return;
    }
    _log('token expire target=${_name(intent.target)}');
    _cancelIntentTimers();
    _intent = null;
  }

  void _startWatchdog(_AuthImeIntent<T> intent) {
    if (!_isCurrent(intent) || !isFocused(intent.target)) return;
    if (isImeVisible()) {
      _cancelWatchdog();
      return;
    }
    if (_watchdogTimer != null ||
        intent.watchdogActivations >= _maxWatchdogActivations) {
      return;
    }

    intent.watchdogActivations++;
    intent.watchdogAttempts = 0;
    _scheduleWatchdogAttempt(intent, 0);
  }

  void _scheduleWatchdogAttempt(_AuthImeIntent<T> intent, int delayIndex) {
    if (!_isCurrent(intent) || !isFocused(intent.target)) return;
    _watchdogTimer = schedule(_watchdogDelays[delayIndex], () {
      _watchdogTimer = null;
      if (!_isCurrent(intent) || !isFocused(intent.target)) return;
      if (isImeVisible()) {
        _cancelWatchdog();
        return;
      }

      intent.watchdogAttempts++;
      _log(
        'TextInput.show retry=${intent.watchdogAttempts}/${_watchdogDelays.length} '
        'target=${_name(intent.target)}',
      );
      showIme();
      if (isImeVisible()) {
        _cancelWatchdog();
        return;
      }

      if (intent.watchdogAttempts < _watchdogDelays.length) {
        _scheduleWatchdogAttempt(intent, intent.watchdogAttempts);
      }
    });
  }

  void _maybeRecover(_AuthImeIntent<T> intent) {
    _handoffObservation = null;
    if (!_isCurrent(intent) || intent.recoveryScheduled) return;
    final T? source = recoverySource;
    if (source == null || source == intent.target) return;
    if (!isFocused(source) || isFocused(intent.target)) return;
    if (intent.recoveryAttempts >= maxRecoveryAttempts) return;

    intent.recoveryScheduled = true;
    _recoveryTimer = schedule(Duration.zero, () {
      _recoveryTimer = null;
      intent.recoveryScheduled = false;
      if (!_isCurrent(intent) ||
          !isFocused(source) ||
          isFocused(intent.target) ||
          intent.recoveryAttempts >= maxRecoveryAttempts) {
        return;
      }

      intent.recoveryAttempts++;
      _log(
        'focus recovery attempt=${intent.recoveryAttempts}/$maxRecoveryAttempts '
        'target=${_name(intent.target)}',
      );
      requestFocus(intent.target);

      if (intent.recoveryAttempts < maxRecoveryAttempts &&
          _isCurrent(intent) &&
          !isFocused(intent.target)) {
        _handoffObservation = schedule(handoffRetryDelay, () {
          if (_isCurrent(intent)) _maybeRecover(intent);
        });
      }
    });
  }

  void _scheduleRecoveryObservation(_AuthImeIntent<T> intent) {
    _handoffObservation?.cancel();
    _handoffObservation = schedule(handoffObservationDelay, () {
      if (_isCurrent(intent)) _maybeRecover(intent);
    });
  }

  void _cancelWatchdog() {
    _watchdogTimer?.cancel();
    _watchdogTimer = null;
  }

  void _cancelHandoffTimers() {
    _handoffObservation?.cancel();
    _handoffObservation = null;
    _recoveryTimer?.cancel();
    _recoveryTimer = null;
  }

  void _cancelIntentTimers() {
    _intentExpiry?.cancel();
    _intentExpiry = null;
    _cancelHandoffTimers();
    _cancelWatchdog();
  }

  String _name(T target) => targetName?.call(target) ?? target.toString();

  void _log(String message) {
    if (kDebugMode) onLog?.call('[AuthIme] $message');
  }
}

class _AuthImeIntent<T> {
  _AuthImeIntent({
    required this.target,
    required this.generation,
    required this.expiresAt,
    required this.suppressImeDismissal,
  });

  final T target;
  final int generation;
  final Duration expiresAt;
  bool suppressImeDismissal;
  int recoveryAttempts = 0;
  int watchdogActivations = 0;
  int watchdogAttempts = 0;
  bool recoveryScheduled = false;
}
