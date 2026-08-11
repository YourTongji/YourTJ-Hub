import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../images/image_upload.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';
import '../../widgets/skeletons.dart';

/// 话题详情页(web TopicPage.vue 的移动端形态):
/// 话题信息 + 帖子流(分页)+ markdown 渲染 + 图片查看器 + 互动(点赞/收藏/关注/评论)。
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
  int? _afterPostNo;
  bool _hasMorePosts = false;

  // 互动状态(乐观更新)。
  bool _liked = false;
  bool _bookmarked = false;
  bool _watched = false;
  int _likeCount = 0;

  // 轻量 Markdown 回复输入。
  final TextEditingController _replyController = TextEditingController();
  final FocusNode _replyFocus = FocusNode();
  bool _replying = false;
  bool _uploadingReplyImage = false;
  int _replyToPostId = 0;
  String? _replyImageUrl;
  String? _replyTargetName;
  String? _replyMentionPrefix;

  // 浮动层状态(web TopicFloatingControls / PostComposer 语义)。
  bool _composerOpen = false;
  bool _railOpen = false;
  final Set<int> _expandedReplyGroups = <int>{};

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
    // 记录发起时的缓存世代;401/登出/换账号后世代自增,返回时丢弃旧会话数据。
    final int epoch = ref.read(offlineCacheEpochProvider);
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .topicDetail(widget.topicId);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
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
      // 仅当前世代允许写入,避免旧会话在途响应写回上一账号数据。
      if (epoch == ref.read(offlineCacheEpochProvider)) {
        await _cachePut(widget.topicId, payload.toJson());
      }
      // 写入期间会话可能已切换,再次校验世代再更新 UI。
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _page = AsyncValue.data(props);
        _posts.clear();
        _posts.addAll(props.postStream.posts);
        _afterPostNo = props.postStream.afterPostNo;
        _hasMorePosts = props.postStream.hasAfter;
        _liked = props.topic.isLiked;
        _bookmarked = props.topic.isBookmarked;
        _watched = props.topic.isWatched;
        _likeCount = props.topic.likeCount;
      });
    } catch (e, st) {
      // 网络失败:回退 drift 离线缓存(已浏览话题离线可读)。
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      // 无会话令牌(如 401 后进程被杀重启)时不得回退上一账号残留缓存。
      if (!await hasSessionToken(ref.read(tokenStorageProvider))) {
        // 静默刷新失败时保留当前内容,不打断阅读。
        if (!silent) setState(() => _page = AsyncValue.error(e, st));
        return;
      }
      final PagePayload? cached = await _cacheGet(widget.topicId);
      // 读缓存期间会话可能已切换,再次校验世代再更新 UI。
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      final props = cached == null
          ? null
          : parsePageProps<TopicDetailProps>(cached);
      if (props != null) {
        setState(() {
          _page = AsyncValue.data(props);
          _posts.clear();
          _posts.addAll(props.postStream.posts);
          _afterPostNo = props.postStream.afterPostNo;
          _hasMorePosts = props.postStream.hasAfter;
          _liked = props.topic.isLiked;
          _bookmarked = props.topic.isBookmarked;
          _watched = props.topic.isWatched;
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
    if (!_hasMorePosts || _loadingMore) return;
    setState(() => _loadingMore = true);
    try {
      final int? previousAfterPostNo = _afterPostNo;
      final PostWindowPayload window = await ref
          .read(topicRepositoryProvider)
          .getPostWindow(topicId: widget.topicId, afterPostNo: _afterPostNo);
      if (!mounted) return;
      final Set<int> existingIds = _posts
          .map((PostPayload post) => post.id)
          .toSet();
      setState(() {
        _posts.addAll(
          window.posts.where((PostPayload post) => existingIds.add(post.id)),
        );
        final int? nextAfterPostNo = window.afterPostNo ?? previousAfterPostNo;
        _afterPostNo = nextAfterPostNo;
        _hasMorePosts =
            window.posts.isNotEmpty &&
            window.hasAfter &&
            nextAfterPostNo != null &&
            (previousAfterPostNo == null ||
                nextAfterPostNo > previousAfterPostNo);
      });
    } catch (_) {
      // 加载更多失败静默。
    } finally {
      if (mounted) setState(() => _loadingMore = false);
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
      if (!mounted) return;
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
      if (!mounted) return;
    } catch (_) {
      setState(() => _bookmarked = !target);
    }
  }

  Future<void> _toggleWatch() async {
    final TopicDetailPayload? topic = _page.value?.topic;
    if (topic == null) return;
    final bool target = !_watched;
    setState(() => _watched = target);
    try {
      await ref
          .read(topicRepositoryProvider)
          .watchTopic(topicId: topic.id, action: target ? 1 : 2);
    } catch (_) {
      if (mounted) setState(() => _watched = !target);
    }
  }

  void _openComposer({PostPayload? replyTo}) {
    if (replyTo != null) {
      final String mention = '@${replyTo.author.username} ';
      _replyToPostId = replyTo.id;
      _replyMentionPrefix = mention;
      _replyController.text = mention;
      _replyTargetName = replyTo.author.nickname ?? replyTo.author.username;
      _replyController.selection = TextSelection.collapsed(
        offset: _replyController.text.length,
      );
    } else if (_replyController.text.trim().isEmpty) {
      _replyController.clear();
    }
    setState(() {
      _composerOpen = true;
      _railOpen = false;
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _replyFocus.requestFocus();
    });
  }

  void _removeGeneratedMention() {
    final String? mention = _replyMentionPrefix;
    if (mention == null || !_replyController.text.startsWith(mention)) return;

    final String updated = _replyController.text.substring(mention.length);
    _replyController.value = TextEditingValue(
      text: updated,
      selection: TextSelection.collapsed(offset: updated.length),
    );
  }

  void _clearReplyTarget() {
    _removeGeneratedMention();
    _replyToPostId = 0;
    _replyTargetName = null;
    _replyMentionPrefix = null;
  }

  void _closeComposer() {
    _replyFocus.unfocus();
    _clearReplyTarget();
    setState(() => _composerOpen = false);
  }

  void _insertReplyImage(String url) {
    final String oldToken = _replyImageUrl == null
        ? ''
        : '![image]($_replyImageUrl)';
    if (oldToken.isNotEmpty && _replyController.text.contains(oldToken)) {
      _replyController.text = _replyController.text.replaceFirst(oldToken, '');
    }

    final String text = _replyController.text;
    final int rawOffset = _replyController.selection.baseOffset;
    final int offset = rawOffset.clamp(0, text.length);
    final String before = text.substring(0, offset);
    final String after = text.substring(offset);
    final String prefix = before.isEmpty || before.endsWith('\n') ? '' : '\n';
    final String suffix = after.isEmpty || after.startsWith('\n') ? '' : '\n';
    final String insertion = '$prefix![image]($url)$suffix';
    _replyController.value = _replyController.value.copyWith(
      text: '$before$insertion$after',
      selection: TextSelection.collapsed(offset: offset + insertion.length),
      composing: TextRange.empty,
    );
    setState(() => _replyImageUrl = url);
  }

  void _removeReplyImage() {
    final String? url = _replyImageUrl;
    if (url == null) return;
    final String token = '![image]($url)';
    final TextEditingValue value = _replyController.value;
    final int start = value.text.indexOf(token);
    if (start < 0) return;
    final int end = start + token.length;

    int adjustOffset(int offset) {
      if (offset < 0 || offset <= start) return offset;
      if (offset <= end) return start;
      return offset - token.length;
    }

    _replyController.value = value.copyWith(
      text: value.text.replaceRange(start, end, ''),
      selection: TextSelection(
        baseOffset: adjustOffset(value.selection.baseOffset),
        extentOffset: adjustOffset(value.selection.extentOffset),
        affinity: value.selection.affinity,
        isDirectional: value.selection.isDirectional,
      ),
      composing: TextRange.empty,
    );
    setState(() => _replyImageUrl = null);
  }

  Future<void> _pickReplyImage() async {
    if (_uploadingReplyImage) return;
    setState(() => _uploadingReplyImage = true);
    try {
      final String? url = await pickAndUploadImage(ref: ref);
      if (url != null && mounted) _insertReplyImage(url);
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).publishImageFailed('$e'),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _uploadingReplyImage = false);
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
      if (!mounted) return;
      _clearReplyTarget();
      _replyController.clear();
      setState(() {
        _replyImageUrl = null;
        _composerOpen = false;
      });
      if (mounted) {
        showGfToast(context, AppLocalizations.of(context).topicReplySuccess);
      }
      // 局部刷新:回复成功后静默重载(不置 loading、不清空列表,
      // 保留滚动位置,对齐 web 乐观追加语义)。
      if (mounted) {
        await _load(silent: true);
      }
    } on ApiException catch (e) {
      if (mounted) {
        showGfToast(context, e.messageKey, error: true);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).topicReplyFailed('$e'),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _replying = false);
    }
  }

  Future<void> _reportPost(PostPayload post) async {
    await _reportTarget(targetType: 'post', targetId: post.id);
  }

  Future<void> _reportTopic(TopicDetailPayload topic) async {
    await _reportTarget(targetType: 'topic', targetId: topic.id);
  }

  Future<void> _reportTarget({
    required String targetType,
    required int targetId,
  }) async {
    if (!mounted) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final reason = await showGfModal<String>(
      context,
      builder: (ctx) {
        final ctrl = TextEditingController();
        return Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Text(l10n.topicReport, style: GfTheme.typographyOf(ctx).title3),
            const SizedBox(height: 16),
            GfInput(
              controller: ctrl,
              maxLines: 3,
              hintText: l10n.topicReportHint,
            ),
            const SizedBox(height: 20),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: <Widget>[
                GfButton(
                  label: l10n.commonCancel,
                  variant: GfButtonVariant.ghost,
                  onPressed: () => Navigator.pop(ctx),
                ),
                const SizedBox(width: 8),
                GfButton(
                  label: l10n.topicReportSubmit,
                  onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
                ),
              ],
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
            targetType: targetType,
            targetId: targetId,
            reason: reason,
            note: '',
          );
      if (mounted) {
        showGfToast(context, l10n.topicReportSubmitted);
      }
    } catch (e) {
      if (mounted) {
        showGfToast(context, l10n.topicReportFailed('$e'), error: true);
      }
    }
  }

  List<_PostGroup> _postGroups({PostPayload? mainPost}) {
    final int? mainPostId = mainPost?.id;
    final Map<int, PostPayload> byId = <int, PostPayload>{
      for (final PostPayload post in _posts) post.id: post,
    };
    final List<PostPayload> roots = <PostPayload>[];
    final Map<int, List<PostPayload>> repliesByRoot =
        <int, List<PostPayload>>{};

    PostPayload resolveRoot(PostPayload post) {
      final Set<int> visited = <int>{};
      PostPayload cursor = post;
      while (cursor.replyToPostId != null && cursor.replyToPostId! > 0) {
        // 直接回复主帖的帖子本身就是根,不再向上追溯。
        if (cursor.replyToPostId == mainPostId) break;
        if (!visited.add(cursor.id)) break;
        final PostPayload? parent = byId[cursor.replyToPostId!];
        if (parent == null) break;
        cursor = parent;
      }
      return cursor;
    }

    for (final PostPayload post in _posts) {
      // 主帖由 _TopicHeader 单独渲染,不进入回复分组。
      if (post.id == mainPostId) continue;
      final PostPayload? parent = post.replyToPostId == null
          ? null
          : byId[post.replyToPostId!];
      if (parent == null || parent.id == mainPostId) {
        roots.add(post);
        continue;
      }
      final PostPayload root = resolveRoot(post);
      repliesByRoot.putIfAbsent(root.id, () => <PostPayload>[]).add(post);
    }
    roots.sort((a, b) => a.postNo.compareTo(b.postNo));
    return <_PostGroup>[
      for (final PostPayload root in roots)
        _PostGroup(
          root: root,
          replies: (repliesByRoot[root.id] ?? <PostPayload>[])
            ..sort((a, b) => a.postNo.compareTo(b.postNo)),
        ),
    ];
  }

  void _goBack() {
    if (context.canPop()) {
      context.pop();
      return;
    }
    context.go('/');
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);

    final String appBarTitle = _page.value?.topic.title ?? l10n.topicTitle;
    return Scaffold(
      appBar: GfAppBar(
        leading: GfIconButton(
          icon: Icons.arrow_back,
          tooltip: l10n.commonBack,
          size: 44,
          onPressed: _goBack,
        ),
        title: Text(appBarTitle, maxLines: 1, overflow: TextOverflow.ellipsis),
      ),
      body: _page.when(
        loading: () => const GfTopicDetailSkeleton(),
        error: (e, _) => GfErrorRetry(message: '$e', onRetry: _load),
        data: (props) {
          final PostPayload? mainPost = _mainPost(_posts);
          final List<_PostGroup> groups = _postGroups(mainPost: mainPost);

          return Stack(
            children: <Widget>[
              Positioned.fill(
                child: GfScrollToTop(
                  semanticLabel: l10n.commonBackToTop,
                  showButton: !_composerOpen,
                  threshold: 360,
                  bottomInset: 84,
                  builder: (context, scrollController) {
                    return RefreshIndicator(
                      onRefresh: () => _load(silent: true),
                      child: CustomScrollView(
                        controller: scrollController,
                        physics: const AlwaysScrollableScrollPhysics(),
                        slivers: <Widget>[
                          SliverToBoxAdapter(
                            child: _TopicHeader(
                              topic: props.topic,
                              mainPost: mainPost,
                              liked: _liked,
                              bookmarked: _bookmarked,
                              watched: _watched,
                              likeCount: _likeCount,
                              canReportTopic: !props.permissions.isOwnTopic,
                              onLike: _toggleLike,
                              onBookmark: _toggleBookmark,
                              onWatch: _toggleWatch,
                              onReportTopic: () => _reportTopic(props.topic),
                            ),
                          ),
                          const SliverToBoxAdapter(child: GfDivider()),
                          SliverToBoxAdapter(
                            child: _ReplySectionHeader(
                              count: props.topic.replyCount,
                            ),
                          ),
                          if (groups.isEmpty)
                            SliverToBoxAdapter(
                              child: GfEmpty(
                                icon: Icons.forum_outlined,
                                message: l10n.topicReplies(0),
                                description: l10n.topicReplyHint,
                              ),
                            )
                          else
                            SliverList.builder(
                              itemCount: groups.length,
                              itemBuilder: (BuildContext context, int index) {
                                final _PostGroup group = groups[index];
                                return RepaintBoundary(
                                  child: Column(
                                    children: <Widget>[
                                      _PostGroupView(
                                        group: group,
                                        expanded: _expandedReplyGroups.contains(
                                          group.root.id,
                                        ),
                                        onToggleReplies: () => setState(() {
                                          final int id = group.root.id;
                                          if (!_expandedReplyGroups.add(id)) {
                                            _expandedReplyGroups.remove(id);
                                          }
                                        }),
                                        onReply: (PostPayload post) =>
                                            _openComposer(replyTo: post),
                                        onReport: _reportPost,
                                      ),
                                      if (index < groups.length - 1)
                                        const GfDivider(),
                                    ],
                                  ),
                                );
                              },
                            ),
                          SliverToBoxAdapter(
                            child: GfListFooter(
                              loading: _loadingMore,
                              hasMore: _hasMorePosts,
                              onLoadMore: _loadMore,
                            ),
                          ),
                          const SliverToBoxAdapter(
                            child: SizedBox(height: 104),
                          ),
                        ],
                      ),
                    );
                  },
                ),
              ),
              Positioned(
                left: 12,
                right: 12,
                bottom: 12,
                child: SafeArea(
                  top: false,
                  child: Center(
                    child: _composerOpen
                        ? ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 560),
                            child: ValueListenableBuilder<TextEditingValue>(
                              valueListenable: _replyController,
                              builder: (context, value, _) {
                                return GfPostComposer(
                                  controller: _replyController,
                                  focusNode: _replyFocus,
                                  targetName: _replyTargetName,
                                  targetLabel: _replyTargetName == null
                                      ? null
                                      : l10n.topicReplyTarget(
                                          _replyTargetName!,
                                        ),
                                  onCloseTarget: () {
                                    _clearReplyTarget();
                                    setState(() {});
                                  },
                                  onPickImage: _pickReplyImage,
                                  imageTooltip: l10n.publishToolImage,
                                  imageUrl: _replyImageUrl == null
                                      ? null
                                      : resolveApiAssetUrl(_replyImageUrl!),
                                  onRemoveImage: _removeReplyImage,
                                  removeImageTooltip: l10n.publishRemoveImage,
                                  uploading: _uploadingReplyImage,
                                  publishing: _replying,
                                  canPublish: value.text.trim().isNotEmpty,
                                  publishLabel: l10n.commonSend,
                                  hintText: l10n.topicReplyHint,
                                  onPublish: _submitReply,
                                  toolbar: Align(
                                    alignment: Alignment.centerRight,
                                    child: GfIconButton(
                                      icon: Icons.keyboard_arrow_down_rounded,
                                      tooltip: l10n.commonCancel,
                                      size: 44,
                                      onPressed: _closeComposer,
                                    ),
                                  ),
                                );
                              },
                            ),
                          )
                        : GfFloatingControls(
                            actions: <GfTopicAction>[
                              GfTopicAction(
                                icon: Icons.favorite_border,
                                active: _liked,
                                activeColor: colors.error,
                                onTap: _toggleLike,
                              ),
                              GfTopicAction(
                                icon: Icons.bookmark_border,
                                active: _bookmarked,
                                activeColor: colors.primary,
                                onTap: _toggleBookmark,
                              ),
                              GfTopicAction(
                                icon: _watched
                                    ? Icons.notifications
                                    : Icons.notifications_none,
                                active: _watched,
                                activeColor: colors.success,
                                onTap: _toggleWatch,
                              ),
                            ],
                            onOpenReply: () => _openComposer(),
                            currentNo: mainPost?.postNo ?? 1,
                            maxNo: props.postStream.maxPostNo,
                            onFloorTap: () =>
                                setState(() => _railOpen = !_railOpen),
                          ),
                  ),
                ),
              ),
              if (_railOpen && !_composerOpen)
                Positioned(
                  left: 16,
                  right: 16,
                  bottom: 76,
                  child: SafeArea(
                    top: false,
                    child: GfFloatingSurface(
                      padding: const EdgeInsets.all(8),
                      child: GfPostPositionRail(
                        current: mainPost?.postNo ?? 1,
                        max: props.postStream.maxPostNo,
                        onSelect: (floor) {
                          setState(() => _railOpen = false);
                          showGfToast(context, l10n.topicFloorSelected(floor));
                        },
                        onEarliest: () => _load(silent: true),
                        onLatest: _loadMore,
                      ),
                    ),
                  ),
                ),
            ],
          );
        },
      ),
    );
  }

  PostPayload? _mainPost(List<PostPayload> posts) {
    for (final PostPayload post in posts) {
      if (post.postNo == 1) return post;
    }
    return null;
  }
}

