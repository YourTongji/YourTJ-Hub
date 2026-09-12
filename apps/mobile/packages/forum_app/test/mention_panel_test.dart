// @mention 候选面板 widget 测试(issue #565)。
//
// 覆盖:会话开合跟随、候选行信息(avatar/昵称/@username/上下文标签)、
// Semantics 身份标签、点选回调、320 logical px + textScale 2.0 无关键溢出。
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:forum_app/src/pages/topic/mention_panel.dart';
import 'package:forum_app/src/pages/topic/mention_session.dart';

MentionUser user(
  int id,
  String username, {
  String? nickname,
  MentionTag? tag,
}) => MentionUser(
  id: id,
  username: username,
  nickname: nickname ?? username,
  avatarUrl: '',
  tag: tag,
);

const MentionPanelMessages messages = MentionPanelMessages(
  listboxLabel: 'Mention users',
  loading: 'Searching users…',
  noResults: 'No matching users',
  searchFailed: 'User search failed, keep typing',
  keepTyping: 'Keep typing to search users',
  tagReplyTarget: 'Replying to',
  tagTopicAuthor: 'Topic author',
  tagParticipant: 'Participant',
);

Widget host(
  MentionSessionController session, {
  void Function(MentionToken token, MentionUser user)? onSelect,
  TextScaler textScaler = TextScaler.noScaling,
}) {
  return MaterialApp(
    theme: gfThemeData(Brightness.light),
    home: MediaQuery(
      data: MediaQueryData(textScaler: textScaler),
      child: Scaffold(
        body: Align(
          alignment: Alignment.bottomCenter,
          child: MentionCandidatesPanel(
            session: session,
            messages: messages,
            onSelect: onSelect ?? (_, _) {},
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('会话关闭时面板不渲染', (tester) async {
    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    await tester.pumpWidget(host(session));
    expect(find.text('Mention users'), findsNothing);
  });

  testWidgets('候选行展示昵称/@username/上下文标签,点选回调携带 token', (tester) async {
    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    session.updateContext(
      local: [
        user(2, 'author', nickname: '主题作者', tag: MentionTag.topicAuthor),
        user(3, 'bob', nickname: '小明', tag: MentionTag.participant),
      ],
      currentUserId: 0,
    );
    session.handleValue('hello @');
    (MentionToken, MentionUser)? selected;
    await tester.pumpWidget(
      host(session, onSelect: (token, user) => selected = (token, user)),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Mention users'),
      findsNothing,
    ); // listbox label 只进 Semantics
    expect(find.text('主题作者'), findsOneWidget);
    expect(find.text('@author'), findsOneWidget);
    expect(find.text('Topic author'), findsOneWidget);
    expect(find.text('小明'), findsOneWidget);
    expect(find.text('@bob'), findsOneWidget);
    expect(find.text('Participant'), findsOneWidget);

    await tester.tap(find.text('小明'));
    expect(selected, isNotNull);
    expect(selected!.$1.query, '');
    expect(selected!.$2.username, 'bob');
  });

  testWidgets('空 query 无本地候选展示继续输入提示', (tester) async {
    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    session.handleValue('hello @');
    await tester.pumpWidget(host(session));
    await tester.pumpAndSettle();
    expect(find.text('Keep typing to search users'), findsOneWidget);
  });

  testWidgets('候选行 Semantics 含昵称、@username 与上下文身份', (tester) async {
    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    session.updateContext(
      local: [user(2, 'author', nickname: '主题作者', tag: MentionTag.topicAuthor)],
      currentUserId: 0,
    );
    session.handleValue('hello @');
    await tester.pumpWidget(host(session));
    await tester.pumpAndSettle();

    final semantics = tester.getSemantics(find.text('主题作者'));
    final label = semantics.label;
    expect(label, contains('主题作者'));
    expect(label, contains('@author'));
    // 读屏激活:tap 动作必须暴露在 Semantics 节点上(ExcludeSemantics 剥掉
    // InkWell 语义后,外层节点补挂 onTap)。
    expect(semantics.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
  });

  testWidgets('无界高度容器(Column 非弹性子项)不触发 unbounded viewport', (tester) async {
    // 复刻 topic_page 的 Positioned + Column 挂载:垂直方向无界,
    // 面板必须自带 maxHeight 封顶。
    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    session.updateContext(
      local: [
        for (var i = 0; i < 8; i++)
          user(i + 10, 'user$i', tag: MentionTag.participant),
      ],
      currentUserId: 0,
    );
    session.handleValue('hello @');
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: Stack(
            children: [
              Positioned(
                left: 12,
                right: 12,
                bottom: 12,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    MentionCandidatesPanel(
                      session: session,
                      messages: messages,
                      onSelect: (_, _) {},
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.text('user0'), findsOneWidget);
  });

  testWidgets('320 logical px + textScale 2.0 无关键布局溢出', (tester) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final session = MentionSessionController(searchUsers: (q) async => []);
    addTearDown(session.dispose);
    session.updateContext(
      local: [
        user(2, 'author', nickname: '很长很长的中文昵称测试', tag: MentionTag.topicAuthor),
        user(
          3,
          'a-very-long-username-handle',
          nickname: 'X',
          tag: MentionTag.participant,
        ),
      ],
      currentUserId: 0,
    );
    session.handleValue('hello @');
    await tester.pumpWidget(host(session, textScaler: TextScaler.linear(2.0)));
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('很长很长的中文昵称测试'), findsOneWidget);
  });
}
