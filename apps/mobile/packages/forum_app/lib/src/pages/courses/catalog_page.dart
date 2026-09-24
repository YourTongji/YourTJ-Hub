import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';
import 'course_common.dart';

/// 课程目录页（web CourseCatalogPage.vue 的移动端形态）。
///
/// 搜索（300ms 防抖）+ 院系/学期/校区/教师多值筛选 chip 行（多值并集，
/// 由仓库按逗号拼接查询）+ 只看有评价开关 + 滚动分页（hasNext）+ 下拉刷新。
/// 筛选值域（院系/学期/校区）来自页面级数据通道 `/courses` 的 SSR props
/// （与 web 同源）；教师为自由输入多值（web 亦为文本输入）。
class CourseCatalogPage extends ConsumerWidget {
  const CourseCatalogPage({super.key, this.initialQuery = ''});
  final String initialQuery;

  @override
  Widget build(BuildContext context, WidgetRef ref) => _CourseCatalogContent(
    key: ValueKey(ref.watch(offlineCacheEpochProvider)),
    initialQuery: initialQuery,
  );
}

// Recreate the full query and permission state on an account/site change.
class _CourseCatalogContent extends ConsumerStatefulWidget {
  const _CourseCatalogContent({super.key, required this.initialQuery});
  final String initialQuery;

  @override
  ConsumerState<_CourseCatalogContent> createState() =>
      _CourseCatalogPageState();
}

class _CourseCatalogPageState extends ConsumerState<_CourseCatalogContent> {
  static const Duration _searchDebounce = Duration(milliseconds: 300);
  static const int _pageSize = 20;

  final TextEditingController _searchController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  Timer? _debounce;

  AsyncValue<CourseListResultPayload> _page = const AsyncValue.loading();
  bool _canManageCourses = false;
  List<CourseSummaryPayload> _courses = const <CourseSummaryPayload>[];
  bool _hasNext = false;
  bool _loadingMore = false;
  int _nextPage = 2;
  int _generation = 0;
  bool _refreshing = false;
  Object? _refreshError, _loadMoreError;
  _CatalogQuery? _appliedQuery;
  late final int _sessionEpoch;
  bool _optionsLoading = true;
  Object? _optionsError;
  int _optionsGeneration = 0;

  // 筛选值域（SSR 页面通道，best-effort）。
  List<String> _departmentOptions = const <String>[];
  Map<String, String> _termOptions = const <String, String>{};
  List<String> _campusOptions = const <String>[];

  final Set<String> _selectedDepartments = <String>{};
  final Set<String> _selectedTerms = <String>{};
  final Set<String> _selectedCampuses = <String>{};
  final List<String> _selectedInstructors = <String>[];
  bool _onlyWithReviews = false;

  // 行内学期展开（最近学期 +N 折叠）。
  final Set<int> _expandedTermRows = <int>{};

  CourseRepository get _repository => ref.read(courseRepositoryProvider);

