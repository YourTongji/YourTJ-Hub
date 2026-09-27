import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../providers.dart';
import '../../app_config.dart';
import '../../current_user.dart';
import '../../campus_widget/schedule_widget_bridge.dart';
import '../../campus_widget/schedule_widget_projection.dart';
import '../../offline/campus_snapshot_store.dart';
import 'campus_memory_cache.dart';

const campusTabKeys = <String, List<String>>{
  'today': ['profile', 'calendar', 'messages', 'today'],
  'timetable': ['calendar', 'timetable'],
  'academics': ['summary', 'grades', 'cet'],
  'messages': ['messages'],
  'calendars': ['calendar', 'terms'],
  'connection': [],
};

class CampusViewState {
  const CampusViewState({
    this.status,
    this.loading = true,
    this.busy = false,
    this.refreshing = false,
    this.error,
    this.data = const {},
    this.errors = const {},
    this.snapshot,
    this.fetching = const {},
  });
  final CampusStatus? status;
  final bool loading;
  final bool busy;
  final bool refreshing;
  final Object? error;
  final Map<String, CampusDataset> data;
  final Map<String, Object> errors;
  final CampusSnapshot? snapshot;
  final Set<String> fetching;
  bool get needsAuthorization =>
      status?.binding?.needsAuthorization == true ||
      errors.values.any(isCampusAuthorizationError);
}

bool isCampusAuthorizationError(Object? error) =>
    error is ApiException &&
    const [
      'campus.authorizationRequired',
      'campus.messageAuthorizationRequired',
    ].contains(error.messageCode);

/// Disposable private view state. Only selected datasets enter the bounded
/// foreground cache, after status verification and generation checks.
class CampusController extends StateNotifier<CampusViewState> {
  CampusController(
    this.repository, {
    CampusMemoryCache? cache,
    this.persistentStore,
    this.widgetBridge,
    this.scope,
  }) : cache = cache ?? CampusMemoryCache(),
       _ownsCache = cache == null,
       super(const CampusViewState());
  final CampusMemoryCache cache;
  final bool _ownsCache;
  final _receivedAt = <String, DateTime>{};
  bool _refreshing = false;
  final CampusRepository repository;
  final CampusSnapshotStore? persistentStore;
  final ScheduleWidgetBridge? widgetBridge;
  final CampusCacheScope? scope;
  CancelToken _cancel = CancelToken();
  int _generation = 0;
  final Set<String> _loading = {};
  String tab = 'today';
  CampusSnapshot? _snapshot;

  Future<CampusCalendarExport?> exportCalendar({
    bool applyAdjustments = true,
  }) async {
    final generation = _generation;
    try {
      final result = await repository.exportCalendar(
        applyAdjustments: applyAdjustments,
        cancelToken: _cancel,
      );
      return mounted && generation == _generation ? result : null;
    } catch (e) {
      if (!mounted || generation != _generation) return null;
      invalidateForError(e);
      rethrow;
    }
  }

  void _cancelReads() {
    ++_generation;
    _cancel.cancel();
    _cancel = CancelToken();
    _loading.clear();
  }

  void invalidateForError(Object error) {
    if (isCampusIdentityError(error)) _dropPrivateData(error);
  }

