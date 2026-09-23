/// 单篇正文最多接管的卡片数。取 5 以对齐接口边界：`LinkPreviewResolveRequest`
/// 的 `maxItems` 是 5，`LinkPreviewRepository.resolve` 也按 5 截断，所以一篇帖子
/// 的候选恰好能一次请求拿完。
const int maxLinkPreviewCandidates = 5;

class LinkPreviewMarkdownBlock {
  const LinkPreviewMarkdownBlock._({required this.markdown, this.url});

  factory LinkPreviewMarkdownBlock.markdown(String markdown) =>
      LinkPreviewMarkdownBlock._(markdown: markdown);

  factory LinkPreviewMarkdownBlock.preview(String markdown, String url) =>
      LinkPreviewMarkdownBlock._(markdown: markdown, url: url);

  final String markdown;
  final String? url;

  bool get isPreview => url != null;
}

List<LinkPreviewMarkdownBlock> splitLinkPreviewMarkdown(String markdown) {
  if (markdown.trim().isEmpty) {
    return <LinkPreviewMarkdownBlock>[
      LinkPreviewMarkdownBlock.markdown(markdown),
    ];
  }

  final String normalized = markdown
      .replaceAll('\r\n', '\n')
      .replaceAll('\r', '\n');
  final List<String> lines = normalized.split('\n');
  final List<LinkPreviewMarkdownBlock> blocks = <LinkPreviewMarkdownBlock>[];
  int normalStart = 0;
  int previewCount = 0;
  String? fence;

  void addNormal(int end) {
    final String value = lines.sublist(normalStart, end).join('\n');
    if (value.trim().isNotEmpty) {
      blocks.add(LinkPreviewMarkdownBlock.markdown(value));
    }
  }

  for (int index = 0; index < lines.length; index++) {
    // CommonMark：行首缩进 ≥4 空格（制表符按 4 计）是缩进代码块，其中的裸 URL
    // 不是候选——Web 端 markdown-it 会把它渲染成 pre/code 同样不出卡，两侧必须
    // 同语义（共享 fixture 的 indented code block 用例）。缩进行也不得切换围栏
    // 状态：缩进的 ``` 只是代码内容，不是围栏标记。
    if (fence == null && _indentOf(lines[index]) >= 4) continue;
    final String trimmedLeft = lines[index].trimLeft();
    final String? marker = _fenceMarker(trimmedLeft);
    if (marker != null) {
      if (fence == null) {
        fence = marker;
      } else if (marker == fence) {
        fence = null;
      }
      continue;
    }
    if (fence != null || previewCount >= maxLinkPreviewCandidates) continue;

    final bool previousBlank = index == 0 || lines[index - 1].trim().isEmpty;
    final bool nextBlank =
        index == lines.length - 1 || lines[index + 1].trim().isEmpty;
    final String? url = previousBlank && nextBlank
        ? _standaloneHttpUrl(lines[index])
        : null;
    if (url == null) continue;

    addNormal(index);
    blocks.add(LinkPreviewMarkdownBlock.preview(lines[index], url));
    previewCount++;
    normalStart = index + 1;
  }

  addNormal(lines.length);
  if (blocks.isEmpty) blocks.add(LinkPreviewMarkdownBlock.markdown(normalized));
  return blocks;
}

List<String> scanLinkPreviewCandidates(String markdown) =>
    splitLinkPreviewMarkdown(markdown)
        .where((LinkPreviewMarkdownBlock block) => block.isPreview)
        .map((block) => block.url!)
        .toList(growable: false);

String? _fenceMarker(String line) {
  if (line.length < 3) return null;
  final String character = line[0];
  if (character != '`' && character != '~') return null;
  int count = 0;
  while (count < line.length && line[count] == character) {
    count++;
  }
  return count >= 3 ? character : null;
}

int _indentOf(String line) {
  int indent = 0;
  for (int index = 0; index < line.length; index++) {
    final int codeUnit = line.codeUnitAt(index);
    if (codeUnit == 0x20) {
      indent++;
    } else if (codeUnit == 0x09) {
      indent += 4;
    } else {
      break;
    }
  }
  return indent;
}

String? _standaloneHttpUrl(String raw) {
  final String value = raw.trim();
  if (value.isEmpty || value.length > 2048 || value.contains('\\')) return null;
  for (final int rune in value.runes) {
    if (rune < 0x20 || rune == 0x7f) return null;
  }
  final Uri? uri = Uri.tryParse(value);
  if (uri == null ||
      !uri.hasAuthority ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty) {
    return null;
  }
  final String scheme = uri.scheme.toLowerCase();
  return scheme == 'http' || scheme == 'https' ? value : null;
}
