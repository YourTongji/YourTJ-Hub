import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:markdown_widget/markdown_widget.dart' show MarkdownConfig, MarkdownWidget;
import 'package:forum_app/src/reading_preferences.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';
import 'package:forum_app/src/widgets/rich_content/gf_code_block.dart';
import 'package:forum_app/src/widgets/rich_content/gf_html_content.dart';
import 'package:forum_app/src/widgets/rich_content/gf_table.dart';
import 'package:ui_kit/ui_kit.dart';

import 'schedule_time_grid_test.dart' show UnevenTextScaler;

/// Which rich-content renderer a case exercises.
enum _Path {
  postMarkdown('post markdown'),
  wikiHtml('wiki html'),
  reviewHtml('course review html');

  const _Path(this.label);

  final String label;
}

const String _longUrl =
    'https://example.com/very/long/path/that/keeps/going/and/going/'
    'with/query/and/more?token=abcdefghijklmnopqrstuvwxyz0123456789';

const String _longCodeLine =
    'final veryLongIdentifierNameThatNeverWraps = '
    "'https://example.com/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';";

const String _markdownFixture =
    '''
# 一级标题 heading

## 二级标题 heading

### 三级标题 heading

#### 四级标题 heading

##### 五级标题 heading

###### 六级标题 heading

正文段落一，包含**粗体**、*斜体* 与 `inline code`，以及一个很长的链接 $_longUrl。

正文段落二，连续段落用于确认段间距与自动换行在窄屏下仍然自然。

> 引用块第一行
> 引用块第二行

- 列表项一
  - 嵌套列表项
- 列表项二

1. 有序项一
2. 有序项二

```dart
final int answer = 42;
final String label = 'answer';
```

```python
def solve():
    return 42
```

```notalanguage
plain text body without highlighting
```

```
$_longCodeLine
```

| 左对齐 | 居中 | 右对齐 | 第四列 | 第五列 | 第六列 | 第七列 | 第八列 |
| :----- | :--: | -----: | ------ | ------ | ------ | ------ | ------ |
| 甲 | 中 | 右 | 1111111111 | 2222222222 | 3333333333 | 4444444444 | 5555555555 |

![图片](https://example.com/image.png)
''';

const String _htmlFixture = '''
<h1 id="h1">一级标题 heading</h1>
<h2 id="h2">二级标题 heading</h2>
<h3 id="h3">三级标题 heading</h3>
<h4 id="h4">四级标题 heading</h4>
<h5 id="h5">五级标题 heading</h5>
<h6 id="h6">六级标题 heading</h6>
<p>正文段落一，包含<strong>粗体</strong>、<em>斜体</em> 与 <code>inline code</code>，
以及一个很长的链接 <a href="$_longUrl">$_longUrl</a>。</p>
<p>正文段落二，连续段落用于确认段间距与自动换行在窄屏下仍然自然。</p>
<blockquote><p>引用块第一行<br>引用块第二行</p></blockquote>
<ul><li>列表项一<ul><li>嵌套列表项</li></ul></li><li>列表项二</li></ul>
<ol><li>有序项一</li><li>有序项二</li></ol>
<pre><code class="language-dart">final int answer = 42;</code></pre>
<pre><code class="language-python">def solve():
    return 42</code></pre>
<pre><code class="language-notalanguage">plain text body</code></pre>
<pre><code>$_longCodeLine</code></pre>
<table><thead><tr><th>左对齐</th><th>居中</th><th>右对齐</th><th>第四列</th><th>第五列</th>
<th>第六列</th><th>第七列</th><th>第八列</th></tr></thead>
<tbody><tr><td style="text-align:left">甲</td><td style="text-align:center">中</td>
<td style="text-align:right">右</td><td>1111111111</td><td>2222222222</td><td>3333333333</td>
<td>4444444444</td><td>5555555555</td></tr></tbody></table>
<p><img src="https://example.com/image.png" alt="图片"></p>
''';

