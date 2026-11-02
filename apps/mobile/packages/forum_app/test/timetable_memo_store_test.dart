import 'dart:async';
import 'dart:convert';

import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/campus/timetable/timetable_memo_store.dart';
import 'package:forum_app/src/storage/user_work_database.dart';

class ControlledMemoDatabase extends UserWorkDatabase {
  ControlledMemoDatabase(this.backing) : super(NativeDatabase.memory());

  final UserWorkDatabase backing;
  final readStarted = Completer<void>();
  Completer<void>? readGate;
  Completer<void>? writeGate;
  bool failWrites = false;

  @override
  int get generation => backing.generation;

  @override
  Future<Map<String, String>> readDomain(String scope, String domain) async {
    final result = await backing.readDomain(scope, domain);
    if (!readStarted.isCompleted) readStarted.complete();
    await readGate?.future;
    return result;
  }

  @override
  Future<void> writeBatch(
    String scope,
    String domain,
    Map<String, String?> entries, {
    bool Function()? isCurrent,
    int? generation,
  }) async {
    await writeGate?.future;
    if (failWrites) throw StateError('disk full');
    await backing.writeBatch(
      scope,
      domain,
      entries,
      isCurrent: isCurrent,
      generation: generation,
    );
  }
}

const newBlock = TimetableBlock(
  id: 'new',
  title: 'New block',
  days: [1],
  start: 1,
  end: 2,
);

void main() {
  test(
    'corrupt memo entries drop without blocking the rest of the domain',
    () async {
      final database = UserWorkDatabase.instance;
      const scope = 'corrupt-memo';
      await database.writeBatch(scope, kTimetableMemoDomain, {
        'notes': '{broken',
        'blocks': jsonEncode([
          newBlock.toJson()..['id'] = 'kept',
          {'id': 42},
        ]),
      });
      final store = TimetableMemoStore(database: database, scope: scope);
      addTearDown(store.dispose);
      await pumpEventQueue();
      expect(store.state.loaded, isTrue);
      expect(store.state.error, isNull);
      // The bad entry and the unparseable key drop; the good block survives.
      expect(store.state.notes, isEmpty);
      expect(store.state.blocks.map((b) => b.id), ['kept']);

      // The domain stays editable and the next write rewrites the cleaned key.
      expect(await store.setNote('course', null, 'replacement'), isTrue);
      // Editing one key rewrites only that key; the other keeps its raw
      // bytes until it is edited, and the loader keeps dropping bad entries.
      final afterNote = await database.readDomain(scope, kTimetableMemoDomain);
      expect((jsonDecode(afterNote['notes']!) as List).single, isA<Map>());
      expect((jsonDecode(afterNote['blocks']!) as List).length, 2);
      expect(await store.deleteBlock('missing'), isTrue);
      final afterBlock = await database.readDomain(scope, kTimetableMemoDomain);
      expect((jsonDecode(afterBlock['blocks']!) as List).single['id'], 'kept');
    },
  );

  test('an edit during initial load preserves the stored blocks', () async {
    final database = ControlledMemoDatabase(UserWorkDatabase.instance)
      ..readGate = Completer<void>();
    addTearDown(database.close);
    await database.backing.writeBatch('a', kTimetableMemoDomain, {
      'blocks': jsonEncode([
        newBlock.copyWith(title: 'Old block').toJson()..['id'] = 'old',
      ]),
    });
    final store = TimetableMemoStore(database: database, scope: 'a');
    addTearDown(store.dispose);
    await database.readStarted.future;
    final save = store.saveBlock(newBlock);
    database.readGate!.complete();
    await save;
    expect(store.state.blocks.map((b) => b.id), ['old', 'new']);
    final disk = await database.backing.readDomain('a', kTimetableMemoDomain);
    expect((jsonDecode(disk['blocks']!) as List).length, 2);
  });

  test('failed writes retain committed state and can be retried', () async {
    final database = ControlledMemoDatabase(UserWorkDatabase.instance);
    addTearDown(database.close);
    final store = TimetableMemoStore(database: database, scope: 'a');
    addTearDown(store.dispose);
    await database.readStarted.future;
    await Future<void>.delayed(Duration.zero);
    database.failWrites = true;
    await store.saveBlock(newBlock);
    expect(store.state.blocks, isEmpty);
    expect(store.state.error, isNotNull);
    database.failWrites = false;
    await store.saveBlock(newBlock);
    expect(store.state.blocks.single.id, 'new');
    expect(store.state.error, isNull);
  });

  test(
    'a queued write cannot commit after its account store is disposed',
    () async {
      final database = ControlledMemoDatabase(UserWorkDatabase.instance)
        ..writeGate = Completer<void>();
      addTearDown(database.close);
      final store = TimetableMemoStore(database: database, scope: 'a');
      await database.readStarted.future;
      await Future<void>.delayed(Duration.zero);
      final save = store.saveBlock(newBlock);
      await Future<void>.delayed(Duration.zero);
      store.dispose();
      database.writeGate!.complete();
      await save;
      expect(
        await database.backing.readDomain('a', kTimetableMemoDomain),
        isEmpty,
      );
    },
  );
}
