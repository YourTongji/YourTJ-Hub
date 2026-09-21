import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/login_captcha_handoff.dart';

void main() {
  group('LoginCaptchaForwardHandoff', () {
    test('prefetched captcha receives exactly one post-frame focus', () {
      final h = _Harness(captchaEligible: true);

      h.handoff.begin();
      h.handoff.captchaEligibilityChanged();
      h.flushPostFrame();

      expect(h.focusRequests, 1);
      expect(h.showRequests, 0);
    });

    test(
      'in-flight captcha stays unfocused until payload becomes eligible',
      () {
        final h = _Harness();

        h.handoff.begin();
        h.flushPostFrame();
        expect(h.focusRequests, 0);

        h.captchaEligible = true;
        h.handoff.captchaEligibilityChanged();
        h.flushPostFrame();
        h.handoff.captchaEligibilityChanged();
        h.flushPostFrame();

        expect(h.focusRequests, 1);
        expect(h.showRequests, 0);
      },
    );

    test('explicit username pointer-down cancels automatic captcha focus', () {
      final h = _Harness(captchaEligible: true);

      h.handoff.begin();
      h.handoff.cancel();
      h.flushPostFrame();
      h.handoff.captchaEligibilityChanged();
      h.flushPostFrame();

      expect(h.focusRequests, 0);
    });

    test('blank-space handoff can begin the same forward focus generation', () {
      final h = _Harness(captchaEligible: true);

      h.handoff.begin();
      h.flushPostFrame();

      expect(h.focusRequests, 1);
    });

    test('mode switch and disposal cancel pending focus', () {
      final h = _Harness();

      h.handoff.begin();
      h.handoff.cancel();
      h.captchaEligible = true;
      h.handoff.captchaEligibilityChanged();
      h.flushPostFrame();
      expect(h.focusRequests, 0);

      h.handoff.begin();
      h.handoff.dispose();
      h.handoff.captchaEligibilityChanged();
      h.flushPostFrame();
      expect(h.focusRequests, 0);
    });
  });
}

class _Harness {
  _Harness({this.captchaEligible = false}) {
    handoff = LoginCaptchaForwardHandoff(
      isMounted: () => mounted,
      isCaptchaEligible: () => captchaEligible,
      requestCaptchaFocus: () => focusRequests++,
      schedule: (callback) {
        pending.add(callback);
        return _CallbackTimer(() => pending.remove(callback));
      },
    );
  }

  bool mounted = true;
  bool captchaEligible;
  int focusRequests = 0;
  int showRequests = 0;
  final List<void Function()> pending = <void Function()>[];
  late final LoginCaptchaForwardHandoff handoff;

  void flushPostFrame() {
    final callbacks = List<void Function()>.of(pending);
    pending.clear();
    for (final callback in callbacks) {
      callback();
    }
  }
}

class _CallbackTimer implements LoginCaptchaForwardTimer {
  _CallbackTimer(this._cancelCallback);

  final void Function() _cancelCallback;

  @override
  void cancel() => _cancelCallback();
}
