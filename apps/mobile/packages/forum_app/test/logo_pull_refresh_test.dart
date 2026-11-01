import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/logo_pull_refresh.dart';
import 'package:ui_kit/ui_kit.dart';

class _Feed extends StatefulWidget {
  const _Feed({required this.onRefresh, this.onController, this.shown});

  final Future<void> Function() onRefresh;
  final ValueChanged<LogoRefreshController>? onController;

  /// Unmounts the feed while false, as a cache clear does.
  final ValueListenable<bool>? shown;

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
        child: ValueListenableBuilder<bool>(
          valueListenable: widget.shown ?? const AlwaysStoppedAnimation(true),
          builder: (context, shown, _) => shown
              ? LogoPullRefresh(
                  controller: controller,
                  onRefresh: widget.onRefresh,
                  child: ListView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    children: const [SizedBox(height: 1800)],
                  ),
                )
              : const SizedBox(),
        ),
      ),
    ],
  );
}

Widget _app(Widget child, TargetPlatform platform, {bool reduced = false}) =>
    MaterialApp(
      theme: gfThemeData(Brightness.light).copyWith(platform: platform),
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(context).copyWith(disableAnimations: reduced),
        child: child!,
      ),
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

  testWidgets('reduced motion holds without travel and announces loading', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    final loading = RegExp(
      RegExp.escape(lookupAppLocalizations(const Locale('zh')).commonLoading),
    );
    late LogoRefreshController controller;
    final done = Completer<void>();
    await tester.pumpWidget(
      _app(
        _Feed(
          onController: (value) => controller = value,
          onRefresh: () => done.future,
        ),
        TargetPlatform.android,
        reduced: true,
      ),
    );

    final gesture = await tester.startGesture(const Offset(200, 300));
    for (final step in [20.0, 40.0, 60.0]) {
      await gesture.moveBy(Offset(0, step));
      await tester.pump();
    }
    expect(controller.phase.value, LogoRefreshPhase.armed);
    expect(find.bySemanticsLabel(loading), findsNothing);
    await gesture.up();
    await tester.pump();
    expect(controller.phase.value, LogoRefreshPhase.refreshing);
    expect(controller.stretch.value, 30, reason: 'no spring travel');
    // The mark's own name joins the announcement.
    expect(
      tester.getSemantics(find.bySemanticsLabel(loading)),
      isSemantics(isLiveRegion: true),
    );

    done.complete();
    await tester.pump();
    await tester.pump();
    expect(controller.phase.value, LogoRefreshPhase.idle);
    expect(controller.stretch.value, 0);
    expect(find.bySemanticsLabel(loading), findsNothing);
    semantics.dispose();
  });

  testWidgets(
    'a pull cut off by the feed unmounting settles without refreshing',
    (tester) async {
      late LogoRefreshController controller;
      var calls = 0;
      final shown = ValueNotifier(true);
      addTearDown(shown.dispose);
      await tester.pumpWidget(
        _app(
          _Feed(
            onController: (value) => controller = value,
            onRefresh: () async => calls++,
            shown: shown,
          ),
          TargetPlatform.android,
        ),
      );

      final gesture = await tester.startGesture(const Offset(200, 300));
      for (final step in [20.0, 40.0, 60.0]) {
        await gesture.moveBy(Offset(0, step));
        await tester.pump();
      }
      expect(controller.phase.value, LogoRefreshPhase.armed);

      shown.value = false;
      await tester.pumpAndSettle();
      expect(controller.phase.value, LogoRefreshPhase.idle);
      expect(controller.stretch.value, moreOrLessEquals(0, epsilon: 0.5));
      expect(controller.contentShift.value, moreOrLessEquals(0, epsilon: 0.5));
      await gesture.up();

      // A later touch that does not travel cannot fire the stale armed pull.
      shown.value = true;
      await tester.pumpAndSettle();
      await tester.dragFrom(const Offset(200, 300), const Offset(0, 4));
      await tester.pumpAndSettle();
      expect(calls, 0);
    },
  );

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
