import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart' show RenderProxyBox;
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:ui_kit/ui_kit.dart';

/// 单个公式的长度上限（与 Web `MAX_MATH_LENGTH` 一致），更长的按原文显示。
const int maxRenderedMathLength = 5000;

/// KaTeX 默认把公式排成正文字号的 1.21 倍（katex.css `.katex { font-size: 1.21em }`）。
const double mathFontScale = 1.21;

final RegExp _tag = RegExp(r'\\tag(\*?)\{([^{}]*)\}');
final RegExp _dropped = RegExp(r'\\label\{[^{}]*\}|\\nonumber\b|\\notag\b');

/// Web 用的 KaTeX 认识、flutter_math 不认识的写法换成等价排版：编号环境去掉外壳，
/// `align`／`split` 用 `aligned`，`gather` 用居中的 `matrix`，`\tag{n}` 显示为右侧的 (n)。
@visibleForTesting
String texForFlutterMath(String tex) {
  var result = tex;
  for (final String name in <String>['equation*', 'equation']) {
    result = result
        .replaceAll('\\begin{$name}', '')
        .replaceAll('\\end{$name}', '');
  }
  for (final (String from, String to) in <(String, String)>[
    ('align*', 'aligned'),
    ('align', 'aligned'),
    ('split', 'aligned'),
    ('gather*', 'matrix'),
    ('gather', 'matrix'),
  ]) {
    result = result
        .replaceAll('\\begin{$from}', '\\begin{$to}')
        .replaceAll('\\end{$from}', '\\end{$to}');
  }
  return result
      .replaceAllMapped(
        _tag,
        (Match match) => match.group(1)!.isEmpty
            ? '\\qquad(${match.group(2)})'
            : '\\qquad ${match.group(2)}',
      )
      .replaceAll(_dropped, '');
}

/// 正文里的一个数学公式（与 Web KaTeX 同一组分隔符，见 core `extractMathSegments`）。
///
/// 行内公式坐在文字行上；独立公式占满一行、靠起始边排版（与 Web 窄屏一致），
/// 太宽时在自身区域内横向滚动。解析失败或过长时原样显示含分隔符的原文 [source]。
class GfMath extends StatelessWidget {
  const GfMath({
    super.key,
    required this.tex,
    required this.display,
    required this.source,
    this.style,
  });

  final String tex;
  final bool display;
  final String source;
  final TextStyle? style;

  @override
  Widget build(BuildContext context) {
    final TextStyle base = DefaultTextStyle.of(context).style.merge(style);
    final Widget fallback = Text(source, style: base);
    if (tex.length > maxRenderedMathLength) return fallback;
    final Widget math = Math.tex(
      texForFlutterMath(tex),
      mathStyle: display ? MathStyle.display : MathStyle.text,
      textStyle: base.copyWith(
        fontSize: (base.fontSize ?? 16) * mathFontScale,
        // 公式不画行内代码等外层底色
        backgroundColor: Colors.transparent,
      ),
      onErrorFallback: (_) => fallback,
    );
    final Widget labelled = Semantics(
      label: source,
      excludeSemantics: true,
      child: _DryBaselineEstimate(child: math),
    );
    if (!display) return labelled;
    return SizedBox(
      width: double.infinity,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: GfHorizontalScrollView(
          child: Padding(
            // 留出横向滚动提示条的位置
            padding: const EdgeInsets.only(bottom: 6),
            child: labelled,
          ),
        ),
      ),
    );
  }
}

/// 表格按内容自然列宽排版时会先“试排”文字行（dry layout），需要行内组件的基线；
/// flutter_math 的行盒不提供试排基线，这里按高度估算（只影响试排，正式排版用真实基线）。
class _DryBaselineEstimate extends SingleChildRenderObjectWidget {
  const _DryBaselineEstimate({required super.child});

  @override
  RenderObject createRenderObject(BuildContext context) =>
      _RenderDryBaselineEstimate();
}

class _RenderDryBaselineEstimate extends RenderProxyBox {
  @override
  double? computeDryBaseline(
    BoxConstraints constraints,
    TextBaseline baseline,
  ) => getDryLayout(constraints).height * 0.8;
}
