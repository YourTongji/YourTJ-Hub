import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';

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

Future<void> showCourseReviewReportSheet(
  BuildContext context, {
  required CourseRepository repository,
  required int reviewId,
}) => showGfBottomSheet<void>(
  context,
  keyboardAware: true,
  barrierDismissible: false,
  enableDrag: false,
  builder: (_) =>
      _CourseReviewReportSheet(repository: repository, reviewId: reviewId),
);

class _CourseReviewReportSheet extends ConsumerStatefulWidget {
  const _CourseReviewReportSheet({
    required this.repository,
    required this.reviewId,
  });

  final CourseRepository repository;
  final int reviewId;

  @override
  ConsumerState<_CourseReviewReportSheet> createState() =>
      _CourseReviewReportSheetState();
}

class _CourseReviewReportSheetState
    extends ConsumerState<_CourseReviewReportSheet> {
  final TextEditingController _note = TextEditingController();
  String? _reason;
  String? _error;
  bool _busy = false;
  late final int _epoch = ref.read(offlineCacheEpochProvider);

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final reason = _reason;
    if (_busy ||
        reason == null ||
        _epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    final l10n = AppLocalizations.of(context);
    try {
      await widget.repository.reportReview(
        reviewId: widget.reviewId,
        reason: reason,
        note: _note.text.trim(),
      );
      if (!mounted || _epoch != ref.read(offlineCacheEpochProvider)) return;
      showGfToast(context, l10n.topicReportSubmitted);
      Navigator.of(context).pop();
    } catch (error) {
      if (mounted) setState(() => _error = resolveErrorMessage(l10n, error));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final stale = _epoch != ref.watch(offlineCacheEpochProvider);
    final reasons = <String, String>{
      'spam': l10n.reportSpam,
      'abuse': l10n.reportAbuse,
      'illegal': l10n.reportIllegal,
      'irrelevant': l10n.reportIrrelevant,
      'other': l10n.reportOther,
    };
    return PopScope(
      canPop: !_busy,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.8,
        ),
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                l10n.contentReport,
                style: GfTheme.typographyOf(context).heading,
              ),
              const SizedBox(height: 16),
              DropdownButtonFormField<String>(
                initialValue: _reason,
                isExpanded: true,
                decoration: InputDecoration(
                  hintText: l10n.courseReviewReportReasonHint,
                ),
                items: [
                  for (final entry in reasons.entries)
                    DropdownMenuItem(
                      value: entry.key,
                      child: Text(entry.value),
                    ),
                ],
                onChanged: _busy || stale
                    ? null
                    : (value) => setState(() => _reason = value),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: _note,
                enabled: !_busy && !stale,
                inputFormatters: const [_RuneLengthLimitingFormatter(300)],
                onChanged: (_) => setState(() {}),
                maxLines: 3,
                decoration: InputDecoration(
                  labelText: l10n.courseReviewReportNoteLabel,
                  hintText: l10n.topicReportHint,
                  counterText: '${_note.text.runes.length}/300',
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 4),
                Text(
                  _error!,
                  style: TextStyle(color: GfTheme.colorsOf(context).error),
                ),
              ],
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: _busy ? null : () => Navigator.pop(context),
                      child: Text(l10n.commonCancel),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: _busy || stale || _reason == null
                          ? null
                          : _submit,
                      child: Text(l10n.topicReportSubmit),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
