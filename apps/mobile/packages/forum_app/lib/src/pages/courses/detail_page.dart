import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../reading_preferences.dart';
import '../../widgets/rich_content/gf_html_content.dart';
import '../../format.dart';
import '../../current_user.dart';
import '../../images/image_save.dart';
import '../../link_navigation.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import 'course_common.dart';
import 'review_form_sheet.dart';
import 'review_delete_dialog.dart';
import 'review_reaction.dart';
import 'review_report_sheet.dart';
import '../../widgets/share/course_review_share_card.dart';
import '../../widgets/share/share_image_preview.dart';

/// 课程详情页（web CourseDetailPage.vue 的移动端形态，路由 `/courses/:courseId`）。
///
/// [focusOfferingId] 对应 web `?offeringId=` 聚焦语义：仅展示该教学班的课评。
///
/// 页面纵向堆叠（对齐 web 移动端 DOM 顺序）：头部信息 → 评分分布卡 → AI 总结
/// → 课评列表（cursor 分页 + 写评/编辑/删除/有用）+ 浮动写评 → 开课记录
/// → 相关课程 + 课程沿革。收藏按钮在 AppBar（乐观切换 + 失败回滚）。
class CourseDetailPage extends ConsumerStatefulWidget {
  const CourseDetailPage({
    super.key,
    required this.courseId,
    this.focusOfferingId,
    this.focusReviewId,
  });

  final int courseId;
  final int? focusOfferingId;
  final int? focusReviewId;

  @override
  ConsumerState<CourseDetailPage> createState() => _CourseDetailPageState();
}

class _CourseDetailPageState extends ConsumerState<CourseDetailPage> {
  static const double _loadMoreThreshold = 400;

  final ScrollController _scrollController = ScrollController();
  final GlobalKey _reviewsSectionKey = GlobalKey();
  final GlobalKey _targetReviewKey = GlobalKey();
  bool _reviewRevealed = false;

  AsyncValue<CourseDetailPayload> _detail = const AsyncValue.loading();
  AsyncValue<CourseRelatedResult> _related = const AsyncValue.loading();

  // 收藏（乐观切换）。
  bool _bookmarked = false;

  // 课评列表。
  List<ReviewPayload> _reviews = <ReviewPayload>[];
  String _nextCursor = '';
  bool _reviewsLoading = false;
  bool _reviewsLoaded = false;
  bool _loadingMore = false;
  int _reviewTotal = 0;
  // 写操作代际守卫：写成功后自增，使 in-flight 列表响应失效。
  int _reviewsSeq = 0;
  final Set<int> _reviewReactionBusy = <int>{};
  final Set<int> _reviewReportBusy = <int>{};
  // 每个评价的反应本地版本号：列表响应合并时用它判断请求在途期间是否发生过
  // 反应切换（版本变化）或请求发起时写入仍未完成（busy 快照）。
  final Map<int, int> _reviewReactionVersion = <int, int>{};

  /// 聚焦教学班（null = 课程全部课评）。初始值来自路由参数，详情加载后校验。
  int? _activeOfferingId;

  CourseRepository get _repository => ref.read(courseRepositoryProvider);

  /// 已登录（token 可解析出用户）。未登录时课评卡片不提供举报入口。
  bool get _signedIn => ref.watch(currentUserProvider).valueOrNull != null;

  @override
  void initState() {
    super.initState();
    _load();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (!_scrollController.hasClients) return;
    if (_scrollController.position.extentAfter < _loadMoreThreshold) {
      _loadMoreReviews();
    }
  }

