import 'dart:ui' show SemanticsRole;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

Widget motionApp(Widget child, {bool reduced = false}) => MaterialApp(
  theme: gfThemeData(Brightness.light),
  builder: (context, child) => MediaQuery(
    data: MediaQuery.of(context).copyWith(disableAnimations: reduced),
    child: child!,
  ),
  home: Scaffold(body: child),
);

void main() {
  testWidgets('changing reduced motion preserves state inside transitions', (
    tester,
  ) async {
    var reduced = false;
    late StateSetter update;
    await tester.pumpWidget(
      StatefulBuilder(
        builder: (_, setState) {
          update = setState;
          return motionApp(
            const GfFadeTransition(
              animation: AlwaysStoppedAnimation(1),
              child: TextField(),
            ),
            reduced: reduced,
          );
        },
      ),
    );
    await tester.enterText(find.byType(TextField), 'Keep my draft');
    update(() => reduced = true);
    await tester.pumpAndSettle();
    expect(find.text('Keep my draft'), findsOneWidget);
    update(() => reduced = false);
    await tester.pumpAndSettle();
    expect(find.text('Keep my draft'), findsOneWidget);
  });

  testWidgets(
    'fade and logical-pixel rise share progress and reverse continuously',
    (tester) async {
      final controller = AnimationController(
        vsync: tester,
        duration: GfMotion.layout,
      );
      addTearDown(controller.dispose);
      await tester.pumpWidget(
        motionApp(
          GfFadeTransition(
            key: const Key('transition'),
            animation: controller,
            child: const SizedBox(key: Key('surface'), width: 200, height: 400),
          ),
        ),
      );
      controller.forward();
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 60));
      double opacity() => tester
          .widget<Opacity>(
            find.descendant(
              of: find.byKey(const Key('transition')),
              matching: find.byType(Opacity),
            ),
          )
          .opacity;
      final before = opacity();
      final transform = tester.widget<Transform>(
        find.descendant(
          of: find.byKey(const Key('transition')),
          matching: find.byType(Transform),
        ),
      );
      expect(transform.transform.entry(1, 3), closeTo(6 * (1 - before), .0001));
      controller.reverse();
      await tester.pump();
      expect(opacity(), closeTo(before, .0001));
      await tester.pumpAndSettle();
      expect(opacity(), 0);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'reduced indeterminate progress never announces a fake percentage',
    (tester) async {
      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        motionApp(
          Semantics(
            key: const Key('loading'),
            label: 'Loading',
            child: const GfProgressIndicator(),
          ),
          reduced: true,
        ),
      );
      expect(
        tester
            .widget<CircularProgressIndicator>(
              find.byType(CircularProgressIndicator),
            )
            .value,
        isNotNull,
      );
      expect(
        tester
            .getSemantics(find.byKey(const Key('loading')))
            .getSemanticsData()
            .value,
        isEmpty,
      );
      await tester.pumpAndSettle();
      semantics.dispose();
    },
  );

  testWidgets('standalone reduced progress retains loading semantics', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      motionApp(const GfProgressIndicator(key: Key('progress')), reduced: true),
    );
    final data = tester
        .getSemantics(find.byKey(const Key('progress')))
        .getSemanticsData();
    expect(data.role, SemanticsRole.loadingSpinner);
    expect(data.value, isEmpty);
    await tester.pumpAndSettle();
    semantics.dispose();
  });

  testWidgets(
    'toggle feedback is local, bounded and never replays passive updates',
    (tester) async {
      var active = false;
      var calls = 0;
      late StateSetter update;
      await tester.pumpWidget(
        motionApp(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return GfActionFeedback(
                active: active,
                onPressed: () {
                  calls++;
                  setState(() => active = !active);
                },
                child: const SizedBox(key: Key('glyph'), width: 20, height: 20),
                builder: (activate, visual) =>
                    IconButton(onPressed: activate, icon: visual),
              );
            },
          ),
        ),
      );
      double scale() => tester
          .widget<Transform>(
            find
                .ancestor(
                  of: find.byKey(const Key('glyph')),
                  matching: find.byType(Transform),
                )
                .first,
          )
          .transform
          .entry(0, 0);
      update(() => active = true);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));
      expect(scale(), 1);
      update(() => active = false);
      await tester.pump();
      await tester.tap(find.byType(IconButton));
      expect(calls, 1); // The request is not delayed for the animation.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 110));
      expect(scale(), inInclusiveRange(1.01, 1.061));
      await tester.pump(const Duration(milliseconds: 130));
      expect(scale(), closeTo(1, .0001));
      expect(
        tester.getSize(find.byKey(const Key('glyph'))),
        const Size(20, 20),
      );
    },
  );

  testWidgets(
    'enabling reduced motion stops an active toggle without losing its action',
    (tester) async {
      var reduced = false;
      var calls = 0;
      late StateSetter update;
      await tester.pumpWidget(
        StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return motionApp(
              GfActionFeedback(
                active: false,
                onPressed: () => calls++,
                child: const Text('Like'),
                builder: (activate, visual) =>
                    TextButton(onPressed: activate, child: visual),
              ),
              reduced: reduced,
            );
          },
        ),
      );
      await tester.tap(find.text('Like'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 60));
      update(() => reduced = true);
      await tester.pump();
      final transform = tester.widget<Transform>(
        find
            .ancestor(of: find.text('Like'), matching: find.byType(Transform))
            .first,
      );
      expect(transform.transform.entry(0, 0), 1);
      await tester.tap(find.text('Like'));
      expect(calls, 2);
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'bottom sheets use the shared cadence and have a zero-motion path',
    (tester) async {
      for (final reduced in [false, true]) {
        late BuildContext host;
        await tester.pumpWidget(
          motionApp(
            Builder(
              builder: (context) {
                host = context;
                return const SizedBox();
              },
            ),
            reduced: reduced,
          ),
        );
        showGfBottomSheet<void>(host, builder: (_) => const Text('Sheet'));
        await tester.pump();
        final route = ModalRoute.of(tester.element(find.text('Sheet')))!;
        expect(
          route.transitionDuration,
          reduced ? Duration.zero : const Duration(milliseconds: 280),
        );
        expect(
          route.reverseTransitionDuration,
          reduced ? Duration.zero : const Duration(milliseconds: 220),
        );
        await tester.pumpAndSettle();
        final y = tester.getTopLeft(find.text('Sheet')).dy;
        expect(y, greaterThan(0));
        Navigator.of(tester.element(find.text('Sheet'))).pop();
        await tester.pumpAndSettle();
      }
    },
  );

  testWidgets('reducing motion during toast exit finishes removal', (
    tester,
  ) async {
    var reduced = false;
    late StateSetter update;
    late BuildContext host;
    await tester.pumpWidget(
      StatefulBuilder(
        builder: (_, setState) {
          update = setState;
          return motionApp(
            Builder(
              builder: (context) {
                host = context;
                return const SizedBox();
              },
            ),
            reduced: reduced,
          );
        },
      ),
    );
    showGfToast(host, 'Saved');
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Close'));
    await tester.pump();
    update(() => reduced = true);
    await tester.pumpAndSettle();
    expect(find.text('Saved'), findsNothing);
    showGfToast(host, 'Again');
    await tester.pump();
    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();
    expect(find.text('Again'), findsNothing);
  });

  testWidgets(
    'a replacement arriving during exit survives and stays interactive',
    (tester) async {
      late BuildContext host;
      await tester.pumpWidget(
        motionApp(
          Builder(
            builder: (context) {
              host = context;
              return const SizedBox();
            },
          ),
        ),
      );
      showGfToast(host, 'Old');
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Close'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 40));
      showGfToast(host, 'New');
      await tester.pumpAndSettle();
      expect(find.text('New'), findsOneWidget);
      expect(find.text('Old'), findsNothing);
      await tester.tap(find.byTooltip('Close'));
      await tester.pumpAndSettle();
      expect(find.text('New'), findsNothing);
    },
  );

  testWidgets('image navigation snaps with reduced motion', (tester) async {
    await tester.pumpWidget(
      motionApp(
        const GfImageViewer(
          images: ['https://example.test/a.png', 'https://example.test/b.png'],
        ),
        reduced: true,
      ),
    );
    await tester.tap(
      find.byWidgetPredicate(
        (widget) => widget is GfSymbol && widget.name == 'chevron-right',
      ),
    );
    await tester.pump();
    final pages = tester.widget<PageView>(find.byType(PageView));
    expect(pages.controller!.page, 1);
    expect(find.text('2 / 2'), findsOneWidget);
  });

  testWidgets(
    'programmatic reading jumps are instantaneous with reduced motion',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);
      late BuildContext host;
      await tester.pumpWidget(
        motionApp(
          Builder(
            builder: (context) {
              host = context;
              return ListView.builder(
                controller: controller,
                itemExtent: 80,
                itemCount: 50,
                itemBuilder: (_, i) => Text('$i'),
              );
            },
          ),
          reduced: true,
        ),
      );
      await GfMotion.scrollTo(host, controller, 400);
      expect(controller.offset, 400);
    },
  );
}
