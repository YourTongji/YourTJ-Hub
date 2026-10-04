import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_reply_dialog.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/realtime/realtime_updates.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Posts extends PostRepository {
  _Posts(super.client);
  bool fail = true;
  final attempts = <String>[];
  @override
  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    attempts.add(content);
    if (fail) throw const ApiException(fallbackMessage: 'Save failed');
    return UpdatePostResult(
      id: postId,
      content: content,
      renderedContent: '',
      updatedAt: '',
      pendingReview: true,
      checking: true,
    );
  }
}

void main() {
  test(
    'content invalidation preserves other counters and reconnect reconciles content',
    () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final notifier = container.read(realtimeInvalidationsProvider.notifier);
      notifier.chat(9);
      notifier.content();
      var state = container.read(realtimeInvalidationsProvider);
      expect(state.contentRevision, 1);
      expect(state.chatConvId, 9);
      notifier.resync();
      state = container.read(realtimeInvalidationsProvider);
      expect(state.contentRevision, 2);
      expect(state.chatConvId, 0);
    },
  );

  testWidgets('rejected reply edits survive failure and can be resubmitted', (
    tester,
  ) async {
    final repo = _Posts(
      GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showDialog<bool>(
                  context: context,
                  builder: (_) => const ContentReplyDialog(
                    item: UserContentItem(
                      id: 42,
                      contentType: 'post',
                      title: '',
                      content: 'Rejected body',
                      processStatus: 1,
                    ),
                  ),
                ),
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();
    expect(find.text('Rejected body'), findsOneWidget);
    await tester.enterText(find.byType(TextField), 'Corrected body');
    await tester.tap(find.widgetWithText(TextButton, 'Edit and resubmit'));
    await tester.pumpAndSettle();
    expect(find.text('Corrected body'), findsOneWidget);
    expect(find.byType(AlertDialog), findsOneWidget);
    repo.fail = false;
    await tester.tap(find.widgetWithText(TextButton, 'Edit and resubmit'));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(repo.attempts, ['Corrected body', 'Corrected body']);
    expect(tester.takeException(), isNull);
  });
}
