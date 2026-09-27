import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('SSE hello requires REST reconciliation and advertises visible read', () {
    final hello = ForumRealtimeHello.fromJson({
      'version': 1,
      'heartbeatSeconds': 15,
      'resync': true,
      'capabilities': {'visibleRead': true},
    });
    expect(hello.resync, isTrue);
    expect(hello.heartbeatSeconds, 15);
    expect(hello.visibleRead, isTrue);
  });

  test('chat change carries only the conversation identity and hint', () {
    final changed = ForumRealtimeChatChanged.fromJson({
      'convId': 42,
      'change': 'received',
    });
    expect(changed.convId, 42);
    expect(changed.change, 'received');
    expect(ForumRealtimeEventType.chatChanged.wireName, 'chat.changed');
  });
}
