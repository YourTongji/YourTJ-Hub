import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/settings/storage_settings_panel.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/storage/cache_coordinator.dart';
import 'package:forum_app/src/storage/device_storage.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:forum_app/src/providers.dart';
import 'package:core/core.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

class FakeDeviceStorage implements DeviceStorage {
  FakeDeviceStorage({this.recoveryPlans = 0, this.legacyPlans = 0});
  final int recoveryPlans;
  final int legacyPlans;
  Set<CacheCategory>? cleared;
  bool fail = false;
  int resets = 0;
  @override
  Future<DeviceStorageUsage> usage() async => DeviceStorageUsage(
    cacheBytes: {for (final category in CacheCategory.values) category: 1024},
    physicalCacheBytes: 5000,
    workBytes: 1000,
    drafts: 3,
    chatDrafts: 1,
    plans: 2,
    unsyncedPlans: 1,
    recoveryPlans: recoveryPlans,
    legacyPlans: legacyPlans,
    pending: fail ? {CacheCategory.media} : {},
  );
  @override
  Future<CacheClearResult> clear(Set<CacheCategory> categories) async {
    cleared = categories;
    return CacheClearResult(fail ? {CacheCategory.media} : {}, 1024);
  }

  @override
  Future<CacheClearResult> resume() => clear({CacheCategory.media});
  @override
  Future<void> reset() async {
    resets++;
  }
}

class GuestTokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

Widget fixture(
  FakeDeviceStorage service, {
  Widget? child,
  String locale = 'zh',
  Brightness brightness = Brightness.light,
}) => ProviderScope(
  overrides: [
    deviceStorageProvider.overrideWithValue(service),
    tokenStorageProvider.overrideWithValue(GuestTokens()),
  ],
  child: MaterialApp(
    theme: gfThemeData(brightness),
    locale: Locale(locale),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child ?? const StorageSettingsPanel()),
  ),
);
void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  testWidgets(
    'guest can access device storage and cloud management is absent',
    (tester) async {
      await tester.pumpWidget(
        fixture(
          FakeDeviceStorage(),
          child: const SettingsPage(initialSection: 'privacy'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('本机缓存占用'), findsOneWidget);
      expect(find.text('内容管理'), findsNothing);
      expect(find.text('回收站'), findsNothing);
    },
  );
  testWidgets(
    'cache confirmation selects forum chat media and preserves campus by default',
    (tester) async {
      final service = FakeDeviceStorage();
      await tester.pumpWidget(fixture(service));
      await tester.pumpAndSettle();
      final button = find.text('清理所选缓存');
      await tester.scrollUntilVisible(
        button,
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(button);
      await tester.pumpAndSettle();
      expect(service.cleared, isNull);
      expect(find.textContaining('登录、草稿、未发送私信和排课方案都会保留'), findsWidgets);
      await tester.tap(find.text('确认'));
      await tester.pumpAndSettle();
      expect(service.cleared, {
        CacheCategory.forum,
        CacheCategory.chat,
        CacheCategory.media,
      });
      expect(service.resets, 0);
    },
  );
  testWidgets('local reset shows unsynced counts and cancel preserves work', (
    tester,
  ) async {
    final service = FakeDeviceStorage();
    await tester.pumpWidget(fixture(service));
    await tester.pumpAndSettle();
    final button = find.widgetWithText(GfButton, '重置本机数据');
    await tester.scrollUntilVisible(
      button,
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(button);
    await tester.pumpAndSettle();
    expect(
      find.textContaining('3 份草稿 · 1 份未发送私信 · 2 个排课方案（1 个未同步）'),
      findsWidgets,
    );
    expect(find.textContaining('永久删除'), findsOneWidget);
    await tester.tap(find.text('取消'));
    await tester.pumpAndSettle();
    expect(service.resets, 0);
  });
  for (final (locale, recovery, legacy, reset, cancel) in [
    ('zh', '排课恢复草稿：2', '1 份旧排课数据待确认归属', '重置本机数据', '取消'),
    (
      'en',
      'Schedule recovery drafts: 2',
      '1 legacy schedules need an owner',
      'Reset local data',
      'Cancel',
    ),
    (
      'de',
      'Wiederherstellungsentwürfe für Stundenpläne: 2',
      '1 alte Stundenpläne benötigen eine Zuordnung',
      'Lokale Daten zurücksetzen',
      'Abbrechen',
    ),
    ('ja', '履修計画の復元用下書き：2 件', '旧履修計画 1 件の所有者を確認', '端末データをリセット', 'キャンセル'),
  ]) {
    testWidgets('recovery drafts appear in storage and reset summary $locale', (
      tester,
    ) async {
      final service = FakeDeviceStorage(recoveryPlans: 2, legacyPlans: 1);
      await tester.pumpWidget(fixture(service, locale: locale));
      await tester.pumpAndSettle();
      final summary = find.textContaining(recovery);
      await tester.scrollUntilVisible(
        summary,
        200,
        scrollable: find.byType(Scrollable).first,
      );
      expect(summary, findsOneWidget);
      await tester.scrollUntilVisible(
        find.textContaining(legacy),
        100,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.textContaining(legacy), findsOneWidget);
      final resetButton = find.widgetWithText(GfButton, reset);
      await tester.scrollUntilVisible(
        resetButton,
        200,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.tap(resetButton);
      await tester.pumpAndSettle();
      expect(
        find.descendant(
          of: find.byType(GfAlertDialog),
          matching: find.textContaining(recovery),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: find.byType(GfAlertDialog),
          matching: find.textContaining(legacy),
        ),
        findsOneWidget,
      );
      await tester.tap(find.text(cancel));
      await tester.pumpAndSettle();
      expect(service.resets, 0);
    });
  }
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    for (final brightness in Brightness.values) {
      testWidgets(
        'storage remains scrollable at 320px 200% $locale $brightness',
        (tester) async {
          tester.view.physicalSize = const Size(320, 844);
          tester.view.devicePixelRatio = 1;
          tester.platformDispatcher.textScaleFactorTestValue = 2;
          addTearDown(tester.view.reset);
          addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
          await tester.pumpWidget(
            fixture(
              FakeDeviceStorage(),
              locale: locale,
              brightness: brightness,
            ),
          );
          await tester.pumpAndSettle();
          await tester.drag(find.byType(ListView), const Offset(0, -700));
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}
