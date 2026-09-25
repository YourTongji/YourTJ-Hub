import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/private_notes.dart';
import 'package:forum_app/src/providers.dart';

const note = PrivateNotePayload(
  targetUserId: 2,
  username: 'alice',
  note: 'Lab partner',
);

class MemoryTokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

class NotesRepository extends UserRepository {
  NotesRepository()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokens()));
  bool fail = true;
  final saved = <String>[];
  @override
  Future<void> setPrivateNote(int targetUserId, String note) async {
    if (fail) throw Exception('offline');
    saved.add(note);
  }
}

Widget app(Widget child) => MaterialApp(
  locale: const Locale('en'),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);
void main() {
  testWidgets('pending or failed notes cannot open an empty editor', (
    tester,
  ) async {
    final pending = Completer<Map<int, PrivateNotePayload>>();
    var attempt = 0;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: 'owner'),
          ),
          privateNotesProvider((1, 0)).overrideWith((ref) {
            attempt++;
            return attempt == 1 ? pending.future : Future.value({2: note});
          }),
        ],
        child: app(
          const PrivateNotesHost(
            child: PrivateNoteButton(userId: 2, username: 'alice'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<TextButton>(find.widgetWithText(TextButton, 'Edit note'))
          .onPressed,
      isNull,
    );
    pending.completeError(Exception('offline'));
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsNothing);
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit note'));
    await tester.pumpAndSettle();
    expect(find.text('Lab partner'), findsOneWidget);
  });

  testWidgets('private names react without changing canonical identity', (
    tester,
  ) async {
    final notes = ValueNotifier<Map<int, PrivateNotePayload>>({2: note});
    await tester.pumpWidget(
      app(
        ValueListenableBuilder(
          valueListenable: notes,
          builder: (_, data, _) => PrivateNotesScope(
            ownerId: 1,
            notes: data,
            child: Builder(
              builder: (context) =>
                  Text(privateDisplayName(context, 2, 'alice', 'Nickname')),
            ),
          ),
        ),
      ),
    );
    expect(find.text('Lab partner(Nickname)'), findsOneWidget);
    notes.value = {};
    await tester.pump();
    expect(find.text('Nickname'), findsOneWidget);
    expect(note.username, 'alice');
    notes.dispose();
  });
  testWidgets(
    'an old session response cannot repopulate notes after an epoch change',
    (tester) async {
      final pending = Completer<Map<int, PrivateNotePayload>>();
      final container = ProviderContainer(
        overrides: [
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: 'owner'),
          ),
          privateNotesProvider((1, 0)).overrideWith((ref) => pending.future),
          privateNotesProvider((1, 1)).overrideWith((ref) async => {}),
        ],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: app(
            PrivateNotesHost(
              child: Builder(
                builder: (context) =>
                    Text(privateDisplayName(context, 2, 'alice')),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      pending.complete({2: note});
      await tester.pumpAndSettle();
      expect(find.text('Lab partner(alice)'), findsNothing);
      expect(find.text('alice'), findsOneWidget);
    },
  );
  testWidgets(
    'private note editor retains failed input and permits retry and clear',
    (tester) async {
      tester.view.physicalSize = const Size(320, 640);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final repo = NotesRepository();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [userRepositoryProvider.overrideWithValue(repo)],
          child: app(
            const PrivateNotesScope(
              ownerId: 1,
              notes: {2: note},
              child: PrivateNoteButton(userId: 2, username: 'alice'),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Edit note'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'New note');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(find.text('New note'), findsOneWidget);
      repo.fail = false;
      await tester.enterText(find.byType(TextField), '');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(repo.saved, ['']);
      expect(find.byType(AlertDialog), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
}
