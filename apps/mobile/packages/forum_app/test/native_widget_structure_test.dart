import 'dart:io';
import 'dart:math' as math;

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
    expect(next, contains('android:targetCellWidth="4"'));
    expect(next, contains('android:minWidth="250dp"'));
    expect(next, contains('android:minResizeWidth="250dp"'));
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
    expect(source, contains('day = colorFor(Configuration.UI_MODE_NIGHT_NO)'));
    expect(
      source,
      contains('night = colorFor(Configuration.UI_MODE_NIGHT_YES)'),
    );
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
      source.indexOf('private fun LargeDayColumn'),
    );
    final dayColumn = source.substring(
      source.indexOf('private fun LargeDayColumn'),
      source.indexOf('private fun Divider'),
    );

    expect(source, isNot(contains('(LocalSize.current.width -')));
    expect(large, contains(RegExp(r'LargeDayColumn\(\s*"今天"')));
    expect(large, contains(RegExp(r'LargeDayColumn\(\s*"明天"')));
    expect(dayColumn, contains('LazyColumn'));
    expect(dayColumn, contains('items = day.courses'));
    expect(
      dayColumn,
      contains('itemId = { course -> course.id.hashCode().toLong() }'),
    );
    expect(large, contains('defaultWeight()'));
    expect(large, contains('fillMaxHeight()'));
    expect(large, contains('Divider('));
    expect(dayColumn, contains('contentDescription = description'));
    expect(dayColumn, isNot(contains('CoursePairRow')));
    expect(dayColumn, isNot(contains('items(itemCount)')));
    expect(dayColumn, isNot(contains('take(2)')));
    expect(dayColumn, isNot(contains('Remaining(')));
  });

  test(
    'Android 12 widget picker scales the full-canvas preview within host bounds',
    () {
      final preview = read(
        'android/app/src/main/res/layout/today_schedule_widget_preview_v2.xml',
      );
      final provider = read(
        'android/app/src/main/res/xml/today_schedule_widget_info.xml',
      );

      expect(preview, contains('<ImageView'));
      expect(preview, contains('android:layout_width="match_parent"'));
      expect(preview, contains('android:layout_height="match_parent"'));
      expect(preview, contains('android:scaleType="fitCenter"'));
      expect(preview, isNot(contains('<TextView')));
      expect(
        preview,
        contains('android:src="@drawable/today_schedule_widget_preview_v2"'),
      );
      expect(
        provider,
        contains(
          'android:previewLayout="@layout/today_schedule_widget_preview_v2"',
        ),
      );
      expect(
        provider,
        contains(
          'android:previewImage="@drawable/today_schedule_widget_preview_v2"',
        ),
      );
    },
  );

  test('Android keeps 4dp bars and iOS uses height-fitting course stripes', () {
    final ios = read('ios/ScheduleWidgets/ScheduleWidgets.swift');

    expect(ios, isNot(contains('.frame(width: 4, height: 48)')));
    expect(ios, contains('courseStripeColor(course.colorSlot'));
    final nextClass = ios.substring(
      ios.indexOf('private struct NextClassView'),
      ios.indexOf('private struct TodayScheduleView'),
    );
    expect(nextClass, isNot(contains('Capsule()')));
    expect(nextClass, isNot(contains('distanceText(')));
    final courseRow = ios.substring(ios.indexOf('private struct CourseRow'));
    expect(courseRow, isNot(contains('course.campus')));
    expect(courseRow, contains('compactRoom(course.room)'));
    expect(courseRow, contains('course.room'));
    expect(courseRow, contains('course.teacher'));
    expect(ios, isNot(contains('width: current ?')));
  });

  test('course bars stay 4dp regardless of current state', () {
    final android = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );

    expect(android, contains('.width(4.dp)'));
    expect(android, isNot(contains('.width(if (current)')));
  });

  test('iOS course stripe colors match the web timetable CSS', () {
    final ios = read('ios/ScheduleWidgets/ScheduleWidgets.swift');
    final css = read('../../../gooseforum/resource/src/styles/tokens.css');
    final cssColors =
        RegExp(
          r'--gf-color-course-[1-8]: oklch\(([\d.]+)% ([\d.]+) ([\d.]+)\)',
        ).allMatches(css).map((match) {
          final lightness = double.parse(match.group(1)!) / 100;
          final chroma = double.parse(match.group(2)!);
          final hue = double.parse(match.group(3)!) * math.pi / 180;
          final a = chroma * math.cos(hue);
          final b = chroma * math.sin(hue);
          final l = math.pow(
            lightness + 0.3963377774 * a + 0.2158037573 * b,
            3,
          );
          final m = math.pow(
            lightness - 0.1055613458 * a - 0.0638541728 * b,
            3,
          );
          final s = math.pow(lightness - 0.0894841775 * a - 1.291485548 * b, 3);
          final channels = <double>[
            4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
            -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
            -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
          ];
          final rgb = channels.map((value) {
            final channel = value > 0.0031308
                ? 1.055 * math.pow(value, 1 / 2.4) - 0.055
                : 12.92 * value;
            return (channel.clamp(0, 1) * 255).round();
          }).toList();
          return (rgb[0] << 16) | (rgb[1] << 8) | rgb[2];
        }).toList();

    List<int> nativeColors(String name) {
      final block = RegExp(
        'private let $name: \\[UInt32\\] = \\[([\\s\\S]*?)\\]',
      ).firstMatch(ios);
      expect(block, isNotNull);
      return RegExp(r'0x[0-9A-F]{6}')
          .allMatches(block!.group(1)!)
          .map((match) => int.parse(match.group(0)!.substring(2), radix: 16))
          .toList();
    }

    expect(cssColors.length, 16);
    expect(nativeColors('lightCourseStripeRGB'), cssColors.sublist(0, 8));
    expect(nativeColors('darkCourseStripeRGB'), cssColors.sublist(8));
  });

  test('iOS widget text stays at least 11 points', () {
    final source = read('ios/ScheduleWidgets/ScheduleWidgets.swift');
    final sizes = RegExp(
      r'\.font\(\.system\(size: (\d+)',
    ).allMatches(source).map((match) => int.parse(match.group(1)!));
    expect(sizes, isNotEmpty);
    expect(sizes.every((size) => size >= 11), isTrue);
  });

  test('iOS compact widgets retain two full courses and one preview', () {
    final source = read('ios/ScheduleWidgets/ScheduleWidgets.swift');
    final small = source.substring(
      source.indexOf('private struct NextClassView'),
      source.indexOf('private struct TodayScheduleView'),
    );
    expect(small, contains('prefix(3)'));
    expect(small, contains('CompactCoursePreview(course: courses[index])'));
    expect(small, contains('CourseRow(course: courses[index]'));
    expect(source, contains('"教学北楼": "北"'));
  });

  test(
    'iOS medium widget uses compact date headings and adapts its two columns',
    () {
      final source = read('ios/ScheduleWidgets/ScheduleWidgets.swift');
      expect(source, contains('private struct MediumSchedule'));
      final medium = source.substring(
        source.indexOf('private struct MediumSchedule'),
        source.indexOf('private struct DayHeading'),
      );

      expect(medium, contains('shortWeekText(day)'));
      expect(medium, contains('weekdayText(day, fallbackDate: fallbackDate)'));
      expect(
        medium,
        contains('shortDateText(day, fallbackDate: fallbackDate)'),
      );
      expect(source, contains('return day.courses.count <= 2'));
      expect(medium, contains('MediumSplitCourses'));
      expect(medium, contains('columns(visibleCount: 4)'));
      expect(medium, contains('columns(visibleCount: 3)'));
      expect(medium, contains('columns(visibleCount: 2)'));
      expect(medium, contains('tomorrow'));
      expect(source, isNot(contains('.frame(width: 68')));
    },
  );

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
    expect(nextClass, contains('course?.teacher?.let(::teacherDisplayName)'));
    expect(nextClass, contains('detailText("教师", "Teacher", it)'));
    expect(source, contains('generatedAt?.let(::updatedText)'));
    expect(source, contains('distanceText('));
    expect(data, contains('fun nextUpdateAt('));
    expect(source, contains('HomeWidgetScheduler.schedule('));
  });

  test('NextClass picker shares one full-canvas image across preview APIs', () {
    final preview = read(
      'android/app/src/main/res/layout/next_class_widget_preview.xml',
    );
    final provider = read(
      'android/app/src/main/res/xml/next_class_widget_info.xml',
    );

    expect(preview, contains('<ImageView'));
    expect(preview, contains('android:layout_width="match_parent"'));
    expect(preview, contains('android:layout_height="match_parent"'));
    expect(preview, contains('android:scaleType="fitCenter"'));
    expect(
      preview,
      contains('android:src="@drawable/next_class_widget_preview"'),
    );
    expect(preview, isNot(contains('<TextView')));
    expect(provider, contains('android:targetCellWidth="4"'));
    expect(provider, contains('android:minWidth="250dp"'));
    expect(provider, contains('android:minHeight="50dp"'));
    expect(
      provider,
      contains('android:previewLayout="@layout/next_class_widget_preview"'),
    );
    expect(
      provider,
      contains('android:previewImage="@drawable/next_class_widget_preview"'),
    );
    expect(
      File(
        'android/app/src/main/res/drawable-nodpi/next_class_widget_preview.png',
      ).existsSync(),
      isTrue,
    );
    expect(
      File(
        'android/app/src/main/res/drawable-night-nodpi/next_class_widget_preview.png',
      ).existsSync(),
      isTrue,
    );
  });

  test('Android course timeline is resizable, switchable and fully previewed', () {
    final source = read(
      'android/app/src/main/kotlin/tj/yourtj/forum_app/widget/ScheduleWidgets.kt',
    );
    final manifest = read('android/app/src/main/AndroidManifest.xml');
    final bridge = read('lib/src/campus_widget/schedule_widget_bridge.dart');
    final provider = read(
      'android/app/src/main/res/xml/course_timeline_widget_info.xml',
    );
    final preview = read(
      'android/app/src/main/res/layout/course_timeline_widget_preview.xml',
    );
    final timeline = source.substring(
      source.indexOf('class CourseTimelineWidget :'),
      source.indexOf('@Composable\nprivate fun TimelineHeader'),
    );
    final timelineCard = source.substring(
      source.indexOf('@Composable\nprivate fun TimelineCourseCard'),
      source.indexOf('class ToggleCourseTimelineDayAction'),
    );
    final timelineItem = source.substring(
      source.indexOf('@Composable\nprivate fun TimelineCourseListItem'),
      source.indexOf('@Composable\nprivate fun TimelineHeader'),
    );
    final locationIcon = read(
      'android/app/src/main/res/drawable/course_timeline_location.xml',
    );

    expect(manifest, contains('CourseTimelineWidgetReceiver'));
    expect(provider, contains('android:targetCellWidth="4"'));
    expect(provider, contains('android:targetCellHeight="3"'));
    expect(provider, contains('android:minWidth="250dp"'));
    expect(provider, contains('android:minHeight="180dp"'));
    expect(
      provider,
      contains(
        'android:previewLayout="@layout/course_timeline_widget_preview"',
      ),
    );
    expect(
      provider,
      contains(
        'android:previewImage="@drawable/course_timeline_widget_preview"',
      ),
    );
    expect(preview, contains('android:scaleType="fitCenter"'));
    expect(timeline, contains('LazyColumn'));
    expect(timeline, contains('items = day.courses'));
    expect(timeline, contains('TimelineCourseListItem(context, course)'));
    expect(timeline, contains('startSection'));
    expect(timeline, contains('endSection'));
    expect(timeline, contains('teacherDisplayName(course.teacher)'));
    expect(timelineItem, contains('TimelineCourseCard(context, course)'));
    expect(timelineItem, contains('Spacer(GlanceModifier.height(10.dp))'));
    expect(timelineCard, isNot(contains('padding(bottom = 8.dp)')));
    expect(
      timelineCard,
      contains('ImageProvider(R.drawable.course_timeline_location)'),
    );
    expect(timelineCard, isNot(contains('"地点"')));
    expect(locationIcon, contains('Icons.place_outlined'));
    expect(
      source,
      contains('actionRunCallback<ToggleCourseTimelineDayAction>()'),
    );
    expect(source, contains(r'course_timeline_show_tomorrow:$appWidgetId'));
    expect(bridge, contains('CourseTimelineWidgetReceiver'));
    expect(
      File(
        'android/app/src/main/res/drawable-nodpi/course_timeline_widget_preview.png',
      ).existsSync(),
      isTrue,
    );
    expect(
      File(
        'android/app/src/main/res/drawable-night-nodpi/course_timeline_widget_preview.png',
      ).existsSync(),
      isTrue,
    );
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
    expect(source, contains('containerBackground(for: .widget)'));
    expect(
      source,
      contains('ContainerRelativeShape().fill(.ultraThinMaterial)'),
    );
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
    // The iOS home_widget plugin only forwards links carrying this marker.
    expect(source, contains('yourtj://campus/today?homeWidget=true'));
    expect(source, contains('&homeWidget=true'));
    expect(source, contains('HStack(alignment: .top, spacing: 12)'));
    expect(source, isNot(contains('.teal')));
    expect(source, isNot(contains('.indigo')));
    for (final unsupportedApi in [
      'date: .now',
      'entry(at: .now)',
      '.foregroundStyle(',
      '.tint(',
      'in: ContainerRelativeShape()',
    ]) {
      expect(source, isNot(contains(unsupportedApi)));
    }
    expect(source, contains('lastUpdatedText('));
    expect(
      '.frame(maxWidth: .infinity'.allMatches(source).length,
      greaterThanOrEqualTo(2),
    );
    expect(source, contains('ViewThatFits(in: .vertical)'));
    expect(source, contains('courses.prefix(count)'));
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

  test('iOS 13 widget reload is a no-op so login cache clearing can finish', () {
    final plugin = read(
      '../../third_party/home_widget/ios/home_widget/Sources/home_widget/HomeWidgetPlugin.swift',
    );
    final reload = plugin.substring(
      plugin.indexOf('} else if call.method == "updateWidget" {'),
      plugin.indexOf('} else if call.method == "updateWidgetPreview" {'),
    );

    expect(reload, isNot(contains('Widgets are only available on iOS 14.0')));
    expect(reload, contains('result(false)'));
  });

  test('iOS debug widget inherits the Flutter app version', () {
    final project = read('ios/Runner.xcodeproj/project.pbxproj');
    final debugWidget = project.substring(
      project.indexOf('76000000000000000000000C /* Debug */ = {'),
      project.indexOf('76000000000000000000000D /* Release */ = {'),
    );
    final widgetConfig = read('ios/Flutter/WidgetDebug.xcconfig');

    expect(debugWidget, contains('WidgetDebug.xcconfig'));
    expect(widgetConfig, contains('#include "Generated.xcconfig"'));
  });

  test('development APK helper keeps all supported ABI splits', () {
    final script = read('../../scripts/build_dev_apk.sh');
    expect(script, contains('--split-per-abi'));
    expect(script, contains('app-*-debug.apk'));
  });
}
