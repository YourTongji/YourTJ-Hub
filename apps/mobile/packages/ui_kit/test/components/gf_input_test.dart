import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets('form fields use a quiet rounded surface in $brightness', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(const GfInput(labelText: 'Account'), brightness: brightness),
      );
      final context = tester.element(find.byType(TextField));
      final field = tester.widget<TextField>(find.byType(TextField));
      final decoration = field.decoration!.applyDefaults(
        Theme.of(context).inputDecorationTheme,
      );
      expect(
        WidgetStateProperty.resolveAs<Color>(decoration.fillColor!, {}),
        GfTheme.colorsOf(context).base300,
      );
      expect(
        decoration.border!.isOutline,
        isFalse,
        reason:
            'Filled labels stay inside the surface, without a border notch.',
      );
      for (final border in [
        decoration.enabledBorder,
        decoration.focusedBorder,
        decoration.errorBorder,
        decoration.focusedErrorBorder,
        decoration.disabledBorder,
      ]) {
        expect(
          (border! as UnderlineInputBorder).borderRadius,
          BorderRadius.circular(16),
        );
      }
      expect(decoration.constraints!.minHeight, greaterThanOrEqualTo(48));
      await tester.tap(find.byType(TextField));
      await tester.pumpAndSettle();
      final surfaces = tester.widgetList<AnimatedContainer>(
        find.descendant(
          of: find.byType(GfInput),
          matching: find.byType(AnimatedContainer),
        ),
      );
      expect(
        surfaces.where(
          (surface) =>
              surface.decoration is BoxDecoration &&
              ((surface.decoration! as BoxDecoration).boxShadow?.isNotEmpty ??
                  false),
        ),
        isEmpty,
      );
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('forwards native form properties and input callbacks', (
    tester,
  ) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    var taps = 0;
    String? submitted;
    await tester.pumpWidget(
      gfApp(
        GfInput(
          controller: controller,
          labelText: 'Account',
          autofillHints: const [AutofillHints.username],
          autocorrect: false,
          enableSuggestions: false,
          textCapitalization: TextCapitalization.characters,
          keyboardType: TextInputType.emailAddress,
          textInputAction: TextInputAction.done,
          textAlignVertical: TextAlignVertical.center,
          inputFormatters: [LengthLimitingTextInputFormatter(5)],
          onTap: () => taps++,
          onSubmitted: (value) => submitted = value,
        ),
      ),
    );
    final field = tester.widget<TextField>(find.byType(TextField));
    expect(field.autofillHints, [AutofillHints.username]);
    expect(field.autocorrect, isFalse);
    expect(field.enableSuggestions, isFalse);
    expect(field.textCapitalization, TextCapitalization.characters);
    expect(field.keyboardType, TextInputType.emailAddress);
    expect(field.textAlignVertical, TextAlignVertical.center);
    await tester.tap(find.byType(TextField));
    expect(taps, 1);
    await tester.enterText(find.byType(TextField), 'abcdef');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    expect(controller.text, 'abcde');
    expect(submitted, 'abcde');
  });

  testWidgets('switching between owned and supplied focus nodes is safe', (
    tester,
  ) async {
    final external = FocusNode();
    addTearDown(external.dispose);
    await tester.pumpWidget(gfApp(const GfInput()));
    await tester.tap(find.byType(TextField));
    await tester.pumpWidget(gfApp(GfInput(focusNode: external)));
    external.requestFocus();
    await tester.pumpAndSettle();
    expect(external.hasFocus, isTrue);
    await tester.pumpWidget(gfApp(const GfInput()));
    await tester.tap(find.byType(TextField));
    await tester.pumpAndSettle();
    expect(external.hasFocus, isFalse);
    expect(tester.takeException(), isNull);
  });
}
