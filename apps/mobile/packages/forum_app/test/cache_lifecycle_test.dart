import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:sqlite3/sqlite3.dart';
import 'package:forum_app/src/storage/private_database.dart';
import 'fixtures/page_fixtures.dart';
import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/offline/drift_cache.dart';

ChatItemPayload conversation(int id) => ChatItemPayload(
  id: id,
  peerId: id,
  peerUsername: 'fixture',
  peerAvatar: '',
  lastMsg: 'fixture',
  lastMsgTime: '',
  unreadCount: 0,
  convId: id,
  peerUrl: '',
);

ChatMessagePayload message(int id) => ChatMessagePayload(
  id: id,
  senderId: 1,
  content: 'fixture message',
  msgType: 1,
  isRead: 0,
  createdAt: '',
  isSelf: false,
);

void main() {
  group('storage lifecycle', lifecycleCases);
  test('replacing conversation membership removes orphan messages', () async {
    final db = AppDatabase(NativeDatabase.memory());
    addTearDown(db.close);
    final cache = DriftOfflineCache(db);
    await cache.putConversations([conversation(1)]);
    await cache.putMessages(1, [message(1)]);
    await cache.putConversations([
      for (var id = 2; id <= 51; id++) conversation(id),
    ]);
    expect(await cache.getConversations(), hasLength(50));
    expect(await cache.getMessages(1), isEmpty);
  });

  test(
    'successful complete empty snapshot removes old conversations',
    () async {
      final db = AppDatabase(NativeDatabase.memory());
      addTearDown(db.close);
      final cache = DriftOfflineCache(db);
      await cache.putConversations([conversation(1)]);
      await cache.putConversations([]);
      expect(await cache.getConversations(), isEmpty);
    },
  );
}

