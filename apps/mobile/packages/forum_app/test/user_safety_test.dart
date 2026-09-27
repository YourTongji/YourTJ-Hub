import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/report_content.dart';
import 'package:forum_app/src/user_blocks.dart';

class Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

GfApiClient client() => GfApiClient(dio: Dio(), tokenStorage: Tokens());

class Reports extends PostRepository {
  Reports() : super(client());
  final sent = <Map<String, Object>>[];
  @override
  Future<bool> report({
    required String targetType,
    required int targetId,
    required String reason,
    required String note,
  }) async {
    sent.add({
      'targetType': targetType,
      'targetId': targetId,
      'reason': reason,
      'note': note,
    });
    return true;
  }
}

class Blocks extends UserRepository {
  Blocks() : super(client());
  bool blocked = false;
  final writes = <bool>[];
  @override
  Future<UserBlocksPayload> getUserBlocks() async => UserBlocksPayload(
    ownerId: 1,
    blocks: [
      if (blocked) const BlockedUserPayload(targetUserId: 2, username: 'alice'),
    ],
  );
  @override
  Future<void> setUserBlock(int targetUserId, bool value) async {
    writes.add(value);
    blocked = value;
  }
}

Widget app(Widget child) => MaterialApp(
  locale: const Locale('en'),
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);
void main() {
  for (final target in ['post', 'chat_message']) {
    testWidgets('$target sends a valid reason and preserves the explanation', (
      tester,
    ) async {
      final repository = Reports();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repository)],
          child: app(
            Builder(
              builder: (context) => TextButton(
                onPressed: () => showContentReport(
                  context,
                  targetType: target,
                  targetId: 12,
                ),
                child: const Text('Open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Open'));
      await tester.pumpAndSettle();
      if (target == 'chat_message') {
        expect(find.textContaining('rest of the conversation'), findsOneWidget);
      }
      await tester.enterText(find.byType(TextField), '这条消息持续骚扰我');
      await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
      await tester.pumpAndSettle();
      expect(repository.sent, [
        {
          'targetType': target,
          'targetId': 12,
          'reason': 'abuse',
          'note': '这条消息持续骚扰我',
        },
      ]);
    });
  }
  testWidgets('blocking requires confirmation and can be reversed', (
    tester,
  ) async {
    final repository = Blocks();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: 'owner'),
          ),
          userRepositoryProvider.overrideWithValue(repository),
        ],
        child: app(const UserBlockButton(userId: 2)),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Block user'));
    await tester.pumpAndSettle();
    expect(repository.writes, isEmpty);
    await tester.tap(find.widgetWithText(FilledButton, 'Block user'));
    await tester.pumpAndSettle();
    expect(repository.writes, [true]);
    await tester.tap(find.byTooltip('Unblock user'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Unblock user'));
    await tester.pumpAndSettle();
    expect(repository.writes, [true, false]);
  });
}