const List<double> _widths = <double>[320, 360, 390, 600, 840];
const List<double> _scales = <double>[1, 1.3, 2];

/// Fixed reader preference so the Markdown renderer resolves the same profile
/// the assertions expect, without touching SharedPreferences.
class _FixedContentFontScale extends ContentFontScaleNotifier {
  _FixedContentFontScale(this.scale);

  final double scale;

  @override
  double build() => scale;
}

GfRichContentTypography _profileFor(_Path path, double userScale) =>
    GfRichContentTypography.standard(
      typography: GfTypography.standard(GfColors.light.baseContent),
      colors: GfColors.light,
      userScale: userScale,
      bodySize: path == _Path.reviewHtml
          ? GfRichContentTypography.compactBodySize
          : GfRichContentTypography.readingBodySize,
    );

Widget _contentFor(_Path path, double userScale) => switch (path) {
  _Path.postMarkdown => const GfMarkdownView(data: _markdownFixture),
  // Wiki keeps its own WidgetFactory/anchors; both HTML surfaces share the
  // same shell, so exercising one covers the shared policy.
  _Path.wikiHtml => GfHtmlContent(
    html: _htmlFixture,
    profile: _profileFor(path, userScale),
  ),
  _Path.reviewHtml => GfHtmlContent(
    html: _htmlFixture,
    profile: _profileFor(path, userScale),
  ),
};

