import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';
import '../helpers.dart';

void main() {
  testWidgets(
    'search clears once, retains focus and forwards keyboard submission',
    (tester) async {
      final controller = TextEditingController();
      addTearDown(controller.dispose);
      final changes = <String>[];
      final submitted = <String>[];
      await tester.pumpWidget(
        gfApp(
          GfSearchField(
            controller: controller,
            hintText: 'Find courses',
            clearLabel: 'Clear search',
            onChanged: changes.add,
            onSubmitted: submitted.add,
          ),
        ),
      );
      expect(find.byTooltip('Clear search'), findsNothing);
      await tester.enterText(find.byType(TextField), 'Calculus');
      await tester.testTextInput.receiveAction(TextInputAction.search);
      await tester.pump();
      expect(submitted, ['Calculus']);
      await tester.tap(find.byTooltip('Clear search'));
      await tester.pump();
      expect(controller.text, isEmpty);
      expect(changes, ['Calculus', '']);
      expect(
        tester.widget<TextField>(find.byType(TextField)).focusNode!.hasFocus,
        isTrue,
      );
      expect(find.byTooltip('Clear search'), findsNothing);
      controller.text = 'Programmatic search';
      await tester.pump();
      expect(find.byTooltip('Clear search'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      controller.text = 'Caller still owns controller';
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'clear override handles a search once and large text remains readable',
    (tester) async {
      var cleared = 0;
      var changed = 0;
      await forEachBrightness(tester, (tester, brightness) async {
        await tester.pumpWidget(
          gfApp(
            MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: SizedBox(
                width: 320,
                child: GfSearchField(
                  hintText: 'Konversationen durchsuchen',
                  clearLabel: 'Leeren',
                  onChanged: (_) => changed++,
                  onClear: () => cleared++,
                ),
              ),
            ),
            brightness: brightness,
          ),
        );
        await tester.enterText(find.byType(TextField), 'term');
        await tester.pump();
        await tester.tap(find.byTooltip('Leeren'));
        await tester.pump();
        expect(tester.takeException(), isNull);
      });
      expect(changed, 2);
      expect(cleared, 2);
    },
  );
}
