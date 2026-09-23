import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

String read(String path) => File(path).readAsStringSync();

void main() {
  test('Android Glance widgets are local, resizable and reboot-safe', () {
    final source = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );
    final data = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgetData.kt',
    );
    final manifest = read('android/app/src/main/AndroidManifest.xml');
    final next = read(
      'android/app/src/main/res/xml/next_class_widget_info.xml',
    );
    final today = read(
      'android/app/src/main/res/xml/today_schedule_widget_info.xml',
    );
    expect(source, contains('SizeMode.Exact'));
    expect(source, contains('LocalSize.current'));
    expect(source, contains('.semantics'));
    expect(source, contains('yourtj://campus/today'));
    expect('$source$data', contains('authorizationRequired'));
    expect('$source$data', isNot(contains('java.net')));
    expect('$source$data', isNot(contains('OkHttp')));
    expect(manifest, contains('HomeWidgetScheduledUpdateReceiver'));
    expect(manifest, contains('android.intent.action.BOOT_COMPLETED'));
    expect(
      manifest,
      isNot(
        contains(
          'uses-permission android:name="android.permission.SCHEDULE_EXACT_ALARM"',
        ),
      ),
    );
    expect(next, contains('android:minWidth="110dp"'));
    expect(today, contains('android:resizeMode="horizontal|vertical"'));
    expect('$next$today', contains('android:updatePeriodMillis="0"'));
    expect(
      read('android/app/src/main/res/values-v31/widget_colors.xml'),
      contains('@android:color/system_accent1_600'),
    );
    expect(data, contains('"days"'));
    expect(data, contains('optionalText'));
    expect(data, contains('"null"'));
    expect(data, contains('"undefined"'));
    expect(source, contains('LargeSchedule'));
    expect(source, contains('cornerRadius(24.dp)'));
    expect(source, contains('今天'));
    expect(source, contains('明天'));
    expect(source, contains('更新于'));
    expect('$data$source', contains('今天暂无课程安排'));
    expect('$data$source', contains('打开 YourTJ 刷新课表'));
    expect(source, isNot(contains('TextOverflow.Ellipsis')));
    expect(source, contains(RegExp(r'course\.name,\s+maxLines = 2')));
  });

  test('Android large schedule has two independent equal-width collections', () {
    final source = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );
    final large = source.substring(
      source.indexOf('private fun LargeSchedule'),
      source.indexOf('private fun DayHeader'),
    );

    expect(source, isNot(contains('(LocalSize.current.width -')));
    expect(large, contains(RegExp(r'LargeDayColumn\(\s*"今天"')));
    expect(large, contains(RegExp(r'LargeDayColumn\(\s*"明天"')));
    expect(large, contains('LazyColumn'));
    expect(large, contains('items = day.courses'));
    expect(
      large,
      contains('itemId = { course -> course.id.hashCode().toLong() }'),
    );
    expect(large, contains('defaultWeight()'));
    expect(large, contains('fillMaxHeight()'));
    expect(large, contains('Divider('));
    expect(large, contains('contentDescription = description'));
    expect(large, isNot(contains('CoursePairRow')));
    expect(large, isNot(contains('items(itemCount)')));
    expect(large, isNot(contains('take(2)')));
    expect(large, isNot(contains('Remaining(')));
  });

  test('course bars stay 4dp regardless of current state', () {
    final android = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );
    final ios = read('ios/ScheduleWidgets/ScheduleWidgets.swift');

    expect(android, contains('.width(4.dp)'));
    expect(android, isNot(contains('.width(if (current)')));
    expect(ios, contains('.frame(width: 4, height: 48)'));
    expect(ios, isNot(contains('width: current ?')));
  });

  test('NextClass selects future days and has a dense wide compact layout', () {
    final source = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );
    final data = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgetData.kt',
    );

    expect(data, contains('fun nextClass('));
    expect(data, contains('days.asSequence()'));
    expect(data, contains('val campus: String'));
    expect(data, contains('fun ScheduleCourse.locationText()'));
    expect(data, contains('listOf(campus, room)'));
    expect(data, contains("filter { it.isNotBlank() }"));
    expect(source, contains('enum class NextClassSize'));
    expect(source, contains('nextClassSize(widgetSize)'));
    expect(source, contains('NextClassSize.WideCompact'));
    expect(source, contains('NextClassSize.Expanded'));
    expect(source, contains('size.width >= 250.dp'));
    expect(source, contains('WideCompactNextClass'));
    final nextClass = source.substring(
      source.indexOf('class NextClassWidget'),
      source.indexOf('class TodayScheduleWidget'),
    );
    expect(nextClass, contains('course?.locationText()'));
    expect(nextClass, contains('detailText("地点", "Location", it)'));
    expect(nextClass, contains('course?.teacher?.takeIf { it.isNotBlank() }'));
    expect(nextClass, contains('detailText("教师", "Teacher", it)'));
    expect(source, contains('generatedAt?.let(::updatedText)'));
    expect(source, contains('distanceText('));
  });

  test('iOS WidgetKit target, App Group and timeline are fully configured', () {
    final project = read('ios/Runner.xcodeproj/project.pbxproj');
    final source = read('ios/ScheduleWidgets/ScheduleWidgets.swift');
    final runnerEntitlements = read('ios/Runner/Runner.entitlements');
    final widgetEntitlements = read(
      'ios/ScheduleWidgets/ScheduleWidgets.entitlements',
    );
    for (final expected in [
      'ScheduleWidgets.appex',
      'Embed App Extensions',
      'com.apple.product-type.app-extension',
      'tj.yourtj.forumApp.ScheduleWidgets',
    ]) {
      expect(project, contains(expected));
    }
    expect(runnerEntitlements, contains('group.tj.yourtj.forumApp.widgets'));
    expect(widgetEntitlements, contains('group.tj.yourtj.forumApp.widgets'));
    expect(source, contains('.supportedFamilies([.systemSmall])'));
    expect(source, contains('[.systemMedium, .systemLarge]'));
    expect(
      source,
      contains('containerBackground(for: .widget)'),
    );
    expect(source, contains('ContainerRelativeShape().fill(.ultraThinMaterial)'));
    expect(source, contains('widgetAccentable()'));
    expect(project, contains('IPHONEOS_DEPLOYMENT_TARGET = 14.0'));
    expect(source, contains('timelineDates(after:'));
    expect(source, contains('Asia/Shanghai'));
    expect(source, contains('emptyStateKey'));
    expect(source, contains('safeText'));
    expect(source, contains('projection.days'));
    expect(source, contains('LargeDayColumn'));
    expect(source, contains('今天'));
    expect(source, contains('明天'));
    expect(source, contains('更新于'));
    expect(source, contains('今天暂无课程安排'));
    expect(source, contains('打开 YourTJ 刷新课表'));
    expect(source, isNot(contains('.lineLimit(1)')));
    expect(source, isNot(contains('.truncationMode')));
    expect(source, isNot(contains('URLSession')));
    expect(source, isNot(contains('import Network')));
    expect(source, contains('func nextClass(at date: Date)'));
    expect(source, contains('Divider()'));
    expect(source, contains('.frame(width: 4, height: 48)'));
    expect(
      '.frame(maxWidth: .infinity'.allMatches(source).length,
      greaterThanOrEqualTo(2),
    );
    expect(source, contains('.prefix(2)'));
    expect(source, contains('Remaining(count:'));
    expect(source, isNot(contains('ScrollView')));
  });

  test('home_widget bridge stays compatible with the iOS 13 Runner', () {
    final project = read('ios/Runner.xcodeproj/project.pbxproj');
    final mobilePubspec = read('../../pubspec.yaml');
    final package = read(
      '../../third_party/home_widget/ios/home_widget/Package.swift',
    );
    final podspec = read(
      '../../third_party/home_widget/ios/home_widget.podspec',
    );

    expect(
      RegExp(r'IPHONEOS_DEPLOYMENT_TARGET = 13\.0;').allMatches(project).length,
      3,
    );
    expect(
      RegExp(r'IPHONEOS_DEPLOYMENT_TARGET = 14\.0;').allMatches(project).length,
      3,
    );
    expect(mobilePubspec, contains('path: third_party/home_widget'));
    expect(package, contains('.iOS("13.0")'));
    expect(podspec, contains("s.platform = :ios, '13.0'"));
  });

  test('development APK helper keeps all supported ABI splits', () {
    final script = read('../../scripts/build_dev_apk.sh');
    expect(script, contains('--split-per-abi'));
    expect(script, contains('app-*-debug.apk'));
  });
}
