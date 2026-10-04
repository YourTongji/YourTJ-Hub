import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';

/// Keep the author's changes in the editor when a resubmission fails.
class ContentReplyDialog extends ConsumerStatefulWidget {
  const ContentReplyDialog({super.key, required this.item});
  final UserContentItem item;

  @override
  ConsumerState<ContentReplyDialog> createState() => _ContentReplyDialogState();
}

class _ContentReplyDialogState extends ConsumerState<ContentReplyDialog> {
  late final _controller = TextEditingController(text: widget.item.content);
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_busy) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(postRepositoryProvider)
          .updatePost(postId: widget.item.id, content: _controller.text);
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        Navigator.pop(context, true);
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      if (mounted) Navigator.pop(context);
    });
    return AlertDialog(
      title: Text(l10n.contentReviewRetry),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: _controller,
              enabled: !_busy,
              minLines: 4,
              maxLines: 12,
              decoration: InputDecoration(labelText: l10n.contentReviewView),
            ),
            if (_error != null)
              Text(
                _error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.pop(context),
          child: Text(l10n.commonCancel),
        ),
        TextButton(
          onPressed: _busy ? null : _save,
          child: Text(l10n.contentReviewRetry),
        ),
      ],
    );
  }
}
