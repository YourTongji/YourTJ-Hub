import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:core/core.dart';

import '../fixtures/page_fixtures.dart';
import '../golden_helper.dart';
import '../pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

/// Page-level golden baselines (390x844 mobile surface, Roboto, zh locale).
///
/// Regenerate intentionally after a visual change:
/// `flutter test --update-goldens test/golden/pages_golden_test.dart`
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  Future<ProviderContainer> makeContainer() async {
    final storage = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
        pageRepositoryProvider.overrideWithValue(FakePageRepository(client)),
        topicRepositoryProvider.overrideWithValue(FakeTopicRepository(client)),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  testWidgets('home page golden', (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page.png'),
    );
  });

  testWidgets('topic page golden', (tester) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const TopicPage(topicId: 100),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/topic_page.png'),
    );
    // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
    // 需推进时钟让其过期,避免 "Timer is still pending"。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });
}
