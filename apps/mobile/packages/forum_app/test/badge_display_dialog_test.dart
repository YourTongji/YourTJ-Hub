import 'dart:async';
import 'dart:ui' show SemanticsAction;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/badge_display_dialog.dart';
import 'package:ui_kit/ui_kit.dart';

UserBadgePayload badge(int i) => UserBadgePayload(
  code: 'b$i',
  type: 'system',
  grantMode: 'manual',
  name: 'Badge $i',
  description: '',
  iconType: 'image',
  iconKey: '',
  iconUrl: '',
  color: 'blue',
  level: 'normal',
  isEnabled: true,
  isWearable: true,
  sortOrder: i,
  source: 'manual',
  reason: '',
  grantedAt: '',
);
void main() {
  Future<void> pump(
    WidgetTester tester,
    List<UserBadgePayload> selected,
    Future<void> Function(List<String>) save, {
    String worn = '',
    Future<void> Function(String)? wear,
  }) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => Navigator.of(context).push<bool>(
                MaterialPageRoute(
                  builder: (_) => BadgeDisplayDialog(
                    badges: List.generate(6, badge),
                    selected: selected,
                    onSave: save,
                    wearableBadges: [badge(0), badge(1)],
                    wornBadgeCode: worn,
                    onWear: wear,
                  ),
                ),
              ),
              child: const Text('Edit'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
  }

  testWidgets('wear-only edits preserve the automatic profile display choice', (
    tester,
  ) async {
    final worn = <String>[];
    var displayWrites = 0;
    await pump(
      tester,
      [badge(0)],
      (_) async {
        displayWrites++;
      },
      worn: 'b0',
      wear: (code) async {
        worn.add(code);
      },
    );
    await tester.tap(find.byType(RadioListTile<String>).last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(worn, ['b1']);
    expect(displayWrites, 0);
  });

  testWidgets('wear and display stay drafts and retry only the failed step', (
    tester,
  ) async {
    final worn = <String>[];
    var displayAttempts = 0;
    await pump(
      tester,
      [badge(0)],
      (_) async {
        if (displayAttempts++ == 0) throw StateError('display unavailable');
      },
      worn: 'b0',
      wear: (code) async {
        worn.add(code);
      },
    );
    await tester.tap(find.byType(RadioListTile<String>).last);
    await tester.pumpAndSettle();
    final display = find.byKey(const Key('badge-select-b0'));
    await tester.scrollUntilVisible(
      display,
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(display);
    await tester.pumpAndSettle();
    expect(worn, isEmpty);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(worn, ['b1']);
    expect(displayAttempts, 1);
    expect(find.byType(BadgeDisplayDialog), findsOneWidget);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(worn, ['b1']);
    expect(displayAttempts, 2);
    expect(find.byType(BadgeDisplayDialog), findsNothing);
  });

  testWidgets(
    'badge reorder semantics describe values and freeze during save',
    (tester) async {
      final semantics = tester.ensureSemantics();
      final pending = Completer<void>();
      await pump(tester, [badge(0), badge(1)], (_) => pending.future);
      final l = AppLocalizations.of(
        tester.element(find.byType(BadgeDisplayDialog)),
      );
      final handles = find.byWidgetPredicate(
        (widget) =>
            widget is Semantics &&
            widget.properties.label == l.badgeDisplayReorder,
      );
      final first = tester.getSemantics(handles.first).getSemanticsData();
      expect(first.value, '1');
      expect(first.increasedValue, '2');
      expect(first.hasAction(SemanticsAction.increase), isTrue);
      expect(first.hasAction(SemanticsAction.decrease), isFalse);
      final last = tester.getSemantics(handles.last).getSemanticsData();
      expect(last.value, '2');
      expect(last.decreasedValue, '1');
      expect(last.hasAction(SemanticsAction.decrease), isTrue);
      expect(last.hasAction(SemanticsAction.increase), isFalse);
      await tester.tap(find.text('Save'));
      await tester.pump();
      final saving = tester.getSemantics(handles.first).getSemanticsData();
      expect(saving.hasAction(SemanticsAction.increase), isFalse);
      expect(saving.hasAction(SemanticsAction.decrease), isFalse);
      expect(tester.takeException(), isNull);
      pending.complete();
      await tester.pumpAndSettle();
      semantics.dispose();
    },
  );

  testWidgets('save preserves explicit badge order', (tester) async {
    List<String>? saved;
    await pump(tester, [badge(0), badge(1)], (codes) async {
      saved = codes;
    });
    expect(find.text('Badge 0'), findsOneWidget);
    expect(find.text('Badge 1'), findsOneWidget);
    final handle = find.byType(ReorderableDragStartListener).first;
    final gesture = await tester.startGesture(tester.getCenter(handle));
    await tester.pump();
    await gesture.moveBy(const Offset(0, 20));
    await tester.pump(const Duration(milliseconds: 100));
    await gesture.moveBy(const Offset(0, 130));
    await tester.pump(const Duration(milliseconds: 400));
    await gesture.up();
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(saved, ['b1', 'b0']);
  });
  testWidgets('max five selection and empty display survive failed save', (
    tester,
  ) async {
    List<String>? saved;
    var attempts = 0;
    await pump(tester, List.generate(5, badge), (codes) async {
      saved = codes;
      if (attempts++ == 0) throw StateError('offline');
    });
    await tester.scrollUntilVisible(
      find.byKey(const Key('badge-select-b5')),
      150,
      scrollable: find.byType(Scrollable).first,
    );
    expect(
      tester
          .widget<Checkbox>(find.byKey(const Key('badge-select-b5')))
          .onChanged,
      isNull,
    );
    for (var i = 0; i < 5; i++) {
      final target = find.byKey(Key('badge-select-b$i'));
      await tester.ensureVisible(target);
      await tester.tap(target);
      await tester.pumpAndSettle();
    }
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(saved, isEmpty);
    expect(find.byType(BadgeDisplayDialog), findsOneWidget);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(attempts, 2);
    expect(find.byType(BadgeDisplayDialog), findsNothing);
  });
}
