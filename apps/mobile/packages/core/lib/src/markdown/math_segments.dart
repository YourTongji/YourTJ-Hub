/// Markdown 数学公式分隔符：`$…$`、`$$…$$`、`\(…\)`、`\[…\]` 与常用
/// `\begin{env}…\end{env}`。与服务端 `markdown2html/math_protection.go`、Web
/// `runtime/math-segments.ts` 同一规则，共用
/// `packages/api-contract/fixtures/markdown-math-segments.json`。
library;

import 'package:markdown/markdown.dart' as md;

/// 单个公式片段：[text] 交给渲染器（环境保留 begin/end），[start]/[end] 为原文中
/// 含分隔符的范围（UTF-16 下标）。
class MathSegment {
  const MathSegment({
    required this.text,
    required this.display,
    required this.start,
    required this.end,
  });

  final String text;
  final bool display;
  final int start;
  final int end;
}

class _Delimiter {
  const _Delimiter(this.open, this.close, this.display, {this.single = false});

  final String open;
  final String close;
  final bool display;
  final bool single;
}

const List<String> _environments = <String>[
  'equation',
  'equation*',
  'align',
  'align*',
  'aligned',
  'gather',
  'gather*',
  'split',
  'cases',
  'matrix',
  'pmatrix',
  'bmatrix',
  'vmatrix',
  'Vmatrix',
];

final List<_Delimiter> _delimiters = <_Delimiter>[
  const _Delimiter(r'$$', r'$$', true),
  const _Delimiter(r'\[', r'\]', true),
  for (final String environment in _environments)
    _Delimiter('\\begin{$environment}', '\\end{$environment}', true),
  const _Delimiter(r'$', r'$', false, single: true),
  const _Delimiter(r'\(', r'\)', false),
];

bool _escaped(String source, int index) {
  var backslashes = 0;
  for (var i = index - 1; i >= 0 && source.codeUnitAt(i) == 0x5C; i--) {
    backslashes++;
  }
  return backslashes.isOdd;
}

bool _whitespace(String source, int index) {
  if (index < 0 || index >= source.length) return false;
  return const <int>{
    0x20,
    0x09,
    0x0A,
    0x0D,
    0x0C,
    0x0B,
  }.contains(source.codeUnitAt(index));
}

/// 跟踪 `{}` 层级、跳过反斜杠转义，找闭合分隔符（KaTeX auto-render 的 findEndOfMath）。
int _closing(String source, String delimiter, int start) {
  var level = 0;
  var index = start;
  while (index < source.length) {
    if (level <= 0 && source.startsWith(delimiter, index)) return index;
    switch (source.codeUnitAt(index)) {
      case 0x5C:
        index++;
      case 0x7B:
        level++;
      case 0x7D:
        level--;
    }
    index++;
  }
  return -1;
}

/// 行内 `$` 的闭合：紧邻空白的 `$` 不能闭合，此时整个候选不算公式（价格、shell 变量）。
int _inlineClosing(String source, int start) {
  var level = 0;
  var index = start;
  while (index < source.length) {
    final int char = source.codeUnitAt(index);
    if (level <= 0 && char == 0x24) {
      return index > start && !_whitespace(source, index - 1) ? index : -1;
    }
    switch (char) {
      case 0x5C:
        index++;
      case 0x7B:
        level++;
      case 0x7D:
        level--;
    }
    index++;
  }
  return -1;
}

(int, _Delimiter)? _next(String source, int start) {
  (int, _Delimiter)? best;
  for (final _Delimiter delimiter in _delimiters) {
    var index = source.indexOf(delimiter.open, start);
    while (index != -1 && _escaped(source, index)) {
      index = source.indexOf(delimiter.open, index + 1);
    }
    if (index != -1 && (best == null || index < best.$1)) {
      best = (index, delimiter);
    }
  }
  return best;
}