  void _goBack() {
    final NavigatorState navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }
    context.go('/courses');
  }

  Future<void> _load() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _detail = const AsyncValue.loading());
    try {
      final CourseDetailPayload detail = await _repository.detail(
        widget.courseId,
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _detail = AsyncValue.data(detail);
        _validateInitialFocus(detail);
      });
      _loadRelated();
      _loadReviews();
    } catch (e, st) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _detail = AsyncValue.error(e, st));
    }
  }

  /// 路由 `?offeringId=` 只接受当前课程可见开课实例（防跨课程/隐藏 offering）。
  void _validateInitialFocus(CourseDetailPayload detail) {
    final int? raw = widget.focusOfferingId;
    if (raw == null) return;
    final bool visible =
        detail.offerings?.any((CourseOfferingPayload o) => o.id == raw) ??
        false;
    _activeOfferingId = visible ? raw : null;
  }

  Future<void> _loadRelated() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _related = const AsyncValue.loading());
    try {
      final CourseRelatedResult result = await _repository.related(
        widget.courseId,
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _related = AsyncValue.data(result));
    } catch (e, st) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _related = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadReviews() async {
    final int seq = ++_reviewsSeq;
    final int epoch = ref.read(offlineCacheEpochProvider);
    final Map<int, int> reactionVersions = Map<int, int>.of(
      _reviewReactionVersion,
    );
    final Set<int> reactionBusy = Set<int>.of(_reviewReactionBusy);
    setState(() {
      _reviewsLoading = true;
      _reviewsLoaded = false;
    });
    try {
      final ReviewListResult result = await _repository.reviews(
        widget.courseId,
        offeringId: _activeOfferingId,
      );
      if (!mounted ||
          seq != _reviewsSeq ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        // freezed 解析出的 list 可能是不可变包装；后续行内替换/删除依赖可变列表。
        _reviews = List<ReviewPayload>.of(
          mergeCourseReviewReactions(
            incoming: result.list,
            local: _reviews,
            versionsAtStart: reactionVersions,
            currentVersions: _reviewReactionVersion,
            busyAtStart: reactionBusy,
          ),
        );
        _prioritizeEditableReviews();
        _nextCursor = result.nextCursor ?? '';
        _reviewTotal = result.total;
        _reviewsLoading = false;
        _reviewsLoaded = true;
      });
      WidgetsBinding.instance.addPostFrameCallback(
        (_) => _revealTargetReview(),
      );
    } catch (_) {
      if (!mounted || seq != _reviewsSeq) return;
      setState(() {
        _reviewsLoading = false;
        _reviewsLoaded = true;
      });
      _toast(_copy().reviewsLoadFailed, error: true);
    }
  }

  Future<void> _revealTargetReview() async {
    if (_reviewRevealed ||
        widget.focusReviewId == null ||
        !_reviews.any((review) => review.id == widget.focusReviewId)) {
      return;
    }
    _reviewRevealed = true;
    // The review section is lazy-built after the summary. Advance only until
    // the requested row mounts, then align that row below the app bar.
    for (var attempt = 0; mounted && attempt < 12; attempt++) {
      final target = _targetReviewKey.currentContext;
      if (target != null && target.mounted && mounted) {
        await Scrollable.ensureVisible(
          target,
          alignment: .1,
          duration: GfMotion.duration(context, GfMotion.layout),
        );
        return;
      }
      if (!_scrollController.hasClients) return;
      final position = _scrollController.position;
      final next = (position.pixels + position.viewportDimension * .7).clamp(
        0.0,
        position.maxScrollExtent,
      );
      if (next == position.pixels) return;
      if (GfMotion.reducedOf(context)) {
        _scrollController.jumpTo(next);
        // A lazy review needs a layout frame before its key can be inspected.
        await WidgetsBinding.instance.endOfFrame;
      } else {
        await _scrollController.animateTo(
          next,
          duration: GfMotion.press,
          curve: GfMotion.enterCurve,
        );
      }
    }
  }

  Future<void> _loadMoreReviews() async {
    if (_loadingMore ||
        _reviewsLoading ||
        _nextCursor.isEmpty ||
        !_reviewsLoaded) {
      return;
    }
    final int seq = _reviewsSeq;
    final int epoch = ref.read(offlineCacheEpochProvider);
    final Map<int, int> reactionVersions = Map<int, int>.of(
      _reviewReactionVersion,
    );
    final Set<int> reactionBusy = Set<int>.of(_reviewReactionBusy);
    setState(() => _loadingMore = true);
    try {
      final ReviewListResult result = await _repository.reviews(
        widget.courseId,
        offeringId: _activeOfferingId,
        cursor: _nextCursor,
      );
      if (!mounted ||
          seq != _reviewsSeq ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        _reviews = <ReviewPayload>[
          ..._reviews,
          ...mergeCourseReviewReactions(
            incoming: result.list,
            local: _reviews,
            versionsAtStart: reactionVersions,
            currentVersions: _reviewReactionVersion,
            busyAtStart: reactionBusy,
          ),
        ];
        _prioritizeEditableReviews();
        _nextCursor = result.nextCursor ?? '';
        _reviewTotal = result.total;
      });
    } catch (_) {
      // 加载更多失败静默，滚动可再次触发。
    } finally {
      if (mounted &&
          seq == _reviewsSeq &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _loadingMore = false);
      }
    }
  }

  CourseCopy _copy() => CourseCopy(AppLocalizations.of(context));

  void _prioritizeEditableReviews() {
    final editable = <ReviewPayload>[];
    final other = <ReviewPayload>[];
    for (final review in _reviews) {
      (review.viewer.canEdit ? editable : other).add(review);
    }
    _reviews
      ..clear()
      ..addAll(editable)
      ..addAll(other);
  }

  void _toast(String message, {bool error = false}) {
    if (!mounted) return;
    showGfToast(context, message, error: error);
  }

  bool _isUnauthorized(Object error) => error is UnauthorizedException;

  // ---- 收藏 ----

  Future<void> _toggleBookmark() async {
    final bool target = !_bookmarked;
    setState(() => _bookmarked = target);
    try {
      await _repository.bookmark(courseId: widget.courseId, bookmarked: target);
    } catch (e) {
      // 401 由全局会话失效处理（跳登录），此处仅回滚本地态不打扰用户。
      if (!mounted) return;
      setState(() => _bookmarked = !target);
      if (!_isUnauthorized(e)) {
        _toast(_copy().operationFailed, error: true);
      }
    }
  }

  // ---- 教学班聚焦 ----

  void _focusOffering(int offeringId) {
    setState(() => _activeOfferingId = offeringId);
    _loadReviews();
    // 回到课评区顶部展示聚焦横幅。
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final BuildContext? section = _reviewsSectionKey.currentContext;
      if (section != null && mounted) {
        Scrollable.ensureVisible(
          section,
          duration: GfMotion.duration(context, GfMotion.overlay),
          curve: GfMotion.enterCurve,
          alignment: 0.08,
        );
      }
    });
  }

  void _clearOfferingFocus() {
    setState(() => _activeOfferingId = null);
    _loadReviews();
  }

  // ---- 课评操作 ----

  Future<void> _openWriteSheet() async {
    final CourseDetailPayload? detail = _detail.value;
    final List<CourseOfferingPayload> offerings = detail?.offerings ?? const [];
    if (offerings.isEmpty) return;
    final int defaultOffering = _activeOfferingId ?? offerings.first.id;
    final ReviewPayload? created = await showGfBottomSheet<ReviewPayload>(
      context,
      height: 600,
      keyboardAware: true,
      barrierDismissible: false,
      enableDrag: false,
      builder: (BuildContext ctx) => CourseReviewFormSheet(
        pageContext: context,
        repository: _repository,
        offerings: offerings,
        initialOfferingId: defaultOffering,
      ),
    );
    if (created == null || !mounted) return;
    // 聚焦教学班且新评价不属于该班时，重新拉取对齐；否则前置插入。
    if (_activeOfferingId == null || created.offeringId == _activeOfferingId) {
      _reviewsSeq += 1;
      setState(() {
        _reviews = <ReviewPayload>[created, ..._reviews];
        _reviewTotal += 1;
      });
    } else {
      _loadReviews();
    }
    _toast(_copy().submitSuccess);
  }

  Future<void> _openEditSheet(ReviewPayload review) async {
    final CourseDetailPayload? detail = _detail.value;
    final List<CourseOfferingPayload> offerings = detail?.offerings ?? const [];
    final ReviewPayload? updated = await showGfBottomSheet<ReviewPayload>(
      context,
      height: 600,
      keyboardAware: true,
      barrierDismissible: false,
      enableDrag: false,
      builder: (BuildContext ctx) => CourseReviewFormSheet(
        pageContext: context,
        repository: _repository,
        offerings: offerings,
        editing: review,
      ),
    );
    if (updated == null || !mounted) return;
    setState(() {
      final int index = _reviews.indexWhere(
        (ReviewPayload r) => r.id == updated.id,
      );
      if (index >= 0) _reviews[index] = updated;
    });
    _toast(_copy().updateSuccess);
  }

  Future<void> _confirmDeleteReview(ReviewPayload review) async {
    final CourseCopy copy = _copy();
    final bool confirmed = await confirmCourseReviewDeletion(context, review);
    if (confirmed != true || !mounted) return;
    try {
      await _repository.deleteReview(review.id);
      if (!mounted) return;
      setState(() {
        _reviews.removeWhere((ReviewPayload r) => r.id == review.id);
        _reviewTotal = _reviewTotal > 0 ? _reviewTotal - 1 : 0;
      });
      _toast(copy.reviewDeleted);
    } catch (e) {
      if (!mounted) return;
      if (!_isUnauthorized(e)) {
        _toast(courseReviewError(AppLocalizations.of(context), e), error: true);
      }
    }
  }

  Future<bool> _requireSignedIn() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    try {
      final user = await ref.read(currentUserProvider.future);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) {
        return false;
      }
      if (user != null) return true;
      context.push('/login');
      return false;
    } catch (error) {
      if (mounted &&
          epoch == ref.read(offlineCacheEpochProvider) &&
          !_isUnauthorized(error)) {
        _toast(_copy().operationFailed, error: true);
      }
      return false;
    }
  }

  Future<void> _toggleReviewReaction(
    ReviewPayload review,
    CourseReviewReaction reaction,
  ) async {
    if (_reviewReactionBusy.contains(review.id)) return;
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _reviewReactionBusy.add(review.id));
    try {
      final user = await ref.read(currentUserProvider.future);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      if (user == null) {
        context.push('/login');
        return;
      }
      final int index = _reviews.indexWhere((r) => r.id == review.id);
      if (index < 0) return;
      final ReviewPayload current = _reviews[index];
      final bool on = reaction == CourseReviewReaction.helpful
          ? !current.viewer.isHelpful
          : !current.viewer.isDisliked;
      // 版本号在乐观更新前自增：在途列表响应合并时据此判断本地反应是否比
      // 服务端快照更新，避免慢响应回滚。
      _reviewReactionVersion[review.id] =
          (_reviewReactionVersion[review.id] ?? 0) + 1;
      // 单次本地乐观更新 + 对新旧服务端都正确的请求序列：旧服务端不会在
      // markTarget(true) 时自动清掉另一侧，所以切换时先删除相反状态再写目标
      // （新服务端两个操作均幂等）。不重读整表、不触发列表 loading/重排。
      setState(() {
        _reviews[index] = applyCourseReviewReaction(current, reaction, on: on);
      });
      final CourseReviewReaction opposite =
          reaction == CourseReviewReaction.helpful
          ? CourseReviewReaction.dislike
          : CourseReviewReaction.helpful;
      final bool oppositeSelected = opposite == CourseReviewReaction.helpful
          ? current.viewer.isHelpful
          : current.viewer.isDisliked;
      bool oppositeCleared = false;
      try {
        if (on && oppositeSelected) {
          await _writeReviewReaction(current.id, opposite, false);
          oppositeCleared = true;
          // `_repository` 每次现读 provider：两步之间退出或切换账号时，
          // 第二步会落到新会话的仓库上，替新账号写入旧操作。
          if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
        }
        await _writeReviewReaction(current.id, reaction, on);
      } catch (e) {
        if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
        // 相反侧已在服务端清掉时回滚到「两侧都未选」，与服务端一致；
        // 否则什么都没提交，恢复原状态。
        final ReviewPayload restored = oppositeCleared
            ? applyCourseReviewReaction(current, opposite, on: false)
            : current;
        setState(() {
          final int i = _reviews.indexWhere((r) => r.id == review.id);
          if (i >= 0) _reviews[i] = restored;
        });
        if (!_isUnauthorized(e)) {
          _toast(_copy().operationFailed, error: true);
        }
      }
    } catch (e) {
      if (mounted &&
          epoch == ref.read(offlineCacheEpochProvider) &&
          !_isUnauthorized(e)) {
        _toast(_copy().operationFailed, error: true);
      }
    } finally {
      if (mounted) setState(() => _reviewReactionBusy.remove(review.id));
    }
  }

  Future<void> _writeReviewReaction(
    int reviewId,
    CourseReviewReaction reaction,
    bool on,
  ) => reaction == CourseReviewReaction.helpful
      ? _repository.markHelpful(reviewId, on: on)
      : _repository.markDislike(reviewId, on: on);

  Future<void> _reportReview(ReviewPayload review) async {
    if (_reviewReportBusy.contains(review.id)) return;
    setState(() => _reviewReportBusy.add(review.id));
    try {
      if (!await _requireSignedIn() || review.viewer.canEdit) return;
      if (!mounted) return;
      await showCourseReviewReportSheet(
        context,
        repository: _repository,
        reviewId: review.id,
      );
    } finally {
      if (mounted) setState(() => _reviewReportBusy.remove(review.id));
    }
  }

  /// 正文链接：复用仓库共享的 LinkNavigation（站内路由/外链确认）。
  Future<bool> _openReviewLink(String url) async {
    await LinkNavigation.open(
      context,
      url,
      baseUrl: ref.read(apiClientProvider).baseUrl,
    );
    return true;
  }

  /// 正文图片：与 wiki/Markdown 阅读一致的灯箱约定（保存/分享）。
  void _openReviewImage(String url) {
    Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          backgroundColor: const Color(0xFF000000),
          body: SafeArea(
            child: GfImageViewer(
              images: <String>[url],
              onSaveImage: (String imageUrl) =>
                  saveImageFromUrl(context, imageUrl),
              saveImageLabel: AppLocalizations.of(context).imageSave,
              onShareImage: (String imageUrl) =>
                  shareImageFromUrl(context, imageUrl),
              shareImageLabel: AppLocalizations.of(context).topicShare,
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _shareReview(CourseDetailPayload course, ReviewPayload review) {
    return showShareImagePreview(
      context,
      fileName: 'course-review-${review.id}.png',
      // 预览与导出共用同一张卡：固定 375 逻辑宽、3× 捕获，主题由预览选择。
      cardBuilder: (theme) => ShareImageCard(
        theme: theme,
        // 卡片浮在更深一档的底色上：浅色主题取 base300，深色主题取更暗的 base200。
        canvasColor: theme.brightness == Brightness.dark
            ? theme.colors.base200
            : theme.colors.base300,
        footerTrailing: Text(
          'yourtj.de',
          style: GfTheme.typographyOf(context).meta.copyWith(
            color: theme.colors.iconMuted,
            fontWeight: FontWeight.w700,
          ),
        ),
        child: CourseReviewShareCard(
          theme: theme,
          course: course,
          review: review,
          offeringLabel: _offeringMeta(course, review.offeringId).chip,
        ),
      ),
    );
  }

  // ---- UI ----

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String title = _detail.value?.name ?? l10n.entryCourses;

    return Scaffold(
      appBar: GfAppBar(
        leading: GfIconButton(
          symbol: 'chevron-left',
          tooltip: l10n.commonBack,
          size: 44,
          onPressed: _goBack,
        ),
        title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
      ),
      body: _detail.when(
        loading: () => const _CourseDetailSkeleton(),
        error: (Object e, StackTrace _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (CourseDetailPayload detail) => _buildBody(l10n, detail),
      ),
    );
  }

  Widget _buildBody(AppLocalizations l10n, CourseDetailPayload detail) {
    final CourseCopy copy = CourseCopy(l10n);
    final bool canWrite = (detail.offerings?.isNotEmpty ?? false);

    return Column(
      children: <Widget>[
        Expanded(
          child: AppRefreshIndicator(
            onRefresh: () async {
              await _load();
              await _loadRelated();
            },
            child: CustomScrollView(
              controller: _scrollController,
              physics: const AlwaysScrollableScrollPhysics(),
              slivers: <Widget>[
                SliverToBoxAdapter(child: _CourseHeader(detail: detail)),
                const SliverToBoxAdapter(child: GfDivider()),
                SliverToBoxAdapter(child: _RatingSection(detail: detail)),
                const SliverToBoxAdapter(child: GfDivider()),
                SliverToBoxAdapter(
                  child: _AiSummaryCard(courseId: widget.courseId),
                ),
                const SliverToBoxAdapter(child: GfDivider()),
                ..._buildReviewsSlivers(l10n, copy, detail),
                const SliverToBoxAdapter(child: GfDivider()),
                SliverToBoxAdapter(
                  child: _OfferingsSection(
                    offerings:
                        detail.offerings ?? const <CourseOfferingPayload>[],
                    activeOfferingId: _activeOfferingId,
                    onFocusOffering: _focusOffering,
                    copy: copy,
                  ),
                ),
                const SliverToBoxAdapter(child: GfDivider()),
                SliverToBoxAdapter(
                  child: _RelatedSection(
                    related: _related,
                    courseId: widget.courseId,
                    onOpenCourse: (int id) =>
                        context.pushReplacement('/courses/$id'),
                    copy: copy,
                  ),
                ),
                SliverToBoxAdapter(
                  child: SizedBox(
                    // dock 现在是真实底栏（不再 overlay），这里只留呼吸间距。
                    height: 24,
                  ),
                ),
              ],
            ),
          ),
        ),
        ColoredBox(
          color: GfTheme.colorsOf(context).base100,
          child: SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  IconButton(
                    tooltip: _bookmarked
                        ? l10n.courseBookmarked
                        : l10n.courseBookmark,
                    icon: GfSymbol(
                      'bookmark',
                      color: _bookmarked
                          ? GfTheme.colorsOf(context).primary
                          : null,
                    ),
                    onPressed: _toggleBookmark,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: GfButton(
                      size: GfButtonSize.extraLarge,
                      expanded: true,
                      onPressed: canWrite ? _openWriteSheet : null,
                      label: l10n.courseWriteReview,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }

  List<Widget> _buildReviewsSlivers(
    AppLocalizations l10n,
    CourseCopy copy,
    CourseDetailPayload detail,
  ) {
    final colors = GfTheme.colorsOf(context);
    return <Widget>[
      SliverToBoxAdapter(
        child: Column(
          key: _reviewsSectionKey,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: Wrap(
                spacing: 6,
                runSpacing: 4,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: <Widget>[
                  Text(
                    l10n.courseDetailReviews,
                    style: GfTheme.typographyOf(
                      context,
                    ).heading.copyWith(fontWeight: FontWeight.w700),
                  ),
                  if (_reviewTotal > 0) ...<Widget>[
                    Text(
                      '$_reviewTotal',
                      style: GfTheme.typographyOf(context).small.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.45),
                      ),
                    ),
                  ],
                ],
              ),
            ),
            if (_activeOfferingId != null)
              _OfferingFocusBanner(
                detail: detail,
                offeringId: _activeOfferingId!,
                copy: copy,
                onClear: _clearOfferingFocus,
              ),
            if (_reviewsLoading)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 24),
                child: Center(child: GfLoadingIndicator()),
              )
            else if (_reviewsLoaded && _reviews.isEmpty)
              GfEmpty(symbol: 'square-pen', message: l10n.reviewsEmpty),
          ],
        ),
      ),
      if (_reviewsLoaded && _reviews.isNotEmpty)
        SliverList(
          delegate: SliverChildBuilderDelegate((context, index) {
            if (index.isOdd) return const GfDivider();
            final review = _reviews[index ~/ 2];
            final offeringMeta = _offeringMeta(detail, review.offeringId);
            return _ReviewRow(
              key: review.id == widget.focusReviewId
                  ? _targetReviewKey
                  : ValueKey(review.id),
              // Course reviews share the rich-content profile, one step more
              // compact than posts and Wiki.
              profile: GfRichContentTypography.of(
                context,
                userScale: ref.watch(contentFontScaleProvider),
                compact: true,
              ),
              review: review,
              offeringMeta: offeringMeta.cardMeta,
              baseUrl: Uri.parse(ref.read(apiClientProvider).baseUrl),
              onImageTap: _openReviewImage,
              onLinkTap: _openReviewLink,
              reportBusy: _reviewReportBusy.contains(review.id),
              onHelpful: () =>
                  _toggleReviewReaction(review, CourseReviewReaction.helpful),
              onDislike: () =>
                  _toggleReviewReaction(review, CourseReviewReaction.dislike),
              onReport: review.viewer.canEdit || !_signedIn
                  ? null
                  : () => _reportReview(review),
              onShare: () => _shareReview(detail, review),
              onEdit: review.viewer.canEdit
                  ? () => _openEditSheet(review)
                  : null,
              onDelete: review.viewer.canDelete
                  ? () => _confirmDeleteReview(review)
                  : null,
            );
          }, childCount: _reviews.length * 2 - 1),
        ),
      if (_reviewsLoaded && _nextCursor.isNotEmpty)
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: Center(
              child: _loadingMore
                  ? const GfLoadingIndicator(small: true)
                  : Text(
                      copy.relatedEmpty,
                      style: GfTheme.typographyOf(context).caption.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.35),
                      ),
                    ),
            ),
          ),
        ),
    ];
  }

  /// 课评卡片的开课信息收敛：委托给 [offeringMetaParts]，详情页只负责按
  /// offeringId 查找当前课程可见的开课实例。
  ({String cardMeta, String chip}) _offeringMeta(
    CourseDetailPayload detail,
    int offeringId,
  ) {
    final CourseOfferingPayload? offering = detail.offerings
        ?.where((CourseOfferingPayload o) => o.id == offeringId)
        .firstOrNull;
    return offeringMetaParts(
      offering,
      locale: AppLocalizations.of(context).localeName,
      fallbackId: offeringId,
    );
  }
}

