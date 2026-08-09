import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfFloatingControls', () {
    testWidgets('renders floor button, actions and join button', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfFloatingControls(
            actions: <GfTopicAction>[
              GfTopicAction(
                icon: Icons.favorite_border,
                active: true,
                activeColor: GfColors.light.error,
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
      expect(find.text('3 / 120'), findsOneWidget);
      expect(find.byIcon(Icons.favorite_border), findsOneWidget);
      expect(find.text('参与讨论'), findsOneWidget);
    });

    testWidgets('hides floor button when maxNo is null', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfFloatingControls(
            actions: const <GfTopicAction>[],
            onOpenReply: () {},
          ),
        ),
      );
      expect(find.textContaining('/'), findsNothing);
      expect(find.text('参与讨论'), findsOneWidget);
    });
  });

  group('GfPostPositionRail', () {
    testWidgets('reports floor selection on tap', (tester) async {
      int? selected;
      await tester.pumpWidget(
        gfApp(
          SizedBox(
            height: 300,
            child: GfPostPositionRail(
              current: 1,
              max: 10,
              onSelect: (floor) => selected = floor,
              onEarliest: () {},
              onLatest: () {},
            ),
          ),
        ),
      );
      expect(find.text('1 / 10'), findsOneWidget);
      expect(find.text('最早'), findsOneWidget);
      expect(find.text('最新'), findsOneWidget);
      // 点击轨道下半部 → 选择较后的楼层。
      final Rect rect = tester.getRect(find.byType(GfPostPositionRail));
      await tester.tapAt(Offset(rect.center.dx, rect.bottom - 40));
      await tester.pump();
      expect(selected, isNotNull);
      expect(selected, inInclusiveRange(5, 10));
    });
  });

  group('GfNotificationRow', () {
    testWidgets('renders unread styling with dot', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfNotificationRow(
            icon: Icons.message,
            tone: GfNotificationTone.primary,
            title: '有人回复了你',
            subtitle: '内容预览',
            time: '3 分钟前',
            unread: true,
          ),
        ),
      );
      expect(find.text('有人回复了你'), findsOneWidget);
      expect(find.text('内容预览'), findsOneWidget);
      expect(find.text('3 分钟前'), findsOneWidget);
    });

    testWidgets('renders read row without dot', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfNotificationRow(
            icon: Icons.message,
            tone: GfNotificationTone.success,
            title: '已读通知',
            subtitle: '',
            time: '昨天',
            unread: false,
          ),
        ),
      );
      expect(find.text('已读通知'), findsOneWidget);
    });
  });

  group('GfConversationRow / GfMessageBubble', () {
    testWidgets('renders conversation row with unread count', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfConversationRow(
            avatarUrl: '',
            name: 'Alice',
            lastMessage: '你好',
            time: '10:30',
            unreadCount: 3,
          ),
        ),
      );
      expect(find.text('Alice'), findsOneWidget);
      expect(find.text('你好'), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (Widget widget) =>
              widget is Semantics && widget.properties.label == '3 unread',
        ),
        findsOneWidget,
      );
    });

    testWidgets('renders own and other message bubbles', (tester) async {
      await tester.pumpWidget(
        gfApp(
          Column(
            children: const <Widget>[
              GfMessageBubble(text: '我的消息', mine: true),
              GfMessageBubble(text: '对方消息', mine: false),
            ],
          ),
        ),
      );
      expect(find.text('我的消息'), findsOneWidget);
      expect(find.text('对方消息'), findsOneWidget);
    });
  });

  group('GfDraftRow / GfUserCard / GfSettingRow', () {
    testWidgets('renders draft row', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfDraftRow(
            title: '草稿标题',
            description: '草稿描述',
            categories: const <GfTopicCategory>[],
            blocked: false,
            meta: '2026-08-01 · 3 浏览',
            updatedTime: '昨天',
          ),
        ),
      );
      expect(find.text('草稿标题'), findsOneWidget);
      expect(find.text('草稿描述'), findsOneWidget);
    });

    testWidgets('renders four-column user stats at mobile width', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 844);
      addTearDown(tester.view.reset);

      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: const MediaQueryData(textScaler: TextScaler.linear(1.3)),
            child: GfUserCard(
              avatarUrl: '',
              name: 'Tongji',
              username: 'tongji',
              bio: '你好',
              stats: const <(String, String)>[
                ('话题', '12'),
                ('回复', '34'),
                ('关注', '56'),
                ('粉丝', '78'),
              ],
            ),
          ),
        ),
      );

      expect(find.text('Tongji'), findsOneWidget);
      expect(find.text('@tongji'), findsOneWidget);
      expect(find.text('12'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('renders setting rows', (tester) async {
      await tester.pumpWidget(
        gfApp(
          Column(
            children: <Widget>[
              const GfSettingRow(
                title: '昵称',
                description: '修改昵称',
                icon: Icons.badge_outlined,
              ),
              GfSwitchRow(title: '开启通知', value: true, onChanged: (_) {}),
            ],
          ),
        ),
      );
      expect(find.text('昵称'), findsOneWidget);
      expect(find.text('开启通知'), findsOneWidget);
    });
  });
}
