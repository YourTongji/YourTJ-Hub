import 'dart:io';

import 'package:drift/native.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/storage/private_database.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:sqlite3/sqlite3.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.path);
  final String path;
  @override
  Future<String?> getApplicationSupportPath() async => path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

Future<int> _bytes(File file) async {
  var total = 0;
  for (final suffix in ['', '-wal', '-shm']) {
    final candidate = File('${file.path}$suffix');
    if (await candidate.exists()) total += await candidate.length();
  }
  return total;
}

Future<int> _pragma(AppDatabase db, String name) async =>
    (await db.customSelect('PRAGMA $name').getSingle()).read<int>(name);

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform previousPaths;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp('cache-budget-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    FlutterSecureStorage.setMockInitialValues({});
  });
  tearDown(() async {
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  test('upgrade reclaims old allocation and enables incremental vacuum', () async {
    final file = File('${directory.path}/yourtj_cache.sqlite');
    final raw = sqlite3.open(file.path);
    raw.execute('PRAGMA user_version = 1');
    raw.execute(
      'CREATE TABLE cached_topics(id INTEGER PRIMARY KEY, payload BLOB)',
    );
    // The old count-only cache allowed individual payloads over the new limit.
    raw.execute('INSERT INTO cached_topics VALUES(1, zeroblob(68157440))');
    raw.execute(
      'CREATE TABLE campus_snapshots(site TEXT,account_id INTEGER,binding_revision TEXT,schema_version INTEGER,payload TEXT,committed_at TEXT,PRIMARY KEY(site,account_id))',
    );
    raw.execute(
      "INSERT INTO campus_snapshots VALUES('https://fixture.test',7,'binding',1,'keep-campus','now')",
    );
    expect(raw.select('PRAGMA auto_vacuum').single.values.single, 0);
    raw.close();
    final db = AppDatabase(
      openPrivateDatabase(
        name: 'cache',
        disposable: true,
        legacyName: 'yourtj_cache',
      ),
      physicalBytes: () => privateDatabaseBytes('cache'),
    );
    try {
      expect(await _pragma(db, 'auto_vacuum'), 2);
      expect(
        (await db
                .customSelect('SELECT payload FROM campus_snapshots')
                .getSingle())
            .read<String>('payload'),
        'keep-campus',
      );
      expect(await privateDatabaseBytes('cache'), lessThan(2 * 1024 * 1024));
    } finally {
      await db.close();
    }
  });

  test(
    'normal batch eviction reclaims free pages without deleting campus',
    () async {
      final file = File('${directory.path}/cache.sqlite');
      final db = AppDatabase(
        NativeDatabase(
          file,
          setup: (raw) {
            raw.execute('PRAGMA auto_vacuum = INCREMENTAL');
            raw.execute('PRAGMA journal_mode = WAL');
          },
        ),
        physicalBytes: () => _bytes(file),
      );
      try {
        await db.customStatement(
          "INSERT INTO campus_snapshots VALUES('https://fixture.test',7,'binding',1,'keep-campus','now')",
        );
        await db.customStatement(
          "INSERT INTO cache_entries VALUES('fixture','chat','old',zeroblob(6291456),0,0,6291456,1)",
        );
        expect(await _bytes(file), greaterThan(4 * 1024 * 1024));
        await DriftOfflineCache(db).putConversations([]);
        expect(await _bytes(file), lessThan(2 * 1024 * 1024));
        expect(
          (await db
                  .customSelect('SELECT payload FROM campus_snapshots')
                  .getSingle())
              .read<String>('payload'),
          'keep-campus',
        );
        expect(await _pragma(db, 'freelist_count'), 0);
      } finally {
        await db.close();
      }
    },
  );

  test(
    'small writes do not vacuum and a physical WAL watermark is reclaimed',
    () async {
      final file = File('${directory.path}/cache.sqlite');
      final db = AppDatabase(
        NativeDatabase(
          file,
          setup: (raw) {
            raw.execute('PRAGMA auto_vacuum = INCREMENTAL');
            raw.execute('PRAGMA journal_mode = WAL');
            raw.execute('PRAGMA wal_autocheckpoint = 0');
          },
        ),
        physicalBytes: () => _bytes(file),
      );
      try {
        await db.customStatement('CREATE TABLE fixture(value BLOB)');
        await db.customStatement(
          'INSERT INTO fixture VALUES(zeroblob(1048576))',
        );
        await db.customStatement('DELETE FROM fixture');
        final freePages = await _pragma(db, 'freelist_count');
        expect(freePages, greaterThan(0));
        await DriftOfflineCache(db).putConversations([]);
        expect(await _pragma(db, 'freelist_count'), greaterThan(0));
        // An uncheckpointed WAL may exceed the target with little live payload.
        for (var i = 0; i < 33; i++) {
          await db.customStatement(
            'INSERT INTO fixture VALUES(zeroblob(2097152))',
          );
          await db.customStatement('DELETE FROM fixture');
        }
        expect(
          await _bytes(file),
          greaterThan(AppDatabase.physicalBudgetBytes),
        );
        await db.maintainBudget();
        expect(await _bytes(file), lessThan(4 * 1024 * 1024));
      } finally {
        await db.close();
      }
    },
  );
}
