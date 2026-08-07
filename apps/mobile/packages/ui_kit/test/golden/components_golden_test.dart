import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../golden_helper.dart';

/// Component-level golden baselines (390x844 mobile surface, Roboto).
///
/// Regenerate intentionally after a visual change:
/// `flutter test --update-goldens test/golden/components_golden_test.dart`
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('GfButton all variants', (tester) async {
    await pumpGfGolden(
      tester,
      Padding(
        padding: const EdgeInsets.all(16),
        child: Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final variant in GfButtonVariant.values)
              GfButton(label: variant.name, variant: variant, onPressed: () {}),
          ],
        ),
      ),
    );
    await expectLater(
      find.byType(Wrap),
      matchesGoldenFile('golden/gf_button_variants.png'),
    );
  });

  testWidgets('GfBadge all variants', (tester) async {
    await pumpGfGolden(
      tester,
      Padding(
        padding: const EdgeInsets.all(16),
        child: Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final variant in GfBadgeVariant.values)
              GfBadge(label: variant.name, variant: variant),
          ],
        ),
      ),
    );
    await expectLater(
      find.byType(Wrap),
      matchesGoldenFile('golden/gf_badge_variants.png'),
    );
  });

  testWidgets('GfSegmented control', (tester) async {
    await pumpGfGolden(
      tester,
      Padding(
        padding: const EdgeInsets.all(16),
        child: GfSegmented<String>(
          segments: const [
            ('登录', 'login'),
            ('注册', 'register'),
            ('找回', 'forgot'),
          ],
          selected: 'login',
          onSelected: (_) {},
        ),
      ),
    );
    await expectLater(
      find.byType(GfSegmented<String>),
      matchesGoldenFile('golden/gf_segmented.png'),
    );
  });

  testWidgets('GfTopicRow states', (tester) async {
    await pumpGfGolden(
      tester,
      SizedBox(
        width: 390,
        child: Column(
          children: [
            GfTopicRow(
              title: '同济大学樱花大道拍照攻略',
              description: '三月末的樱花大道,适合清晨人少时去',
              categories: const [
                GfTopicCategory(name: '校园生活', color: Color(0xFF00BC7D)),
              ],
              participantAvatarUrls: const ['a', 'b', 'c'],
              activityText: '3 小时前',
              replyCount: 42,
              hot: true,
              showDivider: false,
            ),
            GfTopicRow(
              title: '置顶的帖子标题',
              description: '',
              categories: const [],
              participantAvatarUrls: const [],
              activityText: '昨天',
              replyCount: 7,
              pinned: true,
              unseen: true,
            ),
          ],
        ),
      ),
    );
    await expectLater(
      find.byType(Column).first,
      matchesGoldenFile('golden/gf_topic_row_states.png'),
    );
  });

  testWidgets('GfFloatingControls', (tester) async {
    await pumpGfGolden(
      tester,
      Center(
        child: GfFloatingControls(
          actions: [
            GfTopicAction(
              icon: Icons.favorite_border,
              active: true,
              activeColor: GfColors.light.error,
              onTap: () {},
            ),
            GfTopicAction(
              icon: Icons.bookmark_border,
              active: false,
              activeColor: GfColors.light.primary,
              onTap: () {},
            ),
          ],
          onOpenReply: () {},
          currentNo: 3,
          maxNo: 120,
          onFloorTap: () {},
        ),
      ),
    );
    await expectLater(
      find.byType(GfFloatingControls),
      matchesGoldenFile('golden/gf_floating_controls.png'),
    );
  });

  testWidgets('GfNotificationRow states', (tester) async {
    await pumpGfGolden(
      tester,
      SizedBox(
        width: 390,
        child: Column(
          children: [
            GfNotificationRow(
              icon: Icons.message,
              tone: GfNotificationTone.primary,
              title: '有人回复了你的话题',
              subtitle: '内容预览…',
              time: '3 分钟前',
              unread: true,
            ),
            GfNotificationRow(
              icon: Icons.person_add,
              tone: GfNotificationTone.success,
              title: 'Alice 关注了你',
              subtitle: '',
              time: '昨天',
              unread: false,
            ),
          ],
        ),
      ),
    );
    await expectLater(
      find.byType(Column).first,
      matchesGoldenFile('golden/gf_notification_row_states.png'),
    );
  });

  testWidgets('GfMessageBubble states', (tester) async {
    await pumpGfGolden(
      tester,
      const Padding(
        padding: EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            GfMessageBubble(text: '对方的消息内容', mine: false),
            SizedBox(height: 8),
            GfMessageBubble(text: '我的回复内容', mine: true),
          ],
        ),
      ),
    );
    await expectLater(
      find.byType(Column),
      matchesGoldenFile('golden/gf_message_bubble_states.png'),
    );
  });

  testWidgets('GfAvatarStack sm', (tester) async {
    await pumpGfGolden(
      tester,
      const Padding(
        padding: EdgeInsets.all(16),
        child: GfAvatarStack(
          avatarUrls: ['a', 'b', 'c', 'd'],
          size: GfAvatarStackSize.sm,
        ),
      ),
    );
    await expectLater(
      find.byType(GfAvatarStack),
      matchesGoldenFile('golden/gf_avatar_stack_sm.png'),
    );
  });
}