// ---- 头部 ----

class _CourseHeader extends StatelessWidget {
  const _CourseHeader({required this.detail});

  final CourseDetailPayload detail;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final List<String> legacyNames = detail.legacyNames ?? const <String>[];
    final List<String> aliases = detail.aliases ?? const <String>[];
    final String credit = formatCreditText(detail.creditX10);
    final String teacher = _teacherLabel(copy);

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Expanded(
                child: Text(
                  detail.name,
                  style: type.title2.copyWith(color: colors.baseContent),
                ),
              ),
              const SizedBox(width: 8),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: <Widget>[
                  if (detail.reviewScope == 'team' ||
                      detail.reviewScope == 'course')
                    Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: GfBadge(
                        label: detail.reviewScope == 'team'
                            ? copy.reviewScopeTeam
                            : copy.reviewScopeCourse,
                        variant: GfBadgeVariant.info,
                      ),
                    ),
                  GfBadge(label: detail.primaryCode),
                ],
              ),
            ],
          ),
          if (legacyNames.isNotEmpty) ...<Widget>[
            const SizedBox(height: 6),
            Text(
              '${copy.legacyNamesLabel}${legacyNames.join('、')}',
              style: type.caption.copyWith(
                color: colors.baseContent.withValues(alpha: 0.45),
              ),
            ),
          ],
          const SizedBox(height: 10),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: <Widget>[
              _metaChip(
                context,
                symbol: 'university',
                label: detail.department,
              ),
              _metaChip(context, symbol: 'users-round', label: teacher),
              if (credit.isNotEmpty)
                _metaChip(
                  context,
                  symbol: 'graduation-cap',
                  label: '$credit ${copy.creditUnit}',
                  emphasized: credit,
                ),
            ],
          ),
          if (aliases.isNotEmpty) ...<Widget>[
            const SizedBox(height: 8),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                Text(
                  copy.aliasesLabel,
                  style: type.caption.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.45),
                  ),
                ),
                for (final String alias in aliases)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: colors.base200.withValues(alpha: 0.6),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      alias,
                      style: type.meta.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.7),
                      ),
                    ),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  String _teacherLabel(CourseCopy copy) {
    final List<String> team = (detail.teamInstructors ?? const <String>[])
        .map((name) => name.trim())
        .where((name) => name.isNotEmpty)
        .toSet()
        .toList();
    if (detail.reviewScope == 'team' && team.isNotEmpty) {
      final String joined = team.join('、');
      return team.length > 1
          ? '${copy.teamInstructorsPrefix}$joined'
                '${copy.teamInstructorsSuffix(team.length)}'
          : '${copy.teamInstructorsPrefix}$joined';
    }
    if (detail.teacherName != null && detail.teacherName!.isNotEmpty) {
      return detail.teacherName!;
    }
    return copy.noTeacher;
  }

  Widget _metaChip(
    BuildContext context, {
    required String symbol,
    required String label,
    String? emphasized,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: colors.line),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          GfSymbol(
            symbol,
            size: 14,
            color: colors.primary.withValues(alpha: 0.7),
          ),
          const SizedBox(width: 5),
          Flexible(
            child: Text(
              label,
              style: GfTheme.typographyOf(context).caption.copyWith(
                color: colors.baseContent.withValues(alpha: 0.7),
                fontWeight: emphasized != null ? FontWeight.w600 : null,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _OfferingFocusBanner extends StatelessWidget {
  const _OfferingFocusBanner({
    required this.detail,
    required this.offeringId,
    required this.copy,
    required this.onClear,
  });

  final CourseDetailPayload detail;
  final int offeringId;
  final CourseCopy copy;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final String label = _offeringShortLabel(context);
    return Container(
      margin: const EdgeInsets.fromLTRB(16, 4, 16, 8),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: colors.info.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colors.primary.withValues(alpha: 0.25)),
      ),
      child: Row(
        children: <Widget>[
          GfSymbol('sliders-horizontal', size: 16, color: colors.primary),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              '${copy.offeringFocusLabel} · $label',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: GfTheme.typographyOf(context).caption.copyWith(
                color: colors.baseContent.withValues(alpha: 0.75),
              ),
            ),
          ),
          InkWell(
            onTap: onClear,
            child: Text(
              copy.offeringFocusClear,
              style: GfTheme.typographyOf(context).caption.copyWith(
                color: colors.primary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }

  String _offeringShortLabel(BuildContext context) {
    final CourseOfferingPayload? offering = detail.offerings
        ?.where((CourseOfferingPayload o) => o.id == offeringId)
        .firstOrNull;
    if (offering == null) return '#$offeringId';
    final String classLabel = <String>[
      offering.className?.trim() ?? '',
      offering.classCode?.trim() ?? '',
    ].where((part) => part.isNotEmpty).toSet().join(' · ');
    return <String>[
      shortTerm(
        offering.termCode,
        locale: AppLocalizations.of(context).localeName,
      ),
      classLabel,
    ].join(' · ');
  }
}

// ---- 评分卡 ----

class _RatingSection extends StatelessWidget {
  const _RatingSection({required this.detail});

  final CourseDetailPayload detail;

  @override
  Widget build(BuildContext context) {
    final CourseCopy copy = CourseCopy(AppLocalizations.of(context));
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final int reviewCount = detail.reviewCount ?? 0;
    final double? avg = detail.ratingAvg;
    final List<int> distribution = detail.ratingDistribution ?? const <int>[];
    final bool hasRating = avg != null && avg > 0 && reviewCount > 0;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Wrap(
            spacing: 8,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: <Widget>[
              Text(
                copy.ratingTitle,
                style: type.heading.copyWith(fontWeight: FontWeight.w700),
              ),
              if (reviewCount > 0)
                Text(
                  AppLocalizations.of(context).coursesRatingCount(reviewCount),
                  style: type.caption.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.45),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          if (!hasRating && distribution.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Text(
                copy.noRatingQuiet,
                style: type.small.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.5),
                ),
              ),
            )
          else
            LayoutBuilder(
              builder: (context, constraints) {
                // Keep the score and its distribution readable on narrow phones
                // and when the user enlarges text.
                final Widget ring = _RatingRing(
                  avg: avg,
                  outOf: copy.ratingOutOf,
                  colors: colors,
                  type: type,
                );
                // 分布条 5★ → 1★。
                final Widget bars = Column(
                  children: <Widget>[
                    for (int star = 5; star >= 1; star--)
                      _DistributionRow(
                        star: star,
                        count: distribution.length >= star
                            ? distribution[star - 1]
                            : 0,
                        max: distribution.isEmpty
                            ? 1
                            : distribution.reduce(
                                (int a, int b) => a > b ? a : b,
                              ),
                      ),
                  ],
                );
                // 左环右条（对齐 Web RatingSummaryCard）：极窄宽度下改为纵向堆叠，
                // 保证条形列仍有可读宽度。
                if (constraints.maxWidth < 260) {
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [ring, const SizedBox(height: 16), bars],
                  );
                }
                return Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    ring,
                    const SizedBox(width: 20),
                    Expanded(child: bars),
                  ],
                );
              },
            ),
        ],
      ),
    );
  }
}

