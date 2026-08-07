import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';

/// 话题详情页(web TopicPage.vue 的移动端形态):
/// 话题信息 + 帖子流(分页)+ markdown 渲染 + 图片查看器 + 互动(点赞/收藏/评论)。
class TopicPage extends ConsumerStatefulWidget {
  const TopicPage({super.key, required this.topicId});

  final int topicId;

  @override
  ConsumerState<TopicPage> createState() => _TopicPageState();
}

class _TopicPageState extends ConsumerState<TopicPage> {
  AsyncValue<TopicDetailProps> _page = const AsyncValue.loading();
  bool _loadingMore = false;
  final List<PostPayload> _posts = [];

  // 互动状态(乐观更新)。
  bool _liked = false;
  bool _bookmarked = false;
  int _likeCount = 0;

  // 评论输入。
  final TextEditingController _replyController = TextEditingController();
  bool _replying = false;
  int _replyToPostId = 0;

  final FocusNode _replyFocus = FocusNode();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _replyController.dispose();
    _replyFocus.dispose();
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .topicDetail(widget.topicId);
      final props = parsePageProps<TopicDetailProps>(payload);
      if (props == null) {
        setState(
          () => _page = AsyncValue.error(
            AppLocalizations.of(context).commonParseFailed,
            StackTrace.current,
          ),
        );
        return;
      }
      // 写入 drift 离线缓存(供断网时回读);缓存失败静默降级。
      await _cachePut(widget.topicId, payload.toJson());
      setState(() {
        _page = AsyncValue.data(props);
        _posts.clear();
        _posts.addAll(props.postStream.posts);
        _liked = props.topic.isLiked;
        _bookmarked = props.topic.isBookmarked;
        _likeCount = props.topic.likeCount;
      });
    } catch (e, st) {
      // 网络失败:回退 drift 离线缓存(已浏览话题离线可读)。
      final PagePayload? cached = await _cacheGet(widget.topicId);
      final props = cached == null
          ? null
          : parsePageProps<TopicDetailProps>(cached);
      if (props != null) {
        setState(() {
          _page = AsyncValue.data(props);
          _posts.clear();
          _posts.addAll(props.postStream.posts);
          _liked = props.topic.isLiked;
          _bookmarked = props.topic.isBookmarked;
          _likeCount = props.topic.likeCount;
        });
      } else {
        setState(() => _page = AsyncValue.error(e, st));
      }
    }
  }

  /// 写缓存;失败静默(缓存不可用不影响页面加载)。
  Future<void> _cachePut(int topicId, Map<String, dynamic> json) async {
    try {
      await ref.read(offlineTopicCacheProvider).put(topicId, json);
    } catch (_) {
      // 缓存不可用时忽略(页面加载不依赖缓存)。
    }
  }

  /// 读缓存;失败返回 null。
  Future<PagePayload?> _cacheGet(int topicId) async {
    try {
      return await ref.read(offlineTopicCacheProvider).get(topicId);
    } catch (_) {
      return null;
    }
  }

  Future<void> _loadMore() async {
    final props = _page.value;
    if (props == null || !props.postStream.hasAfter || _loadingMore) return;
    setState(() => _loadingMore = true);
    try {
      final window = await ref
          .read(topicRepositoryProvider)
          .getPostWindow(
            topicId: widget.topicId,
            afterPostNo: props.postStream.afterPostNo,
          );
      if (window.posts.isNotEmpty) {
        setState(() => _posts.addAll(window.posts));
      }
    } catch (_) {
      // 加载更多失败静默。
    } finally {
      setState(() => _loadingMore = false);
    }
  }

  Future<void> _toggleLike() async {
    final topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_liked;
    setState(() {
      _liked = target;
      _likeCount += target ? 1 : -1;
    });
    try {
      await ref
          .read(topicRepositoryProvider)
          .likeTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      // 回滚。
      setState(() {
        _liked = !target;
        _likeCount += target ? -1 : 1;
      });
    }
  }

  Future<void> _toggleBookmark() async {
    final topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_bookmarked;
    setState(() => _bookmarked = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .bookmarkTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      setState(() => _bookmarked = !target);
    }
  }

  Future<void> _submitReply() async {
    final String content = _replyController.text.trim();
    if (content.isEmpty) return;
    setState(() => _replying = true);
    try {
      await ref
          .read(postRepositoryProvider)
          .createPost(
            topicId: widget.topicId,
            content: content,
            replyToPostId: _replyToPostId,
          );
      _replyController.clear();
      _replyToPostId = 0;
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(AppLocalizations.of(context).topicReplySuccess),
          ),
        );
      }
      // 局部刷新:回复成功后静默重载(不置 loading、不清空列表,
      // 保留滚动位置,对齐 web 乐观追加语义)。
      if (mounted) {
        await _load(silent: true);
      }
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(e.messageKey)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(AppLocalizations.of(context).topicReplyFailed('$e')),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _replying = false);
    }
  }

  Future<void> _reportPost(PostPayload post) async {
    if (!mounted) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final reason = await showDialog<String>(
      context: context,
      builder: (ctx) {
        final ctrl = TextEditingController();
        return AlertDialog(
          title: Text(l10n.topicReport),
          content: TextField(
            controller: ctrl,
            maxLines: 3,
            decoration: InputDecoration(
              hintText: l10n.topicReportHint,
              border: const OutlineInputBorder(),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: Text(l10n.commonCancel),
            ),
            TextButton(
              onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
              child: Text(l10n.topicReportSubmit),
            ),
          ],
        );
      },
    );
    if (reason == null || reason.isEmpty) return;
    try {
      await ref
          .read(postRepositoryProvider)
          .report(
            targetType: 'post',
            targetId: post.id,
            reason: reason,
            note: '',
          );
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.topicReportSubmitted)));
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(l10n.topicReportFailed('$e'))));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.topicTitle)),
      body: _page.when(
        loading: () => const GfLoading(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          return Column(
            children: [
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () => _load(silent: true),
                  child: ListView.separated(
                    padding: const EdgeInsets.only(bottom: 12),
                    itemCount: _posts.length + 2,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (context, index) {
                      if (index == 0) {
                        return _TopicHeader(
                          topic: props.topic,
                          liked: _liked,
                          bookmarked: _bookmarked,
                          likeCount: _likeCount,
                          onLike: _toggleLike,
                          onBookmark: _toggleBookmark,
                        );
                      }
                      if (index == _posts.length + 1) {
                        return GfListFooter(
                          loading: _loadingMore,
                          hasMore: props.postStream.hasAfter,
                          onLoadMore: _loadMore,
                        );
                      }
                      return _PostCard(
                        post: _posts[index - 1],
                        onReply: () {
                          _replyToPostId = _posts[index - 1].id;
                          _replyController.text =
                              '@${_posts[index - 1].author.username} ';
                          _replyController.selection = TextSelection.collapsed(
                            offset: _replyController.text.length,
                          );
                          FocusScope.of(context).requestFocus(_replyFocus);
                        },
                        onReport: () => _reportPost(_posts[index - 1]),
                      );
                    },
                  ),
                ),
              ),
              // 底部回复输入条。
              SafeArea(
                top: false,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    color: colors.base100,
                    border: Border(
                      top: BorderSide(color: colors.line, width: 0.5),
                    ),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _replyController,
                          focusNode: _replyFocus,
                          minLines: 1,
                          maxLines: 4,
                          decoration: InputDecoration(
                            hintText: _replyToPostId != 0
                                ? l10n.topicReplying
                                : l10n.topicReplyHint,
                            isDense: true,
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(20),
                            ),
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 10,
                            ),
                          ),
                          onTapOutside: (_) => FocusScope.of(context).unfocus(),
                        ),
                      ),
                      const SizedBox(width: 8),
                      GfButton(
                        label: l10n.commonSend,
                        variant: GfButtonVariant.primary,
                        size: GfButtonSize.small,
                        loading: _replying,
                        onPressed: _replying ? null : _submitReply,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

class _TopicHeader extends StatelessWidget {
  const _TopicHeader({
    required this.topic,
    required this.liked,
    required this.bookmarked,
    required this.likeCount,
    required this.onLike,
    required this.onBookmark,
  });

  final TopicDetailPayload topic;
  final bool liked;
  final bool bookmarked;
  final int likeCount;
  final VoidCallback onLike;
  final VoidCallback onBookmark;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            topic.title,
            style: GfTheme.typographyOf(context).title1,
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              CircleAvatar(
                radius: 12,
                backgroundImage: NetworkImage(topic.author.avatarUrl),
                onBackgroundImageError: (_, _) {},
              ),
              const SizedBox(width: 8),
              Text(
                topic.author.nickname ?? topic.author.username,
                style: GfTheme.typographyOf(context).bodyStrong,
              ),
              const SizedBox(width: 12),
              Text(
                timeAgo(topic.createdAt),
                style: GfTheme.typographyOf(context).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
            ],
          ),
          Wrap(
            spacing: 6,
            children: [
              for (final cat in topic.categories)
                GfChip(label: cat.name, color: colorFromHex(cat.color)),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              _MetaItem(
                icon: Icons.visibility_outlined,
                value: formatNumber(topic.viewCount),
              ),
              const SizedBox(width: 16),
              _MetaItem(
                icon: Icons.chat_bubble_outline,
                value: formatNumber(topic.replyCount),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onLike,
                borderRadius: BorderRadius.circular(16),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  child: _MetaItem(
                    icon: liked ? Icons.favorite : Icons.favorite_border,
                    value: formatNumber(likeCount),
                    color: liked ? colors.error : null,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              InkWell(
                onTap: onBookmark,
                borderRadius: BorderRadius.circular(16),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  child: Icon(
                    bookmarked ? Icons.bookmark : Icons.bookmark_border,
                    size: 18,
                    color: bookmarked
                        ? colors.primary
                        : GfTheme.colorsOf(context).iconMuted,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({required this.icon, required this.value, this.color});

  final IconData icon;
  final String value;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(
          icon,
          size: 15,
          color: color ?? GfTheme.colorsOf(context).iconMuted,
        ),
        const SizedBox(width: 4),
        Text(
          value,
          style: GfTheme.typographyOf(context).caption.copyWith(color: color ?? GfTheme.colorsOf(context).iconMuted),
        ),
      ],
    );
  }
}

class _PostCard extends StatelessWidget {
  const _PostCard({
    required this.post,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              CircleAvatar(
                radius: 12,
                backgroundImage: NetworkImage(post.author.avatarUrl),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  post.author.nickname ?? post.author.username,
                  style: GfTheme.typographyOf(context).bodyStrong,
                ),
              ),
              if (post.postNo > 0)
                Text(
                  '#${post.postNo}',
                  style: GfTheme.typographyOf(context).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
                ),
            ],
          ),
          if (post.replyToUsername != null) ...[
            const SizedBox(height: 6),
            Text(
              '${l10n.topicReply} @${post.replyToUsername}',
              style: GfTheme.typographyOf(context).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
            ),
          ],
          const SizedBox(height: 10),
          GfMarkdownView(data: post.content),
          const SizedBox(height: 10),
          Row(
            children: [
              Text(
                timeAgo(post.createdAt),
                style: GfTheme.typographyOf(context).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
              const Spacer(),
              Icon(
                post.isLiked ? Icons.favorite : Icons.favorite_border,
                size: 16,
                color: post.isLiked
                    ? colors.error
                    : GfTheme.colorsOf(context).iconMuted,
              ),
              const SizedBox(width: 4),
              Text(
                '${post.likeCount}',
                style: GfTheme.typographyOf(context).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onReply,
                child: Icon(
                  Icons.reply_outlined,
                  size: 16,
                  color: GfTheme.colorsOf(context).iconMuted,
                ),
              ),
              const SizedBox(width: 16),
              InkWell(
                onTap: onReport,
                child: Icon(
                  Icons.flag_outlined,
                  size: 16,
                  color: GfTheme.colorsOf(context).iconMuted,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
