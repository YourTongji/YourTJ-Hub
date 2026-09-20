import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../providers.dart';
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
  });
  final CampusStatus? status;
  final bool loading;
  final bool busy;
  final bool refreshing;
  final Object? error;
  final Map<String, CampusDataset> data;
  final Map<String, Object> errors;
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
  CampusController(this.repository, {CampusMemoryCache? cache})
    : cache = cache ?? CampusMemoryCache(),
      _ownsCache = cache == null,
      super(const CampusViewState());
  final CampusMemoryCache cache;
  final bool _ownsCache;
  final _receivedAt = <String, DateTime>{};
  bool _refreshing = false;
  final CampusRepository repository;
  CancelToken _cancel = CancelToken();
  int _generation = 0;
  final Set<String> _loading = {};
  String tab = 'today';

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
    if (_invalidatesIdentity(error)) _dropPrivateData(error);
  }

  void _dropPrivateData(Object error, {String key = 'status'}) {
    _cancelReads();
    cache.clear();
    _refreshing = false;
    _receivedAt.clear();
    state = CampusViewState(
      status: isCampusAuthorizationError(error) ? state.status : null,
      loading: false,
      error: error,
      errors: {key: error},
    );
  }

  Future<void> refresh({bool reuseCache = false}) async {
    _cancelReads();
    final generation = _generation;
    final cacheFence = cache.generation;
    _refreshing = true;
    _expireTeachingDate();
    final previous = state;
    state = CampusViewState(
      status: previous.status,
      loading: previous.status == null,
      refreshing: true,
      data: previous.data,
      errors: previous.errors,
    );
    try {
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
      final restored = cache.restore(binding?.revision);
      final sameIdentity =
          binding != null &&
          previous.status?.binding?.revision == binding.revision;
      if (!sameIdentity) _receivedAt.clear();
      final data = sameIdentity
          ? Map<String, CampusDataset>.of(previous.data)
          : <String, CampusDataset>{};
      if (reuseCache && binding != null) {
        for (final entry in restored.entries) {
          data[entry.key] = entry.value.data;
          _receivedAt[entry.key] = entry.value.receivedAt;
        }
      }
      state = CampusViewState(
        status: status,
        loading: false,
        refreshing: true,
        data: data,
      );
      if (binding != null) await loadTab(tab, force: !reuseCache);
    } catch (e) {
      if (mounted && generation == _generation) _dropPrivateData(e);
    } finally {
      if (mounted && generation == _generation) {
        state = CampusViewState(
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
    await loadTab(tab);
  }

  Future<void> loadTab(String value, {bool force = false}) async {
    tab = value;
    if (state.status?.binding == null || state.needsAuthorization) return;
    _expireTeachingDate();
    await Future.wait(
      (campusTabKeys[value] ?? []).map((key) => load(key, force: force)),
    );
  }

  Future<void> load(String key, {bool force = false}) async {
    if (state.status?.binding == null || state.needsAuthorization) return;
    final receivedAt = _receivedAt[key];
    if (!force &&
        const ['ready', 'empty'].contains(state.data[key]?.status) &&
        (!CampusMemoryCache.keys.contains(key) ||
            (receivedAt != null && cache.fresh(key, receivedAt)))) {
      return;
    }
    if (!_loading.add(key)) return;
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
      if (_invalidatesIdentity(e)) {
        _dropPrivateData(e, key: key);
        return;
      }
      cache.forget(key);
      _receivedAt.remove(key);
      state = CampusViewState(
        status: state.status,
        loading: false,
        busy: state.busy,
        refreshing: state.refreshing,
        error: state.error,
        data: {...state.data}..remove(key),
        errors: {...state.errors, key: e},
      );
    } finally {
      if (generation == _generation) _loading.remove(key);
    }
  }

  Future<bool> change(Future<void> Function(CancelToken) action) async {
    if (state.busy) return false;
    final generation = _generation;
    state = CampusViewState(
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
      state = const CampusViewState();
      await refresh();
      return mounted && state.error == null;
    } catch (e) {
      if (mounted && generation == _generation) {
        if (_invalidatesIdentity(e)) {
          _dropPrivateData(e);
          return false;
        }
        state = CampusViewState(
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
}

final campusControllerProvider =
    StateNotifierProvider.autoDispose<CampusController, CampusViewState>((ref) {
      ref.watch(offlineCacheEpochProvider);
      return CampusController(
        ref.watch(campusRepositoryProvider),
        cache: ref.watch(campusMemoryCacheProvider),
      )..refresh(reuseCache: true);
    });

// A 403 can reject an operation (CSRF/email verification) without invalidating
// the session or school binding. Only explicit identity failures clear all data.
bool _invalidatesIdentity(Object error) =>
    isCampusAuthorizationError(error) ||
    (error is ApiException &&
        (error.statusCode == 401 ||
            const [
              'campus.connectionChanged',
              'campus.disabled',
              'permission.resolveFailed',
              'permission.userFrozen',
            ].contains(error.messageCode)));
