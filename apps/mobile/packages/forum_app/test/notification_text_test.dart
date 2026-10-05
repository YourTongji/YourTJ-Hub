import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_ja.dart';
import 'package:forum_app/l10n/app_localizations_de.dart';
import 'package:forum_app/src/pages/notifications/notification_target.dart';
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
  test(
    'review copy uses the reviewed subject instead of legacy preview text',
    () {
      for (final event in [
        'review_pending',
        'review_approved',
        'review_rejected',
      ]) {
        final subject = event == 'review_rejected' ? '首******尾' : '无标题正文摘要';
        final copy = notificationText(
          notification(
            event: event,
            topicTitle: subject,
            content: 'Legacy raw preview',
          ),
          AppLocalizationsEn(),
        );
        expect(copy.$2, startsWith(subject));
        expect(copy.$2, isNot(contains('Legacy raw preview')));
        expect(copy.$2, isNot(contains('Actual preview')));
      }
    },
  );

  test('manual review and rejection explain the next step in all locales', () {
    for (final l10n in [
      AppLocalizationsZh(),
      AppLocalizationsEn(),
      AppLocalizationsJa(),
      AppLocalizationsDe(),
    ]) {
      final pending = notificationText(
        notification(template: 'notifications.templates.reviewPending'),
        l10n,
      );
      expect(pending.$1, l10n.notificationReviewPending);
      expect(pending.$2, contains(l10n.notificationReviewPendingDetail));
      final rejected = notificationText(
        notification(template: 'notifications.templates.reviewRejected'),
        l10n,
      );
      expect(rejected.$1, l10n.notificationReviewRejected);
      expect(rejected.$2, contains(l10n.notificationReviewRejectedDetail));
    }
  });

  test('mention text in every locale and stable floor navigation', () {
    for (final locale in <AppLocalizations>[
      AppLocalizationsZh(),
      AppLocalizationsEn(),
      AppLocalizationsJa(),
      AppLocalizationsDe(),
    ]) {
      final text = notificationText(
        notification(
          event: 'mention',
          template: 'notifications.templates.mention',
        ),
        locale,
      );
      expect(text.$1, locale.notificationMention('Alice'));
      expect(text.$2, 'Actual preview');
    }
    for (final floor in <int?>[null, 0, 8]) {
      final item = NotificationPayload.fromJson({
        'id': 2048,
        'eventType': 'mention',
        'isRead': false,
        'createdAt': '',
        'title': '',
        'content': '',
        'actor': {'id': 1024, 'username': 'Alice'},
        'payload': {
          'actorId': 1024,
          'topicId': 512,
          'postId': 4096,
          'postNo': ?floor,
          'templateKey': 'notifications.templates.mention',
        },
      });
      expect(item.payload.templateKey, 'notifications.templates.mention');
      expect(
        notificationTarget(item),
        floor == 8 ? '/p/512?postNo=8' : '/p/512',
      );
    }
    expect(
      notificationTarget(notification(event: 'review_rejected')),
      '/my-content',
    );
    expect(notificationTarget(notification(event: 'mention')), isNull);
    expect(notificationTarget(notification(event: 'follow')), '/u/1');
  });
  test('actor nickname takes precedence for note(display name)', () {
    final en = AppLocalizationsEn();
    final item = NotificationPayload(
      id: 1,
      eventType: 'post_reply',
      isRead: false,
      createdAt: '',
      title: '',
      content: '',
      actor: const NotificationActorPayload(
        id: 7,
        username: 'alice',
        nickname: '昵称甲',
      ),
      payload: const NotificationInnerPayload(
        actorId: 7,
        templateKey: 'notifications.templates.postReply',
      ),
    );
    // 通知标题按当前昵称渲染，而不是存储的用户名。
    expect(notificationText(item, en).$1, en.notificationPostReply('昵称甲'));
    // 备注渲染回调拿到的是昵称，保证 note(display name)。
    expect(
      notificationActorName(
        item,
        en,
        displayName: (id, username) => '备注($username)',
      ),
      '备注(昵称甲)',
    );
  });

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
