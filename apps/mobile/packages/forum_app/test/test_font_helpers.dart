import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

/// Loads bundled fonts for realistic Chinese text layout and overflow tests.
/// Using explicit font bytes keeps layout assertions independent of host fonts.
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
