// @mention 会话纯逻辑测试(issue #565,镜像 Web mention.test.ts 语料)。
//
// 覆盖:token 识别边界/关闭、候选排序去重/排除自己/上限、选中替换的
// selection offset、防抖与 stale 响应丢弃、物理键盘处理。
import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

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
  avatarUrl: '/a$id.png',
  tag: tag,
);

void main() {
  group('extractMentionToken(token 识别)', () {
    test('行首 @ 触发,query 为 @ 后到光标前的文本', () {
      expect(
        extractMentionToken('@wa'),
        const MentionToken(start: 0, length: 3, query: 'wa'),
      );
    });

    test('空白分隔的 @ 触发', () {
      expect(
        extractMentionToken('你好 @wa'),
        const MentionToken(start: 3, length: 3, query: 'wa'),
      );
      expect(
        extractMentionToken('foo\n@wa'),
        const MentionToken(start: 4, length: 3, query: 'wa'),
      );
    });

    test('CJK 字符前 @ 触发(中文语境直接 @ 昵称)', () {
      expect(
        extractMentionToken('感谢@张三'),
        const MentionToken(start: 2, length: 3, query: '张三'),
      );
    });

    test('常见标点分隔的 @ 触发', () {
      expect(
        extractMentionToken('（@wa'),
        const MentionToken(start: 1, length: 3, query: 'wa'),
      );
    });

    test('仅输入 @ 时 query 为空(展示本地上下文)', () {
      expect(
        extractMentionToken('@'),
        const MentionToken(start: 0, length: 1, query: ''),
      );
      expect(
        extractMentionToken('你好 @'),
        const MentionToken(start: 3, length: 1, query: ''),
      );
    });

    test('邮箱/标识符内嵌 @ 不触发', () {
      expect(extractMentionToken('a@b'), isNull);
      expect(extractMentionToken('foo@bar'), isNull);
      expect(extractMentionToken('mail to a@b.com'), isNull);
    });

    test('@ 连写:第二个 @ 前的字符非边界,不触发', () {
      expect(extractMentionToken('@foo@bar'), isNull);
    });

    test('空格/终止标点关闭 token', () {
      expect(extractMentionToken('@wa '), isNull);
      expect(extractMentionToken('@wa，'), isNull);
      expect(extractMentionToken('@wa。'), isNull);
      expect(extractMentionToken('@wa,'), isNull);
    });

    test('无 @ 不触发', () {
      expect(extractMentionToken('plain text'), isNull);
      expect(extractMentionToken(''), isNull);
    });

    test('token 偏移按 UTF-16 code unit 计,与 selection offset 一致', () {
      // emoji 为代理对(2 code unit),start 仍指向 @ 的字节偏移。
      final token = extractMentionToken('😀 @wa');
      expect(token, isNotNull);
      expect(token!.start, 3);
      expect(token.length, 3);
      expect(
        '😀 @wa'.substring(token.start, token.start + token.length),
        '@wa',
      );
    });
  });

  group('rankMentionCandidates(候选排序/去重/排除自己)', () {
    final local = <MentionUser>[
      user(1, 'target', nickname: '回复目标', tag: MentionTag.replyTarget),
      user(2, 'author', nickname: '主题作者', tag: MentionTag.topicAuthor),
      user(3, 'participant', nickname: '参与者', tag: MentionTag.participant),
    ];

    test('空 query:仅本地上下文,最多 5 个,保持 回复目标>主题作者>参与者 顺序', () {
      final result = rankMentionCandidates(
        local: local,
        server: [user(9, 'anyone')],
        query: '',
        currentUserId: 0,
      );
      expect(result.map((u) => u.id).toList(), [1, 2, 3]);
    });

    test('空 query 且无本地上下文:返回空(提示继续输入,不查服务端)', () {
      expect(
        rankMentionCandidates(
          local: const [],
          server: [user(9, 'anyone')],
          query: '',
        ),
        isEmpty,
      );
    });

    test('空 query 本地超过 5 个截断', () {
      final many = List.generate(
        7,
        (i) => user(i + 10, 'p$i', tag: MentionTag.participant),
      );
      final result = rankMentionCandidates(local: many, server: [], query: '');
      expect(result, hasLength(5));
    });

    test('有 query:本地上下文匹配者优先,再按匹配强度排服务端结果', () {
      final server = [
        user(20, 'alpha'),
        user(21, 'wavery'),
        user(22, 'xiaowang'),
        user(23, 'wangwu'),
      ];
      final result = rankMentionCandidates(
        local: local,
        server: server,
        query: 'wa',
        currentUserId: 0,
      );
      // 本地上下文均不含 "wa" → 全部过滤;username 前缀匹配排在包含匹配前。
      expect(result.map((u) => u.username).toList(), [
        'wavery',
        'wangwu',
        'xiaowang',
      ]);
    });

    test('有 query:本地匹配候选排服务端之前', () {
      final local2 = [user(1, 'wade', tag: MentionTag.participant)];
      final result = rankMentionCandidates(
        local: local2,
        server: [user(21, 'wavery')],
        query: 'wa',
        currentUserId: 0,
      );
      expect(result.map((u) => u.username).toList(), ['wade', 'wavery']);
    });

    test('按 userId 去重,本地优先', () {
      final result = rankMentionCandidates(
        local: [user(1, 'target', tag: MentionTag.replyTarget)],
        server: [user(1, 'target')],
        query: 'ta',
        currentUserId: 0,
      );
      expect(result, hasLength(1));
      expect(result.first.tag, MentionTag.replyTarget);
    });

    test('当前用户不进入候选', () {
      final result = rankMentionCandidates(
        local: [user(1, 'target', tag: MentionTag.replyTarget)],
        server: [user(9, 'me'), user(21, 'wavery')],
        query: 'a',
        currentUserId: 9,
      );
      expect(result.map((u) => u.id), isNot(contains(9)));
    });

    test('有 query 上限 8', () {
      final server = List.generate(12, (i) => user(i + 100, 'u${i}wa'));
      final result = rankMentionCandidates(
        local: [],
        server: server,
        query: 'wa',
        currentUserId: 0,
      );
      expect(result, hasLength(8));
    });

    test('昵称匹配强度:exact > prefix > contains', () {
      final server = [
        user(1, 'zz', nickname: 'xwangz'),
        user(2, 'yy', nickname: 'wang'),
        user(3, 'xx', nickname: 'wangx'),
      ];
      final result = rankMentionCandidates(
        local: [],
        server: server,
        query: 'wang',
        currentUserId: 0,
      );
      expect(result.map((u) => u.id).toList(), [2, 3, 1]);
    });
  });

  group('applyMentionReplacement(选中写入与 selection offset)', () {
    test('原位替换 token 为 @username + 单空格,selection 落在插入后', () {
      const value = TextEditingValue(
        text: 'hello @wa',
        selection: TextSelection.collapsed(offset: 9),
      );
      final next = applyMentionReplacement(
        value,
        const MentionReplacement(start: 6, length: 3, replacement: '@wavery '),
      );
      expect(next.text, 'hello @wavery ');
      expect(next.selection.baseOffset, 14);
      expect(next.selection.isCollapsed, isTrue);
      expect(next.composing, TextRange.empty);
    });

    test('中段 token 替换保持其余文本不动', () {
      const value = TextEditingValue(
        text: 'foo @wa bar',
        selection: TextSelection.collapsed(offset: 7),
      );
      final next = applyMentionReplacement(
        value,
        const MentionReplacement(start: 4, length: 3, replacement: '@zed'),
      );
      expect(next.text, 'foo @zed bar');
      expect(next.selection.baseOffset, 8);
    });
  });

  group('MentionSessionController(防抖/stale 丢弃/失败降级)', () {
    testWidgets('空 query 只出本地候选,不触发搜索', (tester) async {
      var searchCalls = 0;
      final session = MentionSessionController(
        searchUsers: (q) async {
          searchCalls++;
          return <MentionUser>[];
        },
      );
      addTearDown(session.dispose);
      session.updateContext(local: const [], currentUserId: 0);
      session.handleValue('你好 @');
      expect(session.open, isTrue);
      expect(session.candidates, isEmpty);
      await tester.pumpAndSettle();
      expect(searchCalls, 0);
    });

    testWidgets('防抖后搜索,结果与本地合并', (tester) async {
      var searchCalls = 0;
      String? receivedQuery;
      final session = MentionSessionController(
        searchUsers: (q) async {
          searchCalls++;
          receivedQuery = q;
          return [user(21, 'wavery')];
        },
      );
      addTearDown(session.dispose);
      session.handleValue('hello @wa');
      expect(searchCalls, 0); // 防抖窗口内未发起
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      expect(searchCalls, 1);
      expect(receivedQuery, 'wa');
      expect(session.candidates.map((u) => u.username), ['wavery']);
      expect(session.loading, isFalse);
    });

    testWidgets('query 变化废弃在途响应:仅最新 query 结果生效', (tester) async {
      final completer1 = Completer<List<MentionUser>>();
      final completer2 = Completer<List<MentionUser>>();
      final queries = <String>[];
      final session = MentionSessionController(
        searchUsers: (q) async {
          queries.add(q);
          return queries.length == 1 ? completer1.future : completer2.future;
        },
      );
      addTearDown(session.dispose);

      session.handleValue('@wa'); // 第一轮
      await tester.pump(const Duration(milliseconds: 350));
      session.handleValue('@wav'); // 第二轮:旧响应作废
      await tester.pump(const Duration(milliseconds: 350));

      completer1.complete([user(30, 'wa-old')]);
      await tester.pumpAndSettle();
      // stale 响应不得进入候选。
      expect(
        session.candidates.map((u) => u.username),
        isNot(contains('wa-old')),
      );

      completer2.complete([user(31, 'wavery')]);
      await tester.pumpAndSettle();
      expect(session.candidates.map((u) => u.username), ['wavery']);
      expect(queries, ['wa', 'wav']);
    });

    testWidgets('搜索失败不阻塞编辑:保留本地匹配并标记 failed', (tester) async {
      final session = MentionSessionController(
        searchUsers: (q) async {
          throw Exception('network down');
        },
      );
      addTearDown(session.dispose);
      final local = [user(1, 'wade', tag: MentionTag.participant)];
      session.updateContext(local: local, currentUserId: 0);
      session.handleValue('@wa');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      expect(session.failed, isTrue);
      expect(session.candidates.map((u) => u.username), ['wade']);
      expect(session.open, isTrue);
    });

    testWidgets('token 失效(caret 离开)自动关闭并取消防抖', (tester) async {
      var searchCalls = 0;
      final session = MentionSessionController(
        searchUsers: (q) async {
          searchCalls++;
          return <MentionUser>[];
        },
      );
      addTearDown(session.dispose);
      session.handleValue('@wa');
      expect(session.open, isTrue);
      session.handleValue('@wa '); // 空格终止 token
      expect(session.open, isFalse);
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      expect(searchCalls, 0);
    });

    test('close 幂等且不抛出', () {
      final session = MentionSessionController(searchUsers: (q) async => []);
      addTearDown(session.dispose);
      session.close();
      session.close();
      expect(session.open, isFalse);
    });
    testWidgets('会话开启时上下文变化立即重排(清除回复目标不再残留居首)', (tester) async {
      final session = MentionSessionController(searchUsers: (q) async => []);
      addTearDown(session.dispose);
      session.updateContext(
        local: [
          user(1, 'target', tag: MentionTag.replyTarget),
          user(2, 'author', tag: MentionTag.topicAuthor),
        ],
        currentUserId: 0,
      );
      session.handleValue('hello @');
      expect(session.open, isTrue);
      expect(session.candidates.map((u) => u.id).toList(), [1, 2]);

      // 清除回复目标(如用户点掉回复条)后,候选立即去掉该用户。
      session.updateContext(
        local: [user(2, 'author', tag: MentionTag.topicAuthor)],
        currentUserId: 0,
      );
      expect(session.candidates.map((u) => u.id).toList(), [2]);
      expect(session.activeIndex, 0);
    });

    testWidgets('dispose 后在途搜索落地不再通知(不触发 used-after-dispose)', (tester) async {
      final completer = Completer<List<MentionUser>>();
      final session = MentionSessionController(
        searchUsers: (q) async => completer.future,
      );
      session.handleValue('@wa');
      await tester.pump(const Duration(milliseconds: 350));
      session.dispose();
      completer.complete([user(21, 'wavery')]);
      await tester.pumpAndSettle();
      // dispose 后落地的响应不得触发 used-after-dispose 断言(open 状态本身不变)。
      expect(tester.takeException(), isNull);
    });
  });

  group('handleMentionKeyEvent(物理键盘)', () {
    testWidgets('Escape 只关候选,不修改文本', (tester) async {
      final session = MentionSessionController(searchUsers: (q) async => []);
      addTearDown(session.dispose);
      session.handleValue('@wa');
      final controller = TextEditingController(text: 'hello @wa');
      controller.selection = const TextSelection.collapsed(offset: 9);

      final result = handleMentionKeyEvent(
        session: session,
        controller: controller,
        event: KeyDownEvent(
          timeStamp: Duration.zero,
          physicalKey: PhysicalKeyboardKey.escape,
          logicalKey: LogicalKeyboardKey.escape,
          character: null,
        ),
        onSelect: (_, _) => fail('Escape 不应选中'),
      );
      expect(result, KeyEventResult.handled);
      expect(session.open, isFalse);
      expect(controller.text, 'hello @wa');
    });

    testWidgets('方向键移动 active,Enter 选中并回调', (tester) async {
      final session = MentionSessionController(
        searchUsers: (q) async => [
          user(21, 'wavery'),
          user(22, 'xiaowang'),
          user(23, 'wangwu'),
        ],
      );
      addTearDown(session.dispose);
      session.handleValue('@wa');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      expect(session.activeIndex, 0);

      final controller = TextEditingController(text: 'hello @wa');
      controller.selection = const TextSelection.collapsed(offset: 9);

      (MentionToken, MentionUser)? selected;
      KeyEventResult send(LogicalKeyboardKey key) => handleMentionKeyEvent(
        session: session,
        controller: controller,
        event: KeyDownEvent(
          timeStamp: Duration.zero,
          physicalKey: PhysicalKeyboardKey.arrowDown,
          logicalKey: key,
          character: null,
        ),
        onSelect: (token, user) {
          selected = (token, user);
        },
      );

      expect(send(LogicalKeyboardKey.arrowDown), KeyEventResult.handled);
      expect(session.activeIndex, 1);
      expect(send(LogicalKeyboardKey.arrowDown), KeyEventResult.handled);
      expect(session.activeIndex, 2);
      expect(send(LogicalKeyboardKey.arrowDown), KeyEventResult.handled);
      expect(session.activeIndex, 0); // 环绕
      expect(send(LogicalKeyboardKey.arrowUp), KeyEventResult.handled);
      expect(session.activeIndex, 2);

      expect(
        handleMentionKeyEvent(
          session: session,
          controller: controller,
          event: KeyDownEvent(
            timeStamp: Duration.zero,
            physicalKey: PhysicalKeyboardKey.enter,
            logicalKey: LogicalKeyboardKey.enter,
            character: null,
          ),
          onSelect: (token, user) {
            selected = (token, user);
          },
        ),
        KeyEventResult.handled,
      );
      expect(session.open, isFalse);
      expect(selected, isNotNull);
      expect(selected!.$1.query, 'wa');
      expect(selected!.$2.username, 'xiaowang');
    });

    test('会话未开启时不拦截任何按键', () {
      final session = MentionSessionController(searchUsers: (q) async => []);
      addTearDown(session.dispose);
      final controller = TextEditingController();
      final result = handleMentionKeyEvent(
        session: session,
        controller: controller,
        event: KeyDownEvent(
          timeStamp: Duration.zero,
          physicalKey: PhysicalKeyboardKey.escape,
          logicalKey: LogicalKeyboardKey.escape,
          character: null,
        ),
        onSelect: (_, _) {},
      );
      expect(result, KeyEventResult.ignored);
    });
    testWidgets('KeyUpEvent 被忽略:方向键一次只移动一行', (tester) async {
      final session = MentionSessionController(
        searchUsers: (q) async => [user(21, 'wavery'), user(22, 'xiaowang')],
      );
      addTearDown(session.dispose);
      session.handleValue('@wa');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      final controller = TextEditingController(text: 'hello @wa');
      controller.selection = const TextSelection.collapsed(offset: 9);

      KeyEventResult sendUp(LogicalKeyboardKey key) => handleMentionKeyEvent(
        session: session,
        controller: controller,
        event: KeyUpEvent(
          timeStamp: Duration.zero,
          physicalKey: PhysicalKeyboardKey.arrowDown,
          logicalKey: key,
        ),
        onSelect: (_, _) {},
      );

      expect(sendUp(LogicalKeyboardKey.arrowDown), KeyEventResult.ignored);
      expect(session.activeIndex, 0);
    });

    testWidgets('KeyRepeatEvent 长按连续移动 active', (tester) async {
      final session = MentionSessionController(
        searchUsers: (q) async => [
          user(21, 'wavery'),
          user(22, 'xiaowang'),
          user(23, 'wangwu'),
        ],
      );
      addTearDown(session.dispose);
      session.handleValue('@wa');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      final controller = TextEditingController(text: 'hello @wa');
      controller.selection = const TextSelection.collapsed(offset: 9);

      KeyEventResult sendRepeat(LogicalKeyboardKey key) =>
          handleMentionKeyEvent(
            session: session,
            controller: controller,
            event: KeyRepeatEvent(
              timeStamp: Duration.zero,
              physicalKey: PhysicalKeyboardKey.arrowDown,
              logicalKey: key,
              character: null,
            ),
            onSelect: (_, _) {},
          );

      expect(sendRepeat(LogicalKeyboardKey.arrowDown), KeyEventResult.handled);
      expect(session.activeIndex, 1);
      expect(sendRepeat(LogicalKeyboardKey.arrowDown), KeyEventResult.handled);
      expect(session.activeIndex, 2);
    });
  });
}