class _RatingRing extends StatelessWidget {
  const _RatingRing({
    required this.avg,
    required this.outOf,
    required this.colors,
    required this.type,
  });

  static const double diameter = 104;

  final double? avg;
  final String outOf;
  final GfColors colors;
  final GfTypography type;

  @override
  Widget build(BuildContext context) {
    final double ratio = avg == null || avg! <= 0
        ? 0
        : (avg! / 5).clamp(0.0, 1.0);
    return Semantics(
      label: '${formatRating(avg)} $outOf',
      child: SizedBox.square(
        dimension: diameter,
        child: CustomPaint(
          key: const Key('course-rating-ring'),
          painter: _RatingRingPainter(
            ratio: ratio,
            progressStart: colors.warning,
            progressEnd: colors.primary,
            endDotFill: colors.primary,
            endDotStroke: colors.base100,
            track: colors.base300.withValues(alpha: 0.45),
          ),
          child: Center(
            // 环心内容随字号增长（200% 下 24px 分数会变 48px），但环直径是固定
            // 几何：FittedBox(scaleDown) 把自然尺寸的内容等比缩进环内，既不会
            // RenderFlex 溢出，也不会在 100% 字号下缩小（不放大）。
            child: FittedBox(
              fit: BoxFit.scaleDown,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  GfSymbol('star-filled', size: 14, color: colors.warning),
                  const SizedBox(height: 2),
                  Text(
                    formatRating(avg),
                    key: const ValueKey('course-rating-score'),
                    style: type.title1.copyWith(
                      fontWeight: FontWeight.w700,
                      height: 1.1,
                    ),
                  ),
                  Text(
                    outOf,
                    style: type.label.copyWith(color: colors.iconMuted),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// 均分进度环（对齐 web `RatingSummaryCard.vue`）：底槽圆环 + `warning→primary`
/// 扫掠渐变弧（圆角端点、起点正上方、顺时针）+ 弧尾光点。
class _RatingRingPainter extends CustomPainter {
  const _RatingRingPainter({
    required this.ratio,
    required this.progressStart,
    required this.progressEnd,
    required this.endDotFill,
    required this.endDotStroke,
    required this.track,
  });

  static const double _strokeWidth = 8;
  static const double _endDotRadius = 4.5;
  static const double _endDotStrokeWidth = 2.5;

  final double ratio;

  /// 弧起点色（web `--gf-color-warning`）。
  final Color progressStart;

  /// 弧终点色（web `--gf-color-primary`）。
  final Color progressEnd;

  /// 弧尾光点填充/描边色（web：primary 实心 + base-100 描边）。
  final Color endDotFill;
  final Color endDotStroke;

  final Color track;

  @override
  void paint(Canvas canvas, Size size) {
    final Offset center = size.center(Offset.zero);
    final double radius = (size.shortestSide - _strokeWidth) / 2;
    final Rect arcRect = Rect.fromCircle(center: center, radius: radius);
    final Paint trackPaint = Paint()
      ..color = track
      ..style = PaintingStyle.stroke
      ..strokeWidth = _strokeWidth;
    canvas.drawCircle(center, radius, trackPaint);
    if (ratio <= 0) return;

    final double sweep = 2 * math.pi * ratio;
    final Paint progressPaint = Paint()
      ..shader = ratingRingGradient(
        start: progressStart,
        end: progressEnd,
      ).createShader(arcRect)
      ..style = PaintingStyle.stroke
      ..strokeWidth = _strokeWidth
      ..strokeCap = StrokeCap.round;
    canvas.drawArc(arcRect, -math.pi / 2, sweep, false, progressPaint);

    // 弧尾光点：渐变终点色实心 + 卡片底色描边，把弧尾与底槽分开。
    final double endAngle = sweep - math.pi / 2;
    final Offset end = Offset(
      center.dx + radius * math.cos(endAngle),
      center.dy + radius * math.sin(endAngle),
    );
    canvas.drawCircle(end, _endDotRadius, Paint()..color = endDotFill);
    canvas.drawCircle(
      end,
      _endDotRadius,
      Paint()
        ..color = endDotStroke
        ..style = PaintingStyle.stroke
        ..strokeWidth = _endDotStrokeWidth,
    );
  }

  @override
  bool shouldRepaint(_RatingRingPainter oldDelegate) =>
      oldDelegate.ratio != ratio ||
      oldDelegate.progressStart != progressStart ||
      oldDelegate.progressEnd != progressEnd ||
      oldDelegate.endDotFill != endDotFill ||
      oldDelegate.endDotStroke != endDotStroke ||
      oldDelegate.track != track;
}

class _DistributionRow extends StatelessWidget {
  const _DistributionRow({
    required this.star,
    required this.count,
    required this.max,
  });

  final int star;
  final int count;
  final int max;

  // Web ROW_OPACITY：index 0 = 5★ 满亮，index 4 = 1★ 最暗（高分满亮、低分窄暗）。
  static const List<double> _opacity = <double>[0.95, 0.72, 0.5, 0.34, 0.24];

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final double ratio = max <= 0 ? 0 : count / max;
    // 数字列宽随字号缩放：大字号下 3 位计数不被截断，且统一右对齐。
    final double countWidth = MediaQuery.textScalerOf(context).scale(12) * 2.2;
    return Padding(
      key: ValueKey<String>('rating-distribution-$star'),
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: <Widget>[
          SizedBox(
            width: 16 + MediaQuery.textScalerOf(context).scale(12),
            child: Row(
              children: <Widget>[
                GfSymbol(
                  'star-filled',
                  size: 12,
                  color: colors.baseContent.withValues(alpha: 0.3),
                ),
                const SizedBox(width: 2),
                Text(
                  '$star',
                  style: GfTheme.typographyOf(
                    context,
                  ).meta.copyWith(color: colors.iconMuted),
                ),
              ],
            ),
          ),
          Expanded(
            child: Container(
              height: 6,
              decoration: BoxDecoration(
                color: colors.base300.withValues(alpha: 0.45),
                borderRadius: BorderRadius.circular(999),
              ),
              child: FractionallySizedBox(
                alignment: Alignment.centerLeft,
                widthFactor: ratio,
                child: Container(
                  decoration: BoxDecoration(
                    color: colors.warning.withValues(alpha: _opacity[5 - star]),
                    borderRadius: BorderRadius.circular(999),
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: 8),
          SizedBox(
            width: countWidth,
            child: Text(
              '$count',
              textAlign: TextAlign.right,
              style: GfTheme.typographyOf(
                context,
              ).meta.copyWith(color: colors.iconMuted),
            ),
          ),
        ],
      ),
    );
  }
}

// ---- AI 总结卡 ----

/// AI 课程总结卡（对齐 web AISummaryCard.vue 状态机，轻量版）。
///
/// disabled → 不渲染；cached/generated → 默认折叠；none → 首次展开触发
/// 生成；insufficient_data → 安静提示；ready 态刷新保留内容、失败瞬态提示。
class _AiSummaryCard extends ConsumerStatefulWidget {
  const _AiSummaryCard({required this.courseId});

  final int courseId;

  @override
  ConsumerState<_AiSummaryCard> createState() => _AiSummaryCardState();
}

class _AiSummaryCardState extends ConsumerState<_AiSummaryCard> {
  static const List<String> _consensusOrder = <String>[
    'strong_recommend',
    'recommend',
    'neutral',
    'cautious',
    'not_recommend',
  ];

  CourseRepository get _repository => ref.read(courseRepositoryProvider);

  // idle=折叠未生成；loading=生成中；ready=有内容；insufficient；error。
  String _status = 'idle';
  CourseAiSummaryPayload? _summary;
  bool _expanded = false;
  bool _refreshing = false;
  String? _refreshNotice;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    try {
      final CourseAiSummaryResult result = await _repository.aiSummary(
        widget.courseId,
        check: true,
      );
      if (!mounted) return;
      switch (result.status) {
        case 'cached':
        case 'generated':
          if (result.summary != null) {
            setState(() {
              _status = 'ready';
              _summary = result.summary;
            });
          }
          break;
        case 'insufficient_data':
          setState(() => _status = 'insufficient');
          break;
        case 'disabled':
          // 不渲染整个卡片。
          setState(() => _status = 'disabled');
          break;
        default:
          // none / 未知：折叠，首次展开触发生成。
          break;
      }
    } catch (_) {
      // 预检网络失败：保持折叠静默。
    }
  }

  Future<void> _load({bool refresh = false}) async {
    if (_refreshing) return;
    final bool keepContent = _status == 'ready' && refresh;
    if (!keepContent && !refresh) {
      setState(() => _status = 'loading');
    }
    setState(() => _refreshing = true);
    try {
      final CourseAiSummaryResult result = await _repository.aiSummary(
        widget.courseId,
        refresh: refresh,
      );
      if (!mounted) return;
      setState(() {
        _refreshNotice = null;
        switch (result.status) {
          case 'cached':
          case 'generated':
            if (result.summary != null) {
              _status = 'ready';
              _summary = result.summary;
              _expanded = true;
            } else if (!keepContent) {
              _status = 'error';
            }
            break;
          case 'insufficient_data':
            _status = 'insufficient';
            break;
          case 'disabled':
            _status = 'disabled';
            break;
          default:
            if (!keepContent) _status = 'error';
        }
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        if (keepContent) {
          _refreshNotice = CourseCopy(
            AppLocalizations.of(context),
          ).summaryLoadFailed;
        } else {
          _status = 'error';
        }
      });
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  void _onToggle() {
    if (_status == 'ready') {
      setState(() => _expanded = !_expanded);
      return;
    }
    if (_status == 'idle' || _status == 'error' || _status == 'insufficient') {
      if (_status == 'insufficient') {
        // 数据不足为终态提示，仅做展示切换。
        setState(() => _expanded = !_expanded);
        return;
      }
      setState(() => _expanded = true);
      _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_status == 'disabled') return const SizedBox.shrink();
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final CourseAiSummaryPayload? payload = _summary;
    final bool ready = _status == 'ready' && payload != null;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Row(
          children: <Widget>[
            Expanded(
              child: InkWell(
                onTap: _onToggle,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 9, 8, 9),
                  child: Row(
                    children: <Widget>[
                      // 小方标：主色淡底 + sparkles，给 AI 区一个稳定的视觉锚点。
                      Container(
                        width: 26,
                        height: 26,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: colors.primary.withValues(alpha: 0.1),
                          borderRadius: BorderRadius.circular(
                            GfTheme.radiiOf(context).selector,
                          ),
                        ),
                        child: GfSymbol(
                          'sparkles',
                          size: 15,
                          color: colors.primary,
                        ),
                      ),
                      const SizedBox(width: 10),
                      Flexible(
                        child: Text(
                          l10n.courseDetailAiSummary,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.small.copyWith(
                            fontWeight: FontWeight.w700,
                            color: colors.baseContent,
                          ),
                        ),
                      ),
                      // 结论 pill 跟在标题后：折叠时也能一眼看到「推荐/谨慎」，
                      // 不再单独占一行。
                      if (ready) ...<Widget>[
                        const SizedBox(width: 8),
                        _ConsensusBadge(
                          level: _consensusLevel(payload),
                          copy: copy,
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ),
            // 刷新与展开是一组并排的图标按钮（同宽 40、高 44 命中区），贴右对齐；
            // 展开 chevron 的图标右缘与正文 16dp 边距对齐。
            if (ready)
              IconButton(
                key: const ValueKey<String>('ai-summary-refresh'),
                icon: _refreshing
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: GfProgressIndicator(strokeWidth: 2),
                      )
                    : const GfSymbol('refresh-cw', size: 18),
                tooltip: copy.summaryRefresh,
                onPressed: _refreshing ? null : () => _load(refresh: true),
                constraints: const BoxConstraints.tightFor(
                  width: 40,
                  height: 44,
                ),
                padding: EdgeInsets.zero,
                color: colors.iconMuted,
              ),
            IconButton(
              key: const ValueKey<String>('ai-summary-toggle'),
              icon: GfSymbol(
                _expanded ? 'chevron-up' : 'chevron-down',
                size: 18,
              ),
              tooltip: _expanded ? copy.summaryCollapse : copy.summaryExpand,
              onPressed: _onToggle,
              constraints: const BoxConstraints.tightFor(width: 40, height: 44),
              padding: EdgeInsets.zero,
              color: colors.iconMuted,
            ),
            const SizedBox(width: 5),
          ],
        ),
        if (_expanded)
          Padding(
            padding: const EdgeInsets.only(bottom: 16),
            child: _buildContent(copy, colors, type),
          ),
      ],
    );
  }

  String _consensusLevel(CourseAiSummaryPayload payload) =>
      _consensusOrder.contains(payload.consensus)
      ? payload.consensus
      : 'neutral';

  Widget _buildContent(CourseCopy copy, GfColors colors, GfTypography type) {
    const EdgeInsets inset = EdgeInsets.symmetric(horizontal: 16);
    if (_status == 'loading') {
      // 骨架行宽与 Web 对齐（3/4、满行、1/2），组内间距 12。
      return const Padding(
        padding: EdgeInsets.fromLTRB(16, 4, 16, 0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            FractionallySizedBox(
              widthFactor: 0.75,
              child: GfSkeleton(height: 14, radius: 5),
            ),
            SizedBox(height: 12),
            GfSkeleton(height: 14, radius: 5),
            SizedBox(height: 12),
            FractionallySizedBox(
              widthFactor: 0.5,
              child: GfSkeleton(height: 14, radius: 5),
            ),
          ],
        ),
      );
    }
    if (_status == 'insufficient') {
      return Padding(
        padding: inset,
        child: Text(
          copy.summaryInsufficient,
          style: type.caption.copyWith(
            color: colors.baseContent.withValues(alpha: 0.6),
          ),
        ),
      );
    }
    if (_status == 'error') {
      return Padding(
        padding: const EdgeInsets.only(left: 16, right: 4),
        child: Row(
          children: <Widget>[
            Expanded(
              child: Text(
                copy.summaryLoadFailed,
                style: type.caption.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.6),
                ),
              ),
            ),
            TextButton(
              onPressed: () => _load(),
              child: Text(copy.summaryRefresh),
            ),
          ],
        ),
      );
    }
    final CourseAiSummaryPayload? payload = _summary;
    if (payload == null) return const SizedBox.shrink();
    final List<String> keywords = payload.keywords.take(6).toList();
    final List<CourseAiSummaryRepresentativeReview> quotes = payload
        .representativeReviews
        .take(3)
        .toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        // 一句话结论做导语，正文字号，不再另起标签行。
        Padding(
          padding: inset,
          child: Text(
            copy.summaryConsensusText(_consensusLevel(payload)),
            style: type.bodyStrong.copyWith(color: colors.baseContent),
          ),
        ),
        // 关键词：单行横滑 hashtag，宽度不够时不换行、不截断，边缘渐隐提示可滑。
        if (keywords.isNotEmpty) ...<Widget>[
          const SizedBox(height: 10),
          Semantics(
            label: copy.summaryKeywords,
            child: _EdgeFadeScroller(
              child: Row(
                children: <Widget>[
                  for (int i = 0; i < keywords.length; i++) ...<Widget>[
                    if (i > 0) const SizedBox(width: 6),
                    _KeywordTag(label: keywords[i]),
                  ],
                ],
              ),
            ),
          ),
        ],
        // 优缺点合成一列：前置 +/− 圆点区分，不再各占标题行、也不分两栏高低不齐。
        if (payload.pros.isNotEmpty || payload.cons.isNotEmpty) ...<Widget>[
          const SizedBox(height: 14),
          Padding(
            padding: inset,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                for (final String item in payload.pros.take(4))
                  _SummaryPoint(
                    text: item,
                    positive: true,
                    semanticsPrefix: copy.summaryPros,
                  ),
                // 优→缺转折处多留 4dp，扫读时两组自然分开（仍是单列）。
                if (payload.pros.isNotEmpty && payload.cons.isNotEmpty)
                  const SizedBox(height: 4),
                for (final String item in payload.cons.take(4))
                  _SummaryPoint(
                    text: item,
                    positive: false,
                    semanticsPrefix: copy.summaryCons,
                  ),
              ],
            ),
          ),
        ],
        // 代表性评价：回复引用样式；多条时等高横滑卡片，下一张露边提示可滑。
        if (quotes.isNotEmpty) ...<Widget>[
          const SizedBox(height: 6),
          Semantics(
            label: copy.summaryRepresentativeReviews,
            child: quotes.length == 1
                ? Padding(
                    padding: inset,
                    child: _SummaryQuote(item: quotes.single, copy: copy),
                  )
                : LayoutBuilder(
                    builder: (context, constraints) {
                      final double width = (constraints.maxWidth - 32) * 0.84;
                      return _EdgeFadeScroller(
                        child: IntrinsicHeight(
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: <Widget>[
                              for (
                                int i = 0;
                                i < quotes.length;
                                i++
                              ) ...<Widget>[
                                if (i > 0) const SizedBox(width: 8),
                                SizedBox(
                                  width: width,
                                  child: _SummaryQuote(
                                    item: quotes[i],
                                    copy: copy,
                                  ),
                                ),
                              ],
                            ],
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
        const SizedBox(height: 12),
        Padding(
          padding: inset,
          child: Text(
            _refreshNotice ?? copy.summaryDisclaimer,
            style: type.meta.copyWith(
              fontWeight: FontWeight.w400,
              color: _refreshNotice != null
                  ? colors.error
                  : colors.baseContent.withValues(alpha: 0.55),
            ),
          ),
        ),
      ],
    );
  }
}

class _ConsensusBadge extends StatelessWidget {
  const _ConsensusBadge({required this.level, required this.copy});

  final String level;
  final CourseCopy copy;

  @override
  Widget build(BuildContext context) {
    final GfBadgeVariant variant = switch (level) {
      'strong_recommend' => GfBadgeVariant.success,
      'recommend' => GfBadgeVariant.info,
      'cautious' => GfBadgeVariant.warning,
      'not_recommend' => GfBadgeVariant.error,
      _ => GfBadgeVariant.muted,
    };
    return GfBadge(label: copy.summaryConsensus(level), variant: variant);
  }
}

/// 横向滚动行：左右 16dp 内边距与正文对齐，边缘渐隐只落在内边距上。
class _EdgeFadeScroller extends StatelessWidget {
  const _EdgeFadeScroller({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) => ShaderMask(
    shaderCallback: (Rect bounds) {
      final double gutter = (16 / bounds.width).clamp(0.0, .25);
      return LinearGradient(
        colors: const <Color>[
          Colors.transparent,
          Colors.black,
          Colors.black,
          Colors.transparent,
        ],
        stops: <double>[0, gutter, 1 - gutter, 1],
      ).createShader(bounds);
    },
    blendMode: BlendMode.dstIn,
    child: SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: child,
    ),
  );
}

class _KeywordTag extends StatelessWidget {
  const _KeywordTag({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).selector),
      ),
      child: Text.rich(
        TextSpan(
          children: <InlineSpan>[
            TextSpan(
              text: '#',
              style: TextStyle(color: colors.primary),
            ),
            TextSpan(text: label),
          ],
        ),
        maxLines: 1,
        style: type.caption.copyWith(
          height: 1.4,
          fontWeight: FontWeight.w500,
          color: colors.baseContent.withValues(alpha: 0.8),
        ),
      ),
    );
  }
}

class _SummaryPoint extends StatelessWidget {
  const _SummaryPoint({
    required this.text,
    required this.positive,
    required this.semanticsPrefix,
  });

  final String text;
  final bool positive;
  final String semanticsPrefix;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final Color tone = positive ? colors.success : colors.error;
    return Semantics(
      label: '$semanticsPrefix：$text',
      excludeSemantics: true,
      child: Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            // 18dp 圆点与 15/21 正文首行垂直居中（(21-18)/2）。
            Container(
              key: ValueKey<String>(positive ? 'summary-pro' : 'summary-con'),
              margin: const EdgeInsets.only(top: 1.5),
              width: 18,
              height: 18,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: tone.withValues(alpha: 0.14),
                shape: BoxShape.circle,
              ),
              child: GfSymbol(
                positive ? 'plus' : 'minus',
                size: 12,
                color: tone,
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Text(
                text,
                style: type.small.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.85),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 代表性评价：与课评卡同族的中性小卡（base200 + box 圆角），
/// 情感只用一个小圆点 + 弱化标签表达，不用彩条/彩底。
class _SummaryQuote extends StatelessWidget {
  const _SummaryQuote({required this.item, required this.copy});

  final CourseAiSummaryRepresentativeReview item;
  final CourseCopy copy;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final Color tone = switch (item.sentiment) {
      'positive' => colors.success,
      'negative' => colors.error,
      _ => colors.iconMuted,
    };
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 12),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          Row(
            children: <Widget>[
              Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(color: tone, shape: BoxShape.circle),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  copy.summarySentiment(item.sentiment),
                  style: type.meta.copyWith(
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent.withValues(alpha: 0.65),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Text(
            item.excerpt,
            style: type.caption.copyWith(
              color: colors.baseContent.withValues(alpha: 0.85),
            ),
          ),
        ],
      ),
    );
  }
}

// ---- 课评行 ----

class _ReviewRow extends StatelessWidget {
  const _ReviewRow({
    super.key,
    required this.profile,
    required this.review,
    required this.offeringMeta,
    required this.baseUrl,
    required this.onImageTap,
    required this.onLinkTap,
    required this.reportBusy,
    required this.onHelpful,
    required this.onDislike,
    required this.onShare,
    this.onReport,
    this.onEdit,
    this.onDelete,
  });

  final GfRichContentTypography profile;
  final ReviewPayload review;

  /// 单行元信息（学期 · 班次 · 教师），日期由卡片拼短格式。
  final String offeringMeta;

  /// 正文相对链接与图片的解析基准（站点/API 同源）。
  final Uri baseUrl;
  final ValueChanged<String> onImageTap;
  final Future<bool> Function(String url) onLinkTap;
  final bool reportBusy;
  final VoidCallback onHelpful;
  final VoidCallback onDislike;
  final VoidCallback onShare;
  final VoidCallback? onReport;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final int? rating = review.rating;
    // 自己的评价把编辑/删除收进右上溢出菜单；他人评价（已登录）只留举报。
    final bool hasMenu = onEdit != null || onDelete != null || onReport != null;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              reviewAvatar(review, size: 40),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      reviewAuthorLabel(review, copy),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.small.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Expanded(
                          child: Text(
                            // 单行元信息：学期 · 班次 · 教师 · 短日期。班号/校区/
                            // 院系已在课程头部与开课记录里，不在卡片重复；短日期（相对/
                            // M月D日）让常规字号下能单行放下，过长才换行、不截断。
                            <String>[
                                  offeringMeta,
                                  formatReviewDate(
                                    review.createdAt,
                                    l10n: l10n,
                                  ),
                                ]
                                .where((String part) => part.isNotEmpty)
                                .join(' · '),
                            style: type.meta.copyWith(color: colors.iconMuted),
                          ),
                        ),
                        if (rating != null && rating > 0) ...<Widget>[
                          const SizedBox(width: 8),
                          for (int star = 1; star <= 5; star++)
                            GfSymbol(
                              star <= rating ? 'star-filled' : 'star',
                              size: 15,
                              color: star <= rating
                                  ? colors.warning
                                  : colors.baseContent.withValues(alpha: 0.2),
                            ),
                        ],
                      ],
                    ),
                  ],
                ),
              ),
              if (hasMenu) ...<Widget>[
                const SizedBox(width: 4),
                _overflowMenu(context, l10n, copy, colors),
              ],
            ],
          ),
          const SizedBox(height: 8),
          // 服务端 `contentHtml` 已归一化历史课评标题,移动端直接消费,
          // 不再渲染原始 Markdown 文本。相对图片/链接按站点源解析；图片点击
          // 走 App 图片查看器，链接走共享的 LinkNavigation。
          GfHtmlContent(
            html: review.contentHtml,
            profile: profile,
            baseUrl: baseUrl,
            onTapUrl: onLinkTap,
            customWidgetBuilder: (element) {
              if (element.localName != 'img') return null;
              final String? src = element.attributes['src'];
              // data: URI 交给 fwfh 内置 data-image 渲染,不当作网络图处理。
              if (src == null || src.isEmpty || src.startsWith('data:')) {
                return null;
              }
              final String resolved = resolveApiAssetUrl(src);
              final String? alt = element.attributes['alt'];
              return Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                alignment: Alignment.center,
                child: GestureDetector(
                  onTap: () => onImageTap(resolved),
                  child: Container(
                    decoration: BoxDecoration(
                      border: Border.all(color: colors.line),
                      borderRadius: BorderRadius.circular(
                        GfTheme.radiiOf(context).box,
                      ),
                    ),
                    clipBehavior: Clip.antiAlias,
                    child: GfNetworkImage(
                      resolved,
                      fit: BoxFit.contain,
                      cacheWidth:
                          (MediaQuery.sizeOf(context).width *
                                  MediaQuery.devicePixelRatioOf(context))
                              .round(),
                      semanticLabel: alt,
                      errorBuilder: (_, _, _) =>
                          const GfSymbol('image-off', size: 20),
                    ),
                  ),
                ),
              );
            },
          ),
          const SizedBox(height: 8),
          _ReviewActionBar(
            review: review,
            onHelpful: onHelpful,
            onDislike: onDislike,
            onShare: onShare,
          ),
        ],
      ),
    );
  }

  /// 卡片右上溢出菜单（与帖子/话题同一范式）：自己的评价是编辑/删除，
  /// 他人评价（已登录）是举报；一个都没有时调用方不渲染按钮。
  Widget _overflowMenu(
    BuildContext context,
    AppLocalizations l10n,
    CourseCopy copy,
    GfColors colors,
  ) {
    return PopupMenuButton<String>(
      key: ValueKey<String>('review-menu-${review.id}'),
      tooltip: l10n.profileMore,
      icon: const GfSymbol('ellipsis', size: 20),
      useRootNavigator: true,
      onSelected: (String value) {
        switch (value) {
          case 'edit':
            onEdit?.call();
          case 'delete':
            onDelete?.call();
          case 'report':
            onReport?.call();
        }
      },
      itemBuilder: (_) => <PopupMenuEntry<String>>[
        if (onEdit != null)
          PopupMenuItem<String>(value: 'edit', child: Text(l10n.commonEdit)),
        if (onDelete != null)
          PopupMenuItem<String>(
            value: 'delete',
            child: Text(copy.delete, style: TextStyle(color: colors.error)),
          ),
        if (onReport != null)
          PopupMenuItem<String>(
            value: 'report',
            enabled: !reportBusy,
            child: Text(l10n.contentReport),
          ),
      ],
    );
  }
}