class _TopicHeader extends StatelessWidget {
  const _TopicHeader({
    required this.topic,
    required this.mainPost,
    required this.liked,
    required this.bookmarked,
    required this.watched,
    required this.likeCount,
    required this.canReportTopic,
    required this.onLike,
    required this.onBookmark,
    required this.onWatch,
    required this.onReportTopic,
  });

  final TopicDetailPayload topic;
  final PostPayload? mainPost;
  final bool liked;
  final bool bookmarked;
  final bool watched;
  final int likeCount;
  final bool canReportTopic;
  final VoidCallback onLike;
  final VoidCallback onBookmark;
  final VoidCallback onWatch;
  final VoidCallback onReportTopic;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String authorName = topic.author.nickname ?? topic.author.username;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              GfAvatar(
                src: resolveApiAssetUrl(topic.author.avatarUrl),
                size: 40,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      authorName,
                      style: GfTheme.typographyOf(context).bodyStrong,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '@${topic.author.username} · ${timeAgo(topic.createdAt, l10n: l10n)}',
                      style: GfTheme.typographyOf(
                        context,
                      ).caption.copyWith(color: colors.iconMuted),
                    ),
                  ],
                ),
              ),
              GfIconButton(
                icon: watched ? Icons.notifications : Icons.notifications_none,
                size: 44,
                iconSize: 20,
                tooltip: watched ? l10n.topicUnwatch : l10n.topicWatch,
                onPressed: onWatch,
              ),
              if (canReportTopic) ...<Widget>[
                const SizedBox(width: 4),
                GfIconButton(
                  icon: Icons.flag_outlined,
                  size: 44,
                  iconSize: 20,
                  tooltip: l10n.topicReport,
                  onPressed: onReportTopic,
                ),
              ],
            ],
          ),
          if (topic.categories.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: <Widget>[
                for (final CategoryBriefPayload category in topic.categories)
                  GfChip(
                    label: category.name,
                    color: colorFromHex(category.color),
                  ),
              ],
            ),
          ],
          const SizedBox(height: 14),
          Text(topic.title, style: GfTheme.typographyOf(context).title1),
          if (mainPost != null) ...<Widget>[
            const SizedBox(height: 16),
            GfMarkdownView(data: mainPost!.content),
          ] else if (topic.description.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            Text(topic.description, style: GfTheme.typographyOf(context).body),
          ],
          const SizedBox(height: 18),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
            decoration: BoxDecoration(
              color: colors.base200,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: colors.line),
            ),
            child: Row(
              children: <Widget>[
                Expanded(
                  child: _MetaItem(
                    icon: Icons.visibility_outlined,
                    value: formatNumber(topic.viewCount),
                  ),
                ),
                Expanded(
                  child: _MetaItem(
                    icon: Icons.chat_bubble_outline,
                    value: formatNumber(topic.replyCount),
                  ),
                ),
                Expanded(
                  child: _MetaItem(
                    icon: liked ? Icons.favorite : Icons.favorite_border,
                    value: formatNumber(likeCount),
                    color: liked ? colors.error : null,
                    onTap: onLike,
                  ),
                ),
                Expanded(
                  child: _MetaItem(
                    icon: bookmarked ? Icons.bookmark : Icons.bookmark_border,
                    value: '',
                    color: bookmarked ? colors.primary : null,
                    onTap: onBookmark,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ReplySectionHeader extends StatelessWidget {
  const _ReplySectionHeader({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 18, 16, 10),
      child: Row(
        children: <Widget>[
          Text(
            AppLocalizations.of(context).topicReplies(count),
            style: GfTheme.typographyOf(context).title3,
          ),
          const Spacer(),
          Icon(
            Icons.forum_outlined,
            size: 18,
            color: GfTheme.colorsOf(context).iconMuted,
          ),
        ],
      ),
    );
  }
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({
    required this.icon,
    required this.value,
    this.color,
    this.onTap,
  });

  final IconData icon;
  final String value;
  final Color? color;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final Color foreground = color ?? GfTheme.colorsOf(context).iconMuted;
    final Widget content = Row(
      mainAxisAlignment: MainAxisAlignment.center,
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Icon(icon, size: 17, color: foreground),
        if (value.isNotEmpty) ...<Widget>[
          const SizedBox(width: 4),
          Text(
            value,
            style: GfTheme.typographyOf(
              context,
            ).caption.copyWith(color: foreground),
          ),
        ],
      ],
    );

    if (onTap == null) {
      return ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 44),
        child: Center(child: content),
      );
    }
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 44),
        child: Center(child: content),
      ),
    );
  }
}

