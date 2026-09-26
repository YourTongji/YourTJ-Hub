import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/auth_ime_stabilizer.dart';

void main() {
  group('AuthImeStabilizer', () {
    test(
      'visible password IME dismissal releases password focus once and stops recovery',
      () {
        final h = _Harness()..imeVisible = true;

        h.focus('password');
        h.stabilizer.pointerDown('password');
        h.stabilizer.focusChanged('password', true);
        h.stabilizer.metricsChanged();

        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        h.scheduler.elapse(const Duration(minutes: 1));

        expect(h.releaseFocusCalls, 1);
        expect(h.showCalls, 0);
        expect(h.focused, isNot(contains('password')));
        expect(h.logs, anyElement(contains('user dismiss')));
      },
    );

    test(
      'genuine password dismissal emits a semantic stage-complete callback',
      () {
        var dismissed = 0;
        final h = _Harness(onDismissed: (_) => dismissed++)..imeVisible = true;

        h.focus('password');
        h.stabilizer.pointerDown('password');
        h.stabilizer.focusChanged('password', true);
        h.stabilizer.metricsChanged();
        h.imeVisible = false;
        h.stabilizer.metricsChanged();

        expect(dismissed, 1);
        expect(h.releaseFocusCalls, 1);
        expect(h.recoveryCalls, 0);
        expect(h.showCalls, 0);
      },
    );

    test('initial hidden metrics sample does not release password focus', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.metricsChanged();

      expect(h.releaseFocusCalls, 0);
    });

    test(
      'username handoff suppresses password dismissal while target is settling',
      () {
        final h = _Harness()..imeVisible = true;

        h.focus('password');
        h.stabilizer.pointerDown('password');
        h.stabilizer.focusChanged('password', true);
        h.stabilizer.metricsChanged();
        h.stabilizer.pointerDown('username');

        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        h.scheduler.elapse(const Duration(seconds: 1));

        expect(h.releaseFocusCalls, 0);
        expect(h.recoveryCalls, 1);
        expect(h.focused, contains('username'));
      },
    );

    test(
      'captcha handoff suppresses password dismissal while target is settling',
      () {
        final h = _Harness()..imeVisible = true;

        h.focus('password');
        h.stabilizer.pointerDown('password');
        h.stabilizer.focusChanged('password', true);
        h.stabilizer.metricsChanged();
        h.stabilizer.pointerDown('captcha');

        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        h.scheduler.elapse(const Duration(seconds: 1));

        expect(h.releaseFocusCalls, 0);
        expect(h.recoveryCalls, 1);
        expect(h.focused, contains('captcha'));
      },
    );

    test('direct password pointer token does not suppress dismissal', () {
      final h = _Harness()..imeVisible = true;

      h.focus('password');
      h.stabilizer.pointerDown('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.metricsChanged();
      h.imeVisible = false;
      h.stabilizer.metricsChanged();
      h.scheduler.elapse(const Duration(minutes: 1));

      expect(h.releaseFocusCalls, 1);
      expect(h.showCalls, 0);
    });

    test(
      'normal IME to password IME transition does not release password early',
      () {
        final h = _Harness()..imeVisible = true;

        // 1. Username is focused with the normal IME visible.
        h.focus('username');
        h.stabilizer.focusChanged('username', true);
        h.stabilizer.metricsChanged();

        // 2-3. Pointer down starts a password entry transition before the
        // default TextField handling moves focus to password.
        h.stabilizer.pointerDown('password');
        h.focus('password');
        h.stabilizer.focusChanged('password', true);

        // 4-5. The OEM hides the normal IME before the secure IME appears.
        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        expect(h.releaseFocusCalls, 0);
        expect(h.focused, contains('password'));

        // 6. The secure IME becomes visible and consumes the entry grace.
        h.imeVisible = true;
        h.stabilizer.metricsChanged();

        // 7. A later hide with no new handoff is a genuine user dismissal.
        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        h.scheduler.elapse(const Duration(minutes: 1));

        expect(h.releaseFocusCalls, 1);
        expect(h.showCalls, 0);
        expect(h.focused, isNot(contains('password')));
      },
    );

    test(
      'first password IME sample consumes entry suppression before dismissal',
      () {
        final h = _Harness();

        // No metrics sample has been observed yet; password is not focused.
        h.stabilizer.pointerDown('password');
        h.focus('password');
        h.stabilizer.focusChanged('password', true);

        // The first sample observes the secure IME becoming visible.
        h.imeVisible = true;
        h.stabilizer.metricsChanged();

        // An immediate user dismissal must release the password field.
        h.imeVisible = false;
        h.stabilizer.metricsChanged();
        h.scheduler.elapse(const Duration(minutes: 1));

        expect(h.releaseFocusCalls, 1);
        expect(h.showCalls, 0);
        expect(h.focused, isNot(contains('password')));
      },
    );

    test('focused target with hidden IME invokes show after settling', () {
      final h = _Harness();

      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.scheduler.elapse(Duration.zero);

      expect(h.showCalls, 1);
    });

    test(
      'tapping the focused field can reopen a hidden IME without a focus event',
      () {
        final h = _Harness();
        h.focus('username');

        h.stabilizer.pointerDown('username');
        h.scheduler.elapse(Duration.zero);

        expect(h.showCalls, 1);
        expect(h.recoveryCalls, 0);
      },
    );

    test('dismissing the username keyboard cancels its show watchdog', () {
      final h = _Harness()..imeVisible = true;
      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.stabilizer.metricsChanged();

      h.imeVisible = false;
      h.stabilizer.metricsChanged();
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 0);
      expect(h.focused, isNot(contains('username')));
    });

    test('secure-to-normal IME handoff settles before honoring dismissal', () {
      final h = _Harness()..imeVisible = true;
      h.focus('password');
      h.stabilizer.metricsChanged();
      h.stabilizer.pointerDown('username');
      h.focus('username');
      h.stabilizer.focusChanged('username', true);

      h.imeVisible = false;
      h.stabilizer.metricsChanged();
      expect(h.releaseFocusCalls, 0);
      expect(h.focused, contains('username'));
      h.imeVisible = true;
      h.stabilizer.metricsChanged();
      h.imeVisible = false;
      h.stabilizer.metricsChanged();
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 0);
      expect(h.focused, isNot(contains('username')));
    });

    test('does not invoke show when the IME is already visible', () {
      final h = _Harness()..imeVisible = true;

      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 0);
    });

    test('watchdog has a bounded retry budget and expires', () {
      final h = _Harness();

      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 5);
      expect(h.scheduler.pendingCount, 0);
      h.scheduler.elapse(const Duration(minutes: 1));
      expect(h.showCalls, 5);
    });

    test('focus leave cancels the watchdog', () {
      final h = _Harness();

      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.blur('username');
      h.stabilizer.focusChanged('username', false);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 0);
    });

    test('a new pointer token supersedes the old watchdog', () {
      final h = _Harness();

      h.focus('username');
      h.stabilizer.pointerDown('username');
      h.stabilizer.focusChanged('username', true);
      h.blur('username');
      h.stabilizer.focusChanged('username', false);

      h.focus('password');
      h.stabilizer.pointerDown('password');
      h.stabilizer.focusChanged('password', true);
      h.scheduler.elapse(Duration.zero);

      expect(h.showCalls, 1);
    });

    test('pointer-down handoff does not recover when target keeps focus', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.pointerDown('username');
      h.blur('password');
      h.stabilizer.focusChanged('password', false);
      h.focus('username');
      h.stabilizer.focusChanged('username', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.recoveryCalls, 0);
      expect(h.focused, contains('username'));
    });

    test('password rebound gets exactly one bounded recovery', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.pointerDown('username');
      h.blur('password');
      h.stabilizer.focusChanged('password', false);
      h.focus('username');
      h.stabilizer.focusChanged('username', true);
      h.blur('username');
      h.stabilizer.focusChanged('username', false);
      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.recoveryCalls, 1);
      expect(h.focused, contains('username'));
      h.scheduler.elapse(const Duration(minutes: 1));
      expect(h.recoveryCalls, 1);
    });

    test('blank-space or button taps have no recovery target', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.recoveryCalls, 0);
    });

    test('recovered target starts its own hidden-IME watchdog', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.pointerDown('username');
      h.blur('password');
      h.stabilizer.focusChanged('password', false);
      h.focus('username');
      h.stabilizer.focusChanged('username', true);
      h.blur('username');
      h.stabilizer.focusChanged('username', false);
      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.recoveryCalls, 1);
      expect(h.showCalls, greaterThan(1));
    });

    test('password rebound to captcha recovers and watches the hidden IME', () {
      final h = _Harness();

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.pointerDown('captcha');
      h.blur('password');
      h.stabilizer.focusChanged('password', false);
      h.focus('captcha');
      h.stabilizer.focusChanged('captcha', true);
      h.blur('captcha');
      h.stabilizer.focusChanged('captcha', false);
      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.recoveryCalls, 1);
      expect(h.focused, contains('captcha'));
      expect(h.showCalls, greaterThan(1));
      expect(h.logs, anyElement(contains('target=captcha')));
      h.scheduler.elapse(const Duration(minutes: 1));
      expect(h.recoveryCalls, 1);
    });

    test(
      'cancelled captcha token cannot recover after the field disappears',
      () {
        final h = _Harness();

        h.focus('password');
        h.stabilizer.focusChanged('password', true);
        h.stabilizer.pointerDown('captcha');
        h.stabilizer.cancel();
        h.scheduler.elapse(const Duration(seconds: 2));

        expect(h.recoveryCalls, 0);
        expect(h.showCalls, 0);
      },
    );

    test('disabled non-Android path is a no-op', () {
      final h = _Harness(enabled: false);

      h.focus('password');
      h.stabilizer.focusChanged('password', true);
      h.stabilizer.pointerDown('username');
      h.focus('username');
      h.stabilizer.focusChanged('username', true);
      h.scheduler.elapse(const Duration(seconds: 2));

      expect(h.showCalls, 0);
      expect(h.recoveryCalls, 0);
    });
  });
}

