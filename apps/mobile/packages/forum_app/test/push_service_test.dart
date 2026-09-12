import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/push/push_service.dart';

class MemoryStorage implements TokenStorage {
  String? value = 'session';
  @override
  Future<String?> read() async => value;
  @override
  Future<void> write(String token) async {
    value = token;
  }

  @override
  Future<void> clear() async {
    value = null;
  }
}

class Driver extends PushDriver {
  String transport = 'apns';
  bool supported = true, allowed = true;
  int requests = 0, registrations = 0, stopped = 0;
  String? deviceToken = 'native-token';
  void Function(String)? changed;
  Completer<String?>? pending;
  Completer<void>? pendingStop;
  @override
  String get platform => transport == 'apns' ? 'ios' : 'android';
  @override
  String get provider => transport;
  @override
  Future<bool> configured() async => supported;
  @override
  Future<bool> permission({required bool request}) async {
    if (request) requests++;
    return allowed;
  }

  @override
  Future<String?> token() async {
    registrations++;
    return pending == null ? deviceToken : pending!.future;
  }

  @override
  Future<void> stop() async {
    stopped++;
    await pendingStop?.future;
  }

  @override
  Future<String?> initialRoute() async => null;
  @override
  void listen(
    void Function(String) tokenChanged,
    void Function(String) opened,
  ) {
    changed = tokenChanged;
  }

  @override
  void dispose() {}
}

class Repository extends PushRepository {
  Repository() : super(GfApiClient(dio: Dio(), tokenStorage: MemoryStorage()));
  PushConfigPayload configValue = const PushConfigPayload(
    configured: false,
    native: NativePushConfig(
      apnsEnabled: true,
      fcmEnabled: false,
      jpushEnabled: true,
    ),
  );
  Completer<bool>? pendingRegistration;
  Completer<void>? pendingSession;
  @override
  Future<PushRepository> forSession() async {
    await pendingSession?.future;
    return this;
  }

  final registered = <String>[];
  final unregistered = <String>[];
  bool reject = false, offline = false;
  int reads = 0;
  @override
  Future<PushConfigPayload> config() async {
    reads++;
    if (offline) throw StateError('offline');
    return configValue;
  }

  @override
  Future<bool> registerDevice({
    required String platform,
    required String token,
    String? provider,
  }) async {
    registered.add('$platform:$provider:$token');
    return pendingRegistration == null ? !reject : pendingRegistration!.future;
  }

