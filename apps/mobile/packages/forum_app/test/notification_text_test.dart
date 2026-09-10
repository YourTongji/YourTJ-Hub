import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/pages/notifications/notification_text.dart';

NotificationPayload notification({
  String event = 'unknown',
  String? template,
  String title = 'notifications.templates.unknown',
  String actor = 'Alice',
  String content = '',
  String? topicTitle,
  String? payloadTitle,
  NotificationMetadata? metadata,
}) => NotificationPayload(
  id: 1,
  eventType: event,
  isRead: false,
  createdAt: '',
  title: title,
  content: content,
  actor: NotificationActorPayload(id: 1, username: actor),
  payload: NotificationInnerPayload(
    actorId: 1,
    title: payloadTitle,
    topicTitle: topicTitle,
    templateKey: template,
    metadata: metadata,
    templateParams: const NotificationTemplateParams(preview: 'Actual preview'),
  ),
);

void main() {
  test('user text beginning with notifications stays literal', () {
    final en = AppLocalizationsEn();
    final item = notification(
      event: 'comment',
      content: 'notifications.example',
    );
    expect(notificationText(item, en).$2, 'notifications.example');
    expect(
      notificationText(
        notification(
          event: 'badge',
          metadata: const NotificationMetadata(
            badgeName: 'notifications.badge',
          ),
        ),
        en,
      ).$1,
      en.notificationBadge('notifications.badge'),
    );
  });
  test(
    'legacy recognized events prefer literal headings from either field',
    () {
      final en = AppLocalizationsEn();
      expect(
        notificationText(
          notification(event: 'post_reply', title: 'Reply from legacy'),
          en,
        ).$1,
        'Reply from legacy',
      );
      expect(
        notificationText(
          notification(event: 'comment', payloadTitle: 'Legacy payload'),
          en,
        ).$1,
        'Legacy payload',
      );
    },
  );
  test(
    'topic notifications preserve comment previews ahead of topic titles',
    () {
      final en = AppLocalizationsEn();
      expect(
        notificationText(
          notification(
            event: 'comment',
            topicTitle: 'Topic',
            content: 'Reply text',
          ),
          en,
        ).$2,
        'Reply text',
      );
      expect(
        notificationText(
          notification(event: 'comment', topicTitle: 'Topic'),
          en,
        ).$2,
        'Actual preview',
      );
    },
  );

  test('all Web template keys resolve in both supported locales', () {
    for (final l10n in [AppLocalizationsEn(), AppLocalizationsZh()]) {
      for (final key in [
        'comment',
        'postReply',
        'mention',
        'topicPost',
        'follow',
        'like',
        'wikiUpdated',
        'badge',
      ]) {
        final (title, subtitle) = notificationText(
          notification(
            template: 'notifications.templates.$key',
            metadata: const NotificationMetadata(badgeName: '维护者'),
          ),
          l10n,
        );
        expect(title, isNot(contains('notifications.')));
        expect(title, contains(key == 'badge' ? '维护者' : 'Alice'));
        expect(subtitle, 'Actual preview');
      }
    }
  });
  test('event fallback and legacy literal titles remain readable', () {
    final en = AppLocalizationsEn();
    expect(
      notificationText(notification(event: 'badge'), en).$1,
      en.notificationBadgeUnnamed,
    );
    expect(
      notificationText(notification(event: 'post_reply', actor: ''), en).$1,
      en.notificationPostReply(en.notificationSomeone),
    );
    expect(
      notificationText(notification(title: 'Service update'), en).$1,
      'Service update',
    );
    expect(notificationText(notification(), en).$1, en.notificationNew);
    expect(
      notificationText(
        notification(
          event: 'follow',
          actor: '',
          metadata: const NotificationMetadata(followerName: 'Bob'),
        ),
        en,
      ).$1,
      en.notificationFollow('Bob'),
    );
  });
}
