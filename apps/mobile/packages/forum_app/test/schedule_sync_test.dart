// 排课方案云同步（issue #537）语义测试：
// - 进页对账四分支：云端空（localEmpty 不动 / 有货上行）/ 快照
//   （localEmpty 整包采用 / dirty 或时钟不一致 → 冲突 / 一致不动）。
// - 整包采用：applyingRemote 守卫防回灌（不触发上行）、原子替换四字段、
//   syncedAt 推进并持久化。
// - 防抖上行：可注入计时器（不依赖 fake_async），成功推进 syncedAt 并清
//   dirty；400 静默停本轮；401 停至下次进页；网络错误保持 dirty 待重试。
// - 登出/未登录：零网络请求。
import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/schedule/schedule_sync.dart';

/// 内存 TokenStorage（登出 = clear）。
class _MemoryTokenStorage implements TokenStorage {
  String? _token;
  int userId = 1;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// Fake 传输层：记录请求，可编程响应。
class _FakeTransport implements PkPlansTransport {
  PkPlansSnapshot? remote;
  Object? fetchError;
  Completer<PkPlansSnapshot?>? pendingFetch;
  Completer<String>? pendingUpload;
  Object? uploadError;
  int fetchCount = 0;
  int uploadCount = 0;
  final List<PkPlanSnapshotPayload> uploaded = <PkPlanSnapshotPayload>[];
  String nextUpdatedAt = '2026-09-08T08:30:00Z';

  @override
  Future<PkPlansSnapshot?> fetchPlans() async {
    fetchCount++;
    final Object? error = fetchError;
    if (error != null) throw error;
    if (pendingFetch != null) return pendingFetch!.future;
    return remote;
  }

  @override
  Future<String> uploadPlans(PkPlanSnapshotPayload payload) async {
    uploadCount++;
    uploaded.add(payload);
    final Object? error = uploadError;
    if (error != null) throw error;
    if (pendingUpload != null) return pendingUpload!.future;
    return nextUpdatedAt;
  }
}

/// 可取消的手动计时器：防抖回调显式触发，测试确定性（免 fake_async）。
class _ManualTimer implements Timer {
  _ManualTimer(this._owner, this._onFire);

  final _ManualTimers _owner;
  final void Function() _onFire;
  bool _active = true;

  @override
  int get tick => 0;

  @override
  bool get isActive => _active;

  @override
  void cancel() {
    if (!_active) return;
    _active = false;
    _owner._remove(this);
  }

  void fire() {
    if (!_active) return;
    _active = false;
    _onFire();
  }
}

class _ManualTimers {
  final List<_ManualTimer> _pending = <_ManualTimer>[];
  final List<Duration> delays = <Duration>[];

  Timer schedule(Duration delay, void Function() onFire) {
    final _ManualTimer timer = _ManualTimer(this, onFire);
    _pending.add(timer);
    delays.add(delay);
    return timer;
  }

  /// 挂起防抖数（断言用）。
  int get pendingCount => _pending.length;

  /// 触发全部挂起防抖（模拟防抖窗口到期）。
  void firePending() {
    final List<_ManualTimer> ready = List.of(_pending);
    _pending.clear();
    for (final _ManualTimer timer in ready) {
      timer.fire();
    }
  }

