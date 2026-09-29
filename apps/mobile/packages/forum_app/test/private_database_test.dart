import 'dart:io';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/storage/private_database.dart';
import 'package:sqlite3/sqlite3.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.root);
  final String root;
  @override
  Future<String?> getApplicationSupportPath() async => root;
  @override
  Future<String?> getApplicationDocumentsPath() async => root;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform oldPaths;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp('yourtj-encryption-');
    oldPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    FlutterSecureStorage.setMockInitialValues({});
  });
  tearDown(() async {
    PathProviderPlatform.instance = oldPaths;
    await directory.delete(recursive: true);
  });
  test(
    'production opener commits encrypted files and preserves work when key is missing',
    () async {
      final db = AppDatabase(openPrivateDatabase(name: 'user_work'));
      await db.setOperation('fixture', 'private-unpublished-text');
      await db.close();
      final file = File('${directory.path}/yourtj_private/user_work.sqlite');
      final bytes = await file.readAsBytes();
      expect(latin1.decode(bytes), isNot(contains('private-unpublished-text')));
      FlutterSecureStorage.setMockInitialValues({});
      final unavailable = AppDatabase(openPrivateDatabase(name: 'user_work'));
      await expectLater(unavailable.operation('fixture'), throwsStateError);
      // LazyDatabase.close also awaits the failed opener; preserve that failure
      // rather than treating it as a second successful open.
      await expectLater(unavailable.close(), throwsStateError);
      expect(await file.readAsBytes(), bytes);
    },
  );
  test(
    'legacy plaintext migration verifies encrypted campus copy then removes plaintext',
    () async {
      final legacy = File('${directory.path}/yourtj_cache.sqlite');
      final raw = sqlite3.open(legacy.path);
      raw.execute('PRAGMA user_version=1');
      raw.execute(
        'CREATE TABLE campus_snapshots (site TEXT,account_id INTEGER,binding_revision TEXT,schema_version INTEGER,payload TEXT,committed_at TEXT,PRIMARY KEY(site,account_id))',
      );
      raw.execute(
        "INSERT INTO campus_snapshots VALUES ('https://dev.example',7,'binding',1,'campus-fixture','now')",
      );
      raw.close();
      final db = AppDatabase(
        openPrivateDatabase(
          name: 'cache',
          disposable: true,
          legacyName: 'yourtj_cache',
        ),
      );
      final rows = await db
          .customSelect('SELECT payload FROM campus_snapshots')
          .get();
      expect(rows.single.read<String>('payload'), 'campus-fixture');
      await db.close();
      expect(await legacy.exists(), isFalse);
      final encrypted = File('${directory.path}/yourtj_private/cache.sqlite');
      expect(
        latin1.decode(await encrypted.readAsBytes()),
        isNot(contains('campus-fixture')),
      );
      // Simulate interruption between rename and legacy cleanup: a valid installed
      // encrypted copy wins, and the obsolete plaintext is cleaned on reopen.
      final leftover = sqlite3.open(legacy.path);
      leftover.execute('CREATE TABLE old (value TEXT)');
      leftover.close();
      final reopened = AppDatabase(
        openPrivateDatabase(
          name: 'cache',
          disposable: true,
          legacyName: 'yourtj_cache',
        ),
      );
      expect(
        await reopened
            .customSelect('SELECT payload FROM campus_snapshots')
            .get(),
        hasLength(1),
      );
      await reopened.close();
      expect(await legacy.exists(), isFalse);
    },
  );
}
