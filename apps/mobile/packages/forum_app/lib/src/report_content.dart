import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../l10n/app_localizations.dart';
import 'providers.dart';
import 'server_messages.dart';

Future<void> showContentReport(
  BuildContext context, {
  required String targetType,
  required int targetId,
}) => showDialog<void>(
  context: context,
  animationStyle: GfMotion.dialogStyle(context),
  builder: (_) => _ReportDialog(targetType: targetType, targetId: targetId),
);

class _ReportDialog extends ConsumerStatefulWidget {
  const _ReportDialog({required this.targetType, required this.targetId});
  final String targetType;
  final int targetId;
  @override
  ConsumerState<_ReportDialog> createState() => _ReportDialogState();
}

class _ReportDialogState extends ConsumerState<_ReportDialog> {
  final _note = TextEditingController();
  String _reason = 'abuse';
  bool _busy = false;
  String? _error;
  late final int _epoch = ref.read(offlineCacheEpochProvider);
  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    if (_busy || _epoch != ref.read(offlineCacheEpochProvider)) return;
    final l10n = AppLocalizations.of(context);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(postRepositoryProvider)
          .report(
            targetType: widget.targetType,
            targetId: widget.targetId,
            reason: _reason,
            note: _note.text,
          );
      if (!mounted || _epoch != ref.read(offlineCacheEpochProvider)) return;
      showGfToast(context, l10n.topicReportSubmitted);
      Navigator.pop(context);
    } catch (error) {
      if (mounted) setState(() => _error = resolveErrorMessage(l10n, error));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final reasons = {
      'spam': l10n.reportSpam,
      'abuse': l10n.reportAbuse,
      'illegal': l10n.reportIllegal,
      'irrelevant': l10n.reportIrrelevant,
      'other': l10n.reportOther,
    };
    final stale = _epoch != ref.watch(offlineCacheEpochProvider);
    return PopScope(
      canPop: !_busy,
      child: AlertDialog(
        scrollable: true,
        title: Text(
          widget.targetType == 'chat_message'
              ? l10n.messageReport
              : l10n.contentReport,
        ),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (widget.targetType == 'chat_message')
              Text(l10n.messageReportPrivacy),
            DropdownButtonFormField<String>(
              initialValue: _reason,
              isExpanded: true,
              items: [
                for (final entry in reasons.entries)
                  DropdownMenuItem(value: entry.key, child: Text(entry.value)),
              ],
              onChanged: _busy || stale
                  ? null
                  : (value) => setState(() => _reason = value!),
            ),
            TextField(
              controller: _note,
              enabled: !_busy && !stale,
              maxLength: 300,
              maxLines: 3,
              decoration: InputDecoration(
                hintText: l10n.topicReportHint,
                errorText: _error,
                errorMaxLines: 4,
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: _busy ? null : () => Navigator.pop(context),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: _busy || stale ? null : _send,
            child: Text(l10n.topicReportSubmit),
          ),
        ],
      ),
    );
  }
}
