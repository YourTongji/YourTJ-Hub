import 'dart:convert';

import '../offline/drift_cache.dart';

/// A storage owner exposes lifecycle operations, not its business data.
class CacheOwner {
  const CacheOwner({
    required this.invalidate,
    required this.clear,
    required this.bytes,
    this.resume,
  });
  final void Function() invalidate;
  final void Function()? resume;
  final Future<void> Function() clear;
  final Future<int> Function() bytes;
}

class CacheClearResult {
  const CacheClearResult(this.failed, this.reclaimedBytes);
  final Set<CacheCategory> failed;
  final int reclaimedBytes;
  bool get succeeded => failed.isEmpty;
}

/// Coordinates independently-owned stores with a durable intent. It does not
/// own drafts, network requests, permissions or domain merge semantics.
class CacheCoordinator {
  CacheCoordinator({
    required this.database,
    required this.owners,
    required this.databaseBytes,
    required this.onInvalidated,
  });
  final AppDatabase database;
  final Map<CacheCategory, CacheOwner> owners;
  final Future<int> Function() databaseBytes;
  final void Function(Set<CacheCategory>) onInvalidated;
  Future<CacheClearResult>? _running;
  // Includes database-only categories so an unsuccessful initial journal write
  // can still be retried in this process. An owner receives one balanced hold,
  // regardless of how many deletion attempts fail.
  final Set<CacheCategory> _heldOwners = {};

  Future<int> totalBytes() async {
    var result = await databaseBytes();
    for (final owner in owners.values) {
      result += await owner.bytes();
    }
    return result;
  }

  Future<Set<CacheCategory>> pending() async {
    final raw = await database.operation('clear');
    if (raw == null) return {};
    try {
      final names = (jsonDecode(raw) as List).cast<String>();
      return CacheCategory.values.where((c) => names.contains(c.name)).toSet();
    } catch (_) {
      // An unreadable deletion intent must not expose residual private data.
      return CacheCategory.values.toSet();
    }
  }

  Future<CacheClearResult> resume() async {
    final categories = {...await pending(), ..._heldOwners};
    return categories.isEmpty
        ? const CacheClearResult({}, 0)
        : clear(categories);
  }

  Future<CacheClearResult> clear(Set<CacheCategory> categories) {
    if (_running != null) {
      return Future.error(StateError('Storage cleanup already running'));
    }
    if (categories.isEmpty) return Future.value(const CacheClearResult({}, 0));
    final selected = Set<CacheCategory>.unmodifiable(categories);
    // Fence every owner synchronously, including downloads already in flight.
    _hold(selected);
    final operation = _clear(selected);
    _running = operation;
    operation.then<void>(
      (_) => _running = null,
      onError: (Object _, StackTrace _) {
        _running = null;
      },
    );
    return operation;
  }

  void _hold(Set<CacheCategory> categories) {
    if (categories.isEmpty) return;
    database.suspend(categories);
    for (final category in categories) {
      if (_heldOwners.add(category)) owners[category]?.invalidate();
    }
    onInvalidated(categories);
  }

  void _release(Set<CacheCategory> categories) {
    for (final category in categories) {
      if (_heldOwners.contains(category)) {
        owners[category]?.resume?.call();
        _heldOwners.remove(category);
      }
    }
    database.release(categories);
  }

  Future<CacheClearResult> _clear(Set<CacheCategory> selected) async {
    final all = {...await pending(), ..._heldOwners, ...selected};
    // A previous process may have journaled categories not selected by this
    // action. Fence their external owners before any awaited cleanup starts.
    _hold(all.difference(selected));
    await database.setOperation(
      'clear',
      jsonEncode(all.map((c) => c.name).toList()),
    );
    var before = 0;
    try {
      before = await totalBytes();
    } catch (_) {
      /* Accounting cannot prevent deletion. */
    }
    final failed = <CacheCategory>{};
    try {
      await database.clearCategories(all);
    } catch (_) {
      failed.addAll(all);
    }
    for (final category in all) {
      try {
        await owners[category]?.clear();
      } catch (_) {
        failed.add(category);
      }
    }
    try {
      await database.compact();
    } catch (_) {
      failed.addAll(all.difference({CacheCategory.media}));
    }
    if (failed.isEmpty) {
      await database.finishOperation('clear');
      _release(all);
    } else {
      await database.setOperation(
        'clear',
        jsonEncode(failed.map((c) => c.name).toList()),
      );
      _release(all.difference(failed));
    }
    var after = before;
    try {
      after = await totalBytes();
    } catch (_) {
      /* No fabricated reclaimed size. */
    }
    return CacheClearResult(
      Set.unmodifiable(failed),
      (before - after).clamp(0, before),
    );
  }
}
