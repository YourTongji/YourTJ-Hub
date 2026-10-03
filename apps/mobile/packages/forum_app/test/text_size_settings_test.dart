import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/text_size_settings_page.dart';
import 'package:forum_app/src/reading_preferences.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

Future<ProviderContainer> _mount(
  WidgetTester tester, {
  Size size = const Size(390, 844),
  double systemScale = 1,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  final container = ProviderContainer(
    overrides: <Override>[
      contentFontScaleProvider.overrideWith(
        () => ContentFontScaleNotifier(persistDelay: Duration.zero),
      ),
      appFontScaleProvider.overrideWith(
        () => AppFontScaleNotifier(persistDelay: Duration.zero),
      ),
    ],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (BuildContext context, Widget? child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(systemScale)),
          child: AppTextScaleScope(child: child!),
        ),
        home: const TextSizeSettingsPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return container;
}

/// Painted size of the first sized span under [finder], optionally in the
/// paragraph that starts with [prefix].
double _painted(WidgetTester tester, Finder finder, {String? prefix}) {
  for (final Element element
      in find
          .descendant(of: finder, matching: find.byType(RichText))
          .evaluate()) {
    final RichText text = element.widget as RichText;
    if (prefix != null && !text.text.toPlainText().startsWith(prefix)) {
      continue;
    }
    double? size;
    text.text.visitChildren((InlineSpan span) {
      size = span.style?.fontSize;
      return size == null;
    });
    if (size != null) return text.textScaler.scale(size!);
  }
  throw StateError('No sized text under $finder');
}

/// Disposes the tree and drains the Markdown detectors' pending timers.
Future<void> _dispose(WidgetTester tester) async {
  await tester.pumpWidget(const SizedBox.shrink());
  await tester.pump(const Duration(milliseconds: 600));
}

Future<void> _dragToEnd(WidgetTester tester, String axis) async {
  await tester.drag(
    find.byKey(ValueKey('text-size-$axis')),
    const Offset(600, 0),
  );
  await tester.pumpAndSettle();
}

double _opacityOf(WidgetTester tester, Finder child) => tester
    .widget<AnimatedOpacity>(
      find.ancestor(of: child, matching: find.byType(AnimatedOpacity)).first,
    )
    .opacity;

void main() {
  setUp(() => SharedPreferences.setMockInitialValues(<String, Object>{}));

  final row = find.byKey(const ValueKey('text-size-preview-row'));
  final body = find.byKey(const ValueKey('text-size-preview-body'));
  final tabs = find.byKey(const ValueKey('text-size-preview-tabs'));
  final navigation = find.byKey(const ValueKey('text-size-preview-navigation'));
  final reset = find.byKey(const ValueKey('text-size-reset'));
  double rowTitle(WidgetTester tester) =>
      _painted(tester, row, prefix: 'Drag a slider');
  double bodyText(WidgetTester tester) =>
      _painted(tester, body, prefix: 'Reading text');

  testWidgets('preview shows real interface elements around the body', (
    tester,
  ) async {
    await _mount(tester);
    expect(tabs, findsOneWidget);
    expect(row, findsOneWidget);
    expect(navigation, findsOneWidget);
    expect(find.text('Reply'), findsOneWidget);
    await _dispose(tester);
  });

  testWidgets('app slider scales interface and body text together', (
    tester,
  ) async {
    final container = await _mount(tester);
    final double titleBefore = rowTitle(tester);
    final double bodyBefore = bodyText(tester);
    final double tabsBefore = _painted(tester, tabs);
    final double labelBefore = _painted(
      tester,
      find.byKey(const ValueKey('text-size-app-label')),
    );
    expect(find.text('Default'), findsNWidgets(2));

    await _dragToEnd(tester, 'app');

    expect(container.read(appFontScaleProvider), 1.3);
    expect(find.text('130%'), findsOneWidget);
    expect(rowTitle(tester), closeTo(titleBefore * 1.3, .01));
    expect(_painted(tester, tabs), closeTo(tabsBefore * 1.3, .01));
    expect(bodyText(tester), closeTo(bodyBefore * 1.3, .01));
    // The control panel stays at the default size under the finger.
    expect(
      _painted(tester, find.byKey(const ValueKey('text-size-app-label'))),
      closeTo(labelBefore, .01),
    );
    expect(tester.takeException(), isNull);
    await _dispose(tester);
  });

  testWidgets('reading slider changes only the body and dims the rest', (
    tester,
  ) async {
    final container = await _mount(tester);
    final double titleBefore = rowTitle(tester);
    final double bodyBefore = bodyText(tester);
    expect(_opacityOf(tester, tabs), 1);

    final TestGesture gesture = await tester.startGesture(
      tester.getCenter(find.byKey(const ValueKey('text-size-reading'))),
    );
    await gesture.moveBy(const Offset(600, 0));
    await tester.pumpAndSettle();
    expect(container.read(contentFontScaleProvider), 1.4);
    expect(_opacityOf(tester, tabs), lessThan(1));
    expect(_opacityOf(tester, row), lessThan(1));
    // Pinned chrome stays on screen around the scrolled body.
    final Rect preview = tester.getRect(
      find.byKey(const ValueKey('text-size-preview')),
    );
    expect(tester.getRect(tabs).top, closeTo(preview.top, 2));
    expect(tester.getRect(navigation).bottom, closeTo(preview.bottom, 2));
    final Rect viewport = tester.getRect(
      find.byKey(const ValueKey('text-size-preview-scroll')),
    );
    expect(tester.getRect(body).overlaps(viewport), isTrue);
    expect(bodyText(tester), greaterThan(bodyBefore * 1.3));
    expect(rowTitle(tester), closeTo(titleBefore, .01));

    await gesture.up();
    await tester.pump(const Duration(seconds: 1));
    await tester.pumpAndSettle();
    expect(_opacityOf(tester, tabs), 1);
    await _dispose(tester);
  });

  testWidgets('reset restores both sizes and persists them', (tester) async {
    final container = await _mount(tester);
    expect(tester.widget<TextButton>(reset).onPressed, isNull);

    await _dragToEnd(tester, 'app');
    await _dragToEnd(tester, 'reading');
    expect(tester.widget<TextButton>(reset).onPressed, isNotNull);
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getDouble(AppFontScaleNotifier.prefsKey), 1.3);
    expect(prefs.getDouble(ContentFontScaleNotifier.prefsKey), 1.4);

    await tester.tap(reset);
    await tester.pumpAndSettle();
    expect(container.read(appFontScaleProvider), 1);
    expect(container.read(contentFontScaleProvider), 1);
    expect(find.text('Default'), findsNWidgets(2));
    expect(tester.widget<TextButton>(reset).onPressed, isNull);
    expect(prefs.getDouble(AppFontScaleNotifier.prefsKey), 1);
    expect(prefs.getDouble(ContentFontScaleNotifier.prefsKey), 1);
    await tester.pump(const Duration(seconds: 1));
    await _dispose(tester);
  });

  testWidgets('narrow window with large system text does not overflow', (
    tester,
  ) async {
    final container = await _mount(
      tester,
      size: const Size(320, 568),
      systemScale: 2,
    );
    await _dragToEnd(tester, 'reading');
    container.read(appFontScaleProvider.notifier)
      ..setScale(1.3)
      ..persistScale();
    await tester.pump(const Duration(seconds: 1));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await _dispose(tester);
  });

  testWidgets('Android starts from the adapted default size', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    await _mount(tester);
    final RichText text = tester.widget<RichText>(
      find.descendant(of: row, matching: find.byType(RichText)).first,
    );
    expect(text.textScaler.scale(17), closeTo(16, .001));
    await _dispose(tester);
    debugDefaultTargetPlatformOverride = null;
  });
}
