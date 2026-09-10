import 'dart:async';
import 'dart:io';

import 'package:app_settings/app_settings.dart';
import 'package:core/core.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../current_user.dart';
import '../providers.dart';
import '../router.dart';

enum PushChannelStatus {
  unknown,
  unsupported,
  serverDisabled,
  disabled,
  enabled,
  permissionDenied,
  registrationFailed,
}

/// Native bridge: iOS APNs; Android JPush + configured OEM offline channels.
/// No SDK initialization or device collection before the user enables push.
class PushDriver {
  static const channel = MethodChannel('yourtj/push');
  String get platform => Platform.isIOS ? 'ios' : 'android';
  String get provider => Platform.isIOS ? 'apns' : 'jpush';
  Future<bool> configured() async =>
      await channel.invokeMethod<bool>('configured') ?? false;
  Future<bool> permission({required bool request}) async =>
      await channel.invokeMethod<bool>(
        request ? 'requestPermission' : 'permission',
      ) ??
      false;
  Future<String?> token() => channel.invokeMethod<String>('register');
  Future<void> stop() => channel.invokeMethod<void>('stop');
  Future<String?> initialRoute() =>
      channel.invokeMethod<String>('initialRoute');
  void listen(
    void Function(String) tokenChanged,
    void Function(String) opened,
  ) {
    channel.setMethodCallHandler((call) async {
      if (call.method == 'token' && call.arguments is String) {
        tokenChanged(call.arguments as String);
      } else if (call.method == 'opened' && call.arguments is String) {
        opened(call.arguments as String);
      }
    });
  }

  void dispose() => channel.setMethodCallHandler(null);
}

final pushDriverProvider = Provider<PushDriver>((ref) => PushDriver());

/// Notifications may navigate only to notification-owned in-app routes.
String? pushRoute(String? value) {
  if (value == null) return null;
  final uri = Uri.tryParse(value);
  if (uri == null || uri.hasScheme || uri.hasAuthority || uri.hasFragment) {
    return null;
  }
  if (RegExp(r'^/p/[0-9]+$').hasMatch(uri.path) ||
      RegExp(r'^/u/[0-9]+$').hasMatch(uri.path) ||
      uri.path == '/notifications') {
    return value;
  }
  return null;
}

