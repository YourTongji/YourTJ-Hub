/// 表情包 token（`[:sticker:name:]`）纯文本解析（MADR 0030）。
///
/// 规则对齐服务端 markdown2html.stickerTokenRe 与 web 端 sticker-token.ts：
/// name 为 1-64 个非空白/冒号/方括号字符（Unicode，支持中文）；未知或停用
/// token 保持原文。服务端在代码块/链接等 AST 区间内不展开 token，移动端
/// 帖子也按 Markdown AST 排除这些上下文；私信按纯文本分段。
library;

import 'dart:ui' show TextRange;

import 'package:markdown/markdown.dart' as md;

/// token 匹配规则（与 markdown2html.stickerTokenRe 一致）。
final RegExp stickerTokenPattern = RegExp(r'\[:sticker:([^:\[\]\s]{1,64}):\]');

// Match before block parsing so display math spanning blank lines stays opaque.
final _stickerMathPattern = RegExp(
  r'\$\$[\s\S]+?\$\$|\$(?!\s)[^\n]+?(?<!\s)\$'
  r'|\\\([\s\S]+?\\\)|\\\[[\s\S]+?\\\]'
  r'|\\begin\{(equation\*?|align\*?|aligned|gather\*?|split|cases|matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}[\s\S]+?\\end\{\1\}',
);

/// 小字视图（列表行、置顶条）的纯文本贴纸文案：可解析的贴纸显示 `[名字]`，
/// 未知/停用/个人贴纸省略；卡片与预览仍由 `buildStickerMessageSpan` 渲染图片。
String stickerLabelText(String content, Map<String, String> urlByName) {
  if (!containsStickerToken(content)) return content;
  final buffer = StringBuffer();
  var cursor = 0;
  for (final match in stickerTokenPattern.allMatches(content)) {
    buffer.write(content.substring(cursor, match.start));
    cursor = match.end;
    if (urlByName[match.group(1)] case final String url when url.isNotEmpty) {
      buffer.write('[${match.group(1)}]');
    }
  }
  buffer.write(content.substring(cursor));
  return buffer.toString().replaceAll(RegExp(r'\s+'), ' ').trim();
}

/// 内容是否可能包含表情包 token（各渲染入口的快路径判断）。
bool containsStickerToken(String content) => content.contains('[:sticker:');

/// 帖子 markdown 渲染前的 token 展开：`[:sticker:name:]` 重写为
/// `![sticker:name](url)`. The alt label is not proof of sticker origin: renderers
/// must supply private per-render sources and map only those sources to the
/// sticker image branch, excluding those entries from photo galleries.
///
/// [urlByName] 中缺失（未知/停用）的 token 保持原文；无 token 或全部未知
/// 时原样返回同一字符串。
String expandStickerTokens(String content, Map<String, String> urlByName) {
  if (!containsStickerToken(content) || urlByName.isEmpty) return content;
  final tokens = stickerTokenRanges(content, markdown: true);
  if (tokens.isEmpty) return content;
  final expanded = StringBuffer();
  int cursor = 0;
  for (final token in tokens) {
    final url = urlByName[token.name];
    if (url == null || url.isEmpty) continue;
    final safeUrl = url.replaceAllMapped(
      RegExp(r'[()<>\s\\]'),
      (m) => Uri.encodeComponent(m[0]!),
    );
    // Uri.encodeComponent leaves parentheses unescaped.
    final destination = safeUrl.replaceAll('(', '%28').replaceAll(')', '%29');
    expanded
      ..write(content.substring(cursor, token.start))
      ..write('![sticker:${token.name}]($destination)');
    cursor = token.end;
  }
  if (cursor == 0) return content;
  return (expanded..write(content.substring(cursor))).toString();
}

/// 一个可展开为贴纸图的 token 在文本里的区间 `[start, end)`。
class StickerTokenRange {
  const StickerTokenRange(this.start, this.end, this.name);
  final int start;
  final int end;
  final String name;

  @override
  bool operator ==(Object other) =>
      other is StickerTokenRange &&
      other.start == start &&
      other.end == end &&
      other.name == name;

  @override
  int get hashCode => Object.hash(start, end, name);

  @override
  String toString() => 'StickerTokenRange($start, $end, $name)';
}

/// 会被渲染为贴纸的 token 区间（不判断贴纸是否存在）。
///
/// [markdown] 为真时按帖子规则排除代码、链接、原始 HTML 与公式里的 token
/// （与服务端 markdown2html 一致）；私信是纯文本，传 false 只排除被反斜杠转义的。
List<StickerTokenRange> stickerTokenRanges(
  String content, {
  required bool markdown,
}) {
  if (!containsStickerToken(content)) return const [];
  final matches = stickerTokenPattern.allMatches(content).toList();
  if (matches.isEmpty) return const [];
  final eligible = markdown ? _markdownEligible(content) : null;
  final math = markdown
      ? _stickerMathPattern.allMatches(content).toList()
      : const <RegExpMatch>[];
  final ranges = <StickerTokenRange>[];
  for (int i = 0; i < matches.length; i++) {
    final match = matches[i];
    if (eligible != null && !eligible.contains(i)) continue;
    int slashes = 0;
    for (int j = match.start - 1; j >= 0 && content[j] == r'\'; j--) {
      slashes++;
    }
    if (slashes.isOdd ||
        math.any((span) => match.start < span.end && span.start < match.end)) {
      continue;
    }
    ranges.add(StickerTokenRange(match.start, match.end, match.group(1)!));
  }
  return ranges;
}