/// 课评卡片底部功能区：**单行**（横向可滚）三项——有用计数 / 无用计数 / 分享。
///
/// 可见 pill 高 32dp（13pt 标签、16pt 图标、左右 12dp 内边距），命中区仍是
/// 44×44 的透明外扩（better-accessibility：视觉缩小、触控目标不缩水）。标签只保留计数与短文案，不再把「有用 / 无用」长文案
/// 重复一遍（320px + 200% 字号下也不会折行或出现按钮孤行）。写入期间按钮
/// 保持 enabled（Material disabled 会改色/透明，造成闪烁）；重复点击由
/// `_toggleReviewReaction` 的 busy 守卫丢弃。
class _ReviewActionBar extends StatelessWidget {
  const _ReviewActionBar({
    required this.review,
    required this.onHelpful,
    required this.onDislike,
    required this.onShare,
  });

  final ReviewPayload review;
  final VoidCallback onHelpful;
  final VoidCallback onDislike;
  final VoidCallback onShare;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final int helpfulCount = review.helpfulCount;
    final int dislikeCount = review.dislikeCount;
    // 计数标签固定宽度 + tabular figures：1→10 或点击忙碌都不会改变 chip
    // 宽度，整行动作不会因为计数进位而横向抽动。
    final double countLabelWidth =
        MediaQuery.textScalerOf(context).scale(13) * 1.8;

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: <Widget>[
          _actionChip(
            context,
            key: ValueKey<String>('review-helpful-${review.id}'),
            label: '$helpfulCount',
            labelWidth: countLabelWidth,
            semanticsLabel: '${l10n.reviewHelpful} $helpfulCount',
            symbol: 'thumbs-up',
            active: review.viewer.isHelpful,
            selectedSemantics: true,
            activeColor: colors.warning,
            onTap: onHelpful,
          ),
          const SizedBox(width: 8),
          _actionChip(
            context,
            key: ValueKey<String>('review-dislike-${review.id}'),
            label: '$dislikeCount',
            labelWidth: countLabelWidth,
            semanticsLabel: '${l10n.reviewDislike} $dislikeCount',
            symbol: 'thumbs-down',
            active: review.viewer.isDisliked,
            selectedSemantics: true,
            activeColor: colors.error,
            onTap: onDislike,
          ),
          const SizedBox(width: 8),
          _actionChip(
            context,
            key: ValueKey<String>('review-share-${review.id}'),
            label: l10n.courseReviewShare,
            symbol: 'share-2',
            active: false,
            onTap: onShare,
          ),
        ],
      ),
    );
  }

  Widget _actionChip(
    BuildContext context, {
    required Key key,
    required String label,
    required String symbol,
    required bool active,
    VoidCallback? onTap,
    double? labelWidth,
    Color? activeColor,
    String? semanticsLabel,
    bool selectedSemantics = false,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color? tint = active ? (activeColor ?? colors.primary) : null;
    // MergeSemantics 把 label/toggled 与 TextButton 的可点击/按钮/启用语义
    // 合并成同一个节点：外层 Semantics 单独存在时只有 label 没有 onTap，
    // uiautomator/读屏会看到一个不可点击的按钮节点。
    // 命中区保持 44dp，但可见 pill 只有 32dp（better-accessibility/hit-areas：
    // 视觉尺寸做小、透明外扩命中区），信息密度更接近阅读型列表。
    return MergeSemantics(
      child: Semantics(
        key: key,
        toggled: selectedSemantics ? active : null,
        button: true,
        label: semanticsLabel,
        child: TextButton(
          onPressed: onTap,
          style: TextButton.styleFrom(
            minimumSize: const Size(44, 44),
            padding: EdgeInsets.zero,
            foregroundColor: tint ?? colors.baseContent.withValues(alpha: 0.7),
            backgroundColor: Colors.transparent,
            shape: const StadiumBorder(),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          ),
          child: Center(
            child: Container(
              height: 32,
              padding: const EdgeInsets.symmetric(horizontal: 12),
              decoration: BoxDecoration(
                color: active
                    ? (tint ?? colors.primary).withValues(alpha: 0.1)
                    : colors.base100,
                borderRadius: BorderRadius.circular(999),
                border: Border.all(
                  color: active
                      ? (tint ?? colors.primary).withValues(alpha: 0.4)
                      : colors.line.withValues(alpha: 0.7),
                ),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  if (symbol == 'thumbs-down')
                    const RotatedBox(
                      quarterTurns: 2,
                      child: GfSymbol('thumbs-up', size: 16),
                    )
                  else
                    GfSymbol(symbol, size: 16),
                  const SizedBox(width: 6),
                  SizedBox(
                    width: labelWidth,
                    child: Text(
                      label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        fontSize: 13,
                        height: 1.2,
                        color:
                            tint ?? colors.baseContent.withValues(alpha: 0.7),
                        fontFeatures: labelWidth == null
                            ? null
                            : const <FontFeature>[FontFeature.tabularFigures()],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ---- 开课记录 ----

class _OfferingsSection extends StatefulWidget {
  const _OfferingsSection({
    required this.offerings,
    required this.activeOfferingId,
    required this.onFocusOffering,
    required this.copy,
  });

  final List<CourseOfferingPayload> offerings;
  final int? activeOfferingId;
  final ValueChanged<int> onFocusOffering;
  final CourseCopy copy;

  @override
  State<_OfferingsSection> createState() => _OfferingsSectionState();
}

class _OfferingsSectionState extends State<_OfferingsSection> {
  final Set<String> _collapsedTerms = <String>{};

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final GfColors colors = GfTheme.colorsOf(context);

    // 按 termCode 分组（服务端已按学期排序，保持原序）。
    final Map<String, List<CourseOfferingPayload>> groups =
        <String, List<CourseOfferingPayload>>{};
    for (final CourseOfferingPayload offering in widget.offerings) {
      groups
          .putIfAbsent(offering.termCode, () => <CourseOfferingPayload>[])
          .add(offering);
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text(
            l10n.courseDetailOfferings,
            style: type.heading.copyWith(fontWeight: FontWeight.w700),
          ),
        ),
        if (widget.offerings.isEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              widget.copy.offeringsEmpty,
              style: type.small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          )
        else
          for (final MapEntry<String, List<CourseOfferingPayload>> entry
              in groups.entries)
            _OfferingTermGroup(
              termCode: entry.key,
              offerings: entry.value,
              collapsed: _collapsedTerms.contains(entry.key),
              onToggleCollapsed: () {
                setState(() {
                  if (!_collapsedTerms.remove(entry.key)) {
                    _collapsedTerms.add(entry.key);
                  }
                });
              },
              activeOfferingId: widget.activeOfferingId,
              onFocusOffering: widget.onFocusOffering,
            ),
      ],
    );
  }
}

class _OfferingTermGroup extends StatelessWidget {
  const _OfferingTermGroup({
    required this.termCode,
    required this.offerings,
    required this.collapsed,
    required this.onToggleCollapsed,
    required this.activeOfferingId,
    required this.onFocusOffering,
  });

  final String termCode;
  final List<CourseOfferingPayload> offerings;
  final bool collapsed;
  final VoidCallback onToggleCollapsed;
  final int? activeOfferingId;
  final ValueChanged<int> onFocusOffering;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final String termName =
        offerings.first.termName ??
        shortTerm(termCode, locale: AppLocalizations.of(context).localeName);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        InkWell(
          onTap: onToggleCollapsed,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 4),
            child: Row(
              children: <Widget>[
                GfSymbol(
                  collapsed ? 'chevron-right' : 'chevron-down',
                  size: 18,
                  color: colors.baseContent.withValues(alpha: 0.45),
                ),
                const SizedBox(width: 4),
                Flexible(
                  child: Text(
                    termName,
                    style: type.small.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent.withValues(alpha: 0.75),
                    ),
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  '${offerings.length}',
                  style: type.caption.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.4),
                  ),
                ),
              ],
            ),
          ),
        ),
        if (!collapsed)
          for (final CourseOfferingPayload offering in offerings)
            _OfferingRow(
              offering: offering,
              active: activeOfferingId == offering.id,
              onTap: () => onFocusOffering(offering.id),
            ),
      ],
    );
  }
}

