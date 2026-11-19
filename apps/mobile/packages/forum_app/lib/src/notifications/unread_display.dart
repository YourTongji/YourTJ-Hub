import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';

class UnreadDisplayState {
  const UnreadDisplayState({
    this.style = UnreadBadgeStyle.number,
    this.categories = notificationBadgeCategories,
    this.loaded = false,
    this.saving = false,
    this.error,
  });
  final UnreadBadgeStyle style;
  final List<String> categories;
  final bool loaded, saving;
  final Object? error;
}

/// Account-scoped presentation. Read completions cannot undo a later selection;
/// unread counts and delivery permissions remain owned by their existing flows.
class UnreadDisplayController extends StateNotifier<UnreadDisplayState> {
  UnreadDisplayController({
    required this.loadPreferences,
    required this.savePreferences,
    required this.isCurrent,
    this.saveCategories,
  }) : super(const UnreadDisplayState());
  final Future<NotificationPreferences> Function() loadPreferences;
  final Future<NotificationPreferences> Function(UnreadBadgeStyle)
  savePreferences;
  final Future<NotificationPreferences> Function(List<String>)? saveCategories;
  final bool Function() isCurrent;
  int _revision = 0;
  bool _disposed = false;
  Future<bool>? _loading;
  int get revision => _revision;
  bool get _current => !_disposed && isCurrent();

  void acceptRemote(
    UnreadBadgeStyle style, {
    required int expectedRevision,
    List<String>? categories,
  }) {
    if (!_current || state.saving || expectedRevision != _revision) return;
    _revision++;
    state = UnreadDisplayState(
      style: style,
      categories: categories ?? state.categories,
      loaded: true,
    );
  }

  Future<bool> load() {
    if (!_current) return Future.value(false);
    if (state.loaded) return Future.value(true);
    if (_loading != null) return _loading!;
    final captured = _revision;
    final operation = () async {
      try {
        final result = await loadPreferences();
        acceptRemote(
          result.badgeStyle,
          categories: result.badgeCategories,
          expectedRevision: captured,
        );
        return _current && state.loaded;
      } catch (error) {
        if (_current && captured == _revision) {
          state = UnreadDisplayState(
            style: state.style,
            categories: state.categories,
            loaded: state.loaded,
            error: error,
          );
        }
        return false;
      }
    }();
    _loading = operation.whenComplete(() => _loading = null);
    return _loading!;
  }

  Future<bool> update(UnreadBadgeStyle style) => _update(style: style);
  Future<bool> updateCategories(List<String> categories) =>
      _update(categories: List.unmodifiable(categories));

  Future<bool> _update({
    UnreadBadgeStyle? style,
    List<String>? categories,
  }) async {
    if (!_current || state.saving || !state.loaded) return false;
    if (categories == null && style == state.style) return true;
    final previous = state;
    final captured = ++_revision;
    state = UnreadDisplayState(
      style: style ?? previous.style,
      categories: categories ?? previous.categories,
      loaded: true,
      saving: true,
    );
    try {
      final result = categories == null
          ? await savePreferences(style!)
          : await saveCategories!(categories);
      if (!_current || captured != _revision) return false;
      _revision++;
      state = UnreadDisplayState(
        style: result.badgeStyle,
        categories: result.badgeCategories,
        loaded: true,
      );
      return true;
    } catch (error) {
      if (_current && captured == _revision) {
        _revision++;
        state = UnreadDisplayState(
          style: previous.style,
          categories: previous.categories,
          loaded: true,
          error: error,
        );
      }
      return false;
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}

final unreadDisplayProvider =
    StateNotifierProvider<UnreadDisplayController, UnreadDisplayState>((ref) {
      final epoch = ref.watch(offlineCacheEpochProvider);
      final session = ref.read(offlineCacheEpochProvider.notifier);
      final repository = ref.watch(notificationRepositoryProvider);
      Future<NotificationRepository> owned() async {
        final bound = await repository.forSession();
        if (!session.isCurrent(epoch)) throw StateError('Session changed');
        return bound;
      }

      return UnreadDisplayController(
        loadPreferences: () async => (await owned()).preferences(),
        savePreferences: (style) async =>
            (await owned()).updatePreferences(style),
        saveCategories: (categories) async =>
            (await owned()).updateBadgeCategories(categories),
        isCurrent: () => session.isCurrent(epoch),
      );
    });
