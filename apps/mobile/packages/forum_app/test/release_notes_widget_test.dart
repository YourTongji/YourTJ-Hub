import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/updates/release_notes.dart';
import 'package:forum_app/src/updates/release_notes_view.dart';
import 'package:forum_app/src/updates/update_prompt_sheet.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  for (final brightness in Brightness.values) {
    for (final locale in const [Locale('en'), Locale('de')]) {
      testWidgets(
        '${brightness.name} ${locale.languageCode}: notes and actions fit 320px at 2x text',
        (tester) async {
          final semantics = tester.ensureSemantics();
          tester.view.physicalSize = const Size(640, 1280);
          tester.view.devicePixelRatio = 2;
          addTearDown(tester.view.reset);

          final l10n = await AppLocalizations.delegate.load(locale);
          final notes = [
            for (var i = 0; i < 8; i++)
              ReleaseNote(
                id: 'feature-$i',
                title: 'Feature $i',
                summary:
                    'A detailed improvement that wraps across a narrow screen.',
                platforms: const {'android'},
                kind: 'feature',
              ),
            for (var i = 0; i < 3; i++)
              ReleaseNote(
                id: 'required-$i',
                title: 'Required action $i',
                summary: 'Please review this required action before updating.',
                platforms: const {'android'},
                kind: 'security',
                required: true,
              ),
          ];
          final key = GlobalKey<_DialogHarnessState>();
          await tester.pumpWidget(
            _testApp(
              brightness: brightness,
              locale: locale,
              child: _DialogHarness(key: key, notes: notes),
            ),
          );
          await tester.pumpAndSettle();

          expect(tester.takeException(), isNull);
          expect(find.text('Feature 0'), findsOneWidget);
          expect(find.text('Feature 5'), findsNothing);
          expect(find.text('Required action 0'), findsOneWidget);
          expect(find.text('Required action 2'), findsOneWidget);

          final moreButton = find.widgetWithText(
            GfButton,
            l10n.releaseNotesShowRemaining(3),
          );
          expect(moreButton, findsOneWidget);
          await tester.ensureVisible(moreButton);
          expect(_insideWindow(tester, moreButton), isTrue);
          await tester.tap(moreButton);
          await tester.pumpAndSettle();
          expect(find.text('Feature 7'), findsOneWidget);
          final fewerButton = find.widgetWithText(
            GfButton,
            l10n.releaseNotesFewer,
          );
          await tester.ensureVisible(fewerButton);
          await tester.tap(fewerButton);
          await tester.pumpAndSettle();
          expect(find.text('Feature 7'), findsNothing);

          final finalRequiredAction = find.text('Required action 2');
          await tester.ensureVisible(finalRequiredAction);
          expect(_insideWindow(tester, finalRequiredAction), isTrue);

          final downloadButton = find.widgetWithText(
            FilledButton,
            l10n.updateDownload,
          );
          expect(downloadButton, findsOneWidget);
          expect(
            tester.getSemantics(downloadButton),
            matchesSemantics(
              label: l10n.updateDownload,
              isButton: true,
              hasTapAction: true,
              hasFocusAction: true,
              isFocusable: true,
              hasEnabledState: true,
              isEnabled: true,
            ),
          );
          expect(_insideWindow(tester, downloadButton), isTrue);
          await tester.tap(downloadButton);
          await tester.pump();
          expect(key.currentState!.downloadTapped, 1);

          key.currentState!.setStatus(working: true);
          await tester.pumpAndSettle();
          expect(find.text('Feature 0'), findsOneWidget);
          expect(find.text('Required action 2'), findsOneWidget);
          expect(find.text(l10n.updateDownload), findsNothing);
          expect(find.text(l10n.updateSkip), findsNothing);
          final cancelButton = find.widgetWithText(
            GfButton,
            l10n.updateCancelDownload,
          );
          // At 200% text the pinned action region scrolls on its own.
          await tester.ensureVisible(cancelButton);
          await tester.pumpAndSettle();
          expect(_insideWindow(tester, cancelButton), isTrue);
          await tester.tap(cancelButton);
          await tester.pump();
          expect(key.currentState!.cancelTapped, 1);

          key.currentState!.setStatus(failed: true);
          await tester.pumpAndSettle();
          expect(find.text(l10n.updateFailed), findsOneWidget);
          expect(find.text('Required action 2'), findsOneWidget);
          expect(find.text(l10n.updateRetry), findsOneWidget);

          key.currentState!.setStatus(needsPermission: true);
          await tester.pumpAndSettle();
          final permissionButton = find.widgetWithText(
            FilledButton,
            l10n.updateOpenPermissionSettings,
          );
          expect(permissionButton, findsOneWidget);
          // At 200% text the pinned action region scrolls on its own.
          await tester.ensureVisible(permissionButton);
          await tester.pumpAndSettle();
          expect(_insideWindow(tester, permissionButton), isTrue);
          await tester.tap(permissionButton);
          await tester.pump();
          expect(key.currentState!.permissionTapped, 1);
          expect(find.text('Required action 2'), findsOneWidget);

          key.currentState!.setStatus(ready: true);
          await tester.pumpAndSettle();
          expect(find.text(l10n.updateReady), findsOneWidget);
          expect(find.text('Required action 2'), findsOneWidget);
          expect(tester.takeException(), isNull);
          semantics.dispose();
        },
      );
    }
  }

  testWidgets('note levels never outsize the prompt title', (tester) async {
    const locale = Locale('zh');
    final l10n = await AppLocalizations.delegate.load(locale);
    await tester.pumpWidget(
      _testApp(
        brightness: Brightness.light,
        locale: locale,
        child: Scaffold(
          body: UpdatePromptSheet(
            version: '1.2.0',
            notes: const [
              ReleaseNote(
                id: 'required',
                title: 'Required title',
                summary: 'Required summary',
                platforms: {'android'},
                kind: 'security',
                required: true,
              ),
              ReleaseNote(
                id: 'fix',
                title: 'Fix title',
                summary: 'Fix summary',
                platforms: {'android'},
                kind: 'fix',
              ),
            ],
            historyComplete: true,
            primaryLabel: l10n.updateDownload,
            onPrimary: () {},
            onLater: () {},
          ),
        ),
      ),
    );
    double size(String text) => tester
        .renderObject<RenderParagraph>(find.text(text))
        .text
        .style!
        .fontSize!;
    final title = size(l10n.updateAvailable);
    final group = size(l10n.releaseNotesKindFix);
    final noteTitle = size('Fix title');
    final summary = size('Fix summary');
    expect(find.text(l10n.releaseNotesRequired), findsOneWidget);
    expect(noteTitle, lessThan(title));
    expect(summary, lessThanOrEqualTo(noteTitle));
    expect(group, lessThan(noteTitle));
  });

  testWidgets('a summary that restates its title is shown once', (
    tester,
  ) async {
    await tester.pumpWidget(
      _testApp(
        brightness: Brightness.light,
        locale: const Locale('zh'),
        child: const Scaffold(
          body: ReleaseNotesView(
            notes: [
              ReleaseNote(
                id: 'pr-9',
                title: '修复信息流跳回顶部',
                summary: '修复信息流跳回顶部。',
                platforms: {'android'},
                kind: 'fix',
              ),
            ],
          ),
        ),
      ),
    );
    expect(find.text('修复信息流跳回顶部'), findsOneWidget);
    expect(find.text('修复信息流跳回顶部。'), findsNothing);
  });

  testWidgets('update status is one stable live-region announcement', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    tester.view.physicalSize = const Size(640, 1280);
    tester.view.devicePixelRatio = 2;
    addTearDown(tester.view.reset);

    const locale = Locale('en');
    final l10n = await AppLocalizations.delegate.load(locale);
    final key = GlobalKey<_StatusHarnessState>();
    await tester.pumpWidget(
      _testApp(
        brightness: Brightness.dark,
        locale: locale,
        child: Scaffold(body: _StatusHarness(key: key)),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('update-status')), findsNothing);
    key.currentState!.setStatus(working: true);
    await tester.pump();
    _expectStatusSemantics(tester, l10n.updatePreparing);

    key.currentState!.setStatus(working: true, receivedBytes: 1);
    await tester.pump();
    _expectStatusSemantics(tester, l10n.updateDownloading);
    final firstDownloadStatus = tester.getSemantics(
      find.byKey(const ValueKey('update-status')),
    );
    key.currentState!.setStatus(working: true, receivedBytes: 2);
    await tester.pump();
    final nextDownloadStatus = tester.getSemantics(
      find.byKey(const ValueKey('update-status')),
    );
    expect(nextDownloadStatus.id, firstDownloadStatus.id);
    expect(nextDownloadStatus.label, l10n.updateDownloading);
    expect(find.bySemanticsLabel(l10n.updateDownloading), findsOneWidget);

    key.currentState!.setStatus(failed: true);
    await tester.pump();
    _expectStatusSemantics(tester, l10n.updateFailed);
    key.currentState!.setStatus(needsPermission: true);
    await tester.pump();
    _expectStatusSemantics(tester, l10n.updatePermission);
    key.currentState!.setStatus(ready: true);
    await tester.pump();
    _expectStatusSemantics(tester, l10n.updateReady);
    expect(tester.takeException(), isNull);
    semantics.dispose();
  });
}

