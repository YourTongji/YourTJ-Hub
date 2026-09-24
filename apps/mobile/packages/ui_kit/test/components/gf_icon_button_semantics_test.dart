import 'dart:ui' show Tristate, SemanticsAction;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  for (final enabled in [true, false]) {
    testWidgets('icon action announces label and enabled state ($enabled)', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: GfIconButton(
              icon: Icons.arrow_back,
              tooltip: 'Back',
              onPressed: enabled ? () {} : null,
            ),
          ),
        ),
      );
      final node = tester.getSemantics(find.byType(GfIconButton));
      expect(node.label, 'Back');
      expect(node.flagsCollection.isButton, isTrue);
      expect(
        node.flagsCollection.isEnabled,
        enabled ? Tristate.isTrue : Tristate.isFalse,
      );
      expect(node.getSemanticsData().hasAction(SemanticsAction.tap), enabled);
      semantics.dispose();
    });
  }
}
