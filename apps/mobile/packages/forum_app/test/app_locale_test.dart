import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_locale.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/language_picker.dart';

class _HeadersAdapter implements HttpClientAdapter {
  final languages = <Object?>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? stream,
    Future<void>? cancel,
  ) async {
    languages.add(options.headers['Accept-Language']);
    return ResponseBody.fromString(
      '{}',
      200,
      headers: {
        'content-type': ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));
  test('four complete catalogs preserve every placeholder', () {
    final template =
        jsonDecode(File('lib/l10n/app_en.arb').readAsStringSync())
            as Map<String, dynamic>;
    final keys = template.keys.where((k) => !k.startsWith('@')).toSet();
    for (final language in appLanguageNames.keys) {
      final catalog =
          jsonDecode(File('lib/l10n/app_$language.arb').readAsStringSync())
              as Map<String, dynamic>;
      expect(catalog.keys.where((k) => !k.startsWith('@')).toSet(), keys);
      for (final key in keys) {
        Set<String?> parameters(String value) =>
            RegExp(r'\{(\w+)\}').allMatches(value).map((m) => m[1]).toSet();
        expect(catalog[key], isNotEmpty, reason: '$language:$key');
        expect(
          parameters(catalog[key]),
          parameters(template[key]),
          reason: '$language:$key',
        );
      }
    }
    expect(
      AppLocalizations.supportedLocales.map((l) => l.languageCode).toSet(),
      appLanguageNames.keys.toSet(),
    );
  });
  test('course copy and term names follow the selected language', () async {
    final ja = CourseCopy(
      await AppLocalizations.delegate.load(const Locale('ja')),
    );
    final de = CourseCopy(
      await AppLocalizations.delegate.load(const Locale('de')),
    );
    expect(ja.done, '完了');
    expect(de.done, 'Fertig');
    expect(ja.summaryConsensus('unknown'), 'unknown');
    expect(de.summaryConsensus('recommend'), isNot('Recommended'));
    expect(de.selectedCount(3), contains('3'));
    expect(shortTerm('2025-2026-1', locale: 'de'), '25 WiSe');
    expect(shortTerm('2025-2026-2', locale: 'en'), '25 Spring');
    expect(shortTerm('2025-2026-2', locale: 'ja'), '25春');
    expect(shortTerm('custom', locale: 'de'), 'custom');
  });
  test('regional locale resolution follows Web and falls back to Chinese', () {
    expect(normalizeAppLocale('DE-at'), const Locale('de'));
    expect(
      resolveAppLocale(null, [const Locale('fr'), const Locale('ja', 'JP')]),
      const Locale('ja'),
    );
    expect(resolveAppLocale(null, [const Locale('fr')]), const Locale('zh'));
    expect(
      resolveAppLocale(const Locale('en'), [const Locale('de')]),
      const Locale('en'),
    );
  });
  test(
    'manual selection survives restore and persists across containers',
    () async {
      SharedPreferences.setMockInitialValues({'app_locale': 'ja'});
      final container = ProviderContainer();
      container.read(appLocaleProvider.notifier).setLocale(const Locale('de'));
      await Future<void>.delayed(Duration.zero);
      expect(container.read(appLocaleProvider), const Locale('de'));
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString('app_locale'), 'de');
      container.dispose();
      final restored = ProviderContainer();
      expect(restored.read(appLocaleProvider), isNull);
      await Future<void>.delayed(Duration.zero);
      expect(restored.read(appLocaleProvider), const Locale('de'));
      restored.read(appLocaleProvider.notifier).setLocale(null);
      await Future<void>.delayed(Duration.zero);
      expect(prefs.getString('app_locale'), isNull);
      restored.dispose();
    },
  );
  test(
    'API and sign-in requests use the current language without rebuilding clients',
    () async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      for (final provider in [dioProvider, authDioProvider]) {
        final dio = container.read(provider);
        final adapter = _HeadersAdapter();
        dio.httpClientAdapter = adapter;
        container
            .read(appLocaleProvider.notifier)
            .setLocale(const Locale('ja'));
        await dio.get('https://example.test/');
        container
            .read(appLocaleProvider.notifier)
            .setLocale(const Locale('de'));
        await dio.get('https://example.test/');
        expect(adapter.languages, ['ja', 'de']);
        expect(identical(container.read(provider), dio), isTrue);
      }
    },
  );
  testWidgets(
    'language picker updates strings while retaining in-progress input',
    (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(appLocaleProvider.notifier).setLocale(const Locale('en'));
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: Consumer(
            builder: (context, ref, _) => MaterialApp(
              locale: ref.watch(appLocaleProvider),
              supportedLocales: AppLocalizations.supportedLocales,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              home: Scaffold(
                body: Builder(
                  builder: (context) => Column(
                    children: [
                      const TextField(key: ValueKey('draft')),
                      TextButton(
                        onPressed: () => showAppLanguagePicker(context),
                        child: Text(
                          AppLocalizations.of(context).settingsAppLanguage,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'unsaved');
      await tester.tap(find.text('App language'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('日本語'));
      await tester.pumpAndSettle();
      expect(find.text('アプリの言語'), findsOneWidget);
      expect(find.text('unsaved'), findsOneWidget);
      expect(container.read(appLocaleProvider), const Locale('ja'));
    },
  );
}
