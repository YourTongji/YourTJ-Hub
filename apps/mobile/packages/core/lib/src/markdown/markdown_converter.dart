import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:flutter_quill/quill_delta.dart' as quill_delta;
import 'package:markdown/markdown.dart' as md;
import 'package:markdown_quill/markdown_quill.dart';

import 'sticker_token.dart';

/// Markdown ↔ flutter_quill Delta/Document 转换层。
///
/// 读取链路:markdown → MarkdownToDelta(GFM + 可嵌入表格语法)→ Delta → Document;
/// 保存链路:Document → Delta → DeltaToMarkdown → markdown。
/// 已知取舍(与 web 端一致):
/// - 图片保留 URL,alt 文本丢失;
/// - 任务列表双向转换 OK;
/// - 表格只读保留(编辑不支持);
/// - 引用内标题降级为引用文本。
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
        },
      );

  final MarkdownToDelta _mdToDelta;
  final DeltaToMarkdown _deltaToMarkdown;

  /// markdown → Delta。
  ///
  /// 空输入返回空 Delta(保持 `deltaToMarkdown` 往返为空语义)。
  quill_delta.Delta mdToDelta(String markdown) {
    final normalized = markdown.trimRight();
    return _mdToDelta.convert(normalized.isEmpty ? '\n' : normalized);
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
    return _deltaToMarkdown.convert(delta);
  }

  /// Document → markdown(编辑器保存)。
  String documentToMarkdown(quill.Document document) {
    return deltaToMarkdown(document.toDelta());
  }
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
