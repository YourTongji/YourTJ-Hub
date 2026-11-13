import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:flutter_quill/quill_delta.dart' as quill_delta;
import 'package:markdown/markdown.dart' as md;
import 'package:markdown_quill/markdown_quill.dart';

import 'blank_lines.dart';
import 'sticker_embed.dart';
import 'sticker_token.dart';

/// Markdown ↔ flutter_quill Delta/Document 转换层。
///
/// 读取链路:markdown → MarkdownToDelta(GFM + 可嵌入表格语法)→ Delta → Document;
/// 保存链路:Document → Delta → DeltaToMarkdown → markdown。
/// 已知取舍(与 web 端一致):
/// - 图片保留 URL,alt 文本丢失;
/// - 任务列表双向转换 OK;
/// - 表格只读保留(编辑不支持);
/// - 引用内标题降级为引用文本;
/// - 顶层块之间的连续空行按 Web 编辑器同一约定往返:k 个空段写成 k+1 行空行
///   (docs/decisions/0133-visible-blank-lines.md)。
class MarkdownConverter {
  MarkdownConverter()
    : _mdToDelta = MarkdownToDelta(
        markdownDocument: md.Document(
          encodeHtml: false,
          extensionSet: md.ExtensionSet.gitHubFlavored,
          blockSyntaxes: [const EmbeddableTableSyntax()],
        ),
        customElementToEmbeddable: {
          EmbeddableTable.tableType: EmbeddableTable.fromMdSyntax,
        },
      ),
      _deltaToMarkdown = DeltaToMarkdown(
        customContentHandler: _writeStickerAwareText,
        customEmbedHandlers: {
          EmbeddableTable.tableType: EmbeddableTable.toMdSyntax,
          stickerEmbedType: (embed, out) => out.write(stickerEmbedToken(embed)),
        },
      );

  final MarkdownToDelta _mdToDelta;
  final DeltaToMarkdown _deltaToMarkdown;

  /// markdown → Delta。
  ///
  /// 空输入返回空 Delta(保持 `deltaToMarkdown` 往返为空语义)。
  quill_delta.Delta mdToDelta(String markdown) {
    final normalized = markdown.trimRight();
    // 解析器会合并连续空行:载入前插入标记段落,转换后清空成空行,保留作者的空段。
    final marker = blankLineMarker(normalized);
    final marked = markBlankLines(normalized, marker, max: null);
    var delta = _mdToDelta.convert(marked.isEmpty ? '\n' : marked);
    if (marked != normalized) delta = _clearMarkerLines(delta, marker);
    // Prose tokens load as sticker embeds: the editor shows images, not tokens.
    return embedStickerTokens(delta);
  }

  /// markdown → Document(编辑器读取)。
  ///
  /// flutter_quill 的 Document 拒绝空 Delta("Document Delta cannot be
  /// empty"),因此空输入/空转换结果兜底为只含单个换行的最小 Document。
  quill.Document mdToDocument(String markdown) {
    final delta = mdToDelta(markdown);
    return delta.isEmpty
        ? quill.Document.fromDelta(quill_delta.Delta()..insert('\n'))
        : quill.Document.fromDelta(delta);
  }

  /// Delta → markdown。
  String deltaToMarkdown(quill_delta.Delta delta) {
    if (delta.isEmpty) return '';
    // 上游把每个空行写成两个换行(标题后只写一个);先换成标记段落,输出后把
    // 含 k 个标记的空白区统一写成 k+1 行空行,与 Web 编辑器一致。
    final marker = blankLineMarker(_plainText(delta));
    final marked = _markEmptyLines(delta, marker);
    final markdown = _deltaToMarkdown.convert(marked);
    if (identical(marked, delta)) return markdown;
    return _markerRunsToBlankLines(markdown, marker);
  }

  /// Document → markdown(编辑器保存)。
  String documentToMarkdown(quill.Document document) {
    return deltaToMarkdown(document.toDelta());
  }
}

String _plainText(quill_delta.Delta delta) =>
    delta.toList().map((op) => op.data).whereType<String>().join();