Future<void> _pumpFixture(
  WidgetTester tester, {
  required _Path path,
  required double width,
  required double userScale,
  required Brightness brightness,
  double systemScale = 1,
  TextScaler? textScaler,
}) async {
  tester.view.physicalSize = Size(width, 1200);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    ProviderScope(
      overrides: <Override>[
        contentFontScaleProvider.overrideWith(
          () => _FixedContentFontScale(userScale),
        ),
      ],
      child: MaterialApp(
        theme: gfThemeData(brightness),
        builder: (BuildContext context, Widget? child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            textScaler: textScaler ?? TextScaler.linear(systemScale),
          ),
          child: child!,
        ),
        home: Scaffold(
          body: SingleChildScrollView(
            key: const ValueKey('rich-content-page'),
            child: _contentFor(path, userScale),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

/// Disposes the tree and drains the detectors' pending timers, the same way
/// the existing Markdown tests do.
Future<void> _dispose(WidgetTester tester) async {
  await tester.pumpWidget(const SizedBox.shrink());
  await tester.pump(const Duration(milliseconds: 600));
}

/// Horizontal scrollables that are *not* inside a code block or a table would
/// mean the page itself can be dragged sideways.
void _expectOnlyRichContentScrolls({required bool html}) {
  for (final Element element in find.byType(Scrollable).evaluate()) {
    final Scrollable scrollable = element.widget as Scrollable;
    if (scrollable.axis != Axis.horizontal) continue;
    final Finder target = find.byWidget(element.widget);
    final bool inCode = find
        .ancestor(of: target, matching: find.byType(GfCodeBlock))
        .evaluate()
        .isNotEmpty;
    // Markdown wraps the table (ancestor), the HTML renderer wraps it from
    // outside too, so the table is found below the scrollable.
    final bool inTable =
        find
            .ancestor(of: target, matching: find.byType(GfTableViewport))
            .evaluate()
            .isNotEmpty ||
        find
            .descendant(of: target, matching: find.byType(HtmlTable))
            .evaluate()
            .isNotEmpty;
    expect(
      inCode || inTable,
      isTrue,
      reason: '${html ? 'html' : 'markdown'} horizontal scroll must live in '
          'a code block or a table',
    );
  }
}

void main() {
  for (final _Path path in _Path.values) {
    for (final double width in _widths) {
      for (final double scale in _scales) {
        for (final Brightness brightness in Brightness.values) {
          testWidgets(
            '${path.label} ${width.toInt()}px x$scale ${brightness.name} '
            'has no page overflow',
            (tester) async {
              await _pumpFixture(
                tester,
                path: path,
                width: width,
                userScale: 1,
                systemScale: scale,
                brightness: brightness,
              );

              expect(tester.takeException(), isNull);
              expect(
                tester.getSize(find.byKey(const ValueKey('rich-content-page'))).width,
                lessThanOrEqualTo(width),
              );
              // Non-vacuous: the fixture really rendered code and a table, and
              // the code block always owns a horizontal scroll view.
              expect(find.byType(GfCodeBlock), findsWidgets);
              expect(
                find.byType(GfTableViewport).evaluate().isNotEmpty ||
                    find.byType(HtmlTable).evaluate().isNotEmpty,
                isTrue,
              );
              expect(
                find
                    .descendant(
                      of: find.byType(GfCodeBlock).first,
                      matching: find.byType(Scrollable),
                    )
                    .evaluate()
                    .isNotEmpty,
                isTrue,
              );
              _expectOnlyRichContentScrolls(html: path != _Path.postMarkdown);
              await _dispose(tester);
            },
          );
        }
      }
    }
  }

  testWidgets('body text follows the profile and the reader preference', (
    tester,
  ) async {
    RichText paragraphOf(WidgetTester tester) => tester.widget<RichText>(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText &&
            widget.text.toPlainText().startsWith('正文段落一'),
      ),
    );

    for (final double readerScale in <double>[1, 1.4]) {
      await _pumpFixture(
        tester,
        path: _Path.postMarkdown,
        width: 390,
        userScale: readerScale,
        brightness: Brightness.light,
      );
      final MarkdownConfig config = tester
          .widget<MarkdownWidget>(find.byType(MarkdownWidget))
          .config!;
      // The framework applies the system scaler while painting, so the
      // configured size is design baseline × reader preference.
      expect(
        config.p.textStyle.fontSize,
        closeTo(17 * readerScale, .001),
        reason: 'markdown body at reader scale $readerScale',
      );
      await _dispose(tester);

      await _pumpFixture(
        tester,
        path: _Path.wikiHtml,
        width: 390,
        userScale: readerScale,
        systemScale: 1.3,
        brightness: Brightness.light,
      );
      // flutter_widget_from_html folds the system factor into the style
      // itself; the painted size is still scaled exactly once (asserted by the
      // dedicated scaling test below).
      expect(
        paragraphOf(tester).text.style?.fontSize,
        closeTo(17 * readerScale * 1.3, .001),
        reason: 'html body at reader scale $readerScale',
      );
      await _dispose(tester);

      await _pumpFixture(
        tester,
        path: _Path.reviewHtml,
        width: 390,
        userScale: readerScale,
        brightness: Brightness.light,
      );
      expect(
        paragraphOf(tester).text.style?.fontSize,
        closeTo(GfRichContentTypography.compactBodySize * readerScale, .001),
        reason: 'compact body at reader scale $readerScale',
      );
      await _dispose(tester);
    }
  });

  testWidgets('server HTML text is scaled once, never twice', (tester) async {
    Future<double> textWidth(double systemScale) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: gfThemeData(Brightness.light),
          builder: (BuildContext context, Widget? child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(systemScale)),
            child: child!,
          ),
          home: Scaffold(
            // A min-size row gives the rendered text its intrinsic width, so
            // the width ratio exposes whether scaling is applied twice.
            body: Row(
              mainAxisSize: MainAxisSize.min,
              children: <Widget>[
                GfHtmlContent(
                  html: '<p>MMMM</p>',
                  profile: _profileFor(_Path.wikiHtml, 1),
                ),
              ],
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final double width = tester
          .getSize(
            find.byWidgetPredicate(
              (Widget widget) =>
                  widget is RichText &&
                  widget.text.toPlainText().contains('MMMM'),
            ),
          )
          .width;
      await _dispose(tester);
      return width;
    }

    final double single = await textWidth(1);
    final double scaled = await textWidth(1.5);
    expect(scaled / single, closeTo(1.5, .05));
  });

  testWidgets('long code lines scroll inside the block, not the page', (
    tester,
  ) async {
    for (final _Path path in <_Path>[_Path.postMarkdown, _Path.wikiHtml]) {
      await _pumpFixture(
        tester,
        path: path,
        width: 320,
        userScale: 1,
        brightness: Brightness.light,
      );
      expect(tester.takeException(), isNull);

      final Finder block = find.byType(GfCodeBlock).first;
      final Finder scrollable = find.descendant(
        of: block,
        matching: find.byType(Scrollable),
      );
      final ScrollableState state = tester.state<ScrollableState>(scrollable);
      expect(
        state.position.maxScrollExtent,
        greaterThan(0),
        reason: '${path.label} code block scrolls horizontally',
      );

      await tester.drag(scrollable, const Offset(-120, 0));
      await tester.pumpAndSettle();
      expect(state.position.pixels, greaterThan(0));
      expect(
        tester.getSize(find.byKey(const ValueKey('rich-content-page'))).width,
        lessThanOrEqualTo(320),
      );
      await _dispose(tester);
    }
  });

  testWidgets('wide tables scroll inside their own region', (tester) async {
    for (final _Path path in <_Path>[_Path.postMarkdown, _Path.wikiHtml]) {
      await _pumpFixture(
        tester,
        path: path,
        width: 320,
        userScale: 1,
        brightness: Brightness.light,
      );
      expect(tester.takeException(), isNull);

      final Finder tableArea = path == _Path.postMarkdown
          ? find.byType(GfTableViewport)
          : find.byType(HtmlTable);
      expect(tableArea, findsOneWidget);
      // Markdown wraps the table in our viewport; the HTML renderer wraps it
      // from outside, so the scroll view is an ancestor there.
      final Finder scrollable = path == _Path.postMarkdown
          ? find.descendant(
              of: tableArea,
              matching: find.byType(Scrollable),
            )
          : find.ancestor(of: tableArea, matching: find.byType(Scrollable));
      expect(scrollable, findsWidgets);
      expect(
        tester
            .state<ScrollableState>(scrollable.first)
            .position
            .maxScrollExtent,
        greaterThan(0),
        reason: '${path.label} table scrolls horizontally',
      );
      expect(
        tester.getSize(find.byKey(const ValueKey('rich-content-page'))).width,
        lessThanOrEqualTo(320),
      );
      await _dispose(tester);
    }
  });

  testWidgets('nonlinear system text scaling still lays out and reflows', (
    tester,
  ) async {
    for (final _Path path in _Path.values) {
      await _pumpFixture(
        tester,
        path: path,
        width: 320,
        userScale: 1,
        brightness: Brightness.dark,
        textScaler: const UnevenTextScaler(),
      );
      expect(tester.takeException(), isNull);
      _expectOnlyRichContentScrolls(html: path != _Path.postMarkdown);
      await _dispose(tester);
    }
  });

  testWidgets('reader preference never replaces system font scaling', (
    tester,
  ) async {
    await _pumpFixture(
      tester,
      path: _Path.postMarkdown,
      width: 390,
      userScale: 1.4,
      systemScale: 1.3,
      brightness: Brightness.light,
    );
    final MarkdownConfig config = tester
        .widget<MarkdownWidget>(find.byType(MarkdownWidget))
        .config!;
    // Design baseline × reader preference; the system scaler is separate and
    // still applied by the framework when painting.
    expect(config.p.textStyle.fontSize, closeTo(17 * 1.4, .001));
    expect(
      MediaQuery.textScalerOf(
        tester.element(find.byType(MarkdownWidget)),
      ).scale(17),
      closeTo(17 * 1.3, .001),
    );
    await _dispose(tester);
  });

  test('the app never disables or pins system text scaling', () {
    final List<String> offenders = <String>[];
    for (final FileSystemEntity entity in Directory('lib').listSync(
      recursive: true,
    )) {
      if (entity is! File || !entity.path.endsWith('.dart')) continue;
      final String source = entity.readAsStringSync();
      if (source.contains('TextScaler.noScaling') ||
          RegExp(r'textScaleFactor\s*:').hasMatch(source) ||
          RegExp(r'textScaleFactor\s*=').hasMatch(source)) {
        offenders.add(entity.path);
      }
    }
    expect(offenders, isEmpty);
  });

  testWidgets('GFM table alignment is honoured in both renderers', (
    tester,
  ) async {
    Finder cell(String text) => find.byWidgetPredicate(
      (Widget widget) =>
          widget is RichText && widget.text.toPlainText() == text,
    );

    await _pumpFixture(
      tester,
      path: _Path.postMarkdown,
      width: 840,
      userScale: 1,
      brightness: Brightness.light,
    );
    expect(
      tester
          .widget<Align>(
            find.ancestor(of: cell('中'), matching: find.byType(Align)).first,
          )
          .alignment,
      Alignment.center,
    );
    expect(
      tester
          .widget<Align>(
            find
                .ancestor(of: cell('右'), matching: find.byType(Align))
                .first,
          )
          .alignment,
      Alignment.centerRight,
    );
    await _dispose(tester);

    await _pumpFixture(
      tester,
      path: _Path.reviewHtml,
      width: 840,
      userScale: 1,
      brightness: Brightness.light,
    );
    expect(tester.widget<RichText>(cell('中')).textAlign, TextAlign.center);
    expect(
      tester.widget<RichText>(cell('右')).textAlign,
      TextAlign.right,
    );
    await _dispose(tester);
  });

  testWidgets('server headings in course reviews use the compact profile', (
    tester,
  ) async {
    final GfRichContentTypography profile = _profileFor(_Path.reviewHtml, 1);
    await _pumpFixture(
      tester,
      path: _Path.reviewHtml,
      width: 390,
      userScale: 1,
      brightness: Brightness.light,
    );
    final RichText h2 = tester.widget<RichText>(
      find.byWidgetPredicate(
        (Widget widget) =>
            widget is RichText &&
            widget.text.toPlainText().startsWith('二级标题'),
      ),
    );
    expect(h2.text.style?.fontSize, closeTo(profile.h2.fontSize!, .001));
    expect(
      h2.text.style?.fontSize,
      greaterThan(profile.body.fontSize!),
    );
    await _dispose(tester);
  });

  testWidgets('system text 2.0 reflows instead of clipping', (tester) async {
    Future<double> contentHeight(_Path path, double systemScale) async {
      await _pumpFixture(
        tester,
        path: path,
        width: 320,
        userScale: 1,
        systemScale: systemScale,
        brightness: Brightness.light,
      );
      final double height = tester.getSize(
        switch (path) {
          _Path.postMarkdown => find.byType(GfMarkdownView),
          _ => find.byType(GfHtmlContent),
        },
      ).height;
      await _dispose(tester);
      return height;
    }

    for (final _Path path in _Path.values) {
      final double single = await contentHeight(path, 1);
      final double doubled = await contentHeight(path, 2);
      expect(
        doubled,
        greaterThan(single * 1.5),
        reason: '${path.label} body must grow with system text',
      );
    }
  });

  testWidgets('code block survives an unbounded-width host', (tester) async {
    tester.view.physicalSize = const Size(390, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      MaterialApp(
        theme: gfThemeData(Brightness.light),
        home: Scaffold(
          body: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: GfCodeBlock(
              profile: _profileFor(_Path.wikiHtml, 1),
              code: 'final int answer = 42;',
              language: 'dart',
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.byType(GfCodeBlock), findsOneWidget);
    await _dispose(tester);
  });
}
