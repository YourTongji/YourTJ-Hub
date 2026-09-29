import '../offline/drift_cache.dart';
import 'cache_coordinator.dart';

class DeviceStorageUsage {
  const DeviceStorageUsage({
    required this.cacheBytes,
    required this.physicalCacheBytes,
    required this.workBytes,
    required this.drafts,
    required this.chatDrafts,
    required this.plans,
    required this.unsyncedPlans,
    this.recoveryPlans = 0,
    this.resetPending = false,
    this.pending = const {},
  });
  final Map<CacheCategory, int> cacheBytes;
  final int physicalCacheBytes,
      workBytes,
      drafts,
      chatDrafts,
      plans,
      unsyncedPlans,
      recoveryPlans;
  final Set<CacheCategory> pending;
  final bool resetPending;
}

abstract interface class DeviceStorage {
  Future<DeviceStorageUsage> usage();
  Future<CacheClearResult> clear(Set<CacheCategory> categories);
  Future<CacheClearResult> resume();
  Future<void> reset();
}
