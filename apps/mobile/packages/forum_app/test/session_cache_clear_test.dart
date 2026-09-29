import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/storage/cache_coordinator.dart';

class _CountingDatabase extends AppDatabase {
  _CountingDatabase() : super(NativeDatabase.memory());
  int sweeps = 0;
  int compactions = 0;

  @override
  Future<void> clearCategories(Set<CacheCategory> categories) {
    sweeps++;
    return super.clearCategories(categories);
  }

  @override
  Future<void> compact() {
    compactions++;
    return super.compact();
  }
}

class _WidgetBridge extends ScheduleWidgetBridge {
  final states = <String>[];

  @override
  Future<void> clear({String state = 'needsData'}) async => states.add(state);
}

class _IndependentCache implements OfflineTopicCache, OfflineChatCache {
  int clears = 0;

  @override
  Future<void> clear() async => clears++;
  @override
  Future<void> close() async {}
  @override
  Future<PagePayload?> get(int topicId) async => null;
  @override
  Future<List<ChatItemPayload>> getConversations() async => [];
  @override
  Future<List<ChatMessagePayload>> getMessages(int convId) async => [];
  @override
  Future<void> put(int topicId, Map<String, dynamic> payload) async {}
  @override
  Future<void> putConversations(List<ChatItemPayload> conversations) async {}
  @override
  Future<void> putMessages(
    int convId,
    List<ChatMessagePayload> messages,
  ) async {}
}

void main() {
  test('session cleanup sweeps a shared coordinated store only once', () async {
    final db = _CountingDatabase();
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
    final widget = _WidgetBridge();
    var coordinatorRuns = 0, mediaClears = 0, campusClears = 0;
    final coordinator = CacheCoordinator(
      database: db,
      databaseBytes: () async => 0,
      onInvalidated: (_) {},
      owners: {
        CacheCategory.media: CacheOwner(
          invalidate: () {},
          clear: () async => mediaClears++,
          bytes: () async => 0,
        ),
        CacheCategory.campus: CacheOwner(
          invalidate: widget.invalidate,
          clear: () async {
            campusClears++;
            await widget.clear();
          },
          bytes: () async => 0,
        ),
      },
    );
    db.clearAllCaches = () async {
      coordinatorRuns++;
      final result = await coordinator.clear(CacheCategory.values.toSet());
      if (!result.succeeded) throw StateError('Session cleanup incomplete');
    };

    // Topic/chat providers create separate views of the same database.
    await clearOfflineCache(
      DriftOfflineCache(db),
      DriftOfflineCache(db),
      widget,
    );

    expect(coordinatorRuns, 1);
    expect(db.sweeps, 1);
    expect(db.compactions, 1);
    expect(mediaClears, 1);
    expect(campusClears, 1);
    expect(await db.customSelect('SELECT * FROM cache_entries').get(), isEmpty);
    expect(
      await db.customSelect('SELECT * FROM campus_snapshots').get(),
      isEmpty,
    );
    expect(await coordinator.pending(), isEmpty);
    // The session publishes its terminal state after category cleanup.
    expect(widget.states.last, 'signedOut');
  });

  test('separate database owners are both cleared', () async {
    final topicDb = _CountingDatabase(), chatDb = _CountingDatabase();
    addTearDown(topicDb.close);
    addTearDown(chatDb.close);
    final widget = _WidgetBridge();

    await clearOfflineCache(
      DriftOfflineCache(topicDb),
      DriftOfflineCache(chatDb),
      widget,
    );

    expect(topicDb.sweeps, 1);
    expect(chatDb.sweeps, 1);
    expect(widget.states, ['signedOut']);
  });

  test('independent cache implementations are both cleared', () async {
    final topics = _IndependentCache(), chats = _IndependentCache();
    final widget = _WidgetBridge();

    await clearOfflineCache(topics, chats, widget);

    expect(topics.clears, 1);
    expect(chats.clears, 1);
    expect(widget.states, ['signedOut']);
  });

  test(
    'one cache object implementing both interfaces is only cleared once',
    () async {
      final cache = _IndependentCache();
      await clearOfflineCache(cache, cache, _WidgetBridge());
      expect(cache.clears, 1);
    },
  );

  test('a shared cleanup failure still blocks the session boundary', () async {
    final db = _CountingDatabase();
    addTearDown(db.close);
    var attempts = 0;
    db.clearAllCaches = () async {
      attempts++;
      throw StateError('Cleanup failed');
    };
    final widget = _WidgetBridge();

    await expectLater(
      clearOfflineCache(DriftOfflineCache(db), DriftOfflineCache(db), widget),
      throwsStateError,
    );

    expect(attempts, 1);
    expect(widget.states, isEmpty);
  });
}
