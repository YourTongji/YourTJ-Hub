import 'dart:convert';
import 'dart:io';

import 'package:home_widget/home_widget.dart';

import 'schedule_widget_projection.dart';

/// Home-screen widgets the app can ask the launcher to place.
enum ScheduleWidgetKind { nextClass, today, timeline }

class ScheduleWidgetBridge {
  static const appGroupId = 'group.tj.yourtj.forumApp.widgets';
  static const projectionKey = 'schedule_widget_projection';
  static const emptyStateKey = 'schedule_widget_empty_state';
  static const transparencyKey = 'schedule_widget_transparency_percent';
  static const defaultTransparencyPercent = 9;
  static const minTransparencyPercent = 0;
  static const maxTransparencyPercent = 15;
  static const _temporaryKey = '${projectionKey}_pending';
  static const _nextReceiver =
      'tj.yourtj.forum_app.widget.NextClassWidgetReceiver';
  static const _todayReceiver =
      'tj.yourtj.forum_app.widget.TodayScheduleWidgetReceiver';
  static const _timelineReceiver =
      'tj.yourtj.forum_app.widget.CourseTimelineWidgetReceiver';

  static const _receivers = {
    ScheduleWidgetKind.nextClass: _nextReceiver,
    ScheduleWidgetKind.today: _todayReceiver,
    ScheduleWidgetKind.timeline: _timelineReceiver,
  };

  /// Android 8+ launchers may place a widget on request. iOS has no such API;
  /// users add widgets from the Home Screen gallery.
  Future<bool> canRequestPin() async {
    if (!Platform.isAndroid) return false;
    try {
      return await HomeWidget.isRequestPinWidgetSupported() ?? false;
    } catch (_) {
      return false;
    }
  }

  /// Opens the launcher's placement dialog; the user still confirms there.
  Future<void> requestPin(ScheduleWidgetKind kind) =>
      HomeWidget.requestPinWidget(qualifiedAndroidName: _receivers[kind]);

  int generation = 0;
  Future<void> _tail = Future.value();

  /// Latest timetable blocks per account scope, from the app's local store.
  final _blocks = <String, List<ScheduleWidgetBlock>>{};
  void invalidate() => generation++;

  /// The native widget container is shared by dev and production app installs.
  /// Clear the previous site's projection before exposing a new environment.
  Future<void> prepareEnvironment(String site) => _serialize(() async {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    await HomeWidget.setAppGroupId(appGroupId);
    const key = 'schedule_widget_site';
    final previous =
        await HomeWidget.getWidgetData<String>(key) ?? 'https://f.yourtj.de';
    if (previous != site) await _clear('needsData');
    await HomeWidget.saveWidgetData<String>(key, site);
  });

  Future<void> _serialize(Future<void> Function() operation) {
    final result = _tail.then((_) => operation());
    _tail = result.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return result;
  }

  Future<void> write(ScheduleWidgetProjection projection) {
    final fence = generation;
    return _serialize(() async {
      if (fence != generation) return;
      await _write(projection);
    });
  }

  /// Publishes the user's own blocks: the stored projection is laid out
  /// again now, and every later school refresh carries them.
  Future<void> setBlocks(
    String accountScope,
    List<ScheduleWidgetBlock> blocks,
  ) {
    final previous = _blocks[accountScope];
    if (previous != null && _sameBlocks(previous, blocks)) {
      return Future.value();
    }
    _blocks[accountScope] = List.unmodifiable(blocks);
    final fence = generation;
    return _serialize(() async {
      if (fence != generation) return;
      final current = await read();
      if (current?.identity.accountScope != accountScope) return;
      await _write(current!);
    });
  }

  static bool _sameBlocks(
    List<ScheduleWidgetBlock> a,
    List<ScheduleWidgetBlock> b,
  ) =>
      jsonEncode([for (final x in a) x.toJson()]) ==
      jsonEncode([for (final x in b) x.toJson()]);

