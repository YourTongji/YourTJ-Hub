import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/l10n/app_localizations.dart';

/// Loads the bundled Roboto + Noto Sans CJK + MaterialIcons fonts so golden
/// baselines render deterministically across platforms.
///
/// The Noto CJK subset is registered as an explicit fallback family so CJK
/// glyphs use the identical bundled font bytes on every host —
/// without it, macOS renders PingFang while CI (ubuntu) renders a different
/// fallback, producing pixel diffs for any Chinese text.
///
/// Must run inside `tester.runAsync` (real file IO never completes in the
/// FakeAsync test zone).
Future<void> loadTestFonts(WidgetTester tester) async {
  await tester.runAsync(() async {
    Future<ByteData> readFont(String path) async {
      final bytes = await File(path).readAsBytes();
      return ByteData.view(bytes.buffer);
    }

    final regular = readFont('test/assets/fonts/Roboto-Regular.ttf');
    final medium = readFont('test/assets/fonts/Roboto-Medium.ttf');
    final bold = readFont('test/assets/fonts/Roboto-Bold.ttf');
    final cjkRegular = readFont('test/assets/fonts/NotoSansCJKsc-Regular.otf');
    final cjkBold = readFont('test/assets/fonts/NotoSansCJKsc-Bold.otf');

    final loader = FontLoader('Roboto')
      ..addFont(regular)
      ..addFont(medium)
      ..addFont(bold);
    await loader.load();

    final cjk = FontLoader('NotoSansCJK')
      ..addFont(cjkRegular)
      ..addFont(cjkBold);
    await cjk.load();

    final icons = FontLoader('MaterialIcons')
      ..addFont(readFont('test/assets/fonts/MaterialIcons-Regular.otf'));
    await icons.load();
  });
}

/// Pumps [child] in a Gf-themed, localized [MaterialApp] on a fixed 390x844
/// mobile surface at DPR 1.0 with Roboto, ready for golden capture.
Future<void> pumpPageGolden(
  WidgetTester tester,
  Widget child, {
  Brightness brightness = Brightness.light,
  Locale locale = const Locale('zh'),
}) async {
  await loadTestFonts(tester);
  tester.view.devicePixelRatio = 1.0;
  tester.view.physicalSize = const Size(390, 844);
  addTearDown(tester.view.reset);

  await tester.pumpWidget(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      theme: gfThemeData(brightness).copyWith(
        textTheme: gfThemeData(
          brightness,
        ).textTheme.apply(
          fontFamily: 'Roboto',
          fontFamilyFallback: const <String>['NotoSansCJK'],
        ),
      ),
      home: Scaffold(
        backgroundColor: GfColors.forBrightness(brightness).base100,
        body: child,
      ),
    ),
  );
  await tester.pumpAndSettle();
}
