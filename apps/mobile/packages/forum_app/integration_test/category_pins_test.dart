import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import '../test/category_pins_test.dart'
    show CategoryPinsRepository, categoryPinsApp, categoryPinsRouter;

/// Real category/Home widgets with clearly labeled synthetic, non-network data.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets('category pins on iOS in both themes', (tester) async {
    SharedPreferences.setMockInitialValues({});
    for (final brightness in Brightness.values) {
      final theme = brightness.name;
      final pages = CategoryPinsRepository(hasNext: true);
      final router = categoryPinsRouter();
      await tester.pumpWidget(categoryPinsApp(pages, router, brightness));
      await tester.pumpAndSettle();
      // A native viewport may prefetch page two before the first assertion.
      tester.widget<GfTopicList>(find.byType(GfTopicList)).onLoadMore();
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('pinned-topic-strip')), findsOneWidget);
      expect(find.text('置顶公告 1（演示）'), findsOneWidget);
      expect(find.text('置顶公告 2（演示）'), findsNothing);
      expect(find.byType(GfTopicCard), findsNWidgets(2));
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot('category-$theme-collapsed');
      }
      await tester.tap(find.byKey(const ValueKey('pinned-strip-line')));
      await tester.pumpAndSettle();
      expect(find.text('置顶公告 2（演示）'), findsOneWidget);
      expect(find.byType(GfTopicCard), findsNWidgets(2));
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot('category-$theme-expanded');
      }
      tester.widget<GfTopicList>(find.byType(GfTopicList)).onLoadMore();
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<GfTopicList>(find.byType(GfTopicList))
            .topics
            .map((topic) => topic.id),
        [101, 102, 103, 104],
      );
      expect(find.text('置顶公告 2（演示）'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      router.dispose();

      final homeRouter = categoryPinsRouter(home: true, tickersEnabled: true);
      await tester.pumpWidget(
        categoryPinsApp(CategoryPinsRepository(), homeRouter, brightness),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('home-category-1')));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('pinned-topic-strip')), findsOneWidget);
      if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
        await binding.takeScreenshot('home-category-$theme');
      }
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      homeRouter.dispose();

      for (final pins in [0, 1, 3]) {
        final boundaryRouter = categoryPinsRouter();
        await tester.pumpWidget(
          categoryPinsApp(
            CategoryPinsRepository(pins: pins),
            boundaryRouter,
            brightness,
          ),
        );
        await tester.pumpAndSettle();
        expect(
          find.byKey(const ValueKey('pinned-topic-strip')),
          pins == 0 ? findsNothing : findsOneWidget,
        );
        expect(find.byType(GfTopicCard), findsNWidgets(3 - pins));
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
        boundaryRouter.dispose();
      }
    }
  });
}
