import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/app_system_ui_overlay.dart';

void main() {
  void usePhoneViewport(WidgetTester tester) {
    tester.view.physicalSize = const Size(393, 852);
    tester.view.devicePixelRatio = 1;
    tester.view.viewPadding = const FakeViewPadding(top: 62, bottom: 34);
    tester.view.padding = const FakeViewPadding(top: 62, bottom: 34);
    addTearDown(tester.view.reset);
  }

  for (final appBar in [false, true]) {
    testWidgets(
      'a light route restores dark status icons after popping ${appBar ? 'an AppBar' : 'a cover region'}',
      (tester) async {
        usePhoneViewport(tester);
        final navigator = GlobalKey<NavigatorState>();
        await tester.pumpWidget(
          MaterialApp(
            navigatorKey: navigator,
            theme: ThemeData.light(),
            builder: (_, child) => AppSystemUiOverlay(child: child!),
            home: const Scaffold(body: Text('Home')),
          ),
        );
        expect(
          SystemChrome.latestStyle!.statusBarIconBrightness,
          Brightness.dark,
        );
        expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.light);

        unawaited(
          navigator.currentState!.push<void>(
            MaterialPageRoute<void>(
              builder: (_) => appBar
                  ? Scaffold(
                      appBar: AppBar(
                        systemOverlayStyle: SystemUiOverlayStyle.light,
                        title: const Text('Cover'),
                      ),
                    )
                  : const AnnotatedRegion<SystemUiOverlayStyle>(
                      value: SystemUiOverlayStyle.light,
                      child: Scaffold(body: Text('Cover')),
                    ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(
          SystemChrome.latestStyle!.statusBarIconBrightness,
          Brightness.light,
        );
        expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.dark);

        navigator.currentState!.pop();
        await tester.pumpAndSettle();
        expect(find.text('Home'), findsOneWidget);
        expect(
          SystemChrome.latestStyle!.statusBarIconBrightness,
          Brightness.dark,
        );
        expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.light);
      },
    );
  }

  testWidgets(
    'plain routes use white status icons in dark mode and follow theme changes',
    (tester) async {
      usePhoneViewport(tester);
      final mode = ValueNotifier(ThemeMode.dark);
      addTearDown(mode.dispose);
      await tester.pumpWidget(
        ValueListenableBuilder<ThemeMode>(
          valueListenable: mode,
          builder: (_, value, _) => MaterialApp(
            theme: ThemeData.light(),
            darkTheme: ThemeData.dark(),
            themeMode: value,
            builder: (_, child) => AppSystemUiOverlay(child: child!),
            home: const Scaffold(body: Text('Home')),
          ),
        ),
      );
      expect(
        SystemChrome.latestStyle!.statusBarIconBrightness,
        Brightness.light,
      );
      expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.dark);

      mode.value = ThemeMode.light;
      await tester.pumpAndSettle();
      expect(
        SystemChrome.latestStyle!.statusBarIconBrightness,
        Brightness.dark,
      );
      expect(SystemChrome.latestStyle!.statusBarBrightness, Brightness.light);
    },
  );
}
