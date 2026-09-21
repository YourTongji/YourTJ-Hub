import 'dart:async';

import 'package:auth/auth.dart';

/// Dart-side source for callbacks forwarded by Android MainActivity.
abstract interface class OidcCallbackSource {
  Stream<Uri> get callbacks;

  Future<Uri?> getInitialCallback();
}

typedef OidcManualBegin = OidcAuthorizationSession Function(String provider);
typedef OidcManualComplete =
    Future<bool> Function(OidcAuthorizationSession session, Uri callbackUri);
typedef OidcExternalLauncher = Future<bool> Function(Uri uri);

/// Coordinates one Android external-browser authorization attempt.
///
/// This class has no platform-channel dependency so its ordering, cancellation
/// and callback filtering can be tested with a pure Dart stream and launcher.
class AndroidOidcCoordinator {
  AndroidOidcCoordinator({
    required this.begin,
    required this.complete,
    required this.reportError,
    required this.callbackSource,
    required this.launchExternal,
    this.timeout = const Duration(minutes: 5),
  });

  final OidcManualBegin begin;
  final OidcManualComplete complete;
  final void Function(String message) reportError;
  final OidcCallbackSource callbackSource;
  final OidcExternalLauncher launchExternal;
  final Duration timeout;

  StreamSubscription<Uri>? _subscription;
  Completer<void>? _cancelCompleter;
  bool _active = false;

  Future<bool> login({required String provider}) async {
    if (_active) {
      reportError('OIDC login already in progress');
      return false;
    }
    _active = true;
    final Completer<void> cancelCompleter = Completer<void>();
    _cancelCompleter = cancelCompleter;
    final Completer<Uri> callbackCompleter = Completer<Uri>();
    Timer? timeoutTimer;

    try {
      final OidcAuthorizationSession session = begin(provider);
      _subscription = callbackSource.callbacks.listen(
        (callback) => _acceptCallback(session, callback, callbackCompleter),
        onError: (_, _) {
          if (!callbackCompleter.isCompleted) {
            callbackCompleter.completeError(StateError('OIDC callback failed'));
          }
        },
      );

      final Object launchResult = await Future.any<Object>(<Future<Object>>[
        launchExternal(
          session.authorizationUri,
        ).then<Object>((launched) => launched),
        cancelCompleter.future.then<Object>((_) => const _OidcCancelled()),
      ]);
      if (launchResult is _OidcCancelled) {
        reportError('OIDC authorization cancelled');
        return false;
      }
      if (launchResult is! bool || !launchResult) {
        reportError('OIDC browser unavailable');
        return false;
      }

      try {
        final Uri? initialCallback = await callbackSource.getInitialCallback();
        if (initialCallback != null) {
          _acceptCallback(session, initialCallback, callbackCompleter);
        }
      } catch (_) {
        // The event channel remains authoritative for a warm callback. A
        // transient initial-intent read failure must not crash the login page.
      }

      final bool result = await Future.any<bool>(<Future<bool>>[
        callbackCompleter.future.then<bool>(
          (callback) => complete(session, callback),
        ),
        cancelCompleter.future.then<bool>((_) {
          reportError('OIDC authorization cancelled');
          return false;
        }),
        (() {
          final Completer<bool> timeoutCompleter = Completer<bool>();
          timeoutTimer = Timer(timeout, () {
            if (timeoutCompleter.isCompleted) return;
            reportError('OIDC authorization timed out');
            timeoutCompleter.complete(false);
          });
          return timeoutCompleter.future;
        })(),
      ]);
      return result;
    } catch (_) {
      reportError('OIDC authorization failed');
      return false;
    } finally {
      timeoutTimer?.cancel();
      await _subscription?.cancel();
      _subscription = null;
      if (identical(_cancelCompleter, cancelCompleter)) {
        _cancelCompleter = null;
      }
      _active = false;
    }
  }

  /// Cancels the pending browser wait and detaches the callback listener.
  void cancel() {
    if (!_active) return;
    if (!(_cancelCompleter?.isCompleted ?? true)) {
      _cancelCompleter!.complete();
    }
    unawaited(_subscription?.cancel());
  }

  void dispose() => cancel();

  void _acceptCallback(
    OidcAuthorizationSession session,
    Uri callback,
    Completer<Uri> callbackCompleter,
  ) {
    if (callbackCompleter.isCompleted || !_matchesSession(session, callback)) {
      return;
    }
    callbackCompleter.complete(callback);
  }

  static bool _matchesSession(OidcAuthorizationSession session, Uri callback) {
    final Uri expected = Uri.parse(session.redirectUri);
    final List<String>? stateValues = callback.queryParametersAll['state'];
    return callback.scheme == expected.scheme &&
        callback.host == expected.host &&
        callback.port == expected.port &&
        callback.path == expected.path &&
        callback.userInfo.isEmpty &&
        callback.fragment.isEmpty &&
        stateValues?.length == 1 &&
        stateValues!.single == session.state;
  }
}

class _OidcCancelled {
  const _OidcCancelled();
}
