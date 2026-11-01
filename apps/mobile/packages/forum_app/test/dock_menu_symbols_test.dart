import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/navigation/tab_scroll_registry.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/widgets/campus_shortcuts.dart';

void main() {
  test('dock shortcut glyphs are distinct and never echo a destination', () {
    final l10n = lookupAppLocalizations(const Locale('zh'));
    final campus = [
      ...campusTabSymbols.values,
      for (final (symbol, _, _) in campusTools(l10n)) symbol,
    ];
    final notifications = [
      for (final (_, symbol) in notificationCategories) symbol,
    ];
    final all = [...campus, ...notifications];

    expect(all.toSet(), hasLength(all.length), reason: '$all');
    for (final destination in GfShellDestination.values) {
      expect(all, isNot(contains(destination.symbol)));
      expect(all, isNot(contains('${destination.symbol}-filled')));
    }
    for (final symbol in all) {
      expect(
        File('../ui_kit/assets/icons/$symbol.svg').existsSync(),
        isTrue,
        reason: symbol,
      );
    }
  });

  test('campus tools keep the glyphs of their landing shortcuts', () {
    final l10n = lookupAppLocalizations(const Locale('zh'));
    expect(
      {for (final (symbol, _, route) in campusTools(l10n)) route: symbol},
      {
        '/courses': 'book-open',
        '/schedule': 'calendar-days',
        '/wiki': 'file-text',
      },
    );
  });
}