  void _dropPrivateData(Object error, {String key = 'status'}) {
    _cancelReads();
    cache.clear();
    _refreshing = false;
    _receivedAt.clear();
    _snapshot = null;
    unawaited(
      _clearPersistent(
        widgetState: isCampusAuthorizationError(error)
            ? 'authorizationRequired'
            : 'needsData',
      ),
    );
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: isCampusAuthorizationError(error) ? state.status : null,
      loading: false,
      error: error,
      errors: {key: error},
    );
  }

  Future<void> refresh({bool reuseCache = false}) async {
    if (_refreshing || state.busy) return;
    _cancelReads();
    final generation = _generation;
    final cacheFence = cache.generation;
    _refreshing = true;
    _expireTeachingDate();
    final previous = state;
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: previous.status,
      loading: previous.status == null,
      refreshing: true,
      data: previous.data,
      errors: previous.errors,
    );
    try {
      CampusSnapshot? persisted;
      var widgetCleared = false;
      if (reuseCache && persistentStore != null && scope != null) {
        persisted = await persistentStore!.read(scope!);
        if (!mounted || generation != _generation) return;
        _snapshot = persisted;
        if (persisted != null && previous.data.isEmpty) {
          state = CampusViewState(
            snapshot: _snapshot,
            fetching: Set.of(_loading),
            status: _offlineStatus(persisted.bindingRevision),
            loading: false,
            refreshing: true,
            data: Map.of(persisted.data),
          );
        }
      }
      final status = await repository.status(cancelToken: _cancel);
      if (!mounted ||
          generation != _generation ||
          !cache.isCurrent(cacheFence)) {
        return;
      }
      final binding =
          status.enabled && status.binding?.needsAuthorization == false
          ? status.binding
          : null;
      if (binding != null &&
          _snapshot != null &&
          _snapshot!.bindingRevision != binding.revision) {
        await _clearPersistent();
        if (!mounted || generation != _generation) return;
        widgetCleared = true;
        persisted = null;
        _snapshot = null;
      }
      final restored = cache.restore(binding?.revision);
      final sameIdentity =
          binding != null &&
          (previous.status?.binding?.revision == binding.revision ||
              persisted?.bindingRevision == binding.revision);
      if (!sameIdentity) _receivedAt.clear();
      final data = sameIdentity
          ? Map<String, CampusDataset>.of(previous.data)
          : <String, CampusDataset>{};
      if (reuseCache && binding != null) {
        if (persisted?.bindingRevision == binding.revision) {
          data.addAll(persisted!.data);
          for (final key in persisted.data.keys) {
            _receivedAt[key] = persisted.committedAt;
          }
        }
        for (final entry in restored.entries) {
          data[entry.key] = entry.value.data;
          _receivedAt[entry.key] = entry.value.receivedAt;
        }
      }
      state = CampusViewState(
        snapshot: _snapshot,
        fetching: Set.of(_loading),
        status: status,
        loading: false,
        refreshing: true,
        data: data,
      );
      _expireTeachingDate();
      if (binding == null) {
        _snapshot = null;
        await _clearPersistent(
          widgetState: status.binding?.needsAuthorization == true
              ? 'authorizationRequired'
              : 'unbound',
        );
      } else if (!reuseCache ||
          (persistentStore != null && scope != null && persisted == null)) {
        if (reuseCache && persisted == null && !widgetCleared) {
          await widgetBridge?.clear();
        }
        final keys = <String>{
          ...campusPersistentKeys,
          ...(campusTabKeys[tab] ?? const <String>[]),
        };
        final storeFence = persistentStore?.generation;
        final dataFence = cache.generation;
        await Future.wait(keys.map((key) => load(key, force: true)));
        if (!mounted ||
            generation != _generation ||
            !cache.isCurrent(dataFence)) {
          return;
        }
        await _commitPersistent(binding.revision, generation, storeFence);
      } else {
        if (persisted == null || tab != 'today') await loadTab(tab);
      }
    } catch (e) {
      if (mounted && generation == _generation) {
        if (isCampusIdentityError(e)) {
          _dropPrivateData(e);
        } else {
          state = CampusViewState(
            snapshot: _snapshot,
            fetching: Set.of(_loading),
            status: state.status,
            loading: false,
            error: e,
            data: state.data,
            errors: {...state.errors, 'status': e},
          );
        }
      }
    } finally {
      if (mounted && generation == _generation) {
        state = CampusViewState(
          snapshot: _snapshot,
          fetching: Set.of(_loading),
          status: state.status,
          loading: false,
          busy: state.busy,
          error: state.error,
          data: state.data,
          errors: state.errors,
        );
      }
      if (mounted && generation == _generation) _refreshing = false;
    }
  }

  void _expireTeachingDate() {
    final expired = _receivedAt.entries
        .where(
          (e) =>
              CampusMemoryCache.dailyKeys.contains(e.key) &&
              CampusMemoryCache.schoolDate(e.value) !=
                  CampusMemoryCache.schoolDate(cache.now()),
        )
        .map((e) => e.key)
        .toList();
    final day = state.data['today']?.teachingDay;
    if (day != null && day.date != CampusMemoryCache.schoolDate(cache.now())) {
      expired.addAll(['today', 'calendar']);
    }
    if (expired.isEmpty) return;
    final data = {...state.data};
    final errors = {...state.errors};
    for (final key in expired) {
      data.remove(key);
      errors.remove(key);
      _receivedAt.remove(key);
      cache.forget(key);
    }
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: state.status,
      loading: state.loading,
      busy: state.busy,
      refreshing: state.refreshing,
      error: state.error,
      data: data,
      errors: errors,
    );
  }

  /// A detail page can keep this controller alive while the overview is hidden.
  Future<void> enterTab(String value) async {
    tab = value;
    if (state.loading || _refreshing) {
      return; // Initial provider refresh already verifies status.
    }
    await refresh(reuseCache: true);
  }

  /// Called by the visible page clock. No background polling.
  Future<void> refreshVisible() async {
    if (_refreshing || state.busy) return;
    _expireTeachingDate();
  }

  Future<void> loadTab(String value, {bool force = false}) async {
    tab = value;
    if (state.status?.binding == null || state.needsAuthorization) return;
    _expireTeachingDate();
    await Future.wait(
      (campusTabKeys[value] ?? []).map((key) => load(key, force: force)),
    );
  }

  /// Persistent datasets form one atomic device/widget snapshot. Retrying one
  /// must complete the full refresh before publishing the replacement snapshot.
  Future<void> retry(String key) =>
      campusPersistentKeys.contains(key) ? refresh() : load(key, force: true);

  Future<void> load(String key, {bool force = false}) async {
    if (state.status?.binding == null || state.needsAuthorization) return;
    if (!force && _snapshot != null && campusPersistentKeys.contains(key)) {
      return;
    }
    final receivedAt = _receivedAt[key];
    if (!force &&
        const ['ready', 'empty'].contains(state.data[key]?.status) &&
        (!CampusMemoryCache.keys.contains(key) ||
            (receivedAt != null && cache.fresh(key, receivedAt)))) {
      return;
    }
    if (!_loading.add(key)) return;
    _notifyLoading();
    final generation = _generation;
    final cacheFence = cache.generation;
    final revision = state.status!.binding!.revision;
    final started = cache.now();
    try {
      final value = await repository.dataset(key, cancelToken: _cancel);
      if (!mounted ||
          generation != _generation ||
          !cache.isCurrent(cacheFence)) {
        return;
      }
      // A request crossing midnight must not reinsert yesterday's teaching data.
      if (CampusMemoryCache.dailyKeys.contains(key) &&
          CampusMemoryCache.schoolDate(started) !=
              CampusMemoryCache.schoolDate(cache.now())) {
        return;
      }
      _receivedAt[key] = started;
      cache.put(revision, value, started, cacheFence);
      state = CampusViewState(
        snapshot: _snapshot,
        fetching: Set.of(_loading),
        status: state.status,
        loading: false,
        busy: state.busy,
        refreshing: state.refreshing,
        error: state.error,
        data: {...state.data, key: value},
        errors: {...state.errors}..remove(key),
      );
    } catch (e) {
      if (!mounted ||
          generation != _generation ||
          !cache.isCurrent(cacheFence)) {
        return;
      }
      if (isCampusIdentityError(e)) {
        _dropPrivateData(e, key: key);
        return;
      }
      state = CampusViewState(
        snapshot: _snapshot,
        fetching: Set.of(_loading),
        status: state.status,
        loading: false,
        busy: state.busy,
        refreshing: state.refreshing,
        error: state.error,
        data: state.data,
        errors: {...state.errors, key: e},
      );
    } finally {
      if (mounted && generation == _generation) {
        _loading.remove(key);
        _notifyLoading();
      }
    }
  }

  Future<bool> change(
    Future<void> Function(CancelToken) action, {
    String widgetState = 'needsData',
  }) async {
    if (state.busy) return false;
    _cancelReads();
    _refreshing = false;
    final generation = _generation;
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: state.status,
      loading: false,
      busy: true,
      data: state.data,
      errors: state.errors,
    );
    try {
      await action(_cancel);
      if (!mounted || generation != _generation) return false;
      cache.clear();
      _receivedAt.clear();
      await _clearPersistent(widgetState: widgetState);
      if (!mounted || generation != _generation) return false;
      _snapshot = null;
      state = const CampusViewState();
      await refresh();
      return mounted && state.error == null;
    } catch (e) {
      if (mounted && generation == _generation) {
        if (isCampusIdentityError(e)) {
          _dropPrivateData(e);
          return false;
        }
        state = CampusViewState(
          snapshot: _snapshot,
          fetching: Set.of(_loading),
          status: state.status,
          loading: false,
          error: e,
          data: state.data,
          errors: state.errors,
        );
      }
      return false;
    }
  }

  @override
  void dispose() {
    ++_generation;
    _cancel.cancel();
    if (_ownsCache) cache.dispose();
    super.dispose();
  }

  Future<void> _commitPersistent(
    String bindingRevision,
    int generation,
    int? storeFence,
  ) async {
    if (persistentStore == null || scope == null) return;
    bool current() =>
        mounted &&
        generation == _generation &&
        storeFence == persistentStore!.generation;
    if (!current() ||
        campusPersistentKeys.any(
          (key) => state.errors.containsKey(key) || state.data[key] == null,
        )) {
      return;
    }
    final CampusSnapshot snapshot;
    try {
      snapshot = await persistentStore!.write(
        scope!,
        bindingRevision,
        state.data,
        committedAt: cache.now(),
        expectedGeneration: storeFence,
      );
    } on CampusSnapshotSuperseded {
      return;
    } catch (error) {
      if (current()) {
        state = CampusViewState(
          snapshot: _snapshot,
          fetching: Set.of(_loading),
          status: state.status,
          loading: false,
          refreshing: state.refreshing,
          data: state.data,
          errors: {...state.errors, 'snapshot': error},
        );
      }
      return;
    }
    if (!current()) return;
    _snapshot = snapshot;
    _notifyLoading();
    if (widgetBridge == null) return;
    CampusCalendarRules? rules;
    try {
      rules = (await repository.calendarRules(cancelToken: _cancel)).rules;
    } catch (_) {
      // Keep the authoritative snapshot and publish unknown future days.
    }
    if (!current()) return;
    try {
      await widgetBridge!.write(
        ScheduleWidgetProjection.fromSnapshot(
          snapshot,
          scope!,
          calendarRules: rules,
        ),
      );
    } catch (_) {
      // The campus snapshot remains readable if the platform bridge fails.
    }
  }

  void _notifyLoading() {
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: state.status,
      loading: state.loading,
      busy: state.busy,
      refreshing: state.refreshing,
      error: state.error,
      data: state.data,
      errors: state.errors,
    );
  }

  /// Called before device-cache deletion, so pending reads cannot refill it.
  void discardLocal() {
    _cancelReads();
    cache.clear();
    _snapshot = null;
    _receivedAt.clear();
    _refreshing = false;
    state = CampusViewState(
      snapshot: _snapshot,
      fetching: Set.of(_loading),
      status: state.status,
      loading: false,
    );
  }

  Future<void> _clearPersistent({String widgetState = 'needsData'}) async {
    if (persistentStore != null && scope != null) {
      await persistentStore!.clearScope(scope!);
    }
    await widgetBridge?.clear(state: widgetState);
  }

  CampusStatus _offlineStatus(String bindingRevision) => CampusStatus(
    enabled: true,
    candidate: null,
    binding: CampusBinding(
      maskedId: '',
      boundAt: '',
      revision: bindingRevision,
      needsAuthorization: false,
    ),
  );
}

