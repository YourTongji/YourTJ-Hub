import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('moderation appeal item mirrors the contract without identity fields', () {
    final ModerationAppealItem item = ModerationAppealItem.fromJson({
      'id': 42,
      'kind': 'topic',
      'targetId': 7,
      'revisionId': 99,
      'topicId': 7,
      'postNo': 1,
      'title': 'candidate',
      'content': 'immutable rejected body',
      'images': ['https://img.example/a.png'],
      'reason': 'Please review',
      'status': 'pending',
      'reviewReason': '',
      'createdAt': '2026-10-18T08:00:00Z',
      'reviewedAt': null,
    });
    expect(item.id, 42);
    expect(item.kind, 'topic');
    expect(item.targetId, 7);
    expect(item.revisionId, 99);
    expect(item.images, ['https://img.example/a.png']);
    expect(item.status, 'pending');
    expect(item.reviewedAt, isNull);
  });

  test('history and queue results parse item lists and totals', () {
    final Map<String, dynamic> item = {
      'id': 1,
      'kind': 'post',
      'targetId': 3,
      'revisionId': 5,
      'topicId': 2,
      'postNo': 4,
      'title': 'topic title',
      'content': 'reply body',
      'images': <String>[],
      'reason': 'explanation',
      'status': 'approved',
      'reviewReason': 'decision explanation',
      'createdAt': '2026-10-18T08:00:00Z',
      'reviewedAt': '2026-10-18T09:00:00Z',
    };
    final ModerationAppealHistoryResult history =
        ModerationAppealHistoryResult.fromJson({
      'items': [item],
    });
    expect(history.items, hasLength(1));
    expect(history.items.single.postNo, 4);
    expect(history.items.single.reviewedAt, '2026-10-18T09:00:00Z');

    final ModerationAppealQueueResult queue =
        ModerationAppealQueueResult.fromJson({
      'items': [item],
      'total': 30,
      'page': 2,
      'pageSize': 20,
    });
    expect(queue.total, 30);
    expect(queue.page, 2);
    expect(queue.pageSize, 20);
    expect(queue.items.single.status, 'approved');
  });

  test('request bodies serialize to the contract shape', () {
    expect(
      const ModerationAppealSubmitRequest(
        kind: 'topic',
        id: 7,
        revisionId: 99,
        reason: 'why',
      ).toJson(),
      {
        'kind': 'topic',
        'id': 7,
        'revisionId': 99,
        'reason': 'why',
      },
    );
    expect(
      const ModerationAppealQueueRequest(page: 2, pageSize: 20, status: 'stale')
          .toJson(),
      {'page': 2, 'pageSize': 20, 'status': 'stale'},
    );
    expect(
      const ModerationAppealActionRequest(id: 42, approve: false, reason: 'no')
          .toJson(),
      {'id': 42, 'approve': false, 'reason': 'no'},
    );
  });
}
