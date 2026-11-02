import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Shared surfaces stay Material on every platform (the Cupertino variants
/// introduced in #1170 were rolled back).
void main() {
  for (final platform in [TargetPlatform.iOS, TargetPlatform.android]) {
    testWidgets('$platform menu button opens a Material popup menu', (
      tester,
    ) async {
      String? value;
      await tester.pumpWidget(
        _app(
          Center(
            child: GfPopupMenuButton<String>(
              onSelected: (selected) => value = selected,
              itemBuilder: (_) => const [
                GfPopupMenuItem(value: 'one', child: Text('One')),
                GfPopupMenuItem(
                  value: 'delete',
                  destructive: true,
                  child: Text('Delete'),
                ),
              ],
            ),
          ),
          platform: platform,
        ),
      );
      expect(find.byType(PopupMenuButton<String>), findsOneWidget);
      await tester.tap(find.byType(GfPopupMenuButton<String>));
      await tester.pumpAndSettle();
      expect(find.byType(CupertinoActionSheet), findsNothing);
      expect(_menuItems, findsNWidgets(2));
      await tester.tap(find.text('One'));
      await tester.pumpAndSettle();
      expect(value, 'one');
      expect(tester.takeException(), isNull);
    });

    testWidgets('$platform anchored menu uses showMenu at the anchor', (
      tester,
    ) async {
      late BuildContext anchor;
      await tester.pumpWidget(
        _app(
          Align(
            alignment: Alignment.topRight,
            child: Builder(
              builder: (context) {
                anchor = context;
                return const SizedBox(width: 40, height: 40);
              },
            ),
          ),
          platform: platform,
        ),
      );
      final result = showGfMenu<String>(
        context: anchor,
        anchorContext: anchor,
        items: const [GfPopupMenuItem(value: 'one', child: Text('One'))],
      );
      await tester.pumpAndSettle();
      expect(find.byType(CupertinoActionSheet), findsNothing);
      expect(_menuItems, findsOneWidget);
      await tester.tap(find.text('One'));
      await tester.pumpAndSettle();
      expect(await result, 'one');
    });

    testWidgets('$platform action sheet is a Material bottom sheet', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 844);
      addTearDown(tester.view.reset);
      late BuildContext host;
      await tester.pumpWidget(
        _app(
          Builder(
            builder: (context) {
              host = context;
              return const SizedBox();
            },
          ),
          platform: platform,
        ),
      );
      final result = showGfActionSheet<String>(
        host,
        title: const Text('Title'),
        initialValue: 'one',
        items: const [
          GfPopupMenuItem(value: 'one', child: Text('One')),
          GfPopupMenuItem(value: 'two', child: Text('Two')),
        ],
      );
      await tester.pumpAndSettle();
      expect(find.byType(CupertinoActionSheet), findsNothing);
      expect(find.widgetWithText(ListTile, 'Two'), findsOneWidget);
      expect(
        find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'check'),
        findsOneWidget,
      );
      await tester.tap(find.text('Two'));
      await tester.pumpAndSettle();
      expect(await result, 'two');
    });

    testWidgets('$platform alert dialog keeps the Gf Material surface', (
      tester,
    ) async {
      late BuildContext host;
      await tester.pumpWidget(
        _app(
          Builder(
            builder: (context) {
              host = context;
              return const SizedBox();
            },
          ),
          platform: platform,
        ),
      );
      final result = showGfAlertDialog<bool>(
        host,
        builder: (context) => GfAlertDialog(
          title: const Text('Discard?'),
          actions: [
            GfDialogAction(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Keep'),
            ),
            GfDialogAction(
              destructive: true,
              onPressed: () => Navigator.pop(context, true),
              child: const Text('Discard'),
            ),
          ],
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CupertinoAlertDialog), findsNothing);
      expect(find.byType(CupertinoDialogAction), findsNothing);
      expect(find.byType(Dialog), findsOneWidget);
      expect(find.byType(GfDialogAction), findsNWidgets(2));
      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();
      expect(await result, isTrue);
    });

    testWidgets('$platform dropdown and radio stay Material', (tester) async {
      var mode = ThemeMode.light;
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (_, setState) => Column(
              children: [
                GfDropdownButtonFormField<String>(
                  initialValue: 'one',
                  items: const [
                    DropdownMenuItem(value: 'one', child: Text('One')),
                    DropdownMenuItem(value: 'two', child: Text('Two')),
                  ],
                  onChanged: (_) {},
                ),
                RadioGroup<ThemeMode>(
                  groupValue: mode,
                  onChanged: (value) => setState(() => mode = value!),
                  child: const Column(
                    children: [
                      GfRadioListTile(
                        value: ThemeMode.light,
                        title: Text('Light'),
                      ),
                      GfRadioListTile(
                        value: ThemeMode.dark,
                        title: Text('Dark'),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          platform: platform,
        ),
      );
      expect(find.byType(DropdownButtonFormField<String>), findsOneWidget);
      expect(find.byType(RadioListTile<ThemeMode>), findsNWidgets(2));
      expect(find.byType(CupertinoButton), findsNothing);
      await tester.tap(find.text('Dark'));
      await tester.pump();
      expect(mode, ThemeMode.dark);
    });
  }
}

Widget _app(Widget child, {required TargetPlatform platform}) => MaterialApp(
  theme: gfThemeData(Brightness.light).copyWith(platform: platform),
  home: Scaffold(body: child),
);

final _menuItems = find.byWidgetPredicate(
  (widget) => widget is PopupMenuItem<String>,
);
