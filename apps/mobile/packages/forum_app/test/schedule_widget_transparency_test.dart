import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/pages/settings/schedule_widget_transparency.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

class _TransparencyBridge extends ScheduleWidgetBridge {
  int stored = 9;
  final calls = <int>[];
  final writes = <int, Completer<void>>{};

  @override
  Future<int> readTransparency() async => stored;

  @override
  Future<void> setTransparency(int value) async {
    calls.add(value);
    await writes.putIfAbsent(value, Completer<void>.new).future;
    stored = value;
  }
}

Future<void> _pump(WidgetTester tester, _TransparencyBridge bridge) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [scheduleWidgetBridgeProvider.overrideWithValue(bridge)],
      child: MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const Scaffold(body: ScheduleWidgetTransparency()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _change(WidgetTester tester, int value) async {
  final slider = tester.widget<Slider>(find.byType(Slider));
  slider.onChanged!(value.toDouble());
  slider.onChangeEnd!(value.toDouble());
  await tester.pump();
}

double _visible(WidgetTester tester) =>
    tester.widget<Slider>(find.byType(Slider)).value;

void main() {
  testWidgets(
    'latest failure rolls back to an earlier successful gesture and later saves still work',
    (tester) async {
      final bridge = _TransparencyBridge();
      await _pump(tester, bridge);
      await _change(tester, 5);
      await _change(tester, 12);
      bridge.writes[5]!.complete();
      await tester.pump();
      expect(bridge.stored, 5);
      expect(
        _visible(tester),
        12,
        reason: 'An older success must not replace a newer preview.',
      );
      bridge.writes[12]!.completeError(StateError('write failed'));
      await tester.pumpAndSettle();
      expect(_visible(tester), 5);
      expect(bridge.stored, 5);

      await _change(tester, 7);
      bridge.writes[7]!.complete();
      await tester.pumpAndSettle();
      expect(bridge.calls, [5, 12, 7]);
      expect(bridge.stored, 7);
      expect(_visible(tester), 7);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('gesture writes cannot finish out of order', (tester) async {
    final bridge = _TransparencyBridge();
    bridge.writes[12] = Completer<void>()..complete();
    await _pump(tester, bridge);
    await _change(tester, 5);
    await _change(tester, 12);
    expect(
      bridge.calls,
      [5],
      reason:
          'The newer write waits even when its platform result could arrive immediately.',
    );
    expect(bridge.stored, 9);
    bridge.writes[5]!.complete();
    await tester.pumpAndSettle();
    expect(bridge.calls, [5, 12]);
    expect(bridge.stored, 12);
    expect(_visible(tester), 12);

    await _change(tester, 15);
    bridge.writes[15]!.completeError(StateError('write failed'));
    await tester.pumpAndSettle();
    expect(_visible(tester), 12);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'submitted gesture writes finish after the settings widget is disposed',
    (tester) async {
      final bridge = _TransparencyBridge();
      await _pump(tester, bridge);
      await _change(tester, 5);
      await _change(tester, 12);
      await tester.pumpWidget(const SizedBox.shrink());
      bridge.writes[5]!.complete();
      await tester.pump();
      expect(bridge.calls, [5, 12]);
      bridge.writes[12]!.complete();
      await tester.pump();
      expect(bridge.stored, 12);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('an older failure does not interrupt a newer drag preview', (
    tester,
  ) async {
    final bridge = _TransparencyBridge();
    await _pump(tester, bridge);
    await _change(tester, 5);
    tester.widget<Slider>(find.byType(Slider)).onChanged!(12);
    await tester.pump();
    bridge.writes[5]!.completeError(StateError('write failed'));
    await tester.pumpAndSettle();
    expect(_visible(tester), 12);
    expect(bridge.stored, 9);
    tester.widget<Slider>(find.byType(Slider)).onChangeEnd!(12);
    await tester.pump();
    bridge.writes[12]!.complete();
    await tester.pumpAndSettle();
    expect(_visible(tester), 12);
    expect(bridge.stored, 12);
    expect(tester.takeException(), isNull);
  });
}
