import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Loads the bundled Roboto + MaterialIcons fonts so golden baselines render
/// deterministically across platforms (tests otherwise use the blocky Ahem
/// font). Must run inside `tester.runAsync` — `testWidgets` runs in a
/// FakeAsync zone where real file IO / font loading never completes.
Future<void> loadTestFonts(WidgetTester tester) async {
  await tester.runAsync(() async {
    Future<ByteData> readFont(String path) async {
      final bytes = await File(path).readAsBytes();
      return ByteData.view(bytes.buffer);
    }

    final regular = readFont('test/assets/fonts/Roboto-Regular.ttf');
    final medium = readFont('test/assets/fonts/Roboto-Medium.ttf');
    final bold = readFont('test/assets/fonts/Roboto-Bold.ttf');

    final loader = FontLoader('Roboto')
      ..addFont(regular)
      ..addFont(medium)
      ..addFont(bold);
    await loader.load();

    final icons = FontLoader('MaterialIcons')
      ..addFont(readFont('test/assets/fonts/MaterialIcons-Regular.otf'));
    await icons.load();
  });
}

/// Pumps [child] in a Gf-themed [MaterialApp] on a fixed 390x844 mobile
/// surface at DPR 1.0, with Roboto applied, ready for golden capture.
Future<void> pumpGfGolden(
  WidgetTester tester,
  Widget child, {
  Brightness brightness = Brightness.light,
  Size surface = const Size(390, 844),
}) async {
  await loadTestFonts(tester);
  tester.view.devicePixelRatio = 1.0;
  tester.view.physicalSize = surface;
  addTearDown(tester.view.reset);

  await tester.pumpWidget(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: gfThemeData(brightness).copyWith(
        textTheme: gfThemeData(
          brightness,
        ).textTheme.apply(fontFamily: 'Roboto'),
      ),
      home: Scaffold(
        backgroundColor: GfColors.forBrightness(brightness).base100,
        body: child,
      ),
    ),
  );
  await tester.pumpAndSettle();
}
