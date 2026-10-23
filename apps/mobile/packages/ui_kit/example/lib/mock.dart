import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Initial scroll offset requested by a deep link (`?y=640`), so long pages
/// can be captured in several frames.
double galleryInitialScroll = 0;

String avatar(int n) => 'https://i.pravatar.cc/240?img=$n';
String photo(String seed, {int w = 1200, int h = 800}) =>
    'https://picsum.photos/seed/$seed/$w/$h';

const Color catCampus = Color(0xFF2563EB);
const Color catStudy = Color(0xFF00A63E);
const Color catTrade = Color(0xFFFE9A00);
const Color catHelp = Color(0xFFE11D48);

/// Scrollable page with a navigation bar, used by every catalog section.
class CatalogScaffold extends StatefulWidget {
  const CatalogScaffold({
    super.key,
    required this.title,
    required this.children,
    this.grouped = false,
  });

  final String title;
  final List<Widget> children;
  final bool grouped;

  @override
  State<CatalogScaffold> createState() => _CatalogScaffoldState();
}

class _CatalogScaffoldState extends State<CatalogScaffold> {
  late final ScrollController _scroll = ScrollController(
    initialScrollOffset: galleryInitialScroll,
  );

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final bg = widget.grouped ? colors.canvasGrouped : colors.canvas;
    return Scaffold(
      backgroundColor: bg,
      appBar: GfAppBar(
        title: GfText(widget.title),
        backgroundColor: bg,
        showDivider: !widget.grouped,
      ),
      body: ListView(
        controller: _scroll,
        padding: const EdgeInsets.only(bottom: 48),
        children: widget.children,
      ),
    );
  }
}

/// Labelled specimen block inside a catalog page.
class Specimen extends StatelessWidget {
  const Specimen({
    super.key,
    required this.title,
    required this.child,
    this.note,
    this.padded = true,
  });

  final String title;
  final String? note;
  final Widget child;
  final bool padded;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final typography = GfTheme.typographyOf(context);
    return Padding(
      padding: const EdgeInsets.only(top: GfSpace.xxl),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: GfSpace.gutter),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                GfText(title, style: typography.heading),
                if (note != null) ...[
                  const SizedBox(width: GfSpace.sm),
                  Expanded(
                    child: GfText(
                      note!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: typography.caption.copyWith(
                        color: colors.textSecondary,
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: GfSpace.md),
          if (padded)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: GfSpace.gutter),
              child: child,
            )
          else
            child,
        ],
      ),
    );
  }
}

/// Grouped-canvas band for specimens that live on inset groups.
class GroupedBand extends StatelessWidget {
  const GroupedBand({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) => ColoredBox(
    color: GfTheme.colorsOf(context).canvasGrouped,
    child: Padding(
      padding: const EdgeInsets.symmetric(vertical: GfSpace.lg),
      child: child,
    ),
  );
}

/// Small caption under a specimen.
class Caption extends StatelessWidget {
  const Caption(this.text, {super.key, this.align = TextAlign.center});
  final String text;
  final TextAlign align;

  @override
  Widget build(BuildContext context) => GfText(
    text,
    textAlign: align,
    style: GfTheme.typographyOf(
      context,
    ).meta.copyWith(color: GfTheme.colorsOf(context).textSecondary),
  );
}
