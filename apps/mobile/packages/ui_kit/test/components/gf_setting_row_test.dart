import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('custom setting avatars preserve their square dimensions', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: GfSettingRow(
            title: 'Identity',
            leading: const GfAvatar(src: '', size: 40),
            onTap: () {},
          ),
        ),
      ),
    );
    expect(tester.getSize(find.byType(GfAvatar)), const Size(40, 40));
    expect(tester.takeException(), isNull);
  });

  testWidgets('small custom artwork keeps settings titles on the icon column', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Column(
            children: [
              GfSettingRow(
                title: 'Standard',
                symbol: 'user-round',
                onTap: () {},
              ),
              GfSettingRow(
                title: 'Avatar',
                leading: const SizedBox(width: 20, height: 20),
                onTap: () {},
              ),
            ],
          ),
        ),
      ),
    );
    expect(
      tester.getTopLeft(find.text('Standard')).dx,
      tester.getTopLeft(find.text('Avatar')).dx,
    );
  });
  testWidgets('destructive setting reads as an action without a chevron', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: GfSettingRow(
            title: 'Close account',
            symbol: 'trash-2',
            destructive: true,
            onTap: () {},
          ),
        ),
      ),
    );
    expect(
      tester.widget<Text>(find.text('Close account')).style!.color,
      GfColors.light.error,
    );
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'chevron-right'),
      findsNothing,
    );
  });
}
