import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../reading_preferences.dart';
import '../../widgets/rich_content/gf_html_content.dart';
import 'course_common.dart';

/// Shared, explicit confirmation for deleting one owned course review.
Future<bool> confirmCourseReviewDeletion(
  BuildContext context,
  ReviewPayload review,
) async {
  final l10n = AppLocalizations.of(context);
  final copy = CourseCopy(l10n);
  final colors = GfTheme.colorsOf(context);
  return await showGfAlertDialog<bool>(
        context,
        builder: (dialogContext) => GfAlertDialog(
          title: GfText(
            copy.deleteReviewTitle,
            style: GfTheme.typographyOf(context).heading,
          ),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              GfText(
                copy.confirmDeleteReview,
                style: GfTheme.typographyOf(context).body,
              ),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: colors.base200,
                  borderRadius: BorderRadius.circular(GfRadius.md),
                ),
                // 与服务端阅读态同源:预览是 contentHtml 摊平后的纯文本。
                child: Consumer(
                  builder: (BuildContext context, WidgetRef ref, Widget? _) =>
                      GfText(
                        gfPlainTextFromHtml(review.contentHtml),
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: GfRichContentTypography.of(
                          context,
                          userScale: ref.watch(contentFontScaleProvider),
                          compact: true,
                        ).body,
                      ),
                ),
              ),
            ],
          ),
          actions: [
            GfButton(
              label: l10n.commonCancel,
              variant: GfButtonVariant.ghost,
              onPressed: () => Navigator.pop(dialogContext, false),
            ),
            GfButton(
              label: copy.delete,
              variant: GfButtonVariant.danger,
              onPressed: () => Navigator.pop(dialogContext, true),
            ),
          ],
        ),
      ) ??
      false;
}
