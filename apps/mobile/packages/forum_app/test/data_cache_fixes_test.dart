import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/external_link_policy.dart';
import 'package:forum_app/src/link_navigation.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/site_theme.dart';
import 'package:forum_app/src/user_blocks.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _StubTokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

class _FailingThemeRepository extends ThemeRepository {
  _FailingThemeRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: _StubTokens()));

  @override
  Future<SiteThemePublicPayload?> fetchTokens() async {
    throw Exception('offline');
  }
}

class _DisabledThemeRepository extends ThemeRepository {
  _DisabledThemeRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: _StubTokens()));

  @override
  Future<SiteThemePublicPayload?> fetchTokens() async =>
      const SiteThemePublicPayload(enabled: false, themes: []);
}

class _StubExternalLinkPolicyRepo extends ExternalLinkPolicyRepository {
  _StubExternalLinkPolicyRepo(this.result, {this.fail = false})
    : super(GfApiClient(dio: Dio(), tokenStorage: _StubTokens()));
  final List<String>? result;
  final bool fail;

  @override
  Future<List<String>?> fetchTrustedDomains() async {
    if (fail) throw Exception('offline');
    return result;
  }
}

class _CountingUserRepository extends UserRepository {
  _CountingUserRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: _StubTokens()));
  int blockCalls = 0;

  @override
  Future<UserBlocksPayload> getUserBlocks() async {
    blockCalls++;
    return const UserBlocksPayload(ownerId: 7, blocks: []);
  }
}

void main() {
  test(
    'SiteThemeController applies persisted theme offline on cold start and clears when disabled',
    () async {
      SharedPreferences.setMockInitialValues({
        'follow_site_theme': true,
        'site_theme_cache': jsonEncode({
          'themes': {
            'light': {'primary': '#1d4ed8'},
            'dark': {'primary': '#60a5fa'},
          },
        }),
      });

      final offlineContainer = ProviderContainer(
        overrides: [
          themeRepositoryProvider.overrideWithValue(_FailingThemeRepository()),
        ],
      );
      addTearDown(offlineContainer.dispose);
      offlineContainer.read(siteThemeProvider);
      await Future<void>.delayed(const Duration(milliseconds: 30));
      final offlineState = offlineContainer.read(siteThemeProvider);
      expect(offlineState.available, isTrue);
      expect(offlineState.following, isTrue);
      expect(offlineState.runtime, isNotNull);

      // When server disables site theme, persisted cache is cleared.
      final disabledContainer = ProviderContainer(
        overrides: [
          themeRepositoryProvider.overrideWithValue(_DisabledThemeRepository()),
        ],
      );
      addTearDown(disabledContainer.dispose);
      disabledContainer.read(siteThemeProvider);
      await Future<void>.delayed(const Duration(milliseconds: 30));
      final disabledState = disabledContainer.read(siteThemeProvider);
      expect(disabledState.available, isFalse);
      expect(disabledState.runtime, isNull);
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('site_theme_cache'), isNull);
    },
  );

  test(
    'ExternalLinkPolicyController persists trusted domains and restores them offline',
    () async {
      SharedPreferences.setMockInitialValues({});
      LinkNavigation.applyTrustedDomains(const []);

      final online = ProviderContainer(
        overrides: [
          externalLinkPolicyRepositoryProvider.overrideWithValue(
            _StubExternalLinkPolicyRepo(const ['tongji.edu.cn']),
          ),
        ],
      );
      addTearDown(online.dispose);
      online.read(externalLinkPolicyProvider);
      await Future<void>.delayed(const Duration(milliseconds: 30));
      expect(LinkNavigation.isPolicyTrusted('www.tongji.edu.cn'), isTrue);

      // Simulate cold start while offline: persisted policy is restored.
      LinkNavigation.applyTrustedDomains(const []);
      final offline = ProviderContainer(
        overrides: [
          externalLinkPolicyRepositoryProvider.overrideWithValue(
            _StubExternalLinkPolicyRepo(null, fail: true),
          ),
        ],
      );
      addTearDown(offline.dispose);
      offline.read(externalLinkPolicyProvider);
      await Future<void>.delayed(const Duration(milliseconds: 30));
      expect(LinkNavigation.isPolicyTrusted('www.tongji.edu.cn'), isTrue);
    },
  );

  test(
    'userBlocksProvider retains list within session epoch and refetches after epoch bump',
    () async {
      final repo = _CountingUserRepository();
      final container = ProviderContainer(
        overrides: [
          userRepositoryProvider.overrideWithValue(repo),
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 7, username: 'alice'),
          ),
        ],
      );
      addTearDown(container.dispose);

      // Watch, close listener, then watch again: stays cached (1 call).
      final sub1 = container.listen(userBlocksProvider, (_, _) {});
      await container.read(userBlocksProvider.future);
      sub1.close();
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final sub2 = container.listen(userBlocksProvider, (_, _) {});
      await container.read(userBlocksProvider.future);
      expect(repo.blockCalls, 1);
      sub2.close();

      // Session epoch change forces a fresh read.
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await container.read(userBlocksProvider.future);
      expect(repo.blockCalls, 2);
    },
  );
}
