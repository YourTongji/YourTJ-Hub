import 'dart:ui' show Tristate;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets(
    'small icon visuals keep a 44px target and native keyboard action',
    (tester) async {
      var taps = 0;
      await tester.pumpWidget(
        gfApp(
          GfIconButton(
            symbol: 'heart',
            size: 28,
            tooltip: 'Like',
            onPressed: () => taps++,
          ),
        ),
      );
      final rect = tester.getRect(find.byType(GfIconButton));
      expect(rect.width, greaterThanOrEqualTo(44));
      expect(rect.height, greaterThanOrEqualTo(44));
      await tester.tapAt(rect.topLeft + const Offset(2, 2));
      expect(taps, 1);
      await tester.sendKeyEvent(LogicalKeyboardKey.tab);
      await tester.sendKeyEvent(LogicalKeyboardKey.enter);
      expect(taps, 2);
    },
  );

  testWidgets(
    'pill options have separate 44px targets and selected semantics',
    (tester) async {
      final semantics = tester.ensureSemantics();
      try {
        String? selected;
        await tester.pumpWidget(
          gfApp(
            GfPillSwitch<String>(
              options: const [
                GfPillOption(
                  label: 'Latest',
                  value: 'latest',
                  icon: Icons.schedule,
                ),
                GfPillOption(
                  label: 'Popular',
                  value: 'popular',
                  icon: Icons.trending_up,
                ),
              ],
              selected: 'latest',
              onSelected: (value) => selected = value,
            ),
          ),
        );
        final first = tester.getSemantics(find.text('Latest'));
        expect(first.flagsCollection.isSelected, Tristate.isTrue);
        final targets = find.descendant(
          of: find.byType(GfPillSwitch<String>),
          matching: find.byType(TextButton),
        );
        expect(targets, findsNWidgets(2));
        for (final target in targets.evaluate()) {
          expect(
            tester.getSize(find.byWidget(target.widget)).height,
            greaterThanOrEqualTo(44),
          );
        }
        await tester.tap(find.text('Popular'));
        expect(selected, 'popular');
      } finally {
        semantics.dispose();
      }
    },
  );

  testWidgets(
    'badge preserves an arbitrary supplied glyph without dropping it',
    (tester) async {
      await tester.pumpWidget(
        gfApp(const GfBadge(label: 'Official', icon: GfSymbol('badge-check'))),
      );
      expect(find.byType(GfSymbol), findsOneWidget);
      expect(
        tester.getSize(find.byType(GfBadge)).height,
        lessThanOrEqualTo(26),
      );
    },
  );

  testWidgets('select tag toggles accessibly and keeps a padded target', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    try {
      bool? selected;
      await tester.pumpWidget(
        gfApp(
          GfSelectTag(
            label: 'Question',
            selected: false,
            onChanged: (value) => selected = value,
          ),
        ),
      );
      expect(
        tester.getSize(find.byType(GfSelectTag)).height,
        greaterThanOrEqualTo(44),
      );
      expect(
        tester.getSemantics(find.text('Question')).flagsCollection.isSelected,
        Tristate.isFalse,
      );
      await tester.tap(find.text('Question'));
      expect(selected, isTrue);
    } finally {
      semantics.dispose();
    }
  });

  testWidgets(
    'empty states use quiet shared symbols and preserve legacy icons',
    (tester) async {
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(const GfEmpty(message: 'Nothing yet'), brightness: brightness),
        );
        await tester.pumpAndSettle();
        expect(tester.widget<GfSymbol>(find.byType(GfSymbol)).name, 'inbox');
        final tile = tester.widget<Container>(
          find.descendant(
            of: find.byType(GfEmpty),
            matching: find.byWidgetPredicate(
              (widget) =>
                  widget is Container && widget.decoration is BoxDecoration,
            ),
          ),
        );
        expect(
          (tile.decoration as BoxDecoration).color,
          GfColors.forBrightness(brightness).base200,
        );
        await tester.pumpWidget(
          gfApp(
            const GfEmpty(message: 'Search', symbol: 'search'),
            brightness: brightness,
          ),
        );
        expect(tester.widget<GfSymbol>(find.byType(GfSymbol)).name, 'search');
        await tester.pumpWidget(
          gfApp(
            const GfEmpty(message: 'Courses', icon: Icons.school_outlined),
            brightness: brightness,
          ),
        );
        expect(find.byIcon(Icons.school_outlined), findsOneWidget);
        await tester.pumpWidget(
          gfApp(
            const GfEmpty(message: 'Loading', loading: true),
            brightness: brightness,
          ),
        );
        expect(find.byType(GfLoadingIndicator), findsOneWidget);
        expect(find.byType(GfSymbol), findsNothing);
      });
    },
  );

  testWidgets('alert actions wrap instead of squeezing long labels', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 640);
    addTearDown(tester.view.reset);
    late BuildContext page;
    await tester.pumpWidget(
      gfApp(
        Builder(
          builder: (context) {
            page = context;
            return const SizedBox();
          },
        ),
      ),
    );
    final result = showGfAlertDialog<bool>(
      page,
      builder: (context) => GfAlertDialog(
        title: const Text('Confirm change'),
        content: const Text('Your content will remain available.'),
        actions: [
          GfButton(
            label: 'Keep editing this content',
            variant: GfButtonVariant.secondary,
            onPressed: () => Navigator.pop(context, false),
          ),
          GfButton(
            label: 'Confirm this change',
            onPressed: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(Dialog), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.tap(find.text('Confirm this change'));
    await tester.pumpAndSettle();
    expect(await result, isTrue);
  });
}
