import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'token';
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  test(
    'StickerLibrary restores disk SWR snapshot and refreshes when stale',
    () async {
      var listCalls = 0;
      var now = DateTime.utc(2026, 11, 26, 12, 0);
      List<StickerItemPayload>? persisted;
      final dio = Dio()
        ..httpClientAdapter = MockAdapter((request) async {
          listCalls++;
          return ResponseData(200, {
            'code': 0,
            'result': [
              {
                'id': 2,
                'name': 'fresh',
                'url': '/file/img/stickers/fresh.webp',
                'displayName': 'Fresh',
                'isOfficial': true,
                'isEnabled': true,
                'reviewStatus': 'approved',
              },
            ],
          });
        });

      final cachedItem = const StickerItemPayload(
        id: 1,
        name: 'cached',
        url: '/file/img/stickers/cached.webp',
        displayName: 'Cached',
      );
      // Case 1: Fresh (<5 min) persisted snapshot returns without network call.
      final freshLib = StickerLibrary(
        StickerRepository(GfApiClient(dio: dio, tokenStorage: _Tokens())),
        readCached: () async => (
          items: [cachedItem],
          savedAt: now.subtract(const Duration(minutes: 2)),
        ),
        writeCached: (items, _) async => persisted = items,
        now: () => now,
      );
      final restored = await freshLib.load();
      expect(restored.single.name, 'cached');
      expect(listCalls, 0);
      freshLib.dispose();

      // Case 2: Stale (10 min, <24h) returns cached immediately and refreshes in background.
      final staleLib = StickerLibrary(
        StickerRepository(GfApiClient(dio: dio, tokenStorage: _Tokens())),
        readCached: () async => (
          items: [cachedItem],
          savedAt: now.subtract(const Duration(minutes: 10)),
        ),
        writeCached: (items, _) async => persisted = items,
        now: () => now,
      );
      final immediate = await staleLib.load();
      expect(immediate.single.name, 'cached');
      await Future<void>.delayed(const Duration(milliseconds: 20));
      expect(listCalls, 1);
      expect(staleLib.urlByName.containsKey('fresh'), isTrue);
      expect(persisted?.single.name, 'fresh');
      staleLib.dispose();
    },
  );

  test(
    'WikiRepository caches tree/namespaces/home when cacheTtl > 0',
    () async {
      var calls = 0;
      var now = DateTime.utc(2026, 11, 26, 12, 0);
      final dio = Dio()
        ..httpClientAdapter = MockAdapter((request) async {
          calls++;
          if (request.path.endsWith('/home')) {
            return ResponseData(200, {
              'code': 0,
              'result': {'namespaces': <Object>[], 'recentUpdates': <Object>[]},
            });
          }
          return ResponseData(200, {
            'code': 0,
            'result': {'namespaces': <Object>[]},
          });
        });

      final repo = WikiRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
        cacheTtl: const Duration(minutes: 5),
        now: () => now,
      );

      await repo.tree();
      await repo.tree();
      expect(calls, 1);
      now = now.add(const Duration(minutes: 6));
      await repo.tree();
      expect(calls, 2);
      repo.clearCache();
      await repo.tree();
      expect(calls, 3);
    },
  );

  test(
    'CourseRepository caches detail/related and invalidates on bookmark',
    () async {
      var detailCalls = 0;
      var relatedCalls = 0;
      var now = DateTime.utc(2026, 11, 26, 12, 0);
      final dio = Dio()
        ..httpClientAdapter = MockAdapter((request) async {
          if (request.path == '/api/forum/courses/42') {
            detailCalls++;
            return ResponseData(200, {
              'code': 0,
              'result': {
                'id': 42,
                'primaryCode': 'CS101',
                'name': 'Intro CS',
                'department': 'CS',
                'creditX10': 30,
                'bookmarked': false,
              },
            });
          }
          if (request.path == '/api/forum/courses/42/related') {
            relatedCalls++;
            return ResponseData(200, {
              'code': 0,
              'result': {
                'teacherOtherCourses': <Object>[],
                'sameCourseOtherTeachers': <Object>[],
              },
            });
          }
          return ResponseData(200, {'code': 0, 'result': <String, Object>{}});
        });

      final repo = CourseRepository(
        GfApiClient(dio: dio, tokenStorage: _Tokens()),
        cacheTtl: const Duration(minutes: 2),
        now: () => now,
      );

      await repo.detail(42);
      await repo.detail(42);
      expect(detailCalls, 1);
      await repo.related(42);
      await repo.related(42);
      expect(relatedCalls, 1);

      // Bookmark mutation invalidates cached course detail.
      await repo.bookmark(courseId: 42, bookmarked: true);
      await repo.detail(42);
      expect(detailCalls, 2);

      now = now.add(const Duration(minutes: 3));
      await repo.detail(42);
      expect(detailCalls, 3);
    },
  );
}
