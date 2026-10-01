import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/topic_actions.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

void main() {
  for (final (width, scale, brightness) in [
    (320.0, 1.0, Brightness.light),
    (320.0, 2.0, Brightness.dark),
    (1024.0, 2.0, Brightness.light),
  ]) {
    testWidgets(
      'topic reading and overflow reflow at $width/$scale/$brightness',
      (tester) async {
        tester.view.physicalSize = Size(width, 800);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        final storage = MemoryTokenStorage();
        final client = GfApiClient(
          dio: Dio(),
          tokenStorage: storage,
          baseUrl: 'https://example.test',
        );
        final container = ProviderContainer(
          overrides: [
            tokenStorageProvider.overrideWithValue(storage),
            pageRepositoryProvider.overrideWithValue(
              FakePageRepository(client),
            ),
            topicRepositoryProvider.overrideWithValue(
              FakeTopicRepository(client),
            ),
            offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
          ],
        );
        addTearDown(container.dispose);
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: gfThemeData(brightness),
              locale: const Locale('en'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: TextScaler.linear(scale)),
                child: child!,
              ),
              home: const TopicPage(topicId: 100),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final title = tester.getRect(find.text('移动端测试话题').last);
        final author = tester.getRect(find.text('alice').first);
        expect(author.bottom, lessThanOrEqualTo(title.top));
        await tester.drag(
          find.byType(CustomScrollView).first,
          const Offset(0, -450),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final scroll = tester
            .state<ScrollableState>(
              find
                  .descendant(
                    of: find.byType(CustomScrollView).first,
                    matching: find.byType(Scrollable),
                  )
                  .first,
            )
            .position;
        final before = scroll.pixels;
        await tester.tap(
          find.descendant(
            of: find.byType(TopicActions),
            matching: find.byTooltip('More options'),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.text('Revision history'), findsOneWidget);
        expect(tester.takeException(), isNull);
        await tester.sendKeyEvent(LogicalKeyboardKey.escape);
        await tester.pumpAndSettle();
        expect(find.text('Revision history'), findsNothing);
        expect(scroll.pixels, before);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(seconds: 1));
      },
    );
  }
}
