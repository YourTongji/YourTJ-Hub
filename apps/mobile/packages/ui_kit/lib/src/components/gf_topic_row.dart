import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'atoms/gf_avatar_stack.dart';
import 'gf_chip.dart';
import 'gf_symbol.dart';

/// Category metadata for [GfTopicRow].
class GfTopicCategory {
  const GfTopicCategory({required this.name, required this.color, this.onTap});

  final String name;
  final Color color;
  final VoidCallback? onTap;
}

enum GfTopicContentType { question, moment, article }

/// Compact title size; text scale beyond [_enlargedTextScaleCeiling] × this
/// lets the title and excerpt wrap instead of staying single-line.
const double _titleFontSize = 15;
const double _enlargedTextScaleCeiling = 1.5;

/// Compact reading row with categories sharing the metadata band. Interactive
/// chips keep their 44px targets; narrow windows/large text wrap the metadata.
/// The inset divider is painted and never adds an empty footer to the layout.
class GfTopicRow extends StatelessWidget {
  const GfTopicRow({
    super.key,
    required this.title,
    required this.description,
    required this.categories,
    required this.participantAvatarUrls,
    required this.activityText,
    required this.replyCount,
    this.onTap,
    this.pinned = false,
    this.pinnedLabel = 'pinned',
    this.unseen = false,
    this.viewCount,
    this.hot = false,
    this.showDivider = true,
    this.home = false,
    this.contentType,
    this.contentTypeLabel,
  });

  final String title;
  final String description;
  final List<GfTopicCategory> categories;
  final List<String> participantAvatarUrls;
  final String activityText;
  final int replyCount;
  final VoidCallback? onTap;
  final bool pinned;
  final String pinnedLabel;
  final bool unseen;
  final int? viewCount;
  final bool hot;
  final bool showDivider;

  /// Retains the web Home minimum; text can always grow beyond it.
  final bool home;
  final GfTopicContentType? contentType;
  final String? contentTypeLabel;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final enlargedText =
        MediaQuery.textScalerOf(context).scale(_titleFontSize) >
        _titleFontSize * _enlargedTextScaleCeiling;
    return InkWell(
      onTap: onTap,
      child: Container(
        constraints: BoxConstraints(minHeight: home ? 88 : 44),
        color: colors.base100,
        child: Stack(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 6),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (title.isNotEmpty || pinned || contentType != null || hot)
                    Row(
                      children: [
                        if (pinned) ...[
                          Semantics(
                            label: pinnedLabel,
                            child: GfSymbol(
                              'pin-filled',
                              size: 16,
                              color: colors.error,
                            ),
                          ),
                          const SizedBox(width: 6),
                        ],
                        Expanded(
                          child: Text(
                            title,
                            maxLines: enlargedText ? 2 : 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: colors.baseContent,
                              fontSize: _titleFontSize,
                              fontWeight: FontWeight.w500,
                              height: 1.5,
                            ),
                          ),
                        ),
                        if (unseen) ...[
                          const SizedBox(width: 6),
                          Container(
                            width: 8,
                            height: 8,
                            decoration: BoxDecoration(
                              color: colors.primary,
                              shape: BoxShape.circle,
                            ),
                          ),
                        ],
                        if (contentType != null &&
                            contentTypeLabel != null) ...[
                          const SizedBox(width: 6),
                          _typeBadge(context),
                        ],
                        if (hot) ...[
                          const SizedBox(width: 6),
                          GfSymbol('flame', size: 12, color: colors.warning),
                          const SizedBox(width: 2),
                          Text(
                            'hot',
                            style: TextStyle(
                              color: colors.warning,
                              fontSize: 11,
                            ),
                          ),
                        ],
                      ],
                    ),
                  if (description.isNotEmpty)
                    Padding(
                      padding: EdgeInsets.only(top: title.isEmpty ? 0 : 2),
                      child: Text(
                        description,
                        maxLines: enlargedText ? 2 : 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: colors.baseContent.withValues(alpha: 0.55),
                          fontSize: 13,
                          height: 1.4,
                        ),
                      ),
                    ),
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: _metadata(context),
                  ),
                ],
              ),
            ),
            if (showDivider)
              PositionedDirectional(
                start: 16,
                end: 16,
                bottom: 0,
                child: IgnorePointer(
                  child: Container(
                    height: 1,
                    color: colors.line.withValues(alpha: 0.7),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _typeBadge(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final (symbol, color) = switch (contentType!) {
      GfTopicContentType.question => (
        'circle-help',
        dark ? colors.success : const Color(0xFF047857),
      ),
      GfTopicContentType.moment => (
        'sparkles',
        dark ? const Color(0xFFC084FC) : const Color(0xFF9333EA),
      ),
      GfTopicContentType.article => (
        'book-open',
        dark ? colors.warning : const Color(0xFF92400E),
      ),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          GfSymbol(symbol, size: 12, color: color),
          const SizedBox(width: 3),
          Text(
            contentTypeLabel!,
            style: TextStyle(
              color: color,
              fontSize: 11,
              height: 1.25,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  Widget _metadata(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final style = TextStyle(
      color: colors.baseContent.withValues(alpha: 0.55),
      fontSize: 12,
    );
    final avatars = participantAvatarUrls.take(4).toList();
    final metrics = Wrap(
      spacing: 8,
      runSpacing: 4,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        if (avatars.isNotEmpty)
          GfAvatarStack(avatarUrls: avatars, size: GfAvatarStackSize.sm),
        Text(activityText, style: style),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfSymbol('message-circle', size: 14, color: style.color),
            const SizedBox(width: 4),
            Text('$replyCount', style: style),
          ],
        ),
      ],
    );
    if (categories.isEmpty) return metrics;
    final categoryRail = SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: [
          for (var i = 0; i < categories.length; i++) ...[
            if (i > 0) const SizedBox(width: 4),
            GfChip(
              label: categories[i].name,
              color: categories[i].color,
              onTap: categories[i].onTap,
            ),
          ],
        ],
      ),
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        // Measure the actual localized/scaled metadata. A fixed-width shortcut
        // clips long German dates or large counts at otherwise ordinary widths.
        // Perf-sensitive: each build runs two TextPainter layouts, but only for
        // rows with categories. ListView builds rows lazily so this stays off
        // the frame-critical path today; revisit here first if list scroll
        // profiling regresses.
        double textWidth(String text) {
          final painter = TextPainter(
            text: TextSpan(
              text: text,
              style: DefaultTextStyle.of(context).style.merge(style),
            ),
            textDirection: Directionality.of(context),
            textScaler: MediaQuery.textScalerOf(context),
          )..layout();
          final width = painter.width;
          painter.dispose();
          return width;
        }

        final avatarWidth = avatars.isEmpty
            ? 0
            : 24 + (avatars.length - 1) * 16 + 8;
        final metricsWidth =
            avatarWidth +
            textWidth(activityText) +
            8 +
            18 +
            textWidth('$replyCount');
        final categoryWidth = math.max(
          88.0,
          MediaQuery.textScalerOf(context).scale(64),
        );
        if (metricsWidth + 8 + categoryWidth > constraints.maxWidth) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [metrics, categoryRail],
          );
        }
        return Row(
          children: [
            metrics,
            const SizedBox(width: 8),
            Expanded(child: categoryRail),
          ],
        );
      },
    );
  }
}
