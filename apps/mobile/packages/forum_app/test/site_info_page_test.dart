import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/info/site_info_page.dart';
import 'package:forum_app/src/providers.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Pages extends PageRepository {
  _Pages(super.client, this.kind, this.props);
  final SiteInfoKind kind;
  final Map<String, dynamic> props;
  final paths = <String>[];
  bool fail = false;
  @override
  Future<PagePayload> fetch(String path) async {
    paths.add(path);
    if (fail) throw StateError('offline');
    return PagePayload.fromJson({
      'component': '${kind.name}.index',
      'props': props,
      'meta': {'title': kind.name},
      'layout': minimalLayoutJson(),
      'url': path,
      'version': '1.0',
    });
  }
}

void main() {
  Future<void> pump(
    WidgetTester tester,
    _Pages pages,
    GfApiClient client,
  ) async {
    final container = ProviderContainer(
      overrides: [
        pageRepositoryProvider.overrideWithValue(pages),
        apiClientProvider.overrideWithValue(client),
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
          home: SiteInfoPage(kind: pages.kind),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  GfApiClient client() => GfApiClient(
    dio: Dio(),
    tokenStorage: MemoryTokenStorage(),
    baseUrl: 'https://example.test',
  );

  testWidgets(
    'privacy policy uses privacy payload and refuses executable links',
    (tester) async {
      final api = client();
      final pages = _Pages(api, SiteInfoKind.privacy, {
        'enabled': true,
        'contentHtml': '<h2>Your privacy</h2><p>Only necessary data.</p>',
      });
      await pump(tester, pages, api);
      expect(pages.paths, ['/privacy']);
      expect(find.text('Your privacy', findRichText: true), findsOneWidget);
      final html = tester.widget<HtmlWidget>(find.byType(HtmlWidget));
      expect(await html.onTapUrl!('javascript:alert(1)'), isTrue);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('disabled policy does not expose stale stored HTML', (
    tester,
  ) async {
    final api = client();
    await pump(
      tester,
      _Pages(api, SiteInfoKind.terms, {
        'enabled': false,
        'contentHtml': '<p>Not published</p>',
      }),
      api,
    );
    expect(find.text('No public content yet'), findsOneWidget);
    expect(find.text('Not published'), findsNothing);
  });

  testWidgets('links recover from an error and preserve group descriptions', (
    tester,
  ) async {
    final api = client();
    final pages = _Pages(api, SiteInfoKind.links, {
      'totalCount': 1,
      'groups': [
        {
          'name': 'Campus tools',
          'emoji': '',
          'color': '',
          'links': [
            {
              'name': 'Library',
              'desc': 'Books and opening hours',
              'url': 'https://library.example.test',
              'logoUrl': '',
            },
          ],
        },
      ],
    })..fail = true;
    await pump(tester, pages, api);
    expect(find.text('Failed to load'), findsOneWidget);
    pages.fail = false;
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(pages.paths, ['/links', '/links']);
    expect(find.text('Books and opening hours'), findsOneWidget);
  });
}
