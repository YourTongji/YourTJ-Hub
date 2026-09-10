import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import 'course_common.dart';
import 'review_form_sheet.dart';
import 'review_delete_dialog.dart';

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

  /// 聚焦教学班（null = 课程全部课评）。初始值来自路由参数，详情加载后校验。
  int? _activeOfferingId;

  CourseRepository get _repository => ref.read(courseRepositoryProvider);

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
        _reviews = List<ReviewPayload>.of(result.list);
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
      if (target != null && target.mounted) {
        await Scrollable.ensureVisible(
          target,
          alignment: .1,
          duration: const Duration(milliseconds: 200),
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
      await _scrollController.animateTo(
        next,
        duration: const Duration(milliseconds: 120),
        curve: Curves.easeOut,
      );
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
        _reviews = <ReviewPayload>[..._reviews, ...result.list];
        _nextCursor = result.nextCursor ?? '';
        _reviewTotal = result.total;
      });
    } catch (_) {
      // 加载更多失败静默，滚动可再次触发。
    } finally {
      if (mounted) setState(() => _loadingMore = false);
    }
  }

  CourseCopy _copy() => CourseCopy(AppLocalizations.of(context));

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
          duration: const Duration(milliseconds: 260),
          curve: Curves.easeOut,
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

  Future<void> _toggleHelpful(ReviewPayload review) async {
    final bool target = !review.viewer.isHelpful;
    final int index = _reviews.indexWhere(
      (ReviewPayload r) => r.id == review.id,
    );
    if (index < 0) return;
    setState(() {
      _reviews[index] = _reviews[index].copyWith(
        viewer: review.viewer.copyWith(isHelpful: target),
        helpfulCount: review.helpfulCount + (target ? 1 : -1),
      );
    });
    try {
      await _repository.markHelpful(review.id, on: target);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        final int i = _reviews.indexWhere(
          (ReviewPayload r) => r.id == review.id,
        );
        if (i >= 0) _reviews[i] = review;
      });
      if (!_isUnauthorized(e)) {
        _toast(_copy().operationFailed, error: true);
      }
    }
  }

  // ---- UI ----

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String title = _detail.value?.name ?? l10n.entryCourses;

    return Scaffold(
      appBar: GfAppBar(
        leading: GfIconButton(
          icon: Icons.arrow_back,
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

    return Stack(
      children: <Widget>[
        Positioned.fill(
          child: AppRefreshIndicator(
            onRefresh: () async {
              await _load();
              await _loadRelated();
            },
            child: ListView(
              controller: _scrollController,
              physics: const AlwaysScrollableScrollPhysics(),
              padding: EdgeInsets.only(
                bottom: 96 + MediaQuery.paddingOf(context).bottom,
              ),
              children: <Widget>[
                _CourseHeader(detail: detail),
                const GfDivider(),
                _RatingSection(detail: detail),
                const GfDivider(),
                _AiSummaryCard(courseId: widget.courseId),
                const GfDivider(),
                _buildReviewsSection(l10n, copy, detail),
                const GfDivider(),
                _OfferingsSection(
                  offerings:
                      detail.offerings ?? const <CourseOfferingPayload>[],
                  activeOfferingId: _activeOfferingId,
                  onFocusOffering: _focusOffering,
                  copy: copy,
                ),
                const GfDivider(),
                _RelatedSection(
                  related: _related,
                  courseId: widget.courseId,
                  onOpenCourse: (int id) =>
                      context.pushReplacement('/courses/$id'),
                  copy: copy,
                ),
              ],
            ),
          ),
        ),
        Positioned(
          left: 0,
          right: 0,
          bottom: 0,
          child: ColoredBox(
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
        ),
      ],
    );
  }

  Widget _buildReviewsSection(
    AppLocalizations l10n,
    CourseCopy copy,
    CourseDetailPayload detail,
  ) {
    return Column(
      key: _reviewsSectionKey,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Row(
            children: <Widget>[
              Text(
                l10n.courseDetailReviews,
                style: GfTheme.typographyOf(
                  context,
                ).heading.copyWith(fontWeight: FontWeight.w700),
              ),
              if (_reviewTotal > 0) ...<Widget>[
                const SizedBox(width: 6),
                Text(
                  '$_reviewTotal',
                  style: GfTheme.typographyOf(context).small.copyWith(
                    color: GfTheme.colorsOf(
                      context,
                    ).baseContent.withValues(alpha: 0.45),
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
          GfEmpty(icon: Icons.rate_review_outlined, message: l10n.reviewsEmpty)
        else if (_reviewsLoaded)
          for (final ReviewPayload review in [
            ..._reviews.where((review) => review.viewer.canEdit),
            ..._reviews.where((review) => !review.viewer.canEdit),
          ]) ...<Widget>[
            _ReviewRow(
              key: review.id == widget.focusReviewId
                  ? _targetReviewKey
                  : ValueKey(review.id),
              review: review,
              offeringLabel: _offeringLabel(detail, review.offeringId),
              onHelpful: () => _toggleHelpful(review),
              onEdit: review.viewer.canEdit
                  ? () => _openEditSheet(review)
                  : null,
              onDelete: review.viewer.canDelete
                  ? () => _confirmDeleteReview(review)
                  : null,
            ),
            const GfDivider(),
          ],
        if (_reviewsLoaded && _nextCursor.isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: Center(
              child: _loadingMore
                  ? const GfLoadingIndicator(small: true)
                  : Text(
                      copy.relatedEmpty,
                      style: GfTheme.typographyOf(context).caption.copyWith(
                        color: GfTheme.colorsOf(
                          context,
                        ).baseContent.withValues(alpha: 0.35),
                      ),
                    ),
            ),
          ),
      ],
    );
  }

  String _offeringLabel(CourseDetailPayload detail, int offeringId) {
    final CourseOfferingPayload? offering = detail.offerings
        ?.where((CourseOfferingPayload o) => o.id == offeringId)
        .firstOrNull;
    if (offering == null) return '#$offeringId';
    final String classLabel = (offering.className?.isNotEmpty ?? false)
        ? offering.className!
        : (offering.classCode ?? '');
    return <String>[
      shortTerm(
        offering.termCode,
        locale: AppLocalizations.of(context).localeName,
      ),
      classLabel,
      offering.campus ?? '',
      offering.instructors?.join('、') ?? '',
    ].where((String s) => s.isNotEmpty).join(' · ');
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
                icon: Icons.account_balance_outlined,
                label: detail.department,
              ),
              _metaChip(context, icon: Icons.people_outline, label: teacher),
              if (credit.isNotEmpty)
                _metaChip(
                  context,
                  icon: Icons.school_outlined,
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
    final List<String> team = detail.teamInstructors ?? const <String>[];
    if (detail.reviewScope == 'team' && team.isNotEmpty) {
      final String joined = team.join('、');
      return team.length > 1
          ? '${copy.teamInstructorsPrefix}$joined${copy.teamInstructorsSuffix(team.length)}'
          : '${copy.teamInstructorsPrefix}$joined';
    }
    if (detail.teacherName != null && detail.teacherName!.isNotEmpty) {
      return detail.teacherName!;
    }
    return copy.noTeacher;
  }

  Widget _metaChip(
    BuildContext context, {
    required IconData icon,
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
          Icon(icon, size: 14, color: colors.primary.withValues(alpha: 0.7)),
          const SizedBox(width: 5),
          Text(
            label,
            style: GfTheme.typographyOf(context).caption.copyWith(
              color: colors.baseContent.withValues(alpha: 0.7),
              fontWeight: emphasized != null ? FontWeight.w600 : null,
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
          Icon(Icons.filter_alt_outlined, size: 16, color: colors.primary),
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
    final String classLabel = (offering.className?.isNotEmpty ?? false)
        ? offering.className!
        : (offering.classCode ?? '');
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
          Row(
            children: <Widget>[
              Text(
                copy.ratingTitle,
                style: type.heading.copyWith(fontWeight: FontWeight.w700),
              ),
              const SizedBox(width: 8),
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
            Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: <Widget>[
                // One text baseline for the score and denominator.
                Text.rich(
                  TextSpan(
                    children: [
                      TextSpan(
                        text: formatRating(avg),
                        style: type.title1.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      TextSpan(
                        text: ' ${copy.ratingOutOf}',
                        style: type.small.copyWith(
                          color: colors.baseContent.withValues(alpha: .5),
                        ),
                      ),
                    ],
                  ),
                  key: const ValueKey('course-rating-score'),
                ),
                const SizedBox(width: 20),
                // 分布条 5★ → 1★。
                Expanded(
                  child: Column(
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
                  ),
                ),
              ],
            ),
        ],
      ),
    );
  }
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

  static const List<double> _opacity = <double>[0.95, 0.72, 0.5, 0.34, 0.24];

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final double ratio = max <= 0 ? 0 : count / max;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: <Widget>[
          SizedBox(
            width: 16 + MediaQuery.textScalerOf(context).scale(12),
            child: Row(
              children: <Widget>[
                Icon(
                  Icons.star,
                  size: 12,
                  color: colors.baseContent.withValues(alpha: 0.3),
                ),
                const SizedBox(width: 2),
                Text(
                  '$star',
                  style: GfTheme.typographyOf(context).meta.copyWith(
                    color: colors.baseContent.withValues(alpha: 0.55),
                  ),
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
                    color: colors.warning.withValues(alpha: _opacity[star - 1]),
                    borderRadius: BorderRadius.circular(999),
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(width: 8),
          SizedBox(
            width: 24,
            child: Text(
              '$count',
              textAlign: TextAlign.right,
              style: GfTheme.typographyOf(context).meta.copyWith(
                color: colors.baseContent.withValues(alpha: 0.45),
              ),
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
/// disabled → 不渲染；cached/generated → 展开展示；none → 折叠，首次展开触发
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
              _expanded = true;
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
    final bool ready = _status == 'ready';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        InkWell(
          onTap: _onToggle,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            child: Row(
              children: <Widget>[
                Icon(Icons.auto_awesome, size: 16, color: colors.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    l10n.courseDetailAiSummary,
                    style: type.small.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                ),
                if (ready)
                  Padding(
                    padding: const EdgeInsets.only(right: 4),
                    child: GfBadge(
                      label: copy.summaryGenerated,
                      variant: GfBadgeVariant.muted,
                    ),
                  ),
                Icon(
                  _expanded ? Icons.expand_less : Icons.expand_more,
                  size: 18,
                  color: colors.baseContent.withValues(alpha: 0.45),
                ),
              ],
            ),
          ),
        ),
        if (ready)
          Align(
            alignment: Alignment.centerRight,
            child: Padding(
              padding: const EdgeInsets.only(right: 8),
              child: IconButton(
                icon: _refreshing
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.refresh, size: 18),
                tooltip: copy.summaryRefresh,
                onPressed: _refreshing ? null : () => _load(refresh: true),
                visualDensity: VisualDensity.compact,
                color: colors.iconMuted,
              ),
            ),
          ),
        if (_expanded)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
            child: _buildContent(l10n, copy, colors, type),
          ),
      ],
    );
  }

  Widget _buildContent(
    AppLocalizations l10n,
    CourseCopy copy,
    GfColors colors,
    GfTypography type,
  ) {
    if (_status == 'loading') {
      return const Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfSkeleton(width: 220, height: 14, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(height: 14, radius: 5),
          SizedBox(height: 8),
          GfSkeleton(width: 140, height: 14, radius: 5),
        ],
      );
    }
    if (_status == 'insufficient') {
      return Row(
        children: <Widget>[
          Icon(
            Icons.auto_awesome,
            size: 14,
            color: colors.baseContent.withValues(alpha: 0.35),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              copy.summaryInsufficient,
              style: type.small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          ),
        ],
      );
    }
    if (_status == 'error') {
      return Row(
        children: <Widget>[
          Expanded(
            child: Text(
              copy.summaryLoadFailed,
              style: type.small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          ),
          TextButton(
            onPressed: () => _load(),
            child: Text(copy.summaryRefresh),
          ),
        ],
      );
    }
    final CourseAiSummaryPayload? payload = _summary;
    if (payload == null) return const SizedBox.shrink();
    final String level = _consensusOrder.contains(payload.consensus)
        ? payload.consensus
        : 'neutral';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Row(
          children: <Widget>[_ConsensusBadge(level: level, copy: copy)],
        ),
        const SizedBox(height: 8),
        Text(
          copy.summaryConsensusText(level),
          style: type.small.copyWith(
            color: colors.baseContent.withValues(alpha: 0.85),
          ),
        ),
        if (payload.keywords.isNotEmpty) ...<Widget>[
          const SizedBox(height: 10),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: <Widget>[
              for (final String keyword in payload.keywords.take(5))
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 3,
                  ),
                  decoration: BoxDecoration(
                    color: colors.base200.withValues(alpha: 0.6),
                    borderRadius: BorderRadius.circular(999),
                    border: Border.all(color: colors.line),
                  ),
                  child: Text(
                    keyword,
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                ),
            ],
          ),
        ],
        if (payload.pros.isNotEmpty || payload.cons.isNotEmpty) ...<Widget>[
          const SizedBox(height: 10),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              if (payload.pros.isNotEmpty)
                Expanded(
                  child: _ProConList(
                    title: copy.summaryPros,
                    items: payload.pros.take(4).toList(),
                    color: colors.success,
                    sign: '+',
                  ),
                ),
              if (payload.pros.isNotEmpty && payload.cons.isNotEmpty)
                const SizedBox(width: 16),
              if (payload.cons.isNotEmpty)
                Expanded(
                  child: _ProConList(
                    title: copy.summaryCons,
                    items: payload.cons.take(4).toList(),
                    color: colors.error,
                    sign: '−',
                  ),
                ),
            ],
          ),
        ],
        if (payload.representativeReviews.isNotEmpty) ...<Widget>[
          const SizedBox(height: 12),
          Text(
            copy.summaryRepresentativeReviews,
            style: type.caption.copyWith(
              color: colors.baseContent.withValues(alpha: 0.7),
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 6),
          for (final CourseAiSummaryRepresentativeReview item
              in payload.representativeReviews.take(3)) ...<Widget>[
            Container(
              margin: const EdgeInsets.only(bottom: 6),
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: colors.base200.withValues(alpha: 0.4),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: colors.line.withValues(alpha: 0.6)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  GfBadge(
                    label: copy.summarySentiment(item.sentiment),
                    variant: item.sentiment == 'positive'
                        ? GfBadgeVariant.success
                        : item.sentiment == 'negative'
                        ? GfBadgeVariant.error
                        : GfBadgeVariant.muted,
                  ),
                  const SizedBox(height: 6),
                  Text(
                    item.excerpt,
                    style: type.small.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.8),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
        if (_refreshNotice != null) ...<Widget>[
          const SizedBox(height: 8),
          Text(
            _refreshNotice!,
            style: type.caption.copyWith(color: colors.error),
          ),
        ],
        const SizedBox(height: 8),
        Text(
          copy.summaryDisclaimer,
          style: type.meta.copyWith(
            color: colors.baseContent.withValues(alpha: 0.4),
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

class _ProConList extends StatelessWidget {
  const _ProConList({
    required this.title,
    required this.items,
    required this.color,
    required this.sign,
  });

  final String title;
  final List<String> items;
  final Color color;
  final String sign;

  @override
  Widget build(BuildContext context) {
    final GfTypography type = GfTheme.typographyOf(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          title,
          style: type.caption.copyWith(
            color: color,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 4),
        for (final String item in items)
          Padding(
            padding: const EdgeInsets.only(bottom: 2),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(sign, style: type.small.copyWith(color: color)),
                const SizedBox(width: 4),
                Expanded(
                  child: Text(
                    item,
                    style: type.small.copyWith(
                      color: GfTheme.colorsOf(
                        context,
                      ).baseContent.withValues(alpha: 0.8),
                    ),
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

// ---- 课评行 ----

class _ReviewRow extends StatelessWidget {
  const _ReviewRow({
    super.key,
    required this.review,
    required this.offeringLabel,
    required this.onHelpful,
    this.onEdit,
    this.onDelete,
  });

  final ReviewPayload review;
  final String offeringLabel;
  final VoidCallback onHelpful;
  final VoidCallback? onEdit;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final int? rating = review.rating;
    final bool helpful = review.viewer.isHelpful;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      _authorLabel(review, copy),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.small.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '$offeringLabel · ${formatDateTime(review.createdAt)}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.meta.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.45),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              if (rating != null && rating > 0)
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: <Widget>[
                    for (int star = 1; star <= 5; star++)
                      Icon(
                        star <= rating ? Icons.star : Icons.star_border,
                        size: 15,
                        color: star <= rating
                            ? colors.warning
                            : colors.baseContent.withValues(alpha: 0.2),
                      ),
                  ],
                ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            review.content,
            style: type.small.copyWith(
              color: colors.baseContent.withValues(alpha: 0.85),
              height: 1.5,
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: <Widget>[
              _actionChip(
                context,
                label: '${review.helpfulCount}',
                hint: l10n.reviewHelpful,
                icon: Icons.thumb_up_outlined,
                active: helpful,
                activeColor: colors.warning,
                onTap: onHelpful,
              ),
              if (onEdit != null) ...<Widget>[
                const SizedBox(width: 8),
                _actionChip(
                  context,
                  label: l10n.commonEdit,
                  hint: l10n.commonEdit,
                  icon: Icons.edit_outlined,
                  active: false,
                  onTap: onEdit,
                ),
              ],
              if (onDelete != null) ...<Widget>[
                const SizedBox(width: 8),
                _actionChip(
                  context,
                  label: copy.delete,
                  hint: copy.delete,
                  icon: Icons.delete_outline,
                  active: false,
                  onTap: onDelete,
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }

  String _authorLabel(ReviewPayload review, CourseCopy copy) {
    if (review.author.kind == 'member') return review.author.label;
    if (review.author.kind == 'legacy') return copy.authorLegacyLabel;
    return copy.authorAnonymousLabel;
  }

  Widget _actionChip(
    BuildContext context, {
    required String label,
    required String hint,
    required IconData icon,
    required bool active,
    VoidCallback? onTap,
    Color? activeColor,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color? tint = active ? (activeColor ?? colors.primary) : null;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
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
            Icon(
              icon,
              size: 13,
              color: tint ?? colors.baseContent.withValues(alpha: 0.45),
            ),
            const SizedBox(width: 4),
            Text(
              label,
              style: GfTheme.typographyOf(context).meta.copyWith(
                color: tint ?? colors.baseContent.withValues(alpha: 0.7),
              ),
            ),
            const SizedBox(width: 2),
            Text(
              hint,
              style: GfTheme.typographyOf(context).meta.copyWith(
                color: tint ?? colors.baseContent.withValues(alpha: 0.45),
              ),
            ),
          ],
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
                Icon(
                  collapsed ? Icons.chevron_right : Icons.expand_more,
                  size: 18,
                  color: colors.baseContent.withValues(alpha: 0.45),
                ),
                const SizedBox(width: 4),
                Text(
                  termName,
                  style: type.small.copyWith(
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent.withValues(alpha: 0.75),
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

    final String classLabel = (offering.className?.isNotEmpty ?? false)
        ? offering.className!
        : (offering.classCode ?? '');
    final List<String> meta = <String>[
      offering.campus ?? '',
      offering.instructors?.join('、') ?? '',
    ].where((String s) => s.isNotEmpty).toList();
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
                  Icon(Icons.star, size: 13, color: colors.warning),
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
                    Icon(Icons.star, size: 13, color: colors.warning),
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
                  Icon(
                    Icons.chevron_right,
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
            child: Icon(
              Icons.arrow_forward,
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
