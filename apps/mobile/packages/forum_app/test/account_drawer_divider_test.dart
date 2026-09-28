import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';
import 'mobile_navigation_test.dart' show accountCard;

/// Mounts the account drawer in a narrow phone viewport and opens it. The
/// viewport is tall enough that both groups are laid out without scrolling.
Future<void> _openDrawer(
  WidgetTester tester, {
  required bool signedIn,
  required Brightness brightness,
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = const Size(320, 1000);
  addTearDown(tester.view.reset);
  final layout = LayoutPayload.fromJson(minimalLayoutJson());
  final container = ProviderContainer(
    overrides: [
      accountLayoutProvider.overrideWith(
        (_) async => layout.copyWith(
          viewer: layout.viewer.copyWith(
            id: 1,
            username: 'viewer',
            isAuthenticated: signedIn,
          ),
        ),
      ),
      accountCardProvider(1).overrideWith((_) async => accountCard()),
    ],
  );
  addTearDown(container.dispose);
  final scaffoldKey = GlobalKey<ScaffoldState>();
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(brightness),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          key: scaffoldKey,
          drawer: const AccountDrawer(),
          body: const SizedBox.expand(),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  scaffoldKey.currentState!.openDrawer();
  await tester.pumpAndSettle();
}

void _expectSectionDivider(
  WidgetTester tester, {
  required String lastAccountEntry,
}) {
  final divider = find.descendant(
    of: find.byType(Drawer),
    matching: find.byKey(drawerSectionDividerKey),
  );
  expect(divider, findsOneWidget);
  final l10n = AppLocalizations.of(tester.element(find.byType(Drawer)));
  final rect = tester.getRect(divider);
  final accountRect = tester.getRect(find.text(lastAccountEntry));
  final settingsRect = tester.getRect(find.text(l10n.settingsTitle));
  expect(
    rect.top,
    greaterThan(accountRect.bottom),
    reason: 'divider starts below the $lastAccountEntry entry',
  );
  expect(
    rect.bottom,
    lessThan(settingsRect.top),
    reason: 'divider ends above the ${l10n.settingsTitle} entry',
  );
  expect(
    rect.height,
    1,
    reason: 'the hairline is actually visible; zero height would hide it',
  );
  final widget = tester.widget<GfDivider>(divider);
  expect(widget.inset, 24);
  expect(
    tester
        .widget<Divider>(
          find.descendant(of: divider, matching: find.byType(Divider)),
        )
        .color,
    GfTheme.colorsOf(tester.element(divider)).line,
  );
  // Decorative: the hairline owns no semantics, tap or focus target.
  expect(
    find.descendant(
      of: divider,
      matching: find.byWidgetPredicate(
        (widget) =>
            widget is Semantics || widget is Focus || widget is GestureDetector,
      ),
    ),
    findsNothing,
  );
}

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'guest drawer separates sign-in from the device entries (${brightness.name})',
      (tester) async {
        await _openDrawer(tester, signedIn: false, brightness: brightness);
        final l10n = AppLocalizations.of(tester.element(find.byType(Drawer)));
        _expectSectionDivider(tester, lastAccountEntry: l10n.loginModeLogin);
      },
    );

    testWidgets(
      'signed-in drawer separates the account entries from the rest (${brightness.name})',
      (tester) async {
        await _openDrawer(tester, signedIn: true, brightness: brightness);
        final l10n = AppLocalizations.of(tester.element(find.byType(Drawer)));
        _expectSectionDivider(
          tester,
          lastAccountEntry: l10n.myCourseReviewsTitle,
        );
      },
    );
  }
}