  @override
  void initState() {
    super.initState();
    _sessionEpoch = ref.read(offlineCacheEpochProvider);
    _searchController.text = widget.initialQuery;
    _loadOptions();
    _load();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  bool get _activeSession =>
      mounted && _sessionEpoch == ref.read(offlineCacheEpochProvider);

  bool _current(int generation, int epoch) =>
      _activeSession &&
      generation == _generation &&
      epoch == ref.read(offlineCacheEpochProvider);

  _CatalogQuery _query() => _CatalogQuery(
    keyword: _searchController.text.trim(),
    departments: _selectedDepartments.toList()..sort(),
    terms: _selectedTerms.toList()..sort(),
    campuses: _selectedCampuses.toList()..sort(),
    instructors: List.of(_selectedInstructors),
    onlyWithReviews: _onlyWithReviews,
  );

  void _scheduleSearch(String _) {
    _debounce?.cancel();
    if (!_activeSession) return;
    // Invalidate immediately, including the debounce interval: an older request
    // must not replace results while the user is already entering a new query.
    setState(() {
      _generation++;
      _page = const AsyncValue.loading();
      _loadingMore = false;
      _hasNext = false;
      _refreshing = false;
      _refreshError = null;
    });
    _debounce = Timer(_searchDebounce, () {
      if (!mounted) return;
      _load();
    });
  }

  void _clearDebounceAndSearch() {
    _debounce?.cancel();
    _load();
  }

  void _clearSearch() {
    _debounce?.cancel();
    _searchController.clear();
    _load();
  }

  /// Options use the existing SSR projection; failure has its own retry and
  /// never masquerades as an empty set of available filters.
  Future<void> _loadOptions() async {
    if (!_activeSession) return;
    final generation = ++_optionsGeneration;
    setState(() {
      _optionsLoading = true;
      _optionsError = null;
    });
    final int epoch = ref.read(offlineCacheEpochProvider);
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/courses');
      if (!mounted ||
          generation != _optionsGeneration ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      final Map<String, dynamic> props = payload.props;
      final List<String> departments =
          (props['departments'] as List<dynamic>? ?? const [])
              .whereType<String>()
              .toList();
      final Map<String, String> terms = <String, String>{};
      final dynamic rawTerms = props['terms'];
      if (rawTerms is List) {
        for (final dynamic item in rawTerms) {
          if (item is Map) {
            final dynamic value = item['value'];
            final dynamic label = item['label'];
            if (value is String && label is String) {
              terms[value] = label;
            }
          }
        }
      }
      final List<String> campuses =
          (props['campuses'] as List<dynamic>? ?? const [])
              .whereType<String>()
              .toList();
      if (!mounted) return;
      setState(() {
        _canManageCourses = payload.layout.viewer.canManageCourses;
        _departmentOptions = departments;
        _termOptions = terms;
        _campusOptions = campuses;
      });
    } catch (error) {
      if (mounted &&
          generation == _optionsGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _optionsError = error);
      }
    } finally {
      if (mounted &&
          generation == _optionsGeneration &&
          epoch == ref.read(offlineCacheEpochProvider)) {
        setState(() => _optionsLoading = false);
      }
    }
  }

  Future<void> _load({bool refresh = false}) async {
    _debounce?.cancel();
    if (!_activeSession) return;
    final generation = ++_generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    final query = _query();
    final previous = refresh ? _page.valueOrNull : null;
    setState(() {
      _refreshing = previous != null;
      _refreshError = null;
      _loadMoreError = null;
      _loadingMore = false;
      if (previous == null) {
        _page = const AsyncValue.loading();
        _courses = [];
        _hasNext = false;
      }
    });
    try {
      final result = await query.fetch(_repository, 1, _pageSize);
      if (!_current(generation, epoch)) return;
      if (result.page != 1) {
        throw const ApiException(fallbackMessage: 'Unexpected course page');
      }
      final seen = <int>{};
      setState(() {
        _courses = result.list.where((course) => seen.add(course.id)).toList();
        _hasNext = result.hasNext;
        _nextPage = 2;
        _appliedQuery = query;
        _expandedTermRows.clear();
        _page = AsyncValue.data(result);
      });
      if (!refresh && _scrollController.hasClients) _scrollController.jumpTo(0);
    } catch (error, stack) {
      if (!_current(generation, epoch)) return;
      setState(() {
        if (previous != null) {
          _refreshError = error;
        } else {
          _page = AsyncValue.error(error, stack);
        }
      });
    } finally {
      if (_current(generation, epoch)) setState(() => _refreshing = false);
    }
  }

  Future<void> _loadMore() async {
    final query = _appliedQuery;
    if (!_activeSession ||
        _loadingMore ||
        _refreshing ||
        !_hasNext ||
        !_page.hasValue ||
        query == null) {
      return;
    }
    final generation = _generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    final page = _nextPage;
    setState(() {
      _loadingMore = true;
      _loadMoreError = null;
    });
    try {
      final result = await query.fetch(_repository, page, _pageSize);
      if (!_current(generation, epoch)) return;
      if (result.page != page) {
        throw const ApiException(fallbackMessage: 'Unexpected course page');
      }
      final seen = _courses.map((course) => course.id).toSet();
      final added = result.list.where((course) => seen.add(course.id)).toList();
      setState(() {
        _courses = [..._courses, ...added];
        _hasNext = result.hasNext;
        if (added.isEmpty && result.hasNext) {
          // Keep this page available for explicit retry, without an automatic
          // request loop when a stale projection stops advancing.
          _loadMoreError = AppLocalizations.of(
            context,
          ).coursesPaginationStalled;
        } else {
          _nextPage = page + 1;
        }
      });
    } catch (error) {
      if (_current(generation, epoch)) setState(() => _loadMoreError = error);
    } finally {
      if (_current(generation, epoch)) setState(() => _loadingMore = false);
    }
  }