class _Harness {
  _Harness({this.enabled = true, this.onDismissed}) {
    stabilizer = AuthImeStabilizer<String>(
      enabled: enabled,
      isFocused: (target) => focused.contains(target),
      isImeVisible: () => imeVisible,
      showIme: () => showCalls++,
      releaseFocus: (target) {
        releaseFocusCalls++;
        blur(target);
      },
      requestFocus: (target) {
        recoveryCalls++;
        focus(target);
        stabilizer.focusChanged(target, true);
      },
      recoverySource: 'password',
      onRecoverySourceDismissed: onDismissed,
      now: () => scheduler.now,
      schedule: scheduler.schedule,
      targetName: (target) => target,
      onLog: logs.add,
    );
  }

  final bool enabled;
  final void Function(String target)? onDismissed;
  final _FakeScheduler scheduler = _FakeScheduler();
  final Set<String> focused = <String>{};
  bool imeVisible = false;
  int showCalls = 0;
  int releaseFocusCalls = 0;
  int recoveryCalls = 0;
  final List<String> logs = <String>[];
  late final AuthImeStabilizer<String> stabilizer;

  void focus(String target) {
    focused
      ..clear()
      ..add(target);
  }

  void blur(String target) => focused.remove(target);
}

class _FakeScheduler {
  Duration now = Duration.zero;
  final List<_FakeTask> _tasks = <_FakeTask>[];

  int get pendingCount => _tasks.where((task) => !task.cancelled).length;

  AuthImeTimer schedule(Duration delay, void Function() callback) {
    final task = _FakeTask(now + delay, callback);
    _tasks.add(task);
    return task;
  }

  void elapse(Duration duration) {
    final end = now + duration;
    while (true) {
      _FakeTask? next;
      for (final task in _tasks) {
        if (task.cancelled || task.due.compareTo(end) > 0) continue;
        if (next == null || task.due.compareTo(next.due) < 0) next = task;
      }
      if (next == null) break;
      _tasks.remove(next);
      now = next.due;
      if (!next.cancelled) next.callback();
    }
    now = end;
    _tasks.removeWhere((task) => task.cancelled);
  }
}

class _FakeTask implements AuthImeTimer {
  _FakeTask(this.due, this.callback);

  final Duration due;
  final void Function() callback;
  bool cancelled = false;

  @override
  void cancel() => cancelled = true;
}
