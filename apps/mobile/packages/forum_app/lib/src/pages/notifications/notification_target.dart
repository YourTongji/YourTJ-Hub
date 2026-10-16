import 'package:core/core.dart';
import '../../push/push_route.dart';

/// Reuse TopicPage's floor query; older payloads safely open the topic.
String? notificationTarget(NotificationPayload item) {
  if (item.eventType == 'review_rejected') return '/my-content';
  if (item.eventType == 'wiki_updated') {
    final route = item.payload.metadata?.profileUrl ?? item.topic?.url;
    return route != null && route.startsWith('/wiki/')
        ? pushRoute(route)
        : null;
  }
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
