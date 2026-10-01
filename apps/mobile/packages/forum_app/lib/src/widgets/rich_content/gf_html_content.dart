import 'package:flutter/material.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:ui_kit/ui_kit.dart';

import 'gf_code_block.dart';
import 'gf_table.dart';

/// Shared shell for server-rendered rich content (Wiki pages, course reviews).
///
/// The data source stays what it is — goldmark HTML rendered by `HtmlWidget`,
/// which keeps heading ids, TOC anchors and relative URLs working — but the
/// typography, code policy and table policy come from the shared
/// [GfRichContentTypography] profile and the shared code/table widgets, so the
/// HTML surfaces cannot drift from the Markdown post renderer.
///
/// Callers keep their own hooks: [factoryBuilder] for anchor semantics,
/// [customWidgetBuilder] for extra element overrides (images) and
/// [customStylesBuilder] for surface-specific styles. Those run *after* the
/// shared policy, except that `<pre>` is always owned by the shared code block.
class GfHtmlContent extends StatelessWidget {
  const GfHtmlContent({
    super.key,
    required this.html,
    required this.profile,
    this.htmlKey,
    this.baseUrl,
    this.factoryBuilder,
    this.customWidgetBuilder,
    this.customStylesBuilder,
    this.onTapUrl,
    this.buildAsync = false,
  });

  final String html;
  final GfRichContentTypography profile;

  /// Forwarded to the underlying `HtmlWidget` so callers keep anchor scrolling.
  final Key? htmlKey;

  /// Forwarded to the underlying `HtmlWidget`; relative and scheme-relative
  /// links resolve against it (Wiki keeps its server-side URL semantics).
  final Uri? baseUrl;

  final WidgetFactory Function()? factoryBuilder;
  final CustomWidgetBuilder? customWidgetBuilder;
  final CustomStylesBuilder? customStylesBuilder;
  final Future<bool> Function(String url)? onTapUrl;
  final bool buildAsync;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final CustomStylesBuilder shared = gfHtmlContentStyles(
      profile: profile,
      colors: colors,
      borders: borders,
    );
    final CustomWidgetBuilder sharedWidgets = gfHtmlContentWidgets(
      profile: profile,
    );

    return HtmlWidget(
      html,
      key: htmlKey,
      baseUrl: baseUrl,
      // The default factory keeps the shared policy for the elements the HTML
      // renderer builds itself (the table scroll wrapper).
      factoryBuilder: factoryBuilder ?? GfHtmlWidgetFactory.new,
      buildAsync: buildAsync,
      textStyle: profile.body,
      onTapUrl: onTapUrl,
      customStylesBuilder: (element) => <String, String>{
        ...?shared(element),
        ...?customStylesBuilder?.call(element),
      },
      customWidgetBuilder: (element) =>
          sharedWidgets(element) ?? customWidgetBuilder?.call(element),
    );
  }
}

/// Widget factory that applies the shared rich-content policy to the elements
/// the HTML renderer builds itself.
///
/// `<pre>` is replaced by [GfCodeBlock] before this matters; the remaining
/// built-in scroll view is the table wrapper, which gets the same viewport and
/// visible scroll cue the Markdown path shows.
class GfHtmlWidgetFactory extends WidgetFactory {
  @override
  Widget? buildHorizontalScrollView(BuildTree tree, Widget child) =>
      GfHorizontalScrollView(child: child);
}