  void _resetFilters() {
    _searchController.clear();
    setState(() {
      _selectedDepartments.clear();
      _selectedTerms.clear();
      _selectedCampuses.clear();
      _selectedInstructors.clear();
      _onlyWithReviews = false;
    });
    _load();
  }

  bool get _hasActiveFilters =>
      _selectedDepartments.isNotEmpty ||
      _selectedTerms.isNotEmpty ||
      _selectedCampuses.isNotEmpty ||
      _selectedInstructors.isNotEmpty ||
      _onlyWithReviews ||
      _searchController.text.trim().isNotEmpty;

  // ---- 筛选操作 ----

  void _applyDepartments(Set<String> values) {
    setState(() {
      _selectedDepartments
        ..clear()
        ..addAll(values);
    });
    _load();
  }

  void _applyTerms(Set<String> values) {
    setState(() {
      _selectedTerms
        ..clear()
        ..addAll(values);
    });
    _load();
  }

  void _applyCampuses(Set<String> values) {
    setState(() {
      _selectedCampuses
        ..clear()
        ..addAll(values);
    });
    _load();
  }

  void _applyInstructors(List<String> values) {
    setState(() {
      _selectedInstructors
        ..clear()
        ..addAll(values);
    });
    _load();
  }

  Future<void> _pickOptionValues({
    required String title,
    required List<String> options,
    required Map<String, String> labelByValue,
    required Set<String> initial,
    required ValueChanged<Set<String>> onApplied,
  }) async {
    if (_optionsLoading) return;
    if (_optionsError != null) {
      await _loadOptions();
      return;
    }
    final epoch = ref.read(offlineCacheEpochProvider);
    final result = await showGfBottomSheet<Set<String>>(
      context,
      height: 520,
      keyboardAware: true,
      builder: (_) => _CourseOptionsSheet(
        title: title,
        options: options,
        labels: labelByValue,
        initial: initial,
      ),
    );
    if (result != null &&
        mounted &&
        epoch == ref.read(offlineCacheEpochProvider)) {
      onApplied(result);
    }
  }