/// Indices of the token matches that sit in Markdown prose.
Set<int> _markdownEligible(String content) {
  // Parse a shadow copy with unique literal markers, then rewrite the original
  // offsets. Markdown owns code/link/HTML boundaries; underscores in names
  // cannot become emphasis. Preserve the original source formatting.
  String prefix = '\uE000STICKER';
  while (content.contains(prefix)) {
    prefix += 'X';
  }
  final markerPattern = RegExp('${RegExp.escape(prefix)}([0-9]+)\uE001');
  int index = 0;
  final shadow = content.replaceAllMapped(
    stickerTokenPattern,
    (_) => '$prefix${index++}\uE001',
  );
  final document = md.Document(
    extensionSet: md.ExtensionSet.gitHubFlavored,
    encodeHtml: false,
    blockSyntaxes: [const _StickerHtmlBlock()],
    inlineSyntaxes: [_StickerHtmlInline()],
  );
  final eligible = <int>{};
  void collect(md.Node node) {
    if (node is md.Element) {
      if (const {
        'code',
        'pre',
        'a',
        'img',
        'sticker-excluded',
      }.contains(node.tag)) {
        return;
      }
      for (final child in node.children ?? <md.Node>[]) {
        collect(child);
      }
    } else if (node is md.Text) {
      for (final marker in markerPattern.allMatches(node.text)) {
        eligible.add(int.parse(marker.group(1)!));
      }
    }
  }

  for (final node in document.parse(shadow)) {
    collect(node);
  }
  return eligible;
}

// The parser normally merges raw HTML into Text. Keep it opaque so attributes
// and HTML blocks cannot be mistaken for Markdown prose.
class _StickerHtmlBlock extends md.HtmlBlockSyntax {
  const _StickerHtmlBlock();
  @override
  md.Node parse(md.BlockParser parser) =>
      md.Element.text('sticker-excluded', super.parse(parser).textContent);
}

class _StickerHtmlInline extends md.InlineHtmlSyntax {
  @override
  bool onMatch(md.InlineParser parser, Match match) {
    parser.addNode(md.Element.text('sticker-excluded', match[0]!));
    return true;
  }
}

/// 贴纸分段：文本段或已解析的内联图片段。
sealed class StickerMessageSegment {
  const StickerMessageSegment();
}

/// 纯文本段（原样展示）。
class StickerTextSegment extends StickerMessageSegment {
  const StickerTextSegment(this.text);

  final String text;
}

/// 已知启用表情包的内联图片段。
class StickerImageSegment extends StickerMessageSegment {
  const StickerImageSegment({required this.name, required this.url});

  final String name;
  final String url;
}

/// 把私信文本切分为文本/贴纸段序列（对齐 web parseStickerSegments）。
///
/// 未识别（不在 [urlByName] 中）或停用的 token 保持原文；无 token 快路径
/// 返回单文本段。整条消息都是未知 token 时也返回单文本段原文（web 模板
/// 在该形态下渲染为空，这里保持原文更安全）。
List<StickerMessageSegment> parseStickerSegments(
  String content,
  Map<String, String> urlByName,
) {
  if (!containsStickerToken(content)) {
    return <StickerMessageSegment>[StickerTextSegment(content)];
  }
  final List<StickerMessageSegment> segments = <StickerMessageSegment>[];
  int cursor = 0;
  for (final RegExpMatch match in stickerTokenPattern.allMatches(content)) {
    final String? url = urlByName[match.group(1)];
    if (url == null || match.start < cursor) continue;
    if (match.start > cursor) {
      segments.add(StickerTextSegment(content.substring(cursor, match.start)));
    }
    segments.add(StickerImageSegment(name: match.group(1)!, url: url));
    cursor = match.end;
  }
  if (segments.isEmpty) {
    return <StickerMessageSegment>[StickerTextSegment(content)];
  }
  if (cursor < content.length) {
    segments.add(StickerTextSegment(content.substring(cursor)));
  }
  return segments;
}

