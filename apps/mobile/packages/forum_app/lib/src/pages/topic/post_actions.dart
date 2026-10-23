import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../format.dart';
import '../../server_messages.dart';
import 'post_editor.dart';
import 'post_history_sheet.dart';
import 'post_share_image.dart';

const double _postActionIconSize = 20;

class PostActions extends ConsumerStatefulWidget {
  const PostActions({
    super.key,
    required this.post,
    this.topicTitle = '',
    this.topicAvailable = true,
    required this.onChanged,
    this.onPostPatched,
    required this.onReply,
    required this.onReport,
  });
  final PostPayload post;
  final String topicTitle;
  final bool topicAvailable;
  final Future<void> Function() onChanged;

  /// 点赞/收藏只改变本楼互动状态(issue #1316):宿主原地替换该楼层,
  /// 不重载评论窗口,已加载的更早回复与滚动位置保持不变。
  final ValueChanged<PostPayload>? onPostPatched;
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
  bool get _available =>
      widget.topicAvailable && !_removed && !widget.post.isHidden;
  void _recordState({bool? liked, bool? bookmarked, int? likeCount}) {
    if (!mounted ||
        (_busy && _actionEpoch != ref.read(offlineCacheEpochProvider))) {
      return;
    }
    final post = widget.post;
    if (!_available) return;
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
          likeCount ??
          count +
              (liked == null || liked == oldLiked
                  ? 0
                  : liked
                  ? 1
                  : -1),
    );
  }

  Future<void> _run(
    Future<void> Function() action, {
    bool reload = true,
  }) async {
    if (_busy) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    _actionEpoch = epoch;
    setState(() => _busy = true);
    try {
      await action();
      if (reload && mounted && epoch == ref.read(offlineCacheEpochProvider)) {
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
    if ((action == 'share' || action == 'shareImage') && !_available) return;
    if (action == 'history') {
      await showGfBottomSheet<void>(
        context,
        builder: (_) => PostHistorySheet(postId: post.id),
      );
      return;
    }
    if (action == 'edit') {
      final saved = await showPostEditor(
        context,
        postId: post.id,
        content: post.content,
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
    if (action == 'shareImage') {
      await showPostShareImage(
        context,
        post: post,
        topicTitle: widget.topicTitle,
        baseUrl: ref.read(apiClientProvider).baseUrl,
      );
      return;
    }
    if (action == 'report') {
      widget.onReport();
      return;
    }
    final confirm = await showGfAlertDialog<bool>(
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
    final available = _available;
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
                        final bool next = !post.isLiked;
                        final int nextCount = next
                            ? post.likeCount + 1
                            : (post.likeCount > 0 ? post.likeCount - 1 : 0);
                        await ref
                            .read(postRepositoryProvider)
                            .likePost(
                              postId: post.id,
                              action: post.isLiked ? 2 : 1,
                            );
                        _recordState(liked: next, likeCount: nextCount);
                        if (_actionEpoch ==
                            ref.read(offlineCacheEpochProvider)) {
                          widget.onPostPatched?.call(
                            post.copyWith(isLiked: next, likeCount: nextCount),
                          );
                        }
                      }, reload: false),
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
                    GfText(
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
                  final bool next = !post.isBookmarked;
                  await ref
                      .read(postRepositoryProvider)
                      .bookmarkPost(
                        postId: post.id,
                        action: post.isBookmarked ? 2 : 1,
                      );
                  _recordState(bookmarked: next);
                  if (_actionEpoch == ref.read(offlineCacheEpochProvider)) {
                    widget.onPostPatched?.call(
                      post.copyWith(isBookmarked: next),
                    );
                  }
                }, reload: false),
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
      GfPopupMenuButton<String>(
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
          if (post.isOwnPost &&
              widget.topicAvailable &&
              !_removed &&
              (!post.isHidden || post.processStatus == 2))
            GfPopupMenuItem(value: 'edit', child: GfText(l10n.commonEdit)),
          if (post.isOwnPost && available)
            GfPopupMenuItem(
              destructive: true,
              value: 'delete',
              child: GfText(l10n.contentDelete),
            ),
          GfPopupMenuItem(value: 'history', child: GfText(l10n.topicHistory)),
          if (available) ...[
            GfPopupMenuItem(value: 'share', child: GfText(l10n.topicShare)),
            GfPopupMenuItem(
              value: 'shareImage',
              child: GfText(l10n.topicShareImage),
            ),
          ],
          if (!post.isOwnPost && available)
            GfPopupMenuItem(value: 'report', child: GfText(l10n.topicReport)),
          if (post.canModerate && post.processStatus == 0)
            GfPopupMenuItem(
              destructive: true,
              value: 'ban',
              child: GfText(l10n.topicModerateBan),
            ),
          if (post.canModerate && post.processStatus == 1)
            GfPopupMenuItem(
              value: 'unban',
              child: GfText(l10n.topicModerateUnban),
            ),
        ],
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
