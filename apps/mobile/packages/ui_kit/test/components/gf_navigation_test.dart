import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:tdesign_flutter/tdesign_flutter.dart' as td;
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  testWidgets('GfAppBar delegates navigation chrome to TDesign', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          appBar: GfAppBar(
            title: const Text('首页'),
            actions: <Widget>[
              GfIconButton(icon: Icons.settings, onPressed: () {}),
            ],
          ),
        ),
      ),
    );

    expect(find.byType(td.TNavBar), findsOneWidget);
    expect(find.text('首页'), findsOneWidget);
    expect(find.byIcon(Icons.settings), findsOneWidget);
  });

  testWidgets('GfBottomNavigation reports the selected TDesign tab', (
    tester,
  ) async {
    int selected = 0;
    await tester.pumpWidget(
      gfApp(
        GfBottomNavigation(
          currentIndex: selected,
          onSelected: (int value) => selected = value,
          items: const <GfBottomNavigationItem>[
            GfBottomNavigationItem(
              label: '首页',
              icon: Icons.home_outlined,
              selectedIcon: Icons.home,
            ),
            GfBottomNavigationItem(
              label: '我的',
              icon: Icons.person_outline,
              selectedIcon: Icons.person,
              badge: true,
            ),
          ],
        ),
      ),
    );

    expect(find.byType(td.TTabBar), findsOneWidget);
    await tester.tap(find.text('我的'));
    expect(selected, 1);
  });
}
