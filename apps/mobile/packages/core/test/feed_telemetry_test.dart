import 'dart:convert';
import 'dart:io';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:core/core.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('new route mirrors parse the contract fixtures', () {
    final root = Directory.current.path;
    final events =
        jsonDecode(
              File(
                '$root/../../../../packages/api-contract/fixtures/feed-events-success.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final result = FeedEventsResponse.fromJson(events);
    expect(result.code, 0);
    expect(result.accepted, true);
    final summary =
        jsonDecode(
              File(
                '$root/../../../../packages/api-contract/fixtures/feed-summary-success.json',
              ).readAsStringSync(),
            )
            as Map<String, dynamic>;
    final parsed = FeedSummary.fromJson(
      summary['result'] as Map<String, dynamic>,
    );
    expect(parsed.rawRetentionDays, 30);
    expect(parsed.rolloutPercent, 20);
  });
  test(
    'account change clears attribution and a batch contains no client opened claim',
    () async {
      final telemetry = FeedTelemetry();
      telemetry.bindAccount(12);
      final topic = TopicPayload.fromJson({
        'id': 31,
        'title': 'topic',
        'url': '/p/post/31',
        'feedTrace': 'signed',
        'feedPosition': 0,
        'author': {'id': 1, 'username': 'author', 'avatarUrl': ''},
        'categories': [],
        'participants': [],
        'lastUpdateTime': '',
        'activityText': '',
        'viewCount': 0,
        'replyCount': 0,
        'likeCount': 0,
        'pinWeight': 0,
        'processStatus': 0,
        'description': '',
      });
      telemetry.select(topic);
      expect(telemetry.headers('/p/post/31')['X-Goose-Feed-Trace'], 'signed');
      expect(telemetry.headers('/p/post/310'), isEmpty);
      expect(telemetry.headers('https://external.invalid/p/post/31'), isEmpty);
      final batches = <Map<String, dynamic>>[];
      telemetry.send = (batch) async {
        batches.add(batch);
      };
      telemetry.visible(topic);
      await telemetry.flush();
      final patch = (batches.single['patches'] as List).single as Map;
      expect(patch['visibleMask'], 1);
      expect(patch.containsKey('openedMask'), false);
      telemetry.beginDetail(31);
      expect(
        telemetry.headers('/api/forum/topics/like')['X-Goose-Feed-Trace'],
        'signed',
      );
      telemetry.endDetail();
      expect(telemetry.headers('/p/post/31'), isEmpty);
      expect(telemetry.headers('/api/forum/topics/like'), isEmpty);
      telemetry.bindAccount(13);
      expect(telemetry.headers('/api/forum/topic-like'), isEmpty);
      telemetry.didChangeAppLifecycleState(AppLifecycleState.paused);
      telemetry.visible(topic);
      await telemetry.flush();
      expect(batches.length, 1);
      telemetry.bindAccount(0);
    },
  );
}