  /// 教师筛选：web 为自由文本输入（逗号分隔），移动端改为底部 sheet 内
  /// 逐条添加 token；已选值以 chip 呈现并可移除。
  Future<void> _pickInstructors() async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final List<String>? result = await showGfBottomSheet<List<String>>(
      context,
      height: 420,
      keyboardAware: true,
      builder: (BuildContext ctx) => _InstructorPickerSheet(
        title: l10n.coursesFilterInstructor,
        copy: CourseCopy(l10n),
        initial: _selectedInstructors,
      ),
    );
    if (result != null &&
        mounted &&
        epoch == ref.read(offlineCacheEpochProvider)) {
      _applyInstructors(result);
    }
  }

  // ---- UI ----

  @override
  Widget build(BuildContext context) {
    if (ref.watch(offlineCacheEpochProvider) != _sessionEpoch) {
      return const SizedBox.shrink();
    }
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(
        title: Text(
          l10n.coursesTitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          if (_canManageCourses)
            PopupMenuButton<String>(
              tooltip: l10n.coursesManagement,
              useRootNavigator: true,
              onSelected: (path) => context.push(path),
              itemBuilder: (_) => [
                PopupMenuItem(
                  value: '/moderation/courses',
                  child: Text(l10n.coursesManagement),
                ),
                PopupMenuItem(
                  value: '/moderation/course-reviews',
                  child: Text(l10n.coursesReviewModeration),
                ),
              ],
            ),
        ],
      ),
      body: Column(
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
            child: GfSearchField(
              controller: _searchController,
              hintText: l10n.coursesSearchHint,
              clearLabel: l10n.courseCopyClearSearch,
              onChanged: _scheduleSearch,
              onSubmitted: (_) => _clearDebounceAndSearch(),
              onClear: _clearSearch,
            ),
          ),
          _buildFilterRow(l10n),
          if (_optionsLoading) const LinearProgressIndicator(minHeight: 2),
          if (_optionsError != null)
            _retryNotice(l10n.coursesFilterLoadFailed, _loadOptions),
          if (_refreshing) const LinearProgressIndicator(minHeight: 2),
          if (_refreshError != null)
            _retryNotice(
              resolveErrorMessage(l10n, _refreshError!),
              () => _load(refresh: true),
            ),
          if (_hasActiveFilters)
            Align(
              alignment: Alignment.centerRight,
              child: TextButton(
                onPressed: _resetFilters,
                child: Text(l10n.coursesResetSearch),
              ),
            ),
          Expanded(
            child: _page.when(
              loading: () => const _CourseListSkeleton(),
              error: (Object e, StackTrace _) => GfErrorRetry(
                message: resolveErrorMessage(l10n, e),
                onRetry: _load,
              ),
              data: (_) => _buildList(l10n),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterRow(AppLocalizations l10n) {
    final Widget dept = _filterChip(
      label: l10n.coursesFilterDepartment,
      count: _selectedDepartments.length,
      selected: _selectedDepartments.isNotEmpty,
      onTap: _optionsLoading || _optionsError != null
          ? null
          : () => _pickOptionValues(
              title: l10n.coursesFilterDepartment,
              options: _departmentOptions,
              labelByValue: const <String, String>{},
              initial: _selectedDepartments,
              onApplied: _applyDepartments,
            ),
    );
    final Widget term = _filterChip(
      label: l10n.coursesFilterTerm,
      count: _selectedTerms.length,
      selected: _selectedTerms.isNotEmpty,
      onTap: _optionsLoading || _optionsError != null
          ? null
          : () => _pickOptionValues(
              title: l10n.coursesFilterTerm,
              options: _termOptions.keys.toList()..sort(),
              labelByValue: _termOptions,
              initial: _selectedTerms,
              onApplied: _applyTerms,
            ),
    );
    final Widget campus = _filterChip(
      label: l10n.coursesFilterCampus,
      count: _selectedCampuses.length,
      selected: _selectedCampuses.isNotEmpty,
      onTap: _optionsLoading || _optionsError != null
          ? null
          : () => _pickOptionValues(
              title: l10n.coursesFilterCampus,
              options: _campusOptions,
              labelByValue: const <String, String>{},
              initial: _selectedCampuses,
              onApplied: _applyCampuses,
            ),
    );
    final Widget instructor = _filterChip(
      label: l10n.coursesFilterInstructor,
      count: _selectedInstructors.length,
      selected: _selectedInstructors.isNotEmpty,
      onTap: _pickInstructors,
    );
    final Widget onlyReviews = _filterChip(
      label: l10n.coursesOnlyWithReviews,
      selected: _onlyWithReviews,
      icon: _onlyWithReviews ? Icons.check : null,
      onTap: () {
        setState(() => _onlyWithReviews = !_onlyWithReviews);
        _load();
      },
    );

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        children: <Widget>[
          dept,
          const SizedBox(width: 8),
          term,
          const SizedBox(width: 8),
          campus,
          const SizedBox(width: 8),
          instructor,
          const SizedBox(width: 8),
          onlyReviews,
          const SizedBox(width: 4),
        ],
      ),
    );
  }

  Widget _filterChip({
    required String label,
    required bool selected,
    required VoidCallback? onTap,
    int? count,
    IconData? icon,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        constraints: const BoxConstraints(minHeight: 44),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected
              ? colors.primary.withValues(alpha: 0.1)
              : colors.base100,
          borderRadius: BorderRadius.circular(999),
          border: Border.all(
            color: selected
                ? colors.primary.withValues(alpha: 0.5)
                : colors.line,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            if (icon != null) ...<Widget>[
              Icon(icon, size: 14, color: colors.primary),
              const SizedBox(width: 4),
            ],
            Text(
              label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: selected
                    ? colors.primary
                    : colors.baseContent.withValues(alpha: 0.75),
              ),
            ),
            if ((count ?? 0) > 0) ...<Widget>[
              const SizedBox(width: 6),
              Container(
                constraints: const BoxConstraints(minWidth: 18),
                padding: const EdgeInsets.symmetric(horizontal: 4),
                height: 18,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: colors.primary,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  '$count',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: colors.primaryContent,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _retryNotice(String message, VoidCallback retry) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16),
    child: Row(
      children: [
        Expanded(child: Text(message)),
        TextButton(
          onPressed: retry,
          child: Text(AppLocalizations.of(context).commonRetry),
        ),
      ],
    ),
  );

  Widget _buildList(AppLocalizations l10n) {
    final copy = CourseCopy(l10n);
    return AppRefreshIndicator(
      onRefresh: () => _load(refresh: true),
      child: CustomScrollView(
        key: const Key('course-catalog-list'),
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
        slivers: [
          if (_courses.isEmpty)
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 40),
                child: GfEmpty(
                  icon: Icons.menu_book_outlined,
                  message: _hasActiveFilters
                      ? copy.noFilterResults
                      : copy.catalogEmptyTitle,
                  description: _hasActiveFilters
                      ? copy.noFilterResultsDescription
                      : copy.catalogEmptyDescription,
                ),
              ),
            ),
          SliverList.separated(
            itemCount: _courses.length,
            separatorBuilder: (_, _) => const GfDivider(),
            itemBuilder: (context, index) {
              final course = _courses[index];
              return _CourseRow(
                course: course,
                termsExpanded: _expandedTermRows.contains(course.id),
                onToggleTerms: () => setState(() {
                  if (!_expandedTermRows.remove(course.id)) {
                    _expandedTermRows.add(course.id);
                  }
                }),
                onTap: () => context.push('/courses/${course.id}'),
              );
            },
          ),
          if (_courses.isNotEmpty || _hasNext)
            SliverToBoxAdapter(
              child: GfListFooter(
                loading: _loadingMore,
                hasMore: _hasNext,
                autoLoad: !_refreshing && _refreshError == null,
                progressKey: (_generation, _courses.length, _nextPage),
                error: _loadMoreError == null
                    ? null
                    : resolveErrorMessage(l10n, _loadMoreError!),
                onLoadMore: _loadMore,
              ),
            ),
        ],
      ),
    );
  }
}

