import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    for (final scale in [1.0, 2.0]) {
      testWidgets(
        'profile identity is compact without overlapping avatar or actions ($brightness, $scale)',
        (tester) async {
          tester.view.physicalSize = const Size(320, 1000);
          tester.view.devicePixelRatio = 1;
          addTearDown(tester.view.reset);
          await tester.pumpWidget(
            gfApp(
              MediaQuery(
                data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                child: SingleChildScrollView(
                  child: GfUserCard(
                    avatarUrl: '',
                    name: 'Alice',
                    username: 'alice',
                    bio: 'Campus life',
                    signature: 'Stay curious',
                    actions: OutlinedButton(
                      onPressed: () {},
                      style: OutlinedButton.styleFrom(
                        minimumSize: const Size(96, 44),
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 8,
                        ),
                        textStyle: const TextStyle(
                          fontSize: 14,
                          height: 1.4,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      child: const Text('Edit profile'),
                    ),
                  ),
                ),
              ),
              brightness: brightness,
            ),
          );
          final cover = tester.getRect(
            find.byKey(const Key('profile-cover-image')),
          );
          final name = tester.getRect(find.text('Alice'));
          final username = tester.getRect(find.text('@alice'));
          final bio = tester.getRect(find.text('Campus life'));
          final avatar = tester.getRect(find.byType(GfAvatar));
          final action = tester.getRect(find.byType(OutlinedButton));
          expect(cover.height, GfUserCard.coverHeightFor(320));
          // GfAvatar is inside a 4px outer ring on every side.
          expect(name.top, greaterThanOrEqualTo(avatar.bottom + 4 + 8));
          // The test font can wrap "Edit profile" even at 1x. Natural action
          // height must win over the 56px minimum instead of clipping the label.
          expect(
            name.top - cover.bottom,
            closeTo((action.height + 8).clamp(56, double.infinity), .01),
          );
          expect(username.top - name.bottom, closeTo(2, .01));
          expect(bio.top - username.bottom, closeTo(8, .01));
          expect(action.height, greaterThanOrEqualTo(44));
          expect(name.top, greaterThanOrEqualTo(action.bottom + 4));
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}
