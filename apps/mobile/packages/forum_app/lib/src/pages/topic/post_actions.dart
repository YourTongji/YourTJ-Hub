import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import 'post_edit_sheet.dart';
import 'post_history_sheet.dart';

const double _postActionIconSize = 20;

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
  int? _actionEpoch;
  bool get _removed =>
      widget.post.isAuthorDeleted || widget.post.isModeratorRemoved;
  void _recordState({bool? liked, bool? bookmarked}) {
    if (!mounted ||
        (_busy && _actionEpoch != ref.read(offlineCacheEpochProvider))) {
      return;
    }
    final post = widget.post;
    if (_removed || post.isHidden) return;
    final states = ref.read(postReturnStatesProvider);
    final previous = (liked != null || bookmarked != null)
        ? states[post.id]
        : null;
    final oldLiked = previous?.liked ?? post.isLiked;
    final count = previous?.likeCount ?? post.likeCount;
    states[post.id] = (
      liked: liked ?? oldLiked,
      bookmarked: bookmarked ?? previous?.bookmarked ?? post.isBookmarked,
      likeCount:
          count +
          (liked == null || liked == oldLiked
              ? 0
              : liked
              ? 1
              : -1),
    );
  }

  Future<void> _run(Future<void> Function() action) async {
    if (_busy) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    _actionEpoch = epoch;
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
      await showGfBottomSheet<void>(
        context,
        builder: (_) => PostHistorySheet(postId: post.id),
      );
      return;
    }
    if (action == 'edit') {
      final saved = await showGfBottomSheet<bool>(
        context,
        barrierDismissible: false,
        keyboardAware: true,
        enableDrag: false,
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
    final likeColor = (post.isLiked ? colors.error : colors.iconMuted)
        .withValues(alpha: _busy ? .38 : 1);
    final controls = <Widget>[
      if (available) ...[
        MergeSemantics(
          child: Semantics(
            label: l10n.topicLike,
            child: Tooltip(
              message: l10n.topicLike,
              excludeFromSemantics: true,
              child: TextButton(
                onPressed: _busy
                    ? null
                    : () => _run(() async {
                        await ref
                            .read(postRepositoryProvider)
                            .likePost(
                              postId: post.id,
                              action: post.isLiked ? 2 : 1,
                            );
                        _recordState(liked: !post.isLiked);
                      }),
                style: TextButton.styleFrom(
                  minimumSize: const Size(44, 44),
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  visualDensity: VisualDensity.standard,
                  foregroundColor: likeColor,
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    GfSymbol(
                      post.isLiked ? 'heart-filled' : 'heart',
                      size: _postActionIconSize,
                      color: likeColor,
                    ),
                    const SizedBox(width: 4),
                    Text(
                      formatNumber(post.likeCount),
                      style: GfTheme.typographyOf(context).caption.copyWith(
                        fontSize: 14,
                        height: 1.2,
                        color: likeColor,
                      ),
                    ),
                  ],
                ),
              ),
            ),
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
                  _recordState(bookmarked: !post.isBookmarked);
                }),
          icon: GfSymbol(
            post.isBookmarked ? 'bookmark-filled' : 'bookmark',
            size: _postActionIconSize,
            color: post.isBookmarked
                ? colors.primary.withValues(alpha: _busy ? .38 : 1)
                : null,
          ),
        ),
        if (widget.onReply != null)
          IconButton(
            tooltip: l10n.topicReply,
            onPressed: _busy ? null : widget.onReply,
            icon: const GfSymbol('corner-down-left', size: _postActionIconSize),
          ),
      ],
      PopupMenuButton<String>(
        icon: GfSymbol(
          'ellipsis',
          size: _postActionIconSize,
          color: colors.iconMuted.withValues(alpha: _busy ? .38 : 1),
        ),
        tooltip: l10n.profileMore,
        useRootNavigator: true,
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
            PopupMenuItem(value: 'unban', child: Text(l10n.topicModerateUnban)),
        ],
      ),
      if (!post.isOwnPost && available)
        IconButton(
          tooltip: l10n.topicReport,
          onPressed: _busy ? null : widget.onReport,
          icon: const GfSymbol('flag', size: _postActionIconSize),
        ),
    ];
    return IconButtonTheme(
      data: IconButtonThemeData(
        style: (IconButtonTheme.of(context).style ?? const ButtonStyle())
            .copyWith(
              minimumSize: const WidgetStatePropertyAll(Size.square(44)),
              maximumSize: const WidgetStatePropertyAll(Size.square(44)),
              padding: const WidgetStatePropertyAll(EdgeInsets.zero),
              foregroundColor: WidgetStateProperty.resolveWith(
                (states) => colors.iconMuted.withValues(
                  alpha: states.contains(WidgetState.disabled) ? .38 : 1,
                ),
              ),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              visualDensity: VisualDensity.standard,
            ),
      ),
      child: Wrap(
        alignment: WrapAlignment.start,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: controls,
      ),
    );
  }
}
