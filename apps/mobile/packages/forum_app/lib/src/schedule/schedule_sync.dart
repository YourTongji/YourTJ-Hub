// Schedule cloud synchronization shares the Web reconciliation rules: no writes
// before a successful read or while a conflict is unresolved; stale writes return
// 409 and re-open reconciliation. Retained local plans have an account owner.
import 'dart:async';

import 'package:core/core.dart';
import 'package:auth/auth.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers.dart';
import 'schedule_store.dart';

/// 方案快照传输抽象（GET/PUT /api/pk/plans；测试注入 fake）。
abstract class PkPlansTransport {
  /// 拉取云端快照；云端为空返回 null。
  Future<PkPlansSnapshot?> fetchPlans();

  /// 整包上行快照；返回服务端新同步时钟 updatedAt。
  Future<String> uploadPlans(PkPlanSnapshotPayload payload);
}

/// [PkPlansTransport] 的生产实现：桥接 [PkRepository]。
class PkPlansRepositoryTransport implements PkPlansTransport {
  PkPlansRepositoryTransport(this._repository);

  final PkRepository _repository;

  @override
  Future<PkPlansSnapshot?> fetchPlans() => _repository.getPlans();

  @override
  Future<String> uploadPlans(PkPlanSnapshotPayload payload) async =>
      (await _repository.putPlans(payload)).updatedAt;
}

/// Reconciles before writing and guards uploads with the observed server revision.
class ScheduleSyncController {
  ScheduleSyncController({
    required this.transport,
    required this.tokenStorage,
    required this.store,
    required this.readUserId,
    Timer Function(Duration delay, void Function() onFire)? debounceTimer,
  }) : _debounceTimer = debounceTimer ?? _defaultDebounceTimer {
    scheduleLocalPlansChanged = _onLocalPlansChanged;
  }

  static Timer _defaultDebounceTimer(Duration delay, void Function() onFire) =>
      Timer(delay, onFire);
  static const Duration debounceDelay = Duration(seconds: 3);
  final PkPlansTransport transport;
  final TokenStorage tokenStorage;
  final ScheduleStoreNotifier store;
  final Future<int?> Function() readUserId;
  final Timer Function(Duration delay, void Function() onFire) _debounceTimer;
  final ValueNotifier<PkPlansSnapshot?> conflict = ValueNotifier(null);
  bool _dirty = false;
  bool _stopped = false;
  bool _disposed = false;
  bool _reconciled = false;
  bool _entering = false;
  bool _recheckAfterEntry = false;
  int _localChangeSeq = 0;
  int? _owner;
  String _baseUpdatedAt = '';
  Timer? _pendingUpload;
  Future<void>? _uploadFuture;

  bool get isDirty => _dirty || store.syncDirty;

  void dispose() {
    _disposed = true;
    cancelPendingUpload();
    conflict.dispose();
    if (identical(scheduleLocalPlansChanged, _onLocalPlansChanged)) {
      scheduleLocalPlansChanged = null;
    }
  }

  Future<int?> _identity() async {
    try {
      final token = await tokenStorage.read();
      if (_disposed || token == null || token.isEmpty) return null;
      return await readUserId();
    } catch (_) {
      return null;
    }
  }

  Future<bool> _isCurrent() async {
    final id = await _identity();
    return !_disposed && _owner != null && id == _owner;
  }

  Future<PkPlansSnapshot?> syncOnEnter() async {
    if (_disposed || _entering) return null;
    _entering = true;
    _reconciled = false;
    cancelPendingUpload();
    try {
      await store.ready;
      if (_uploadFuture != null) await _uploadFuture;
      _owner = await _identity();
      if (_owner == null || _disposed) return null;
      _stopped = false;
      final remote = await transport.fetchPlans();
      if (!await _isCurrent()) return null;
      _baseUpdatedAt = remote?.updatedAt ?? '';
      if (store.syncOwner != null && store.syncOwner != _owner) {
        conflict.value =
            remote ??
            PkPlansSnapshot(
              plans: [],
              activePlanId: '',
              majorSelected: PkMajorSelection(),
              weekView: PkWeekView(),
              updatedAt: '',
            );
        return conflict.value;
      }
      if (!await store.setSyncOwner(_owner!)) return null;
      if (remote == null) {
        _reconciled = true;
        if (!store.isLocalEmpty || isDirty) {
          _dirty = true;
          store.markSyncDirty();
          await _uploadNow();
        }
        return null;
      }
      if (!isDirty && store.syncedAt.isEmpty && store.isLocalEmpty) {
        await adoptRemote(remote);
        return null;
      }
      if (isDirty || store.syncedAt != remote.updatedAt) {
        conflict.value = remote;
        return remote;
      }
      _reconciled = true;
      return null;
    } on ApiException catch (e) {
      if (e.statusCode == 401) _stopped = true;
      return null;
    } catch (_) {
      return null;
    } finally {
      _entering = false;
      if (_recheckAfterEntry && !_disposed) {
        _recheckAfterEntry = false;
        unawaited(syncOnEnter());
      }
    }
  }

