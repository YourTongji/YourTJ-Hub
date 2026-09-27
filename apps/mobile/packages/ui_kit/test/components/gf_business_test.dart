import 'dart:ui' show SemanticsAction;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

void main() {
  group('GfFloatingControls', () {
    testWidgets('review regression: action titles label tappable semantics', (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      var taps = 0;
      await tester.pumpWidget(
        gfApp(
          GfFloatingControls(
            onOpenReply: null,
            actions: [
              for (final label in ['点赞 · 12', '收藏', '关注帖子'])
                GfTopicAction(
                  symbol: 'heart',
                  title: label,
                  active: label == '收藏',
                  activeColor: GfColors.light.primary,
                  onTap: () => taps++,
                ),
            ],
          ),
        ),
      );
      for (final label in ['点赞 · 12', '收藏', '关注帖子']) {
        expect(find.byTooltip(label), findsOneWidget);
        expect(
          tester.getSemantics(find.bySemanticsLabel(label)),
          matchesSemantics(
            label: label,
            isButton: true,
            hasTapAction: true,
            hasEnabledState: true,
            isEnabled: true,
            hasToggledState: true,
            isToggled: label == '收藏',
          ),
        );
        await tester.tap(find.byTooltip(label));
      }
      expect(taps, 3);
      semantics.dispose();
    });

    testWidgets('localized dock fits narrow screens and large text', (
      tester,
    ) async {
      for (final width in [280.0, 360.0]) {
        for (final scale in [1.0, 2.0]) {
          var replies = 0;
          await tester.pumpWidget(
            gfApp(
              MediaQuery(
                data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                child: SizedBox(
                  width: width,
                  child: GfFloatingControls(
                    currentNo: 1234,
                    maxNo: 9999,
                    onFloorTap: () {},
                    joinLabel: 'An Diskussion teilnehmen',
                    onOpenReply: () => replies++,
                    actions: List.generate(
                      3,
                      (_) => GfTopicAction(
                        symbol: 'heart',
                        active: false,
                        activeColor: GfColors.light.error,
                        onTap: () {},
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
          final reply = find.byWidgetPredicate(
            (w) => w is GfSymbol && w.name == 'corner-down-left',
          );
          await tester.tap(reply);
          expect(replies, 1);
        }
      }
    });

    testWidgets('renders floor button, actions and join button', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfFloatingControls(
            actions: <GfTopicAction>[
              GfTopicAction(
                symbol: 'heart-filled',
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
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'heart-filled',
        ),
        findsOneWidget,
      );
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
    testWidgets('one-floor topics have a finite, disabled position slider', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          GfPostPositionRail(
            current: 1,
            max: 1,
            onSelect: (_) {},
            onEarliest: () {},
            onLatest: () {},
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(tester.widget<Slider>(find.byType(Slider)).onChanged, isNull);
    });

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
      // The visible horizontal slider commits the selected floor.
      final Rect rect = tester.getRect(find.byType(Slider));
      await tester.tapAt(Offset(rect.left + rect.width * 0.8, rect.center.dy));
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
            symbol: 'message-circle',
            tone: GfNotificationTone.primary,
            title: '有人回复了你',
            subtitle: '内容预览',
            time: '3 分钟前',
            unread: true,
          ),
        ),
      );
      expect(find.textContaining('有人回复了你'), findsOneWidget);
      expect(find.text('内容预览'), findsOneWidget);
      expect(find.textContaining('3 分钟前'), findsOneWidget);
    });

    testWidgets('renders read row without dot', (tester) async {
      await tester.pumpWidget(
        gfApp(
          GfNotificationRow(
            symbol: 'message-circle',
            tone: GfNotificationTone.success,
            title: '已读通知',
            subtitle: '',
            time: '昨天',
            unread: false,
          ),
        ),
      );
      expect(find.textContaining('已读通知'), findsOneWidget);
    });
  });

  group('GfConversationRow / GfMessageBubble', () {
    testWidgets('long conversation dates leave room for the name at 2x text', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          MediaQuery(
            data: const MediaQueryData(textScaler: TextScaler.linear(2)),
            child: const SizedBox(
              width: 320,
              child: GfConversationRow(
                avatarUrl: '',
                name: 'Campus friend',
                lastMessage: 'See you on campus',
                time: '15. September 2026, 10:30',
                unreadCount: 1,
              ),
            ),
          ),
        ),
      );
      expect(tester.takeException(), isNull);
      expect(tester.getRect(find.text('Campus friend')).width, greaterThan(80));
      expect(
        tester.getRect(find.text('15. September 2026, 10:30')).top,
        greaterThan(tester.getRect(find.text('Campus friend')).bottom),
      );
    });

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

    testWidgets('contentSpan replaces plain text body, text path unchanged', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          Column(
            children: <Widget>[
              GfMessageBubble(
                text: '[:sticker:smile:]',
                mine: false,
                contentSpan: TextSpan(
                  children: <InlineSpan>[
                    const TextSpan(text: '前 '),
                    WidgetSpan(
                      alignment: PlaceholderAlignment.middle,
                      child: SizedBox(
                        width: 56,
                        height: 56,
                        child: ColoredBox(color: Colors.amber),
                      ),
                    ),
                  ],
                ),
              ),
              GfMessageBubble(text: '纯文本', mine: true),
            ],
          ),
        ),
      );
      // 富内容路径:token 原文不出现,内联占位渲染。
      expect(find.text('[:sticker:smile:]'), findsNothing);
      expect(find.textContaining('前'), findsOneWidget);
      expect(find.byType(ColoredBox), findsOneWidget);
      // 纯文本路径不受影响。
      expect(find.text('纯文本'), findsOneWidget);
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

    testWidgets('renders five-column user stats at mobile width', (
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
                ('获赞', '56'),
                ('粉丝', '78'),
                ('关注', '90'),
              ],
            ),
          ),
        ),
      );

      expect(find.text('Tongji'), findsOneWidget);
      expect(find.text('@tongji'), findsOneWidget);
      expect(find.text('12'), findsOneWidget);
      expect(find.text('90'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets(
      'profile statistic actions have keyboard activation and combined semantics',
      (tester) async {
        final semantics = tester.ensureSemantics();
        var calls = 0;
        await tester.pumpWidget(
          gfApp(
            GfUserCard(
              avatarUrl: '',
              name: 'Alice',
              username: 'alice',
              stats: const [('话题', '5'), ('粉丝', '20')],
              statActions: {1: () => calls++},
            ),
          ),
        );
        final action = find
            .ancestor(of: find.text('粉丝'), matching: find.byType(InkWell))
            .first;
        expect(tester.getSize(action).height, greaterThanOrEqualTo(48));
        expect(tester.getSize(action).width, greaterThanOrEqualTo(48));
        final node = tester.getSemantics(find.bySemanticsLabel('20\n粉丝'));
        expect(node.getSemanticsData().flagsCollection.isButton, isTrue);
        expect(node.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
        Focus.of(tester.element(find.text('粉丝'))).requestFocus();
        await tester.pump();
        await tester.sendKeyEvent(LogicalKeyboardKey.enter);
        await tester.pump();
        expect(calls, 1);
        await tester.tap(find.text('粉丝'));
        expect(calls, 2);
        await tester.tap(find.text('话题'));
        expect(calls, 2);
        semantics.dispose();
      },
    );

    testWidgets('profile trims bio and signature boundary whitespace', (
      tester,
    ) async {
      await tester.pumpWidget(
        gfApp(
          const GfUserCard(
            avatarUrl: '',
            name: 'Alice',
            username: 'alice',
            bio: '  Hello\n\n',
            signature: '  Stay curious\n',
          ),
        ),
      );
      expect(find.text('Hello'), findsOneWidget);
      expect(find.text('Stay curious'), findsOneWidget);
    });

    testWidgets('renders setting rows', (tester) async {
      await tester.pumpWidget(
        gfApp(
          Column(
            children: <Widget>[
              const GfSettingRow(
                title: '昵称',
                description: '修改昵称',
                symbol: 'award',
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

  group('GfPostComposer', () {
    testWidgets(
      'compact composer fits enlarged text and preserves collapse action',
      (tester) async {
        final controller = TextEditingController(text: 'A reply draft');
        addTearDown(controller.dispose);
        var collapsed = false;
        await tester.pumpWidget(
          gfApp(
            MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: SizedBox(
                width: 320,
                child: GfPostComposer(
                  controller: controller,
                  publishLabel: 'Senden',
                  hintText: 'Reply',
                  targetName: 'A long author name',
                  onCloseTarget: () {},
                  onPublish: () {},
                  onPickImage: () {},
                  imageTooltip: 'Image',
                  onPickSticker: () {},
                  stickerTooltip: 'Stickers',
                  onCollapse: () => collapsed = true,
                  collapseLabel: 'Collapse',
                  toolbar: const Text('Verification required'),
                ),
              ),
            ),
          ),
        );
        expect(tester.takeException(), isNull);
        expect(find.text('Verification required'), findsOneWidget);
        await tester.tap(find.byTooltip('Collapse'));
        expect(collapsed, isTrue);
        expect(controller.text, 'A reply draft');
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );

    testWidgets('keyboard dismissal retains reply text', (tester) async {
      final controller = TextEditingController(text: 'Unsent reply');
      final focus = FocusNode();
      await tester.pumpWidget(
        gfApp(
          GfPostComposer(
            controller: controller,
            focusNode: focus,
            onPublish: () {},
            publishLabel: 'Send',
            hintText: 'Reply',
          ),
        ),
      );
      await tester.tap(find.byType(TextField));
      await tester.pump();
      expect(focus.hasFocus, isTrue);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'keyboard-hide',
        ),
        findsOneWidget,
      );
      await tester.tap(
        find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'keyboard-hide',
        ),
      );
      await tester.pump();
      expect(focus.hasFocus, isFalse);
      expect(controller.text, 'Unsent reply');
      await tester.pumpWidget(const SizedBox.shrink());
      controller.dispose();
      focus.dispose();
    });

    testWidgets(
      'keeps 44 pixel image and keyboard actions below the filled reply input',
      (tester) async {
        int imageTaps = 0;
        final TextEditingController controller = TextEditingController();
        addTearDown(controller.dispose);

        await tester.pumpWidget(
          gfApp(
            SizedBox(
              width: 360,
              child: GfPostComposer(
                controller: controller,
                publishLabel: '发送',
                hintText: '参与讨论',
                onPublish: () {},
                onPickImage: () => imageTaps++,
                imageTooltip: '添加图片',
                uploading: true,
              ),
            ),
          ),
        );

        expect(find.byType(CircularProgressIndicator), findsOneWidget);
        final Finder imageButton = find.byTooltip('添加图片');
        expect(imageButton, findsOneWidget);
        await tester.tap(imageButton);
        expect(imageTaps, 0);
        final Offset imageTopLeft = tester.getTopLeft(imageButton);
        final Offset inputTopLeft = tester.getTopLeft(find.byType(TextField));
        expect(imageTopLeft.dy, greaterThan(inputTopLeft.dy));
        expect(
          tester.getCenter(imageButton).dy,
          tester
              .getCenter(
                find.byWidgetPredicate(
                  (widget) =>
                      widget is GfSymbol && widget.name == 'keyboard-hide',
                ),
              )
              .dy,
        );
        final field = tester.widget<TextField>(find.byType(TextField));
        expect(field.decoration!.focusedBorder, InputBorder.none);
        expect(field.decoration!.filled, isFalse);
        expect(field.minLines, 1);
        expect(field.maxLines, 4);
        final surface = tester.widget<AnimatedContainer>(
          find.byKey(const Key('reply-input-surface')),
        );
        expect(
          (surface.decoration! as BoxDecoration).borderRadius,
          BorderRadius.circular(24),
        );
        expect(
          tester
              .widget<Material>(find.byKey(const Key('reply-composer-surface')))
              .elevation,
          0,
        );
        expect(tester.getSize(imageButton), const Size(44, 44));
        expect(
          tester.getSize(find.byType(GfPostComposer)).height,
          lessThan(190),
        );
      },
    );
  });
}
