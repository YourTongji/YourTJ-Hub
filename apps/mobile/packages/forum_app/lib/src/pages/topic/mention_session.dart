/// 帖子回复编辑器 @mention 会话纯逻辑（issue #565）。
///
/// 与 Web 端 `apps/gooseforum/resource/src/runtime/mention.ts` 语义对齐：token
/// 触发边界、匹配强度排序、userId 去重、排除自己、空 query 上限 5 / 有 query
/// 上限 8。会话编排镜像 `useMentionAutocomplete.ts`：300ms 防抖、query 变化即
/// 废弃在途响应、stale 响应丢弃、搜索失败不阻塞编辑。TextField 的 caret 采集
/// 与写入由 topic_page 桥接；搜索依赖由调用方注入（core 依赖不进入本文件）。
library;

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

/// 本地上下文弱标签：正在回复 / 主题作者 / 参与者（仅本地候选携带）。
enum MentionTag { replyTarget, topicAuthor, participant }

/// @mention 候选用户：本地上下文与服务端搜索（scope=users）共用形状。
@immutable
class MentionUser {
  const MentionUser({
    required this.id,
    required this.username,
    required this.avatarUrl,
    this.nickname,
    this.tag,
  });

  final int id;
  final String username;
  final String? nickname;
  final String avatarUrl;
  final MentionTag? tag;

  String get displayName =>
      (nickname == null || nickname!.isEmpty) ? username : nickname!;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MentionUser &&
          other.id == id &&
          other.username == username &&
          other.nickname == nickname &&
          other.avatarUrl == avatarUrl &&
          other.tag == tag;

  @override
  int get hashCode => Object.hash(id, username, nickname, avatarUrl, tag);
}

/// 识别出的 mention token：光标前文本中 [start, start + length) 即 "@query"。
/// start/length 基于 UTF-16 code unit，与 Flutter selection offset 语义一致。
@immutable
class MentionToken {
  const MentionToken({
    required this.start,
    required this.length,
    required this.query,
  });

  final int start;
  final int length;
  final String query;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MentionToken &&
          other.start == start &&
          other.length == length &&
          other.query == query;

  @override
  int get hashCode => Object.hash(start, length, query);
}

/// 选中候选后的替换指令：将文本 [start, start + length) 替换为 [replacement]，
/// 光标落在替换内容之后（宿主经 [applyMentionReplacement] 写入编辑器）。
@immutable
class MentionReplacement {
  const MentionReplacement({
    required this.start,
    required this.length,
    required this.replacement,
  });

  final int start;
  final int length;
  final String replacement;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MentionReplacement &&
          other.start == start &&
          other.length == length &&
          other.replacement == replacement;

  @override
  int get hashCode => Object.hash(start, length, replacement);
}

/// @ 触发边界：@ 前一个字符不能是单词字符/数字/下划线（镜像 Web
/// MENTION_TRIGGER_BOUNDARY = /[^A-Za-z0-9_@/\\-]/）。
/// "a@b" / "foo@bar"（邮箱、标识符）不触发；"谢谢@张三"（CJK 前）触发。
bool _isMentionBoundaryPrev(int unit) {
  const underscore = 0x5F;
  const at = 0x40;
  const slash = 0x2F;
  const backslash = 0x5C;
  const dash = 0x2D;
  if (unit == underscore ||
      unit == at ||
      unit == slash ||
      unit == backslash ||
      unit == dash) {
    return false;
  }
  if (unit >= 0x30 && unit <= 0x39) return false; // 0-9
  if (unit >= 0x41 && unit <= 0x5A) return false; // A-Z
  if (unit >= 0x61 && unit <= 0x7A) return false; // a-z
  return true;
}

/// token 内终止字符：空白与常见（中英文）终止标点（镜像 Web TOKEN_TERMINATOR）。
bool _isTokenTerminator(int unit) {
  // \s（JS 语义）与 ASCII 控制区。
  if (unit <= 0x20) return true;
  const terminators = <int>{
    0xFF0C, 0x3002, 0xFF01, 0xFF1F, 0xFF1B, 0xFF1A, 0x3001, // ，。！？；：、
    0xFF08, 0xFF09, 0x3010, 0x3011, // （）【】
    0x300C, 0x300D, 0x300E, 0x300F, // 「」『』
    0x201C, 0x201D, 0x2018, 0x2019, 0x2026, // “”‘’…
    0x2C, 0x2E, 0x21, 0x3F, 0x3B, 0x3A, // , . ! ? ; :
    0x28, 0x29, 0x3C, 0x3E, // ( ) < >
    0x22, 0x27, 0x2F, 0x5C, 0x7C, // " ' / \ |
    0x7B, 0x7D, 0x5B, 0x5D, // { } [ ]
    0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF, // Unicode 空白
  };
  if (terminators.contains(unit)) return true;
  return unit >= 0x2000 && unit <= 0x200A; // 各级 en/em 空格
}

