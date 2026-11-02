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