MaterialApp _testApp({
  required Brightness brightness,
  required Locale locale,
  required Widget child,
}) => MaterialApp(
  theme: gfThemeData(brightness),
  locale: locale,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: MediaQuery(
    data: const MediaQueryData(
      size: Size(320, 640),
      textScaler: TextScaler.linear(2),
      disableAnimations: true,
    ),
    child: child,
  ),
);

bool _insideWindow(WidgetTester tester, Finder finder) {
  final rect = tester.getRect(finder);
  return rect.left >= 0 &&
      rect.top >= 0 &&
      rect.right <= 320 &&
      rect.bottom <= 640;
}

void _expectStatusSemantics(WidgetTester tester, String label) {
  final finder = find.byKey(const ValueKey('update-status'));
  expect(finder, findsOneWidget);
  expect(
    tester.getSemantics(finder),
    matchesSemantics(
      label: label,
      isLiveRegion: true,
      children: const <Matcher>[],
    ),
  );
}

class _StatusHarness extends StatefulWidget {
  const _StatusHarness({super.key});

  @override
  State<_StatusHarness> createState() => _StatusHarnessState();
}

class _StatusHarnessState extends State<_StatusHarness> {
  bool working = false;
  bool failed = false;
  bool needsPermission = false;
  bool ready = false;
  int receivedBytes = 0;

