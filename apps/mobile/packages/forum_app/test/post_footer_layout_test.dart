import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

void main() {
  for (final size in [(390.0, 1.0), (320.0, 1.0), (320.0, 2.0)]) {
    testWidgets(
      'reply footer keeps full content width at ${size.$1}/${size.$2}',
      (tester) async {
        SharedPreferences.setMockInitialValues({});
        tester.view.physicalSize = Size(size.$1, 1600);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final storage = MemoryTokenStorage();
        final client = GfApiClient(
          dio: Dio(),
          tokenStorage: storage,
          baseUrl: 'http://fake.local',
        );
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              tokenStorageProvider.overrideWithValue(storage),
              currentUserProvider.overrideWith((_) async => null),
              pageRepositoryProvider.overrideWithValue(
                FakePageRepository(client),
              ),
              topicRepositoryProvider.overrideWithValue(
                FakeTopicRepository(client),
              ),
              offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
            ],
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              locale: const Locale('en'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: TextScaler.linear(size.$2)),
                child: child!,
              ),
              home: const TopicPage(topicId: 100),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final actions = find.byType(PostActions).first;
        await tester.ensureVisible(actions);
        await tester.pumpAndSettle();
        final body = find.byWidgetPredicate(
          (widget) => widget is GfMarkdownView && widget.data == '独立回复',
        );
        final bounds = tester.getRect(actions);
        expect(bounds.left, closeTo(tester.getRect(body).left, .01));
        expect(bounds.width, closeTo(tester.getRect(body).width, .01));
        final glyphs = find.descendant(
          of: actions,
          matching: find.byType(GfSymbol),
        );
        expect(glyphs, findsNWidgets(5));
        final centers = [
          for (var index = 0; index < 5; index++)
            tester.getCenter(glyphs.at(index)),
        ];
        expect(centers.first.dx - bounds.left, 22);
        // A normal phone has enough room for all five actions on one line.
        if (size.$2 == 1) {
          expect(bounds.height, 44);
          for (final center in centers) {
            expect(center.dy, closeTo(centers.first.dy, .01));
          }
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 600));
      },
    );
  }
}