/// 会话列表预览/回复引用用可读标签。传入 `stickerLabel` 时所有 token
/// 统一替换为该本地化占位（不暴露随机 token 名，issue #1071）；缺省回退
/// 为 `[name]`（旧行为兼容，目前仅测试覆盖）。面向用户的预览与引用必须传 label。
String stickerPreviewLabel(String content, {String? stickerLabel}) {
  if (!containsStickerToken(content)) return content;
  return content.replaceAllMapped(
    stickerTokenPattern,
    (Match match) => '[${stickerLabel ?? match.group(1)}]',
  );
}

const String _stickerTokenPrefix = '[:sticker:';
const int _stickerNameMax = 64;

bool _isStickerNameChar(String content, int index) =>
    index < content.length && !RegExp(r'[\s:\[\]]').hasMatch(content[index]);

/// 编辑器里应作为一个原子块删除的区间：完整 token 与未闭合片段（对齐 web
/// sticker-token.ts 的 stickerAtomSpans，issue #1332）。片段 = `[:sticker:` +
/// 至多 64 个名字字符且未以 `:]` 收尾；名字后紧跟的单个 `:` 也算片段的一部分。
List<TextRange> stickerAtomRanges(String content) {
  final ranges = <TextRange>[];
  for (
    int start = content.indexOf(_stickerTokenPrefix);
    start != -1;
    start = content.indexOf(_stickerTokenPrefix, start + 1)
  ) {
    final nameStart = start + _stickerTokenPrefix.length;
    int end = nameStart;
    while (end - nameStart < _stickerNameMax &&
        _isStickerNameChar(content, end)) {
      end++;
    }
    // Longer than a valid name: neither a token nor a fragment.
    if (_isStickerNameChar(content, end)) continue;
    final hasName = end > nameStart;
    if (hasName && content.startsWith(':]', end)) {
      ranges.add(TextRange(start: start, end: end + 2));
      continue;
    }
    if (hasName && end < content.length && content[end] == ':') {
      ranges.add(TextRange(start: start, end: end + 1));
      continue;
    }
    if (hasName || end == content.length) {
      ranges.add(TextRange(start: start, end: end));
    }
  }
  return ranges;
}

/// 删除了 `[start, end)` 后应实际删除的区间：与之相交但未被完整覆盖的贴纸
/// token/片段整体并入（一次退格删掉整个贴纸）；不涉及贴纸时原样返回。
TextRange expandStickerDeletion(String content, int start, int end) {
  var from = start;
  var to = end;
  for (final atom in stickerAtomRanges(content)) {
    final overlaps = atom.start < end && start < atom.end;
    if (!overlaps) continue;
    if (atom.start < from) from = atom.start;
    if (atom.end > to) to = atom.end;
  }
  return TextRange(start: from, end: to);
}

/// 光标前正在手动输入的贴纸片段（`[:sticker:` + 至多 64 个名字字符）。
class StickerQuery {
  const StickerQuery({required this.length, required this.query});

  /// 片段长度（含 `[:sticker:` 前缀），选中候选时整段替换。
  final int length;
  final String query;

  @override
  bool operator ==(Object other) =>
      other is StickerQuery && other.length == length && other.query == query;

  @override
  int get hashCode => Object.hash(length, query);
}

final RegExp _stickerQueryPattern = RegExp(r'\[:sticker:([^\s:\[\]]{0,64})$');

/// 从光标前文本识别手动输入中的片段；被奇数个反斜杠转义时不算（对齐 web）。
StickerQuery? extractStickerQuery(String prefix) {
  final match = _stickerQueryPattern.firstMatch(prefix);
  if (match == null) return null;
  int slashes = 0;
  for (int i = match.start - 1; i >= 0 && prefix[i] == r'\'; i--) {
    slashes++;
  }
  if (slashes.isOdd) return null;
  return StickerQuery(length: match[0]!.length, query: match.group(1)!);
}

/// 候选排序（对齐 web rankStickerSuggestions）：精确 > 名字/显示名前缀 > 包含；
/// 同级保持来源顺序，名字去重时先出现的来源优先；空查询按来源顺序取前 [limit] 个。
List<T> rankStickerSuggestions<T>(
  Iterable<T> items,
  String query, {
  required String Function(T) name,
  required String Function(T) label,
  int limit = 8,
}) {
  final q = query.toLowerCase();
  final seen = <String>{};
  final scored = <({T item, int score, int index})>[];
  var index = 0;
  for (final item in items) {
    final itemName = name(item);
    final position = index++;
    if (!seen.add(itemName)) continue;
    if (q.isEmpty) {
      scored.add((item: item, score: 0, index: position));
      continue;
    }
    final n = itemName.toLowerCase();
    final l = label(item).toLowerCase();
    final score = n == q
        ? 4
        : n.startsWith(q) || l.startsWith(q)
        ? 3
        : n.contains(q) || l.contains(q)
        ? 2
        : 0;
    if (score > 0) scored.add((item: item, score: score, index: position));
  }
  scored.sort(
    (a, b) => a.score != b.score ? b.score - a.score : a.index - b.index,
  );
  return [for (final entry in scored.take(limit)) entry.item];
}