final campusControllerProvider =
    StateNotifierProvider.autoDispose<CampusController, CampusViewState>((ref) {
      ref.watch(offlineCacheEpochProvider);
      final controller = CampusController(
        ref.watch(campusRepositoryProvider),
        cache: ref.watch(campusMemoryCacheProvider),
        persistentStore: ref.watch(campusSnapshotStoreProvider),
        widgetBridge: ref.watch(scheduleWidgetBridgeProvider),
        scope: switch (ref.watch(currentUserProvider).valueOrNull?.id) {
          final int accountId => CampusCacheScope(
            site: Uri.parse(
              AppConfig.apiBaseUrl.isNotEmpty
                  ? AppConfig.apiBaseUrl
                  : GfApiClient.defaultBaseUrl,
            ).origin,
            accountId: accountId,
          ),
          _ => null,
        },
      )..refresh(reuseCache: true);
      ref.listen(campusCacheEpochProvider, (_, _) => controller.discardLocal());
      return controller;
    });

// A 403 can reject an operation (CSRF/email verification) without invalidating
// the session or school binding. Only explicit identity failures clear all data.
bool isCampusIdentityError(Object? error) =>
    isCampusAuthorizationError(error) ||
    (error is ApiException &&
        (error.statusCode == 401 ||
            const [
              'campus.connectionChanged',
              'campus.disabled',
              'permission.resolveFailed',
              'permission.userFrozen',
            ].contains(error.messageCode)));
