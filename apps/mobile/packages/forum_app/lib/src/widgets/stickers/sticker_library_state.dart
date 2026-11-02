import 'dart:async';

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

  /// Official pack names and covers; empty until loaded or on servers
  /// without pack metadata, where the picker falls back to pack keys.
  List<StickerOfficialPackPayload> officialPacks = [];
  List<StickerItemPayload> mine = [];
  final List<StickerItemPayload> recent = [];
  Object? officialError;
  Object? mineError;
  bool loadingOfficial = false;
  bool loadingMine = false;
  bool officialLoaded = false;
  bool mineLoaded = false;
  bool busy = false;

  /// Subscribed market packs (issue #1429), in the server's subscription order.
  List<StickerMarketPackPayload> packs = [];
  Object? packsError;
  bool loadingPacks = false;
  bool packsLoaded = false;
  final Map<int, List<StickerItemPayload>> packStickers = {};
  final Map<int, Object> packStickerErrors = {};
  final Set<int> loadingPackStickers = {};
  bool _active = true;
  int _membershipRevision = 0;
  int _packMembershipRevision = 0;
  bool get active => _active;
  void _notify() {
    if (_active) notifyListeners();
  }

  Future<void> loadOfficial({bool refresh = false}) async {
    if (loadingOfficial || (officialLoaded && !refresh)) return;
    loadingOfficial = true;
    officialError = null;
    _notify();
    // Pack metadata only names and orders the groups; it loads beside the
    // stickers and never holds them back.
    unawaited(_loadOfficialPacks());
    try {
      final items = await library.load(refresh: refresh);
      if (!_active) return;
      official = items;
      final enabledNames = items
          .where((item) => item.isAvailable)
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

  Future<void> _loadOfficialPacks() async {
    try {
      final packs = await repository.officialPacks();
      if (!_active) return;
      officialPacks = packs;
      _notify();
    } catch (_) {
      // Older servers lack the endpoint; the picker falls back to pack keys.
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
          .where((item) => !item.isAvailable)
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

  Future<void> loadPacks({bool refresh = false}) async {
    if (loadingPacks || (packsLoaded && !refresh)) return;
    final revision = _packMembershipRevision;
    var changedWhileLoading = false;
    loadingPacks = true;
    packsError = null;
    _notify();
    try {
      final items = await repository.myPacks();
      if (!_active) return;
      if (revision != _packMembershipRevision) {
        changedWhileLoading = true;
        return;
      }
      packs = items;
      packsLoaded = true;
    } catch (error) {
      if (_active) packsError = error;
    } finally {
      loadingPacks = false;
      _notify();
      if (_active && changedWhileLoading) await loadPacks(refresh: true);
    }
  }

  /// Stickers of one published pack; cached for the session and remembered by
  /// the resolver so inserted tokens render before the server echoes them.
  Future<void> loadPackStickers(int packId, {bool refresh = false}) async {
    if (loadingPackStickers.contains(packId) ||
        (packStickers.containsKey(packId) && !refresh)) {
      return;
    }
    loadingPackStickers.add(packId);
    packStickerErrors.remove(packId);
    _notify();
    try {
      final detail = await repository.marketPack(packId);
      if (!_active) return;
      packStickers[packId] = detail.stickers;
      library.remember(detail.stickers);
    } catch (error) {
      if (_active) packStickerErrors[packId] = error;
    } finally {
      loadingPackStickers.remove(packId);
      _notify();
    }
  }

  /// Official groups in display order: named packs first, then any pack key
  /// the metadata does not cover yet, so no enabled sticker becomes unreachable.
  List<StickerOfficialPackPayload> get officialGroups {
    final keys = <String>{
      for (final item in official)
        if (item.isAvailable && item.pack.isNotEmpty) item.pack,
    };
    return [
      for (final pack in officialPacks)
        if (keys.remove(pack.key)) pack,
      for (final key in keys) StickerOfficialPackPayload(key: key, name: key),
    ];
  }

  /// Official stickers grouped by [officialGroups], keeping each pack's order.
  List<StickerItemPayload> get officialInPackOrder {
    final order = {
      for (final (index, pack) in officialGroups.indexed) pack.key: index,
    };
    final buckets = List.generate(
      order.length + 1,
      (_) => <StickerItemPayload>[],
    );
    for (final item in official) {
      buckets[order[item.pack] ?? order.length].add(item);
    }
    return [for (final bucket in buckets) ...bucket];
  }

  bool hasPack(int packId) => packs.any((pack) => pack.id == packId);

  /// Subscription changes wait for the server: a pack never appears added
  /// before its stickers are usable on every device.
  Future<void> addPack(StickerMarketPackPayload pack) => _mutate(() async {
    final saved = await repository.savePack(pack.id);
    if (!_active) return;
    _packMembershipRevision++;
    if (hasPack(saved.id)) return;
    packs = [...packs, saved];
  });

  Future<void> removePack(StickerMarketPackPayload pack) => _mutate(() async {
    await repository.removePack(pack.id);
    if (!_active) return;
    _packMembershipRevision++;
    packs = packs.where((item) => item.id != pack.id).toList();
  });

  void used(StickerItemPayload item) {
    if (!_active || !item.isAvailable) return;
    recent.removeWhere((e) => e.name == item.name);
    recent.insert(0, item);
    if (recent.length > 30) recent.removeLast();
    _notify();
  }

  Future<void> _mutate(Future<void> Function() action) async {
    // Every successful completion must acknowledge an executed write. Callers
    // surface this failure and keep their draft/selection available for retry.
    if (!_active) throw StateError('Sticker collection is no longer active');
    if (busy) throw StateError('A sticker collection write is already pending');
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
