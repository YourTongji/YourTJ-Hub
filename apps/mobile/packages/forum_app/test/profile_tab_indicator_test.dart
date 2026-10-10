import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/profile/profile_tabs.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('profile tab indicator stretches one edge first during a swipe', (
    tester,
  ) async {
    final progress = ValueNotifier(
      const GfTabSwipeProgress(originIndex: 0, offset: 0),
    );
    addTearDown(progress.dispose);
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: GfTabSwipeProgressScope(
            progress: progress,
            child: ProfileTabs(
              tabs: const [
                TabItemPayload(
                  key: 'following',
                  label: '关注',
                  url: '',
                  active: true,
                ),
                TabItemPayload(
                  key: 'followers',
                  label: '粉丝',
                  url: '',
                  active: false,
                ),
              ],
              index: 0,
              onChanged: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final indicator = find.byKey(const ValueKey('profile-tab-indicator'));
    final Rect resting = tester.getRect(indicator);

    progress.value = const GfTabSwipeProgress(
      originIndex: 0,
      offset: .3,
      targetIndex: 1,
    );
    await tester.pump();
    final Rect early = tester.getRect(indicator);
    // The edge facing the destination leads; the trailing edge lags.
    final double lead = early.right - resting.right;
    final double trail = early.left - resting.left;
    expect(lead, greaterThan(0));
    expect(trail, greaterThanOrEqualTo(0));
    expect(lead, greaterThan(trail * 3));

    progress.value = const GfTabSwipeProgress(
      originIndex: 0,
      offset: .9,
      targetIndex: 1,
    );
    await tester.pump();
    final Rect late = tester.getRect(indicator);
    // Near the end the trailing edge catches up and the line contracts.
    expect(late.left - early.left, greaterThan(late.right - early.right));
  });
}
