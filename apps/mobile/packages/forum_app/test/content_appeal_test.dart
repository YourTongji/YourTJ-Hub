import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

GfApiClient _client() => GfApiClient(
  dio: Dio(),
  tokenStorage: MemoryTokenStorage(),
  baseUrl: 'https://example.test',
);

class _Content extends ContentRepository {
  _Content(this.item) : super(_client());
  UserContentItem item;
  final appeals = <String>[];
  ApiException? failure;

  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) async => UserContentPage(
    items: contentType == item.contentType ? [item] : [],
    hasMore: false,
    nextCursorId: 0,
  );

  @override
  Future<void> submitAppeal(UserContentItem target, String reason) async {
    expect(target.revisionId, 70);
    if (failure != null) throw failure!;
    appeals.add(reason);
    item = UserContentItem(
      id: item.id,
      contentType: item.contentType,
      title: item.title,
      processStatus: 1,
      revisionId: 70,
      reviewReason: item.reviewReason,
      appeal: ModerationAppealOwnerSummary(
        status: 'pending',
        reason: reason,
        createdAt: DateTime.now().toUtc().toIso8601String(),
      ),
    );
  }
}

UserContentItem _rejected({
  bool canAppeal = true,
  ModerationAppealOwnerSummary? appeal,
}) => UserContentItem(
  id: 7,
  contentType: 'topic',
  title: 'Rejected topic',
  content: 'Rejected body',
  processStatus: 1,
  revisionId: 70,
  reviewReason: 'Looks like spam',
  canAppeal: canAppeal,
  appeal: appeal,
);

Future<void> _pump(WidgetTester tester, _Content repo) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [contentRepositoryProvider.overrideWithValue(repo)],
      child: MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ContentPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('author appeals a rejected topic from its review sheet', (
    tester,
  ) async {
    final repo = _Content(_rejected());
    await _pump(tester, repo);
    expect(find.text('Not approved'), findsOneWidget);
    expect(find.text('Looks like spam'), findsOneWidget);
    await tester.tap(find.text('Rejected topic'));
    await tester.pumpAndSettle();
    expect(find.text('Review result'), findsOneWidget);
    expect(find.text('Reviewer’s reason'), findsOneWidget);
    await tester.tap(find.byKey(const Key('appeal-start')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('appeal-form')), findsOneWidget);
    expect(find.text('Each version can be appealed once.'), findsOneWidget);
    final submit = find.byKey(const Key('appeal-submit'));
    expect(tester.widget<GfButton>(submit).onPressed, isNull);
    await tester.enterText(find.byType(TextField), '  It quotes the handout  ');
    await tester.pump();
    expect(find.text('25/512'), findsOneWidget);
    await tester.tap(submit);
    await tester.pumpAndSettle();
    expect(repo.appeals, ['It quotes the handout']);
    expect(find.byKey(const Key('appeal-form')), findsNothing);
    expect(find.text('Appeal under review'), findsOneWidget);
  });

  testWidgets('submission failures keep the reason and show the cause', (
    tester,
  ) async {
    final repo = _Content(_rejected())
      ..failure = const ApiException(
        fallbackMessage: 'duplicate',
        messageCode: 'content.appeal.duplicate',
      );
    await _pump(tester, repo);
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Appeal'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Context');
    await tester.pump();
    await tester.tap(find.byKey(const Key('appeal-submit')));
    await tester.pumpAndSettle();
    expect(
      find.textContaining('You’ve already appealed this version'),
      findsOneWidget,
    );
    expect(find.text('Context'), findsOneWidget);
  });

  testWidgets('decided appeals explain the outcome and hide the entry', (
    tester,
  ) async {
    final repo = _Content(
      _rejected(
        canAppeal: false,
        appeal: const ModerationAppealOwnerSummary(
          status: 'rejected',
          reason: 'Context',
          reviewReason: 'Still breaks the rules',
          createdAt: '2026-10-10T00:00:00Z',
          reviewedAt: '2026-10-11T00:00:00Z',
        ),
      ),
    );
    await _pump(tester, repo);
    expect(find.text('Appeal not approved'), findsOneWidget);
    await tester.tap(find.text('Rejected topic'));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('appeal-start')), findsNothing);
    expect(find.text('Still breaks the rules'), findsOneWidget);
    expect(find.textContaining('can’t be appealed again'), findsOneWidget);
  });

  testWidgets('editing during a pending appeal asks first', (tester) async {
    final repo = _Content(
      _rejected(
        canAppeal: false,
        appeal: const ModerationAppealOwnerSummary(
          status: 'pending',
          reason: 'Context',
          createdAt: '2026-10-10T00:00:00Z',
        ),
      ),
    );
    await _pump(tester, repo);
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    expect(find.text('Appeal'), findsNothing);
    await tester.tap(find.text('Edit and resubmit'));
    await tester.pumpAndSettle();
    expect(
      find.text('Editing and resubmitting will close the appeal under review.'),
      findsOneWidget,
    );
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(find.text('Rejected topic'), findsOneWidget);
  });
}
