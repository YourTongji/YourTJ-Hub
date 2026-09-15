/// 表情包 token（`[:sticker:name:]`）纯文本解析（MADR 0022）。
///
/// 规则对齐服务端 markdown2html.stickerTokenRe 与 web 端 sticker-token.ts：
/// name 为 1-64 个非空白/冒号/方括号字符（Unicode，支持中文）；未知或停用
/// token 保持原文。服务端在代码块/链接等 AST 区间内不展开 token，移动端
/// 输入是纯文本场景（帖子 raw markdown / 私信纯文本），不做 AST 排除。
library;

/// token 匹配规则（与 markdown2html.stickerTokenRe 一致）。
final RegExp stickerTokenPattern = RegExp(r'\[:sticker:([^:\[\]\s]{1,64}):\]');

/// 内容是否可能包含表情包 token（各渲染入口的快路径判断）。
bool containsStickerToken(String content) => content.contains('[:sticker:');

/// 帖子 markdown 渲染前的 token 展开：`[:sticker:name:]` 重写为
/// `![sticker:name](url)` 标准图片语法，复用 markdown 渲染链路（移动端
/// markdown 视图的图片点击查看等既有行为随之生效）。
///
/// [urlByName] 中缺失（未知/停用）的 token 保持原文；无 token 或全部未知
/// 时原样返回同一字符串。
String expandStickerTokens(String content, Map<String, String> urlByName) {
  if (!containsStickerToken(content)) return content;
  final StringBuffer expanded = StringBuffer();
  int cursor = 0;
  for (final RegExpMatch match in stickerTokenPattern.allMatches(content)) {
    final String? url = urlByName[match.group(1)];
    if (url == null || match.start < cursor) continue;
    expanded
      ..write(content.substring(cursor, match.start))
      ..write('![sticker:${match.group(1)}]($url)');
    cursor = match.end;
  }
  if (cursor == 0) return content;
  return (expanded..write(content.substring(cursor))).toString();
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