/// 从光标前文本中提取最近的 @mention token。
///
/// 规则（与 Web/后端统一）：@ 位于开头或由空白/常见标点分隔；空格、换行、
/// 终止标点或 caret 离开 token 后关闭；邮箱/标识符内嵌的 @ 不触发。
MentionToken? extractMentionToken(String prefix) {
  if (prefix.isEmpty) return null;
  final queryUnits = <int>[];
  for (var i = prefix.length - 1; i >= 0; i--) {
    final unit = prefix.codeUnitAt(i);
    if (unit == 0x40 /* @ */ ) {
      final boundaryOk =
          i == 0 || _isMentionBoundaryPrev(prefix.codeUnitAt(i - 1));
      if (boundaryOk) {
        final query = String.fromCharCodes(queryUnits.reversed);
        return MentionToken(start: i, length: query.length + 1, query: query);
      }
      // 邮箱/标识符内的 @（如 "a@b"、"@foo@bar" 的第二个 @）：不视为 mention。
      return null;
    }
    if (_isTokenTerminator(unit)) return null;
    queryUnits.add(unit);
  }
  return null;
}

/// 用户匹配强度分值：username exact > username prefix > nickname exact >
/// nickname prefix > username 包含 > nickname 包含（镜像 Web userMatchScore）。
int mentionUserMatchScore(MentionUser user, String query) {
  final q = query.toLowerCase();
  final username = user.username.toLowerCase();
  final nickname =
      (user.nickname == null || user.nickname!.isEmpty
              ? user.username
              : user.nickname!)
          .toLowerCase();
  if (username == q) return 100;
  if (username.startsWith(q)) return 80;
  if (nickname == q) return 60;
  if (nickname.startsWith(q)) return 40;
  if (username.contains(q)) return 20;
  if (nickname.contains(q)) return 10;
  return 0;
}

int _tagOrder(MentionTag? tag) => switch (tag) {
  MentionTag.replyTarget => 0,
  MentionTag.topicAuthor => 1,
  MentionTag.participant => 2,
  null => 3,
};

/// 合并去重并排序候选（镜像 Web rankMentionCandidates；排序稳定，与 Web
/// Array.sort 稳定语义一致）：
/// - 空 query：仅本地上下文，最多 5 个，不查服务端；
/// - 有 query：本地上下文（须匹配 query）优先，再按匹配强度排服务端结果；
/// - 按 userId 去重；当前用户默认不进入候选。
List<MentionUser> rankMentionCandidates({
  required List<MentionUser> local,
  required List<MentionUser> server,
  required String query,
  int currentUserId = 0,
  int limit = 8,
}) {
  final seen = <int>{};
  final out = <MentionUser>[];
  void push(MentionUser user) {
    if (user.id == currentUserId || seen.contains(user.id)) return;
    seen.add(user.id);
    out.add(user);
  }

  final localSorted =
      <MentionUser>[for (var i = 0; i < local.length; i++) local[i]]
        ..sort((a, b) {
          final byTag = _tagOrder(a.tag).compareTo(_tagOrder(b.tag));
          if (byTag != 0) return byTag;
          return local.indexOf(a).compareTo(local.indexOf(b));
        });

  final q = query.trim();
  if (q.isEmpty) {
    for (final user in localSorted) {
      push(user);
    }
    return out.take(5).toList();
  }

  for (final user in localSorted) {
    if (mentionUserMatchScore(user, q) > 0) push(user);
  }
  final serverRanked =
      <MapEntry<int, MentionUser>>[
        for (var i = 0; i < server.length; i++)
          if (mentionUserMatchScore(server[i], q) > 0) MapEntry(i, server[i]),
      ]..sort((a, b) {
        final byScore = mentionUserMatchScore(
          b.value,
          q,
        ).compareTo(mentionUserMatchScore(a.value, q));
        if (byScore != 0) return byScore;
        return a.key.compareTo(b.key);
      });
  for (final entry in serverRanked) {
    push(entry.value);
  }
  return out.take(limit).toList();
}

