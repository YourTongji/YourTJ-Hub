/// 表情包 token（`[:sticker:name:]`）纯文本解析（MADR 0030）。
///
/// 规则对齐服务端 markdown2html.stickerTokenRe 与 web 端 sticker-token.ts：
/// name 为 1-64 个非空白/冒号/方括号字符（Unicode，支持中文）；未知或停用
/// token 保持原文。服务端在代码块/链接等 AST 区间内不展开 token，移动端
/// 帖子也按 Markdown AST 排除这些上下文；私信按纯文本分段。
library;

import 'package:markdown/markdown.dart' as md;

/// token 匹配规则（与 markdown2html.stickerTokenRe 一致）。
final RegExp stickerTokenPattern = RegExp(r'\[:sticker:([^:\[\]\s]{1,64}):\]');

// Match before block parsing so display math spanning blank lines stays opaque.
final _stickerMathPattern = RegExp(
  r'\$\$[\s\S]+?\$\$|\$(?!\s)[^\n]+?(?<!\s)\$'
  r'|\\\([\s\S]+?\\\)|\\\[[\s\S]+?\\\]'
  r'|\\begin\{(equation\*?|align\*?|aligned|gather\*?|split|cases|matrix|pmatrix|bmatrix|vmatrix|Vmatrix)\}[\s\S]+?\\end\{\1\}',
);

/// 内容是否可能包含表情包 token（各渲染入口的快路径判断）。
bool containsStickerToken(String content) => content.contains('[:sticker:');

/// 帖子 markdown 渲染前的 token 展开：`[:sticker:name:]` 重写为
/// `![sticker:name](url)` with an explicit semantic marker. Renderers must use
/// the sticker image branch and exclude these entries from photo galleries.
///
/// [urlByName] 中缺失（未知/停用）的 token 保持原文；无 token 或全部未知
/// 时原样返回同一字符串。
String expandStickerTokens(String content, Map<String, String> urlByName) {
  if (!containsStickerToken(content) || urlByName.isEmpty) return content;
  final matches = stickerTokenPattern.allMatches(content).toList();
  if (matches.isEmpty) return content;

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
  final math = _stickerMathPattern.allMatches(content).toList();
  final expanded = StringBuffer();
  int cursor = 0;
  for (int i = 0; i < matches.length; i++) {
    final match = matches[i];
    final url = urlByName[match.group(1)];
    if (!eligible.contains(i) || url == null || url.isEmpty) continue;
    int slashes = 0;
    for (int j = match.start - 1; j >= 0 && content[j] == r'\'; j--) {
      slashes++;
    }
    if (slashes.isOdd ||
        math.any((span) => match.start < span.end && span.start < match.end)) {
      continue;
    }
    final safeUrl = url.replaceAllMapped(
      RegExp(r'[()<>\s\\]'),
      (m) => Uri.encodeComponent(m[0]!),
    );
    // Uri.encodeComponent leaves parentheses unescaped.
    final destination = safeUrl.replaceAll('(', '%28').replaceAll(')', '%29');
    expanded
      ..write(content.substring(cursor, match.start))
      ..write('![sticker:${match.group(1)}]($destination)');
    cursor = match.end;
  }
  if (cursor == 0) return content;
  return (expanded..write(content.substring(cursor))).toString();
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

/// 会话列表预览用可读标签：token 缩写为 `[name]`（预览层不区分启停，
/// 对齐 web stickerPreviewLabel）。
String stickerPreviewLabel(String content) {
  if (!containsStickerToken(content)) return content;
  return content.replaceAllMapped(
    stickerTokenPattern,
    (Match match) => '[${match.group(1)}]',
  );
}
