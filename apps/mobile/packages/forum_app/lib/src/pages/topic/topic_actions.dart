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
import 'post_share_image.dart';
import '../../report_content.dart';

class TopicActions extends ConsumerStatefulWidget {
  const TopicActions({
    super.key,
    required this.props,
    required this.onChanged,
    this.firstPost,
  });
  final TopicDetailProps props;
  final PostPayload? firstPost;
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
      await showGfBottomSheet<void>(
        context,
        builder: (_) => PostHistorySheet(postId: widget.firstPost!.id),
      );
      return;
    }
    if (action == 'report') {
      await showContentReport(context, targetType: 'topic', targetId: topic.id);
      return;
    }
    if (action == 'shareImage' && widget.firstPost != null) {
      await showPostShareImage(
        context,
        post: widget.firstPost!,
        topicTitle: topic.title,
        baseUrl: ref.read(apiClientProvider).baseUrl,
        topicLink: true,
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
    final confirmed = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: GfText(
          action == 'delete'
              ? l10n.contentDelete
              : action == 'ban'
              ? l10n.topicModerateBan
              : l10n.topicModerateUnban,
        ),
        content: GfText(
          action == 'delete'
              ? l10n.topicDeleteConfirm
              : l10n.topicModerateConfirm,
        ),
        actions: [
          GfDialogAction(
            onPressed: () => Navigator.pop(context, false),
            child: GfText(l10n.commonCancel),
          ),
          GfDialogAction(
            destructive: true,
            onPressed: () => Navigator.pop(context, true),
            child: GfText(l10n.commonConfirm),
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
        ref.read(removedTopicsProvider.notifier).add(topic.id);
        ref.read(topicReturnStatesProvider).remove(topic.id);
        if (context.canPop()) {
          context.pop();
        } else {
          context.go('/');
        }
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
    return GfPopupMenuButton<String>(
      tooltip: l10n.profileMore,
      icon: const GfSymbol('ellipsis', size: 20),
      useRootNavigator: true,
      enabled: !_busy,
      onSelected: _action,
      itemBuilder: (_) => [
        if (props.permissions.isOwnTopic && available)
          GfPopupMenuItem(value: 'edit', child: GfText(l10n.commonEdit)),
        if (props.permissions.isOwnTopic && available)
          GfPopupMenuItem(
            destructive: true,
            value: 'delete',
            child: GfText(l10n.contentDelete),
          ),
        if (widget.firstPost != null)
          GfPopupMenuItem(value: 'history', child: GfText(l10n.topicHistory)),
        GfPopupMenuItem(value: 'share', child: GfText(l10n.topicShare)),
        if (available &&
            widget.firstPost != null &&
            !widget.firstPost!.isHidden &&
            !widget.firstPost!.isAuthorDeleted &&
            !widget.firstPost!.isModeratorRemoved)
          GfPopupMenuItem(
            value: 'shareImage',
            child: GfText(l10n.topicShareImage),
          ),
        if (available && !props.permissions.isOwnTopic)
          GfPopupMenuItem(value: 'report', child: GfText(l10n.topicReport)),
        if (props.permissions.canModerateTopic &&
            props.topic.processStatus == 0)
          GfPopupMenuItem(
            destructive: true,
            value: 'ban',
            child: GfText(l10n.topicModerateBan),
          ),
        if (props.permissions.canModerateTopic &&
            props.topic.processStatus == 1)
          GfPopupMenuItem(
            value: 'unban',
            child: GfText(l10n.topicModerateUnban),
          ),
      ],
    );
  }
}
