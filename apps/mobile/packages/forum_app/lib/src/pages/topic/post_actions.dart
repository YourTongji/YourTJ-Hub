import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import 'post_edit_sheet.dart';
import 'post_history_sheet.dart';

class PostActions extends ConsumerStatefulWidget {
  const PostActions({
    super.key,
    required this.post,
    required this.onChanged,
    required this.onReply,
    required this.onReport,
  });
  final PostPayload post;
  final Future<void> Function() onChanged;
  final VoidCallback? onReply;
  final VoidCallback onReport;
  @override
  ConsumerState<PostActions> createState() => _PostActionsState();
}

class _PostActionsState extends ConsumerState<PostActions> {
  bool _busy = false;
  bool get _removed =>
      widget.post.isAuthorDeleted || widget.post.isModeratorRemoved;
  Future<void> _run(Future<void> Function() action) async {
    if (_busy) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _busy = true);
    try {
      await action();
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        await widget.onChanged();
      }
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _action(String action) async {
    if (_busy) return;
    final l10n = AppLocalizations.of(context);
    final post = widget.post;
    final epoch = ref.read(offlineCacheEpochProvider);
    if (action == 'history') {
      await showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        useSafeArea: true,
        builder: (_) => PostHistorySheet(postId: post.id),
      );
      return;
    }
    if (action == 'edit') {
      final saved = await showModalBottomSheet<bool>(
        context: context,
        isScrollControlled: true,
        isDismissible: false,
        enableDrag: false,
        useSafeArea: true,
        builder: (_) => PostEditSheet(post: post),
      );
      if (saved == true &&
          mounted &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        await widget.onChanged();
      }
      return;
    }
    if (action == 'share') {
      final url = Uri.parse(ref.read(apiClientProvider).baseUrl).replace(
        path: '/p/post/${post.topicId}/${post.postNo}',
        query: null,
        fragment: null,
      );
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
    final confirm = await showDialog<bool>(
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
        confirm != true ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    await _run(() async {
      final repo = ref.read(postRepositoryProvider);
      if (action == 'delete') {
        await repo.deletePost(postId: post.id);
      } else {
        await repo.moderate(postId: post.id, ban: action == 'ban');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final post = widget.post;
    final available = !_removed && !post.isHidden;
    final colors = GfTheme.colorsOf(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (available) ...[
          Tooltip(
            message: l10n.topicLike,
            child: TextButton.icon(
              onPressed: _busy
                  ? null
                  : () => _run(() async {
                      await ref
                          .read(postRepositoryProvider)
                          .likePost(
                            postId: post.id,
                            action: post.isLiked ? 2 : 1,
                          );
                    }),
              icon: Icon(
                post.isLiked ? Icons.favorite : Icons.favorite_border,
                size: 18,
                color: post.isLiked ? colors.error : colors.iconMuted,
              ),
              label: Text('${post.likeCount}'),
            ),
          ),
          IconButton(
            tooltip: post.isBookmarked
                ? l10n.topicBookmarked
                : l10n.topicBookmark,
            onPressed: _busy
                ? null
                : () => _run(() async {
                    await ref
                        .read(postRepositoryProvider)
                        .bookmarkPost(
                          postId: post.id,
                          action: post.isBookmarked ? 2 : 1,
                        );
                  }),
            icon: Icon(
              post.isBookmarked ? Icons.bookmark : Icons.bookmark_border,
              size: 18,
            ),
          ),
          if (widget.onReply != null)
            IconButton(
              tooltip: l10n.topicReply,
              onPressed: _busy ? null : widget.onReply,
              icon: const Icon(Icons.reply_outlined, size: 18),
            ),
        ],
        PopupMenuButton<String>(
          tooltip: l10n.profileMore,
          enabled: !_busy,
          onSelected: _action,
          itemBuilder: (_) => [
            if (post.isOwnPost && available)
              PopupMenuItem(value: 'edit', child: Text(l10n.commonEdit)),
            if (post.isOwnPost && available)
              PopupMenuItem(value: 'delete', child: Text(l10n.contentDelete)),
            PopupMenuItem(value: 'history', child: Text(l10n.topicHistory)),
            PopupMenuItem(value: 'share', child: Text(l10n.topicShare)),
            if (post.canModerate && post.processStatus == 0)
              PopupMenuItem(value: 'ban', child: Text(l10n.topicModerateBan)),
            if (post.canModerate && post.processStatus == 1)
              PopupMenuItem(
                value: 'unban',
                child: Text(l10n.topicModerateUnban),
              ),
          ],
        ),
        if (!post.isOwnPost && available)
          IconButton(
            tooltip: l10n.topicReport,
            onPressed: _busy ? null : widget.onReport,
            icon: const Icon(Icons.flag_outlined, size: 18),
          ),
      ],
    );
  }
}
