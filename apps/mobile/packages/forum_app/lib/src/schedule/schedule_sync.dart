// Per-plan revisions and persisted ancestors match the Web scheduler. Only dirty
// plans retry; preferences never enter a CAS and no clean-state timer polls.
import 'dart:async';
import 'package:core/core.dart';
import 'package:auth/auth.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import '../providers.dart';
import 'schedule_store.dart';

abstract class PkPlansTransport {
  Future<List<PkPlanItem>> list();
  Future<PkPlanItem> put(PkPlan plan, int baseRevision);
  Future<void> remove(String id, int baseRevision);
}

class PkPlansRepositoryTransport implements PkPlansTransport {
  PkPlansRepositoryTransport(this.repository);
  final PkRepository repository;
  @override
  Future<List<PkPlanItem>> list() => repository.listPlanItems();
  @override
  Future<PkPlanItem> put(PkPlan plan, int baseRevision) =>
      repository.putPlanItem(plan, baseRevision);
  @override
  Future<void> remove(String id, int baseRevision) =>
      repository.deletePlanItem(id, baseRevision);
}

class PlanSyncConflict {
  const PlanSyncConflict(
    this.id,
    this.base,
    this.local,
    this.remote,
    this.fields,
  );
  final String id;
  final PkPlan? base, local;
  final PkPlanItem? remote;
  final List<PkPlanMergeConflict> fields;
}

class ScheduleSyncController {
  ScheduleSyncController({
    required this.transport,
    required this.tokenStorage,
    required this.store,
    required this.readUserId,
    Timer Function(Duration, void Function())? debounceTimer,
  }) : _timerFactory = debounceTimer ?? ((delay, fire) => Timer(delay, fire)) {
    scheduleLocalPlansChanged = _onLocalChange;
  }
  static const debounceDelay = Duration(seconds: 3);
  final PkPlansTransport transport;
  final TokenStorage tokenStorage;
  final ScheduleStoreNotifier store;
  final Future<int?> Function() readUserId;
  final Timer Function(Duration, void Function()) _timerFactory;
  final conflicts = ValueNotifier<List<PlanSyncConflict>>([]);
  final drafts = ValueNotifier<Map<String, PkPlan>>({});
  final needsAdoption = ValueNotifier(false);
  final blocked = ValueNotifier(false);
  Map<String, PkPlanItem> _bases = {};
  String? _placeholderID, _placeholderKey;
  int? _owner;
  int _generation = 0, _retrySeconds = 3;
  bool _disposed = false,
      _reconciled = false,
      _authStopped = false,
      _persistenceFailed = false;
  DateTime? _lastRead;
  Timer? _timer;
  Future<void>? _operation;
  List<PkPlan> get _plans => store.buildSnapshotPayload().plans;
  List<PkPlan> get _content => _plans
      .where(
        (p) => p.id != _placeholderID || schedulePlanKey(p) != _placeholderKey,
      )
      .toList();
  PkPlan? _local(String id) {
    for (final p in _content) {
      if (p.id == id) return p;
    }
    return null;
  }

  List<String> get _dirtyIDs => {..._bases.keys, ..._content.map((p) => p.id)}
      .where(
        (id) =>
            schedulePlanKey(_local(id)) != schedulePlanKey(_bases[id]?.plan),
      )
      .toList();
  bool get isDirty => _persistenceFailed || _dirtyIDs.isNotEmpty;
  void dispose() {
    _disposed = true;
    _generation++;
    cancelPendingUpload();
    conflicts.dispose();
    drafts.dispose();
    needsAdoption.dispose();
    blocked.dispose();
    if (identical(scheduleLocalPlansChanged, _onLocalChange)) {
      scheduleLocalPlansChanged = null;
    }
  }

  Future<int?> _identity() async {
    try {
      final token = await tokenStorage.read();
      return token == null || token.isEmpty ? null : await readUserId();
    } catch (_) {
      return null;
    }
  }

  Future<bool> _current(int run) async {
    final identity = await _identity();
    return !_disposed &&
        run == _generation &&
        _owner != null &&
        identity == _owner &&
        store.syncOwner == _owner;
  }

