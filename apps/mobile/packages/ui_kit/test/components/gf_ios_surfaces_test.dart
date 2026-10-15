import 'dart:ui' show SemanticsAction;

import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  testWidgets('iOS short confirmation has Cupertino actions and safe cancel', (
    tester,
  ) async {
    late BuildContext page;
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(
          Brightness.light,
        ).copyWith(platform: TargetPlatform.iOS),
        home: Scaffold(
          body: Builder(
            builder: (context) {
              page = context;
              return const SizedBox();
            },
          ),
        ),
      ),
    );
    final result = showGfAlertDialog<bool>(
      page,
      builder: (context) => GfAlertDialog(
        title: const Text('Delete draft?'),
        content: const Text('This cannot be undone.'),
        actions: [
          GfButton(
            label: 'Cancel',
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(context, false),
          ),
          GfButton(
            label: 'Delete',
            variant: GfButtonVariant.danger,
            onPressed: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(CupertinoAlertDialog), findsOneWidget);
    expect(
      tester
          .widget<CupertinoDialogAction>(
            find.widgetWithText(CupertinoDialogAction, 'Delete'),
          )
          .isDestructiveAction,
      isTrue,
    );
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(await result, isFalse);
    expect(tester.takeException(), isNull);
  });
  testWidgets('iOS menu keeps disabled, destructive and cancel semantics', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    final semantics = tester.ensureSemantics();

    String? value;
    await tester.pumpWidget(
      _app(
        Center(
          child: GfPopupMenuButton<String>(
            onSelected: (selected) => value = selected,
            itemBuilder: (_) => const [
              GfPopupMenuItem(
                value: 'disabled',
                enabled: false,
                child: Text('Unavailable'),
              ),
              GfPopupMenuItem(
                value: 'delete',
                destructive: true,
                child: Text('Delete'),
              ),
            ],
          ),
        ),
      ),
    );
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    expect(find.byType(CupertinoActionSheet), findsOneWidget);
    expect(
      tester
          .getSemantics(find.text('Unavailable'))
          .getSemanticsData()
          .hasAction(SemanticsAction.tap),
      isFalse,
    );
    expect(
      tester
          .widget<CupertinoActionSheetAction>(
            find.widgetWithText(CupertinoActionSheetAction, 'Delete'),
          )
          .isDestructiveAction,
      isTrue,
    );
    await tester.tap(find.text('Unavailable'), warnIfMissed: false);
    await tester.pump();
    expect(find.byType(CupertinoActionSheet), findsOneWidget);
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();
    expect(find.byType(CupertinoActionSheet), findsNothing);
    expect(value, isNull);
    semantics.dispose();
  });

  for (final brightness in Brightness.values) {
    testWidgets(
      'iPad menu is anchored, scrollable and survives resize $brightness',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(1024, 768);
        addTearDown(tester.view.reset);
        int? value;
        await tester.pumpWidget(
          _app(
            Align(
              alignment: Alignment.topRight,
              child: Padding(
                padding: const EdgeInsets.only(top: 100, right: 80),
                child: GfPopupMenuButton<int>(
                  onSelected: (selected) => value = selected,
                  itemBuilder: (_) => [
                    for (var i = 0; i < 40; i++)
                      GfPopupMenuItem(value: i, child: Text('Option $i')),
                  ],
                ),
              ),
            ),
            brightness: brightness,
            scale: 2,
          ),
        );
        final trigger = tester.getRect(find.byType(GfPopupMenuButton<int>));
        await tester.tap(find.byType(GfPopupMenuButton<int>));
        await tester.pumpAndSettle();
        final rect = tester.getRect(find.byType(CupertinoActionSheet));
        expect(rect.width, lessThanOrEqualTo(380));
        expect(rect.right, lessThanOrEqualTo(1012));
        expect(rect.top, greaterThanOrEqualTo(12));
        expect((rect.left - trigger.left).abs(), lessThan(380));
        await tester.drag(find.text('Option 0'), const Offset(0, -1600));
        await tester.pumpAndSettle();
        expect(find.text('Option 0').hitTestable(), findsNothing);
        tester.view.physicalSize = const Size(500, 768);
        await tester.pumpAndSettle();
        expect(
          tester.getRect(find.byType(CupertinoActionSheet)).width,
          lessThanOrEqualTo(500),
        );
        expect(tester.takeException(), isNull);
        await tester.tap(find.text('Cancel'));
        await tester.pumpAndSettle();
        expect(value, isNull);
      },
    );
  }

  testWidgets('Android retains popup presentation and selected values', (
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
            ],
          ),
        ),
        platform: TargetPlatform.android,
      ),
    );
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    expect(find.byType(CupertinoActionSheet), findsNothing);
    await tester.tap(find.text('One'));
    await tester.pumpAndSettle();
    expect(value, 'one');
  });

  for (final platform in [TargetPlatform.iOS, TargetPlatform.android]) {
    for (final useRootNavigator in [false, true]) {
      testWidgets(
        '$platform popup respects useRootNavigator=$useRootNavigator',
        (tester) async {
          final root = GlobalKey<NavigatorState>();
          final nested = GlobalKey<NavigatorState>();
          String? selected;
          await tester.pumpWidget(
            MaterialApp(
              navigatorKey: root,
              theme: gfThemeData(Brightness.light).copyWith(platform: platform),
              home: Navigator(
                key: nested,
                onGenerateRoute: (_) => MaterialPageRoute<void>(
                  builder: (_) => Scaffold(
                    body: Center(
                      child: GfPopupMenuButton<String>(
                        useRootNavigator: useRootNavigator,
                        onSelected: (value) => selected = value,
                        itemBuilder: (_) => const [
                          GfPopupMenuItem(value: 'one', child: Text('One')),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.tap(find.byType(GfPopupMenuButton<String>));
          await tester.pumpAndSettle();
          expect(
            Navigator.of(tester.element(find.text('One'))),
            same(useRootNavigator ? root.currentState : nested.currentState),
          );
          await tester.tap(find.text('One'));
          await tester.pumpAndSettle();
          expect(selected, 'one');
          expect(nested.currentState!.canPop(), isFalse);
          expect(root.currentState!.canPop(), isFalse);
          expect(tester.takeException(), isNull);
        },
      );
    }
  }

  testWidgets('iOS radio group updates checked rows and selection semantics', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    var selected = 'one';
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (_, setState) => RadioGroup<String>(
            groupValue: selected,
            onChanged: (value) => setState(() => selected = value!),
            child: const Column(
              children: [
                GfRadioListTile(value: 'one', title: Text('One')),
                GfRadioListTile(value: 'two', title: Text('Two')),
              ],
            ),
          ),
        ),
      ),
    );
    expect(find.byIcon(CupertinoIcons.check_mark), findsOneWidget);
    expect(
      tester.getSemantics(find.text('One')),
      matchesSemantics(
        label: 'One',
        isButton: true,
        hasEnabledState: true,
        isEnabled: true,
        hasSelectedState: true,
        hasCheckedState: true,
        isChecked: true,
        isInMutuallyExclusiveGroup: true,
        hasTapAction: true,
        isFocusable: true,
        hasFocusAction: true,
      ),
    );
    await tester.tap(find.text('Two'));
    await tester.pump();
    expect(selected, 'two');
    expect(find.byIcon(CupertinoIcons.check_mark), findsOneWidget);
    expect(
      find.descendant(
        of: find.widgetWithText(ListTile, 'Two'),
        matching: find.byIcon(CupertinoIcons.check_mark),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
    semantics.dispose();
  });

  testWidgets('iOS nullable choice distinguishes selecting null from cancel', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    final calls = <int?>[];
    await tester.pumpWidget(
      _app(
        Center(
          child: GfDropdownButton<int?>(
            value: 1,
            items: const [
              DropdownMenuItem(value: null, child: Text('All weeks')),
              DropdownMenuItem(value: 1, child: Text('Week 1')),
            ],
            onChanged: calls.add,
          ),
        ),
      ),
    );
    await tester.tap(find.text('Week 1'));
    await tester.pumpAndSettle();
    expect(find.byIcon(CupertinoIcons.check_mark), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(calls, isEmpty);
    await tester.tap(find.text('Week 1'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('All weeks'));
    await tester.pumpAndSettle();
    expect(calls, [null]);
  });

  testWidgets(
    'iOS form choices validate, reset and follow external initial values',
    (tester) async {
      final form = GlobalKey<FormState>();
      var initial = 'one';
      late StateSetter update;
      String? saved;
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (_, setState) {
              update = setState;
              return Form(
                key: form,
                child: GfDropdownButtonFormField<String>(
                  initialValue: initial,
                  items: const [
                    DropdownMenuItem(value: 'one', child: Text('One')),
                    DropdownMenuItem(value: 'two', child: Text('Two')),
                  ],
                  onChanged: (_) {},
                  onSaved: (value) => saved = value,
                  validator: (value) => value == 'two' ? null : 'Choose two',
                ),
              );
            },
          ),
        ),
      );
      expect(form.currentState!.validate(), isFalse);
      await tester.pump();
      expect(find.text('Choose two'), findsOneWidget);
      update(() => initial = 'two');
      await tester.pump();
      expect(find.text('Two'), findsOneWidget);
      expect(form.currentState!.validate(), isTrue);
      form.currentState!.save();
      expect(saved, 'two');
      form.currentState!.reset();
      await tester.pump();
      expect(find.text('Two'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}

Widget _app(
  Widget child, {
  TargetPlatform platform = TargetPlatform.iOS,
  Brightness brightness = Brightness.light,
  double scale = 1,
}) => MaterialApp(
  theme: gfThemeData(brightness).copyWith(platform: platform),
  builder: (context, child) => MediaQuery(
    data: MediaQuery.of(context).copyWith(textScaler: TextScaler.linear(scale)),
    child: child!,
  ),
  home: Scaffold(body: child),
);
