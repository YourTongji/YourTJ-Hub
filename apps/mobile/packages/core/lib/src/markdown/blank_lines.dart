/// 连续空行（#1584 / #1630 DQ-1，docs/decisions/0128-visible-blank-lines.md）。
///
/// Markdown 原文是唯一来源。渲染时，顶层块之间一段空行里第一行之后的每个空行
/// 显示为一个空行占位，每处最多 [maxRenderedBlankLines] 个；代码、数学、列表与
/// 缩进续行内部不处理。做法是在这些位置插入只含标记的段落，交给原有解析器后再
/// 换成占位，因此不改变块结构。与 Go `markdown2html/blank_lines.go`、Web
/// `runtime/blank-lines.ts` 同一规则，共用
/// `packages/api-contract/fixtures/markdown-blank-lines.json`。
library;

const int maxRenderedBlankLines = 3;

const String _markerPrefix = 'GFBLANKLINE';
final RegExp _fence = RegExp(r'^(`{3,}|~{3,})');
final RegExp _listItem = RegExp(r'^(?:[*+-]|\d{1,9}[.)])(?:[ \t]|$)');
// CommonMark 的空行只含空格与制表符；全角空格等算正文。
final RegExp _blank = RegExp(r'^[ \t]*$');
final RegExp _run = RegExp(r'\n[ \t]*\r?\n[ \t]*\r?\n');
final RegExp _lineBreak = RegExp(r'\r\n?|\n');
final RegExp _leadingSpaces = RegExp(r'^[ \t]+');
final RegExp _trailingSpaces = RegExp(r'[ \t]+$');

/// 返回原文里不存在的标记，保证换回占位时不会误伤正文。
String blankLineMarker(String source) {
  var marker = _markerPrefix;
  for (var suffix = 0; source.contains(marker); suffix++) {
    marker = '$_markerPrefix$suffix';
  }
  return marker;
}

/// 在可见空行处插入标记段落；没有可见空行时原样返回 [source]。
/// [max] 为每处上限，编辑器载入时传 null 以保留作者的全部空段。
String markBlankLines(
  String source,
  String marker, {
  int? max = maxRenderedBlankLines,
}) {
  if ((max != null && max < 1) || !_run.hasMatch(source)) return source;

  final List<String> out = <String>[];
  List<String> pending = <String>[];
  String? fence;
  var math = false;
  var inList = false;
  var seenBlock = false;
  var changed = false;

  for (final String line in source.split(_lineBreak)) {
    if (fence != null || math) {
      out.add(line);
      if (fence != null ? _closesFence(line, fence) : _trim(line) == r'$$') {
        fence = null;
        math = false;
      }
      continue;
    }
    if (_blank.hasMatch(line)) {
      pending.add(line);
      continue;
    }

    final int indent = _indentWidth(line);
    final String body = line.replaceFirst(_leadingSpaces, '');
    final bool listItem = indent < 4 && _listItem.hasMatch(body);
    var extra = pending.length - 1;
    if (max != null && extra > max) extra = max;
    // 只在不缩进的新块之前插入：缩进行可能是列表续行或缩进代码，同类列表项
    // 之间的空行属于同一个列表。不确定时宁可不显示，也不改动结构。
    if (seenBlock && extra > 0 && indent == 0 && !(inList && listItem)) {
      out.add('');
      for (var count = 0; count < extra; count++) {
        out
          ..add(marker)
          ..add('');
      }
      changed = true;
    } else {
      out.addAll(pending);
    }
    if (listItem) {
      inList = true;
    } else if (pending.isNotEmpty && indent == 0) {
      inList = false;
    }
    pending = <String>[];
    out.add(line);
    seenBlock = true;

    if (indent < 4) {
      final String? open = _fence.firstMatch(body)?.group(1);
      if (open != null &&
          !(open[0] == '`' && body.substring(open.length).contains('`'))) {
        fence = open;
      } else if (body.replaceFirst(_trailingSpaces, '') == r'$$') {
        math = true;
      }
    }
  }
  out.addAll(pending);
  return changed ? out.join('\n') : source;
}

bool _closesFence(String line, String fence) {
  if (_indentWidth(line) >= 4) return false;
  final String body = _trim(line);
  if (body.length < fence.length) return false;
  for (var i = 0; i < body.length; i++) {
    if (body[i] != fence[0]) return false;
  }
  return true;
}

String _trim(String line) =>
    line.replaceFirst(_leadingSpaces, '').replaceFirst(_trailingSpaces, '');

int _indentWidth(String line) {
  var width = 0;
  for (var i = 0; i < line.length; i++) {
    final String char = line[i];
    if (char == ' ') {
      width++;
    } else if (char == '\t') {
      width += 4 - width % 4;
    } else {
      break;
    }
  }
  return width;
}
