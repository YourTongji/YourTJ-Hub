import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/app_locale.dart';
import 'package:forum_app/src/site_theme.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_platform_interface.dart';

class DelayedPreferences extends InMemorySharedPreferencesStore {
  DelayedPreferences() : super.empty();
  final started = Completer<void>();
  final release = Completer<void>();
  @override
  Future<bool> setValue(String type, String key, Object value) async {
    if (!started.isCompleted) started.complete();
    await release.future;
    return super.setValue(type, key, value);
  }
}

class NoTheme extends ThemeRepository {
  NoTheme() : super(GfApiClient(dio: Dio(), tokenStorage: NoTokens()));
  @override
  Future<SiteThemePublicPayload?> fetchTokens() async => null;
}

class NoTokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  for (final owner in ['theme', 'locale', 'site']) {
    test('reset drains an already issued $owner preference write', () async {
      SharedPreferences.setMockInitialValues({});
      final store = DelayedPreferences();
      SharedPreferencesStorePlatform.instance = store;
      final container = ProviderContainer(
        overrides: [themeRepositoryProvider.overrideWithValue(NoTheme())],
      );
      addTearDown(container.dispose);
      late Future<void> Function() reset;
      if (owner == 'theme') {
        final notifier = container.read(themeModeProvider.notifier);
        notifier.setMode(ThemeMode.dark);
        reset = notifier.resetToDefault;
      } else if (owner == 'locale') {
        final notifier = container.read(appLocaleProvider.notifier);
        notifier.setLocale(const Locale('en'));
        reset = notifier.resetToDefault;
      } else {
        final notifier = container.read(siteThemeProvider.notifier);
        unawaited(notifier.setFollowing(false));
        reset = notifier.resetToDefault;
      }
      await store.started.future;
      var completed = false;
      final operation = reset().then((_) => completed = true);
      await Future<void>.delayed(Duration.zero);
      final completedBeforeWrite = completed;
      store.release.complete();
      await operation;
      await Future<void>.delayed(Duration.zero);
      expect(completedBeforeWrite, isFalse);
      expect(await store.getAll(), isEmpty);
    });
  }
}
