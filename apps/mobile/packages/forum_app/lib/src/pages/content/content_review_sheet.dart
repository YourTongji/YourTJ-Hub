import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';

/// What the author picked in the review sheet; the content page owns navigation.
enum ContentReviewAction { view, edit, appealed }

/// Server limit for appeal explanations, counted in Unicode code points.
const int appealReasonMaxLength = 512;

/// Review result for one rejected or pending item, with the appeal entry and
/// its progress (#1114). Mirrors the web content-management card.
Future<ContentReviewAction?> showContentReviewSheet(
  BuildContext context,
  UserContentItem item, {
  bool startAppeal = false,
}) => showGfBottomSheet<ContentReviewAction>(
  context,
  keyboardAware: true,
  builder: (_) => ContentReviewSheet(item: item, startAppeal: startAppeal),
);

class ContentReviewSheet extends ConsumerStatefulWidget {
  const ContentReviewSheet({
    super.key,
    required this.item,
    this.startAppeal = false,
  });

  final UserContentItem item;
  final bool startAppeal;

  @override
  ConsumerState<ContentReviewSheet> createState() => _ContentReviewSheetState();
}

class _ContentReviewSheetState extends ConsumerState<ContentReviewSheet> {
  final TextEditingController _reason = TextEditingController();
  late bool _appealing = widget.startAppeal && _canAppeal;
  bool _busy = false;
  String? _error;
  late final int _epoch = ref.read(offlineCacheEpochProvider);

  bool get _canAppeal =>
      widget.item.processStatus == 1 &&
      widget.item.canAppeal &&
      widget.item.appeal == null;

