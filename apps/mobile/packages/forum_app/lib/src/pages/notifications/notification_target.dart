import 'package:core/core.dart';
import '../../push/push_route.dart';
import 'notification_text.dart';

/// Reuse TopicPage's floor query; older payloads safely open the topic.
String? notificationTarget(NotificationPayload item) {
  // 与 Web NotificationsPage 的深链判定一致：审核拒绝、申诉驳回/失效与申诉受理回执
  // 都落在“我的内容”；申诉通过（review_approved）按话题打开已公开的内容。
  if (item.eventType == 'review_rejected' ||
      notificationAppeal(item) == 'appealSubmitted') {
    return '/my-content';
  }
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
