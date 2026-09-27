import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/scheduler.dart';

import 'src/app.dart';
import 'src/startup_metrics.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  initializeStartupMetrics();
  final binding = SchedulerBinding.instance;
  late TimingsCallback callback;
  callback = (timings) {
    if (!timings.any((timing) => timing.rasterDuration > Duration.zero)) return;
    recordStartupMilestone('startup.flutter_first_frame_rasterized');
    recordStartupDeviceProfile();
    binding.removeTimingsCallback(callback);
  };
  binding.addTimingsCallback(callback);
  runApp(const ProviderScope(child: GfApp()));
}
