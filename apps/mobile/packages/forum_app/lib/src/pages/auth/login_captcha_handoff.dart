import 'package:flutter/foundation.dart';

/// Cancellable post-frame focus request for the password-to-captcha handoff.
abstract interface class LoginCaptchaForwardTimer {
  void cancel();
}

typedef LoginCaptchaForwardSchedule =
    LoginCaptchaForwardTimer Function(VoidCallback callback);

/// Owns the one-way focus generation after the password stage completes.
///
/// It deliberately does not know about Flutter focus nodes or IME APIs. A
/// later captcha payload can make a pending generation eligible, while an
/// explicit user target, mode switch, or disposal invalidates it.
class LoginCaptchaForwardHandoff {
  LoginCaptchaForwardHandoff({
    required this.isMounted,
    required this.isCaptchaEligible,
    required this.requestCaptchaFocus,
    required this.schedule,
  });

  final bool Function() isMounted;
  final bool Function() isCaptchaEligible;
  final VoidCallback requestCaptchaFocus;
  final LoginCaptchaForwardSchedule schedule;

  LoginCaptchaForwardTimer? _pending;
  int _generation = 0;
  bool _disposed = false;
  bool _active = false;
  bool _focusRequested = false;

  void begin() {
    if (_disposed) return;
    _cancelPending();
    _generation++;
    _active = true;
    _focusRequested = false;
    _scheduleIfEligible();
  }

  void captchaEligibilityChanged() {
    if (_disposed || !_active || _focusRequested) return;
    _scheduleIfEligible();
  }

  void cancel() {
    if (_disposed) return;
    _generation++;
    _active = false;
    _focusRequested = false;
    _cancelPending();
  }

  void dispose() {
    if (_disposed) return;
    _disposed = true;
    _generation++;
    _active = false;
    _cancelPending();
  }

  void _scheduleIfEligible() {
    if (!isMounted() || !isCaptchaEligible() || _pending != null) return;
    final int generation = _generation;
    _pending = schedule(() {
      _pending = null;
      if (_disposed ||
          !_active ||
          _focusRequested ||
          generation != _generation ||
          !isMounted() ||
          !isCaptchaEligible()) {
        return;
      }
      _focusRequested = true;
      requestCaptchaFocus();
    });
  }

  void _cancelPending() {
    _pending?.cancel();
    _pending = null;
  }
}
