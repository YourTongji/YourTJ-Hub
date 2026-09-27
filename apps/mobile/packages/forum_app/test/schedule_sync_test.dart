import 'dart:async';
import 'dart:convert';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/schedule/schedule_sync.dart';

PkPlan plan([String id = 'p', String name = 'Plan']) => PkPlan(
  id: id,
  name: name,
  createdAt: 1,
  stagedCourses: [],
  selectedCourses: [],
  customEvents: [],
);
PkPlanItem item([PkPlan? p, int revision = 1]) => PkPlanItem(
  plan: p ?? plan(),
  revision: revision,
  updatedAt: '2026-09-21T00:00:00Z',
);
PkPlan edit(PkPlan p, Map<String, dynamic> values) =>
    PkPlan.fromJson({...p.toJson(), ...values});

class Tokens implements TokenStorage {
  String? token = 'token';
  int owner = 7;
  @override
  Future<String?> read() async => token;
  @override
  Future<void> write(String value) async {
    token = value;
  }

  @override
  Future<void> clear() async {
    token = null;
  }
}

class FakeTimer implements Timer {
  FakeTimer(this.callback);
  final void Function() callback;
  bool active = true;
  @override
  bool get isActive => active;
  @override
  int get tick => 0;
  @override
  void cancel() {
    active = false;
  }

  void fire() {
    if (active) {
      active = false;
      callback();
    }
  }
}

class Transport implements PkPlansTransport {
  List<PkPlanItem> remote = [item()];
  int reads = 0;
  Object? readError, writeError;
  Completer<List<PkPlanItem>>? pendingRead;
  Completer<PkPlanItem>? pendingWrite;
  final writes = <(PkPlan, int)>[];
  final deletes = <(String, int)>[];
  @override
  Future<List<PkPlanItem>> list() async {
    reads++;
    if (readError != null) throw readError!;
    if (pendingRead != null) return pendingRead!.future;
    return remote.map((i) => PkPlanItem.fromJson(i.toJson())).toList();
  }

  @override
  Future<PkPlanItem> put(PkPlan p, int revision) async {
    writes.add((p, revision));
    if (writeError != null) {
      final error = writeError!;
      writeError = null;
      throw error;
    }
    if (pendingWrite != null) return pendingWrite!.future;
    final next = item(p, revision + 1);
    remote = [...remote.where((i) => i.plan.id != p.id), next];
    return next;
  }

  @override
  Future<void> remove(String id, int revision) async {
    deletes.add((id, revision));
    remote = remote.where((i) => i.plan.id != id).toList();
  }
}

