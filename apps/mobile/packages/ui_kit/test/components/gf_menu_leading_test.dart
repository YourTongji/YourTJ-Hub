import 'package:flutter/cupertino.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

const _leadingKey = ValueKey('leading');
const _disabledKey = ValueKey('disabled-leading');
const _legacyKey = ValueKey('legacy-leading');

List<PopupMenuEntry<String>> _items() => const [
  GfPopupMenuItem(
    value: 'one',
    leading: GfSymbol('image', key: _leadingKey, size: 22),
    child: Text('Photo library'),
  ),
  GfPopupMenuItem(
    value: 'disabled',
    enabled: false,
    leading: GfSymbol('camera', key: _disabledKey, size: 22),
    child: Text('Unavailable'),
  ),
  GfPopupMenuItem(
    value: 'delete',
    destructive: true,
    leading: GfSymbol('trash-2', size: 22),
    child: Text('Remove'),
  ),
  GfPopupMenuItem(
    value: 'legacy',
    child: Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        GfSymbol('refresh-cw', key: _legacyKey, size: 18),
        SizedBox(width: 10),
        Flexible(child: Text('Legacy custom row')),
      ],
    ),
  ),
  GfPopupMenuItem(value: 'plain', child: Text('Plain item')),
];

void main() {
  test('leading is optional and does not replace the public child', () {
    const label = Text('Label');
    const icon = Icon(Icons.photo);
    const item = GfPopupMenuItem(value: 'one', leading: icon, child: label);
    expect(item.leading, same(icon));
    expect(item.child, same(label));
    expect(const GfPopupMenuItem(child: label).leading, isNull);
  });

  for (final platform in [TargetPlatform.android, TargetPlatform.iOS]) {
    for (final brightness in Brightness.values) {
      for (final width in [320.0, 1120.0]) {
        testWidgets(
          '$platform $brightness $width short-choice surface preserves items',
          (tester) async {
            late BuildContext host;
            await _pumpHost(tester, platform, brightness, width, (context) {
              host = context;
              return const SizedBox();
            });
            var completed = false;
            final result = showGfActionSheet<String>(
              host,
              initialValue: 'one',
              items: _items(),
            ).whenComplete(() => completed = true);
            await tester.pumpAndSettle();

            _expectContents(tester);
            expect(find.byType(CupertinoActionSheet), findsNothing);
            expect(
              ModalRoute.of(tester.element(find.text('Photo library'))),
              width < 760
                  ? isA<ModalBottomSheetRoute<String>>()
                  : isA<DialogRoute<String>>(),
            );
            final selected = tester.widget<ListTile>(
              find.widgetWithText(ListTile, 'Photo library'),
            );
            expect(selected.selected, isTrue);
            expect(selected.trailing, isNotNull);
            final removeContext = tester.element(find.text('Remove'));
            expect(
              DefaultTextStyle.of(removeContext).style.color,
              GfTheme.colorsOf(removeContext).error,
            );

            await tester.tap(find.text('Unavailable'));
            await tester.pumpAndSettle();
            expect(completed, isFalse);
            await tester.tap(find.text('Photo library'));
            await tester.pumpAndSettle();
            expect(await result, 'one');

            final cancelled = showGfActionSheet<String>(host, items: _items());
            await tester.pumpAndSettle();
            Navigator.of(host).pop();
            await tester.pumpAndSettle();
            expect(await cancelled, isNull);
            expect(find.byKey(_leadingKey), findsNothing);
            expect(tester.takeException(), isNull);
          },
        );
      }

      testWidgets('$platform $brightness anchored menu preserves leading', (
        tester,
      ) async {
        late BuildContext anchor;
        await _pumpHost(tester, platform, brightness, 1120, (context) {
          anchor = context;
          return const SizedBox(width: 40, height: 40);
        });
        var completed = false;
        final result = showGfMenu<String>(
          context: anchor,
          anchorContext: anchor,
          initialValue: 'one',
          items: _items(),
        ).whenComplete(() => completed = true);
        await tester.pumpAndSettle();
        _expectContents(tester);
        await tester.tap(find.text('Unavailable'));
        await tester.pumpAndSettle();
        expect(completed, isFalse);
        await tester.tap(find.text('Photo library'));
        await tester.pumpAndSettle();
        expect(await result, 'one');
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets('popup button renders leading once and retains selection', (
    tester,
  ) async {
    String? selected;
    await _pumpHost(tester, TargetPlatform.android, Brightness.light, 390, (_) {
      return GfPopupMenuButton<String>(
        itemBuilder: (_) => _items(),
        onSelected: (value) => selected = value,
        child: const Text('Open'),
      );
    });
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();
    _expectContents(tester);
    await tester.tap(find.text('Legacy custom row'));
    await tester.pumpAndSettle();
    expect(selected, 'legacy');
    expect(tester.takeException(), isNull);
  });
}

Future<void> _pumpHost(
  WidgetTester tester,
  TargetPlatform platform,
  Brightness brightness,
  double width,
  WidgetBuilder builder,
) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = Size(width, 900);
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    MaterialApp(
      theme: gfThemeData(brightness).copyWith(platform: platform),
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(context).copyWith(textScaler: TextScaler.linear(2)),
        child: child!,
      ),
      home: Scaffold(
        body: Align(
          alignment: Alignment.topLeft,
          child: Builder(builder: builder),
        ),
      ),
    ),
  );
}

void _expectContents(WidgetTester tester) {
  expect(find.byKey(_leadingKey), findsOneWidget);
  expect(find.byKey(_disabledKey), findsOneWidget);
  expect(find.byKey(_legacyKey), findsOneWidget);
  expect(find.text('Photo library'), findsOneWidget);
  expect(find.text('Plain item'), findsOneWidget);
  final icon = tester.getRect(find.byKey(_leadingKey));
  final label = tester.getRect(find.text('Photo library'));
  expect(icon.right, lessThan(label.left));
  expect(label.left - icon.right, GfSpace.md);
  expect(tester.takeException(), isNull);
}