class _PostGroup {
  const _PostGroup({required this.root, required this.replies});

  final PostPayload root;
  final List<PostPayload> replies;
}

class _PostGroupView extends StatelessWidget {
  const _PostGroupView({
    required this.group,
    required this.expanded,
    required this.onToggleReplies,
    required this.onReply,
    required this.onReport,
  });

  static const int _previewCount = 3;

  final _PostGroup group;
  final bool expanded;
  final VoidCallback onToggleReplies;
  final ValueChanged<PostPayload> onReply;
  final ValueChanged<PostPayload> onReport;

  @override
  Widget build(BuildContext context) {
    final List<PostPayload> replies = expanded
        ? group.replies
        : group.replies.take(_previewCount).toList();
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 16, 12, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          _PostCard(
            post: group.root,
            onReply: () => onReply(group.root),
            onReport: () => onReport(group.root),
          ),
          if (replies.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              decoration: BoxDecoration(
                color: GfTheme.colorsOf(context).base200.withValues(alpha: 0.7),
                borderRadius: BorderRadius.circular(
                  GfTheme.radiiOf(context).field,
                ),
                border: Border.all(color: GfTheme.colorsOf(context).line),
              ),
              child: Column(
                children: <Widget>[
                  for (
                    int index = 0;
                    index < replies.length;
                    index++
                  ) ...<Widget>[
                    if (index > 0) const GfDivider(),
                    _NestedPostCard(
                      post: replies[index],
                      onReply: () => onReply(replies[index]),
                      onReport: () => onReport(replies[index]),
                    ),
                  ],
                  if (group.replies.length > _previewCount)
                    TextButton.icon(
                      onPressed: onToggleReplies,
                      icon: Icon(
                        expanded
                            ? Icons.expand_less
                            : Icons.keyboard_arrow_down,
                        size: 18,
                      ),
                      label: Text(
                        expanded
                            ? AppLocalizations.of(context).commonClose
                            : '+${group.replies.length - _previewCount}',
                      ),
                    ),
                ],
              ),
            ),
          ],
        ],
      ),
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
        children: <Widget>[
          Row(
            children: <Widget>[
              GfAvatar(
                src: resolveApiAssetUrl(post.author.avatarUrl),
                size: 24,
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
                  style: GfTheme.typographyOf(context).caption.copyWith(
                    color: GfTheme.colorsOf(context).iconMuted,
                  ),
                ),
            ],
          ),
          if (post.replyToUsername != null) ...[
            const SizedBox(height: 6),
            Text(
              '${l10n.topicReply} @${post.replyToUsername}',
              style: GfTheme.typographyOf(
                context,
              ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
            ),
          ],
          const SizedBox(height: 10),
          GfMarkdownView(data: post.content),
          const SizedBox(height: 10),
          Row(
            children: <Widget>[
              Text(
                timeAgo(post.createdAt, l10n: l10n),
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
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
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
              GfIconButton(
                icon: Icons.reply_outlined,
                size: 44,
                iconSize: 18,
                tooltip: l10n.topicReply,
                onPressed: onReply,
              ),
              const SizedBox(width: 4),
              GfIconButton(
                icon: Icons.flag_outlined,
                size: 44,
                iconSize: 18,
                tooltip: l10n.topicReport,
                onPressed: onReport,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _NestedPostCard extends StatelessWidget {
  const _NestedPostCard({
    required this.post,
    required this.onReply,
    required this.onReport,
  });

  final PostPayload post;
  final VoidCallback onReply;
  final VoidCallback onReport;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 11),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfAvatar(
            src: resolveApiAssetUrl(post.author.avatarUrl),
            size: 28,
            ring: true,
          ),
          const SizedBox(width: 9),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Row(
                  children: <Widget>[
                    Expanded(
                      child: Text(
                        post.author.nickname ?? post.author.username,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    if (post.postNo > 0)
                      Text(
                        '#${post.postNo}',
                        style: TextStyle(
                          color: colors.baseContent.withValues(alpha: 0.55),
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                  ],
                ),
                if (post.replyToUsername != null) ...<Widget>[
                  const SizedBox(height: 6),
                  _ReplyReference(username: post.replyToUsername!),
                ],
                const SizedBox(height: 8),
                GfMarkdownView(data: post.content),
                const SizedBox(height: 8),
                Row(
                  children: <Widget>[
                    Text(
                      timeAgo(post.createdAt, l10n: l10n),
                      style: GfTheme.typographyOf(
                        context,
                      ).caption.copyWith(color: colors.iconMuted),
                    ),
                    const Spacer(),
                    GfIconButton(
                      icon: Icons.reply_outlined,
                      size: 44,
                      iconSize: 18,
                      tooltip: l10n.topicReply,
                      onPressed: onReply,
                    ),
                    const SizedBox(width: 4),
                    GfIconButton(
                      icon: Icons.flag_outlined,
                      size: 44,
                      iconSize: 18,
                      tooltip: l10n.topicReport,
                      onPressed: onReport,
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ReplyReference extends StatelessWidget {
  const _ReplyReference({required this.username});

  final String username;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: colors.base200,
        border: Border(left: BorderSide(color: colors.primary, width: 2)),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        '${AppLocalizations.of(context).topicReply} @$username',
        style: TextStyle(
          color: colors.baseContent.withValues(alpha: 0.55),
          fontSize: 12,
        ),
      ),
    );
  }
}
