import 'dart:io';

import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:forum_app/src/app_locale.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/messages/chat_drafts.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/site_theme.dart';
import 'package:forum_app/src/storage/cache_coordinator.dart';
import 'package:forum_app/src/storage/device_storage.dart';
import 'package:forum_app/src/storage/private_database.dart';
import 'package:forum_app/src/storage/reset_journal.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'package:forum_app/src/theme_mode.dart';

class _Paths extends PathProviderPlatform {
  _Paths(this.root);
  final String root;
  @override
  Future<String?> getApplicationSupportPath() async => root;
}

class _Tokens implements TokenStorage {
  String? value = 'fixture-session';
  int clears = 0;
  @override
  Future<String?> read() async => value;
  @override
  Future<void> write(String token) async => value = token;
  @override
  Future<void> clear() async {
    clears++;
    value = null;
  }
}

class _Push extends PushController {
  _Push(this.tokens);
  final _Tokens tokens;
  final seenSessions = <String?>[];
  @override
  PushChannelStatus build() => PushChannelStatus.disabled;
  @override
  Future<void> handleLogout() async {
    seenSessions.add(await tokens.read());
  }
}

class _Chats extends ChatDraftStore {
  int count = 1;
  int clears = 0;
  bool failNextClear = false;
  @override
  Future<({int count, int bytes})> usage() async =>
      (count: count, bytes: count * 100);
  @override
  Future<void> clearAll() async {
    clears++;
    if (failNextClear) {
      failNextClear = false;
      throw StateError('Fixture secure storage temporarily unavailable');
    }
    count = 0;
  }
}

class _WidgetBridge extends ScheduleWidgetBridge {
  int clears = 0;
  int? transparency;
  @override
  Future<void> clear({String state = 'needsData'}) async {
    invalidate();
    clears++;
  }

  @override
  Future<int> usageBytes() async => 0;
  @override
  Future<void> setTransparency(int value) async => transparency = value;
}

class _Theme extends ThemeModeNotifier {
  @override
  ThemeMode build() => ThemeMode.system;
  @override
  Future<void> resetToDefault() async => state = ThemeMode.system;
}

class _Locale extends AppLocaleNotifier {
  @override
  Locale? build() => null;
  @override
  Future<void> resetToDefault() async => state = null;
}

class _SiteTheme extends SiteThemeController {
  @override
  SiteThemeState build() =>
      const SiteThemeState(available: false, following: true);
  @override
  Future<void> resetToDefault() async => state = build();
}

class _MediaOwner {
  bool suspended = false;
  bool hasData = true;
  bool failNextClear = false;
  int clears = 0;
  CacheOwner get owner => CacheOwner(
    invalidate: () => suspended = true,
    resume: () => suspended = false,
    bytes: () async => hasData ? 4096 : 0,
    clear: () async {
      clears++;
      if (failNextClear) {
        failNextClear = false;
        throw StateError('Fixture media filesystem unavailable');
      }
      hasData = false;
    },
  );
}

/// The production DeviceStorage and CacheCoordinator run against real SQLite.
/// Only OS/network owners are replaced; no DeviceStorage implementation is faked.
class _Fixture {
  _Fixture(this.database, this.work, {_Chats? chatOwner})
    : chats = chatOwner ?? _Chats() {
    coordinator = CacheCoordinator(
      database: database,
      databaseBytes: () async => 0,
      onInvalidated: invalidated.add,
      owners: {
        CacheCategory.media: media.owner,
        CacheCategory.campus: CacheOwner(
          invalidate: widget.invalidate,
          clear: widget.clear,
          bytes: widget.usageBytes,
        ),
      },
    );
    reopenService();
    cache = DriftOfflineCache(
      database,
      resolveScope: () async => CacheScope(site, 7),
    );
  }
  static const site = 'https://dev.example';
  static final scope = writingScope(site, 7);
  final AppDatabase database;
  final UserWorkDatabase work;
  final tokens = _Tokens();
  final _Chats chats;
  final widget = _WidgetBridge();
  final media = _MediaOwner();
  final invalidated = <Set<CacheCategory>>[];
  late _Push push;
  late final CacheCoordinator coordinator;
  late final DriftOfflineCache cache;
  ProviderContainer? _container;
  ProviderContainer get container => _container!;
  DeviceStorage get service => container.read(deviceStorageProvider);

