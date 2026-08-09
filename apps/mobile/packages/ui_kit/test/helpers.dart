import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Wraps [child] in a `MaterialApp` themed with the Gf theme for
/// [brightness], so tests exercise the real design tokens.
Widget gfApp(Widget child, {Brightness brightness = Brightness.light}) {
  return MaterialApp(
    theme: gfThemeData(brightness),
    home: Scaffold(body: Center(child: child)),
  );
}

/// Runs [body] once per brightness so every component is exercised in both
/// light and dark mode.
Future<void> forEachBrightness(
  WidgetTester tester,
  Future<void> Function(WidgetTester tester, Brightness brightness) body,
) async {
  for (final Brightness brightness in <Brightness>[Brightness.light, Brightness.dark]) {
    await body(tester, brightness);
  }
}
