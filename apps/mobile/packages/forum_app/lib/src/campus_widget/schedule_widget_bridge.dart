import 'dart:convert';
import 'dart:io';

import 'package:home_widget/home_widget.dart';

import 'schedule_widget_projection.dart';

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

  Future<void> write(ScheduleWidgetProjection projection) async {
    if (!Platform.isAndroid && !Platform.isIOS) return;
    await HomeWidget.setAppGroupId(appGroupId);
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

  Future<void> clear({String state = 'needsData'}) async {
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
