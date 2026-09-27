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
}