  void reopenService() {
    _container?.dispose();
    push = _Push(tokens);
    _container = ProviderContainer(
      overrides: [
        offlineDatabaseProvider.overrideWithValue(database),
        userWorkDatabaseProvider.overrideWithValue(work),
        cacheCoordinatorProvider.overrideWithValue(coordinator),
        tokenStorageProvider.overrideWithValue(tokens),
        chatDraftStoreProvider.overrideWithValue(chats),
        scheduleWidgetBridgeProvider.overrideWithValue(widget),
        pushControllerProvider.overrideWith(() => push),
        themeModeProvider.overrideWith(_Theme.new),
        appLocaleProvider.overrideWith(_Locale.new),
        siteThemeProvider.overrideWith(_SiteTheme.new),
      ],
    );
  }

  static LocalDraft draft(String content) => LocalDraft(
    key: 'fixture-draft',
    title: 'Local work',
    content: content,
    contentType: 1,
    topicId: 0,
    categories: const [],
    images: const [],
    updatedAt: 1,
  );
  static ChatMessagePayload message(int id) => ChatMessagePayload(
    id: id,
    senderId: 7,
    content: 'Synced fixture message',
    msgType: 1,
    isRead: 0,
    createdAt: '',
    isSelf: true,
  );
  Future<void> seed() async {
    await container
        .read(writingStoreProvider)
        .save(scope, draft('keep until reset'));
    await work.writeBatch(scope, 'schedule', {
      'pk.plans': '[{"id":"fixture-plan","name":"Local plan"}]',
    });
    await cache.putMessages(1, [message(1)]);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('fixture.preference', 'value');
  }

  void dispose() => _container?.dispose();
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late AppDatabase database;
  late _Fixture fixture;
  late Directory directory;
  late PathProviderPlatform oldPaths;
  setUp(() async {
    directory = await Directory.systemTemp.createTemp('yourtj-reset-');
    oldPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Paths(directory.path);
    FlutterSecureStorage.setMockInitialValues({});
    database = AppDatabase(NativeDatabase.memory());
    await database.customSelect('SELECT 1').get();
    fixture = _Fixture(database, UserWorkDatabase.instance);
    await fixture.seed();
  });
  tearDown(() async {
    fixture.dispose();
    await database.close();
    PathProviderPlatform.instance = oldPaths;
    await directory.delete(recursive: true);
  });

  test(
    'ordinary cache clear preserves login, drafts, plans, chat drafts and preferences',
    () async {
      final epoch = fixture.container.read(offlineCacheEpochProvider);
      final result = await fixture.service.clear(CacheCategory.values.toSet());
      expect(result.succeeded, isTrue);
      expect(await fixture.cache.getMessages(1), isEmpty);
      expect(await fixture.database.operation('clear'), isNull);
      expect((await fixture.work.usage()).draftCount, 1);
      expect((await fixture.work.usage()).planCount, 1);
      expect(fixture.chats.count, 1);
      expect(await fixture.tokens.read(), 'fixture-session');
      expect(
        (await SharedPreferences.getInstance()).getString('fixture.preference'),
        'value',
      );
      expect(fixture.container.read(offlineCacheEpochProvider), epoch);
    },
  );

  test(
    'reset destroys local work and preferences and clears its durable intent',
    () async {
      final epoch = fixture.container.read(offlineCacheEpochProvider);
      await fixture.service.reset();
      expect(await fixture.database.operation('reset'), isNull);
      expect(await ResetJournal().isPending(), isFalse);
      expect(await fixture.database.operation('clear'), isNull);
      expect(await fixture.cache.getMessages(1), isEmpty);
      final work = await fixture.work.usage();
      expect(work.draftCount, 0);
      expect(work.planCount, 0);
      expect(fixture.chats.count, 0);
      expect(await fixture.tokens.read(), isNull);
      expect(fixture.push.seenSessions.first, 'fixture-session');
      expect(fixture.widget.transparency, 9);
      expect((await SharedPreferences.getInstance()).getKeys(), isEmpty);
      expect(
        fixture.container.read(offlineCacheEpochProvider),
        greaterThan(epoch),
      );
      // A new editing session after a completed reset uses the new generation.
      await fixture.container
          .read(writingStoreProvider)
          .save(_Fixture.scope, _Fixture.draft('new work'));
      expect((await fixture.work.usage()).draftCount, 1);
    },
  );

  test(
    'a fresh service resumes a persisted partial reset and finishes independent owners',
    () async {
      fixture.chats.failNextClear = true;
      await expectLater(fixture.service.reset(), throwsStateError);
      expect(await fixture.database.operation('reset'), '1');
      expect(await ResetJournal().isPending(), isTrue);
      expect(
        fixture.chats.count,
        1,
        reason: 'The failed owner retains its data for retry.',
      );
      expect(
        (await fixture.work.usage()).draftCount,
        0,
        reason: 'Other authorized deletion owners still finish.',
      );
      expect(await fixture.tokens.read(), isNull);
      fixture.reopenService();
      await fixture.container.read(storageBootstrapProvider.future);
      expect(fixture.chats.clears, 2);
      expect(fixture.chats.count, 0);
      expect(await fixture.database.operation('reset'), isNull);
      expect(await ResetJournal().isPending(), isFalse);
      expect(await fixture.database.operation('clear'), isNull);
      expect((await fixture.work.usage()).draftCount, 0);
    },
  );

