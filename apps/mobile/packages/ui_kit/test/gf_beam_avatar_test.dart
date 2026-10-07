import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Parity contract for the shared "beam" identity avatar.
///
/// Expected values below were produced by the Web/TypeScript implementation in
/// `apps/gooseforum/resource/src/site/utils/course-review-share.ts`
/// (`buildBeamAvatarDataUri` + `beamWrapper`, itself a port of
/// `boring-avatars@2.0.4` `variant="beam"`), so a Dart-only change to the
/// integer math, palette order or geometry fails here instead of silently
/// giving users two different avatars on Web and App.
void main() {
  test('shared identity palette matches the Web AVATAR_COLORS order', () {
    expect(beamColors, const <Color>[
      Color(0xFF0F172A),
      Color(0xFF38BDF8),
      Color(0xFFF8FAFC),
      Color(0xFFF59E0B),
      Color(0xFF22C55E),
    ]);
  });

  test('derived values match the Web beam port for a positive hash seed', () {
    final data = GfBeamAvatarData(seed: 'bob-1');

    expect(data.hash, 93907481);
    expect(data.wrapperColor, const Color(0xFF38BDF8));
    expect(data.faceColor, Colors.black);
    expect(data.backgroundColor, const Color(0xFF22C55E));
    expect(data.wrapperTranslateX, 3);
    expect(data.wrapperTranslateY, 3);
    expect(data.wrapperRotate, 41);
    expect(data.wrapperScale, closeTo(1.2, 1e-9));
    expect(data.isMouthOpen, isTrue);
    expect(data.isCircle, isTrue);
    expect(data.eyeSpread, 1);
    expect(data.mouthSpread, 2);
    expect(data.faceRotate, 1);
    expect(data.faceTranslateX, -1);
    expect(data.faceTranslateY, -3);
  });

  test('a seed whose 32-bit hash is negative stays identical to the Web', () {
    // 'alice-4' hashes to -914490329 before abs(); a missing toSigned(32)
    // (or a 64-bit Dart int) would change every derived value.
    final data = GfBeamAvatarData(seed: 'alice-4');

    expect(data.hash, 914490329);
    expect(data.wrapperColor, const Color(0xFF22C55E));
    expect(data.faceColor, Colors.black);
    expect(data.backgroundColor, const Color(0xFFF8FAFC));
    expect(data.wrapperTranslateX, -5);
    expect(data.wrapperTranslateY, 9);
    expect(data.wrapperRotate, 329);
    expect(data.wrapperScale, closeTo(1.2, 1e-9));
    expect(data.isMouthOpen, isFalse);
    expect(data.isCircle, isTrue);
    expect(data.eyeSpread, 4);
    expect(data.mouthSpread, 2);
    expect(data.faceRotate, -9);
    expect(data.faceTranslateX, -1);
    expect(data.faceTranslateY, 4.5);
  });

  test('an empty seed renders the shared anonymous face', () {
    final anonymous = GfBeamAvatarData(seed: '');
    final explicit = GfBeamAvatarData(seed: beamAnonymousSeed);

    expect(anonymous.hash, explicit.hash);
    expect(anonymous.hash, 656508733);
    expect(anonymous.wrapperColor, const Color(0xFFF59E0B));
    expect(anonymous.backgroundColor, const Color(0xFF38BDF8));
    expect(anonymous.faceColor, Colors.black);
    expect(anonymous.isCircle, isFalse);
    expect(anonymous.faceTranslateX, 3.5);
  });

  testWidgets('empty seed paints a circular avatar without throwing', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Center(child: GfBeamAvatar(seed: '', size: 40)),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(tester.getSize(find.byType(GfBeamAvatar)), const Size(40, 40));
    expect(find.byType(CustomPaint), findsWidgets);
  });
}
