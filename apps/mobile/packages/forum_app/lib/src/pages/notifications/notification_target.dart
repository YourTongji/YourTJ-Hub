import 'package:core/core.dart';

/// Reuse TopicPage's floor query; older payloads safely open the topic.
String? notificationTarget(NotificationPayload item) {
  final topicId = item.topic?.id ?? item.payload.topicId;
  if (topicId != null && topicId > 0) {
    final postNo = item.payload.postNo;
    return postNo != null && postNo > 0
        ? '/p/$topicId?postNo=$postNo'
        : '/p/$topicId';
  }
  if (item.eventType == 'follow' ||
      item.eventType == 'badge' ||
      item.payload.metadata?.profileUrl != null) {
    return item.actor.id > 0 ? '/u/${item.actor.id}' : null;
  }
  return null;
}
