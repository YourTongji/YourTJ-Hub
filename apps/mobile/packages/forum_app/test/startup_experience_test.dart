import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/startup_experience.dart';
import 'package:forum_app/src/widgets/app_system_ui_overlay.dart';
import 'package:ui_kit/ui_kit.dart';

Widget startupApp({
  bool reducedMotion = false,
  ThemeMode mode = ThemeMode.light,
  Widget child = const Scaffold(body: Text('Ready')),
}) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  darkTheme: gfThemeData(Brightness.dark),
  themeMode: mode,
  builder: (context, route) => AppSystemUiOverlay(
    child: MediaQuery(
      data: MediaQuery.of(context).copyWith(disableAnimations: reducedMotion),
      child: StartupExperience(child: route!),
    ),
  ),
  home: child,
);

void expectRestoredSystemUi(WidgetTester tester) {
  final rootRegion = tester.widget<AnnotatedRegion<SystemUiOverlayStyle>>(
    find
        .descendant(
          of: find.byType(AppSystemUiOverlay),
          matching: find.byType(AnnotatedRegion<SystemUiOverlayStyle>),
        )
        .first,
  );
  expect(rootRegion.value.statusBarColor, Colors.transparent);
  expect(rootRegion.value.systemNavigationBarColor, Colors.transparent);
  final style = SystemChrome.latestStyle!;
  if (defaultTargetPlatform == TargetPlatform.iOS) {
    // iOS consumes statusBarBrightness; Android bar colors are not an iOS
    // platform contract and may be null in the last emitted style.
    expect(style.statusBarBrightness, Brightness.light);
  } else {
    expect(style.statusBarIconBrightness, Brightness.dark);
    expect(style.statusBarColor, Colors.transparent);
    expect(style.systemNavigationBarColor, Colors.transparent);
    expect(style.systemNavigationBarIconBrightness, Brightness.dark);
  }
}

void main() {
  for (final platform in [TargetPlatform.iOS, TargetPlatform.android]) {
    testWidgets(
      '$platform reveals the mounted application within its deadline',
      (tester) async {
        tester.view.physicalSize = const Size(393, 852);
        tester.view.devicePixelRatio = 1;
        tester.view.padding = const FakeViewPadding(top: 44, bottom: 24);
        addTearDown(tester.view.reset);
        var taps = 0;
        await tester.pumpWidget(
          startupApp(
            child: Scaffold(
              body: Center(
                child: TextButton(
                  onPressed: () => taps++,
                  child: const Text('Ready'),
                ),
              ),
            ),
          ),
        );
        final ready = tester.element(find.text('Ready'));
        expect(find.text('未济非终，皆有可能'), findsOneWidget);
        await tester.tap(find.text('Ready'), warnIfMissed: false);
        expect(taps, 0);
        await tester.pump(const Duration(milliseconds: 100));
        await tester.pump(const Duration(milliseconds: 900));
        await tester.pump(const Duration(milliseconds: 300));
        await tester.pump();
        expect(find.text('未济非终，皆有可能'), findsNothing);
        expect(tester.element(find.text('Ready')), same(ready));
        await tester.tap(find.text('Ready'));
        expect(taps, 1);
        await tester.pump();
        expectRestoredSystemUi(tester);
        expect(tester.takeException(), isNull);
      },
      variant: TargetPlatformVariant.only(platform),
    );
  }

  testWidgets(
    'reduced motion immediately exposes the application',
    (tester) async {
      await tester.pumpWidget(startupApp(reducedMotion: true));
      expect(find.text('未济非终，皆有可能'), findsNothing);
      expect(find.byType(Image), findsNothing);
      expect(
        tester
            .widget<AbsorbPointer>(
              find
                  .descendant(
                    of: find.byType(StartupExperience),
                    matching: find.byType(AbsorbPointer),
                  )
                  .first,
            )
            .absorbing,
        isFalse,
      );
      expect(tester.hasRunningAnimations, isFalse);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'enabling reduced motion during startup cancels the sequence',
    (tester) async {
      await tester.pumpWidget(startupApp());
      await tester.pumpWidget(startupApp(reducedMotion: true));
      await tester.pump(const Duration(milliseconds: 1500));
      expect(find.text('未济非终，皆有可能'), findsNothing);
      expect(tester.hasRunningAnimations, isFalse);
      expect(tester.takeException(), isNull);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );

  testWidgets(
    'desktop bypasses startup decoration',
    (tester) async {
      await tester.pumpWidget(startupApp());
      expect(find.text('未济非终，皆有可能'), findsNothing);
      expect(find.byType(Image), findsNothing);
      expect(tester.hasRunningAnimations, isFalse);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.linux),
  );

  testWidgets(
    'startup follows the theme and then yields system bars to the app',
    (tester) async {
      final mode = ValueNotifier(ThemeMode.dark);
      addTearDown(mode.dispose);
      await tester.pumpWidget(
        ValueListenableBuilder<ThemeMode>(
          valueListenable: mode,
          builder: (_, value, _) => startupApp(mode: value),
        ),
      );
      expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.dark);
      await tester.pump(const Duration(milliseconds: 1300));
      await tester.pump();
      mode.value = ThemeMode.light;
      await tester.pumpAndSettle();
      expect(find.text('未济非终，皆有可能'), findsNothing);
      expectRestoredSystemUi(tester);
      expect(tester.takeException(), isNull);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'disposal during the startup delay leaves no active sequence',
    (tester) async {
      await tester.pumpWidget(startupApp());
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 1500));
      expect(tester.takeException(), isNull);
    },
    variant: TargetPlatformVariant.only(TargetPlatform.android),
  );
}
