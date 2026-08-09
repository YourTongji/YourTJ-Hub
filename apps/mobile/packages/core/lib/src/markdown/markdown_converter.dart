import 'package:flutter_quill/flutter_quill.dart' as quill;
import 'package:flutter_quill/quill_delta.dart' as quill_delta;
import 'package:markdown/markdown.dart' as md;
import 'package:markdown_quill/markdown_quill.dart';

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