  Future<void> _write(ScheduleWidgetProjection published) async {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    await HomeWidget.setAppGroupId(appGroupId);
    var projection = published;
    final scope = projection.identity.accountScope;
    // Before the local store loads, keep the blocks already published.
    final blocks =
        _blocks[scope] ??
        switch (await read()) {
          final stored? when stored.identity.accountScope == scope =>
            stored.blocks,
          _ => null,
        };
    if (blocks != null) projection = projection.withBlocks(blocks);
    final encoded = projection.encode();
    await HomeWidget.saveWidgetData<String>(_temporaryKey, encoded);
    final staged = await HomeWidget.getWidgetData<String>(_temporaryKey);
    ScheduleWidgetProjection.fromJson(
      jsonDecode(staged ?? '') as Map<String, dynamic>,
    );
    await HomeWidget.saveWidgetData<String>(projectionKey, staged);
    await HomeWidget.saveWidgetData<String>(emptyStateKey, 'ready');
    await HomeWidget.saveWidgetData<String>(_temporaryKey, null);
    final times = projection.updateTimesAfter(DateTime.now());
    await Future.wait([
      HomeWidget.scheduleWidgetUpdates(
        times,
        qualifiedAndroidName: _nextReceiver,
      ),
      HomeWidget.scheduleWidgetUpdates(
        times,
        qualifiedAndroidName: _todayReceiver,
      ),
    ]);
    await _reload();
  }

  Future<ScheduleWidgetProjection?> read() async {
    if (!Platform.isAndroid && !Platform.isIOS) return null;
    await HomeWidget.setAppGroupId(appGroupId);
    final encoded = await HomeWidget.getWidgetData<String>(projectionKey);
    if (encoded == null || encoded.isEmpty) return null;
    try {
      return ScheduleWidgetProjection.fromJson(
        jsonDecode(encoded) as Map<String, dynamic>,
      );
    } catch (_) {
      return null;
    }
  }

  Future<int> usageBytes() async {
    if (!Platform.isAndroid && !Platform.isIOS) return 0;
    await HomeWidget.setAppGroupId(appGroupId);
    var total = 0;
    for (final key in [projectionKey, _temporaryKey]) {
      final value = await HomeWidget.getWidgetData<String>(key);
      if (value != null) total += utf8.encode(value).length;
    }
    return total;
  }

  Future<int> readTransparency() async {
    if (!Platform.isAndroid && !Platform.isIOS) {
      return defaultTransparencyPercent;
    }
    await HomeWidget.setAppGroupId(appGroupId);
    final value = await HomeWidget.getWidgetData<int>(transparencyKey);
    return (value ?? defaultTransparencyPercent)
        .clamp(minTransparencyPercent, maxTransparencyPercent)
        .toInt();
  }

  Future<void> setTransparency(int value) async {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    await HomeWidget.setAppGroupId(appGroupId);
    await HomeWidget.saveWidgetData<int>(
      transparencyKey,
      value.clamp(minTransparencyPercent, maxTransparencyPercent).toInt(),
    );
    await _reload();
  }

  Future<void> clear({String state = 'needsData'}) {
    invalidate();
    return _serialize(() => _clear(state));
  }

  Future<void> _clear(String state) async {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    await HomeWidget.setAppGroupId(appGroupId);
    await HomeWidget.saveWidgetData<String>(_temporaryKey, null);
    await HomeWidget.saveWidgetData<String>(projectionKey, null);
    await HomeWidget.saveWidgetData<String>(emptyStateKey, state);
    await Future.wait([
      HomeWidget.cancelScheduledWidgetUpdates(
        qualifiedAndroidName: _nextReceiver,
      ),
      HomeWidget.cancelScheduledWidgetUpdates(
        qualifiedAndroidName: _todayReceiver,
      ),
    ]);
    await _reload();
  }

  Stream<Uri?> links() async* {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    yield await HomeWidget.initiallyLaunchedFromHomeWidget();
    yield* HomeWidget.widgetClicked;
  }

  Future<void> _reload() {
    final updates = <Future<void>>[
      HomeWidget.updateWidget(
        qualifiedAndroidName: _nextReceiver,
        iOSName: 'NextClassWidget',
      ),
      HomeWidget.updateWidget(
        qualifiedAndroidName: _todayReceiver,
        iOSName: 'TodayScheduleWidget',
      ),
    ];
    if (Platform.isAndroid) {
      updates.add(
        HomeWidget.updateWidget(qualifiedAndroidName: _timelineReceiver),
      );
    }
    return Future.wait(updates);
  }
}