class _OfferingRow extends StatelessWidget {
  const _OfferingRow({
    required this.offering,
    required this.active,
    required this.onTap,
  });

  final CourseOfferingPayload offering;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final String classLabel = <String>[
      offering.className?.trim() ?? '',
      offering.classCode?.trim() ?? '',
    ].where((part) => part.isNotEmpty).toSet().join(' · ');
    final List<String> meta = <String>[
      offering.campus ?? '',
      offering.faculty?.trim() ?? '',
      ...?offering.instructors?.map((name) => name.trim()),
    ].where((String s) => s.isNotEmpty).toSet().toList();
    final double? ratingAvg = offering.ratingAvg;
    final int reviewCount = offering.reviewCount ?? 0;

    return InkWell(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.fromLTRB(16, 2, 16, 2),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: active ? colors.info.withValues(alpha: 0.08) : null,
          borderRadius: BorderRadius.circular(8),
          border: active
              ? Border.all(color: colors.primary.withValues(alpha: 0.35))
              : null,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                Expanded(
                  child: Text(
                    classLabel.isEmpty ? '#${offering.id}' : classLabel,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: type.small.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent.withValues(alpha: 0.85),
                    ),
                  ),
                ),
                if (ratingAvg != null && ratingAvg > 0) ...<Widget>[
                  GfSymbol('star-filled', size: 13, color: colors.warning),
                  const SizedBox(width: 2),
                  Text(
                    formatRating(ratingAvg),
                    style: type.caption.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  if (reviewCount > 0) ...<Widget>[
                    const SizedBox(width: 2),
                    Text(
                      '($reviewCount)',
                      style: type.caption.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.45),
                      ),
                    ),
                  ],
                ],
              ],
            ),
            if (meta.isNotEmpty) ...<Widget>[
              const SizedBox(height: 2),
              Text(
                meta.join(' · '),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: type.caption.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// ---- 相关课程 + 沿革 ----

class _RelatedSection extends StatelessWidget {
  const _RelatedSection({
    required this.related,
    required this.courseId,
    required this.onOpenCourse,
    required this.copy,
  });

  final AsyncValue<CourseRelatedResult> related;
  final int courseId;
  final ValueChanged<int> onOpenCourse;
  final CourseCopy copy;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text(
            l10n.courseDetailRelated,
            style: type.heading.copyWith(fontWeight: FontWeight.w700),
          ),
        ),
        related.when(
          loading: () => const Padding(
            padding: EdgeInsets.all(20),
            child: Center(child: GfLoadingIndicator(small: true)),
          ),
          error: (Object e, StackTrace _) => Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              AppLocalizations.of(context).commonLoadFailed,
              style: type.small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          ),
          data: (CourseRelatedResult result) {
            final List<RelatedCourseItem> teacherCourses =
                result.teacherOtherCourses;
            final List<RelatedCourseItem> otherTeachers =
                result.sameCourseOtherTeachers;
            final List<RelationItem> lineage = result.lineage ?? const [];
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                if (teacherCourses.isNotEmpty)
                  _RelatedGroup(
                    title: copy.relatedTeacherCoursesTitle,
                    items: teacherCourses,
                    onOpenCourse: onOpenCourse,
                  ),
                if (otherTeachers.isNotEmpty)
                  _RelatedGroup(
                    title: copy.relatedOtherTeachersTitle,
                    items: otherTeachers,
                    onOpenCourse: onOpenCourse,
                  ),
                if (teacherCourses.isEmpty && otherTeachers.isEmpty)
                  Padding(
                    padding: const EdgeInsets.all(16),
                    child: Text(
                      copy.relatedEmpty,
                      style: type.small.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.5),
                      ),
                    ),
                  ),
                if (lineage.isNotEmpty) ...<Widget>[
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
                    child: Text(
                      l10n.courseDetailLineage,
                      style: type.small.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent.withValues(alpha: 0.75),
                      ),
                    ),
                  ),
                  for (final RelationItem item in lineage)
                    _LineageRow(
                      item: item,
                      courseId: courseId,
                      onOpenCourse: onOpenCourse,
                    ),
                ],
              ],
            );
          },
        ),
      ],
    );
  }
}

