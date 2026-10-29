import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/storage/user_work_database.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/editor/rich_markdown_editor.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;

// Real local HTTP, native encrypted SQLite and complete app routes. Existing
// session supplied separately; never sends content to a shared deployment.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  const token = String.fromEnvironment('YOURTJ_TEST_TOKEN');
  testWidgets(
    'new and legacy local drafts reopen with their original content',
    (tester) async {
      expect([
        'localhost',
        '127.0.0.1',
        '::1',
      ], contains(Uri.parse(AppConfig.apiBaseUrl).host));
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(MemTokenStorage(token)),
          pushBootstrapProvider.overrideWith((_) {}),
        ],
      );
      Future<void> until(bool Function() ready) async {
        final end = DateTime.now().add(const Duration(seconds: 40));
        while (!ready() && DateTime.now().isBefore(end)) {
          await tester.pump(const Duration(milliseconds: 100));
        }
        expect(ready(), isTrue);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
      }

      Future<void> capture(String name) async {
        await tester.pumpAndSettle();
        if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
          await binding.takeScreenshot(name);
        }
      }

      appRouter.go('/drafts');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('zh')),
        ),
      );
      appRouter.go('/drafts');
      final owner = await container.read(writingScopeProvider.future);
      expect(owner.endsWith(':0'), isFalse);
      final prefs = await SharedPreferences.getInstance();
      if (const bool.fromEnvironment('YOURTJ_TEST_RESUME_DRAFTS')) {
        final savedStamp = prefs.getInt('duo-draft-e2e-stamp');
        expect(
          savedStamp,
          isNotNull,
          reason: 'previous process persisted the marker',
        );
        await until(() => find.byType(TextField).evaluate().isNotEmpty);
        for (final type in [1, 3]) {
          final title = 'Duo 草稿 $type $savedStamp';
          await tester.enterText(find.byType(TextField), title);
          FocusManager.instance.primaryFocus?.unfocus();
          await tester.pumpAndSettle();
          debugPrint('DRAFT COLD: opening type $type');
          final saved =
              (await container.read(writingStoreProvider).drafts(owner))
                  .singleWhere((d) => d.title == title);
          final row = find.byKey(ValueKey(saved.key));
          await until(() => row.evaluate().isNotEmpty);
          FocusManager.instance.primaryFocus?.unfocus();
          await tester.pumpAndSettle();
          final open = find.descendant(of: row, matching: find.text('继续编辑'));
          await tester.ensureVisible(open);
          await tester.pumpAndSettle();
          expect(open.hitTestable(), findsOneWidget);
          await tester.tap(open);
          final titleField = find.byKey(const Key('publish-title'));
          await until(() => titleField.evaluate().isNotEmpty);
          expect(tester.widget<TextField>(titleField).controller!.text, title);
          final editor = find.byKey(const Key('publish-editor'));
          final body = '恢复正文 $type $savedStamp';
          expect(saved.content, contains(body));
          if (type == 3) {
            expect(
              tester
                  .widget<RichMarkdownEditor>(editor)
                  .controller
                  .document
                  .toPlainText(),
              contains(body),
            );
          } else {
            expect(tester.widget<TextField>(editor).controller!.text, body);
          }
          await capture('1437-cold-restored-$type');
          final l = AppLocalizations.of(
            tester.element(find.byType(PublishPage)),
          );
          await tester.tap(
            find
                .descendant(
                  of: find.byType(PublishPage),
                  matching: find.byTooltip(l.commonBack),
                )
                .first,
          );
          await until(
            () => find.text(l.draftKeepAndLeave).evaluate().isNotEmpty,
          );
          await tester.tap(find.text(l.draftKeepAndLeave));
          await until(() => find.byType(PublishPage).evaluate().isEmpty);
        }
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
        container.dispose();
        return;
      }
      final stamp = DateTime.now().millisecondsSinceEpoch;
      final legacy = LocalDraft(
        key: 'legacy-$stamp',
        title: '旧版草稿 $stamp',
        content: '旧版保存的正文',
        contentType: 1,
        topicId: 0,
        categories: [],
        images: [],
        updatedAt: stamp,
      );
      await UserWorkDatabase.instance.writeBatch(owner, 'draft', {
        legacy.key: jsonEncode({...legacy.toJson(), 'identity': ''}),
      });
      appRouter.go('/settings');
      await tester.pumpAndSettle();
      appRouter.go('/drafts');
      await until(() => find.text(legacy.title).evaluate().isNotEmpty);
      await capture('1437-legacy-recovered');
      for (final type in [1, 3]) {
        container
            .read(themeModeProvider.notifier)
            .setMode(type == 1 ? ThemeMode.light : ThemeMode.dark);
        await tester.tap(find.byTooltip('发布'));
        await tester.pumpAndSettle();
        await tester.tap(find.byKey(ValueKey('compose-$type')));
        final titleField = find.byKey(const Key('publish-title'));
        await until(
          () =>
              titleField.evaluate().isNotEmpty &&
              find.byKey(const Key('publish-editor')).evaluate().isNotEmpty,
        );
        final title = 'Duo 草稿 $type $stamp';
        final body = '恢复正文 $type $stamp';
        await tester.enterText(titleField, title);
        final editor = find.byKey(const Key('publish-editor'));
        if (type == 3) {
          // Quill has a RawEditor text client rather than EditableText. Exercise
          // its public controller for text entry, then use UI for save and restore.
          tester
              .widget<RichMarkdownEditor>(editor)
              .controller
              .document
              .insert(0, body);
        } else {
          await tester.enterText(editor, body);
        }
        FocusManager.instance.primaryFocus?.unfocus();
        await tester.pumpAndSettle();
        final l = AppLocalizations.of(tester.element(find.byType(PublishPage)));
        await until(() => find.text(l.draftLocalSaved).evaluate().isNotEmpty);
        await tester.tap(
          find
              .descendant(
                of: find.byType(PublishPage),
                matching: find.byTooltip(l.commonBack),
              )
              .first,
        );
        await until(() => find.text(l.draftKeepAndLeave).evaluate().isNotEmpty);
        await tester.tap(find.text(l.draftKeepAndLeave));
        await until(() => find.byType(PublishPage).evaluate().isEmpty);
        await until(() => find.text(title).evaluate().isNotEmpty);
        debugPrint('DRAFT E2E: saved type $type in list');
        final saved = (await container.read(writingStoreProvider).drafts(owner))
            .singleWhere((d) => d.title == title);
        expect(saved.identity, 'member');
        expect(saved.content, contains(body));
        await capture('1437-list-$type');
        await tester.tap(find.text(title));
        await until(() => titleField.evaluate().isNotEmpty);
        expect(tester.widget<TextField>(titleField).controller!.text, title);
        if (type == 3) {
          expect(
            tester
                .widget<RichMarkdownEditor>(editor)
                .controller
                .document
                .toPlainText(),
            contains(body),
          );
        } else {
          expect(tester.widget<TextField>(editor).controller!.text, body);
        }
        await capture('1437-restored-$type');
        await tester.tap(
          find
              .descendant(
                of: find.byType(PublishPage),
                matching: find.byTooltip(l.commonBack),
              )
              .first,
        );
        await until(() => find.text(l.draftKeepAndLeave).evaluate().isNotEmpty);
        await tester.tap(find.text(l.draftKeepAndLeave));
        await until(() => find.byType(PublishPage).evaluate().isEmpty);
      }
      // Re-enter through a fresh page state, rather than a retained list snapshot.
      appRouter.go('/settings');
      await tester.pumpAndSettle();
      appRouter.go('/drafts');
      await until(() => find.byType(TextField).evaluate().isNotEmpty);
      await tester.enterText(find.byType(TextField), legacy.title);
      await until(() => find.text(legacy.title).evaluate().isNotEmpty);
      expect(
        (await container.read(writingStoreProvider).drafts(owner)).where(
          (d) => d.title.contains('$stamp'),
        ),
        hasLength(3),
      );
      expect(
        await container
            .read(writingStoreProvider)
            .drafts(writingScope(AppConfig.apiBaseUrl, 99999)),
        isEmpty,
      );
      await prefs.setInt('duo-draft-e2e-stamp', stamp);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LOCAL_DRAFTS'),
  );
}
