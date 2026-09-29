import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../app_locale.dart';
import '../current_user.dart';
import '../messages/chat_drafts.dart';
import '../local/writing_store.dart';
import '../schedule/schedule_store.dart';
import '../offline/drift_cache.dart';
import '../pages/campus/campus_memory_cache.dart';
import '../providers.dart';
import '../push/push_service.dart';
import '../theme_mode.dart';
import '../site_theme.dart';
import 'cache_coordinator.dart';
import 'device_storage.dart';
import 'media_repository.dart';
import 'private_database.dart';
import 'user_work_database.dart';

final mediaRepositoryProvider = Provider<MediaRepository>((ref) {
  final repository = MediaRepository();
  ref.onDispose(repository.dispose);
  ref.listen(offlineCacheEpochProvider, (_, _) => repository.invalidate());
  return repository;
});

final Provider<CacheCoordinator> cacheCoordinatorProvider =
    Provider<CacheCoordinator>((ref) {
      final db = ref.watch(offlineDatabaseProvider);
      final media = ref.watch(mediaRepositoryProvider);
      final campus = ref.watch(campusSnapshotStoreProvider);
      final widget = ref.watch(scheduleWidgetBridgeProvider);
      return CacheCoordinator(
        database: db,
        databaseBytes: () => privateDatabaseBytes('cache'),
        onInvalidated: (categories) {
          ref.read(cacheClearEpochProvider.notifier).invalidate(categories);
          if (categories.contains(CacheCategory.forum)) {
            ref.invalidate(topicReturnStatesProvider);
            ref.invalidate(postReturnStatesProvider);
          }
          if (categories.contains(CacheCategory.campus)) {
            ref.read(campusCacheEpochProvider.notifier).invalidate();
            ref.read(campusMemoryCacheProvider).clear();
          }
        },
        owners: {
          CacheCategory.media: CacheOwner(
            invalidate: media.suspend,
            resume: media.resume,
            clear: media.clear,
            bytes: media.usageBytes,
          ),
          CacheCategory.campus: CacheOwner(
            invalidate: () {
              campus.invalidate();
              widget.invalidate();
            },
            clear: () async {
              Object? failure;
              try {
                await campus.clear();
              } catch (e) {
                failure = e;
              }
              try {
                await widget.clear();
              } catch (e) {
                failure ??= e;
              }
              if (failure != null) throw failure;
            },
            bytes: widget.usageBytes,
          ),
        },
      );
    });

final deviceStorageProvider = Provider<DeviceStorage>(
  (ref) => _DeviceStorage(ref),
);

/// A pending destructive reset blocks new local work until recovery finishes.
final storageResetStateProvider = StateProvider<AsyncValue<void>?>(
  (ref) => null,
);

/// A failed recovery remains visible in storage settings; suspended categories
/// cannot serve or refill until their durable deletion intent has completed.
final storageBootstrapProvider = FutureProvider<void>((ref) async {
  final service = ref.read(deviceStorageProvider);
  final db = ref.read(offlineDatabaseProvider);
  if (await db.operation('reset') != null) {
    await service.reset();
  } else {
    // Ordinary partial cleanup leaves only failed categories suspended. Its
    // retry remains available in settings; it must not block unrelated work.
    await service.resume();
    ref.read(storageResetStateProvider.notifier).state = null;
  }
});

class _DeviceStorage implements DeviceStorage {
  _DeviceStorage(this.ref);
  final Ref ref;
  bool _resetting = false;
  @override
  Future<DeviceStorageUsage> usage() async {
    final db = ref.read(offlineDatabaseProvider);
    final coordinator = ref.read(cacheCoordinatorProvider);
    final work = await ref.read(userWorkDatabaseProvider).usage();
    final chats = await ref.read(chatDraftStoreProvider).usage();
    final bytes = await db.cacheUsage();
    bytes[CacheCategory.media] = await ref
        .read(mediaRepositoryProvider)
        .usageBytes();
    bytes[CacheCategory.campus] =
        bytes[CacheCategory.campus]! +
        await ref.read(scheduleWidgetBridgeProvider).usageBytes();
    return DeviceStorageUsage(
      cacheBytes: bytes,
      physicalCacheBytes: await coordinator.totalBytes(),
      workBytes: await privateDatabaseBytes('user_work') + chats.bytes,
      drafts: work.draftCount,
      chatDrafts: chats.count,
      plans: work.planCount,
      unsyncedPlans: work.unsyncedPlanCount,
      recoveryPlans: work.legacyPlanCount,
      pending: await coordinator.pending(),
      resetPending: await db.operation('reset') != null,
    );
  }

  @override
  Future<CacheClearResult> clear(Set<CacheCategory> categories) =>
      ref.read(cacheCoordinatorProvider).clear(categories);
  @override
  Future<CacheClearResult> resume() =>
      ref.read(cacheCoordinatorProvider).resume();

  @override
  Future<void> reset() async {
    if (_resetting) throw StateError('Local reset already running');
    _resetting = true;
    ref.read(storageResetStateProvider.notifier).state =
        const AsyncValue.loading();
    final db = ref.read(offlineDatabaseProvider);
    try {
      // Write the intent before destroying any local work. The same idempotent
      // sequence resumes on launch if any independent owner fails or the app dies.
      await db.setOperation('reset', '1');
      ref.read(offlineCacheEpochProvider.notifier).invalidate();
      ref.invalidate(currentUserProvider);
      Object? failure;
      Future<void> attempt(Future<void> Function() action) async {
        try {
          await action();
        } catch (error) {
          failure ??= error;
        }
      }

      // Unbind push while the current credential is still available. Reset does
      // not erase the remote account or cloud drafts/plans.
      await attempt(
        () => ref.read(pushControllerProvider.notifier).handleLogout(),
      );
      await attempt(ref.read(tokenStorageProvider).clear);
      ref.invalidate(currentUserProvider);
      await attempt(() async {
        final result = await clear(CacheCategory.values.toSet());
        if (!result.succeeded) throw StateError('Cache reset incomplete');
      });
      await attempt(ref.read(chatDraftStoreProvider).clearAll);
      await attempt(ref.read(userWorkDatabaseProvider).clearAll);
      await attempt(ref.read(themeModeProvider.notifier).resetToDefault);
      await attempt(ref.read(appLocaleProvider.notifier).resetToDefault);
      await attempt(ref.read(siteThemeProvider.notifier).resetToDefault);
      // Durable owner tombstones prevent legacy records from being reimported.
      await attempt(() async {
        final prefs = await SharedPreferences.getInstance();
        if (!await prefs.clear()) throw StateError('Preferences reset failed');
        await prefs.reload();
        if (prefs.getKeys().isNotEmpty) {
          throw StateError('Preferences reset verification failed');
        }
      });
      await attempt(
        () => ref.read(scheduleWidgetBridgeProvider).setTransparency(9),
      );
      if (failure != null) throw failure!;
      ref.invalidate(writingStoreProvider);
      ref.invalidate(scheduleStoreProvider);
      ref.invalidate(appLocaleProvider);
      ref.invalidate(themeModeProvider);
      ref.invalidate(siteThemeProvider);
      await db.finishOperation('reset');
      ref.read(storageResetStateProvider.notifier).state = null;
    } catch (error, stack) {
      ref.read(storageResetStateProvider.notifier).state = AsyncValue.error(
        error,
        stack,
      );
      rethrow;
    } finally {
      _resetting = false;
    }
  }
}
