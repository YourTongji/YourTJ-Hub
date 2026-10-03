import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';
import '../server_messages.dart';

/// AI 图文审查拦截码(issue #975),与 Web `runtime/moderation-blocked.ts` 一致。
const String _policyBlockedCode = 'content.aiModeration.blocked';
const String _externalImageBlockedCode =
    'content.aiModeration.externalImageBlocked';

/// [error] 是否为 AI 图文审查拦截。
bool isModerationBlocked(Object error) =>
    error is ApiException &&
    (error.messageCode == _policyBlockedCode ||
        error.messageCode == _externalImageBlockedCode);

/// 内容被拦截时弹出友好提示:说明未发布原因、正文与图片仍保留在编辑器、
/// 如何修改后重发;不展示模型类别或概率。非拦截错误返回 false,调用方
/// 继续沿用原有错误提示。
Future<bool> showModerationBlockedDialog(
  BuildContext context,
  Object error,
) async {
  if (!isModerationBlocked(error)) return false;
  final AppLocalizations l10n = AppLocalizations.of(context);
  final bool externalImage =
      (error as ApiException).messageCode == _externalImageBlockedCode;
  final GfColors colors = GfTheme.colorsOf(context);
  await showGfAlertDialog<void>(
    context,
    barrierDismissible: true,
    builder: (dialogContext) => GfAlertDialog(
      title: Text(l10n.moderationBlockedTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(resolveErrorMessage(l10n, error)),
          const SizedBox(height: 8),
          Text(
            externalImage
                ? l10n.moderationBlockedExternalHint
                : l10n.moderationBlockedPolicyHint,
            style: TextStyle(color: colors.iconMuted),
          ),
          const SizedBox(height: 12),
          DecoratedBox(
            decoration: BoxDecoration(
              color: colors.base200,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              child: Text(
                l10n.moderationBlockedDraftKept,
                style: TextStyle(fontSize: 13, color: colors.iconMuted),
              ),
            ),
          ),
        ],
      ),
      actions: [
        GfButton(
          label: l10n.moderationBlockedBack,
          onPressed: () => Navigator.pop(dialogContext),
        ),
      ],
    ),
  );
  return true;
}
