import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_highlight/themes/a11y-dark.dart';
import 'package:flutter_highlight/themes/a11y-light.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';

/// Shared code block for rich content: language label, syntax highlighting,
/// its own horizontal scrolling and a copy action.
///
/// Both renderer paths use this one widget — the Markdown path through
/// `PreConfig.builder` and the server-HTML path through a `<pre>` override — so
/// posts, Wiki and course reviews cannot drift apart.
///
/// The block scrolls horizontally and never wraps: code keeps `white-space:
/// pre` semantics and the page itself stays horizontally fixed. Highlighting is
/// best-effort: any failure falls back to plain monospace text instead of
/// blocking the surrounding article.
class GfCodeBlock extends StatelessWidget {
  const GfCodeBlock({
    super.key,
    required this.profile,
    required this.code,
    this.language,
  });

  final GfRichContentTypography profile;
  final String code;

  /// Fenced language, when the source declared one.
  final String? language;

  /// Fenced blocks and `<pre>` both carry a trailing newline; dropping it
  /// keeps what is shown and what is copied identical.
  String get _source => code.trimRight();

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    // Rich content is also rendered outside a localised MaterialApp (share
    // images, bare widget tests), so the delegate stays optional.
    final AppLocalizations? l10n = Localizations.of<AppLocalizations>(
      context,
      AppLocalizations,
    );
    final String label = (language ?? '').trim().toUpperCase();
    final String copyLabel = l10n?.richContentCopyCode ?? 'Copy code';

    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        // A horizontal host (share images, carousels) hands us unbounded
        // width; stretching or flexing then throws, so the block keeps its
        // intrinsic width there and lets the host scroll.
        final bool bounded = constraints.hasBoundedWidth;
        final Widget languageLabel = Padding(
          padding: const EdgeInsets.only(left: 12),
          child: Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: GfTheme.typographyOf(
              context,
            ).meta.copyWith(color: colors.iconMuted),
          ),
        );

        return Container(
          margin: EdgeInsets.symmetric(vertical: profile.blockSpacing),
          decoration: BoxDecoration(
            color: colors.base200,
            border: Border.all(color: colors.line, width: borders.width),
            borderRadius: BorderRadius.circular(radii.box),
          ),
          child: Column(
            crossAxisAlignment: bounded
                ? CrossAxisAlignment.stretch
                : CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Row(
                mainAxisSize: bounded ? MainAxisSize.max : MainAxisSize.min,
                children: <Widget>[
                  // The label is user input (any fenced info string), so it
                  // must shrink instead of overflowing a narrow code block;
                  // the button keeps its 48×48 target either way.
                  if (bounded) Expanded(child: languageLabel) else languageLabel,
                  IconButton(
                    onPressed: () => _copy(
                      context,
                      l10n?.richContentCodeCopied ?? 'Code copied',
                    ),
                    tooltip: copyLabel,
                    iconSize: 18,
                    color: colors.iconMuted,
                    // 48×48 touch target even though the glyph stays small.
                    constraints: const BoxConstraints.tightFor(
                      width: 48,
                      height: 48,
                    ),
                    icon: const GfSymbol('copy', size: 18),
                  ),
                ],
              ),
              Padding(
                padding: EdgeInsets.fromLTRB(
                  profile.codePadding.left,
                  0,
                  profile.codePadding.right,
                  profile.codePadding.bottom,
                ),
                child: GfHorizontalScrollView(child: _code(context)),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _code(BuildContext context) {
    final TextStyle style = profile.code;
    final TextStyle plain = style.copyWith(
      color: GfTheme.colorsOf(context).baseContent,
    );
    final bool dark = Theme.of(context).brightness == Brightness.dark;
    final List<InlineSpan>? spans = _highlight(dark, style, plain);
    if (spans == null) return Text(_source, style: plain);
    return RichText(text: TextSpan(children: spans, style: style));
  }

  /// Returns null when highlighting is unavailable or fails, so the caller can
  /// fall back to plain monospace text.
  List<InlineSpan>? _highlight(bool dark, TextStyle style, TextStyle plain) {
    try {
      return highLightSpans(
        _source,
        language: language,
        theme: dark ? a11yDarkTheme : a11yLightTheme,
        // The block style carries no color, so each token keeps the theme's.
        textStyle: style,
        styleNotMatched: plain,
      );
    } catch (_) {
      return null;
    }
  }

  Future<void> _copy(BuildContext context, String copiedLabel) async {
    await Clipboard.setData(ClipboardData(text: _source));
    if (!context.mounted) return;
    // SnackBar is the app's existing acknowledgement surface; stay silent when
    // the block is rendered outside a Scaffold (bare widget tests).
    ScaffoldMessenger.maybeOf(context)
      ?..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(copiedLabel),
          duration: const Duration(seconds: 2),
        ),
      );
  }
}
