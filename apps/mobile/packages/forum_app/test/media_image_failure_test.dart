import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/media_image_failure.dart';

/// Renders the host fallback the way `GfMediaScope.imageErrorBuilder` does.
Future<void> pumpFallback(
  WidgetTester tester, {
  required Object error,
  required VoidCallback retry,
  Locale locale = const Locale('zh'),
}) async {
  await tester.pumpWidget(
    MaterialApp(
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: Builder(
          builder: (context) => mediaImageFailure(
            context,
            error,
            retry,
            'https://example.test/a.png',
          ),
        ),
      ),
    ),
  );
}

void main() {
  test('classifyMediaFailure separates transport, format and policy failures', () {
    expect(
      classifyMediaFailure(StateError('Image request failed (403)')),
      GfMediaFailureKind.network,
    );
    expect(
      classifyMediaFailure(StateError('Image exceeds download limit')),
      GfMediaFailureKind.network,
    );
    expect(
      classifyMediaFailure(StateError('Media loading suspended during clear')),
      GfMediaFailureKind.blocked,
    );
    expect(
      classifyMediaFailure(const FormatException('Untrusted image origin')),
      GfMediaFailureKind.blocked,
    );
    expect(
      classifyMediaFailure(const FormatException('Invalid image data')),
      GfMediaFailureKind.format,
    );
    expect(
      classifyMediaFailure(StateError('Codec failed to produce an image')),
      GfMediaFailureKind.format,
    );
    expect(classifyMediaFailure(Exception('boom')), GfMediaFailureKind.unknown);
  });

  testWidgets('network failures offer retry and browser actions', (
    tester,
  ) async {
    var retried = 0;
    await pumpFallback(
      tester,
      error: StateError('Image request failed (403)'),
      retry: () => retried++,
    );

    expect(find.text('网络受限，暂时无法加载图片'), findsOneWidget);
    expect(find.text('在浏览器打开'), findsOneWidget);
    await tester.tap(find.text('重试'));
    await tester.pump();
    expect(retried, 1);
  });

  testWidgets('unsupported formats are told apart from network failures', (
    tester,
  ) async {
    await pumpFallback(
      tester,
      error: StateError('Codec failed to produce an image'),
      retry: () {},
    );

    expect(find.text('不支持的图片格式'), findsOneWidget);
    expect(find.text('网络受限，暂时无法加载图片'), findsNothing);
    expect(find.text('重试'), findsOneWidget);
  });

  testWidgets('copy is localized for every shipped locale', (tester) async {
    for (final Locale locale in AppLocalizations.supportedLocales) {
      await pumpFallback(
        tester,
        error: StateError('Image request failed (500)'),
        retry: () {},
        locale: locale,
      );
      expect(find.byType(TextButton), findsNWidgets(2));
      final AppLocalizations l10n = AppLocalizations.of(
        tester.element(find.byType(TextButton).first),
      );
      expect(find.text(l10n.imageUnavailableNetwork), findsOneWidget);
      expect(l10n.imageRetry, isNotEmpty);
      expect(l10n.imageOpenInBrowser, isNotEmpty);
      expect(l10n.imageUnavailable, isNotEmpty);
      expect(l10n.imageUnavailableFormat, isNotEmpty);
    }
  });
}