  test(
    'reset recovery survives loss of the disposable cache journal',
    () async {
      fixture.chats.failNextClear = true;
      await expectLater(fixture.service.reset(), throwsStateError);
      expect(fixture.chats.count, 1);
      // Rebuilding an unreadable cache loses all storage_operations rows.
      await database.customStatement('DELETE FROM storage_operations');
      fixture.reopenService();
      await fixture.container.read(storageBootstrapProvider.future);
      expect(fixture.chats.count, 0);
      expect(fixture.chats.clears, 2);
      expect(fixture.container.read(storageResetStateProvider), isNull);
    },
  );

  for (final failure in ['missing cache key', 'corrupt cache']) {
    test(
      'production bootstrap continues a partial reset after $failure',
      () async {
        fixture.dispose();
        await database.close();
        database = AppDatabase(
          openPrivateDatabase(name: 'cache', disposable: true),
        );
        fixture = _Fixture(database, UserWorkDatabase.instance);
        await fixture.seed();
        fixture.chats.failNextClear = true;
        await expectLater(fixture.service.reset(), throwsStateError);
        final chats = fixture.chats;
        fixture.dispose();
        await database.close();
        if (failure == 'missing cache key') {
          FlutterSecureStorage.setMockInitialValues({});
        } else {
          await File(
            '${directory.path}/yourtj_private/cache.sqlite',
          ).writeAsBytes(List.filled(4096, 42), flush: true);
        }
        database = AppDatabase(
          openPrivateDatabase(name: 'cache', disposable: true),
        );
        fixture = _Fixture(
          database,
          UserWorkDatabase.instance,
          chatOwner: chats,
        );
        await fixture.container.read(storageBootstrapProvider.future);
        expect(chats.count, 0);
        expect(chats.clears, 2);
        expect(await ResetJournal().isPending(), isFalse);
        expect(await database.operation('reset'), isNull);
        expect(fixture.container.read(storageResetStateProvider), isNull);
      },
    );
  }

  test(
    'failure to persist reset intent leaves every owner untouched',
    () async {
      final marker = Directory('${directory.path}/yourtj_private/reset.intent');
      await marker.create(recursive: true);
      await expectLater(
        fixture.service.reset(),
        throwsA(isA<FileSystemException>()),
      );
      expect(fixture.tokens.clears, 0);
      expect(fixture.chats.clears, 0);
      expect(fixture.media.clears, 0);
      expect((await fixture.work.usage()).draftCount, 1);
      expect(
        fixture.container.read(storageResetStateProvider)?.hasError,
        isTrue,
      );
    },
  );

  test(
    'an interrupted marker write remains pending until the reset completes',
    () async {
      final marker = File('${directory.path}/yourtj_private/reset.intent');
      await marker.parent.create(recursive: true);
      await marker.writeAsString('', flush: true);
      fixture.reopenService();
      await fixture.container.read(storageBootstrapProvider.future);
      expect((await fixture.work.usage()).draftCount, 0);
      expect(fixture.chats.count, 0);
      expect(await marker.exists(), isFalse);
    },
  );

  test(
    'reset prevents pre-reset editors and captured network writers from refilling storage',
    () async {
      final writer = fixture.container.read(writingStoreProvider);
      final request = fixture.cache.capture();
      await fixture.service.reset();
      await expectLater(
        writer.save(_Fixture.scope, _Fixture.draft('late editor save')),
        throwsStateError,
      );
      await request.putMessages(1, [_Fixture.message(99)]);
      expect(await fixture.cache.getMessages(1), isEmpty);
      expect((await fixture.work.usage()).draftCount, 0);
    },
  );

  test(
    'a failed independent cache owner stays suspended until its journal is retried',
    () async {
      fixture.media.failNextClear = true;
      final failed = await fixture.service.clear({CacheCategory.media});
      expect(failed.failed, {CacheCategory.media});
      expect(await fixture.coordinator.pending(), {CacheCategory.media});
      expect(
        fixture.media.suspended,
        isTrue,
        reason: 'A failed clear must not serve or refill the remaining media.',
      );
      final retried = await fixture.service.resume();
      expect(retried.succeeded, isTrue);
      expect(fixture.media.suspended, isFalse);
      expect(fixture.media.hasData, isFalse);
      expect(await fixture.database.operation('clear'), isNull);
    },
  );
}