/// 将选中候选写入编辑值：原位替换 token 为 "@username "（追加单个空格），
/// selection 落在插入内容之后，清空 IME composing 区。
TextEditingValue applyMentionReplacement(
  TextEditingValue value,
  MentionReplacement replacement,
) {
  final text = value.text;
  final start = replacement.start.clamp(0, text.length);
  final end = (replacement.start + replacement.length).clamp(
    start,
    text.length,
  );
  return TextEditingValue(
    text: text.replaceRange(start, end, replacement.replacement),
    selection: TextSelection.collapsed(
      offset: start + replacement.replacement.length,
    ),
    composing: TextRange.empty,
  );
}

/// 物理键盘会话处理（宿主挂在回复编辑器的 FocusNode.onKeyEvent 上）：
/// - Escape：只关候选、保留已输入的 @query（不删字符）；
/// - 方向键：移动 active 候选；
/// - Enter：选中 active 候选（经 [onSelect] 写入）；
/// - 仅 KeyDownEvent/KeyRepeatEvent 参与会话；KeyUpEvent 永远忽略，
///   方向键一次只移动一行（FocusNode.onKeyEvent 会同时收到 down 与 up）。
/// - IME 组合中的按键交给输入法确认，不触发补全；
/// - 会话未开启或无候选时返回 ignored，不劫持任何按键（含系统返回通道）。
KeyEventResult handleMentionKeyEvent({
  required MentionSessionController session,
  required TextEditingController controller,
  required KeyEvent event,
  required void Function(MentionToken token, MentionUser user) onSelect,
}) {
  if (!session.open) return KeyEventResult.ignored;
  if (event is! KeyDownEvent && event is! KeyRepeatEvent) {
    return KeyEventResult.ignored;
  }
  if (event.logicalKey == LogicalKeyboardKey.escape) {
    session.close();
    return KeyEventResult.handled;
  }
  if (controller.value.composing != TextRange.empty) {
    return KeyEventResult.ignored;
  }
  if (session.candidates.isEmpty) return KeyEventResult.ignored;
  switch (event.logicalKey) {
    case LogicalKeyboardKey.arrowDown:
      session.moveActive(1);
      return KeyEventResult.handled;
    case LogicalKeyboardKey.arrowUp:
      session.moveActive(-1);
      return KeyEventResult.handled;
    case LogicalKeyboardKey.enter:
    case LogicalKeyboardKey.numpadEnter:
      final user = session.activeCandidate;
      final token = session.token;
      if (user == null || token == null) return KeyEventResult.ignored;
      session.close();
      onSelect(token, user);
      return KeyEventResult.handled;
  }
  return KeyEventResult.ignored;
}

/// 会话编排：token 识别、空 query 本地候选、300ms 防抖搜索、stale 响应丢弃、
/// active 行管理。UI 通过 [notifyListeners]（ListenableBuilder）订阅。
class MentionSessionController extends ChangeNotifier {
  MentionSessionController({
    required this._searchUsers,
    this.debounce = const Duration(milliseconds: 300),
  });

  /// 空 query 展示上限（仅本地上下文）。
  static const int localContextLimit = 5;

  /// 有 query 时候选总数上限。
  static const int candidateLimit = 8;

  final Future<List<MentionUser>> Function(String query) _searchUsers;
  final Duration debounce;

  bool _open = false;
  MentionToken? _token;
  String _query = '';
  List<MentionUser> _candidates = const <MentionUser>[];
  int _activeIndex = 0;
  bool _loading = false;
  bool _failed = false;
  List<MentionUser> _local = const <MentionUser>[];
  int _currentUserId = 0;

  int _searchSeq = 0;
  Timer? _debounceTimer;
  String _lastScheduledQuery = '';

  bool get open => _open;
  MentionToken? get token => _token;
  String get query => _query;
  List<MentionUser> get candidates => _candidates;
  int get activeIndex => _activeIndex;
  bool get loading => _loading;
  bool get failed => _failed;

  MentionUser? get activeCandidate {
    if (_candidates.isEmpty) return null;
    return _candidates[_activeIndex.clamp(0, _candidates.length - 1)];
  }