/// Shared inline styles for server-rendered rich content.
CustomStylesBuilder gfHtmlContentStyles({
  required GfRichContentTypography profile,
  required GfColors colors,
  required GfBorders borders,
}) => (element) {
  final String line = gfCssColor(colors.line);
  switch (element.localName) {
    case 'p':
      return <String, String>{
        'margin': '0 0 ${profile.paragraphSpacing}px',
      };
    case 'h1':
    case 'h2':
    case 'h3':
    case 'h4':
    case 'h5':
    case 'h6':
      final TextStyle heading = switch (element.localName) {
        'h1' => profile.h1,
        'h2' => profile.h2,
        'h3' => profile.h3,
        'h4' => profile.h4,
        'h5' => profile.h5,
        _ => profile.h6,
      };
      return <String, String>{
        'font-size': '${heading.fontSize}px',
        'font-weight': '${heading.fontWeight?.value ?? 400}',
        'line-height': '${heading.height}',
        'color': gfCssColor(heading.color ?? colors.baseContent),
        'margin': '${profile.blockSpacing * 2}px 0 '
            '${profile.blockSpacing}px',
      };
    case 'strong':
      // Web and the Markdown renderer both use the platform's bold weight;
      // keeping 700 here avoids a per-renderer emphasis drift.
      return <String, String>{'font-weight': '700'};
    case 'em':
      return <String, String>{'font-style': 'italic'};
    case 'code':
      return <String, String>{
        'font-size': '${profile.inlineCode.fontSize}px',
        'font-family': GfRichContentTypography.codeFontFamily,
        'color': gfCssColor(profile.inlineCode.color ?? colors.error),
        'background-color': gfCssColor(colors.base200),
        'padding': '1px 4px',
        'border-radius': '4px',
      };
    case 'blockquote':
      return <String, String>{
        'border-left': '${profile.quoteSide}px solid $line',
        'background-color': gfCssColor(colors.base200.withValues(alpha: .7)),
        'color': gfCssColor(profile.quote.color ?? colors.baseContent),
        'padding': '${profile.quotePadding.top}px '
            '${profile.quotePadding.right}px '
            '${profile.quotePadding.bottom}px '
            '${profile.quotePadding.left}px',
        'margin': '${profile.blockSpacing}px 0',
      };
    case 'table':
      return gfHtmlTableStyles(
        profile: profile,
        colors: colors,
        borders: borders,
      );
    case 'th':
      return gfHtmlTableCellStyles(
        profile: profile,
        colors: colors,
        borders: borders,
        header: true,
      );
    case 'td':
      return gfHtmlTableCellStyles(
        profile: profile,
        colors: colors,
        borders: borders,
        header: false,
      );
    case 'ul':
    case 'ol':
      return <String, String>{
        'padding-left': '${profile.listIndent}px',
        'margin': '${profile.blockSpacing}px 0',
      };
    case 'li':
      return <String, String>{'margin-bottom': '${profile.listSpacing}px'};
    case 'a':
      return <String, String>{'color': gfCssColor(colors.primary)};
    case 'hr':
      return <String, String>{
        'border': '0',
        'border-top': '${borders.width}px solid $line',
        'margin': '${profile.blockSpacing * 2}px 0',
      };
    default:
      return const <String, String>{};
  }
};

/// Shared element overrides for server-rendered rich content.
///
/// Only `<pre>` is owned here: it becomes the same [GfCodeBlock] the Markdown
/// path uses, so highlighting, copy and horizontal scrolling cannot drift.
/// Every other element is left to the caller (and then to the renderer).
CustomWidgetBuilder gfHtmlContentWidgets({
  required GfRichContentTypography profile,
}) => (element) {
  if (element.localName != 'pre') return null;
  final code = element.querySelector('code');
  return GfCodeBlock(
    profile: profile,
    code: code?.text ?? element.text,
    language: _languageFromClass(code?.className ?? element.className),
  );
};

/// Reads `language-xxx` / `lang-xxx` from a class attribute.
String _languageFromClass(String className) {
  for (final String name in className.split(' ')) {
    for (final String prefix in const <String>['language-', 'lang-']) {
      if (name.startsWith(prefix) && name.length > prefix.length) {
        return name.substring(prefix.length);
      }
    }
  }
  return '';
}

final RegExp _htmlBlockBreak = RegExp(
  r'<br\s*/?>|</p>|</li>|</h[1-6]>|</blockquote>|</tr>|</div>',
  caseSensitive: false,
);
final RegExp _htmlScriptOrStyle = RegExp(
  r'<(script|style)[^>]*>.*?</\1>',
  caseSensitive: false,
  dotAll: true,
);
final RegExp _htmlTag = RegExp(r'<[^>]+>');
final RegExp _htmlWhitespace = RegExp(r'[ \t\r\n]+');

/// Flattens server-rendered HTML into a single-line preview string.
///
/// Used by truncated surfaces (my-reviews rows, the delete confirmation) so they
/// derive from the same `contentHtml` the full reader shows instead of leaking
/// raw Markdown markers like `## 课程内容`.
///
/// XXX: strip-tags only — block structure, image alt text and code blocks
/// collapse into one line, and only the common entities are decoded. Swap in a
/// DOM walk if a preview ever needs more than readable prose.
String gfPlainTextFromHtml(String html) => html
    .replaceAll(_htmlScriptOrStyle, ' ')
    .replaceAll(_htmlBlockBreak, ' ')
    .replaceAll(_htmlTag, '')
    .replaceAll('&nbsp;', ' ')
    .replaceAll('&amp;', '&')
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&quot;', '"')
    .replaceAll('&#39;', "'")
    .replaceAll(_htmlWhitespace, ' ')
    .trim();
