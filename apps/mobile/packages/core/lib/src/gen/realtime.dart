/// Hand-maintained mirror of the `/api/forum/events` SSE data contract.
/// The stream only invalidates REST snapshots; these payloads carry no chat
/// body, notification preview, or replay cursor.
enum ForumRealtimeEventType {
  hello('hello'),
  chatChanged('chat.changed'),
  notificationsChanged('notifications.changed'),
  unreadChanged('unread.changed'),
  sessionInvalidated('session.invalidated');

  const ForumRealtimeEventType(this.wireName);

  final String wireName;
}

class ForumRealtimeHello {
  const ForumRealtimeHello({
    required this.version,
    required this.heartbeatSeconds,
    required this.resync,
    required this.visibleRead,
  });

  final int version;
  final int heartbeatSeconds;
  final bool resync;
  final bool visibleRead;

  factory ForumRealtimeHello.fromJson(Map<String, dynamic> json) {
    final capabilities = json['capabilities'];
    return ForumRealtimeHello(
      version: json['version'] as int,
      heartbeatSeconds: json['heartbeatSeconds'] as int,
      resync: json['resync'] as bool,
      visibleRead: capabilities is Map<String, dynamic> &&
          capabilities['visibleRead'] == true,
    );
  }
}

class ForumRealtimeChatChanged {
  const ForumRealtimeChatChanged({required this.convId, required this.change});

  final int convId;
  final String change;

  factory ForumRealtimeChatChanged.fromJson(Map<String, dynamic> json) =>
      ForumRealtimeChatChanged(
        convId: json['convId'] as int,
        change: json['change'] as String,
      );
}

class ForumRealtimeNotificationsChanged {
  const ForumRealtimeNotificationsChanged({required this.change});

  final String change;

  factory ForumRealtimeNotificationsChanged.fromJson(Map<String, dynamic> json) =>
      ForumRealtimeNotificationsChanged(change: json['change'] as String);
}
