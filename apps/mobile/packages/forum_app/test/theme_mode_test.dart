import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_platform_interface.dart';

class _DelayedPreferences extends InMemorySharedPreferencesStore {
  _DelayedPreferences({this.failFirst = false}) : super.empty();

  final bool failFirst;
  final firstWrite = Completer<void>();
  final secondWrite = Completer<void>();
  final started = <Object>[];

  @override
  Future<bool> setValue(String type, String key, Object value) async {
    started.add(value);
    if (started.length == 1) {
      await firstWrite.future;
      if (failFirst) throw StateError('Device storage unavailable');
    }
    final written = await super.setValue(type, key, value);
    if (started.length == 2) secondWrite.complete();
    return written;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));

  test('sends the saved Android theme to the system splash screen', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    const channel = MethodChannel('yourtj/startup');
    final nativeMode = Completer<String>();
    final messenger =
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
    messenger.setMockMethodCallHandler(channel, (call) async {
      nativeMode.complete((call.arguments as Map)['mode'] as String);
      return true;
    });
    addTearDown(() {
      messenger.setMockMethodCallHandler(channel, null);
      debugDefaultTargetPlatformOverride = null;
    });

    final container = ProviderContainer();
    addTearDown(container.dispose);
    container.read(themeModeProvider.notifier).setMode(ThemeMode.dark);
    expect(await nativeMode.future, 'dark');
  });

  test(
    'a new theme choice wins over asynchronous preference restoration',
    () async {
      SharedPreferences.setMockInitialValues({'theme_mode': 'dark'});
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(themeModeProvider.notifier).setMode(ThemeMode.light);
      await Future<void>.delayed(Duration.zero);
      expect(container.read(themeModeProvider), ThemeMode.light);
    },
  );

  test(
    'theme writes finish in selection order, including follow system',
    () async {
      final store = _DelayedPreferences();
      SharedPreferencesStorePlatform.instance = store;
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(themeModeProvider.notifier).setMode(ThemeMode.dark);
      container.read(themeModeProvider.notifier).setMode(ThemeMode.system);
      await Future<void>.delayed(Duration.zero);
      final whileFirstPending = List<Object>.of(store.started);
      store.firstWrite.complete();
      await store.secondWrite.future;
      expect(whileFirstPending, ['dark']);
      expect((await store.getAll())['flutter.theme_mode'], 'system');
      expect(container.read(themeModeProvider), ThemeMode.system);
    },
  );

  for (final mode in ThemeMode.values) {
    test('restores the saved ${mode.name} theme after restart', () async {
      SharedPreferences.setMockInitialValues({'theme_mode': mode.name});
      final container = ProviderContainer();
      addTearDown(container.dispose);
      expect(container.read(themeModeProvider), ThemeMode.system);
      await Future<void>.delayed(Duration.zero);
      expect(container.read(themeModeProvider), mode);
    });
  }

  test(
    'a failed write does not prevent a later choice from being saved',
    () async {
      final store = _DelayedPreferences(failFirst: true);
      SharedPreferencesStorePlatform.instance = store;
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(themeModeProvider.notifier).setMode(ThemeMode.dark);
      container.read(themeModeProvider.notifier).toggleDark(false);
      await Future<void>.delayed(Duration.zero);
      store.firstWrite.complete();
      await Future<void>.delayed(Duration.zero);
      expect((await store.getAll())['flutter.theme_mode'], 'light');
      expect(container.read(themeModeProvider), ThemeMode.light);
    },
  );
}
