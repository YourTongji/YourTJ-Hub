/// 提取「可见文本本身是绝对 http(s) URL」的 Markdown 链接（issue #1200 review）。
///
/// Web 端在渲染后的 DOM 上判定「标签与目标不符」（external-link-guard 的
/// linkRisk）：anchor 文本能解析为站外 URL 且 registrable domain 与目标不同即
/// suspicious，始终弹出确认层。Flutter 的渲染器（markdown_widget）只把 href 交给
/// 点击回调，看不到标签，因此在渲染前用同一份 Markdown AST 取到等价的
/// label/href 对；registrable domain 判定留在 forum_app（PSL 逻辑只有一份）。
library;

import 'package:markdown/markdown.dart' as md;

/// 一个「链接文本是 URL」的链接：label 为渲染后 anchor 的可见文本。
class UrlLabeledLink {
  const UrlLabeledLink({required this.href, required this.label});

  final String href;
  final String label;

  @override
  bool operator ==(Object other) =>
      other is UrlLabeledLink && other.href == href && other.label == label;

  @override
  int get hashCode => Object.hash(href, label);

  @override
  String toString() => 'UrlLabeledLink($href, $label)';
}

/// 解析 [markdown] 并返回文本是绝对 http(s) URL 的链接（含引用式链接与
/// 强调嵌套）。解析参数与 markdown_widget 的默认渲染一致（GFM 扩展集），
/// 保证扫描到的节点与用户实际点击的链接相同。
List<UrlLabeledLink> urlLabeledMarkdownLinks(String markdown) {
  // 链接必须来自 `[...]` 或 `<...>`；其余正文不必再解析一遍。
  if (!markdown.contains('[') && !markdown.contains('<')) {
    return const <UrlLabeledLink>[];
  }
  final List<md.Node> nodes = md.Document(
    extensionSet: md.ExtensionSet.gitHubFlavored,
  ).parseLines(markdown.split('\n'));

  final List<UrlLabeledLink> links = <UrlLabeledLink>[];
  void visit(md.Node node) {
    if (node is! md.Element) return;
    if (node.tag == 'a') {
      final String? href = node.attributes['href'];
      final String label = node.textContent.trim();
      if (href != null &&
          href.isNotEmpty &&
          label.isNotEmpty &&
          isAbsoluteHttpUrl(label)) {
        links.add(UrlLabeledLink(href: href, label: label));
      }
    }
    for (final md.Node child in node.children ?? const <md.Node>[]) {
      visit(child);
    }
  }

  for (final md.Node node in nodes) {
    visit(node);
  }
  return links;
}

/// [value] 本身是绝对 http(s) URL（标签判定的唯一实现，Markdown 与 HTML
/// 链路共用；见 [UrlLabeledLink]）。
bool isAbsoluteHttpUrl(String value) {
  final Uri? uri = Uri.tryParse(value);
  if (uri == null || !uri.hasScheme || uri.host.isEmpty) return false;
  final String scheme = uri.scheme.toLowerCase();
  return scheme == 'http' || scheme == 'https';
}
