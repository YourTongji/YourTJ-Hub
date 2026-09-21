import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/auth/android_oidc_coordinator.dart';
import 'package:auth/auth.dart';

void main() {
  test(
    'subscribes before launching and completes the matching callback',
    () async {
      final events = <String>[];
      final source = _FakeCallbackSource(
        onListen: () => events.add('subscribe'),
      );
      final session = _session('google', 'state-google');
      final coordinator = AndroidOidcCoordinator(
        begin: (provider) {
          events.add('begin:$provider');
          return session;
        },
        complete: (received, callback) async {
          events.add('complete');
          expect(received, same(session));
          expect(callback.queryParameters['state'], session.state);
          return true;
        },
        reportError: (_) {},
        callbackSource: source,
        launchExternal: (uri) async {
          events.add('launch');
          source.add(
            Uri.parse(
              'yourtj://callback?code=code&state=${uri.queryParameters['state']}',
            ),
          );
          return true;
        },
        timeout: const Duration(seconds: 1),
      );

      expect(await coordinator.login(provider: 'google'), isTrue);
      expect(events.indexOf('subscribe'), lessThan(events.indexOf('launch')));
      expect(events, contains('complete'));
    },
  );

  test(
    'stale state and wrong endpoint callbacks do not complete the session',
    () async {
      final source = _FakeCallbackSource();
      final session = _session('github', 'fresh-state');
      var completions = 0;
      final coordinator = AndroidOidcCoordinator(
        begin: (_) => session,
        complete: (_, _) async {
          completions++;
          return true;
        },
        reportError: (_) {},
        callbackSource: source,
        launchExternal: (_) async {
          source.add(
            Uri.parse('other://callback?code=stale&state=${session.state}'),
          );
          source.add(Uri.parse('yourtj://callback?code=stale&state=old-state'));
          source.add(
            Uri.parse('yourtj://callback?code=good&state=${session.state}'),
          );
          return true;
        },
        timeout: const Duration(seconds: 1),
      );

      expect(await coordinator.login(provider: 'github'), isTrue);
      expect(completions, 1);
    },
  );

  test(
    'launch false and launch exception are normal errors without completion',
    () async {
      for (final launch in <Future<bool> Function(Uri)>[
        (_) async => false,
        (_) async => throw StateError('browser unavailable'),
      ]) {
        final source = _FakeCallbackSource();
        final errors = <String>[];
        var completions = 0;
        final coordinator = AndroidOidcCoordinator(
          begin: (_) => _session('tongji', 'state'),
          complete: (_, _) async {
            completions++;
            return true;
          },
          reportError: errors.add,
          callbackSource: source,
          launchExternal: launch,
          timeout: const Duration(seconds: 1),
        );

        expect(await coordinator.login(provider: 'tongji'), isFalse);
        expect(completions, 0);
        expect(errors, isNotEmpty);
      }
    },
  );

  test('cancellation stops a pending attempt', () async {
    final source = _FakeCallbackSource();
    final launchStarted = Completer<void>();
    final launchRelease = Completer<bool>();
    final coordinator = AndroidOidcCoordinator(
      begin: (_) => _session('google', 'state'),
      complete: (_, _) async => true,
      reportError: (_) {},
      callbackSource: source,
      launchExternal: (_) {
        launchStarted.complete();
        return launchRelease.future;
      },
      timeout: const Duration(minutes: 5),
    );

    final pending = coordinator.login(provider: 'google');
    await launchStarted.future;
    coordinator.cancel();

    expect(await pending, isFalse);
    launchRelease.complete(true);
  });

  test(
    'a matching cold initial callback completes after browser launch',
    () async {
      final session = _session('tongji', 'cold-state');
      final source = _FakeCallbackSource(
        initialCallback: Uri.parse(
          'yourtj://callback?code=cold&state=${session.state}',
        ),
      );
      var completions = 0;
      final coordinator = AndroidOidcCoordinator(
        begin: (_) => session,
        complete: (_, _) async {
          completions++;
          return true;
        },
        reportError: (_) {},
        callbackSource: source,
        launchExternal: (_) async => true,
        timeout: const Duration(seconds: 1),
      );

      expect(await coordinator.login(provider: 'tongji'), isTrue);
      expect(completions, 1);
    },
  );
}

OidcAuthorizationSession _session(String provider, String state) {
  return OidcAuthorizationSession(
    authorizationUri: Uri.parse('https://issuer/authorize?state=$state'),
    state: state,
    nonce: 'nonce',
    codeVerifier: 'verifier-verifier-verifier-verifier-verifier-verifier',
    redirectUri: 'yourtj://callback',
    provider: provider,
  );
}

class _FakeCallbackSource implements OidcCallbackSource {
  _FakeCallbackSource({this.onListen, this.initialCallback});

  final void Function()? onListen;
  final Uri? initialCallback;
  late final StreamController<Uri> _controller =
      StreamController<Uri>.broadcast(sync: true, onListen: onListen);

  @override
  Stream<Uri> get callbacks => _controller.stream;

  @override
  Future<Uri?> getInitialCallback() async => initialCallback;

  void add(Uri callback) => _controller.add(callback);
}
