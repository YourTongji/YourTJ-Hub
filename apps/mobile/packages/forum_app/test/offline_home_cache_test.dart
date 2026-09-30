import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/drift_cache.dart';

import 'fixtures/page_fixtures.dart';

void main() {
  test(
    'home pages are scoped by account, API origin and sort, then cleared',
    () async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final cache = DriftOfflineCache(database);
      final payload = PagePayload.fromJson(homePayloadJson());

      await cache.putHomePage(
        accountId: 7,
        baseUrl: 'https://forum.example',
        sort: 'latest',
        payload: payload,
      );

      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNotNull,
      );
      expect(
        await cache.getHomePage(
          accountId: 8,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNull,
      );
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://other.example',
          sort: 'latest',
        ),
        isNull,
      );
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'hot',
        ),
        isNull,
      );

      await cache.clear();
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNull,
      );
    },
  );

  test(
    'with a resolver, home access requires the resolved owner identity',
    () async {
      final database = AppDatabase(NativeDatabase.memory());
      addTearDown(database.close);
      final cache = DriftOfflineCache(
        database,
        resolveScope:
            () async => CacheScope('https://forum.example', 7, language: 'zh'),
      );
      final payload = PagePayload.fromJson(homePayloadJson());

      // A caller presenting the resolved owner reads and writes normally.
      await cache.putHomePage(
        accountId: 7,
        baseUrl: 'https://forum.example',
        sort: 'latest',
        payload: payload,
      );
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNotNull,
      );

      // A mismatched caller is fenced: nothing is written and nothing is read.
      await cache.putHomePage(
        accountId: 9,
        baseUrl: 'https://forum.example',
        sort: 'latest',
        payload: payload,
      );
      expect(
        await cache.getHomePage(
          accountId: 9,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNull,
      );
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNotNull,
      );
      expect(
        await cache.getHomePage(
          accountId: 7,
          baseUrl: 'https://other.example',
          sort: 'latest',
        ),
        isNull,
      );

      // Another representation of the same account shares nothing.
      final english = DriftOfflineCache(
        database,
        resolveScope:
            () async => CacheScope('https://forum.example', 7, language: 'en'),
      );
      expect(
        await english.getHomePage(
          accountId: 7,
          baseUrl: 'https://forum.example',
          sort: 'latest',
        ),
        isNull,
      );
    },
  );
}
