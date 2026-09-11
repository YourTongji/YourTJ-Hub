import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../asset_url.dart';

import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../images/image_upload.dart';
import '../../server_messages.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/status_views.dart';
import '../../widgets/skeletons.dart';
import 'post_actions.dart';
import 'topic_actions.dart';
import 'mention_panel.dart';
import 'mention_search.dart';
import 'mention_session.dart';

/// 话题详情页(web TopicPage.vue 的移动端形态):
/// 话题信息 + 帖子流(分页)+ markdown 渲染 + 图片查看器 + 互动(点赞/收藏/关注/评论)。
class TopicPage extends ConsumerStatefulWidget {
  const TopicPage({super.key, required this.topicId, this.initialPostNo});

  final int topicId;
  final int? initialPostNo;

  @override
  ConsumerState<TopicPage> createState() => _TopicPageState();
}

class _TopicPageState extends ConsumerState<TopicPage> {
  AsyncValue<TopicDetailProps> _page = const AsyncValue.loading();
  bool _viewerAuthenticated = false;
  bool _loadingMore = false;
  bool _jumping = false;
  int _windowGeneration = 0;
  int _currentFloor = 1;
  int? _beforePostNo;
  bool _hasEarlierPosts = false;
  final _scrollToTop = GfScrollToTopController();
  final GlobalKey _discussionKey = GlobalKey();
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
  late final FocusNode _replyFocus = FocusNode(
    onKeyEvent: _replyMentionKeyEvent,
  );
  bool _replying = false;
  CaptchaPayload? _replyCaptcha;
  final _replyCaptchaCode = TextEditingController();
  bool _replyCaptchaLoading = false;
  bool _uploadingReplyImage = false;
  int _replyToPostId = 0;
  String? _replyImageUrl;
  String? _replyTargetName;
  String? _replyMentionPrefix;

  // @mention 候选会话(issue #565):token/候选/防抖逻辑在 mention_session.dart。
  late final MentionSessionController _mentionSession;
  int _viewerId = 0;

  // 浮动层状态(web TopicFloatingControls / PostComposer 语义)。
  bool _composerOpen = false;
  bool _railOpen = false;

  // 引用目标缓存:post id → replyTarget,供平铺引用块渲染被引用内容。
  final Map<int, ReplyTargetPayload> _replyTargets =
      <int, ReplyTargetPayload>{};

  @override
  void initState() {
    super.initState();
    _mentionSession = MentionSessionController(
      searchUsers: ref.read(mentionUserSearchProvider),
    );
    _replyController.addListener(_onReplyValueChanged);
    _replyFocus.addListener(_onReplyFocusChanged);
    _load(postNo: widget.initialPostNo);
  }

