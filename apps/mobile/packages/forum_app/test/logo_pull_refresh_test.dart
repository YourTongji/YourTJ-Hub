import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/logo_pull_refresh.dart';
import 'package:ui_kit/ui_kit.dart';

class _Feed extends StatefulWidget {
  const _Feed({required this.onRefresh, this.onController});

  final Future<void> Function() onRefresh;
  final ValueChanged<LogoRefreshController>? onController;

  @override
  State<_Feed> createState() => _FeedState();
}

class _FeedState extends State<_Feed> with SingleTickerProviderStateMixin {
  late final controller = LogoRefreshController(vsync: this);

  @override
  void initState() {
    super.initState();
    widget.onController?.call(controller);
  }

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Column(
    children: [
      Center(child: RefreshingLogo(controller: controller)),
      Expanded(
        child: LogoPullRefresh(
          controller: controller,
          onRefresh: widget.onRefresh,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            children: const [SizedBox(height: 1800)],
          ),
        ),
      ),
    ],
  );
}

Widget _app(Widget child, TargetPlatform platform) => MaterialApp(
  theme: gfThemeData(Brightness.light).copyWith(platform: platform),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  locale: const Locale('zh'),
  home: Scaffold(body: child),
);

void main() {
  for (final platform in [TargetPlatform.android, TargetPlatform.iOS]) {
    testWidgets('$platform pull stretches the header, holds it, then settles', (
      tester,
    ) async {
      late LogoRefreshController controller;
      var calls = 0;
      final done = Completer<void>();
      await tester.pumpWidget(
        _app(
          _Feed(
            onController: (value) => controller = value,
            onRefresh: () {
              calls++;
              return done.future;
            },
          ),
          platform,
        ),
      );

      final gesture = await tester.startGesture(const Offset(200, 300));
      await gesture.moveBy(const Offset(0, 20));
      await tester.pump();
      await gesture.moveBy(const Offset(0, 40));
      await tester.pump();
      final partial = controller.stretch.value;
      expect(partial, greaterThan(0));
      expect(controller.phase.value, LogoRefreshPhase.pulling);
      // The finger draws the sheen onto the mark.
      expect(find.byType(ShaderMask), findsOneWidget);

      await gesture.moveBy(const Offset(0, 60));
      await tester.pump();
      expect(controller.phase.value, LogoRefreshPhase.armed);
      expect(controller.stretch.value, greaterThan(partial));
      expect(calls, 0, reason: 'only a release refreshes');

      await gesture.up();
      for (var frame = 0; frame < 40; frame++) {
        await tester.pump(const Duration(milliseconds: 16));
      }
      expect(calls, 1);
      expect(controller.phase.value, LogoRefreshPhase.refreshing);
      expect(controller.stretch.value, moreOrLessEquals(30, epsilon: 1));
      expect(controller.contentShift.value, greaterThan(0));

      done.complete();
      await tester.pumpAndSettle();
      expect(controller.phase.value, LogoRefreshPhase.idle);
      expect(controller.stretch.value, moreOrLessEquals(0, epsilon: 0.5));
      expect(find.byType(ShaderMask), findsNothing);
    });

    testWidgets('$platform short pull springs back without refreshing', (
      tester,
    ) async {
      late LogoRefreshController controller;
      var calls = 0;
      await tester.pumpWidget(
        _app(
          _Feed(
            onController: (value) => controller = value,
            onRefresh: () async => calls++,
          ),
          platform,
        ),
      );

      final gesture = await tester.startGesture(const Offset(200, 300));
      await gesture.moveBy(const Offset(0, 20));
      await tester.pump();
      await gesture.moveBy(const Offset(0, 30));
      await tester.pump();
      expect(controller.stretch.value, greaterThan(0));
      await gesture.up();
      await tester.pumpAndSettle();

      expect(calls, 0);
      expect(controller.phase.value, LogoRefreshPhase.idle);
      expect(controller.stretch.value, moreOrLessEquals(0, epsilon: 0.5));
    });
  }

  test('the header resists a long pull', () {
    expect(LogoRefreshController.stretchFor(0), 0);
    expect(LogoRefreshController.stretchFor(20), closeTo(17.1, 0.1));
    expect(
      LogoRefreshController.stretchFor(400),
      lessThan(LogoRefreshController.stretchFor(200) * 1.5),
    );
    expect(LogoRefreshController.stretchFor(10000), lessThan(120));
  });
}