List<MathSegment> extractMathSegments(String source) {
  final List<MathSegment> segments = <MathSegment>[];
  var index = 0;
  while (index < source.length) {
    final (int, _Delimiter)? found = _next(source, index);
    if (found == null) break;
    final (int open, _Delimiter delimiter) = found;
    final int openEnd = open + delimiter.open.length;

    if (delimiter.single) {
      if (_whitespace(source, open + 1)) {
        index = open + 1;
        continue;
      }
      final int close = _inlineClosing(source, openEnd);
      if (close != -1) {
        final String text = source.substring(openEnd, close);
        if (text.isNotEmpty && !text.contains('\n')) {
          segments.add(
            MathSegment(
              text: text,
              display: false,
              start: open,
              end: close + 1,
            ),
          );
          index = close + 1;
          continue;
        }
      }
      index = open + 1;
      continue;
    }

    final int close = _closing(source, delimiter.close, openEnd);
    if (close != -1) {
      final int end = close + delimiter.close.length;
      final String inner = source.substring(openEnd, close);
      if (inner.trim().isNotEmpty) {
        segments.add(
          MathSegment(
            text: delimiter.open.startsWith(r'\begin{')
                ? source.substring(open, end)
                : inner,
            display: delimiter.display,
            start: open,
            end: end,
          ),
        );
        index = end;
        continue;
      }
    }
    index = openEnd;
  }
  return segments;
}

/// 文字与公式交替的片段；[math] 为 null 时是普通文字。
class MathPiece {
  const MathPiece.text(this.text) : math = null;
  const MathPiece.math(MathSegment this.math, this.text);

  /// 普通文字，或公式含分隔符的原文（渲染失败时原样显示）。
  final String text;
  final MathSegment? math;
}

/// 渲染 Markdown 前把公式换成纯字母数字占位，免得 `_`、`*`、`\`、`--` 被当成
/// Markdown 语法改写（服务端 `protectMathSegments` 同理）；渲染时再按占位拆回。
class ProtectedMath {
  ProtectedMath._(this.source, this._prefix, this._segments, this._originals);

  factory ProtectedMath.of(String source) {
    final List<MathSegment> segments = extractMathSegments(source);
    if (segments.isEmpty) {
      return ProtectedMath._(
        source,
        '',
        const <MathSegment>[],
        const <String>[],
      );
    }
    var prefix = 'GFMATH';
    while (source.contains(prefix)) {
      prefix += 'X';
    }
    final StringBuffer buffer = StringBuffer();
    final List<String> originals = <String>[];
    var last = 0;
    for (var i = 0; i < segments.length; i++) {
      buffer
        ..write(source.substring(last, segments[i].start))
        ..write('$prefix${i}Z');
      originals.add(source.substring(segments[i].start, segments[i].end));
      last = segments[i].end;
    }
    buffer.write(source.substring(last));
    return ProtectedMath._(buffer.toString(), prefix, segments, originals);
  }

  /// 替换后的 Markdown（没有公式时即原文）。
  final String source;
  final String _prefix;
  final List<MathSegment> _segments;
  final List<String> _originals;

  bool get isEmpty => _segments.isEmpty;

  late final RegExp _token = RegExp('$_prefix(\\d+)Z');

  /// 按占位拆成文字与公式；没有占位时返回 null（调用方保持原渲染路径）。
  List<MathPiece>? split(String text) {
    if (isEmpty || !text.contains(_prefix)) return null;
    final List<MathPiece> pieces = <MathPiece>[];
    var last = 0;
    for (final RegExpMatch match in _token.allMatches(text)) {
      final int index = int.parse(match.group(1)!);
      if (index >= _segments.length) continue;
      if (match.start > last) {
        pieces.add(MathPiece.text(text.substring(last, match.start)));
      }
      pieces.add(MathPiece.math(_segments[index], _originals[index]));
      last = match.end;
    }
    if (last < text.length) pieces.add(MathPiece.text(text.substring(last)));
    return pieces;
  }

  /// 代码、链接地址等需要原文的地方：占位还原为含分隔符的原文。
  String restore(String text) {
    if (isEmpty || !text.contains(_prefix)) return text;
    return text.replaceAllMapped(_token, (Match match) {
      final int index = int.parse(match.group(1)!);
      return index < _originals.length ? _originals[index] : match.group(0)!;
    });
  }
}

/// Markdown 文本节点的文字；其余节点返回 null（给只依赖 markdown_widget 的渲染层用）。
String? markdownTextOf(md.Node node) => node is md.Text ? node.text : null;
