import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Table viewport for the Markdown renderer.
///
/// A table keeps its natural (intrinsic) column widths and scrolls inside its
/// own region, so a wide table never widens the page. Narrow tables still fill
/// the content width. Scrolling lives here — not on the page — so the outer
/// `NeverScrollableScrollPhysics` body stays fixed and no ancestor clip can
/// swallow the gesture.
class GfTableViewport extends StatelessWidget {
  const GfTableViewport({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (BuildContext context, BoxConstraints constraints) =>
        constraints.maxWidth.isFinite
        ? GfHorizontalScrollView(
            child: ConstrainedBox(
              constraints: BoxConstraints(minWidth: constraints.maxWidth),
              child: child,
            ),
          )
        // Unbounded width means an ancestor already scrolls horizontally;
        // there is nothing to size against, so the table passes through.
        : child,
  );
}

/// Formats a [Color] as CSS (`#rrggbb` / `#rrggbbaa`).
///
/// flutter_widget_from_html follows CSS4 for 8-digit hex values, so the alpha
/// channel must come last — an `#aarrggbb` string shifts every channel.
String gfCssColor(Color color) {
  final int value = color.toARGB32();
  final int alpha = (value >> 24) & 0xFF;
  final String rgb = (value & 0xFFFFFF).toRadixString(16).padLeft(6, '0');
  return alpha == 0xFF
      ? '#$rgb'
      : '#$rgb${alpha.toRadixString(16).padLeft(2, '0')}';
}

/// Shared styles for a server-rendered `<table>`.
///
/// `max-width: 100%` is load-bearing: flutter_widget_from_html only wraps a
/// table in its own horizontal scroll view when the resolved max width is
/// finite (`tag_table.dart`), which is exactly the "wide table scrolls in
/// place, page stays fixed" behaviour the rich-content policy requires.
Map<String, String> gfHtmlTableStyles({
  required GfRichContentTypography profile,
  required GfColors colors,
  required GfBorders borders,
}) => <String, String>{
  'max-width': '100%',
  'border-collapse': 'collapse',
  'border': '${borders.width}px solid ${gfCssColor(colors.line)}',
  'margin': '${profile.blockSpacing}px 0',
};

/// Shared styles for server-rendered `<th>` / `<td>`.
///
/// Alignment is intentionally left alone: goldmark emits GFM alignment as an
/// inline `text-align` style, which the HTML renderer already honours.
Map<String, String> gfHtmlTableCellStyles({
  required GfRichContentTypography profile,
  required GfColors colors,
  required GfBorders borders,
  required bool header,
}) {
  final EdgeInsets padding = profile.tableCellPadding;
  return <String, String>{
    'padding': '${padding.top}px ${padding.right}px '
        '${padding.bottom}px ${padding.left}px',
    'border': '${borders.width}px solid ${gfCssColor(colors.line)}',
    'color': gfCssColor(colors.baseContent),
    'font-size':
        '${header ? profile.tableHeader.fontSize : profile.tableBody.fontSize}px',
    'font-weight': header ? '600' : '400',
    'background-color': header
        ? gfCssColor(colors.base200)
        : gfCssColor(colors.base100),
  };
}
