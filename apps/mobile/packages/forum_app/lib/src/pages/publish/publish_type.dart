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

/// Content-type choice above the composer: the quiet `GfSegmented` track with
/// each type's icon. A null [onChanged] locks the choice (existing topics,
/// pending uploads) while keeping the current type readable.
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
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          for (final item in PublishType.values)
            Expanded(
              child: Semantics(
                key: Key('publish-type-${item.value}'),
                button: true,
                selected: item.value == value,
                enabled: enabled,
                child: Material(
                  animationDuration: GfMotion.duration(
                    context,
                    GfMotion.selection,
                  ),
                  color: item.value == value
                      ? colors.base100
                      : Colors.transparent,
                  elevation: 0,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(13),
                  ),
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
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            GfSymbol(
                              item.symbol,
                              size: 16,
                              color: item.value == value
                                  ? item.color(context)
                                  : colors.iconMuted.withValues(
                                      alpha: enabled ? 1 : .5,
                                    ),
                            ),
                            const SizedBox(width: 6),
                            Flexible(
                              child: Text(
                                item.label(l10n),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
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
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