  @override
  Future<bool> unregisterDevice({required String token}) async {
    unregistered.add(token);
    if (offline) throw StateError('offline');
    return true;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Driver driver;
  late Repository repo;
  late MemoryStorage storage;
  late ProviderContainer container;
  CurrentUser? user;
  Future<void> settle() => pumpEventQueue();
  setUp(() {
    // Steady state: the iOS one-shot permission request already happened, so
    // startup refreshes stay silent. First-launch behavior gets its own tests
    // that clear the marker.
    SharedPreferences.setMockInitialValues(
      const {'push_permission_requested': true},
    );
    driver = Driver();
    repo = Repository();
    storage = MemoryStorage();
    user = const CurrentUser(id: 1, username: 'one');
    container = ProviderContainer(
      overrides: [
        pushDriverProvider.overrideWithValue(driver),
        pushRepositoryProvider.overrideWithValue(repo),
        tokenStorageProvider.overrideWithValue(storage),
        currentUserProvider.overrideWith((ref) async => user),
      ],
    );
    container.read(pushControllerProvider);
  });
  tearDown(() => container.dispose());
  PushController controller() =>
      container.read(pushControllerProvider.notifier);
  PushChannelStatus status() => container.read(pushControllerProvider);
  test(
    'enable waits for a startup refresh and preserves explicit consent',
    () async {
      repo.pendingSession = Completer<void>();
      await settle();
      final enabling = controller().enable();
      await settle();
      repo.pendingSession!.complete();
      await enabling;
      await settle();
      expect(status(), PushChannelStatus.enabled);
      expect(driver.requests, 1);
    },
  );
  test('enable during stop runs after cleanup completes', () async {
    await settle();
    await controller().enable();
    driver.pendingStop = Completer<void>();
    final stopping = controller().disable();
    await settle();
    final enabling = controller().enable();
    driver.pendingStop!.complete();
    await stopping;
    await enabling;
    await settle();
    expect(status(), PushChannelStatus.enabled);
    expect(driver.requests, 2);
  });
  test('a later disable cancels queued consent', () async {
    repo.pendingSession = Completer<void>();
    await settle();
    final enabling = controller().enable();
    await controller().disable();
    repo.pendingSession!.complete();
    await enabling;
    await settle();
    expect(status(), PushChannelStatus.disabled);
    expect(driver.requests, 0);
  });
  test(
    'disabled refresh retries offline cleanup without requesting permission',
    () async {
      await settle();
      await controller().enable();
      repo.offline = true;
      await controller().disable();
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('push_token'), 'native-token');
      repo.offline = false;
      driver.supported = false;
      await controller().refresh();
      expect(prefs.getString('push_token'), isNull);
      expect(repo.unregistered, ['native-token', 'native-token']);
      expect(driver.requests, 1);
      expect(status(), PushChannelStatus.disabled);
    },
  );
  test('another account cannot acknowledge retained owner cleanup', () async {
    await settle();
    await controller().enable();
    repo.offline = true;
    await controller().handleLogout();
    user = const CurrentUser(id: 2, username: 'two');
    container.invalidate(currentUserProvider);
    await settle();
    repo.offline = false;
    repo.unregistered.clear();
    await controller().refresh();
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString('push_token'), 'native-token');
    expect(repo.unregistered, isEmpty);
    user = const CurrentUser(id: 1, username: 'one');
    container.invalidate(currentUserProvider);
    await settle();
    await controller().refresh();
    expect(prefs.getString('push_token'), isNull);
  });
  test(
    'APNs available without Firebase; no permission prompt before consent',
    () async {
      await settle();
      expect(status(), PushChannelStatus.disabled);
      expect(driver.requests, 0);
      await controller().enable();
      expect(status(), PushChannelStatus.enabled);
      expect(repo.registered, ['ios:apns:native-token']);
      expect(driver.requests, 1);
    },
  );
  test(
    'iOS first launch after login requests permission and treats grant as consent',
    () async {
      SharedPreferences.setMockInitialValues({});
      await settle();
      expect(driver.requests, 1);
      expect(status(), PushChannelStatus.enabled);
      expect(repo.registered, ['ios:apns:native-token']);
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getBool('push_enabled'), true);
      expect(prefs.getBool('push_permission_requested'), true);
    },
  );
  test('first-launch request happens once; later refreshes restore silently',
      () async {
    SharedPreferences.setMockInitialValues({});
    await settle();
    expect(driver.requests, 1);
    await controller().refresh();
    expect(driver.requests, 1);
    expect(status(), PushChannelStatus.enabled);
  });
  test('iOS first-launch denial keeps the denied recovery state', () async {
    SharedPreferences.setMockInitialValues({});
    driver.allowed = false;
    await settle();
    expect(driver.requests, 1);
    expect(status(), PushChannelStatus.permissionDenied);
    expect(repo.registered, isEmpty);
    // The marker is recorded, so the next launch does not ask again.
    await controller().refresh();
    expect(driver.requests, 1);
  });
  test('Android first launch never auto-requests permission', () async {
    SharedPreferences.setMockInitialValues({});
    driver.transport = 'jpush';
    await settle();
    expect(driver.requests, 0);
    expect(status(), PushChannelStatus.disabled);
  });
  test('Android uses JPush registration, never FCM/APNs token', () async {
    driver.transport = 'jpush';
    await settle();
    await controller().enable();
    expect(repo.registered, ['android:jpush:native-token']);
  });
  test('repeated native token callbacks do not restart registration', () async {
    await settle();
    await controller().enable();
    driver.changed!('native-token');
    driver.changed!('native-token');
    await settle();
    expect(driver.registrations, 1);
    expect(repo.registered, hasLength(1));
  });
  test('empty device token must not report enabled', () async {
    driver.deviceToken = '';
    await settle();
    await controller().enable();
    expect(status(), PushChannelStatus.registrationFailed);
    expect(repo.registered, isEmpty);
  });
  test('server rejects registration -> failure', () async {
    repo.reject = true;
    await settle();
    await controller().enable();
    expect(status(), PushChannelStatus.registrationFailed);
  });
  test('permission denied does not register', () async {
    driver.allowed = false;
    await settle();
    await controller().enable();
    expect(status(), PushChannelStatus.permissionDenied);
    expect(repo.registered, isEmpty);
  });
  test(
    'only matching server transport enables delivery; permission still requested',
    () async {
      repo.configValue = const PushConfigPayload(
        configured: false,
        native: NativePushConfig(apnsEnabled: false, fcmEnabled: true),
      );
      await settle();
      await controller().enable();
      expect(status(), PushChannelStatus.serverDisabled);
      expect(driver.requests, 1);
      expect(driver.registrations, 0);
    },
  );
  test('server outage is a visible failure, not disabled channel', () async {
    repo.offline = true;
    await settle();
    await controller().enable();
    expect(status(), PushChannelStatus.registrationFailed);
  });
  test('guest makes no authenticated requests', () async {
    storage.value = null;
    await settle();
    await controller().enable();
    expect(repo.reads, 0);
    expect(driver.requests, 0);
  });
  test(
    'resume restores existing consent without requesting permission again',
    () async {
      await settle();
      await controller().enable();
      await controller().refresh();
      expect(driver.requests, 1);
      expect(repo.registered, hasLength(2));
    },
  );
  test('disable cancels late token registration', () async {
    await settle();
    driver.pending = Completer<String?>();
    final enabling = controller().enable();
    await settle();
    await controller().disable();
    driver.pending!.complete('late');
    await enabling;
    expect(repo.registered, isEmpty);
    expect(status(), PushChannelStatus.disabled);
  });
  test('account boundary cancels late token registration', () async {
    await settle();
    driver.pending = Completer<String?>();
    final enabling = controller().enable();
    await settle();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    driver.pending!.complete('late');
    await enabling;
    expect(repo.registered, isEmpty);
  });
  test(
    'logout waits for in-flight server registration and removes it',
    () async {
      await settle();
      repo.pendingRegistration = Completer<bool>();
      final enabling = controller().enable();
      await settle();
      final logout = controller().handleLogout();
      await settle();
      repo.pendingRegistration!.complete(true);
      await enabling;
      await logout;
      expect(repo.unregistered, contains('native-token'));
      expect(status(), PushChannelStatus.disabled);
    },
  );
  test(
    'offline cleanup of a late registration is retried while disabled',
    () async {
      await settle();
      repo.pendingRegistration = Completer<bool>();
      final enabling = controller().enable();
      await settle();
      repo.offline = true;
      final stopping = controller().disable();
      repo.pendingRegistration!.complete(true);
      await enabling;
      await stopping;
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('push_token'), 'native-token');
      expect(prefs.getInt('push_token_owner'), 1);
      repo.offline = false;
      await controller().refresh();
      expect(prefs.getString('push_token'), isNull);
      expect(prefs.getInt('push_token_owner'), isNull);
      expect(status(), PushChannelStatus.disabled);
    },
  );
  test(
    'account boundary clears consent before any automatic refresh',
    () async {
      await settle();
      await controller().enable();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await settle();
      await controller().refresh();
      expect(status(), PushChannelStatus.disabled);
      expect(
        (await SharedPreferences.getInstance()).getBool('push_enabled'),
        false,
      );
    },
  );
  test('logout stops native delivery and unbinds token', () async {
    await settle();
    await controller().enable();
    await controller().handleLogout();
    expect(repo.unregistered, ['native-token']);
    expect(driver.stopped, greaterThan(0));
    expect(
      (await SharedPreferences.getInstance()).getBool('push_enabled'),
      false,
    );
  });
  test('notification navigation is restricted to actual app routes', () {
    for (final route in ['/p/12', '/u/34', '/notifications']) {
      expect(pushRoute(route), route);
    }
    for (final route in [
      'https://evil.test',
      '//evil.test/p/12',
      '/admin',
      '/p/no',
      '/login',
    ]) {
      expect(pushRoute(route), null);
    }
  });
}
