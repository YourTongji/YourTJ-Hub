import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/schedule/schedule_storage_panel.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';

void main() {
  for (final brightness in Brightness.values) {
    testWidgets(
      'recovery shows origin and refuses another account in $brightness',
      (tester) async {
        late ScheduleStoreNotifier store;
        await tester.runAsync(() async {
          SharedPreferences.setMockInitialValues({
            'pk.plans': jsonEncode([
              {
                'id': 'legacy',
                'name': 'Private plan',
                'createdAt': 1,
                'stagedCourses': [],
                'selectedCourses': [],
                'customEvents': [],
              },
            ]),
            'pk.syncOwner': '8',
          });
          store = ScheduleStoreNotifier(site: 'https://dev.example');
          await store.ready;
        });

        tester.view.physicalSize = const Size(320, 900);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              scheduleStoreProvider.overrideWith((ref) => store),
              currentUserProvider.overrideWith(
                (ref) async => const CurrentUser(id: 7, username: 'alice'),
              ),
            ],
            child: MaterialApp(
              theme: ThemeData(brightness: brightness),
              home: const Scaffold(
                body: SingleChildScrollView(child: ScheduleStoragePanel()),
              ),
            ),
          ),
        );
        await tester.runAsync(() async {
          await Future<void>.delayed(const Duration(milliseconds: 30));
        });
        await tester.pumpAndSettle();
        expect(find.text('Legacy schedule data found'), findsOneWidget);
        expect(find.text('Export original data'), findsOneWidget);
        await tester.tap(find.text('Review and restore'));
        await tester.pumpAndSettle();
        expect(find.textContaining('https://dev.example'), findsOneWidget);
        expect(find.textContaining('alice · ID 7'), findsOneWidget);
        expect(find.textContaining('account ID 8'), findsOneWidget);
        expect(
          tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
          isNull,
        );
        await tester.tap(find.text('Keep for later'));
        await tester.pumpAndSettle();
        expect(await store.readUnassignedLegacyPlans(), isNotEmpty);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      },
    );
  }
}
