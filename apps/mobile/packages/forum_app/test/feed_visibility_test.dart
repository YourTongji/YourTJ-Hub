import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:core/core.dart';
import 'package:forum_app/src/widgets/feed_visibility.dart';
import 'fixtures/page_fixtures.dart';

void main() {
  testWidgets(
    'visible rows require a continuous foreground second and recycle context',
    (tester) async {
      final telemetry = FeedTelemetry.instance;
      telemetry.bindAccount(12);
      final batches = <Map<String, dynamic>>[];
      telemetry.send = (batch) async {
        batches.add(batch);
      };
      final topic = parsePageProps<HomeProps>(
        parsePayload(homePayloadJson()),
      )!.topics.first.copyWith(feedTrace: 'signed', feedPosition: 0);
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: ListView(
              children: [
                FeedVisibility(
                  topic: topic,
                  reason: '来自你关注的人',
                  child: const SizedBox(height: 100),
                ),
              ],
            ),
          ),
        ),
      );
      expect(find.text('来自你关注的人'), findsOneWidget);
      await tester.pump(const Duration(milliseconds: 100));
      await tester.pump(const Duration(milliseconds: 900));
      await telemetry.flush();
      expect(batches, isEmpty);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump(const Duration(seconds: 2));
      await telemetry.flush();
      expect(batches, isEmpty);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump(const Duration(milliseconds: 100));
      await tester.pump(const Duration(seconds: 1));
      await telemetry.flush();
      expect((batches.single['patches'] as List).single['visibleMask'], 1);
      await tester.pumpWidget(const SizedBox());
      telemetry.bindAccount(0);
      telemetry.send = null;
    },
  );
  testWidgets('briefly scrolling out of view restarts the full second', (
    tester,
  ) async {
    final telemetry = FeedTelemetry.instance;
    telemetry.bindAccount(12);
    final batches = <Map<String, dynamic>>[];
    telemetry.send = (batch) async {
      batches.add(batch);
    };
    final controller = ScrollController();
    final topic = parsePageProps<HomeProps>(
      parsePayload(homePayloadJson()),
    )!.topics.first.copyWith(feedTrace: 'scroll-trace', feedPosition: 0);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            height: 200,
            child: ListView(
              controller: controller,
              children: [
                FeedVisibility(
                  topic: topic,
                  child: const SizedBox(height: 100),
                ),
                const SizedBox(height: 1000),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 500));
    controller.jumpTo(200);
    await tester.pump(const Duration(milliseconds: 1));
    controller.jumpTo(0);
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 500));
    await telemetry.flush();
    expect(batches, isEmpty);
    await tester.pump(const Duration(seconds: 1));
    await telemetry.flush();
    expect((batches.single['patches'] as List).single['visibleMask'], 1);
    await tester.pumpWidget(const SizedBox());
    controller.dispose();
    telemetry.bindAccount(0);
    telemetry.send = null;
  });
}
