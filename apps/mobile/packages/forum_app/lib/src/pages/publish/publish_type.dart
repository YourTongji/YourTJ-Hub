import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';

/// Wire values and Lucide colours from Web's PublishMenu.vue.
enum PublishType {
  moment(2, 'sparkles', Color(0xFF9333EA), Color(0xFFC084FC)),
  article(3, 'book-open', Color(0xFFD97706), Color(0xFFFBBF24)),
  question(1, 'circle-help', Color(0xFF059669), Color(0xFF34D399));

  const PublishType(this.value, this.symbol, this.light, this.dark);
  final int value;
  final String symbol;
  final Color light;
  final Color dark;

  static PublishType fromValue(int value) =>
      values.firstWhere((type) => type.value == value, orElse: () => article);

  Color color(BuildContext context) =>
      Theme.of(context).brightness == Brightness.dark ? dark : light;

  String label(AppLocalizations l10n) => switch (this) {
    moment => l10n.publishMoment,
    article => l10n.publishArticle,
    question => l10n.publishQuestion,
  };

  String hint(AppLocalizations l10n) => switch (this) {
    moment => l10n.publishMomentHint,
    article => l10n.publishArticleHint,
    question => l10n.publishQuestionHint,
  };
}

class PublishTypeIcon extends StatelessWidget {
  const PublishTypeIcon(this.type, {super.key, this.size = 40});
  final PublishType type;
  final double size;

  @override
  Widget build(BuildContext context) => Container(
    width: size,
    height: size,
    decoration: BoxDecoration(
      color: type.color(context).withValues(alpha: .12),
      shape: BoxShape.circle,
    ),
    alignment: Alignment.center,
    child: GfSymbol(type.symbol, color: type.color(context), size: size * .5),
  );
}

/// Content-type choice above the composer: a quiet track whose raised pill
/// slides to the chosen type. A null [onChanged] locks the choice (existing
/// topics, pending uploads) while keeping the current type readable.
class PublishTypeSwitcher extends StatelessWidget {
  const PublishTypeSwitcher({
    super.key,
    required this.value,
    required this.onChanged,
  });
  final int value;
  final ValueChanged<int>? onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final enabled = onChanged != null;
    final duration = GfMotion.duration(context, GfMotion.layout);
    final index = PublishType.values.indexWhere((t) => t.value == value);
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Stack(
        children: [
          // The pill moves instead of each segment repainting, so the eye
          // follows the change.
          Positioned.fill(
            child: AnimatedAlign(
              duration: duration,
              curve: GfMotion.enterCurve,
              alignment: AlignmentDirectional(
                index * 2 / (PublishType.values.length - 1) - 1,
                0,
              ).resolve(Directionality.of(context)),
              child: FractionallySizedBox(
                widthFactor: 1 / PublishType.values.length,
                heightFactor: 1,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: colors.base100,
                    borderRadius: BorderRadius.circular(13),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.black.withValues(alpha: .06),
                        blurRadius: 6,
                        offset: const Offset(0, 1),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
          Row(
            children: [
              for (final item in PublishType.values)
                Expanded(
                  child: Semantics(
                    key: Key('publish-type-${item.value}'),
                    button: true,
                    selected: item.value == value,
                    enabled: enabled,
                    child: InkWell(
                      borderRadius: BorderRadius.circular(13),
                      onTap: enabled && item.value != value
                          ? () => onChanged!(item.value)
                          : null,
                      child: ConstrainedBox(
                        constraints: const BoxConstraints(minHeight: 40),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 8,
                          ),
                          child: _SegmentLabel(
                            symbol: item.symbol,
                            label: item.label(l10n),
                            duration: duration,
                            iconColor: item.value == value
                                ? item.color(context)
                                : colors.iconMuted.withValues(
                                    alpha: enabled ? 1 : .5,
                                  ),
                            style: type.small.copyWith(
                              fontWeight: FontWeight.w600,
                              color: item.value == value
                                  ? colors.baseContent
                                  : colors.baseContent.withValues(
                                      alpha: enabled ? .6 : .35,
                                    ),
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

class _SegmentLabel extends StatelessWidget {
  const _SegmentLabel({
    required this.symbol,
    required this.label,
    required this.duration,
    required this.iconColor,
    required this.style,
  });

  final String symbol;
  final String label;
  final Duration duration;
  final Color iconColor;
  final TextStyle style;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisAlignment: MainAxisAlignment.center,
    children: [
      TweenAnimationBuilder<Color?>(
        tween: ColorTween(end: iconColor),
        duration: duration,
        builder: (context, color, _) =>
            GfSymbol(symbol, size: 16, color: color),
      ),
      const SizedBox(width: 6),
      Flexible(
        child: AnimatedDefaultTextStyle(
          duration: duration,
          style: style,
          child: Text(label, maxLines: 1, overflow: TextOverflow.ellipsis),
        ),
      ),
    ],
  );
}
