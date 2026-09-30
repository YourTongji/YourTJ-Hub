import 'dart:io';

import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/storage/storage_providers.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.root);
  final String root;
  @override
  Future<String?> getApplicationSupportPath() async => '$root/support';
  @override
  Future<String?> getApplicationDocumentsPath() async => '$root/documents';
  @override
  Future<String?> getApplicationCachePath() async => '$root/cache';
  @override
  Future<String?> getTemporaryPath() async => '$root/temporary';
}

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  Future<void> withStorage(Future<void> Function(ProviderContainer) run) async {
    final root = await Directory.systemTemp.createTemp('session-native-store-');
    final paths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(root.path);
    FlutterSecureStorage.setMockInitialValues({});
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(_Tokens()),
        currentUserProvider.overrideWith((ref) async => null),
      ],
    );
    final db = container.read(offlineDatabaseProvider);
    try {
      await db.customStatement(
        "INSERT INTO cache_entries VALUES ('s','chat','private','{}',0,0,2,1)",
      );
      expect(
        (await db.customSelect('PRAGMA journal_mode').getSingle())
            .data
            .values
            .single,
        'wal',
      );
      await run(container);
    } finally {
      await db.close();
      container.dispose();
      PathProviderPlatform.instance = paths;
      await root.delete(recursive: true);
    }
  }

  test(
    'compaction drains WAL checkpoint before vacuuming',
    () => withStorage((container) async {
      await container.read(offlineDatabaseProvider).compact();
    }),
  );
  test(
    'production session cleanup succeeds with an encrypted WAL file',
    () => withStorage((container) async {
      final db = container.read(offlineDatabaseProvider);
      final oldTopicView = captureTopicCache(
        container.read(offlineTopicCacheProvider),
      );
      final oldChatView = captureChatCache(
        container.read(offlineChatCacheProvider),
      );
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      // Login captures views before awaiting token storage and advancing the
      // session boundary. The old views must still coordinate every owner.
      await clearOfflineCache(
        oldTopicView,
        oldChatView,
        container.read(scheduleWidgetBridgeProvider),
      );
      for (var attempt = 0; attempt < 2; attempt++) {
        await clearOfflineCache(
          container.read(offlineTopicCacheProvider),
          container.read(offlineChatCacheProvider),
          container.read(scheduleWidgetBridgeProvider),
        );
      }
      expect(
        await db.customSelect('SELECT * FROM cache_entries').get(),
        isEmpty,
      );
      expect(await container.read(cacheCoordinatorProvider).pending(), isEmpty);
    }),
  );
  test(
    'real provider cleanup retains failed media intent and recovers on retry',
    () => withStorage((container) async {
      final diagnostics = <String>[];
      final previousPrint = debugPrint;
      debugPrint = (message, {wrapWidth}) => diagnostics.add(message ?? '');
      addTearDown(() => debugPrint = previousPrint);
      final cachePath = await PathProviderPlatform.instance
          .getApplicationCachePath();
      final obstruction = File('$cachePath/yourtj_media_v1');
      await obstruction.parent.create(recursive: true);
      await obstruction.writeAsString('directory unavailable');
      final db = container.read(offlineDatabaseProvider);
      final coordinator = container.read(cacheCoordinatorProvider);
      Future<void> clear() => clearOfflineCache(
        container.read(offlineTopicCacheProvider),
        container.read(offlineChatCacheProvider),
        container.read(scheduleWidgetBridgeProvider),
      );

      await expectLater(clear(), throwsStateError);
      expect(
        diagnostics.join('\n'),
        contains('Cache cleanup failed [media]: FileSystemException'),
      );
      expect(diagnostics.join('\n'), isNot(contains('directory unavailable')));
      expect(await coordinator.pending(), {CacheCategory.media});
      expect(db.available(CacheCategory.media), isFalse);
      expect(container.read(mediaRepositoryProvider).isSuspended, isTrue);

      await obstruction.delete();
      await clear();
      expect(await coordinator.pending(), isEmpty);
      expect(db.available(CacheCategory.media), isTrue);
      expect(container.read(mediaRepositoryProvider).isSuspended, isFalse);
    }),
  );
}