void lifecycleCases() {
  late AppDatabase db;
  late DateTime clock;
  DriftOfflineCache scoped({
    String site = 'https://dev.example',
    int user = 7,
    String language = 'zh',
    int budget = 32768,
  }) => DriftOfflineCache(
    db,
    resolveScope: () async => CacheScope(site, user, language: language),
    now: () => clock,
    chatBudget: budget,
    forumBudget: budget,
  );
  setUp(() {
    db = AppDatabase(NativeDatabase.memory());
    clock = DateTime.utc(2026, 9, 29);
  });
  tearDown(() => db.close());

  test('all private snapshots isolate origin, account and language', () async {
    await scoped().putConversations([conversation(1)]);
    await scoped().putMessages(1, [message(1)]);
    for (final foreign in [
      scoped(site: 'https://prod.example'),
      scoped(user: 8),
      scoped(language: 'en'),
    ]) {
      expect(await foreign.getConversations(), isEmpty);
      expect(await foreign.getMessages(1), isEmpty);
    }
    expect(await scoped().getMessages(1), hasLength(1));
  });

  test(
    'boot purge keeps guest rows and removes every private or unknown scope',
    () async {
      // The purge SQL reads accountId from the JSON scope array; pin the shape.
      expect(
        CacheScope('https://dev.example', 7).key,
        jsonEncode(['https://dev.example', 7, 'zh']),
      );
      expect(
        CacheScope('https://dev.example', 7, language: 'en').key,
        jsonEncode(['https://dev.example', 7, 'en']),
      );
      final guest = scoped(user: 0);
      final topic = topicDetailPayloadJson();
      (topic['props'] as Map)['topic']['topicStatus'] = 1;
      await guest.putConversations([conversation(1)]);
      await guest.putMessages(1, [message(1)]);
      await guest.put(100, topic);
      await scoped().putConversations([conversation(2)]);
      await scoped(
        site: 'https://prod.example',
      ).putConversations([conversation(3)]);
      await db.customStatement(
        "INSERT INTO cache_entries VALUES ('not-json','forum','topic:9','{}',0,0,2,1)",
      );
      await db.customStatement(
        "INSERT INTO campus_snapshots VALUES ('https://dev.example',7,'binding',1,'campus','now')",
      );

      await db.purgePrivateScopes();

      expect(await scoped(user: 0).getConversations(), hasLength(1));
      expect(await scoped(user: 0).getMessages(1), hasLength(1));
      expect(await scoped(user: 0).get(100), isNotNull);
      expect(await scoped().getConversations(), isEmpty);
      expect(
        await scoped(site: 'https://prod.example').getConversations(),
        isEmpty,
      );
      expect(
        await db
            .customSelect('SELECT COUNT(*) AS n FROM cache_entries')
            .getSingle()
            .then((row) => row.read<int>('n')),
        3,
      );
      expect(
        await db
            .customSelect('SELECT COUNT(*) AS n FROM campus_snapshots')
            .getSingle()
            .then((row) => row.read<int>('n')),
        0,
      );
    },
  );

  test(
    'clear fences requests started earlier while new requests may write',
    () async {
      final request = scoped().capture();
      await db.clearCategories({CacheCategory.chat});
      await request.putMessages(1, [message(1)]);
      expect(await scoped().getMessages(1), isEmpty);
      await scoped().capture().putMessages(1, [message(2)]);
      expect((await scoped().getMessages(1)).single.id, 2);
    },
  );

  test('queued write loses to synchronous clear fence', () async {
    final barrier = Completer<void>();
    final occupied = db.serial(() => barrier.future);
    final request = scoped().capture();
    final write = request.putMessages(1, [message(1)]);
    final clear = db.clearCategories({CacheCategory.chat});
    barrier.complete();
    await Future.wait([occupied, write, clear]);
    expect(await scoped().getMessages(1), isEmpty);
  });

  test('reads do not renew retention and future clocks fail closed', () async {
    await scoped().putMessages(1, [message(1)]);
    clock = clock.add(const Duration(days: 29));
    expect(await scoped().getMessages(1), hasLength(1));
    clock = clock.add(const Duration(days: 2));
    expect(await scoped().getMessages(1), isEmpty);
    await scoped().putMessages(2, [message(2)]);
    clock = clock.subtract(const Duration(hours: 1));
    expect(await scoped().getMessages(2), isEmpty);
  });

  test(
    'valid empty snapshot differs from missing cache and empty delta',
    () async {
      final cache = scoped();
      expect(await cache.getConversations(), isEmpty);
      expect(cache.snapshotFound, isFalse);
      await cache.putConversations([]);
      expect(await cache.getConversations(), isEmpty);
      expect(cache.snapshotFound, isTrue);
      await cache.putMessages(1, [message(1)]);
      await cache.putMessages(1, []);
      expect(await cache.getMessages(1), hasLength(1));
    },
  );

  test(
    'byte budget covers all account scopes and oversized entry is removed',
    () async {
      await scoped(budget: 250).putMessages(1, [message(1)]);
      await scoped(user: 8, budget: 250).putMessages(2, [message(2)]);
      expect(
        (await db.cacheUsage())[CacheCategory.chat],
        lessThanOrEqualTo(250),
      );
      await scoped(user: 8, budget: 1).putMessages(2, [message(3)]);
      expect(await scoped(user: 8).getMessages(2), isEmpty);
    },
  );

  test(
    'page projection strips identity permissions and refuses drafts',
    () async {
      final cache = scoped();
      final payload = topicDetailPayloadJson();
      (payload['props'] as Map)['topic']['topicStatus'] = 1;
      (payload['layout'] as Map)['viewer']['email'] = 'private@example.test';
      (payload['props'] as Map)['seenProofs'] = [
        {
          'token': 'private-seen-proof',
          'topicIds': [100],
          'issuedAt': 1,
          'expiresAt': 2,
        },
      ];
      (payload['props'] as Map)['snapshotId'] = 'private-feed-session';
      await cache.put(100, payload);
      final stored = await db
          .customSelect(
            "SELECT payload FROM cache_entries WHERE domain = 'forum'",
          )
          .getSingle();
      expect(
        stored.read<String>('payload'),
        isNot(contains('private@example.test')),
      );
      expect(stored.read<String>('payload'), isNot(contains('canAccessAdmin')));
      expect(
        stored.read<String>('payload'),
        isNot(contains('private-seen-proof')),
      );
      expect(
        stored.read<String>('payload'),
        isNot(contains('private-feed-session')),
      );
      final restored = await cache.get(100);
      expect(restored, isNotNull);
      expect(restored!.layout.viewer.isAuthenticated, isFalse);
      expect(restored.props['permissions']['canPost'], isFalse);
      (payload['props'] as Map)['topic']['topicStatus'] = 0;
      await cache.put(100, payload);
      expect(await cache.get(100), isNull);
    },
  );

  test('unknown document schema and broken JSON discard only that row', () async {
    final cache = scoped();
    await cache.putMessages(1, [message(1)]);
    await cache.putMessages(2, [message(2)]);
    await db.customStatement(
      "UPDATE cache_entries SET document_version=99 WHERE entry_key='messages:1'",
    );
    expect(await cache.getMessages(1), isEmpty);
    expect(await cache.getMessages(2), hasLength(1));
    await db.customStatement(
      "UPDATE cache_entries SET payload='broken' WHERE entry_key='messages:2'",
    );
    expect(await cache.getMessages(2), isEmpty);
  });

  test(
    'cleanup journal survives database reopen and v1 migration preserves campus',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'cache-migration-',
      );
      addTearDown(() => directory.delete(recursive: true));
      final file = File('${directory.path}/legacy.sqlite');
      final raw = sqlite3.open(file.path);
      raw.execute('PRAGMA user_version=1');
      raw.execute(
        'CREATE TABLE cached_topics (id INTEGER PRIMARY KEY, payload TEXT, cached_at TEXT)',
      );
      raw.execute(
        "INSERT INTO cached_topics VALUES (1,'private old page','now')",
      );
      raw.execute(
        'CREATE TABLE campus_snapshots (site TEXT,account_id INTEGER,binding_revision TEXT,schema_version INTEGER,payload TEXT,committed_at TEXT,PRIMARY KEY(site,account_id))',
      );
      raw.execute(
        "INSERT INTO campus_snapshots VALUES ('https://dev.example',7,'binding',1,'campus','now')",
      );
      raw.close();
      final migrated = AppDatabase(NativeDatabase(file));
      expect(
        await migrated.customSelect('SELECT * FROM campus_snapshots').get(),
        hasLength(1),
      );
      expect(
        await migrated
            .customSelect(
              "SELECT name FROM sqlite_master WHERE name='cached_topics'",
            )
            .get(),
        isEmpty,
      );
      await migrated.setOperation('clear', '["chat"]');
      await migrated.close();
      final reopened = AppDatabase(NativeDatabase(file));
      expect(await reopened.operation('clear'), '["chat"]');
      await reopened.close();
    },
  );

  test(
    'native cipher refuses plaintext and encrypted files require key',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'encrypted-cache-',
      );
      addTearDown(() => directory.delete(recursive: true));
      final file = File('${directory.path}/test.sqlite');
      final raw = sqlite3.open(file.path);
      requireCipher(raw);
      raw.execute("PRAGMA key = 'fixture-encryption-key'");
      raw.execute('CREATE TABLE secret (value TEXT)');
      raw.execute("INSERT INTO secret VALUES ('unpublished private work')");
      raw.close();
      expect(
        latin1.decode(await file.readAsBytes()),
        isNot(contains('unpublished private work')),
      );
      final wrong = sqlite3.open(file.path);
      expect(
        () => wrong.select('SELECT * FROM secret'),
        throwsA(isA<SqliteException>()),
      );
      wrong.close();
      final right = sqlite3.open(file.path);
      right.execute("PRAGMA key = 'fixture-encryption-key'");
      expect(
        right.select('SELECT * FROM secret').single['value'],
        'unpublished private work',
      );
      right.close();
    },
  );
}