  void _markPlaceholder() {
    final p = _plans.first;
    _placeholderID = p.id;
    _placeholderKey = schedulePlanKey(p);
  }

  Future<bool> _persist() async {
    if (_disposed ||
        _owner == null ||
        needsAdoption.value ||
        store.syncOwner != _owner) {
      return false;
    }
    final run = _generation;
    final saved = await store.writePlanSyncCache(_owner!, {
      'bases': {for (final e in _bases.entries) e.key: e.value.toJson()},
      'plans': _plans.map((p) => p.toJson()).toList(),
      'drafts': {for (final e in drafts.value.entries) e.key: e.value.toJson()},
      'placeholderID': _placeholderID,
      'placeholderKey': _placeholderKey,
    });
    if (run == _generation && !_disposed) _persistenceFailed = !saved;
    return saved;
  }

  Future<void> _prepare(int owner) async {
    if (_owner == owner) return;
    cancelPendingUpload();
    _generation++;
    _operation = null;
    final run = _generation;
    _owner = owner;
    _reconciled = false;
    _authStopped = false;
    blocked.value = false;
    conflicts.value = [];
    needsAdoption.value = false;
    _placeholderID = null;
    _placeholderKey = null;
    final cache = store.readPlanSyncCache(owner);
    _bases = {};
    drafts.value = {};
    if (cache != null) {
      try {
        _bases = {
          for (final e in (cache['bases'] as Map).entries)
            e.key as String: PkPlanItem.fromJson(
              Map<String, dynamic>.from(e.value as Map),
            ),
        };
        drafts.value = {
          for (final e in (cache['drafts'] as Map).entries)
            e.key as String: PkPlan.fromJson(
              Map<String, dynamic>.from(e.value as Map),
            ),
        };
        _placeholderID = cache['placeholderID'] as String?;
        _placeholderKey = cache['placeholderKey'] as String?;
      } catch (_) {
        _bases = {};
        drafts.value = {};
      }
    }
    if (store.syncOwner != null && store.syncOwner != owner) {
      if (!await store.setSyncOwner(
        owner,
        canWrite: () => !_disposed && run == _generation,
      )) {
        if (!_disposed && run == _generation) needsAdoption.value = true;
        return;
      }
      final plans = (cache?['plans'] as List? ?? [])
          .map((p) => PkPlan.fromJson(Map<String, dynamic>.from(p as Map)))
          .toList();
      if (!await _current(run)) return;
      store.applyPlanItems(plans);
      if (plans.isEmpty) _markPlaceholder();
    } else if (store.syncOwner == null &&
        _plans.any(
          (p) => p.stagedCourses.isNotEmpty || p.customEvents.isNotEmpty,
        )) {
      if (!_disposed && run == _generation) needsAdoption.value = true;
      return;
    } else if (!await store.setSyncOwner(
      owner,
      canWrite: () => !_disposed && run == _generation,
    )) {
      if (!_disposed && run == _generation) needsAdoption.value = true;
      return;
    }
    if (!await _current(run)) return;
    if (cache == null &&
        !store.syncDirty &&
        _plans.length == 1 &&
        _plans.every(
          (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
        )) {
      _markPlaceholder();
    }
    await _persist();
  }

  void _apply(String id, PkPlan? plan) {
    final plans = _content.toList();
    final index = plans.indexWhere((p) => p.id == id);
    if (index >= 0) {
      if (plan != null) {
        plans[index] = plan;
      } else {
        plans.removeAt(index);
      }
    } else if (plan != null) {
      plans.add(plan);
    } else {
      return;
    }
    store.applyPlanItems(plans);
    if (plans.isEmpty) _markPlaceholder();
  }

  void _reconcile(String id, PkPlanItem? remote) {
    final base = _bases[id]?.plan, local = _local(id);
    final result = mergeSchedulePlan(base, local, remote?.plan);
    conflicts.value = conflicts.value.where((c) => c.id != id).toList();
    if (result.conflicts.isNotEmpty) {
      conflicts.value = [
        ...conflicts.value,
        PlanSyncConflict(id, base, local, remote, result.conflicts),
      ];
      return;
    }
    _apply(id, result.plan);
    if (remote != null) {
      _bases[id] = remote;
    } else if (result.plan == null) {
      _bases.remove(id);
    }
  }

  Future<bool> _archive(String id, PkPlan plan) async {
    final run = _generation;
    drafts.value = {...drafts.value, id: plan};
    // Never remove the final visible copy until recovery storage succeeds.
    if (!await _persist() || !await _current(run)) return false;
    _apply(id, null);
    _bases.remove(id);
    conflicts.value = conflicts.value.where((c) => c.id != id).toList();
    return true;
  }

  void _schedule([int seconds = 3]) {
    cancelPendingUpload();
    if (_disposed ||
        _authStopped ||
        needsAdoption.value ||
        blocked.value ||
        !isDirty) {
      return;
    }
    if (_dirtyIDs.every((id) => conflicts.value.any((c) => c.id == id))) return;
    _timer = _timerFactory(Duration(seconds: seconds), () {
      _timer = null;
      unawaited(_run(!_reconciled));
    });
  }

  void _fail(Object error) {
    if (error is ApiException && error.statusCode == 401) {
      _authStopped = true;
    } else if (error is ApiException && [400, 403].contains(error.statusCode)) {
      blocked.value = true;
    } else {
      _schedule(_retrySeconds);
      _retrySeconds = (_retrySeconds * 2).clamp(3, 60);
    }
  }

  Future<void> _upload(int run) async {
    for (final id in _dirtyIDs) {
      if (!await _current(run)) return;
      if (conflicts.value.any((c) => c.id == id)) continue;
      final local = _local(id);
      final plan = local == null ? null : PkPlan.fromJson(local.toJson());
      final base = _bases[id]?.revision ?? 0;
      try {
        if (plan != null) {
          final item = await transport.put(plan, base);
          if (!await _current(run)) return;
          _bases[id] = item;
        } else {
          await transport.remove(id, base);
          if (!await _current(run)) return;
          _bases.remove(id);
        }
        await _persist();
        if (!await _current(run)) return;
        _retrySeconds = 3;
      } catch (error) {
        if (!await _current(run)) return;
        if (error is ApiException && error.statusCode == 410) {
          _reconcile(id, null);
        } else if (error is ApiException &&
            error.statusCode == 409 &&
            error.responseData is Map) {
          _reconcile(
            id,
            PkPlanItem.fromJson(
              Map<String, dynamic>.from(error.responseData as Map),
            ),
          );
        } else if (error is ApiException && error.statusCode == 409) {
          blocked.value = true;
          return;
        } else {
          _fail(error);
          return;
        }
        await _persist();
      }
    }
    if (await _current(run)) _schedule();
  }

  Future<void> _perform(bool read, int run) async {
    try {
      if (read) {
        final items = await transport.list();
        if (!await _current(run)) return;
        final remote = {for (final item in items) item.plan.id: item};
        if (_bases.isEmpty &&
            store.syncedAt.isNotEmpty &&
            !store.syncDirty &&
            items.isNotEmpty) {
          store.applyPlanItems(items.map((i) => i.plan).toList());
        }
        for (final id in {
          ..._bases.keys,
          ..._content.map((p) => p.id),
          ...remote.keys,
        }) {
          _reconcile(id, remote[id]);
        }
        final overflow = _plans.length - kMaxPlans;
        if (overflow > 0) {
          for (final p
              in _plans
                  .where((p) => !remote.containsKey(p.id))
                  .toList()
                  .reversed
                  .take(overflow)) {
            if (!await _archive(p.id, p)) break;
          }
          if (!await _current(run)) return;
          blocked.value = true;
        }
        _reconciled = true;
        _lastRead = DateTime.now();
        await _persist();
      }
      await _upload(run);
    } catch (error) {
      if (await _current(run)) _fail(error);
    }
  }

  Future<void> _run(bool read) {
    if (_disposed || _authStopped || needsAdoption.value || _owner == null) {
      return Future.value();
    }
    if (_operation != null) return _operation!;
    cancelPendingUpload();
    final operation = _perform(read, _generation);
    _operation = operation;
    return operation.whenComplete(() {
      if (identical(_operation, operation)) _operation = null;
    });
  }

  Future<void> syncOnEnter() async {
    if (_disposed) return;
    await store.ready;
    final owner = await _identity();
    if (_disposed || owner == null) return;
    await _prepare(owner);
    if (_disposed ||
        _owner != owner ||
        needsAdoption.value ||
        !await _current(_generation)) {
      return;
    }
    blocked.value = false;
    await _run(true);
  }

  Future<void> onResume() async {
    if (_disposed) return;
    if (isDirty) {
      await flushPendingUpload();
    }
    if (_lastRead == null ||
        DateTime.now().difference(_lastRead!) >= const Duration(seconds: 30)) {
      await syncOnEnter();
    }
  }

  Future<bool> adoptLocal() async {
    if (_owner == null || _disposed || await _identity() != _owner) {
      return false;
    }
    final run = _generation;
    if (!await store.setSyncOwner(
      _owner!,
      canWrite: () => !_disposed && run == _generation,
    )) {
      return false;
    }
    if (!await _current(run)) return false;
    needsAdoption.value = false;
    _bases = {};
    await _persist();
    await _run(true);
    return !isDirty;
  }

  Future<void> resolveConflict(String id, Map<String, String> choices) async {
    final run = _generation;
    if (_disposed || !await _current(run)) return;
    final matches = conflicts.value.where((c) => c.id == id);
    if (matches.isEmpty) return;
    final conflict = matches.first;
    final result = mergeSchedulePlan(
      conflict.base,
      _local(id),
      conflict.remote?.plan,
      choices,
    );
    if (result.conflicts.isNotEmpty) return;
    if (conflict.remote == null && result.plan != null) {
      if (!await _archive(id, result.plan!)) return;
    } else {
      _apply(id, result.plan);
      if (conflict.remote != null) {
        _bases[id] = conflict.remote!;
      } else {
        _bases.remove(id);
      }
    }
    conflicts.value = conflicts.value.where((c) => c.id != id).toList();
    await _persist();
    if (await _current(run)) await _run(false);
  }

  bool restoreDraft(String id) {
    final draft = drafts.value[id];
    if (_disposed || draft == null) return false;
    if (_content.length >= kMaxPlans) {
      blocked.value = true;
      return false;
    }
    final json = draft.toJson()..['id'] = newScheduleId('plan');
    final plan = PkPlan.fromJson(json);
    _apply(plan.id, plan);
    drafts.value = {...drafts.value}..remove(id);
    blocked.value = false;
    _onLocalChange();
    return true;
  }

  void _onLocalChange() {
    if (_disposed) return;
    store.markSyncDirty();
    unawaited(_persist());
    _schedule();
  }

  Future<void> flushPendingUpload() async {
    cancelPendingUpload();
    if (_owner == null) {
      await syncOnEnter();
    } else {
      await _run(!_reconciled);
    }
  }

  void cancelPendingUpload() {
    _timer?.cancel();
    _timer = null;
  }
}

final scheduleSyncControllerProvider = Provider<ScheduleSyncController>((ref) {
  ref.watch(offlineCacheEpochProvider);
  final storage = ref.watch(tokenStorageProvider);
  final controller = ScheduleSyncController(
    transport: PkPlansRepositoryTransport(ref.watch(pkRepositoryProvider)),
    tokenStorage: storage,
    readUserId: () async =>
        storage is SecureTokenStorage ? storage.readUserId() : null,
    store: ref.watch(scheduleStoreProvider.notifier),
  );
  final subscription = Connectivity().onConnectivityChanged
      .map((types) => types.any((type) => type != ConnectivityResult.none))
      .distinct()
      .listen(
        (online) {
          if (online && controller.isDirty) {
            unawaited(controller.flushPendingUpload());
          }
        },
        onError: (_) {
          /* Dirty retries remain available when platform events fail. */
        },
      );
  ref.onDispose(() {
    unawaited(subscription.cancel());
    controller.dispose();
  });
  return controller;
});
