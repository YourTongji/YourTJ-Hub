import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/editor/rich_markdown_editor.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;

// Real loopback server; all navigation and submission use production UI.
// Quill's public controller enters rich text; no shared-site publications.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'article validation, preview, resize and real submission',
    (tester) async {
      expect([
        'localhost',
        '127.0.0.1',
        '::1',
      ], contains(Uri.parse(AppConfig.apiBaseUrl).host));
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(
            MemTokenStorage(const String.fromEnvironment('YOURTJ_TEST_TOKEN')),
          ),
          pushBootstrapProvider.overrideWith((_) {}),
        ],
      );
      Future<void> until(bool Function() ready) async {
        final end = DateTime.now().add(const Duration(seconds: 55));
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

      appRouter.go('/publish?type=3');
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const GfApp(locale: Locale('zh')),
        ),
      );
      final title = find.byKey(const Key('publish-title'));
      final next = find.byKey(const Key('publish-appbar-submit'));
      await until(() => title.evaluate().isNotEmpty);
      final l = AppLocalizations.of(tester.element(find.byType(PublishPage)));
      final pages = container.read(pageRepositoryProvider);
      final props = parsePageProps<PublishPageProps>(
        await pages.fetch('/publish'),
      )!;
      final category = props.categories.first;
      final topics = container.read(topicRepositoryProvider);
      for (final invalid in [
        (title: '', content: 'body'),
        (title: 'title', content: ''),
      ]) {
        await expectLater(
          topics.writeTopicResult(
            topicId: 0,
            title: invalid.title,
            content: invalid.content,
            categoryIds: [category.id],
            topicStatus: 1,
          ),
          throwsA(
            isA<ApiException>().having(
              (e) => e.messageCode,
              'server validation',
              'common.request.invalidParams',
            ),
          ),
        );
      }
      var writes = 0;
      int? published;
      container
          .read(apiClientProvider)
          .dio
          .interceptors
          .add(
            InterceptorsWrapper(
              onRequest: (options, handler) {
                if (options.path.endsWith('/topics/write')) writes++;
                handler.next(options);
              },
              onResponse: (response, handler) {
                if (response.requestOptions.path.endsWith('/topics/write') &&
                    response.data is Map) {
                  final data = response.data as Map;
                  if (data['code'] == 0 && data['result'] is int) {
                    published = data['result'] as int;
                  }
                }
                handler.next(response);
              },
            ),
          );
      final finalTitle =
          'Duo validation acceptance ${DateTime.now().millisecondsSinceEpoch}';
      const body =
          'Isolated native end-to-end article body. This is disposable local test data.';
      for (final brightness in Brightness.values) {
        container
            .read(themeModeProvider.notifier)
            .setMode(
              brightness == Brightness.dark ? ThemeMode.dark : ThemeMode.light,
            );
        await tester.pumpAndSettle();
        final editor = tester.widget<RichMarkdownEditor>(
          find.byKey(const Key('publish-editor')),
        );
        await tester.enterText(title, '   ');
        editor.controller.replaceText(
          0,
          editor.controller.document.length - 1,
          ' \n',
          const TextSelection.collapsed(offset: 2),
        );
        await tester.pumpAndSettle();
        await tester.tap(next);
        await tester.pumpAndSettle();
        expect(find.text(l.publishTitleRequired), findsOneWidget);
        expect(tester.widget<TextField>(title).focusNode!.hasFocus, isTrue);
        expect(find.byKey(const Key('publish-preview')), findsNothing);
        await until(() => tester.view.viewInsets.bottom > 0);
        await capture('1442-title-${brightness.name}');
        await tester.enterText(title, finalTitle);
        await tester.tap(next);
        await tester.pumpAndSettle();
        expect(find.text(l.publishContentRequired), findsOneWidget);
        expect(editor.focusNode.hasFocus, isTrue);
        expect(writes, 0);
        await capture('1442-body-${brightness.name}');
        editor.controller.replaceText(
          0,
          editor.controller.document.length - 1,
          body,
          TextSelection.collapsed(offset: body.length),
        );
        await tester.pumpAndSettle();
        await tester.tap(next);
        await until(
          () =>
              find.byKey(const Key('publish-preview')).evaluate().isNotEmpty &&
              tester.view.viewInsets.bottom == 0,
        );
        expect(find.text(l.publishContentRequired), findsNothing);
        await capture('1442-preview-${brightness.name}');
        await tester.tap(
          find
              .descendant(
                of: find.byType(PublishPage),
                matching: find.byTooltip(l.commonBack),
              )
              .first,
        );
        await until(() => title.evaluate().isNotEmpty);
        expect(tester.widget<TextField>(title).controller!.text, finalTitle);
        expect(
          tester
              .widget<RichMarkdownEditor>(
                find.byKey(const Key('publish-editor')),
              )
              .controller
              .document
              .toPlainText(),
          contains(body),
        );
      }
      await tester.tap(next);
      await until(
        () =>
            find.byKey(const Key('publish-preview')).evaluate().isNotEmpty &&
            tester.view.viewInsets.bottom == 0,
      );
      if (const bool.fromEnvironment('YOURTJ_TEST_NATIVE_WINDOW')) {
        debugPrint('DUO_ACCEPT_OPEN');
        await until(
          () =>
              tester.view.physicalSize.width / tester.view.devicePixelRatio >
              700,
        );
        expect(find.byKey(const Key('publish-preview')), findsOneWidget);
        expect(tester.widget<TextField>(title).controller!.text, finalTitle);
        await capture('1442-preview-open');
        debugPrint('DUO_ACCEPT_CLOSE');
        await until(
          () =>
              tester.view.physicalSize.width / tester.view.devicePixelRatio <
              700,
        );
      }
      final menu = find.byKey(const Key('publish-category-menu'));
      await tester.ensureVisible(menu);
      await tester.pumpAndSettle();
      await tester.tap(menu);
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(ValueKey('publish-category-${category.id}')));
      await tester.pumpAndSettle();
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pumpAndSettle();
      await tester.tap(next);
      await until(
        () => published != null && find.byType(TopicPage).evaluate().isNotEmpty,
      );
      expect(writes, 1);
      final fresh = await pages.fetch('/p/post/$published');
      expect(fresh.props.toString(), contains(finalTitle));
      expect(fresh.props.toString(), contains(body));
      await capture('1442-published');
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      container.dispose();
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LOCAL_PUBLISH_VALIDATION'),
  );
}