  @override
  void dispose() {
    _reason.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final text = _reason.text.trim();
    if (_busy ||
        text.isEmpty ||
        _epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    final l10n = AppLocalizations.of(context);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref.read(contentRepositoryProvider).submitAppeal(widget.item, text);
      if (!mounted || _epoch != ref.read(offlineCacheEpochProvider)) return;
      showGfToast(context, l10n.appealSubmitted);
      Navigator.of(context).pop(ContentReviewAction.appealed);
    } catch (error) {
      // 业务失败说明具体原因；网络失败提示重试，输入的理由保留在框里。
      if (mounted) {
        setState(
          () => _error =
              error is ApiException && (error.messageCode ?? '').isNotEmpty
              ? resolveErrorMessage(l10n, error)
              : l10n.appealSubmitFailed,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final item = widget.item;
    final stale = _epoch != ref.watch(offlineCacheEpochProvider);
    final subject = item.title.isNotEmpty ? item.title : item.excerpt;
    final muted = colors.baseContent.withValues(alpha: 0.6);
    return PopScope(
      canPop: !_busy,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.85,
        ),
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              GfText(l10n.contentReviewSheetTitle, style: type.heading),
              if (subject.isNotEmpty) ...[
                const SizedBox(height: 2),
                GfText(
                  subject,
                  style: type.small.copyWith(color: muted),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
              const SizedBox(height: 12),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  item.processStatus == 1
                      ? GfBadge(
                          label: l10n.contentReviewBlocked,
                          variant: GfBadgeVariant.error,
                        )
                      : GfBadge(
                          label: l10n.contentReviewPending,
                          variant: GfBadgeVariant.warning,
                        ),
                ],
              ),
              if (item.processStatus == 1 && item.reviewReason.isNotEmpty) ...[
                const SizedBox(height: 12),
                GfText(
                  l10n.contentReviewReasonLabel,
                  style: type.caption.copyWith(color: muted),
                ),
                const SizedBox(height: 2),
                GfText(item.reviewReason, style: type.body),
              ],
              if (item.hasPublishedVersion) ...[
                const SizedBox(height: 8),
                GfText(
                  l10n.contentReviewLive,
                  style: type.small.copyWith(color: muted),
                ),
              ],
              if (item.appeal != null) ...[
                const SizedBox(height: 16),
                AppealStateCard(appeal: item.appeal!),
              ],
              AnimatedSize(
                duration: GfMotion.duration(context, GfMotion.layout),
                curve: GfMotion.enterCurve,
                alignment: Alignment.topCenter,
                child: _appealing
                    ? _AppealForm(
                        controller: _reason,
                        enabled: !_busy && !stale,
                        busy: _busy,
                        error: _error,
                        onChanged: () => setState(() => _error = null),
                        onCancel: () => setState(() {
                          _appealing = false;
                          _error = null;
                        }),
                        onSubmit: _submit,
                      )
                    : const SizedBox(width: double.infinity),
              ),
              if (!_appealing) ...[
                const SizedBox(height: 20),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    GfButton(
                      label: l10n.contentReviewView,
                      icon: const GfSymbol('eye', size: 16),
                      variant: GfButtonVariant.secondary,
                      size: GfButtonSize.small,
                      onPressed: () =>
                          Navigator.of(context).pop(ContentReviewAction.view),
                    ),
                    GfButton(
                      label: l10n.contentReviewRetry,
                      icon: const GfSymbol('pen-line', size: 16),
                      variant: GfButtonVariant.secondary,
                      size: GfButtonSize.small,
                      onPressed: () =>
                          Navigator.of(context).pop(ContentReviewAction.edit),
                    ),
                    if (_canAppeal)
                      GfButton(
                        key: const Key('appeal-start'),
                        label: l10n.appealAction,
                        icon: const GfSymbol('undo-2', size: 16),
                        variant: GfButtonVariant.secondary,
                        size: GfButtonSize.small,
                        onPressed: stale
                            ? null
                            : () => setState(() => _appealing = true),
                      ),
                  ],
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// Progress of the appeal on the latest revision.
class AppealStateCard extends StatelessWidget {
  const AppealStateCard({super.key, required this.appeal});

  final ModerationAppealOwnerSummary appeal;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final muted = colors.baseContent.withValues(alpha: 0.6);
    final (
      String symbol,
      Color tone,
      String title,
      String body,
    ) = switch (appeal.status) {
      'approved' => (
        'circle-check',
        colors.success,
        l10n.appealStateApprovedTitle,
        l10n.appealStateApprovedBody,
      ),
      'rejected' => (
        'circle-alert',
        colors.error,
        l10n.appealStateRejectedTitle,
        l10n.appealStateRejectedBody,
      ),
      'stale' => (
        'circle',
        colors.iconMuted,
        l10n.appealStateStaleTitle,
        l10n.appealStateStaleBody,
      ),
      _ => (
        'clock',
        colors.warning,
        l10n.appealStatePendingTitle,
        l10n.appealStatePendingBody,
      ),
    };
    final times = [
      l10n.appealSubmittedAt(timeAgo(appeal.createdAt, l10n: l10n)),
      if (appeal.reviewedAt != null)
        l10n.appealDecidedAt(timeAgo(appeal.reviewedAt!, l10n: l10n)),
    ].join(' · ');
    return Container(
      key: const Key('appeal-state'),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 2),
            child: GfSymbol(symbol, size: 18, color: tone),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                GfText(title, style: type.bodyStrong),
                const SizedBox(height: 4),
                GfText(body, style: type.small.copyWith(color: muted)),
                if (appeal.status == 'rejected') ...[
                  const SizedBox(height: 10),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 8,
                    ),
                    decoration: BoxDecoration(
                      color: colors.base100,
                      borderRadius: BorderRadius.circular(
                        GfTheme.radiiOf(context).field,
                      ),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        GfText(
                          l10n.appealReviewerNote,
                          style: type.caption.copyWith(color: muted),
                        ),
                        const SizedBox(height: 2),
                        GfText(
                          appeal.reviewReason.isNotEmpty
                              ? appeal.reviewReason
                              : l10n.appealNoReviewerNote,
                          style: type.small,
                        ),
                      ],
                    ),
                  ),
                ],
                const SizedBox(height: 10),
                GfText(
                  l10n.appealYourReason,
                  style: type.caption.copyWith(color: muted),
                ),
                const SizedBox(height: 2),
                GfText(
                  appeal.reason,
                  style: type.small,
                  maxLines: 4,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 8),
                GfText(times, style: type.caption.copyWith(color: muted)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _AppealForm extends StatelessWidget {
  const _AppealForm({
    required this.controller,
    required this.enabled,
    required this.busy,
    required this.error,
    required this.onChanged,
    required this.onCancel,
    required this.onSubmit,
  });

  final TextEditingController controller;
  final bool enabled;
  final bool busy;
  final String? error;
  final VoidCallback onChanged;
  final VoidCallback onCancel;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final length = controller.text.runes.length;
    return Padding(
      padding: const EdgeInsets.only(top: 16),
      child: Column(
        key: const Key('appeal-form'),
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          GfText(l10n.appealFormTitle, style: type.bodyStrong),
          const SizedBox(height: 4),
          GfText(
            l10n.appealFormIntro,
            style: type.small.copyWith(
              color: colors.baseContent.withValues(alpha: 0.6),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: controller,
            enabled: enabled,
            autofocus: true,
            minLines: 3,
            maxLines: 6,
            textInputAction: TextInputAction.newline,
            inputFormatters: const [
              _RuneLengthLimitingFormatter(appealReasonMaxLength),
            ],
            onChanged: (_) => onChanged(),
            decoration: InputDecoration(
              labelText: l10n.appealReasonLabel,
              hintText: l10n.appealReasonPlaceholder,
              alignLabelWithHint: true,
              helperText: l10n.appealOncePerVersion,
              errorText: error,
              errorMaxLines: 4,
              counterText: '$length/$appealReasonMaxLength',
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: GfButton(
                  label: l10n.commonCancel,
                  variant: GfButtonVariant.secondary,
                  onPressed: busy ? null : onCancel,
                  expanded: true,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: GfButton(
                  key: const Key('appeal-submit'),
                  label: l10n.appealSubmit,
                  loading: busy,
                  expanded: true,
                  onPressed: !enabled || controller.text.trim().isEmpty
                      ? null
                      : onSubmit,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _RuneLengthLimitingFormatter extends TextInputFormatter {
  const _RuneLengthLimitingFormatter(this.maxLength);

  final int maxLength;

  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final runes = newValue.text.runes;
    if (runes.length <= maxLength) return newValue;
    final text = String.fromCharCodes(runes.take(maxLength));
    final offset = newValue.selection.baseOffset.clamp(0, text.length);
    return newValue.copyWith(
      text: text,
      selection: TextSelection.collapsed(offset: offset),
      composing: TextRange.empty,
    );
  }
}
