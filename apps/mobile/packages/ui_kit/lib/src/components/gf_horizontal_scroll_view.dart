import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';

/// A horizontal scroll view with a quiet, persistent cue when content overflows.
class GfHorizontalScrollView extends StatefulWidget {
  const GfHorizontalScrollView({
    super.key,
    this.scrollViewKey,
    required this.child,
  });

  final Key? scrollViewKey;
  final Widget child;

  @override
  State<GfHorizontalScrollView> createState() => _GfHorizontalScrollViewState();
}

class _GfHorizontalScrollViewState extends State<GfHorizontalScrollView> {
  final ScrollController _controller = ScrollController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ScrollbarTheme(
    data: ScrollbarThemeData(
      thumbColor: WidgetStatePropertyAll<Color?>(
        GfTheme.colorsOf(context).iconMuted.withValues(alpha: .34),
      ),
    ),
    child: Scrollbar(
      controller: _controller,
      thumbVisibility: true,
      thickness: 2,
      radius: const Radius.circular(1),
      interactive: false,
      scrollbarOrientation: ScrollbarOrientation.bottom,
      child: SingleChildScrollView(
        key: widget.scrollViewKey,
        controller: _controller,
        scrollDirection: Axis.horizontal,
        child: widget.child,
      ),
    ),
  );
}
