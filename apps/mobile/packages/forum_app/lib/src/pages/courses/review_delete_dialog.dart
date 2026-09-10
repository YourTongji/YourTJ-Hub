import 'package:flutter/material.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import 'course_common.dart';

/// Shared, explicit confirmation for deleting one owned course review.
Future<bool> confirmCourseReviewDeletion(
  BuildContext context,
  ReviewPayload review,
) async {
  final l10n = AppLocalizations.of(context);
  final copy = CourseCopy(l10n);
  final colors = GfTheme.colorsOf(context);
  return await showDialog<bool>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          backgroundColor: colors.base100,
          surfaceTintColor: Colors.transparent,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(24),
          ),
          insetPadding: const EdgeInsets.symmetric(
            horizontal: 24,
            vertical: 24,
          ),
          titlePadding: const EdgeInsets.fromLTRB(24, 24, 24, 12),
          contentPadding: const EdgeInsets.fromLTRB(24, 0, 24, 0),
          actionsPadding: const EdgeInsets.fromLTRB(24, 12, 24, 12),
          title: Text(
            copy.deleteReviewTitle,
            style: GfTheme.typographyOf(context).heading,
          ),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                copy.confirmDeleteReview,
                style: GfTheme.typographyOf(context).body,
              ),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: colors.base200,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  review.content,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: GfTheme.typographyOf(context).small,
                ),
              ),
            ],
          ),
          actions: [
            Row(
              children: [
                Expanded(
                  child: GfButton(
                    label: l10n.commonCancel,
                    variant: GfButtonVariant.ghost,
                    onPressed: () => Navigator.pop(dialogContext, false),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: GfButton(
                    label: copy.delete,
                    variant: GfButtonVariant.danger,
                    onPressed: () => Navigator.pop(dialogContext, true),
                  ),
                ),
              ],
            ),
          ],
        ),
      ) ??
      false;
}
