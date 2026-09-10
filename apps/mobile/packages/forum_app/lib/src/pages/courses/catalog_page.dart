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
class CourseCatalogPage extends ConsumerStatefulWidget {
  const CourseCatalogPage({super.key});

  @override
  ConsumerState<CourseCatalogPage> createState() => _CourseCatalogPageState();
}

class _CourseCatalogPageState extends ConsumerState<CourseCatalogPage> {
  static const double _loadMoreThreshold = 300;
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
    _loadOptions();
    _load();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (!_scrollController.hasClients) return;
    if (_scrollController.position.extentAfter < _loadMoreThreshold) {
      _loadMore();
    }
  }

  void _scheduleSearch(String _) {
    _debounce?.cancel();
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

  /// 拉取目录筛选值域（院系/学期/校区）。失败静默：chip 仍可点开，
  /// 空值域时 sheet 内展示无选项提示，不阻塞列表主流程。
  Future<void> _loadOptions() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/courses');
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
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
    } catch (_) {
      // 值域加载失败静默，筛选功能降级为可输入/无选项。
    }
  }

  Future<void> _load() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _page = const AsyncValue.loading());
    try {
      final CourseListResultPayload result = await _repository.list(
        keyword: _searchController.text.trim(),
        departments: _selectedDepartments.toList()..sort(),
        terms: _selectedTerms.toList()..sort(),
        campuses: _selectedCampuses.toList()..sort(),
        instructors: List<String>.of(_selectedInstructors),
        onlyWithReviews: _onlyWithReviews,
        page: 1,
        size: _pageSize,
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _courses = result.list;
        _hasNext = result.hasNext;
        _nextPage = 2;
        _page = AsyncValue.data(result);
      });
      if (_scrollController.hasClients) {
        _scrollController.jumpTo(0);
      }
    } catch (e, st) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _page = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadMore() async {
    if (_loadingMore || !_hasNext) return;
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _loadingMore = true);
    try {
      final CourseListResultPayload result = await _repository.list(
        keyword: _searchController.text.trim(),
        departments: _selectedDepartments.toList()..sort(),
        terms: _selectedTerms.toList()..sort(),
        campuses: _selectedCampuses.toList()..sort(),
        instructors: List<String>.of(_selectedInstructors),
        onlyWithReviews: _onlyWithReviews,
        page: _nextPage,
        size: _pageSize,
      );
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() {
        _courses = <CourseSummaryPayload>[..._courses, ...result.list];
        _hasNext = result.hasNext;
        _nextPage += 1;
      });
    } catch (_) {
      // 加载更多失败静默，滚动可再次触发。
    } finally {
      if (mounted) setState(() => _loadingMore = false);
    }
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
    final AppLocalizations l10n = AppLocalizations.of(context);
    final CourseCopy copy = CourseCopy(l10n);
    final Set<String> draft = <String>{...initial};
    final Set<String>? result = await showGfBottomSheet<Set<String>>(
      context,
      height: 460,
      builder: (BuildContext ctx) => SafeArea(
        child: StatefulBuilder(
          builder: (BuildContext ctx, StateSetter setSheet) {
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 8, 4),
                  child: Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          title,
                          style: GfTheme.typographyOf(
                            context,
                          ).heading.copyWith(fontWeight: FontWeight.w700),
                        ),
                      ),
                      TextButton(
                        onPressed: () => Navigator.pop(ctx, draft),
                        child: Text(copy.done),
                      ),
                    ],
                  ),
                ),
                const GfDivider(),
                if (options.isEmpty)
                  Padding(
                    padding: const EdgeInsets.all(24),
                    child: Center(
                      child: Text(
                        copy.noOptions,
                        style: GfTheme.typographyOf(context).small.copyWith(
                          color: GfTheme.colorsOf(
                            context,
                          ).baseContent.withValues(alpha: 0.55),
                        ),
                      ),
                    ),
                  )
                else
                  Expanded(
                    child: ListView.builder(
                      itemCount: options.length,
                      itemBuilder: (BuildContext ctx, int index) {
                        final String value = options[index];
                        final String label = labelByValue[value] ?? value;
                        final bool checked = draft.contains(value);
                        return InkWell(
                          onTap: () {
                            setSheet(() {
                              if (checked) {
                                draft.remove(value);
                              } else {
                                draft.add(value);
                              }
                            });
                          },
                          child: Padding(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 12,
                            ),
                            child: Row(
                              children: <Widget>[
                                Icon(
                                  checked
                                      ? Icons.check_circle
                                      : Icons.radio_button_unchecked,
                                  size: 20,
                                  color: checked
                                      ? GfTheme.colorsOf(context).primary
                                      : GfTheme.colorsOf(
                                          context,
                                        ).baseContent.withValues(alpha: 0.35),
                                ),
                                const SizedBox(width: 12),
                                Expanded(
                                  child: Text(
                                    label,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: GfTheme.typographyOf(context).body
                                        .copyWith(
                                          color: GfTheme.colorsOf(context)
                                              .baseContent
                                              .withValues(
                                                alpha: checked ? 1 : 0.7,
                                              ),
                                          fontWeight: checked
                                              ? FontWeight.w600
                                              : FontWeight.w400,
                                        ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        );
                      },
                    ),
                  ),
              ],
            );
          },
        ),
      ),
    );
    if (result != null && mounted) {
      onApplied(result);
    }
  }

  /// 教师筛选：web 为自由文本输入（逗号分隔），移动端改为底部 sheet 内
  /// 逐条添加 token；已选值以 chip 呈现并可移除。
  Future<void> _pickInstructors() async {
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
    if (result != null && mounted) {
      _applyInstructors(result);
    }
  }

  // ---- UI ----

  @override
  Widget build(BuildContext context) {
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
      onTap: () => _pickOptionValues(
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
      onTap: () => _pickOptionValues(
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
      onTap: () => _pickOptionValues(
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

    return SizedBox(
      height: 46,
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
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
    required VoidCallback onTap,
    int? count,
    IconData? icon,
  }) {
    final GfColors colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(999),
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 12),
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

  Widget _buildList(AppLocalizations l10n) {
    final CourseCopy copy = CourseCopy(l10n);
    return AppRefreshIndicator(
      onRefresh: _load,
      child: _courses.isEmpty
          ? ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              children: <Widget>[
                SizedBox(
                  height: MediaQuery.sizeOf(context).height * 0.6,
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
              ],
            )
          : ListView.separated(
              key: const Key('course-catalog-list'),
              controller: _scrollController,
              physics: const AlwaysScrollableScrollPhysics(),
              itemCount: _courses.length + 1,
              separatorBuilder: (_, _) => const GfDivider(),
              itemBuilder: (BuildContext context, int index) {
                if (index == _courses.length) {
                  return _ListFooter(loading: _loadingMore, hasNext: _hasNext);
                }
                final CourseSummaryPayload course = _courses[index];
                return _CourseRow(
                  course: course,
                  termsExpanded: _expandedTermRows.contains(course.id),
                  onToggleTerms: () {
                    setState(() {
                      if (!_expandedTermRows.remove(course.id)) {
                        _expandedTermRows.add(course.id);
                      }
                    });
                  },
                  onTap: () => context.push('/courses/${course.id}'),
                );
              },
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

class _ListFooter extends StatelessWidget {
  const _ListFooter({required this.loading, required this.hasNext});

  final bool loading;
  final bool hasNext;

  @override
  Widget build(BuildContext context) {
    if (loading) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 16),
        child: Center(child: GfLoadingIndicator(small: true)),
      );
    }
    if (!hasNext) {
      return const SizedBox(height: 24);
    }
    // hasNext 且未在加载：等待滚动触发，占位保持一致高度。
    return const SizedBox(height: 24);
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
