import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers.dart';

class UnreadDisplayState {
  const UnreadDisplayState({
    this.style = UnreadBadgeStyle.number,
    this.loaded = false,
    this.saving = false,
    this.error,
  });
  final UnreadBadgeStyle style;
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
  }) : super(const UnreadDisplayState());
  final Future<NotificationPreferences> Function() loadPreferences;
  final Future<NotificationPreferences> Function(UnreadBadgeStyle)
  savePreferences;
  final bool Function() isCurrent;
  int _revision = 0;
  bool _disposed = false;
  Future<bool>? _loading;
  int get revision => _revision;
  bool get _current => !_disposed && isCurrent();

  void acceptRemote(UnreadBadgeStyle style, {required int expectedRevision}) {
    if (!_current || state.saving || expectedRevision != _revision) return;
    _revision++;
    state = UnreadDisplayState(style: style, loaded: true);
  }

  Future<bool> load() {
    if (!_current) return Future.value(false);
    if (state.loaded) return Future.value(true);
    if (_loading != null) return _loading!;
    final captured = _revision;
    final operation = () async {
      try {
        final result = await loadPreferences();
        acceptRemote(result.badgeStyle, expectedRevision: captured);
        return _current && state.loaded;
      } catch (error) {
        if (_current && captured == _revision) {
          state = UnreadDisplayState(style: state.style, error: error);
        }
        return false;
      }
    }();
    _loading = operation.whenComplete(() => _loading = null);
    return _loading!;
  }

  Future<bool> update(UnreadBadgeStyle style) async {
    if (!_current || state.saving || !state.loaded) return false;
    if (style == state.style) return true;
    final previous = state.style;
    final captured = ++_revision;
    state = UnreadDisplayState(style: style, loaded: true, saving: true);
    try {
      final result = await savePreferences(style);
      if (!_current || captured != _revision) return false;
      _revision++;
      state = UnreadDisplayState(style: result.badgeStyle, loaded: true);
      return true;
    } catch (error) {
      if (_current && captured == _revision) {
        _revision++;
        state = UnreadDisplayState(style: previous, loaded: true, error: error);
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
        isCurrent: () => session.isCurrent(epoch),
      );
    });
