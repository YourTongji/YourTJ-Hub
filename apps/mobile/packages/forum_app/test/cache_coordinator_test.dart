import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/storage/cache_coordinator.dart';

class _JournalDatabase extends AppDatabase {
  _JournalDatabase() : super(NativeDatabase.memory());
  bool failWrite = false;
  bool failFinish = false;
  @override
  Future<void> setOperation(String name, String value) {
    if (failWrite) return Future.error(StateError('Journal write failure'));
    return super.setOperation(name, value);
  }

  @override
  Future<void> finishOperation(String name) {
    if (failFinish) return Future.error(StateError('Journal finish failure'));
    return super.finishOperation(name);
  }
}

void main() {
  test(
    'partial cleanup retains a retry journal and only failed owners stay suspended',
    () async {
      final db = AppDatabase(NativeDatabase.memory());
      addTearDown(db.close);
      var fails = true;
      var invalidations = 0, clears = 0;
      var holds = 0, resumes = 0, depth = 0;
      final coordinator = CacheCoordinator(
        database: db,
        databaseBytes: () async => 100,
        onInvalidated: (_) => invalidations++,
        owners: {
          CacheCategory.media: CacheOwner(
            invalidate: () {
              holds++;
              depth++;
            },
            resume: () {
              resumes++;
              depth--;
            },
            bytes: () async => 20,
            clear: () async {
              clears++;
              if (fails) throw StateError('Disk failure');
            },
          ),
        },
      );
      final first = await coordinator.clear({
        CacheCategory.forum,
        CacheCategory.media,
      });
      expect(first.failed, {CacheCategory.media});
      expect(await coordinator.pending(), {CacheCategory.media});
      expect(db.available(CacheCategory.forum), isTrue);
      expect(db.available(CacheCategory.media), isFalse);
      expect(depth, 1);
      expect(resumes, 0);
      expect((await coordinator.resume()).failed, {CacheCategory.media});
      expect(depth, 1);
      expect(holds, 1);
      fails = false;
      expect((await coordinator.resume()).succeeded, isTrue);
      expect(await coordinator.pending(), isEmpty);
      expect(db.available(CacheCategory.media), isTrue);
      expect(invalidations, 3);
      expect(clears, 3);
      expect(holds, 1);
      expect(resumes, 1);
      expect(depth, 0);
    },
  );

  test(
    'selected cleanup cannot delete other categories or user-work records',
    () async {
      final db = AppDatabase(NativeDatabase.memory());
      addTearDown(db.close);
      await db.customStatement(
        "INSERT INTO cache_entries VALUES ('s','forum','a','{}',0,0,2,1)",
      );
      await db.customStatement(
        "INSERT INTO cache_entries VALUES ('s','chat','b','{}',0,0,2,1)",
      );
      await db.customStatement(
        "INSERT INTO campus_snapshots VALUES ('s',7,'binding',1,'{}','now')",
      );
      final coordinator = CacheCoordinator(
        database: db,
        databaseBytes: () async => 0,
        onInvalidated: (_) {},
        owners: {},
      );
      await coordinator.clear({CacheCategory.forum});
      final rows = await db
          .customSelect('SELECT domain FROM cache_entries')
          .get();
      expect(rows.map((r) => r.read<String>('domain')), ['chat']);
      expect(
        await db.customSelect('SELECT * FROM campus_snapshots').get(),
        hasLength(1),
      );
    },
  );
  test(
    'a failed initial journal write stays fenced and resume retries each hold once',
    () async {
      final db = _JournalDatabase();
      addTearDown(db.close);
      var holds = 0, releases = 0, clears = 0;
      final coordinator = CacheCoordinator(
        database: db,
        databaseBytes: () async => 0,
        onInvalidated: (_) {},
        owners: {
          CacheCategory.media: CacheOwner(
            invalidate: () => holds++,
            resume: () => releases++,
            bytes: () async => 0,
            clear: () async {
              clears++;
            },
          ),
        },
      );
      db.failWrite = true;
      await expectLater(
        coordinator.clear({CacheCategory.forum, CacheCategory.media}),
        throwsStateError,
      );
      expect(holds, 1);
      expect(releases, 0);
      expect(clears, 0);
      expect(db.available(CacheCategory.forum), isFalse);
      expect(db.available(CacheCategory.media), isFalse);
      expect(await coordinator.pending(), isEmpty);
      db.failWrite = false;
      expect((await coordinator.resume()).succeeded, isTrue);
      expect(holds, 1);
      expect(releases, 1);
      expect(clears, 1);
      expect(db.available(CacheCategory.forum), isTrue);
      expect(db.available(CacheCategory.media), isTrue);
    },
  );

  test(
    'pending categories recovered from the journal fence their owners before deletion',
    () async {
      final db = AppDatabase(NativeDatabase.memory());
      addTearDown(db.close);
      await db.setOperation('clear', '["media"]');
      var depth = 0, clears = 0;
      final notifications = <Set<CacheCategory>>[];
      final coordinator = CacheCoordinator(
        database: db,
        databaseBytes: () async => 0,
        onInvalidated: notifications.add,
        owners: {
          CacheCategory.media: CacheOwner(
            invalidate: () => depth++,
            resume: () => depth--,
            bytes: () async => 0,
            clear: () async {
              expect(depth, 1);
              expect(db.available(CacheCategory.media), isFalse);
              clears++;
            },
          ),
        },
      );
      expect(
        (await coordinator.clear({CacheCategory.forum})).succeeded,
        isTrue,
      );
      expect(clears, 1);
      expect(depth, 0);
      expect(
        notifications.expand((c) => c),
        containsAll(
          CacheCategory.values.where(
            (c) => c == CacheCategory.forum || c == CacheCategory.media,
          ),
        ),
      );
      expect(await coordinator.pending(), isEmpty);
    },
  );

  test(
    'a failed final journal commit does not resume an owner before durable success',
    () async {
      final db = _JournalDatabase();
      addTearDown(db.close);
      var holds = 0, releases = 0;
      final coordinator = CacheCoordinator(
        database: db,
        databaseBytes: () async => 0,
        onInvalidated: (_) {},
        owners: {
          CacheCategory.media: CacheOwner(
            invalidate: () => holds++,
            resume: () => releases++,
            bytes: () async => 0,
            clear: () async {},
          ),
        },
      );
      db.failFinish = true;
      await expectLater(
        coordinator.clear({CacheCategory.media}),
        throwsStateError,
      );
      expect(await coordinator.pending(), {CacheCategory.media});
      expect(holds, 1);
      expect(releases, 0);
      db.failFinish = false;
      expect((await coordinator.resume()).succeeded, isTrue);
      expect(holds, 1);
      expect(releases, 1);
    },
  );
}
