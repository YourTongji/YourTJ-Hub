import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../providers.dart';

const campusTabKeys = <String, List<String>>{
  'today': ['profile', 'calendar', 'messages', 'timetable'],
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
    this.error,
    this.data = const {},
    this.errors = const {},
  });
  final CampusStatus? status;
  final bool loading;
  final bool busy;
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

/// Page-lifetime memory only. Cancellation plus a generation fence prevents a
/// refresh, identity replacement or session change from accepting late responses.
class CampusController extends StateNotifier<CampusViewState> {
  CampusController(this.repository) : super(const CampusViewState());
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
      if (isCampusAuthorizationError(e)) {
        state = CampusViewState(
          status: state.status,
          loading: state.loading,
          busy: state.busy,
          error: state.error,
          data: state.data,
          errors: {...state.errors, 'calendar-export': e},
        );
      }
      rethrow;
    }
  }

  Future<void> refresh() async {
    final generation = ++_generation;
    _cancel.cancel();
    _cancel = CancelToken();
    _loading.clear();
    state = const CampusViewState();
    try {
      final status = await repository.status(cancelToken: _cancel);
      if (!mounted || generation != _generation) return;
      state = CampusViewState(status: status, loading: false);
      if (status.binding != null && !status.binding!.needsAuthorization) {
        await loadTab(tab);
      }
    } catch (e) {
      if (mounted && generation == _generation) {
        state = CampusViewState(loading: false, error: e);
      }
    }
  }

  Future<void> loadTab(String value) async {
    tab = value;
    if (state.status?.binding == null || state.needsAuthorization) return;
    await Future.wait((campusTabKeys[value] ?? []).map(load));
  }

  Future<void> load(String key) async {
    if (state.data.containsKey(key) || !_loading.add(key)) return;
    final generation = _generation;
    try {
      final value = await repository.dataset(key, cancelToken: _cancel);
      if (!mounted || generation != _generation) return;
      state = CampusViewState(
        status: state.status,
        loading: false,
        busy: state.busy,
        error: state.error,
        data: {...state.data, key: value},
        errors: {...state.errors}..remove(key),
      );
    } catch (e) {
      if (!mounted || generation != _generation) return;
      state = CampusViewState(
        status: state.status,
        loading: false,
        busy: state.busy,
        error: state.error,
        data: state.data,
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
      await refresh();
      return mounted && state.error == null;
    } catch (e) {
      if (mounted && generation == _generation) {
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
    super.dispose();
  }
}

final campusControllerProvider =
    StateNotifierProvider.autoDispose<CampusController, CampusViewState>((ref) {
      ref.watch(offlineCacheEpochProvider);
      return CampusController(ref.watch(campusRepositoryProvider))..refresh();
    });
