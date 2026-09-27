import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../providers.dart';

bool get supportsNativeAppleSignIn =>
    !kIsWeb && defaultTargetPlatform == TargetPlatform.iOS;

typedef AppleAuthorization = ({
  AppleCredentialRequest request,
  String userIdentifier,
});

/// Native proofs are short lived in memory. Only the Apple user identifier and
/// forum session jti survive locally, to check revocation without affecting a
/// later password/Google/GitHub session (including the same forum account).
class AppleSignInService {
  AppleSignInService({
    this._channel = const MethodChannel('yourtj/apple-auth'),
    this._storage = const FlutterSecureStorage(),
  }) {
    _channel.setMethodCallHandler((call) async {
      if (call.method == 'credentialsChanged') _changes.add(null);
    });
  }
  final MethodChannel _channel;
  final FlutterSecureStorage _storage;
  final _changes = StreamController<void>.broadcast();
  static const _key = 'yourtj.apple-session.v1';
  Stream<void> get changes => _changes.stream;

  Future<AppleAuthorization?> authorize() async {
    final random = Random.secure();
    final nonce = base64UrlEncode(
      List.generate(32, (_) => random.nextInt(256)),
    ).replaceAll('=', '');
    try {
      final result = await _channel.invokeMapMethod<String, dynamic>(
        'authorize',
        {'nonce': nonce},
      );
      final code = result?['authorizationCode'] as String?;
      final token = result?['identityToken'] as String?;
      final user = result?['userIdentifier'] as String?;
      if (code == null ||
          code.isEmpty ||
          token == null ||
          token.isEmpty ||
          user == null ||
          user.isEmpty) {
        throw const FormatException('Incomplete Apple authorization');
      }
      return (
        request: AppleCredentialRequest(
          authorizationCode: code,
          identityToken: token,
          nonce: nonce,
        ),
        userIdentifier: user,
      );
    } on PlatformException catch (error) {
      if (error.code == 'cancelled') return null;
      rethrow;
    }
  }

  Future<void> rememberSession(String token, String? userIdentifier) async {
    if (userIdentifier == null) {
      await _storage.delete(key: _key);
      return;
    }
    final session = _sessionID(token);
    if (session == null) {
      throw const FormatException('Missing forum session ID');
    }
    await _storage.write(
      key: _key,
      value: jsonEncode({'user': userIdentifier, 'session': session}),
    );
  }

  Future<bool> isRevoked(String token) async {
    final raw = await _storage.read(key: _key);
    if (raw == null) return false;
    final marker = jsonDecode(raw) as Map<String, dynamic>;
    final session = _sessionID(token);
    if (session == null || session != marker['session']) return false;
    final user = marker['user'];
    if (user is! String || user.isEmpty) return false;
    final state = await _channel.invokeMethod<String>('credentialState', {
      'userIdentifier': user,
    });
    return state == 'revoked' || state == 'notFound' || state == 'transferred';
  }

  // This is a local session correlation hint, never authentication/authorization.
  static String? _sessionID(String token) {
    try {
      final parts = token.split('.');
      if (parts.length != 3) return null;
      final claims =
          jsonDecode(
                utf8.decode(base64Url.decode(base64Url.normalize(parts[1]))),
              )
              as Map<String, dynamic>;
      final value = claims['jti'];
      return value is String && value.isNotEmpty ? value : null;
    } catch (_) {
      return null;
    }
  }

  void dispose() {
    _channel.setMethodCallHandler(null);
    unawaited(_changes.close());
  }
}

final appleSignInProvider = Provider<AppleSignInService>((ref) {
  final service = AppleSignInService();
  ref.onDispose(service.dispose);
  return service;
});

final appleAuthBootstrapProvider = Provider<void>((ref) {
  if (!supportsNativeAppleSignIn) return;
  final service = ref.watch(appleSignInProvider);
  var disposed = false;
  var checking = false;
  Future<void> check() async {
    if (disposed || checking) return;
    checking = true;
    final epoch = ref.read(offlineCacheEpochProvider);
    final client = ref.read(apiClientProvider);
    try {
      final token = await ref.read(tokenStorageProvider).read();
      if (disposed || token == null || token.isEmpty) return;
      final revoked = await service.isRevoked(token);
      if (!disposed &&
          revoked &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        client.onUnauthorized?.call();
      }
    } catch (_) {
      // A temporary Keychain/Apple/network error is not proof of revocation.
      // The next foreground transition retries without destroying the session.
    } finally {
      checking = false;
    }
  }

  final observer = _AppleLifecycle(() => unawaited(check()));
  WidgetsBinding.instance.addObserver(observer);
  final subscription = service.changes.listen((_) => unawaited(check()));
  ref.onDispose(() {
    disposed = true;
    WidgetsBinding.instance.removeObserver(observer);
    unawaited(subscription.cancel());
  });
  unawaited(check());
});

class _AppleLifecycle with WidgetsBindingObserver {
  _AppleLifecycle(this.check);
  final VoidCallback check;
  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) check();
  }
}