/// 无格式的空行(前一个字符是换行或文档开头的无属性换行)换成标记段落。
/// 代码块、列表等空行的换行带块属性,保持原样。
quill_delta.Delta _markEmptyLines(quill_delta.Delta delta, String marker) {
  final result = quill_delta.Delta();
  var lineStart = true;
  var changed = false;
  for (final op in delta.toList()) {
    final data = op.data;
    if (!op.isInsert || data is! String) {
      result.push(op);
      lineStart = false;
      continue;
    }
    if (op.attributes != null) {
      result.push(op);
      if (data.isNotEmpty) lineStart = data.endsWith('\n');
      continue;
    }
    final text = StringBuffer();
    for (var i = 0; i < data.length; i++) {
      if (data[i] == '\n' && lineStart) {
        text.write(marker);
        changed = true;
      }
      text.write(data[i]);
      lineStart = data[i] == '\n';
    }
    result.insert(text.toString());
  }
  return changed ? result : delta;
}

/// 标记行与相邻空行组成的空白区:k 个标记写成 k+1 行空行。
String _markerRunsToBlankLines(String markdown, String marker) {
  final lines = markdown.split('\n');
  final out = <String>[];
  var index = 0;
  while (index < lines.length) {
    if (lines[index].isNotEmpty && lines[index] != marker) {
      out.add(lines[index++]);
      continue;
    }
    final start = index;
    var markers = 0;
    while (index < lines.length &&
        (lines[index].isEmpty || lines[index] == marker)) {
      if (lines[index++] == marker) markers++;
    }
    out.addAll(
      markers == 0 ? lines.sublist(start, index) : List.filled(markers + 1, ''),
    );
  }
  return out.join('\n');
}

/// 删除只含标记的整行文本,留下空行。
quill_delta.Delta _clearMarkerLines(quill_delta.Delta delta, String marker) {
  final result = quill_delta.Delta();
  var lineStart = true;
  for (final op in delta.toList()) {
    final data = op.data;
    if (!op.isInsert || data is! String || op.attributes != null) {
      result.push(op);
      if (data is String && data.isNotEmpty) {
        lineStart = data.endsWith('\n');
      } else {
        lineStart = false;
      }
      continue;
    }
    final lines = data.split('\n');
    final text = StringBuffer();
    for (var i = 0; i < lines.length; i++) {
      final wholeLine = (i > 0 || lineStart) && i < lines.length - 1;
      if (!(wholeLine && lines[i] == marker)) text.write(lines[i]);
      if (i < lines.length - 1) text.write('\n');
    }
    if (text.isNotEmpty) result.insert(text.toString());
    if (data.isNotEmpty) lineStart = data.endsWith('\n');
  }
  return result;
}

// Keep explicit expression tokens round-trippable. The upstream converter
// escapes brackets and underscores, which otherwise disables token rendering
// on save. Ordinary prose, literal escapes, links and code retain its rules.
void _writeStickerAwareText(quill.QuillText text, StringSink output) {
  final style = text.style;
  if (!containsStickerToken(text.value) ||
      style.containsKey(quill.Attribute.codeBlock.key) ||
      style.containsKey(quill.Attribute.inlineCode.key) ||
      style.containsKey(quill.Attribute.link.key) ||
      (text.parent?.style.containsKey(quill.Attribute.codeBlock.key) ??
          false)) {
    DeltaToMarkdown.escapeSpecialCharacters(text, output);
    return;
  }
  String escape(String value) => value.replaceAllMapped(
    RegExp(r'[\\\`\*\_\{\}\[\]\(\)\#\+\-\.\!\>\<]'),
    (match) => '\\${match[0]}',
  );
  var cursor = 0;
  for (final match in stickerTokenPattern.allMatches(text.value)) {
    var slashes = 0;
    for (var i = match.start - 1; i >= 0 && text.value[i] == r'\'; i--) {
      slashes++;
    }
    if (slashes.isOdd) continue;
    output.write(escape(text.value.substring(cursor, match.start)));
    output.write(match[0]);
    cursor = match.end;
  }
  output.write(escape(text.value.substring(cursor)));
}
