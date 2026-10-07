import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Agent event page decodes public data and encodes the event envelope', () {
    final page = AgentEventPage.fromJson({
      'events': [
        {
          'id': 'evt_1',
          'instanceId': 'dev-a',
          'schemaVersion': 1,
          'type': 'agent.mentioned',
          'occurredAt': '2026-10-04T10:00:00Z',
          'agentId': 34,
          'state': 'active',
          'data': {
            'topicId': 12,
            'postId': 44,
            'postNo': 3,
            'replyToPostId': 42,
            'actorId': 21,
            'actorType': 'human',
            'reasons': ['mention'],
            'url': '/p/post/12/3',
          },
        },
      ],
      'nextCursor': 'opaque-cursor',
      'hasMore': true,
      'replayFloor': 'floor-cursor',
    });

    expect(page.events.single.data?.reasons, ['mention']);
    expect(page.events.single.toJson()['instanceId'], 'dev-a');
    expect(page.toJson()['nextCursor'], 'opaque-cursor');
  });

  test('event write inputs keep the idempotency key out of the JSON body', () {
    final topic = AgentWriteTopicInput(
      title: 'A useful topic',
      content: 'Markdown',
      categoryId: [2],
      sourceEventId: 'evt_1',
      idempotencyKey: 'topic:evt_1',
    );
    final post = AgentCreatePostInput(
      content: 'A reply',
      replyToPostId: 44,
      sourceEventId: 'evt_1',
      idempotencyKey: 'reply:evt_1',
    );

    expect(topic.toJson(), {
      'title': 'A useful topic',
      'content': 'Markdown',
      'categoryId': [2],
      'sourceEventId': 'evt_1',
    });
    expect(post.toJson(), {
      'content': 'A reply',
      'replyToPostId': 44,
      'sourceEventId': 'evt_1',
    });
    expect(AgentAckEventsInput(eventIds: ['evt_1']).toJson(), {
      'eventIds': ['evt_1'],
    });
  });
}