class _RelatedGroup extends StatelessWidget {
  const _RelatedGroup({
    required this.title,
    required this.items,
    required this.onOpenCourse,
  });

  final String title;
  final List<RelatedCourseItem> items;
  final ValueChanged<int> onOpenCourse;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 2),
          child: Text(
            title,
            style: type.caption.copyWith(
              color: colors.baseContent.withValues(alpha: 0.7),
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        for (final RelatedCourseItem item in items)
          InkWell(
            onTap: () => onOpenCourse(item.id),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 7),
              child: Row(
                children: <Widget>[
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          item.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.small.copyWith(
                            fontWeight: FontWeight.w600,
                            color: colors.baseContent,
                          ),
                        ),
                        const SizedBox(height: 1),
                        Text(
                          <String>[
                            item.primaryCode,
                            item.teacherName ?? '',
                          ].where((String s) => s.isNotEmpty).join(' · '),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: type.meta.copyWith(
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  if (item.ratingAvg > 0) ...<Widget>[
                    GfSymbol('star-filled', size: 13, color: colors.warning),
                    const SizedBox(width: 2),
                    Text(
                      formatRating(item.ratingAvg),
                      style: type.caption.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
                    ),
                  ],
                  const SizedBox(width: 8),
                  GfSymbol(
                    'chevron-right',
                    size: 16,
                    color: colors.baseContent.withValues(alpha: 0.35),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}

class _LineageRow extends StatelessWidget {
  const _LineageRow({
    required this.item,
    required this.courseId,
    required this.onOpenCourse,
  });

  final RelationItem item;
  final int courseId;
  final ValueChanged<int> onOpenCourse;

  @override
  Widget build(BuildContext context) {
    final CourseCopy copy = CourseCopy(AppLocalizations.of(context));
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final bool fromClickable =
        item.direction == 'to' && item.status != 'merged';
    final bool toClickable =
        item.direction == 'from' && item.status != 'merged';

    Widget name(String text, {required bool clickable}) {
      return Expanded(
        child: clickable
            ? InkWell(
                onTap: () => onOpenCourse(
                  item.direction == 'to' ? item.fromCourseId : item.toCourseId,
                ),
                child: Text(
                  text,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: type.small.copyWith(
                    color: colors.primary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              )
            : Text(
                text,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: type.small.copyWith(
                  color: colors.baseContent.withValues(alpha: 0.65),
                ),
              ),
      );
    }

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Row(
        children: <Widget>[
          name(item.fromName, clickable: fromClickable),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: GfSymbol(
              'chevron-right',
              size: 14,
              color: colors.baseContent.withValues(alpha: 0.35),
            ),
          ),
          name(item.toName, clickable: toClickable),
          const SizedBox(width: 8),
          GfBadge(
            label: copy.relationLabel(item.relationType),
            variant: item.status == 'merged'
                ? GfBadgeVariant.muted
                : GfBadgeVariant.info,
          ),
        ],
      ),
    );
  }
}

// ---- 骨架 ----

class _CourseDetailSkeleton extends StatelessWidget {
  const _CourseDetailSkeleton();

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const NeverScrollableScrollPhysics(),
      children: const <Widget>[
        Padding(
          padding: EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              GfSkeleton(width: 220, height: 24, radius: 6),
              SizedBox(height: 10),
              GfSkeleton(width: 120, height: 14, radius: 5),
              SizedBox(height: 14),
              GfSkeleton(height: 30, radius: 999),
              SizedBox(height: 8),
              GfSkeleton(width: 200, height: 30, radius: 999),
            ],
          ),
        ),
        GfDivider(),
        Padding(
          padding: EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              GfSkeleton(width: 96, height: 18, radius: 6),
              SizedBox(height: 16),
              Row(
                children: <Widget>[
                  GfSkeleton(width: 96, height: 52, radius: 8),
                  SizedBox(width: 24),
                  Expanded(
                    child: Column(
                      children: <Widget>[
                        GfSkeleton(height: 8, radius: 999),
                        SizedBox(height: 8),
                        GfSkeleton(height: 8, radius: 999),
                        SizedBox(height: 8),
                        GfSkeleton(height: 8, radius: 999),
                      ],
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        GfDivider(),
        Padding(
          padding: EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              GfSkeleton(width: 96, height: 18, radius: 6),
              SizedBox(height: 12),
              GfSkeleton(height: 64, radius: 8),
              SizedBox(height: 10),
              GfSkeleton(height: 64, radius: 8),
            ],
          ),
        ),
      ],
    );
  }
}

// 扩展：firstOrNull 便利。
extension _FirstOrNull<T> on Iterable<T> {
  T? get firstOrNull {
    final Iterator<T> it = iterator;
    return it.moveNext() ? it.current : null;
  }
}