  Future<void> adoptRemote(PkPlansSnapshot snapshot) async {
    if (_disposed) return;
    if (_uploadFuture != null) {
      await _uploadFuture;
      await syncOnEnter();
      return;
    }
    _owner ??= await _identity();
    if (!await _isCurrent()) return;
    cancelPendingUpload();
    store.applyRemoteSnapshot(snapshot);
    _dirty = !await store.markSyncedAt(snapshot.updatedAt);
    if (!await _isCurrent()) return;
    if (!_dirty && !await store.setSyncOwner(_owner!)) return;
    _baseUpdatedAt = snapshot.updatedAt;
    conflict.value = null;
    _reconciled = true;
  }

  Future<void> keepLocal() async {
    if (_disposed || conflict.value == null || !await _isCurrent()) return;
    if (!await store.setSyncOwner(_owner!)) return;
    cancelPendingUpload();
    conflict.value = null;
    _dirty = true;
    store.markSyncDirty();
    _reconciled = true;
    await _uploadNow();
  }

  Future<void> flushPendingUpload() async {
    cancelPendingUpload();
    await _uploadNow();
  }

  void cancelPendingUpload() {
    _pendingUpload?.cancel();
    _pendingUpload = null;
  }

  void _scheduleUpload() {
    cancelPendingUpload();
    if (_disposed || _stopped || !_reconciled) return;
    _pendingUpload = _debounceTimer(debounceDelay, () {
      _pendingUpload = null;
      unawaited(_uploadNow());
    });
  }

  void _onLocalPlansChanged() {
    if (_disposed) return;
    _dirty = true;
    store.markSyncDirty();
    _localChangeSeq++;
    _scheduleUpload();
  }

  Future<void> _uploadNow() {
    if (_disposed ||
        _stopped ||
        !_reconciled ||
        !isDirty ||
        _uploadFuture != null) {
      return Future.value();
    }
    final operation = _performUpload();
    _uploadFuture = operation;
    return operation.whenComplete(() => _uploadFuture = null);
  }

  Future<void> _performUpload() async {
    if (!await _isCurrent()) return;
    final seq = _localChangeSeq;
    try {
      final updatedAt = await transport.uploadPlans(
        store.buildSnapshotPayload(baseUpdatedAt: _baseUpdatedAt),
      );
      if (!await _isCurrent()) return;
      if (updatedAt.isEmpty) throw StateError('Missing sync revision');
      _baseUpdatedAt = updatedAt;
      if (_localChangeSeq != seq) {
        _scheduleUpload();
        return;
      }
      _dirty = !await store.markSyncedAt(updatedAt);
    } on ApiException catch (e) {
      if (!await _isCurrent()) return;
      if (e.statusCode == 401) {
        _stopped = true;
      } else if (e.statusCode == 409) {
        _reconciled = false;
        if (_entering) {
          _recheckAfterEntry = true;
        } else {
          unawaited(syncOnEnter());
        }
      } else if (e.statusCode == 400 || e.statusCode == 403) {
        debugPrint('pk plans sync rejected (${e.statusCode})');
      }
    } catch (_) {
      // Keep the local revision pending for a later edit or lifecycle flush.
    }
  }
}

final Provider<ScheduleSyncController>
scheduleSyncControllerProvider = Provider((ref) {
  // Discard in-flight callbacks and timers when the authenticated session changes.
  ref.watch(offlineCacheEpochProvider);
  final storage = ref.watch(tokenStorageProvider);
  final controller = ScheduleSyncController(
    transport: PkPlansRepositoryTransport(ref.watch(pkRepositoryProvider)),
    tokenStorage: storage,
    readUserId: () async =>
        storage is SecureTokenStorage ? storage.readUserId() : null,
    store: ref.watch(scheduleStoreProvider.notifier),
  );
  ref.onDispose(controller.dispose);
  return controller;
});