  void _remove(_ManualTimer timer) => _pending.remove(timer);
}

ApiException _networkError() =>
    const NetworkException(fallbackMessage: 'offline');

ApiException _badRequest() => const ApiException(
  fallbackMessage: 'Request failed',
  messageCode: 'pk.requestFailed',
  params: <String, dynamic>{'detail': '方案数超过上限'},
  statusCode: 400,
);

ApiException _unauthorized() => const UnauthorizedException();

/// 构造一份云端快照（plan_cloud 一门已选课）。
PkPlansSnapshot _cloudSnapshot({String updatedAt = '2026-09-08T07:00:00Z'}) {
  return PkPlansSnapshot.fromJson(<String, dynamic>{
    'plans': <dynamic>[
      <String, dynamic>{
        'id': 'plan_cloud',
        'name': '云端方案',
        'createdAt': 1725000000000,
        'stagedCourses': <dynamic>[
          <String, dynamic>{
            'courseCode': 'X',
            'courseName': '高等数学',
            'courseNameReserved': '高等数学',
            'credit': 4,
            'courseType': '必',
            'teacher': <dynamic>[
              <String, dynamic>{'teacherName': '张老师', 'teacherCode': 'T001'},
            ],
            'status': 2,
            'courseDetail': <dynamic>[
              <String, dynamic>{
                'code': 'X.01',
                'status': 2,
                'campus': '四平',
                'teachers': <dynamic>[
                  <String, dynamic>{
                    'teacherName': '张老师',
                    'teacherCode': 'T001',
                  },
                ],
                'arrangementInfo': <dynamic>[
                  <String, dynamic>{
                    'arrangementText': '1-16周 周一 1-2节 A101',
                    'occupyDay': 1,
                    'occupyTime': <int>[1, 2],
                    'occupyWeek': List<int>.generate(16, (int i) => i + 1),
                    'occupyRoom': 'A101',
                    'teacherAndCode': '张老师(T001)',
                  },
                ],
              },
            ],
          },
        ],
        'selectedCourses': <String>['X.01'],
        'customEvents': <dynamic>[],
      },
    ],
    'activePlanId': 'plan_cloud',
    'majorSelected': <String, dynamic>{
      'calendarId': 121,
      'grade': 2024,
      'major': '080601',
      'majorName': '计算机科学与技术',
    },
    'weekView': <String, dynamic>{'week': 5, 'useCurrent': true},
    'updatedAt': updatedAt,
  });
}

/// 一门课的教学班详情（本地加课用）。
PkCourseDetail _localDetail(String code) {
  return PkCourseDetail(
    arrangementInfo: <PkArrangement>[
      PkArrangement(
        arrangementText: '1-16周 周一 1-2节 A101',
        occupyDay: 1,
        occupyTime: <int>[1, 2],
        occupyWeek: List<int>.generate(16, (int i) => i + 1),
        occupyRoom: 'A101',
        teacherAndCode: '张老师(T001)',
      ),
    ],
    campus: '四平',
    code: code,
    status: 0,
    teachers: <PkTeacher>[PkTeacher(teacherName: '张老师', teacherCode: 'T001')],
    teachingLanguage: '',
  );
}

typedef _Harness = (
  ScheduleStoreNotifier,
  _FakeTransport,
  _ManualTimers,
  _MemoryTokenStorage,
  ScheduleSyncController,
);

/// 构建测试组件。[seed] 在控制器创建**之前**执行（种子本地方案时钩子
/// 尚未绑定 → dirty 保持 false，用于「干净本地」分支）。
Future<_Harness> _harness({
  bool loggedIn = true,
  String? syncedAt,
  void Function(ScheduleStoreNotifier store)? seed,
}) async {
  SharedPreferences.setMockInitialValues(<String, Object>{
    ScheduleStorageKeys.syncedAt: ?syncedAt,
  });
  final ScheduleStoreNotifier store = ScheduleStoreNotifier();
  await store.ready;
  if (seed != null) {
    seed(store);
    await store.flush;
  }
  final _FakeTransport transport = _FakeTransport();
  final _ManualTimers timers = _ManualTimers();
  final _MemoryTokenStorage tokens = _MemoryTokenStorage();
  if (loggedIn) tokens.write('tok-sync');
  final ScheduleSyncController controller = ScheduleSyncController(
    transport: transport,
    tokenStorage: tokens,
    readUserId: () async => tokens.userId,
    store: store,
    debounceTimer: timers.schedule,
  );
  return (store, transport, timers, tokens, controller);
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('进页对账四分支', () {
    test('云端空 + 本地空壳 → 不动（零上行）', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = null;
      expect(await controller.syncOnEnter(), isNull);
      expect(transport.uploadCount, 0, reason: '云端空且本地空壳不应上行');
      expect(timers.pendingCount, 0);
    });

    test('云端空 + 本地有货 → 立即上行本地', () async {
      final (store, transport, _, _, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = null;
      expect(await controller.syncOnEnter(), isNull);
      expect(transport.uploadCount, 1);
      expect(store.syncedAt, '2026-09-08T08:30:00Z');
      expect(controller.isDirty, isFalse);
    });

    test('云端快照 + 本地空壳 → 整包采用（无弹窗、无上行）', () async {
      final (store, transport, _, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = _cloudSnapshot();
      expect(await controller.syncOnEnter(), isNull, reason: '空壳直接采用');
      expect(transport.uploadCount, 0, reason: '采用路径绝不回灌上行');
      expect(store.state.plans.single.id, 'plan_cloud');
      expect(store.state.activePlanId, 'plan_cloud');
      expect(store.state.majorSelected.calendarId, 121);
      expect(store.state.weekView.week, 5);
      expect(store.syncedAt, '2026-09-08T07:00:00Z');
      await store.flush;
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      expect(
        prefs.getString(ScheduleStorageKeys.syncedAt),
        '2026-09-08T07:00:00Z',
      );
      expect(prefs.getString(ScheduleStorageKeys.plans), isNotNull);
    });

    test('云端快照 + 本地有货（干净、时钟一致）→ 不动', () async {
      final (store, transport, _, _, controller) = await _harness(
        syncedAt: '2026-09-08T07:00:00Z',
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = _cloudSnapshot();
      expect(await controller.syncOnEnter(), isNull, reason: '时钟一致无需弹窗');
      expect(transport.uploadCount, 0);
      expect(store.state.plans.single.id, isNot('plan_cloud'));
    });

    test('云端快照 + 本地有货（时钟不一致）→ 返回冲突快照', () async {
      final (store, transport, _, _, controller) = await _harness(
        syncedAt: '2026-09-08T07:00:00Z',
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = _cloudSnapshot(updatedAt: '2026-09-09T00:00:00Z');
      final PkPlansSnapshot? conflict = await controller.syncOnEnter();
      expect(conflict, isNotNull, reason: '云端更新应上抛冲突');
      expect(conflict!.updatedAt, '2026-09-09T00:00:00Z');
      expect(transport.uploadCount, 0, reason: '弹窗决策前不自动上行');
    });

    test('云端快照 + 本地 dirty → 返回冲突快照', () async {
      final (store, transport, _, _, controller) = await _harness(
        syncedAt: '2026-09-08T07:00:00Z',
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      store.selectClass(_localDetail('L.01'), '线性代数');
      expect(controller.isDirty, isTrue);
      transport.remote = _cloudSnapshot();
      final PkPlansSnapshot? conflict = await controller.syncOnEnter();
      expect(conflict, isNotNull, reason: '本地有未上行变更应上抛冲突');
      expect(transport.uploadCount, 0);
    });
  });

  group('冲突弹窗决策', () {
    test('「使用云端」整包采用且不回灌', () async {
      final (store, transport, timers, _, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      await controller.adoptRemote(_cloudSnapshot());
      await store.flush;
      expect(store.state.plans.single.id, 'plan_cloud');
      expect(store.state.plans.single.stagedCourses.single.courseCode, 'X');
      expect(store.syncedAt, '2026-09-08T07:00:00Z');
      expect(controller.isDirty, isFalse);
      expect(timers.pendingCount, 0, reason: '采用路径不排防抖');
      expect(transport.uploadCount, 0);
    });

    test('「保留本地」立即 PUT 本地快照', () async {
      final (store, transport, _, _, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.remote = _cloudSnapshot();
      await controller.syncOnEnter();
      await controller.keepLocal();
      expect(transport.uploadCount, 1);
      expect(
        transport.uploaded.single.plans.single.stagedCourses.single.courseCode,
        'L',
      );
      expect(store.syncedAt, '2026-09-08T08:30:00Z');
    });
  });

  group('防抖上行', () {
    test('本地变更 → 3s 防抖 → PUT 推进 syncedAt', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      store.selectClass(_localDetail('L.01'), '线性代数');
      expect(controller.isDirty, isTrue);
      expect(timers.delays, <Duration>[ScheduleSyncController.debounceDelay]);
      expect(transport.uploadCount, 0, reason: '防抖窗口内不上行');
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(transport.uploadCount, 1);
      expect(store.syncedAt, '2026-09-08T08:30:00Z');
      expect(controller.isDirty, isFalse);
    });

    test('连续变更合并为一次上行（防抖重置）', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      store.selectClass(_localDetail('L.01'), '线性代数');
      store.selectClass(_localDetail('M.01'), '大学物理');
      expect(timers.pendingCount, 1, reason: '第二次变更重置防抖');
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(transport.uploadCount, 1, reason: '两次变更合并一次上行');
      expect(
        transport.uploaded.single.plans.single.stagedCourses,
        hasLength(2),
      );
    });

    test('paused 冲刷：dirty 且登录 → 立即上行', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      store.selectClass(_localDetail('L.01'), '线性代数');
      expect(timers.pendingCount, 1);
      await controller.flushPendingUpload();
      expect(transport.uploadCount, 1);
      expect(timers.pendingCount, 0, reason: '冲刷取消挂起防抖');
      expect(controller.isDirty, isFalse);
    });
  });

  group('失败分类', () {
    test('400：静默停本轮（dirty 保留，不再自动重试）', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      transport.uploadError = _badRequest();
      store.selectClass(_localDetail('L.01'), '线性代数');
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(transport.uploadCount, 1);
      expect(controller.isDirty, isTrue, reason: '400 保留未保存的本地状态');

      // 后续新变更按新轮次处理（新载荷可能通过校验）。
      store.selectClass(_localDetail('N.01'), '概率论');
      expect(controller.isDirty, isTrue);
    });

    test('网络错误：保持 dirty 待重试（paused 冲刷成功后清零）', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      transport.uploadError = _networkError();
      store.selectClass(_localDetail('L.01'), '线性代数');
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(controller.isDirty, isTrue, reason: '网络错误保留 dirty');

      transport.uploadError = null;
      await controller.flushPendingUpload();
      expect(transport.uploadCount, 2);
      expect(store.syncedAt, '2026-09-08T08:30:00Z');
      expect(controller.isDirty, isFalse);
    });

    test('401：上行停至下次进页；进页复位后恢复', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();

      transport.uploadError = _unauthorized();
      store.selectClass(_localDetail('L.01'), '线性代数');
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(controller.isDirty, isTrue, reason: '401 未上行，dirty 保留');

      // 停止期间本地变更不再排防抖。
      timers.delays.clear();
      store.selectClass(_localDetail('N.01'), '概率论');
      expect(timers.delays, isEmpty, reason: '401 后不再排上行');

      // 下次进页：复位停止标记并恢复对账/上行。
      transport.uploadError = null;
      transport.remote = null;
      await controller.syncOnEnter();
      expect(transport.uploadCount, 2, reason: '进页复位后恢复上行');
      expect(controller.isDirty, isFalse);
    });

    test('进页拉取 401：本次放弃对账并停至下次进页', () async {
      final (store, transport, _, _, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), '线性代数'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      transport.fetchError = _unauthorized();
      expect(await controller.syncOnEnter(), isNull);
      expect(transport.uploadCount, 0, reason: '对账失败不触发上行');

      transport.fetchError = null;
      transport.remote = null;
      await controller.syncOnEnter();
      expect(transport.uploadCount, 1, reason: '下次进页恢复');
    });
  });

  group('登出与未登录', () {
    test('未登录：进页零请求、防抖到期零上行', () async {
      final (store, transport, timers, _, controller) = await _harness(
        loggedIn: false,
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      expect(await controller.syncOnEnter(), isNull);
      expect(transport.fetchCount, 0, reason: '未登录零网络请求');

      store.selectClass(_localDetail('L.01'), '线性代数');
      expect(controller.isDirty, isTrue, reason: 'dirty 仍记录（登录后可对账）');
      // 防抖照常排程，但到期时由 _uploadNow 统一拦截（零网络请求）。
      timers.firePending();
      await Future<void>.delayed(Duration.zero);
      expect(transport.uploadCount, 0, reason: '未登录防抖到期也不上行');
    });

    test('登出后：paused 冲刷零请求', () async {
      final (store, transport, _, tokens, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      store.selectClass(_localDetail('L.01'), '线性代数');
      await tokens.clear();
      await controller.flushPendingUpload();
      expect(transport.uploadCount, 0, reason: '登出后零网络请求');
    });
  });

  group('store 钩子边界（applyingRemote 防回灌）', () {
    test('整包采用期间的持久化不触发上行钩子', () async {
      final (store, transport, timers, _, controller) = await _harness();
      addTearDown(controller.dispose);
      addTearDown(store.dispose);

      // 控制器钩子保持绑定：若 applyingRemote 守卫失效，采用期间的
      // _persistPlanData 会立刻排程防抖（pendingCount > 0）。
      await controller.adoptRemote(_cloudSnapshot());
      await store.flush;
      expect(timers.pendingCount, 0, reason: '采用路径绝不触发上行钩子');
      expect(controller.isDirty, isFalse);
      expect(transport.uploadCount, 0, reason: '无回灌上行');

      // 对照：本地写入路径正常排程。
      store.selectClass(_localDetail('Z.01'), '统计学');
      expect(timers.pendingCount, 1, reason: '本地路径正常触发防抖');
    });
  });
  test('failed reconciliation cannot upload later edits', () async {
    final (store, transport, timers, _, controller) = await _harness();
    addTearDown(controller.dispose);
    addTearDown(store.dispose);
    transport.fetchError = _networkError();
    await controller.syncOnEnter();
    store.selectClass(_localDetail('L.01'), 'local');
    timers.firePending();
    await Future<void>.delayed(Duration.zero);
    expect(transport.uploadCount, 0);
  });

  test('dirty empty plans cannot resurrect deleted courses', () async {
    final (store, transport, _, _, controller) = await _harness();
    addTearDown(controller.dispose);
    addTearDown(store.dispose);
    transport.remote = _cloudSnapshot();
    await controller.syncOnEnter();
    store.clearActivePlan();
    expect(await controller.syncOnEnter(), isNotNull);
    expect(store.isLocalEmpty, isTrue);
  });

  test('keep-local choice survives a transient upload failure', () async {
    final (store, transport, _, _, controller) = await _harness(
      seed: (store) => store.selectClass(_localDetail('L.01'), 'local'),
    );
    addTearDown(controller.dispose);
    addTearDown(store.dispose);
    transport.remote = _cloudSnapshot();
    await controller.syncOnEnter();
    transport.uploadError = _networkError();
    await controller.keepLocal();
    expect(controller.isDirty, isTrue);
    transport.uploadError = null;
    await controller.flushPendingUpload();
    expect(transport.uploadCount, 2);
  });

  test('disposed controller ignores a late cloud response', () async {
    final (store, transport, _, _, controller) = await _harness();
    addTearDown(store.dispose);
    transport.pendingFetch = Completer<PkPlansSnapshot?>();
    final entering = controller.syncOnEnter();
    await Future<void>.delayed(Duration.zero);
    controller.dispose();
    transport.pendingFetch!.complete(_cloudSnapshot());
    await entering;
    expect(store.isLocalEmpty, isTrue);
  });
  test(
    'account change does not auto-upload retained plans to an empty account',
    () async {
      final (store, transport, _, tokens, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), 'local'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      await controller.syncOnEnter();
      tokens.userId = 2;
      final conflict = await controller.syncOnEnter();
      expect(conflict, isNotNull);
      expect(transport.uploadCount, 1);
      await controller.adoptRemote(conflict!);
      expect(store.isLocalEmpty, isTrue);
      expect(store.syncOwner, 2);
    },
  );

  test('reconciliation waits for the active upload before fetching', () async {
    final (store, transport, timers, _, controller) = await _harness();
    addTearDown(controller.dispose);
    addTearDown(store.dispose);
    await controller.syncOnEnter();
    transport.pendingUpload = Completer<String>();
    store.selectClass(_localDetail('L.01'), 'local');
    timers.firePending();
    await Future<void>.delayed(Duration.zero);
    final entering = controller.syncOnEnter();
    await Future<void>.delayed(Duration.zero);
    expect(transport.fetchCount, 1);
    transport.remote = _cloudSnapshot(updatedAt: '2026-09-10T00:00:00Z');
    transport.pendingUpload!.complete('2026-09-09T00:00:00Z');
    final conflict = await entering;
    expect(conflict?.updatedAt, '2026-09-10T00:00:00Z');
    expect(transport.fetchCount, 2);
  });

  test('409 publishes a fresh conflict and preserves dirty state', () async {
    final (store, transport, timers, _, controller) = await _harness();
    addTearDown(controller.dispose);
    addTearDown(store.dispose);
    transport.remote = _cloudSnapshot();
    await controller.syncOnEnter();
    store.selectClass(_localDetail('L.01'), 'local');
    transport.uploadError = const ApiException(
      fallbackMessage: 'conflict',
      statusCode: 409,
    );
    transport.remote = _cloudSnapshot(updatedAt: '2026-09-10T00:00:00Z');
    timers.firePending();
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    expect(transport.uploaded.single.baseUpdatedAt, '2026-09-08T07:00:00Z');
    expect(controller.conflict.value?.updatedAt, '2026-09-10T00:00:00Z');
    expect(controller.isDirty, isTrue);
  });

  test(
    'initial upload conflict also re-fetches rather than remaining stalled',
    () async {
      final (store, transport, _, _, controller) = await _harness(
        seed: (store) => store.selectClass(_localDetail('L.01'), 'local'),
      );
      addTearDown(controller.dispose);
      addTearDown(store.dispose);
      transport.pendingUpload = Completer<String>();
      final entering = controller.syncOnEnter();
      await Future<void>.delayed(Duration.zero);
      transport.remote = _cloudSnapshot();
      transport.pendingUpload!.completeError(
        const ApiException(fallbackMessage: 'conflict', statusCode: 409),
      );
      await entering;
      await Future<void>.delayed(Duration.zero);
      expect(controller.conflict.value, isNotNull);
    },
  );
}