  /// 更新本地上下文候选与当前用户 id（页面数据或回复目标变化时调用）。
  /// 会话开启中且上下文实际变化时立即重排当前候选，避免已清除的回复目标
  /// 以「正在回复」身份残留居首。
  void updateContext({
    required List<MentionUser> local,
    required int currentUserId,
  }) {
    final bool changed =
        currentUserId != _currentUserId || !listEquals(local, _local);
    _local = local;
    _currentUserId = currentUserId;
    if (!changed || !_open) return;
    if (_query.trim().isEmpty) {
      // 空 query：本地候选直接重排（空 query 本就无在途搜索）。
      _cancelPendingSearch();
      _candidates = rankMentionCandidates(
        local: _local,
        server: const <MentionUser>[],
        query: '',
        currentUserId: _currentUserId,
      );
    } else {
      // 有 query：先按本地匹配重排；在途服务端响应落地时会按新上下文合并。
      _candidates = rankMentionCandidates(
        local: _local,
        server: const <MentionUser>[],
        query: _query,
        currentUserId: _currentUserId,
      );
    }
    _activeIndex = 0;
    notifyListeners();
  }

  /// TextField 值变化时驱动会话。[textBeforeCaret] 为光标前文本；null 表示
  /// 光标状态不可用（未获得焦点等），直接关闭会话。
  void handleValue(String? textBeforeCaret) {
    final token = textBeforeCaret == null
        ? null
        : extractMentionToken(textBeforeCaret);
    _token = token;
    if (token == null) {
      close();
      return;
    }
    _open = true;
    _query = token.query;
    if (token.query.trim().isEmpty) {
      // 仅输入 @：本地上下文（最多 5），不查服务端。
      _cancelPendingSearch();
      _failed = false;
      _candidates = rankMentionCandidates(
        local: _local,
        server: const <MentionUser>[],
        query: '',
        currentUserId: _currentUserId,
      );
      _activeIndex = 0;
      notifyListeners();
      return;
    }
    if (token.query != _lastScheduledQuery) {
      // query 变化立即以本地匹配渲染并废弃在途旧响应（防 debounce 窗口内
      // 旧结果覆盖新 query）。
      _candidates = rankMentionCandidates(
        local: _local,
        server: const <MentionUser>[],
        query: token.query,
        currentUserId: _currentUserId,
      );
      _activeIndex = 0;
      _failed = false;
      _lastScheduledQuery = token.query;
      _searchSeq++;
    }
    _debounceTimer?.cancel();
    _debounceTimer = Timer(debounce, _runSearch);
    notifyListeners();
  }

  Future<void> _runSearch() async {
    final seq = ++_searchSeq;
    final query = _query;
    _loading = true;
    notifyListeners();
    try {
      final users = await _searchUsers(query);
      if (seq != _searchSeq) return;
      _candidates = rankMentionCandidates(
        local: _local,
        server: users,
        query: query,
        currentUserId: _currentUserId,
      );
      _activeIndex = 0;
      _failed = false;
    } catch (_) {
      // 搜索失败不阻塞编辑/发布：保留本地匹配候选并提示，可继续输入。
      if (seq != _searchSeq) return;
      _failed = true;
    } finally {
      if (seq == _searchSeq) {
        _loading = false;
        notifyListeners();
      }
    }
  }

  void _cancelPendingSearch() {
    _debounceTimer?.cancel();
    _searchSeq++;
    _loading = false;
  }

  void moveActive(int delta) {
    final count = _candidates.length;
    if (count == 0) return;
    _activeIndex = (_activeIndex + delta + count) % count;
    notifyListeners();
  }

  /// 关闭会话（幂等）：取消防抖与在途搜索，清空候选态。焦点丢失、caret
  /// 离开 token、composer 关闭时调用。
  void close() {
    if (!_open) return;
    _cancelPendingSearch();
    _lastScheduledQuery = '';
    _open = false;
    _token = null;
    _query = '';
    _candidates = const <MentionUser>[];
    _activeIndex = 0;
    _failed = false;
    notifyListeners();
  }

  @override
  void dispose() {
    _debounceTimer?.cancel();
    // 作废在途序列：dispose 后落地的搜索不得再 notifyListeners（调试断言）。
    _searchSeq++;
    super.dispose();
  }
}
