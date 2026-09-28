import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

/// 审核 actor 镜像与契约（TopicAuthorPayload / ReportHandlerPayload）对齐：
/// nickname 可选，缺失时不得影响反序列化（issue #837）。
void main() {
  test('ModerationLogActor keeps the optional nickname', () {
    final actor = ModerationLogActor.fromJson({
      'id': 7,
      'username': 'alice',
      'nickname': '昵称甲',
      'avatarUrl': '/static/pic/1.webp',
    });
    expect(actor.nickname, '昵称甲');
    expect(actor.toJson()['nickname'], '昵称甲');

    final plain = ModerationLogActor.fromJson({
      'id': 8,
      'username': 'bob',
      'avatarUrl': '/static/pic/1.webp',
    });
    expect(plain.nickname, isNull);
  });

  test('ModerationReportItem parses reporter and handler nicknames', () {
    final item = ModerationReportItem.fromJson({
      'id': 1,
      'targetType': 'post',
      'targetId': 2,
      'targetUrl': '/p/post/2',
      'title': '标题',
      'excerpt': '摘要',
      'reason': 'spam',
      'note': '',
      'status': 'open',
      'resolution': '',
      'reporter': {
        'id': 7,
        'username': 'alice',
        'nickname': '昵称甲',
        'avatarUrl': '/static/pic/1.webp',
      },
      'handler': {
        'id': 8,
        'username': 'bob',
        'avatarUrl': '/static/pic/1.webp',
      },
      'categories': <Object>[],
      'createdAt': '2026-09-27T08:00:00Z',
    });
    expect(item.reporter.nickname, '昵称甲');
    expect(item.handler.nickname, isNull);
  });
}
