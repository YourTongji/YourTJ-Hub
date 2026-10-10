import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'schedule_page_test.dart'
    show FakePkRepository, MemoryTokenStorage, makeContainer, wrapApp;

class PendingSearchRepository extends FakePkRepository {
  PendingSearchRepository()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'http://fake.local',
        ),
      ) {
    onSearchCourses = (query) {
      queries.add(query);
      final pending = Completer<PkSearchResult>();
      requests.add(pending);
      return pending.future;
    };
  }

  final queries = <String>[];
  final requests = <Completer<PkSearchResult>>[];
}

Future<void> openScheduleSearch(
  WidgetTester tester,
  PendingSearchRepository repository, {
  Brightness brightness = Brightness.light,
}) async {
  SharedPreferences.setMockInitialValues({});
  final notifier = ScheduleStoreNotifier();
  await notifier.ready;
  notifier.setMajorSelection(
    PkMajorSelection(
      calendarId: 119,
      grade: 2025,
      major: 'm1',
      majorName: '软件工程',
    ),
  );
  await notifier.flush;
  final container = makeContainer(notifier, repository: repository);
  addTearDown(container.dispose);
  await tester.pumpWidget(wrapApp(container, brightness: brightness));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('搜索'));
  await tester.tap(find.text('搜索'));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.byType(TextField));
}

PkSearchResult searchResult(String name) => PkSearchResult(
  courses: <PkSearchCourseItem>[
    PkSearchCourseItem(
      courseCode: '110001',
      courseName: name,
      faculty: '数学系',
      facultyI18n: '',
      credit: 4,
      campus: const <String>[],
      campusList: const <String>[],
      courseNature: const <String>[],
    ),
  ],
  sizeLimit: 100,
);

void expectSearchInput(WidgetTester tester, String text) {
  expect(find.byType(TextField), findsOneWidget);
  final editable = tester.widget<EditableText>(find.byType(EditableText));
  expect(editable.controller.text, text);
  expect(editable.focusNode.hasFocus, isTrue);
  expect(tester.testTextInput.isVisible, isTrue);
}

void main() {
  testWidgets(
    'debounced search keeps input and keyboard through loading',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.enterText(find.byType(TextField), '高');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pump();
      expect(repository.queries, ['高']);
      expectSearchInput(tester, '高');

      await tester.enterText(find.byType(TextField), '高等数学');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pump();
      expect(repository.queries, ['高', '高等数学']);
      repository.requests.last.complete(searchResult('高等数学（新结果）'));
      await tester.pumpAndSettle();
      repository.requests.first.complete(searchResult('高（旧结果）'));
      await tester.pumpAndSettle();
      expectSearchInput(tester, '高等数学');
      expect(find.text('高等数学（新结果）'), findsOneWidget);
      expect(find.text('高（旧结果）'), findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'Chinese composition waits for commit, including unchanged text',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.showKeyboard(find.byType(TextField));
      tester.testTextInput.updateEditingValue(
        const TextEditingValue(
          text: '高等数学',
          selection: TextSelection.collapsed(offset: 4),
          composing: TextRange(start: 0, end: 4),
        ),
      );
      await tester.pump(const Duration(milliseconds: 600));
      expect(repository.queries, isEmpty);
      expectSearchInput(tester, '高等数学');
      tester.testTextInput.updateEditingValue(
        const TextEditingValue(
          text: '高等数学',
          selection: TextSelection.collapsed(offset: 4),
        ),
      );
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pump();
      expect(repository.queries, ['高等数学']);
      repository.requests.single.complete(searchResult('高等数学'));
      await tester.pumpAndSettle();
      expectSearchInput(tester, '高等数学');
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'editing and clearing invalidate pending success and failure',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.enterText(find.byType(TextField), '高');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pump();
      await tester.enterText(find.byType(TextField), '高等数学');
      repository.requests.first.complete(searchResult('高（旧结果）'));
      await tester.pump();
      expect(find.text('高（旧结果）'), findsNothing);
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pump();
      await tester.tap(find.byTooltip('清空搜索'));
      await tester.pump();
      repository.requests.last.completeError(StateError('late failure'));
      await tester.pumpAndSettle();
      expectSearchInput(tester, '');
      expect(find.byType(GfErrorRetry), findsNothing);
      expect(find.byType(GfLoadingIndicator), findsNothing);
      expect(repository.queries, ['高', '高等数学']);
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'search failure and retry preserve text',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.enterText(find.byType(TextField), '高等数学');
      await tester.pump(const Duration(milliseconds: 350));
      repository.requests.single.completeError(StateError('network failure'));
      await tester.pumpAndSettle();
      expectSearchInput(tester, '高等数学');
      expect(find.byType(GfErrorRetry), findsOneWidget);
      tester.widget<GfErrorRetry>(find.byType(GfErrorRetry)).onRetry();
      await tester.pump();
      expectSearchInput(tester, '高等数学');
      expect(repository.queries, ['高等数学', '高等数学']);
      repository.requests.last.complete(searchResult('高等数学'));
      await tester.pumpAndSettle();
      expectSearchInput(tester, '高等数学');
      expect(find.byType(GfErrorRetry), findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'switching segments cancels debounce and retains query',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.enterText(find.byType(TextField), '高等数学');
      await tester.tap(find.text('必修'));
      await tester.pumpAndSettle();
      await tester.pump(const Duration(milliseconds: 400));
      expect(repository.queries, isEmpty);
      await tester.tap(find.text('搜索'));
      await tester.pump();
      expect(
        tester.widget<EditableText>(find.byType(EditableText)).controller.text,
        '高等数学',
      );
      expect(repository.queries, ['高等数学']);
      await tester.tap(find.text('必修'));
      await tester.pumpAndSettle();
      repository.requests.single.completeError(StateError('obsolete search'));
      await tester.pumpAndSettle();
      expect(find.byType(GfErrorRetry), findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );

  testWidgets(
    'keyboard submission cancels the scheduled duplicate search',
    (tester) async {
      final repository = PendingSearchRepository();
      await openScheduleSearch(tester, repository);
      await tester.enterText(find.byType(TextField), '高等数学');
      await tester.testTextInput.receiveAction(TextInputAction.search);
      await tester.pump();
      expect(repository.queries, ['高等数学']);
      await tester.pump(const Duration(milliseconds: 400));
      expect(repository.queries, ['高等数学']);
      repository.requests.single.complete(searchResult('高等数学'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<EditableText>(find.byType(EditableText)).controller.text,
        '高等数学',
      );
      await tester.pumpWidget(const SizedBox());
    },
    variant: TargetPlatformVariant.only(TargetPlatform.iOS),
  );
}
