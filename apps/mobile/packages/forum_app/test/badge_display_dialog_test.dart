import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/badge_display_dialog.dart';

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
    Future<void> Function(List<String>) save,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => showDialog<bool>(
                context: context,
                builder: (_) => BadgeDisplayDialog(
                  badges: List.generate(6, badge),
                  selected: selected,
                  onSave: save,
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

  testWidgets('save preserves explicit badge order', (tester) async {
    List<String>? saved;
    await pump(tester, [badge(0), badge(1)], (codes) async {
      saved = codes;
    });
    final move = find.byTooltip('Move down').first;
    await tester.ensureVisible(move);
    await tester.tap(move);
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
    expect(
      tester
          .widget<CheckboxListTile>(find.byKey(const Key('badge-select-b5')))
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