  void setStatus({
    bool working = false,
    bool failed = false,
    bool needsPermission = false,
    bool ready = false,
    int? receivedBytes,
  }) => setState(() {
    this.working = working;
    this.failed = failed;
    this.needsPermission = needsPermission;
    this.ready = ready;
    this.receivedBytes = receivedBytes ?? this.receivedBytes;
  });

  @override
  Widget build(BuildContext context) => Center(
    child: SizedBox(
      width: 280,
      child: UpdatePromptSheet(
        version: '1.2.0',
        notes: const [],
        historyComplete: true,
        working: working,
        failed: failed,
        needsPermission: needsPermission,
        ready: ready,
        receivedBytes: receivedBytes,
        primaryLabel: 'Download',
        onPrimary: () {},
        onLater: () {},
      ),
    ),
  );
}

class _DialogHarness extends StatefulWidget {
  const _DialogHarness({super.key, required this.notes});
  final List<ReleaseNote> notes;

  @override
  State<_DialogHarness> createState() => _DialogHarnessState();
}

class _DialogHarnessState extends State<_DialogHarness> {
  bool working = false;
  bool failed = false;
  bool needsPermission = false;
  bool ready = false;
  int receivedBytes = 0;
  int downloadTapped = 0;
  int permissionTapped = 0;
  int cancelTapped = 0;

  void setStatus({
    bool working = false,
    bool failed = false,
    bool needsPermission = false,
    bool ready = false,
    int? receivedBytes,
  }) => setState(() {
    this.working = working;
    this.failed = failed;
    this.needsPermission = needsPermission;
    this.ready = ready;
    this.receivedBytes = receivedBytes ?? this.receivedBytes;
  });

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      body: Align(
        alignment: Alignment.bottomCenter,
        child: UpdatePromptSheet(
          version: '1.2.0',
          installedVersion: '1.1.0',
          sizeBytes: 5000000,
          notes: widget.notes,
          historyComplete: false,
          working: working,
          failed: failed,
          needsPermission: needsPermission,
          ready: ready,
          receivedBytes: receivedBytes,
          primaryLabel: needsPermission
              ? l10n.updateOpenPermissionSettings
              : ready
              ? l10n.updateInstall
              : failed
              ? l10n.updateRetry
              : l10n.updateDownload,
          onPrimary: () {
            if (needsPermission) {
              permissionTapped++;
            } else {
              downloadTapped++;
            }
          },
          onCancel: () => cancelTapped++,
          onLater: () {},
          onSkip: () {},
        ),
      ),
    );
  }
}