class PushController extends Notifier<PushChannelStatus>
    with WidgetsBindingObserver {
  static const _enabledKey = 'push_enabled';
  static const _tokenKey = 'push_token';
  static const _tokenOwnerKey = 'push_token_owner';
  int _generation = 0;
  bool _disposed = false;
  bool _busy = false;
  bool _refreshAgain = false;
  Completer<void>? _pendingEnable;
  late PushDriver _driver;
  String? _lastToken;
  PushRepository? _sessionRepository;
  int? _sessionUserId;
  Future<void>? _pendingRegistration;
  Future<void>? _stopping;

  @override
  PushChannelStatus build() {
    _driver = ref.read(pushDriverProvider);
    _driver.listen((token) {
      if (token.isEmpty || token == _lastToken) return;
      _lastToken = token;
      unawaited(refresh());
    }, _open);
    WidgetsBinding.instance.addObserver(this);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      unawaited(disable());
    });
    ref.listen(currentUserProvider, (_, next) {
      if (next.hasValue) unawaited(refresh());
    });
    ref.onDispose(() {
      _disposed = true;
      _pendingEnable?.complete();
      _pendingEnable = null;
      _generation++;
      WidgetsBinding.instance.removeObserver(this);
      _driver.dispose();
    });
    Future.microtask(refresh);
    return PushChannelStatus.unknown;
  }

  bool _current(int generation, int epoch) =>
      !_disposed &&
      generation == _generation &&
      epoch == ref.read(offlineCacheEpochProvider);

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) unawaited(refresh());
  }

  Future<void> refresh() async {
    await _connect(request: false);
  }

  Future<void> enable() => _connect(request: true);

  Future<void> _connect({required bool request}) async {
    if (_disposed) return;
    if (_busy || _stopping != null) {
      if (request) {
        _pendingEnable ??= Completer<void>();
        return _pendingEnable!.future;
      }
      _refreshAgain = true;
      return;
    }
    _busy = true;
    final generation = ++_generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    final repository = ref.read(pushRepositoryProvider);
    try {
      final prefs = await SharedPreferences.getInstance();
      if (!_current(generation, epoch)) return;
      if (!await hasSessionToken(ref.read(tokenStorageProvider))) {
        if (_current(generation, epoch)) state = PushChannelStatus.disabled;
        return;
      }
      if (!_current(generation, epoch)) return;
      final session = await repository.forSession();
      if (!_current(generation, epoch)) return;
      final user = await ref.read(currentUserProvider.future);
      if (!_current(generation, epoch)) return;
      _sessionRepository = session;
      _sessionUserId = user?.id;
      if (request) await prefs.setBool(_enabledKey, true);
      if (!(prefs.getBool(_enabledKey) ?? false)) {
        state = PushChannelStatus.disabled;
        await _unregister(session, prefs, user?.id);
        return;
      }
      if (!await _driver.configured()) {
        if (_current(generation, epoch)) state = PushChannelStatus.unsupported;
        return;
      }
      if (!_current(generation, epoch)) return;
      // Explicit consent reaches the OS even while delivery configuration is absent.
      final allowed = await _driver.permission(request: request);
      if (!_current(generation, epoch)) return;
      if (!allowed) {
        state = PushChannelStatus.permissionDenied;
        await _driver.stop();
        await _unregister(session, prefs, user?.id);
        return;
      }
      final config = await session.config();
      if (!_current(generation, epoch)) return;
      final available = _driver.provider == 'apns'
          ? config.native?.apnsEnabled == true
          : config.native?.jpushEnabled == true;
      if (!available) {
        state = PushChannelStatus.serverDisabled;
        return;
      }
      final token = await _driver.token().timeout(const Duration(seconds: 25));
      if (!_current(generation, epoch)) return;
      if (token == null || token.isEmpty) throw StateError('No device token');
      _lastToken = token;
      final registration = _register(
        session,
        prefs,
        token,
        generation,
        epoch,
        user?.id,
      );
      _pendingRegistration = registration;
      try {
        await registration;
      } finally {
        _pendingRegistration = null;
      }
      if (!_current(generation, epoch)) return;
      state = PushChannelStatus.enabled;
      _open(await _driver.initialRoute());
    } catch (_) {
      // Do not log tokens, provider payloads or session credentials.
      if (_current(generation, epoch)) {
        state = PushChannelStatus.registrationFailed;
      }
    } finally {
      _busy = false;
      _drain();
    }
  }

  void _drain() {
    if (_disposed || _busy || _stopping != null) return;
    final enabling = _pendingEnable;
    _pendingEnable = null;
    if (enabling != null) {
      _refreshAgain = false;
      unawaited(
        _connect(request: true).then(
          (_) => enabling.complete(),
          onError: (Object error, StackTrace stack) =>
              enabling.completeError(error, stack),
        ),
      );
    } else if (_refreshAgain) {
      _refreshAgain = false;
      unawaited(refresh());
    }
  }

  Future<void> _register(
    PushRepository repository,
    SharedPreferences prefs,
    String token,
    int generation,
    int epoch,
    int? userId,
  ) async {
    final success = await repository.registerDevice(
      platform: _driver.platform,
      token: token,
      provider: _driver.provider,
    );
    if (!success) throw StateError('Device registration rejected');
    final previous = prefs.getString(_tokenKey);
    await prefs.setString(_tokenKey, token);
    if (userId != null) {
      await prefs.setInt(_tokenOwnerKey, userId);
    } else {
      await prefs.remove(_tokenOwnerKey);
    }
    if (!_current(generation, epoch)) {
      // Persist before cleanup so an offline late registration remains retryable.
      await _unregister(repository, prefs, userId);
      return;
    }
    if (previous != null && previous != token) {
      try {
        await repository.unregisterDevice(token: previous);
      } catch (_) {
        // Provider-expired tokens are also removed by the delivery worker.
      }
    }
  }

  Future<void> disable() {
    _pendingEnable?.complete();
    _pendingEnable = null;
    _refreshAgain = false;
    if (_stopping != null) return _stopping!;
    _generation++;
    if (!_disposed) state = PushChannelStatus.disabled;
    final PushRepository repository =
        _sessionRepository ?? ref.read<PushRepository>(pushRepositoryProvider);
    final operation = _stop(repository, _sessionUserId);
    _stopping = operation;
    return operation.whenComplete(() {
      _stopping = null;
      _drain();
    });
  }

  Future<void> _stop(PushRepository repository, int? userId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_enabledKey, false);
    try {
      await _driver.stop();
    } catch (_) {
      /* unregister still required */
    }
    // Wait only for an already issued server write, never for a pending OS token.
    try {
      await _pendingRegistration;
    } catch (_) {
      /* best effort cleanup below */
    }
    await _unregister(repository, prefs, userId);
  }

  /// Require fresh consent for the next account on a shared device.
  Future<void> handleLogout() => disable();

  Future<void> _unregister(
    PushRepository repository,
    SharedPreferences prefs,
    int? userId,
  ) async {
    final token = prefs.getString(_tokenKey);
    if (token == null) return;
    final owner = prefs.getInt(_tokenOwnerKey);
    // Unregister is owner-scoped and returns success even for another owner's
    // token. Keep pending cleanup until that account signs in again.
    if (owner != null && owner != userId) return;
    try {
      if (await repository.unregisterDevice(token: token)) {
        await prefs.remove(_tokenKey);
        await prefs.remove(_tokenOwnerKey);
      }
    } catch (_) {
      /* preserve token for retry */
    }
  }

  void _open(String? value) {
    if (_disposed || state != PushChannelStatus.enabled) return;
    final route = pushRoute(value);
    if (route != null) {
      appRouter.push(route);
      unawaited(_driver.initialRoute().catchError((_) => null));
    }
  }

  Future<void> openSystemSettings() =>
      AppSettings.openAppSettings(type: AppSettingsType.notification);
}

final pushBootstrapProvider = Provider<void>((ref) {
  ref.watch(pushControllerProvider);
});
final pushControllerProvider =
    NotifierProvider<PushController, PushChannelStatus>(PushController.new);
