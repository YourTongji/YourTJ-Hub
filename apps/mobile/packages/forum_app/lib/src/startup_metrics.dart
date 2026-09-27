import 'dart:async';
import 'dart:developer' as developer;

import 'package:flutter/foundation.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

const MethodChannel _startupChannel = MethodChannel('yourtj/startup');
final Stopwatch _startupClock = Stopwatch();
final _StartupLifecycleObserver _lifecycleObserver =
    _StartupLifecycleObserver();
bool _firstFlutterFrameRecorded = false;
bool _firstHomeCardRecorded = false;
int? _firstHomeCardElapsedMicros;
int? _firstHomeMediaElapsedMicros;
int? _fullyDrawnElapsedMicros;

String get _buildMode => kReleaseMode
    ? 'release'
    : kProfileMode
    ? 'profile'
    : 'debug';

void initializeStartupMetrics() {
  _startupClock
    ..reset()
    ..start();
  WidgetsBinding.instance.addObserver(_lifecycleObserver);
}

void recordStartupMilestone(
  String name, {
  Map<String, Object?> arguments = const <String, Object?>{},
}) {
  if (name == 'startup.flutter_first_frame_rasterized') {
    _firstFlutterFrameRecorded = true;
  }
  developer.Timeline.instantSync(
    name,
    arguments: <String, Object?>{
      'elapsedMicros': _startupClock.elapsedMicroseconds,
      'buildMode': _buildMode,
      ...arguments,
    },
  );
}

void recordStartupDeviceProfile({String? launchKindOverride}) {
  unawaited(_recordStartupDeviceProfile(launchKindOverride));
}

Future<void> _recordStartupDeviceProfile(String? launchKindOverride) async {
  Map<dynamic, dynamic>? profile;
  try {
    profile = await _startupChannel.invokeMapMethod<dynamic, dynamic>(
      'getDeviceProfile',
    );
  } on MissingPluginException {
    // Platform channels may be absent in widget tests and desktop tooling.
  } on PlatformException {
    // Startup instrumentation must not affect the app launch path.
  }
  final views = WidgetsBinding.instance.platformDispatcher.views;
  recordStartupMilestone(
    'startup.device_profile',
    arguments: <String, Object?>{
      'platform': defaultTargetPlatform.name,
      'device': profile?['device'] ?? 'unknown',
      'osVersion': profile?['osVersion'] ?? 'unknown',
      'refreshRateHz':
          profile?['refreshRateHz'] ??
          (views.isEmpty ? null : views.first.display.refreshRate),
      'launchKind': launchKindOverride ?? profile?['launchKind'] ?? 'unknown',
    },
  );
}

void recordFirstHomeContentAndFullyDrawn({required bool hasTopics}) {
  if (_firstHomeCardRecorded) return;
  _firstHomeCardRecorded = true;
  _firstHomeCardElapsedMicros = _startupClock.elapsedMicroseconds;
  recordStartupMilestone(
    hasTopics
        ? 'startup.home_first_card_frame_submitted'
        : 'startup.home_empty_state_frame_submitted',
  );
  unawaited(_reportFullyDrawn());
}

void recordFirstHomeMediaFrame() {
  if (_firstHomeMediaElapsedMicros != null) return;
  _firstHomeMediaElapsedMicros = _startupClock.elapsedMicroseconds;
  final fullyDrawnElapsed = _fullyDrawnElapsedMicros;
  recordStartupMilestone(
    'startup.home_first_media_frame_ready',
    arguments: <String, Object?>{
      'firstMediaToFullyDrawnMicros': fullyDrawnElapsed == null
          ? null
          : fullyDrawnElapsed - _firstHomeMediaElapsedMicros!,
    },
  );
}

Future<void> _reportFullyDrawn() async {
  var nativeReported = false;
  try {
    nativeReported =
        await _startupChannel.invokeMethod<bool>('reportFullyDrawn') ?? false;
  } on MissingPluginException {
    // Android reports this through Activity.reportFullyDrawn; other platforms
    // still record the Flutter fully-drawn milestone.
  } on PlatformException {
    // Native reporting is diagnostic and must not affect the UI.
  }
  final firstCardElapsed = _firstHomeCardElapsedMicros;
  _fullyDrawnElapsedMicros = _startupClock.elapsedMicroseconds;
  final firstMediaElapsed = _firstHomeMediaElapsedMicros;
  recordStartupMilestone(
    'startup.fully_drawn',
    arguments: <String, Object?>{
      'androidReportFullyDrawn': nativeReported,
      'firstCardToFullyDrawnMicros': firstCardElapsed == null
          ? null
          : _fullyDrawnElapsedMicros! - firstCardElapsed,
      'firstMediaToFullyDrawnMicros': firstMediaElapsed == null
          ? null
          : _fullyDrawnElapsedMicros! - firstMediaElapsed,
    },
  );
}

class _StartupLifecycleObserver extends WidgetsBindingObserver {
  bool _backgrounded = false;
  final Stopwatch _hotResumeClock = Stopwatch();

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.hidden ||
        state == AppLifecycleState.detached) {
      _backgrounded = true;
      return;
    }
    if (state != AppLifecycleState.resumed ||
        !_backgrounded ||
        !_firstFlutterFrameRecorded) {
      return;
    }
    _backgrounded = false;
    _hotResumeClock
      ..reset()
      ..start();
    recordStartupMilestone(
      'startup.hot_foreground_resumed',
      arguments: const <String, Object?>{'launchKind': 'hot'},
    );
    recordStartupDeviceProfile(launchKindOverride: 'hot');

    final binding = SchedulerBinding.instance;
    late final TimingsCallback callback;
    callback = (timings) {
      if (!timings.any((timing) => timing.rasterDuration > Duration.zero)) {
        return;
      }
      developer.Timeline.instantSync(
        'startup.hot_first_frame_rasterized',
        arguments: <String, Object?>{
          'elapsedMicros': _hotResumeClock.elapsedMicroseconds,
          'buildMode': _buildMode,
          'launchKind': 'hot',
        },
      );
      binding.removeTimingsCallback(callback);
    };
    binding.addTimingsCallback(callback);
  }
}
