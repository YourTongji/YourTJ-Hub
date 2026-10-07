import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';

Future<void> showAnonymousModeration(
  BuildContext context, {
  required int postId,
  required String publicUid,
  required bool canReveal,
}) => showDialog<void>(
  context: context,
  builder: (_) => _AnonymousModeration(
    postId: postId,
    publicUid: publicUid,
    canReveal: canReveal,
  ),
);

class _AnonymousModeration extends ConsumerStatefulWidget {
  const _AnonymousModeration({
    required this.postId,
    required this.publicUid,
    required this.canReveal,
  });
  final int postId;
  final String publicUid;
  final bool canReveal;
  @override
  ConsumerState<_AnonymousModeration> createState() =>
      _AnonymousModerationState();
}

class _AnonymousModerationState extends ConsumerState<_AnonymousModeration> {
  final reason = TextEditingController();
  late final int epoch;
  bool busy = false;
  Object? error;
  AnonymousReveal? revealed;
  @override
  void initState() {
    super.initState();
    epoch = ref.read(offlineCacheEpochProvider);
  }

  @override
  void dispose() {
    reason.dispose();
    super.dispose();
  }

  Future<void> act(String action) async {
    if (busy || reason.text.trim().isEmpty) return;
    setState(() {
      busy = true;
      error = null;
      revealed = null;
    });
    try {
      final repo = AnonymousIdentityRepository(ref.read(apiClientProvider));
      if (action == 'reveal') {
        final result = await repo.reveal(widget.publicUid, reason.text.trim());
        if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
          setState(() => revealed = result);
        }
      } else {
        await repo.govern(widget.postId, action == 'ban', reason.text.trim());
        if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
          Navigator.pop(context);
        }
      }
    } catch (e) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => error = e);
      }
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, next) {
      if (next != epoch && mounted) {
        revealed = null;
        Navigator.pop(context);
      }
    });
    final l = AppLocalizations.of(context);
    return AlertDialog(
      scrollable: true,
      title: Text(l.anonymousManage),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: reason,
            maxLength: 512,
            minLines: 2,
            maxLines: 5,
            decoration: InputDecoration(labelText: l.anonymousReason),
            onChanged: (_) => setState(() {}),
          ),
          if (error != null) Text(resolveErrorMessage(l, error!)),
          if (revealed case final r?) Text('${r.username} · ${r.userId}'),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l.anonymousCancel),
        ),
        TextButton(
          onPressed: busy || reason.text.trim().isEmpty
              ? null
              : () => act('ban'),
          child: Text(l.anonymousBan),
        ),
        TextButton(
          onPressed: busy || reason.text.trim().isEmpty
              ? null
              : () => act('restore'),
          child: Text(l.anonymousRestore),
        ),
        if (widget.canReveal)
          TextButton(
            onPressed: busy || reason.text.trim().isEmpty
                ? null
                : () => act('reveal'),
            child: Text(l.anonymousReveal),
          ),
      ],
    );
  }
}
