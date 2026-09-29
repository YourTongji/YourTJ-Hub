import 'dart:convert';
import 'package:drift/native.dart';
import 'package:sqlite3/sqlite3.dart' as sqlite;
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const scope = 'https%3A%2F%2Fdev.example:7';
  LocalDraft draft(String content) => LocalDraft(
    key: 'one',
    title: '',
    content: content,
    contentType: 1,
    topicId: 0,
    categories: [],
    images: [],
    updatedAt: 1,
  );
  setUp(() => SharedPreferences.setMockInitialValues({}));
  test(
    'draft migration is authoritative and deletion survives stale legacy copies',
    () async {
      final prefs = await SharedPreferences.getInstance();
      final key = 'yourtj:writing:v1:$scope:draft:one';
      await prefs.setString(key, jsonEncode(draft('legacy').toJson()));
      final store = WritingStore();
      expect((await store.drafts(scope)).single.content, 'legacy');
      expect(prefs.containsKey(key), false);
      await store.delete(scope, 'one');
      await prefs.setString(key, jsonEncode(draft('stale').toJson()));
      expect(await WritingStore().drafts(scope), isEmpty);
    },
  );
  test('reset blocks a previously created writing session', () async {
    final store = WritingStore();
    await store.save(scope, draft('before'));
    await UserWorkDatabase.instance.clearAll();
    await expectLater(store.save(scope, draft('after')), throwsStateError);
    expect(await WritingStore().drafts(scope), isEmpty);
  });
  test('legacy plans stay in quarantine without a site', () async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('pk.plans', '[{"id":"legacy"}]');
    final db = UserWorkDatabase.instance;
    expect(await db.readDomain(scope, 'schedule'), isEmpty);
    expect(
      (await db.readDomain('legacy-unassigned', 'schedule-legacy'))['pk.plans'],
      '[{"id":"legacy"}]',
    );
    expect(prefs.containsKey('pk.plans'), false);
  });
  test('a failed transaction does not advance any part of a snapshot', () async {
    final db = UserWorkDatabase.instance;
    await db.writeBatch(scope, 'schedule', {'a': 'old', 'b': 'old'});
    await db.customStatement(
      "CREATE TRIGGER fail_write BEFORE UPDATE ON work_records WHEN NEW.entry_key='b' BEGIN SELECT RAISE(ABORT, 'disk full'); END",
    );
    await expectLater(
      db.writeBatch(scope, 'schedule', {'a': 'new', 'b': 'new'}),
      throwsA(anything),
    );
    expect(await db.readDomain(scope, 'schedule'), {'a': 'old', 'b': 'old'});
  });
  test(
    'migration and reset leave chat and unknown writing records untouched',
    () async {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString('yourtj:writing:v1:$scope:chat:9', 'private');
      await prefs.setString('yourtj:writing:v1:$scope:future', 'unknown');
      await UserWorkDatabase.instance.readDomain(scope, 'draft');
      await UserWorkDatabase.instance.clearAll();
      expect(prefs.getString('yourtj:writing:v1:$scope:chat:9'), 'private');
      expect(prefs.getString('yourtj:writing:v1:$scope:future'), 'unknown');
    },
  );
  test(
    'failed migration leaves the source and rolls back copied records',
    () async {
      final db = UserWorkDatabase.instance;
      final prefs = await SharedPreferences.getInstance();
      final key = 'yourtj:writing:v1:$scope:draft:one';
      await prefs.setString(key, jsonEncode(draft('legacy').toJson()));
      await db.customStatement(
        "CREATE TRIGGER fail_migration BEFORE INSERT ON work_records BEGIN SELECT RAISE(ABORT, 'disk full'); END",
      );
      await expectLater(db.readDomain(scope, 'draft'), throwsA(anything));
      expect(prefs.getString(key), isNotNull);
      expect(
        await db.customSelect('SELECT * FROM work_records').get(),
        isEmpty,
      );
      await db.customStatement('DROP TRIGGER fail_migration');
      expect((await WritingStore().drafts(scope)).single.content, 'legacy');
      expect(prefs.getString(key), isNull);
    },
  );
  test(
    'reset fences a queued save and ignores old preference reappearance',
    () async {
      final db = UserWorkDatabase.instance;
      final save = db.writeBatch(scope, 'draft', {'one': 'old'});
      final rejected = expectLater(save, throwsStateError);
      await db.clearAll();
      await rejected;
      expect(await db.readDomain(scope, 'draft'), isEmpty);
      expect((await db.usage()).draftCount, 0);
    },
  );
  test('history changes serialize across store instances', () async {
    await Future.wait([
      WritingStore().remember(scope, 'one'),
      WritingStore().remember(scope, 'two'),
    ]);
    expect(await WritingStore().history(scope), ['two', 'one']);
  });
  test(
    'legacy recovery requires its recorded account and never imports old CAS bases',
    () async {
      final prefs = await SharedPreferences.getInstance();
      final plan = {
        'id': 'old',
        'name': 'Legacy',
        'createdAt': 1,
        'stagedCourses': [],
        'selectedCourses': [],
        'customEvents': [],
      };
      await prefs.setString('pk.plans', jsonEncode([plan]));
      await prefs.setString('pk.syncOwner', '7');
      await prefs.setString(
        'pk.planSync.v3.7',
        jsonEncode({
          'bases': {
            'old': {'revision': 99, 'plan': plan},
          },
          'drafts': {
            'backup': {...plan, 'id': 'backup'},
          },
        }),
      );
      final db = UserWorkDatabase.instance;
      expect((await db.usage()).legacyPlanCount, 1);
      await expectLater(
        db.recoverLegacySchedule(scope, 8, generation: db.generation),
        throwsStateError,
      );
      expect(await db.readDomain(scope, 'schedule'), isEmpty);
      await db.recoverLegacySchedule(scope, 7, generation: db.generation);
      final values = await db.readDomain(scope, 'schedule');
      expect(jsonDecode(values['pk.plans']!)[0]['id'], 'old');
      expect(jsonDecode(values['pk.planSync.v3.7']!)['bases'], isEmpty);
      expect(
        jsonDecode(values['pk.planSync.v3.7']!)['drafts']['backup']['id'],
        'backup',
      );
      expect(
        await db.readDomain('legacy-unassigned', 'schedule-legacy'),
        isEmpty,
      );
      expect((await db.usage()).legacyPlanCount, 0);
    },
  );
  test('failed legacy recovery preserves source and target', () async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('pk.plans', '[{"id":"old"}]');
    final db = UserWorkDatabase.instance;
    await db.readDomain('legacy-unassigned', 'schedule-legacy');
    await db.customStatement(
      "CREATE TRIGGER fail_recovery BEFORE INSERT ON work_records WHEN NEW.domain='schedule' BEGIN SELECT RAISE(ABORT, 'disk full'); END",
    );
    await expectLater(
      db.recoverLegacySchedule(scope, 7, generation: db.generation),
      throwsA(anything),
    );
    expect(await db.readDomain(scope, 'schedule'), isEmpty);
    expect(
      (await db.readDomain('legacy-unassigned', 'schedule-legacy'))['pk.plans'],
      '[{"id":"old"}]',
    );
  });
  test(
    'schedule plans are isolated by origin and old stores cannot write after reset',
    () async {
      final first = ScheduleStoreNotifier(site: 'https://one.example');
      final second = ScheduleStoreNotifier(site: 'https://two.example');
      await Future.wait([first.ready, second.ready]);
      first.renamePlan(first.state.activePlanId, 'First site');
      await first.flush;
      expect(second.state.plans.single.name, isNot('First site'));
      final reloaded = ScheduleStoreNotifier(site: first.site);
      await reloaded.ready;
      expect(reloaded.state.plans.single.name, 'First site');
      await UserWorkDatabase.instance.clearAll();
      first.renamePlan(first.state.activePlanId, 'Stale write');
      await first.flush;
      expect(first.persistenceError.value, isStateError);
      expect((await UserWorkDatabase.instance.usage()).planCount, 0);
      first.dispose();
      second.dispose();
      reloaded.dispose();
    },
  );
  test('recovery never drops an authored empty plan', () async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('pk.plans', '[{"id":"legacy"}]');
    final db = UserWorkDatabase.instance;
    await db.writeBatch(scope, 'schedule', {
      'pk.plans':
          '[{"id":"authored","name":"My chosen name","stagedCourses":[],"selectedCourses":[],"customEvents":[]}]',
    });
    await db.recoverLegacySchedule(scope, 7, generation: db.generation);
    final plans =
        jsonDecode((await db.readDomain(scope, 'schedule'))['pk.plans']!)
            as List;
    expect(plans.map((p) => p['id']), ['authored', 'legacy']);
  });
  test('database reopen does not resurrect a deleted legacy copy', () async {
    await UserWorkDatabase.instance.close();
    final raw = sqlite.sqlite3.openInMemory();
    final first = UserWorkDatabase(
      NativeDatabase.opened(raw, closeUnderlyingOnClose: false),
    );
    final prefs = await SharedPreferences.getInstance();
    final key = 'yourtj:writing:v1:$scope:draft:one';
    await prefs.setString(key, jsonEncode(draft('legacy').toJson()));
    final store = WritingStore(database: first);
    expect(await store.drafts(scope), hasLength(1));
    await store.delete(scope, 'one');
    await first.close();
    await prefs.setString(key, jsonEncode(draft('stale legacy').toJson()));
    final second = UserWorkDatabase(
      NativeDatabase.opened(raw, closeUnderlyingOnClose: false),
    );
    try {
      expect(await WritingStore(database: second).drafts(scope), isEmpty);
      expect(prefs.containsKey(key), false);
    } finally {
      await second.close();
      raw.close();
    }
  });
}
