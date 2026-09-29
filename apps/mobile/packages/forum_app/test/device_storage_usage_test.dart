import 'dart:io';

import 'package:drift/native.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/storage/cache_coordinator.dart';
import 'package:forum_app/src/storage/media_repository.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:forum_app/src/storage/user_work_database.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.root);
  final String root;
  @override
  Future<String?> getApplicationSupportPath() async => root;
}

class _Chats extends ChatDraftStore {
  @override
  Future<({int count, int bytes})> usage() async => (count: 0, bytes: 0);
}

class _Widget extends ScheduleWidgetBridge {
  @override
  Future<int> usageBytes() async => 0;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform originalPaths;
  late AppDatabase cache;
  late MediaRepository media;
  late ProviderContainer container;

  setUp(() async {
    directory = await Directory.systemTemp.createTemp('yourtj-work-usage-');
    originalPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    cache = AppDatabase(NativeDatabase.memory());
    media = MediaRepository(directory: () async => directory);
    container = ProviderContainer(
      overrides: [
        offlineDatabaseProvider.overrideWithValue(cache),
        mediaRepositoryProvider.overrideWithValue(media),
        chatDraftStoreProvider.overrideWithValue(_Chats()),
        scheduleWidgetBridgeProvider.overrideWithValue(_Widget()),
        cacheCoordinatorProvider.overrideWithValue(
          CacheCoordinator(
            database: cache,
            owners: const {},
            databaseBytes: () async => 0,
            onInvalidated: (_) {},
          ),
        ),
      ],
    );
  });
  tearDown(() async {
    container.dispose();
    media.dispose();
    await cache.close();
    PathProviderPlatform.instance = originalPaths;
    await directory.delete(recursive: true);
  });

  test('usage includes recovery drafts when no active plans remain', () async {
    await UserWorkDatabase.instance.writeBatch('site:7', 'schedule', {
      'pk.plans': '[]',
      'pk.syncOwner': '7',
      'pk.planSync.v3.7': '{"drafts":{"recovered-a":{},"recovered-b":{}}}',
    });
    final usage = await container.read(deviceStorageProvider).usage();
    expect(usage.plans, 0);
    expect(usage.unsyncedPlans, 0);
    expect(usage.recoveryPlans, 2);
    expect(usage.legacyPlans, 0);
  });

  test('recovery drafts and legacy plans retain separate counts', () async {
    final work = UserWorkDatabase.instance;
    await work.writeBatch('site:7', 'schedule', {
      'pk.syncOwner': '7',
      'pk.planSync.v3.7': '{"drafts":{"recovered-a":{},"recovered-b":{}}}',
    });
    await work.writeBatch('legacy-unassigned', 'schedule-legacy', {
      'pk.plans': '[{"id":"legacy-plan"}]',
      'pk.planSync.v3.8': '{"drafts":{"quarantined-recovery":{}}}',
    });
    final usage = await container.read(deviceStorageProvider).usage();
    expect(usage.plans, 1);
    expect(usage.unsyncedPlans, 1);
    expect(usage.recoveryPlans, 3);
    expect(usage.legacyPlans, 1);
  });
}
