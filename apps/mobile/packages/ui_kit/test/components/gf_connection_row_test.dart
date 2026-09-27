import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  for (final width in [320.0, 390.0]) {
    for (final scale in [1.0, 2.0]) {
      testWidgets('connections fit $width at $scale and isolate actions', (
        tester,
      ) async {
        tester.view.physicalSize = Size(width, 1000);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        var opens = 0;
        var follows = 0;
        await tester.pumpWidget(
          gfApp(
            MediaQuery(
              data: MediaQueryData(textScaler: TextScaler.linear(scale)),
              child: SingleChildScrollView(
                child: GfConnectionRow(
                  name: 'A long display name 用户姓名',
                  username: 'long_username',
                  avatarUrl: '',
                  bio: 'Building a campus community. 在同济分享日常。',
                  onTap: () => opens++,
                  action: GfFollowButton(
                    following: true,
                    label: '已关注',
                    onPressed: () => follows++,
                  ),
                ),
              ),
            ),
          ),
        );
        expect(tester.takeException(), isNull);
        await tester.tap(find.text('已关注'));
        expect(follows, 1);
        expect(opens, 0);
        await tester.tap(find.text('@long_username'));
        expect(opens, 1);
      });
    }
  }
}
