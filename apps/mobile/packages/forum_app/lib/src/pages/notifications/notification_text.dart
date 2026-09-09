import 'package:core/core.dart';
import '../../../l10n/app_localizations.dart';

/// Same template/event semantics as Web NotificationsPage. Protocol keys are
/// never user-facing fallback copy; older literal titles remain supported.
(String, String) notificationText(
  NotificationPayload item,
  AppLocalizations l10n,
) {
  String literal(String? value) => (value ?? '').trim();
  // Only title fields can contain unresolved protocol keys. Content, actor
  // names, previews and badges are user data, even when they use this prefix.
  String heading(String? value) {
    final text = literal(value);
    return text.startsWith('notifications.') ? '' : text;
  }

  final actor =
      [
            item.actor.username,
            item.payload.actorName,
            item.payload.metadata?.followerName,
          ]
          .map(literal)
          .firstWhere(
            (s) => s.isNotEmpty,
            orElse: () => l10n.notificationSomeone,
          );
  final key = item.payload.templateKey ?? '';
  final event = switch (key) {
    'notifications.templates.comment' => 'comment',
    'notifications.templates.mention' => 'mention',
    'notifications.templates.postReply' => 'post_reply',
    'notifications.templates.topicPost' => 'topic_post',
    'notifications.templates.follow' => 'follow',
    'notifications.templates.badge' => 'badge',
    'notifications.templates.like' => 'like',
    'notifications.templates.wikiUpdated' => 'wiki_updated',
    _ => item.eventType,
  };
  final badge = literal(item.payload.metadata?.badgeName);
  final legacyTitle = [
    item.title,
    item.payload.title,
  ].map(heading).firstWhere((s) => s.isNotEmpty, orElse: () => '');
  final title = key.isEmpty && legacyTitle.isNotEmpty
      ? legacyTitle
      : switch (event) {
          'comment' => l10n.notificationComment(actor),
          'mention' => l10n.notificationMention(actor),
          'post_reply' => l10n.notificationPostReply(actor),
          'topic_post' => l10n.notificationTopicPost(actor),
          'follow' => l10n.notificationFollow(actor),
          'like' => l10n.notificationLike(actor),
          'wiki_updated' => l10n.notificationWikiUpdated(actor),
          'badge' =>
            badge.isEmpty
                ? l10n.notificationBadgeUnnamed
                : l10n.notificationBadge(badge),
          _ =>
            [item.title, item.payload.title]
                .map(heading)
                .firstWhere(
                  (s) => s.isNotEmpty,
                  orElse: () => l10n.notificationNew,
                ),
        };
  final subtitle =
      [
            item.content,
            item.payload.content,
            item.payload.templateParams?.preview,
            item.topic?.title,
            item.payload.topicTitle,
          ]
          .map(literal)
          .firstWhere((s) => s.isNotEmpty && s != title, orElse: () => '');
  return (title, subtitle);
}