class _CourseListSkeleton extends StatelessWidget {
  const _CourseListSkeleton();

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      physics: const NeverScrollableScrollPhysics(),
      itemCount: 7,
      separatorBuilder: (_, _) => const GfDivider(),
      itemBuilder: (_, _) => const Padding(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            GfSkeleton(width: 200, height: 16, radius: 5),
            SizedBox(height: 8),
            GfSkeleton(width: 260, height: 12, radius: 5),
            SizedBox(height: 10),
            Row(
              children: <Widget>[
                GfSkeleton(width: 64, height: 12, radius: 5),
                SizedBox(width: 10),
                GfSkeleton(width: 90, height: 12, radius: 5),
                Spacer(),
                GfSkeleton(width: 44, height: 12, radius: 5),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// 目录行：课名 + 课号 chip、教师 · 院系、评分/评价数/学分/最近学期 chips。
class _CourseRow extends StatelessWidget {
  const _CourseRow({
    required this.course,
    required this.termsExpanded,
    required this.onToggleTerms,
    required this.onTap,
  });

  final CourseSummaryPayload course;
  final bool termsExpanded;
  final VoidCallback onToggleTerms;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    final List<String> terms = sortedRecentTerms(course.recentTerms);
    final String teacher =
        (course.teacherName != null && course.teacherName!.isNotEmpty)
        ? course.teacherName!
        : (course.instructors?.isNotEmpty ?? false)
        ? course.instructors!.join('、')
        : copy.noTeacher;
    final String credit = formatCreditText(course.creditX10);
    final int reviewCount = course.reviewCount ?? 0;
    final double? ratingAvg = course.ratingAvg;

    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Expanded(
                  child: Text(
                    course.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: type.bodyStrong,
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 2,
                  ),
                  decoration: BoxDecoration(
                    color: colors.base200,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(
                    course.primaryCode,
                    style: type.meta.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              '$teacher · ${course.department}',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: type.small.copyWith(
                color: colors.baseContent.withValues(alpha: 0.6),
              ),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 12,
              runSpacing: 6,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                if (ratingAvg != null && ratingAvg > 0) ...<Widget>[
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: <Widget>[
                      Icon(Icons.star, size: 14, color: colors.warning),
                      const SizedBox(width: 2),
                      Text(
                        formatRating(ratingAvg),
                        style: type.small.copyWith(
                          fontWeight: FontWeight.w600,
                          color: colors.baseContent,
                        ),
                      ),
                      if (reviewCount > 0) ...<Widget>[
                        const SizedBox(width: 2),
                        Text(
                          '($reviewCount)',
                          style: type.caption.copyWith(
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ],
                    ],
                  ),
                ] else if (reviewCount > 0) ...<Widget>[
                  Text(
                    l10n.coursesNoRating,
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.5),
                    ),
                  ),
                  Text(
                    '($reviewCount)',
                    style: type.caption.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.5),
                    ),
                  ),
                ],
                if (credit.isNotEmpty) ...<Widget>[
                  Text(
                    '$credit ${copy.creditUnit}',
                    style: type.small.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                ],
                ..._termChips(context, terms),
              ],
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _termChips(BuildContext context, List<String> terms) {
    if (terms.isEmpty) return const <Widget>[];
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    Widget chip(String label, {VoidCallback? onTap}) {
      return InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(999),
        child: Container(
          height: 22,
          padding: const EdgeInsets.symmetric(horizontal: 8),
          decoration: BoxDecoration(
            color: colors.base200.withValues(alpha: 0.6),
            borderRadius: BorderRadius.circular(999),
          ),
          child: Center(
            widthFactor: 1,
            child: Text(
              label,
              style: type.meta.copyWith(
                color: colors.baseContent.withValues(alpha: 0.7),
              ),
            ),
          ),
        ),
      );
    }

    if (terms.length == 1) {
      return <Widget>[
        chip(
          shortTerm(
            terms.first,
            locale: AppLocalizations.of(context).localeName,
          ),
        ),
      ];
    }
    final List<Widget> visible = termsExpanded
        ? terms
              .map(
                (term) => shortTerm(
                  term,
                  locale: AppLocalizations.of(context).localeName,
                ),
              )
              .map(chip)
              .toList()
        : <Widget>[];
    return <Widget>[
      chip(
        shortTerm(terms.first, locale: AppLocalizations.of(context).localeName),
        onTap: onToggleTerms,
      ),
      if (!termsExpanded) chip('+${terms.length - 1}', onTap: onToggleTerms),
      ...visible,
    ];
  }
}

/// 教师筛选底部 sheet：自持输入框与草稿状态（生命周期随 route 树），
/// 避免在 route 退场动画期间 dispose 输入框（controller used after disposed）。
class _InstructorPickerSheet extends StatefulWidget {
  const _InstructorPickerSheet({
    required this.title,
    required this.copy,
    required this.initial,
  });

  final String title;
  final CourseCopy copy;
  final List<String> initial;

  @override
  State<_InstructorPickerSheet> createState() => _InstructorPickerSheetState();
}

class _InstructorPickerSheetState extends State<_InstructorPickerSheet> {
  late final TextEditingController _input = TextEditingController();
  late final List<String> _draft = List<String>.of(widget.initial);

  @override
  void dispose() {
    _input.dispose();
    super.dispose();
  }

  void _addDraft() {
    final String value = _input.text.trim();
    if (value.isEmpty || _draft.contains(value)) return;
    setState(() => _draft.add(value));
    _input.clear();
  }

  @override
  Widget build(BuildContext context) {
    final CourseCopy copy = widget.copy;
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);

    return SafeArea(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 8, 4),
            child: Row(
              children: <Widget>[
                Expanded(
                  child: Text(
                    widget.title,
                    style: type.heading.copyWith(fontWeight: FontWeight.w700),
                  ),
                ),
                TextButton(
                  onPressed: () => Navigator.pop(context, _draft),
                  child: Text(copy.done),
                ),
              ],
            ),
          ),
          const GfDivider(),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: GfInput(
              controller: _input,
              hintText: copy.instructorInputHint,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _addDraft(),
              suffixIcon: IconButton(
                icon: const Icon(Icons.add),
                tooltip: copy.instructorAdd,
                onPressed: _addDraft,
              ),
            ),
          ),
          Expanded(
            child: _draft.isEmpty
                ? Center(
                    child: Text(
                      copy.instructorEmptyHint,
                      style: type.small.copyWith(
                        color: colors.baseContent.withValues(alpha: 0.5),
                      ),
                    ),
                  )
                : ListView(
                    padding: const EdgeInsets.all(16),
                    children: <Widget>[
                      for (final String name in _draft)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 8),
                          child: Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 12,
                              vertical: 6,
                            ),
                            decoration: BoxDecoration(
                              color: colors.base200,
                              borderRadius: BorderRadius.circular(999),
                            ),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: <Widget>[
                                Flexible(
                                  child: Text(
                                    name,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: type.small,
                                  ),
                                ),
                                const SizedBox(width: 4),
                                InkWell(
                                  onTap: () =>
                                      setState(() => _draft.remove(name)),
                                  child: Icon(
                                    Icons.close,
                                    size: 16,
                                    color: colors.iconMuted,
                                  ),
                                ),
                              ],
                            ),
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

/// Captured criteria belong to one result generation, including all pages.
class _CatalogQuery {
  const _CatalogQuery({
    required this.keyword,
    required this.departments,
    required this.terms,
    required this.campuses,
    required this.instructors,
    required this.onlyWithReviews,
  });
  final String keyword;
  final List<String> departments, terms, campuses, instructors;
  final bool onlyWithReviews;
  Future<CourseListResultPayload> fetch(
    CourseRepository repository,
    int page,
    int size,
  ) => repository.list(
    keyword: keyword,
    departments: departments,
    terms: terms,
    campuses: campuses,
    instructors: instructors,
    onlyWithReviews: onlyWithReviews,
    page: page,
    size: size,
  );
}

class _CourseOptionsSheet extends StatefulWidget {
  const _CourseOptionsSheet({
    required this.title,
    required this.options,
    required this.labels,
    required this.initial,
  });
  final String title;
  final List<String> options;
  final Map<String, String> labels;
  final Set<String> initial;
  @override
  State<_CourseOptionsSheet> createState() => _CourseOptionsSheetState();
}

class _CourseOptionsSheetState extends State<_CourseOptionsSheet> {
  final _search = TextEditingController();
  late final _selected = {...widget.initial};
  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final query = _search.text.trim().toLowerCase();
    final options = widget.options
        .where(
          (value) =>
              value.toLowerCase().contains(query) ||
              (widget.labels[value] ?? '').toLowerCase().contains(query),
        )
        .toList();
    return SafeArea(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 8, 4),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    widget.title,
                    style: GfTheme.typographyOf(context).title2,
                  ),
                ),
                TextButton(
                  onPressed: () => Navigator.pop(context, _selected),
                  child: Text(l10n.courseCopyDone),
                ),
              ],
            ),
          ),
          Expanded(
            child: CustomScrollView(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              slivers: [
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    child: GfSearchField(
                      key: const Key('course-filter-search'),
                      controller: _search,
                      hintText: l10n.coursesFilterSearchHint,
                      clearLabel: l10n.courseCopyClearSearch,
                      onChanged: (_) => setState(() {}),
                      onClear: () => setState(_search.clear),
                    ),
                  ),
                ),
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    child: Wrap(
                      alignment: WrapAlignment.spaceBetween,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      spacing: 8,
                      children: [
                        Text(l10n.courseCopySelectedCount(_selected.length)),
                        TextButton(
                          onPressed: _selected.isEmpty
                              ? null
                              : () => setState(_selected.clear),
                          child: Text(l10n.coursesClearSelection),
                        ),
                      ],
                    ),
                  ),
                ),
                if (options.isEmpty)
                  SliverFillRemaining(
                    hasScrollBody: false,
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Center(
                        child: Text(
                          query.isEmpty
                              ? l10n.courseCopyNoOptions
                              : l10n.coursesFilterNoMatches,
                        ),
                      ),
                    ),
                  )
                else
                  SliverList.builder(
                    itemCount: options.length,
                    itemBuilder: (context, index) {
                      final value = options[index];
                      return CheckboxListTile(
                        value: _selected.contains(value),
                        title: Text(widget.labels[value] ?? value),
                        onChanged: (selected) => setState(() {
                          if (selected == true) {
                            _selected.add(value);
                          } else {
                            _selected.remove(value);
                          }
                        }),
                      );
                    },
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
