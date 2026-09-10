import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import 'post_history_sheet.dart';

class TopicActions extends ConsumerStatefulWidget {
  const TopicActions({
    super.key,
    required this.props,
    required this.onChanged,
    this.firstPostId,
  });
  final TopicDetailProps props;
  final int? firstPostId;
  final Future<void> Function() onChanged;
  @override
  ConsumerState<TopicActions> createState() => _TopicActionsState();
}

class _TopicActionsState extends ConsumerState<TopicActions> {
  bool _busy = false;
  Future<void> _action(String action) async {
    if (_busy) return;
    final l10n = AppLocalizations.of(context);
    final epoch = ref.read(offlineCacheEpochProvider);
    final topic = widget.props.topic;
    if (action == 'edit') {
      await context.push('/publish?id=${topic.id}');
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        await widget.onChanged();
      }
      return;
    }
    if (action == 'history') {
      await showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        useSafeArea: true,
        builder: (_) => PostHistorySheet(postId: widget.firstPostId!),
      );
      return;
    }
    if (action == 'share') {
      final url = Uri.parse(
        ref.read(apiClientProvider).baseUrl,
      ).replace(path: '/p/post/${topic.id}', query: null, fragment: null);
      final box = context.findRenderObject() as RenderBox?;
      setState(() => _busy = true);
      try {
        await SharePlus.instance.share(
          ShareParams(
            text: url.toString(),
            sharePositionOrigin: box == null
                ? null
                : box.localToGlobal(Offset.zero) & box.size,
          ),
        );
      } catch (error) {
        if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
          showGfToast(context, resolveErrorMessage(l10n, error), error: true);
        }
      } finally {
        if (mounted) setState(() => _busy = false);
      }
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          action == 'delete'
              ? l10n.contentDelete
              : action == 'ban'
              ? l10n.topicModerateBan
              : l10n.topicModerateUnban,
        ),
        content: Text(
          action == 'delete'
              ? l10n.topicDeleteConfirm
              : l10n.topicModerateConfirm,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(l10n.commonCancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(l10n.commonConfirm),
          ),
        ],
      ),
    );
    if (!mounted ||
        confirmed != true ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    setState(() => _busy = true);
    try {
      final repo = ref.read(topicRepositoryProvider);
      if (action == 'delete') {
        await repo.deleteTopic(topicId: topic.id);
      } else {
        await repo.moderate(topicId: topic.id, ban: action == 'ban');
      }
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (action == 'delete') {
        context.go('/');
      } else {
        await widget.onChanged();
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(context, resolveErrorMessage(l10n, error), error: true);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final props = widget.props;
    final available =
        !props.topic.authorDeleted && !props.topic.moderatorRemoved;
    return PopupMenuButton<String>(
      tooltip: l10n.profileMore,
      enabled: !_busy,
      onSelected: _action,
      itemBuilder: (_) => [
        if (props.permissions.isOwnTopic && available)
          PopupMenuItem(value: 'edit', child: Text(l10n.commonEdit)),
        if (props.permissions.isOwnTopic && available)
          PopupMenuItem(value: 'delete', child: Text(l10n.contentDelete)),
        if (widget.firstPostId != null)
          PopupMenuItem(value: 'history', child: Text(l10n.topicHistory)),
        PopupMenuItem(value: 'share', child: Text(l10n.topicShare)),
        if (props.permissions.canModerateTopic &&
            props.topic.processStatus == 0)
          PopupMenuItem(value: 'ban', child: Text(l10n.topicModerateBan)),
        if (props.permissions.canModerateTopic &&
            props.topic.processStatus == 1)
          PopupMenuItem(value: 'unban', child: Text(l10n.topicModerateUnban)),
      ],
    );
  }
}
