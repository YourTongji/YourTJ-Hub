import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../golden_helper.dart';

/// Component-level golden baselines (390x844 mobile surface, Roboto).
///
/// Tagged `golden` and excluded from `melos run test` (the CI gate) via
/// `--exclude-tags=golden`. Regenerate intentionally after a visual change:
/// `flutter test --update-goldens test/golden/components_golden_test.dart`
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  // Golden baselines are rendered by flutter_tester on Linux CI (see
  // golden_helper.dart). flutter_tester rasterizes text differently per
  // host OS (hinting/AA), so the same PNG cannot match on macOS; run
  // goldens on Linux and skip elsewhere.
  final bool skipGoldens = !Platform.isLinux;

  for (final brightness in Brightness.values) {
    testWidgets(
      'Profile presentation ${brightness.name}',
      skip: skipGoldens,
      tags: 'golden',
      (tester) async {
        await pumpGfGolden(
          tester,
          SingleChildScrollView(
            child: GfUserCard(
              avatarUrl: '',
              name: 'Grey Goose',
              username: 'greygoose',
              bio: '  在同济，记录日常，也分享一点新发现。\n\n',
              signature: '保持好奇，慢慢前行。',
              coloredBadges: const [
                GfUserBadge(label: '管理员', color: Color(0xFFF59E0B)),
              ],
              details: const Row(
                spacing: 18,
                children: [
                  GfSocialIcon('github'),
                  GfSocialIcon('twitter'),
                  GfSocialIcon('linkedIn'),
                  GfSocialIcon('weibo'),
                  GfSocialIcon('bilibili'),
                  GfSocialIcon('zhihu'),
                ],
              ),
              actions: Row(
                spacing: 8,
                children: [
                  GfButton(
                    label: '已关注',
                    variant: GfButtonVariant.secondary,
                    icon: const GfSymbol('user-round-check', size: 18),
                    onPressed: () {},
                  ),
                  GfButton(
                    label: '新消息',
                    variant: GfButtonVariant.secondary,
                    icon: const GfSymbol('mail', size: 18),
                    onPressed: () {},
                  ),
                ],
              ),
              stats: const [
                ('主题', '24'),
                ('回复', '108'),
                ('点赞', '256'),
                ('关注', '108'),
                ('粉丝', '13'),
              ],
            ),
          ),
          brightness: brightness,
        );
        await expectLater(
          find.byType(Scaffold),
          matchesGoldenFile('golden/gf_profile_${brightness.name}.png'),
        );
      },
    );
    testWidgets(
      'Activity badges and settings ${brightness.name}',
      skip: skipGoldens,
      tags: 'golden',
      (tester) async {
        await pumpGfGolden(
          tester,
          SingleChildScrollView(
            child: Column(
              children: [
                const GfActivityCard(
                  symbol: 'heart',
                  title: '点赞内容：孩子们，挂科了别忘记重考申请。',
                  time: '2026-09-07 14:07',
                  color: Color(0xFFE11D48),
                ),
                const GfActivityCard(
                  symbol: 'message-circle',
                  title: '参与回复：一起分享新学期的选课经验。',
                  time: '2026-09-07 12:37',
                  color: Color(0xFF059669),
                ),
                const Padding(
                  padding: EdgeInsets.all(16),
                  child: GfAchievementCard(
                    title: '社区维护者',
                    description: '协助维护社区秩序',
                    color: Color(0xFF059669),
                    icon: GfSymbol(
                      'shield-check',
                      color: Color(0xFF059669),
                      size: 28,
                    ),
                  ),
                ),
                const GfSettingRow(
                  title: '个人资料',
                  description: '昵称、简介和社交链接',
                  symbol: 'id-card',
                ),
                const GfSettingRow(
                  title: '账号与安全',
                  symbol: 'shield-check',
                  iconColor: Color(0xFF059669),
                ),
                const GfSettingRow(
                  title: 'Sprache der App',
                  description: 'Systemsprache verwenden',
                  symbol: 'languages',
                ),
                const GfSettingRow(
                  title: 'アプリの言語',
                  description: 'システム設定に従う',
                  symbol: 'languages',
                ),
              ],
            ),
          ),
          brightness: brightness,
        );
        await expectLater(
          find.byType(Scaffold),
          matchesGoldenFile('golden/gf_profile_cards_${brightness.name}.png'),
        );
      },
    );
  }

  for (final brightness in Brightness.values) {
    testWidgets(
      'Search and follow controls ${brightness.name}',
      skip: skipGoldens,
      tags: 'golden',
      (tester) async {
        final controller = TextEditingController(text: '高等数学');
        addTearDown(controller.dispose);
        await pumpGfGolden(
          tester,
          Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const GfSearchField(hintText: '搜索会话', clearLabel: '清空搜索'),
                const SizedBox(height: 16),
                GfSearchField(
                  controller: controller,
                  hintText: '搜索课程、教师',
                  clearLabel: '清空搜索',
                ),
                const SizedBox(height: 16),
                const GfSearchField(
                  hintText: 'Konversationen durchsuchen',
                  clearLabel: 'Leeren',
                ),
                const SizedBox(height: 24),
                Row(
                  spacing: 12,
                  children: [
                    GfButton(
                      label: '关注',
                      icon: const GfSymbol('user-round-plus', size: 18),
                      onPressed: () {},
                    ),
                    GfButton(
                      label: '已关注',
                      variant: GfButtonVariant.outline,
                      icon: const GfSymbol('user-round-check', size: 18),
                      onPressed: () {},
                    ),
                  ],
                ),
              ],
            ),
          ),
          brightness: brightness,
        );
        await expectLater(
          find.byType(Scaffold),
          matchesGoldenFile('golden/gf_search_follow_${brightness.name}.png'),
        );
      },
    );
  }

  testWidgets('GfButton all variants', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfBadge all variants', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfSegmented control', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfTopicRow states', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfFloatingControls', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfNotificationRow states', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfMessageBubble states', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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

  testWidgets('GfAvatarStack sm', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
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
