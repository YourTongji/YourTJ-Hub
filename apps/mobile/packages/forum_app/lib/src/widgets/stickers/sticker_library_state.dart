import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../providers.dart';

final stickerCollectionProvider = ChangeNotifierProvider<StickerCollection>((
  ref,
) {
  ref.watch(offlineCacheEpochProvider);
  return StickerCollection(
    StickerRepository(ref.watch(apiClientProvider)),
    ref.watch(stickerLibraryProvider),
  );
});

bool stickerApiUnsupported(Object? error) =>
    error is ApiException &&
    (error.statusCode == 404 || error.statusCode == 405);

/// In-memory recent use belongs to this account/site session. Membership/order
/// are server-owned; switching accounts disposes all pending UI results.
class StickerCollection extends ChangeNotifier {
  StickerCollection(this.repository, this.library);
  final StickerRepository repository;
  final StickerLibrary library;
  List<StickerItemPayload> official = [];
  List<StickerItemPayload> mine = [];
  final List<StickerItemPayload> recent = [];
  Object? officialError;
  Object? mineError;
  bool loadingOfficial = false;
  bool loadingMine = false;
  bool officialLoaded = false;
  bool mineLoaded = false;
  bool busy = false;
  bool _active = true;
  int _membershipRevision = 0;
  bool get active => _active;
  void _notify() {
    if (_active) notifyListeners();
  }

  Future<void> loadOfficial({bool refresh = false}) async {
    if (loadingOfficial || (officialLoaded && !refresh)) return;
    loadingOfficial = true;
    officialError = null;
    _notify();
    try {
      final items = await library.load(refresh: refresh);
      if (!_active) return;
      official = items;
      final enabledNames = items
          .where((item) => item.isEnabled)
          .map((item) => item.name)
          .toSet();
      recent.removeWhere(
        (item) => item.isOfficial && !enabledNames.contains(item.name),
      );
      officialLoaded = true;
    } catch (error) {
      if (_active) officialError = error;
    } finally {
      loadingOfficial = false;
      _notify();
    }
  }

  Future<void> loadMine({bool refresh = false}) async {
    if (loadingMine || (mineLoaded && !refresh)) return;
    final revision = _membershipRevision;
    var changedWhileLoading = false;
    loadingMine = true;
    mineError = null;
    _notify();
    try {
      final items = await repository.mine();
      if (!_active) return;
      if (revision != _membershipRevision) {
        changedWhileLoading = true;
        return;
      }
      mine = items;
      library.remember(items);
      final disabled = items
          .where((item) => !item.isEnabled)
          .map((item) => item.name)
          .toSet();
      recent.removeWhere((item) => disabled.contains(item.name));
      mineLoaded = true;
    } catch (error) {
      if (_active) mineError = error;
    } finally {
      loadingMine = false;
      _notify();
      // A save/removal during the request invalidates its membership snapshot.
      // Fetch the complete collection again after releasing the loading guard.
      if (_active && changedWhileLoading) await loadMine(refresh: true);
    }
  }

  void used(StickerItemPayload item) {
    if (!_active) return;
    recent.removeWhere((e) => e.name == item.name);
    recent.insert(0, item);
    if (recent.length > 30) recent.removeLast();
    _notify();
  }

  Future<void> _mutate(Future<void> Function() action) async {
    if (busy || !_active) return;
    busy = true;
    _notify();
    try {
      await action();
    } finally {
      busy = false;
      _notify();
    }
  }

  Future<void> save({
    String? stickerName,
    String? fileName,
    String? displayName,
  }) => _mutate(() async {
    final item = await repository.save(
      stickerName: stickerName,
      fileName: fileName,
      displayName: displayName,
    );
    if (!_active) return;
    _membershipRevision++;
    final index = mine.indexWhere((e) => e.name == item.name);
    mine = [...mine];
    if (index < 0) {
      mine.add(item);
    } else {
      mine[index] = item;
    }
    library.remember([item]);
    // Saving from a post before opening management must not mark the entire
    // collection loaded: it may contain other server-owned memberships.
  });

  Future<void> remove(Set<String> names) => _mutate(() async {
    // Reflect each acknowledged removal so a partial failure never lies about
    // membership and retry only needs the remaining selection.
    for (final name in names) {
      if (!_active) return;
      await repository.remove(name);
      if (!_active) return;
      _membershipRevision++;
      mine = mine.where((e) => e.name != name).toList();
      _notify();
    }
  });

  Future<void> reorder(int oldIndex, int newIndex) => _mutate(() async {
    final ordered = [...mine];
    ordered.insert(newIndex, ordered.removeAt(oldIndex));
    try {
      await repository.reorder(ordered.map((e) => e.name).toList());
    } catch (_) {
      // A concurrent collection edit changes membership. Refresh instead of
      // silently dropping it when replaying a stale full-order request.
      await loadMine(refresh: true);
      rethrow;
    }
    if (_active) {
      _membershipRevision++;
      mine = ordered;
    }
  });

  @override
  void dispose() {
    _active = false;
    super.dispose();
  }
}