  @override
  void didUpdateWidget(TopicPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.topicId != oldWidget.topicId ||
        widget.initialPostNo != oldWidget.initialPostNo) {
      _railOpen = false;
      _composerOpen = false;
      _replyController.clear();
      _replyImageUrl = null;
      _replyToPostId = 0;
      _replyTargetName = null;
      _replyMentionPrefix = null;
      _mentionSession.close();
      _load(postNo: widget.initialPostNo);
    }
  }

  @override
  void dispose() {
    _replyController.dispose();
    _replyCaptchaCode.dispose();
    _replyFocus.dispose();
    _mentionSession.dispose();
    super.dispose();
  }

  Future<void> _load({bool silent = false, int? postNo}) async {
    final generation = ++_windowGeneration;
    _loadingMore = false;
    // 记录发起时的缓存世代;401/登出/换账号后世代自增,返回时丢弃旧会话数据。
    final int epoch = ref.read(offlineCacheEpochProvider);
    if (!silent) setState(() => _page = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .topicDetail(
            widget.topicId,
            postNo: postNo != null && postNo > 1 ? postNo : null,
          );
      if (!mounted ||
          generation != _windowGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
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
      if (!mounted ||
          generation != _windowGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        _page = AsyncValue.data(props);
        _viewerAuthenticated = payload.layout.viewer.isAuthenticated;
        _viewerId = payload.layout.viewer.id;
        _posts.clear();
        _posts.addAll(props.postStream.posts);
        _replyTargets
          ..clear()
          ..addEntries(
            props.postStream.replyTargets.map(
              (ReplyTargetPayload t) => MapEntry(t.id, t),
            ),
          );
        _currentFloor = postNo != null && postNo > 1
            ? postNo
            : props.postStream.posts.firstOrNull?.postNo ?? 1;
        _beforePostNo = props.postStream.beforePostNo;
        _hasEarlierPosts = props.postStream.hasBefore;
        _afterPostNo = props.postStream.afterPostNo;
        _hasMorePosts = props.postStream.hasAfter;
        _liked = props.topic.isLiked;
        _bookmarked = props.topic.isBookmarked;
        _watched = props.topic.isWatched;
        _likeCount = props.topic.likeCount;
      });
    } catch (e, st) {
      // 网络失败:回退 drift 离线缓存(已浏览话题离线可读)。
      if (!mounted ||
          generation != _windowGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (postNo != null && silent) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), e),
          error: true,
        );
        return;
      }
      // 无会话令牌(如 401 后进程被杀重启)时不得回退上一账号残留缓存。
      if (!await hasSessionToken(ref.read(tokenStorageProvider))) {
        // 静默刷新失败时保留当前内容,不打断阅读。
        if (!silent) setState(() => _page = AsyncValue.error(e, st));
        return;
      }
      final PagePayload? cached = await _cacheGet(widget.topicId);
      // 读缓存期间会话可能已切换,再次校验世代再更新 UI。
      if (!mounted ||
          generation != _windowGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final props = cached == null
          ? null
          : parsePageProps<TopicDetailProps>(cached);
      if (props != null) {
        setState(() {
          _page = AsyncValue.data(props);
          _viewerAuthenticated = cached!.layout.viewer.isAuthenticated;
          _viewerId = cached.layout.viewer.id;
          _posts.clear();
          _posts.addAll(props.postStream.posts);
          _replyTargets
            ..clear()
            ..addEntries(
              props.postStream.replyTargets.map(
                (target) => MapEntry(target.id, target),
              ),
            );
          _currentFloor = props.postStream.posts.firstOrNull?.postNo ?? 1;
          _beforePostNo = props.postStream.beforePostNo;
          _hasEarlierPosts = props.postStream.hasBefore;
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

  Future<void> _loadMore({bool earlier = false}) async {
    if (_loadingMore || (earlier ? !_hasEarlierPosts : !_hasMorePosts)) return;
    final generation = _windowGeneration;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _loadingMore = true);
    try {
      final previous = earlier ? _beforePostNo : _afterPostNo;
      final window = await ref
          .read(topicRepositoryProvider)
          .getPostWindow(
            topicId: widget.topicId,
            beforePostNo: earlier ? _beforePostNo : null,
            afterPostNo: earlier ? null : _afterPostNo,
          );
      if (!mounted ||
          generation != _windowGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final ids = _posts.map((post) => post.id).toSet();
      setState(() {
        _posts.addAll(window.posts.where((post) => ids.add(post.id)));
        for (final target in window.replyTargets) {
          _replyTargets[target.id] = target;
        }
        final next =
            (earlier ? window.beforePostNo : window.afterPostNo) ?? previous;
        final advanced =
            next != null &&
            (previous == null || (earlier ? next < previous : next > previous));
        if (earlier) {
          _beforePostNo = next;
          _hasEarlierPosts =
              window.posts.isNotEmpty && window.hasBefore && advanced;
        } else {
          _afterPostNo = next;
          _hasMorePosts =
              window.posts.isNotEmpty && window.hasAfter && advanced;
        }
      });
    } catch (error) {
      if (mounted &&
          generation == _windowGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted && generation == _windowGeneration) {
        setState(() => _loadingMore = false);
      }
    }
  }

  Future<void> _jumpToFloor(int requested) async {
    if (_jumping) return;
    final max = _page.valueOrNull?.postStream.maxPostNo ?? 1;
    final floor = requested.clamp(1, max < 1 ? 1 : max);
    setState(() {
      _jumping = true;
      _railOpen = false;
    });
    try {
      await _load(silent: true, postNo: floor);
      if (mounted && _currentFloor == floor) {
        await _scrollToTop.scrollToTop();
      }
    } finally {
      if (mounted) setState(() => _jumping = false);
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

  bool get _topicAvailable {
    final topic = _page.valueOrNull?.topic;
    return topic != null && !topic.authorDeleted && !topic.moderatorRemoved;
  }

  bool get _canReply =>
      _topicAvailable &&
      (_page.valueOrNull?.permissions.canPost == true || !_viewerAuthenticated);

  /// 编辑值变化:基于 caret 前文本驱动 @mention 会话(选区/无 caret 时关闭)。
  void _onReplyValueChanged() {
    if (!_composerOpen) return;
    final TextEditingValue value = _replyController.value;
    final int base = value.selection.baseOffset;
    final int extent = value.selection.extentOffset;
    if (base < 0 || base != extent) {
      _mentionSession.handleValue(null);
      return;
    }
    _mentionSession.handleValue(
      value.text.substring(0, base.clamp(0, value.text.length)),
    );
  }

  /// 焦点丢失(键盘收起/点按他处)关闭候选会话,不劫持系统返回。
  void _onReplyFocusChanged() {
    if (!_replyFocus.hasFocus) _mentionSession.close();
  }

  /// 物理键盘:↑/↓ 移动 active、Enter 选中、Escape 只关候选不删 @query。
  KeyEventResult _replyMentionKeyEvent(FocusNode node, KeyEvent event) {
    return handleMentionKeyEvent(
      session: _mentionSession,
      controller: _replyController,
      event: event,
      onSelect: _selectMentionCandidate,
    );
  }

  /// 选中候选:原位替换 @query 为 @username(补单个空格),selection 落在插入后。
  void _selectMentionCandidate(MentionToken token, MentionUser user) {
    _replyController.value = applyMentionReplacement(
      _replyController.value,
      MentionReplacement(
        start: token.start,
        length: token.length,
        replacement: '@${user.username} ',
      ),
    );
  }

  MentionUser _toMentionUser(UserBriefPayload user, MentionTag tag) {
    return MentionUser(
      id: user.id,
      username: user.username,
      nickname: user.nickname,
      avatarUrl: resolveApiAssetUrl(user.avatarUrl),
      tag: tag,
    );
  }

  /// 本地上下文候选:回复目标 > 主题作者 > 已加载参与者(匿名/无效 id 排除)。
  List<MentionUser> _mentionLocalUsers() {
    final TopicDetailProps? props = _page.valueOrNull;
    if (props == null) return const <MentionUser>[];
    final List<MentionUser> users = <MentionUser>[];
    if (_replyToPostId != 0) {
      for (final PostPayload post in _posts) {
        if (post.id == _replyToPostId) {
          if (!post.isAnonymous && post.author.id > 0) {
            users.add(_toMentionUser(post.author, MentionTag.replyTarget));
          }
          break;
        }
      }
    }
    if (props.topic.author.id > 0) {
      users.add(_toMentionUser(props.topic.author, MentionTag.topicAuthor));
    }
    for (final UserBriefPayload participant in props.topic.participants) {
      if (participant.id > 0) {
        users.add(_toMentionUser(participant, MentionTag.participant));
      }
    }
    return users;
  }

  void _syncMentionContext() {
    _mentionSession.updateContext(
      local: _mentionLocalUsers(),
      currentUserId: _viewerId,
    );
  }

  void _openComposer({PostPayload? replyTo}) {
    if (!_canReply) return;
    if (_page.valueOrNull?.permissions.canPost != true) {
      context.push('/login');
      return;
    }
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
    _syncMentionContext();
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
    _syncMentionContext();
  }

  void _closeComposer() {
    _replyFocus.unfocus();
    _clearReplyTarget();
    _mentionSession.close();
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

  Future<void> _loadReplyCaptcha() async {
    if (_replyCaptchaLoading) return;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _replyCaptchaLoading = true);
    try {
      final captcha = await ref.read(authRepositoryProvider).getCaptcha();
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _replyCaptcha = captcha;
        _replyCaptchaCode.clear();
      });
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _replyCaptchaLoading = false);
    }
  }

  Future<void> _submitReply() async {
    if (_replying || _uploadingReplyImage || !_canReply) return;
    final content = _replyController.text.trim();
    if (content.isEmpty) return;
    final topicId = widget.topicId;
    final target = _replyToPostId;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _replying = true);
    try {
      await ref
          .read(postRepositoryProvider)
          .createPost(
            topicId: topicId,
            content: content,
            replyToPostId: target,
            captchaId: _replyCaptcha?.captchaId,
            captchaCode: _replyCaptchaCode.text.trim(),
          );
      if (!mounted ||
          topicId != widget.topicId ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (_replyController.text.trim() == content && _replyToPostId == target) {
        _clearReplyTarget();
        _replyController.clear();
        setState(() {
          _replyImageUrl = null;
          _composerOpen = false;
        });
      }
      setState(() {
        _replyCaptcha = null;
        _replyCaptchaCode.clear();
      });
      showGfToast(context, AppLocalizations.of(context).topicReplySuccess);
      await _load(silent: true);
    } catch (error) {
      if (!mounted ||
          topicId != widget.topicId ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (error is ApiException &&
          (error.messageCode == 'common.captchaRequired' ||
              error.messageCode == 'auth.captcha.invalid')) {
        await _loadReplyCaptcha();
      }
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
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

  /// 楼层平铺：除主帖外全部楼层按 postNo 升序线性展示；
  /// replyToPostId 仅用于引用块、通知路由与回答标记，不再用于分组嵌套。
  List<PostPayload> _visiblePosts({PostPayload? mainPost}) {
    final List<PostPayload> replyPosts = <PostPayload>[
      for (final PostPayload post in _posts)
        if (post.id != mainPost?.id) post,
    ]..sort((PostPayload a, PostPayload b) => a.postNo.compareTo(b.postNo));
    return replyPosts;
  }

  /// 引用块显隐与 web showReplyReference 对齐:回复主帖不显示引用块;
  /// 主帖不在已加载窗口时(深链/离线缓存)用 replyTargets.postNo 判定首楼目标。
  bool _showReplyQuote(PostPayload post, PostPayload? mainPost) {
    final int? targetId = post.replyToPostId;
    if (targetId == null || targetId == 0) return false;
    if (mainPost != null) return targetId != mainPost.id;
    final ReplyTargetPayload? target = _replyTargets[targetId];
    if (target == null) return true;
    return target.postNo != 1;
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
        actions: [
          if (_page.valueOrNull case final props?)
            TopicActions(
              props: props,
              firstPostId: _mainPost(_posts)?.id,
              onChanged: () => _load(silent: true, postNo: _currentFloor),
            ),
        ],
      ),
      body: _page.when(
        loading: () => const GfTopicDetailSkeleton(),
        error: (e, _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (props) {
          final PostPayload? mainPost = _mainPost(_posts);
          final List<PostPayload> replyPosts = _visiblePosts(
            mainPost: mainPost,
          );

          return Stack(
            children: <Widget>[
              Positioned.fill(
                child: GfScrollToTop(
                  semanticLabel: l10n.commonBackToTop,
                  showButton: false,
                  controller: _scrollToTop,
                  threshold: 360,
                  bottomInset: 84,
                  builder: (context, scrollController) {
                    return AppRefreshIndicator(
                      onRefresh: () => _load(silent: true),
                      child: CustomScrollView(
                        keyboardDismissBehavior:
                            ScrollViewKeyboardDismissBehavior.onDrag,
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
                              canReportTopic:
                                  _topicAvailable &&
                                  !props.permissions.isOwnTopic,
                              onLike: _toggleLike,
                              onBookmark: _toggleBookmark,
                              onWatch: _toggleWatch,
                              onReportTopic: () => _reportTopic(props.topic),
                              onReply: _canReply ? () => _openComposer() : null,
                            ),
                          ),
                          const SliverToBoxAdapter(child: GfDivider()),
                          SliverToBoxAdapter(
                            child: SizedBox(
                              key: _discussionKey,
                              child: _ReplySectionHeader(
                                count: props.topic.replyCount,
                              ),
                            ),
                          ),
                          if (_hasEarlierPosts)
                            SliverToBoxAdapter(
                              child: TextButton(
                                onPressed: _loadingMore
                                    ? null
                                    : () => _loadMore(earlier: true),
                                child: Text(l10n.topicEarlierReplies),
                              ),
                            ),
                          if (replyPosts.isEmpty)
                            SliverToBoxAdapter(
                              child: GfEmpty(
                                icon: Icons.forum_outlined,
                                message: l10n.topicReplies(0),
                                description: l10n.topicReplyHint,
                              ),
                            )
                          else
                            SliverList.builder(
                              itemCount: replyPosts.length,
                              itemBuilder: (BuildContext context, int index) {
                                final PostPayload post = replyPosts[index];
                                return RepaintBoundary(
                                  key: ValueKey(post.id),
                                  child: Column(
                                    children: <Widget>[
                                      _PostCard(
                                        post: post,
                                        showReplyQuote: _showReplyQuote(
                                          post,
                                          mainPost,
                                        ),
                                        quoteTarget:
                                            _replyTargets[post.replyToPostId],
                                        onReply: _canReply
                                            ? () => _openComposer(replyTo: post)
                                            : null,
                                        onReport: () => _reportPost(post),
                                        onChanged: () => _load(
                                          silent: true,
                                          postNo: _currentFloor,
                                        ),
                                      ),
                                      if (index < replyPosts.length - 1)
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
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.center,
                      children: <Widget>[
                        if (_composerOpen)
                          ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 560),
                            child: ListenableBuilder(
                              listenable: _mentionSession,
                              builder: (context, _) => MentionCandidatesPanel(
                                session: _mentionSession,
                                messages: MentionPanelMessages(
                                  listboxLabel: l10n.mentionListboxLabel,
                                  loading: l10n.mentionLoading,
                                  noResults: l10n.mentionNoResults,
                                  searchFailed: l10n.mentionSearchFailed,
                                  keepTyping: l10n.mentionKeepTyping,
                                  tagReplyTarget: l10n.mentionTagReplyTarget,
                                  tagTopicAuthor: l10n.mentionTagTopicAuthor,
                                  tagParticipant: l10n.mentionTagParticipant,
                                ),
                                onSelect: _selectMentionCandidate,
                              ),
                            ),
                          ),
                        _composerOpen
                            ? ConstrainedBox(
                                constraints: const BoxConstraints(
                                  maxWidth: 560,
                                ),
                                child: ValueListenableBuilder<TextEditingValue>(
                                  valueListenable: _replyController,
                                  builder: (context, value, _) {
                                    return GfPostComposer(
                                      hideKeyboardLabel:
                                          l10n.commonHideKeyboard,
                                      onCollapse: _closeComposer,
                                      collapseLabel: l10n.commonCancel,
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
                                      removeImageTooltip:
                                          l10n.publishRemoveImage,
                                      uploading: _uploadingReplyImage,
                                      publishing: _replying,
                                      canPublish: value.text.trim().isNotEmpty,
                                      publishLabel: l10n.commonSend,
                                      hintText: l10n.topicReplyHint,
                                      onPublish: _submitReply,
                                      toolbar: _replyCaptcha == null
                                          ? null
                                          : Row(
                                              children: [
                                                InkWell(
                                                  onTap: _replyCaptchaLoading
                                                      ? null
                                                      : _loadReplyCaptcha,
                                                  child: Image.memory(
                                                    base64Decode(
                                                      _replyCaptcha!.captchaImg
                                                          .split(',')
                                                          .last,
                                                    ),
                                                    width: 80,
                                                    height: 42,
                                                    fit: BoxFit.contain,
                                                  ),
                                                ),
                                                const SizedBox(width: 8),
                                                Expanded(
                                                  child: TextField(
                                                    key: const Key(
                                                      'reply-captcha',
                                                    ),
                                                    controller:
                                                        _replyCaptchaCode,
                                                    decoration: InputDecoration(
                                                      labelText:
                                                          l10n.authCaptcha,
                                                    ),
                                                    textCapitalization:
                                                        TextCapitalization
                                                            .characters,
                                                  ),
                                                ),
                                                IconButton(
                                                  tooltip: l10n.commonRefresh,
                                                  onPressed:
                                                      _replyCaptchaLoading
                                                      ? null
                                                      : _loadReplyCaptcha,
                                                  icon: const Icon(
                                                    Icons.refresh,
                                                  ),
                                                ),
                                              ],
                                            ),
                                    );
                                  },
                                ),
                              )
                            : GfFloatingControls(
                                joinLabel: l10n.topicJoinDiscussion,
                                actions: <GfTopicAction>[
                                  if (_topicAvailable) ...[
                                    GfTopicAction(
                                      icon: Icons.favorite_border,
                                      symbol: _liked ? 'heart-filled' : 'heart',
                                      active: _liked,
                                      activeColor: colors.error,
                                      onTap: _toggleLike,
                                    ),
                                    GfTopicAction(
                                      icon: Icons.bookmark_border,
                                      symbol: _bookmarked
                                          ? 'bookmark-filled'
                                          : 'bookmark',
                                      active: _bookmarked,
                                      activeColor: colors.warning,
                                      onTap: _toggleBookmark,
                                    ),
                                    GfTopicAction(
                                      icon: _watched
                                          ? Icons.notifications
                                          : Icons.notifications_none,
                                      symbol: 'bell',
                                      active: _watched,
                                      activeColor: colors.primary,
                                      onTap: _toggleWatch,
                                    ),
                                  ],
                                ],
                                onOpenReply: _canReply
                                    ? () => _openComposer()
                                    : null,
                                currentNo: _currentFloor,
                                maxNo: props.postStream.maxPostNo,
                                onFloorTap: () =>
                                    setState(() => _railOpen = !_railOpen),
                              ),
                      ],
                    ),
                  ),
                ),
              ),
              if (_jumping)
                const Positioned(
                  top: 0,
                  left: 0,
                  right: 0,
                  child: LinearProgressIndicator(),
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
                        current: _currentFloor,
                        max: props.postStream.maxPostNo,
                        startLabel: l10n.topicEarliest,
                        endLabel: l10n.topicLatest,
                        onSelect: _jumpToFloor,
                        onEarliest: () => _jumpToFloor(1),
                        onLatest: () =>
                            _jumpToFloor(props.postStream.maxPostNo),
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
    this.onReply,
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
  final VoidCallback? onReply;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String authorName = topic.author.nickname ?? topic.author.username;
    final available = !topic.authorDeleted && !topic.moderatorRemoved;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              InkWell(
                onTap: topic.author.id > 0
                    ? () => context.push('/u/${topic.author.id}')
                    : null,
                borderRadius: BorderRadius.circular(40),
                child: GfAvatar(
                  src: resolveApiAssetUrl(topic.author.avatarUrl),
                  size: 40,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    InkWell(
                      onTap: topic.author.id > 0
                          ? () => context.push('/u/${topic.author.id}')
                          : null,
                      child: Text(
                        authorName,
                        style: GfTheme.typographyOf(context).bodyStrong,
                      ),
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
          if (available &&
              topic.contentType != 3 &&
              topic.contentType != 0 &&
              topic.images?.isNotEmpty == true) ...[
            const SizedBox(height: 16),
            GfMediaCarousel(
              images: topic.images!.map(resolveApiAssetUrl).toList(),
            ),
          ],
          if (!available) ...<Widget>[
            const SizedBox(height: 16),
            Text(l10n.topicRemoved),
          ] else if (mainPost != null) ...<Widget>[
            const SizedBox(height: 16),
            GfMarkdownView(data: mainPost!.content),
          ] else if (topic.description.isNotEmpty) ...<Widget>[
            const SizedBox(height: 12),
            Text(topic.description, style: GfTheme.typographyOf(context).body),
          ],
          const SizedBox(height: 18),
          Wrap(
            spacing: 16,
            runSpacing: 8,
            children: [
              _TopicStat(
                symbol: 'eye',
                value: formatNumber(topic.viewCount),
                label: l10n.draftsMetaViews(topic.viewCount),
              ),
              _TopicStat(
                symbol: 'message-circle',
                value: formatNumber(topic.replyCount),
                label: l10n.topicReplies(topic.replyCount),
              ),
            ],
          ),
          if (available) ...[
            const SizedBox(height: 12),
            const GfDivider(),
            const SizedBox(height: 8),
            Wrap(
              key: const ValueKey('topic-inline-actions'),
              spacing: 4,
              runSpacing: 4,
              children: [
                if (onReply != null)
                  _TopicAction(
                    symbol: 'corner-down-left',
                    label: l10n.topicReply,
                    tooltip: l10n.topicReply,
                    onTap: onReply,
                    prominent: true,
                  ),
                _TopicAction(
                  symbol: liked ? 'heart-filled' : 'heart',
                  label: likeCount > 0
                      ? formatNumber(likeCount)
                      : l10n.topicLike,
                  tooltip: l10n.topicLike,
                  selected: liked,
                  color: colors.error,
                  onTap: onLike,
                ),
                _TopicAction(
                  symbol: bookmarked ? 'bookmark-filled' : 'bookmark',
                  label: l10n.topicBookmark,
                  tooltip: bookmarked
                      ? l10n.topicBookmarked
                      : l10n.topicBookmark,
                  selected: bookmarked,
                  color: colors.warning,
                  onTap: onBookmark,
                ),
                _TopicAction(
                  symbol: 'bell',
                  label: watched ? l10n.topicUnwatch : l10n.topicWatch,
                  tooltip: watched ? l10n.topicUnwatch : l10n.topicWatch,
                  selected: watched,
                  color: colors.primary,
                  onTap: onWatch,
                ),
              ],
            ),
          ],
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
        ],
      ),
    );
  }
}

/// Passive counts stay visually separate from touch targets, like Web's metadata.
class _TopicStat extends StatelessWidget {
  const _TopicStat({
    required this.symbol,
    required this.value,
    required this.label,
  });
  final String symbol;
  final String value;
  final String label;
  @override
  Widget build(BuildContext context) {
    final color = GfTheme.colorsOf(context).iconMuted;
    return Semantics(
      label: label,
      child: ExcludeSemantics(
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GfSymbol(symbol, size: 16, color: color),
            const SizedBox(width: 5),
            Text(
              value,
              style: GfTheme.typographyOf(
                context,
              ).caption.copyWith(color: color),
            ),
          ],
        ),
      ),
    );
  }
}

class _TopicAction extends StatelessWidget {
  const _TopicAction({
    required this.symbol,
    required this.label,
    required this.tooltip,
    this.selected,
    this.prominent = false,
    this.color,
    this.onTap,
  });
  final String symbol;
  final String label;
  final String tooltip;
  final bool? selected;
  final bool prominent;
  final Color? color;
  final VoidCallback? onTap;
  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final foreground = prominent
        ? colors.primaryContent
        : selected == true
        ? color ?? colors.primary
        : colors.iconMuted;
    return Semantics(
      toggled: selected,
      child: Tooltip(
        message: tooltip,
        child: TextButton.icon(
          onPressed: onTap,
          icon: GfSymbol(symbol, size: 18),
          label: Text(label),
          style: TextButton.styleFrom(
            foregroundColor: foreground,
            backgroundColor: prominent
                ? colors.primary
                : selected == true
                ? (color ?? colors.primary).withValues(alpha: .1)
                : Colors.transparent,
            minimumSize: const Size(44, 44),
            padding: const EdgeInsets.symmetric(horizontal: 12),
            shape: const StadiumBorder(),
          ),
        ),
      ),
    );
  }
}

class _PostCard extends StatelessWidget {
  const _PostCard({
    required this.post,
    required this.showReplyQuote,
    required this.quoteTarget,
    required this.onReply,
    required this.onReport,
    required this.onChanged,
  });

  final PostPayload post;

  /// 平铺模式下的引用块开关：回复其他楼层显示引用块，回复主帖保持轻量文本。
  final bool showReplyQuote;

  /// 被引用楼层的 replyTarget（可空：目标信息缺失时按 unavailable 降级）。
  final ReplyTargetPayload? quoteTarget;
  final VoidCallback? onReply;
  final VoidCallback onReport;
  final Future<void> Function() onChanged;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            children: <Widget>[
              InkWell(
                onTap: post.author.id > 0
                    ? () => context.push('/u/${post.author.id}')
                    : null,
                borderRadius: BorderRadius.circular(24),
                child: GfAvatar(
                  src: resolveApiAssetUrl(post.author.avatarUrl),
                  size: 32,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: InkWell(
                  onTap: post.author.id > 0
                      ? () => context.push('/u/${post.author.id}')
                      : null,
                  child: Text(
                    post.author.nickname ?? post.author.username,
                    style: GfTheme.typographyOf(context).bodyStrong,
                  ),
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
            if (showReplyQuote)
              _ReplyQuote(
                username: post.replyToUsername!,
                avatarUrl: quoteTarget?.author.avatarUrl ?? '',
                postNo: quoteTarget?.postNo,
                contentPreview: _plainTextFromHtml(
                  quoteTarget?.renderedContent,
                ),
                unavailable:
                    quoteTarget == null || quoteTarget?.unavailable == true,
              )
            else
              Text(
                '${l10n.topicReply} @${post.replyToUsername}',
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              ),
          ],
          const SizedBox(height: 4),
          if (post.isAuthorDeleted || post.isModeratorRemoved)
            Text(l10n.topicRemoved)
          else
            GfMarkdownView(data: post.content),
          const SizedBox(height: 4),
          LayoutBuilder(
            builder: (context, constraints) {
              final timestamp = Text(
                timeAgo(post.createdAt, l10n: l10n),
                style: GfTheme.typographyOf(
                  context,
                ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
              );
              final actions = PostActions(
                post: post,
                onChanged: onChanged,
                onReply: onReply,
                onReport: onReport,
              );
              // Keep controls tappable at large text sizes and narrow widths.
              if (constraints.maxWidth < 340 ||
                  MediaQuery.textScalerOf(context).scale(14) > 20) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    timestamp,
                    Align(alignment: Alignment.centerRight, child: actions),
                  ],
                );
              }
              return Row(
                children: [
                  Expanded(child: timestamp),
                  actions,
                ],
              );
            },
          ),
        ],
      ),
    );
  }
}

/// 去除服务端渲染 HTML 的标签，取纯文本预览（移动端引用块轻量展示用）。
String _plainTextFromHtml(String? html) {
  if (html == null || html.isEmpty) return '';
  return html
      .replaceAll(RegExp(r'<br\s*/?>'), '\n')
      .replaceAll(RegExp(r'</(p|div|li|h[1-6]|blockquote|tr)>'), '\n')
      .replaceAll(RegExp(r'<[^>]*>'), '')
      .replaceAll('&nbsp;', ' ')
      .replaceAll('&amp;', '&')
      .replaceAll('&lt;', '<')
      .replaceAll('&gt;', '>')
      .replaceAll('&quot;', '"')
      .replaceAll('&#39;', "'")
      .replaceAll(RegExp(r'\n{3,}'), '\n\n')
      .trim();
}

/// 平铺回复的引用块：被引用楼层作者/楼号 + 纯文本预览，长内容可展开收起。
class _ReplyQuote extends StatefulWidget {
  const _ReplyQuote({
    required this.username,
    required this.unavailable,
    this.postNo,
    this.contentPreview,
    this.avatarUrl = '',
  });

  final String username;
  final int? postNo;
  final String? contentPreview;
  final String avatarUrl;
  final bool unavailable;

  @override
  State<_ReplyQuote> createState() => _ReplyQuoteState();
}

class _ReplyQuoteState extends State<_ReplyQuote> {
  bool _expanded = false;
  @override
  void didUpdateWidget(covariant _ReplyQuote oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.contentPreview != widget.contentPreview ||
        oldWidget.postNo != widget.postNo) {
      _expanded = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String preview = widget.contentPreview ?? '';
    final bool hasPreview = !widget.unavailable && preview.isNotEmpty;
    final style = GfTheme.typographyOf(context).small.copyWith(
      color: colors.baseContent.withValues(alpha: 0.75),
      height: 1.45,
    );
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: colors.base200.withValues(alpha: 0.4),
        border: Border(
          left: BorderSide(
            color: colors.primary.withValues(alpha: 0.45),
            width: 2,
          ),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              if (!widget.unavailable) ...[
                GfAvatar(src: resolveApiAssetUrl(widget.avatarUrl), size: 20),
                const SizedBox(width: 8),
              ],
              Expanded(
                child: Text(
                  '@${widget.username}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: style.copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              if (widget.postNo != null)
                Text(
                  '#${widget.postNo}',
                  style: style.copyWith(color: colors.iconMuted),
                ),
            ],
          ),
          if (widget.unavailable) ...[
            const SizedBox(height: 6),
            Text(l10n.topicReplyTargetUnavailable, style: style),
          ] else if (hasPreview) ...[
            const SizedBox(height: 6),
            LayoutBuilder(
              builder: (context, constraints) {
                final painter = TextPainter(
                  text: TextSpan(text: preview, style: style),
                  maxLines: 4,
                  textDirection: Directionality.of(context),
                  textScaler: MediaQuery.textScalerOf(context),
                )..layout(maxWidth: constraints.maxWidth);
                final overflowing = painter.didExceedMaxLines;
                painter.dispose();
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      preview,
                      maxLines: _expanded ? null : 4,
                      overflow: _expanded
                          ? TextOverflow.clip
                          : TextOverflow.ellipsis,
                      style: style,
                    ),
                    if (overflowing)
                      TextButton.icon(
                        style: TextButton.styleFrom(
                          padding: EdgeInsets.zero,
                          minimumSize: const Size(44, 44),
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                        onPressed: () => setState(() => _expanded = !_expanded),
                        icon: Icon(
                          _expanded ? Icons.expand_less : Icons.expand_more,
                          size: 16,
                        ),
                        label: Text(
                          _expanded
                              ? l10n.replyQuoteCollapse
                              : l10n.replyQuoteExpand,
                        ),
                      ),
                  ],
                );
              },
            ),
          ],
        ],
      ),
    );
  }
}
