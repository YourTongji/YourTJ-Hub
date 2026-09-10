import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:image/image.dart' as img;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/publish/publish_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/pages/settings/profile_image_editor.dart';
import 'package:forum_app/src/widgets/profile_image_crop.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/pages/settings/oauth_bindings_sheet.dart';

/// Opt-in against an isolated, seeded local server. Never point this journey at
/// a shared deployment: it signs in, saves a draft, and opens the real console.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  const username = String.fromEnvironment('YOURTJ_TEST_USERNAME');
  const password = String.fromEnvironment('YOURTJ_TEST_PASSWORD');
  final origin = Uri.tryParse(AppConfig.apiBaseUrl);
  final enabled = username.isNotEmpty && password.isNotEmpty;

  testWidgets('local sign-in, content preview, draft and management handoff', (
    tester,
  ) async {
    expect(origin, isNotNull);
    expect(
      ['localhost', '127.0.0.1', '10.0.2.2', '::1'],
      contains(origin!.host),
      reason: 'This test may only mutate an isolated local database',
    );
    final container = ProviderContainer();
    await container.read(tokenStorageProvider).clear();
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const GfApp(locale: Locale('en')),
      ),
    );
    appRouter.go('/login');

    Future<void> capture(String name) async {
      if (!const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) return;
      await tester.pump(const Duration(milliseconds: 300));
      await binding.takeScreenshot(name);
    }

    Future<void> until(bool Function() ready, String description) async {
      final deadline = DateTime.now().add(const Duration(seconds: 30));
      while (!ready() && DateTime.now().isBefore(deadline)) {
        await tester.pump(const Duration(milliseconds: 200));
      }
      if (!ready()) {
        debugPrint(
          tester
              .widgetList<Text>(find.byType(Text))
              .map((w) => w.data ?? '')
              .join(' | '),
        );
      }
      expect(ready(), isTrue, reason: description);
      expect(tester.takeException(), isNull);
    }

    await until(
      () => find.byType(LoginPage).evaluate().isNotEmpty,
      'Login page opens',
    );
    await tester.pump(const Duration(seconds: 1));
    await tester.enterText(find.byType(TextField).at(0), username);
    await tester.enterText(find.byType(TextField).at(1), password);
    final login = find.byWidgetPredicate(
      (w) => w is GfButton && w.label == 'Sign in',
    );
    await tester.ensureVisible(login);
    await tester.tap(login);
    await until(
      () =>
          find.byType(LoginPage).evaluate().isEmpty &&
          find.byType(HomePage).evaluate().isNotEmpty,
      'Password login reaches home',
    );
    expect(await container.read(tokenStorageProvider).read(), isNotEmpty);
    await capture('home');

    appRouter.push('/publish');
    await until(
      () =>
          find.byType(PublishPage).evaluate().isNotEmpty &&
          find.byKey(const Key('publish-editor')).evaluate().isNotEmpty,
      'Composer loads from the backend',
    );
    final title =
        'Mobile device draft ${DateTime.now().millisecondsSinceEpoch}';
    await tester.enterText(find.byType(TextField).first, title);
    await tester.enterText(
      find.byKey(const Key('publish-editor')),
      'A local integration test of mobile content publishing.',
    );
    await tester.tap(find.byKey(const Key('publish-appbar-submit')));
    await until(
      () => find.byKey(const Key('publish-preview')).evaluate().isNotEmpty,
      'Next opens the classification preview',
    );
    final chips = find.byType(GfSelectTag);
    await until(
      () => chips.evaluate().isNotEmpty,
      'Preview offers server categories',
    );
    await tester.ensureVisible(chips.first);
    await tester.tap(chips.first);
    await capture('publish-preview');
    final save = find.byKey(const Key('publish-save-draft'));
    await tester.ensureVisible(save);
    await tester.tap(save);
    await until(
      () => find.text('Saved as draft').evaluate().isNotEmpty,
      'Draft is accepted by the server',
    );
    final drafts = await container
        .read(pageRepositoryProvider)
        .fetch('/drafts');
    expect(drafts.toJson().toString(), contains(title));
    appRouter.pop();
    await tester.pump(const Duration(milliseconds: 400));

    // Exercise native crop -> real image upload -> profile persistence without
    // mocking HTTP. System photo-picker UI is a separate platform check.
    final pages = container.read(pageRepositoryProvider);
    final users = container.read(userRepositoryProvider);
    final previousCover = parsePageProps<SettingsPageProps>(
      await pages.fetch('/settings'),
    )!.user.profileCoverUrl;
    final sample = img.Image(width: 1200, height: 360);
    for (final pixel in sample) {
      pixel.setRgb(40 + pixel.x * 150 ~/ 1200, 100 + pixel.y * 100 ~/ 360, 210);
    }
    var uploadedCover = '';
    try {
      unawaited(
        Navigator.of(
          tester.element(find.byType(HomePage)),
          rootNavigator: true,
        ).push<bool>(
          MaterialPageRoute(
            builder: (_) => ProfileImageEditor(
              source: prepareProfileCrop(img.encodePng(sample)),
              cover: true,
              onSave: (bytes) async {
                uploadedCover = await container
                    .read(fileRepositoryProvider)
                    .uploadImage(bytes: bytes, filename: 'cover.webp');
                await users.saveUserProfileCover(uploadedCover);
              },
            ),
          ),
        ),
      );
      await until(
        () => find.byType(ProfileImageEditor).evaluate().isNotEmpty,
        'Native cover crop opens',
      );
      await tester.pump(const Duration(milliseconds: 500));
      await tester.timedDrag(
        find.byKey(const Key('profile-image-crop')),
        const Offset(0, -24),
        const Duration(milliseconds: 500),
      );
      await capture('cover-crop');
      await tester.tap(find.text('Save'));
      await until(
        () => find.byType(ProfileImageEditor).evaluate().isEmpty,
        'Cropped cover upload succeeds',
      );
      expect(uploadedCover, isNotEmpty);
      final changed = parsePageProps<SettingsPageProps>(
        await pages.fetch('/settings'),
      )!;
      expect(
        changed.user.profileCoverUrl,
        endsWith(Uri.parse(uploadedCover).path),
      );
    } finally {
      await users.saveUserProfileCover(previousCover);
    }

    appRouter.push('/settings/binding');
    await until(
      () => find.byType(SettingsPage).evaluate().isNotEmpty,
      'Account settings open',
    );
    await tester.tap(find.text('External account connections'));
    await until(
      () =>
          find.byType(OAuthBindingsSheet).evaluate().isNotEmpty &&
          find.text('GitHub').evaluate().isNotEmpty,
      'Native connections load',
    );
    await capture('oauth-connections');
    Navigator.of(tester.element(find.byType(OAuthBindingsSheet))).pop();
    await until(
      () => find.byType(OAuthBindingsSheet).evaluate().isEmpty,
      'Connections close',
    );
    appRouter.pop();
    await tester.pump(const Duration(milliseconds: 400));

    appRouter.push('/admin');
    await until(
      () => find.byType(WebViewWidget).evaluate().isNotEmpty,
      'Native management browser is created',
    );
    final web = tester
        .widget<WebViewWidget>(find.byType(WebViewWidget))
        .platform
        .params
        .controller;
    var loaded = false;
    for (var attempt = 0; attempt < 100 && !loaded; attempt++) {
      await tester.pump(const Duration(milliseconds: 300));
      final uri = Uri.tryParse(await web.currentUrl() ?? '');
      if (uri?.path.startsWith('/admin') == true) {
        final ready = await web.runJavaScriptReturningResult(
          "document.querySelector('[data-slot=sidebar-inset]') !== null",
        );
        loaded = ready == true || ready.toString() == 'true';
      }
    }
    expect(
      loaded,
      isTrue,
      reason: 'The handoff cookie loads the authorized console',
    );
    await tester.pump(const Duration(seconds: 2));
    await capture('admin-overview');
    // Each real administrative module must fit the native viewport. Tables can
    // scroll within their own container; the entire document must not spill.
    for (final path
        in const bool.fromEnvironment('YOURTJ_TEST_FILES_ONLY')
            ? <String>[]
            : [
                '/admin/users',
                '/admin/agents',
                '/admin/roles',
                '/admin/categories',
                '/admin/posts',
                '/admin/links',
                '/admin/wiki',
                '/admin/sponsors',
                '/admin/badges',
                '/admin/files/resources',
                '/admin/opt-records',
                '/admin/review-queue',
                '/admin/data',
                '/admin/settings/site-chrome',
                '/admin/settings/site-info',
                '/admin/settings/mail',
                '/admin/settings/security',
                '/admin/settings/posting',
                '/admin/settings/rate-limit',
                '/admin/settings/mcp',
                '/admin/settings/ai-summary',
                '/admin/settings/announcement',
                '/admin/settings/http-notify',
                '/admin/settings/storage',
                '/admin/settings/terms',
                '/admin/settings/privacy',
                '/admin/settings/onesystem',
                '/admin/settings/schedule',
              ]) {
      await web.runJavaScript("window.location.assign('$path')");
      var ready = false;
      for (var attempt = 0; attempt < 100 && !ready; attempt++) {
        await tester.pump(const Duration(milliseconds: 200));
        final current = Uri.tryParse(await web.currentUrl() ?? '');
        if (current?.path == path) {
          final mounted = await web.runJavaScriptReturningResult(
            "document.querySelector('[data-slot=sidebar-inset] h1')?.textContent.trim().length > 0",
          );
          ready = mounted == true || mounted.toString() == 'true';
        }
      }
      expect(ready, isTrue, reason: 'Management module renders: $path');
      await tester.pump(const Duration(milliseconds: 500));
      if (path == '/admin/links') {
        final visible = await web.runJavaScriptReturningResult(
          "getComputedStyle(document.querySelector('article button[title]').parentElement).opacity === '1' && getComputedStyle(document.querySelector('article > div:last-child')).opacity === '1'",
        );
        expect(
          visible == true || visible.toString() == 'true',
          isTrue,
          reason: 'Touch users can discover link actions without hovering',
        );
        await web.runJavaScript(
          "document.querySelector('[data-testid=admin-link-add]').click()",
        );
        await tester.pump(const Duration(milliseconds: 500));
        final fits = await web.runJavaScriptReturningResult(
          "(()=>{const r=document.querySelector('[role=dialog]').getBoundingClientRect(); return r.top>=0 && r.bottom<=window.innerHeight && r.width<=window.innerWidth;})()",
        );
        final formGeometry = await web.runJavaScriptReturningResult(
          "JSON.stringify({rect:document.querySelector('[role=dialog]').getBoundingClientRect().toJSON(),maxHeight:getComputedStyle(document.querySelector('[role=dialog]')).maxHeight,innerHeight:window.innerHeight,innerWidth:window.innerWidth,visualHeight:window.visualViewport.height})",
        );
        expect(
          fits == true || fits.toString() == 'true',
          isTrue,
          reason: 'The complete link form remains reachable: $formGeometry',
        );
        await capture('link-form');
        final linkName =
            'Mobile management test ${DateTime.now().millisecondsSinceEpoch}';
        await web.runJavaScript('''
          (() => {
            const inputs = document.querySelectorAll('[role=dialog] input');
            inputs[0].value = ${jsonEncode(linkName)};
            inputs[1].value = 'https://mobile.example.invalid/';
            for (const input of inputs) input.dispatchEvent(new Event('input', {bubbles:true}));
          })()
        ''');
        await tester.pump(const Duration(milliseconds: 200));
        await web.runJavaScript(
          "document.querySelector('[role=dialog] form').requestSubmit()",
        );
        Future<bool> publicLinksContainName() async =>
            (await container.read(pageRepositoryProvider).fetch('/links'))
                .toJson()
                .toString()
                .contains(linkName);
        var saved = false;
        for (var i = 0; i < 30 && !saved; i++) {
          await tester.pump(const Duration(milliseconds: 200));
          saved = await publicLinksContainName();
        }
        expect(
          saved,
          isTrue,
          reason: 'Management form saves a publicly visible link',
        );
        await web.runJavaScript('''
          (() => {
            const link = [...document.querySelectorAll('article a')].find(a=>a.textContent.trim()===${jsonEncode(linkName)});
            link.closest('article').querySelector('[data-testid=admin-link-delete]').click();
          })()
        ''');
        await tester.pump(const Duration(milliseconds: 300));
        await web.runJavaScript(
          "document.querySelector('[data-testid=admin-confirm]').click()",
        );
        var deleted = false;
        for (var i = 0; i < 30 && !deleted; i++) {
          await tester.pump(const Duration(milliseconds: 200));
          deleted = !await publicLinksContainName();
        }
        expect(
          deleted,
          isTrue,
          reason: 'Confirmed deletion removes the test link',
        );
      }
      if (path == '/admin/links' || path == '/admin/settings/rate-limit') {
        await capture(path.split('/').last);
      }
      final fits = await web.runJavaScriptReturningResult(
        'document.documentElement.scrollWidth <= window.innerWidth + 1',
      );
      if (fits != true && fits.toString() != 'true') {
        debugPrint(
          (await web.runJavaScriptReturningResult(
            "JSON.stringify([...document.querySelectorAll('body *')].filter(e=>e.getBoundingClientRect().right>window.innerWidth+1).slice(0,12).map(e=>({tag:e.tagName,classes:e.className,width:e.getBoundingClientRect().width})))",
          )).toString(),
        );
      }
      expect(
        fits == true || fits.toString() == 'true',
        isTrue,
        reason: 'Management module fits the mobile viewport: $path',
      );
    }
    // Opt-in manual platform checkpoint: use the real share sheet and document
    // picker. Select mobile-picker-check.json in the system picker; no import
    // is submitted. The ordinary unattended journey leaves these prompts out.
    if (const bool.fromEnvironment('YOURTJ_TEST_FILES')) {
      await web.runJavaScript("window.location.assign('/admin/data')");
      var ready = false;
      for (var i = 0; i < 100 && !ready; i++) {
        await tester.pump(const Duration(milliseconds: 200));
        final value = await web.runJavaScriptReturningResult(
          "!!document.querySelector('[data-testid=admin-export-create]')",
        );
        ready = value == true || value.toString() == 'true';
      }
      expect(ready, isTrue);
      await web.runJavaScript(
        "document.querySelector('[data-testid=admin-export-create]').click()",
      );
      await tester.pump(const Duration(milliseconds: 500));
      await web.runJavaScript(
        "document.querySelector('[data-testid=admin-confirm]').click()",
      );
      var downloadable = false;
      for (var i = 0; i < 150 && !downloadable; i++) {
        await tester.pump(const Duration(milliseconds: 200));
        final value = await web.runJavaScriptReturningResult(
          "!!document.querySelector('[data-testid=admin-export-download]')",
        );
        downloadable = value == true || value.toString() == 'true';
      }
      expect(
        downloadable,
        isTrue,
        reason: 'A real export becomes downloadable',
      );
      await web.runJavaScript(
        "document.querySelector('[data-testid=admin-export-download]').click()",
      );
      debugPrint(
        'NATIVE_EXPORT_OPEN: inspect the system share sheet and dismiss it',
      );
      // Wait for the native share future to finish, not an arbitrary delay:
      // slow system animation/accessibility must not overlap the next picker.
      await tester.pump(const Duration(milliseconds: 500));
      bool sharing() => tester
          .widgetList<LinearProgressIndicator>(
            find.byType(LinearProgressIndicator),
          )
          .any((indicator) => indicator.value == null);
      for (var i = 0; i < 1500 && sharing(); i++) {
        await tester.pump(const Duration(milliseconds: 200));
      }
      expect(sharing(), isFalse, reason: 'System share sheet was dismissed');
      await web.runJavaScript(
        "document.querySelector('input[type=file]').closest('label').scrollIntoView({block:'center'})",
      );
      await tester.pump(const Duration(milliseconds: 500));
      final rawRect = await web.runJavaScriptReturningResult(
        "JSON.stringify((()=>{const r=document.querySelector('input[type=file]').closest('label').getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2};})())",
      );
      var decoded = rawRect is String ? jsonDecode(rawRect) : rawRect;
      if (decoded is String) decoded = jsonDecode(decoded);
      final rect = decoded as Map<String, dynamic>;
      await tester.tapAt(
        tester.getTopLeft(find.byType(WebViewWidget)) +
            Offset(
              (rect['x'] as num).toDouble(),
              (rect['y'] as num).toDouble(),
            ),
      );
      debugPrint('NATIVE_PICKER_OPEN: select mobile-picker-check.json');
      var chosen = false;
      for (var i = 0; i < 1500 && !chosen; i++) {
        await tester.pump(const Duration(milliseconds: 200));
        final value = await web.runJavaScriptReturningResult(
          "document.querySelector('input[type=file]')?.files?.[0]?.name === 'mobile-picker-check.json'",
        );
        chosen = value == true || value.toString() == 'true';
      }
      expect(
        chosen,
        isTrue,
        reason: 'System file selection reaches the Web form',
      );
      debugPrint('NATIVE_PICKER_RETURNED: mobile-picker-check.json');
      await capture('admin-import-selection');
    }
    // Course workspaces have their own permission and handoff targets. Open
    // each from a fresh native route to exercise cookie teardown/recreation.
    for (final path
        in const bool.fromEnvironment('YOURTJ_TEST_FILES_ONLY')
            ? <String>[]
            : ['/moderation/courses', '/moderation/course-reviews']) {
      await tester.tap(find.byTooltip('Close'));
      await until(
        () => find.byType(WebViewWidget).evaluate().isEmpty,
        'Previous management browser closes',
      );
      await tester.pump(const Duration(milliseconds: 400));
      appRouter.push(path);
      await until(
        () => find.byType(WebViewWidget).evaluate().isNotEmpty,
        'Course workspace browser is created: $path',
      );
      final courseWeb = tester
          .widget<WebViewWidget>(find.byType(WebViewWidget))
          .platform
          .params
          .controller;
      var ready = false;
      for (var attempt = 0; attempt < 100 && !ready; attempt++) {
        await tester.pump(const Duration(milliseconds: 200));
        if (Uri.tryParse(await courseWeb.currentUrl() ?? '')?.path == path) {
          final mounted = await courseWeb.runJavaScriptReturningResult(
            "document.querySelector('main h1')?.textContent.trim().length > 0",
          );
          ready = mounted == true || mounted.toString() == 'true';
        }
      }
      expect(ready, isTrue, reason: 'Course handoff renders $path');
      await tester.pump(const Duration(milliseconds: 500));
      final fits = await courseWeb.runJavaScriptReturningResult(
        'document.documentElement.scrollWidth <= window.innerWidth + 1',
      );
      expect(
        fits == true || fits.toString() == 'true',
        isTrue,
        reason: 'Course workspace fits the mobile viewport: $path',
      );
      await capture(path.split('/').last);
    }
    expect(tester.takeException(), isNull);
    appRouter.go('/');
    await tester.pump(const Duration(seconds: 4));
    await tester.pumpWidget(const SizedBox.shrink());
    container.dispose();
  }, skip: !enabled);
}