Future<void> settle() async {
  for (var i = 0; i < 20; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}

class ControlledStore extends ScheduleStoreNotifier {
  bool failCacheWrites = false;
  bool failLargeCacheWrites = false;
  Completer<bool>? nextCacheWrite;
  @override
  Future<bool> writePlanSyncCache(int owner, Map<String, dynamic> value) async {
    if (failCacheWrites) return false;
    if (failLargeCacheWrites && jsonEncode(value).contains('"plans":[{')) {
      return false;
    }
    final pending = nextCacheWrite;
    nextCacheWrite = null;
    if (pending != null) return pending.future;
    return super.writePlanSyncCache(owner, value);
  }
}

void main() {
  late ControlledStore store;
  late ScheduleSyncController sync;
  late Transport transport;
  late Tokens tokens;
  late List<FakeTimer> timers;
  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    store = ControlledStore();
    await store.ready;
    transport = Transport();
    tokens = Tokens();
    timers = [];
    sync = ScheduleSyncController(
      transport: transport,
      tokenStorage: tokens,
      store: store,
      readUserId: () async => tokens.owner,
      debounceTimer: (delay, fire) {
        final timer = FakeTimer(fire);
        timers.add(timer);
        return timer;
      },
    );
  });
  tearDown(() {
    sync.dispose();
    store.dispose();
  });
  test('fresh empty default adopts cloud without uploading', () async {
    await sync.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.map((p) => p.id), ['p']);
    expect(transport.writes, isEmpty);
    expect(sync.isDirty, isFalse);
  });
  test(
    'only modified plan is uploaded and base survives persistence',
    () async {
      transport.remote = [item(), item(plan('q'))];
      await sync.syncOnEnter();
      store.renamePlan('p', 'Changed');
      timers.last.fire();
      await settle();
      expect(transport.writes.length, 1);
      expect(transport.writes.first.$1.id, 'p');
      expect(transport.writes.first.$2, 1);
      await store.flush;
      expect((store.readPlanSyncCache(7)!['bases'] as Map)['p']['revision'], 2);
    },
  );
  test('device preferences do not schedule writes', () async {
    transport.remote = [item(), item(plan('q'))];
    await sync.syncOnEnter();
    store.switchPlan('q');
    store.setWeekView(PkWeekView(week: 4));
    store.setMajorSelection(PkMajorSelection(calendarId: 121));
    await settle();
    expect(transport.writes, isEmpty);
    expect(timers.where((t) => t.active), isEmpty);
  });
  test('clean remote deletion does not revive an old plan', () async {
    await sync.syncOnEnter();
    transport.remote = [];
    await sync.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.any((p) => p.id == 'p'), isFalse);
    expect(transport.writes, isEmpty);
    expect(sync.isDirty, isFalse);
  });
  test(
    'dirty remote deletion is one conflict with an independent recoverable draft',
    () async {
      await sync.syncOnEnter();
      store.renamePlan('p', 'Local');
      transport.remote = [];
      await sync.syncOnEnter();
      await sync.syncOnEnter();
      expect(sync.conflicts.value.length, 1);
      await sync.resolveConflict('p', {'[]': 'local'});
      expect(sync.drafts.value['p']?.name, 'Local');
      expect(sync.restoreDraft('p'), isTrue);
      timers.last.fire();
      await settle();
      expect(transport.writes.first.$1.id, isNot('p'));
      expect(transport.writes.first.$2, 0);
    },
  );
  test(
    'explicit conflict resolution preserves unrelated remote edits',
    () async {
      final base = edit(plan(), {
        'customEvents': [
          {
            'id': 'e',
            'label': 'Original',
            'day': 1,
            'sections': [1],
            'weeks': [1],
          },
        ],
      });
      transport.remote = [item(base)];
      await sync.syncOnEnter();
      final local = edit(base, {
        'customEvents': [
          {
            'id': 'e',
            'label': 'Local',
            'day': 1,
            'sections': [1],
            'weeks': [1],
          },
        ],
      });
      store.applyPlanItems([local]);
      scheduleLocalPlansChanged?.call();
      transport.remote = [
        item(
          edit(base, {
            'name': 'Remote name',
            'customEvents': [
              {
                'id': 'e',
                'label': 'Remote',
                'day': 1,
                'sections': [1],
                'weeks': [1],
              },
            ],
          }),
          2,
        ),
      ];
      await sync.syncOnEnter();
      expect(sync.conflicts.value.first.fields.map((f) => f.path), [
        ['events', 'e', 'label'],
      ]);
      await sync.resolveConflict('p', {'["events","e","label"]': 'local'});
      expect(transport.writes.first.$1.name, 'Remote name');
      expect(transport.writes.first.$1.customEvents.first.label, 'Local');
      expect(transport.writes.first.$2, 2);
    },
  );
  test('409 includes remote item and does not need another GET', () async {
    await sync.syncOnEnter();
    store.renamePlan('p', 'Local');
    final remote = edit(plan(), {
      'customEvents': [
        {
          'id': 'e',
          'label': 'Remote',
          'day': 1,
          'sections': [1],
          'weeks': [1],
        },
      ],
    });
    transport.writeError = ApiException(
      fallbackMessage: 'conflict',
      statusCode: 409,
      responseData: item(remote, 2).toJson(),
    );
    timers.last.fire();
    await settle();
    timers.last.fire();
    await settle();
    expect(transport.reads, 1);
    expect(transport.writes.last.$2, 2);
    expect(transport.writes.last.$1.customEvents.first.label, 'Remote');
    expect(transport.writes.last.$1.name, 'Local');
  });
  test('changes during upload remain dirty', () async {
    await sync.syncOnEnter();
    transport.pendingWrite = Completer();
    store.renamePlan('p', 'First');
    timers.last.fire();
    await settle();
    store.renamePlan('p', 'Second');
    transport.pendingWrite!.complete(item(plan('p', 'First'), 2));
    await settle();
    transport.pendingWrite = null;
    timers.last.fire();
    await settle();
    expect(transport.writes.last.$1.name, 'Second');
    expect(transport.writes.last.$2, 2);
  });
  test('only dirty failures retry and success stops timers', () async {
    await sync.syncOnEnter();
    transport.writeError = Exception('offline');
    store.renamePlan('p', 'Local');
    timers.last.fire();
    await settle();
    expect(timers.last.active, isTrue);
    timers.last.fire();
    await settle();
    expect(transport.writes.length, 2);
    expect(timers.where((t) => t.active), isEmpty);
  });
  test('initial read failures block uploads', () async {
    transport.readError = Exception('offline');
    await sync.syncOnEnter();
    store.renamePlan(store.buildSnapshotPayload().plans.first.id, 'Changed');
    timers.last.fire();
    await settle();
    expect(transport.writes, isEmpty);
    transport.readError = null;
    transport.remote = [];
    timers.last.fire();
    await settle();
    expect(transport.writes.length, 1);
  });
  test('quota and auth failures stop automatic retries', () async {
    await sync.syncOnEnter();
    transport.writeError = const ApiException(
      fallbackMessage: 'quota',
      statusCode: 409,
    );
    store.renamePlan('p', 'Local');
    timers.last.fire();
    await settle();
    expect(sync.blocked.value, isTrue);
    expect(timers.where((t) => t.active), isEmpty);
    expect(sync.isDirty, isTrue);
  });
  test('separate account caches preserve prior offline changes', () async {
    await sync.syncOnEnter();
    store.renamePlan('p', 'Offline');
    await store.flush;
    tokens.owner = 8;
    transport.remote = [item(plan('q', 'Other'))];
    await sync.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.first.id, 'q');
    tokens.owner = 7;
    transport.remote = [item()];
    await sync.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.first.name, 'Offline');
  });
  test('old account response cannot replace new account content', () async {
    await sync.syncOnEnter();
    transport.pendingRead = Completer();
    final pending = sync.syncOnEnter();
    await settle();
    tokens.owner = 8;
    transport.pendingRead!.complete([item(plan('private'))]);
    await pending;
    expect(
      store.buildSnapshotPayload().plans.any((p) => p.id == 'private'),
      isFalse,
    );
  });
  test(
    'switching accounts starts a read before an old read finishes',
    () async {
      await sync.syncOnEnter();
      final oldRead = Completer<List<PkPlanItem>>();
      transport.pendingRead = oldRead;
      final oldRun = sync.syncOnEnter();
      await settle();
      transport.pendingRead = null;
      tokens.owner = 8;
      transport.remote = [item(plan('q', 'Other'))];
      final newRun = sync.syncOnEnter();
      await settle();
      expect(transport.reads, 3);
      expect(store.buildSnapshotPayload().plans.single.id, 'q');
      oldRead.complete([item(plan('private'))]);
      await Future.wait([oldRun, newRun]);
      expect(store.buildSnapshotPayload().plans.single.id, 'q');
    },
  );
  test(
    'failed draft persistence leaves the visible plan and conflict intact',
    () async {
      await sync.syncOnEnter();
      store.renamePlan('p', 'Local');
      transport.remote = [];
      await sync.syncOnEnter();
      store.failCacheWrites = true;
      await sync.resolveConflict('p', {'[]': 'local'});
      expect(store.buildSnapshotPayload().plans.single.name, 'Local');
      expect(sync.conflicts.value.single.id, 'p');
      expect(transport.writes, isEmpty);
    },
  );
  test('old draft persistence cannot remove the next account plan', () async {
    await sync.syncOnEnter();
    store.renamePlan('p', 'Local');
    transport.remote = [];
    await sync.syncOnEnter();
    final save = Completer<bool>();
    store.nextCacheWrite = save;
    final oldResolution = sync.resolveConflict('p', {'[]': 'local'});
    await settle();
    tokens.owner = 8;
    transport.remote = [item(plan('p', 'Other account'))];
    await sync.syncOnEnter();
    save.complete(true);
    await oldResolution;
    expect(store.buildSnapshotPayload().plans.single.name, 'Other account');
    expect(sync.drafts.value, isEmpty);
    expect(transport.deletes, isEmpty);
  });
  test('guest content requires explicit adoption', () async {
    final guest = edit(plan('guest'), {
      'customEvents': [
        {
          'id': 'e',
          'label': 'Guest',
          'day': 1,
          'sections': [1],
          'weeks': [1],
        },
      ],
    });
    store.applyPlanItems([guest]);
    transport.remote = [];
    await sync.syncOnEnter();
    expect(sync.needsAdoption.value, isTrue);
    expect(transport.writes, isEmpty);
    expect(await sync.adoptLocal(), isTrue);
    expect(transport.writes.first.$1.id, 'guest');
  });
  test('local delete uses the observed plan revision', () async {
    transport.remote = [item(), item(plan('q'))];
    await sync.syncOnEnter();
    store.deletePlan('p');
    timers.last.fire();
    await settle();
    expect(transport.deletes, [('p', 1)]);
    expect(transport.writes, isEmpty);
  });
  test('signed-out schedules do not issue requests', () async {
    tokens.token = null;
    await sync.syncOnEnter();
    await sync.flushPendingUpload();
    expect(transport.reads, 0);
    expect(transport.writes, isEmpty);
  });
  test('cache records are independent of local week state', () async {
    await sync.syncOnEnter();
    await store.flush;
    final cached = store.readPlanSyncCache(7)!;
    expect(jsonEncode(cached), isNot(contains('weekView')));
  });
  test('clearing an uploaded placeholder updates it instead of deleting it', () async {
    transport.remote = [];
    await sync.syncOnEnter();
    final defaultPlan = store.buildSnapshotPayload().plans.first;
    store.renamePlan(defaultPlan.id, 'Mine');
    timers.last.fire();
    await settle();
    expect(transport.writes.length, 1);
    store.renamePlan(defaultPlan.id, defaultPlan.name);
    timers.last.fire();
    await settle();
    expect(transport.deletes, isEmpty);
    expect(transport.writes.length, 2);
    expect(transport.writes.last.$1.id, defaultPlan.id);
  });
  test('guest adoption covers selected-only plans', () async {
    final guest = edit(plan('guest'), {'selectedCourses': ['101.01']});
    store.applyPlanItems([guest]);
    transport.remote = [];
    await sync.syncOnEnter();
    expect(sync.needsAdoption.value, isTrue);
    expect(transport.writes, isEmpty);
  });
  test('guest adoption covers multiple empty plans', () async {
    store.applyPlanItems([plan('a'), plan('b')]);
    transport.remote = [];
    await sync.syncOnEnter();
    expect(sync.needsAdoption.value, isTrue);
    expect(transport.writes, isEmpty);
  });
  test('plans are presented in creation order regardless of server order', () async {
    final tieB = edit(plan('b', 'B'), {'createdAt': 100});
    final tieA = edit(plan('a', 'A'), {'createdAt': 100});
    final late = edit(plan('late', 'Late'), {'createdAt': 200});
    transport.remote = [item(late), item(tieB), item(tieA)];
    await sync.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.map((p) => p.id).toList(), [
      'a',
      'b',
      'late',
    ]);
  });
  test('cache degrades without the plans copy and rebuilds from bases', () async {
    await sync.syncOnEnter();
    store.failLargeCacheWrites = true;
    store.renamePlan('p', 'Changed');
    timers.last.fire();
    await settle();
    expect(transport.writes.length, 1);
    await store.flush;
    final cached = store.readPlanSyncCache(7)!;
    expect(cached['plans'] as List, isEmpty);
    expect((cached['bases'] as Map)['p']['revision'], 2);
    // A device switching back to the account rebuilds from the degraded cache.
    final reloaded = ScheduleSyncController(
      transport: transport,
      tokenStorage: tokens,
      store: store,
      readUserId: () async => tokens.owner,
      debounceTimer: (delay, fire) => FakeTimer(fire),
    );
    tokens.owner = 8;
    await reloaded.syncOnEnter();
    tokens.owner = 7;
    await reloaded.syncOnEnter();
    expect(store.buildSnapshotPayload().plans.map((p) => p.id).toList(), ['p']);
    expect(store.buildSnapshotPayload().plans.first.name, 'Changed');
    reloaded.dispose();
  });
  test('write rejection reports rejected state distinct from capacity', () async {
    await sync.syncOnEnter();
    transport.writeError = const ApiException(
      fallbackMessage: 'frozen',
      statusCode: 403,
    );
    store.renamePlan('p', 'Local');
    timers.last.fire();
    await settle();
    expect(sync.blocked.value, isTrue);
    expect(sync.blockedReason.value, 'rejected');
  });
  test('quota conflict reports the capacity reason', () async {
    await sync.syncOnEnter();
    transport.writeError = const ApiException(
      fallbackMessage: 'quota',
      statusCode: 409,
    );
    store.renamePlan('p', 'Local');
    timers.last.fire();
    await settle();
    expect(sync.blocked.value, isTrue);
    expect(sync.blockedReason.value, 'capacity');
  });
}
