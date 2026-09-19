import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/campus_shortcuts.dart';
import '../../widgets/status_views.dart';

final campusCoursesProvider =
    FutureProvider.autoDispose<CourseListResultPayload>(
      (ref) => ref
          .watch(courseRepositoryProvider)
          .list(size: 3, onlyWithReviews: true),
    );

/// Campus tools, course previews and the authenticated official campus workspace.
class CampusPage extends ConsumerStatefulWidget {
  const CampusPage({super.key});
  @override
  ConsumerState<CampusPage> createState() => _CampusPageState();
}

class _CampusPageState extends ConsumerState<CampusPage> {
  final _scroll = GfScrollToTopController();
  late final GfTabScrollRegistry _registry;
  @override
  void initState() {
    super.initState();
    _registry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.campus, _scroll);
  }

  @override
  void dispose() {
    _registry.unregister(GfShellDestination.campus, _scroll);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return RootSurface(
      title: l10n.navCampus,
      actions: [
        IconButton(
          tooltip: l10n.commonSearch,
          icon: const GfSymbol('search'),
          onPressed: () => context.push('/search'),
        ),
      ],
      body: (top, bottom) => GfScrollToTop(
        controller: _scroll,
        showButton: false,
        semanticLabel: l10n.commonBackToTop,
        builder: (_, controller) => AppRefreshIndicator(
          edgeOffset: top,
          onRefresh: () => ref.refresh(campusCoursesProvider.future),
          child: ListView(
            controller: controller,
            physics: const AlwaysScrollableScrollPhysics(),
            padding: EdgeInsets.fromLTRB(20, top + 20, 20, bottom + 24),
            children: [
              InkWell(
                onTap: () => context.push('/courses'),
                borderRadius: BorderRadius.circular(28),
                child: Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: colors.base200,
                    borderRadius: BorderRadius.circular(28),
                  ),
                  child: Row(
                    children: [
                      const GfSymbol('search', size: 20),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(l10n.coursesSearchHint, style: type.small),
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              const CampusShortcuts(),
              const SizedBox(height: 16),
              ListTile(
                leading: const GfSymbol('verified_user'),
                title: Text(l10n.campusOfficialTitle, style: type.heading),
                subtitle: Text(
                  l10n.campusOfficialSubtitle,
                  style: type.caption,
                ),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.push('/campus/official'),
              ),
              const SizedBox(height: 32),
              Text(l10n.campusCoursesTitle, style: type.title2),
              const SizedBox(height: 8),
              Text(
                l10n.campusExploreCourses,
                style: type.small.copyWith(color: colors.iconMuted),
              ),
              const SizedBox(height: 16),
              ref
                  .watch(campusCoursesProvider)
                  .when(
                    loading: () => const Padding(
                      padding: EdgeInsets.all(24),
                      child: GfLoading(),
                    ),
                    error: (_, _) => TextButton(
                      onPressed: () => ref.invalidate(campusCoursesProvider),
                      child: Text(l10n.commonRetry),
                    ),
                    data: (result) => result.list.isEmpty
                        ? GfEmpty(
                            message: l10n.campusCoursesEmpty,
                            icon: Icons.school_outlined,
                          )
                        : Material(
                            color: colors.base200,
                            borderRadius: BorderRadius.circular(20),
                            clipBehavior: Clip.antiAlias,
                            child: Column(
                              children: [
                                for (final course in result.list)
                                  ListTile(
                                    contentPadding: const EdgeInsets.symmetric(
                                      horizontal: 16,
                                      vertical: 8,
                                    ),
                                    title: Text(
                                      course.name,
                                      style: type.heading,
                                    ),
                                    subtitle: Text(
                                      [
                                        course.department,
                                        course.teacherName ?? '',
                                      ].where((s) => s.isNotEmpty).join(' · '),
                                      style: type.caption,
                                    ),
                                    trailing: Row(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        GfSymbol(
                                          'star',
                                          size: 16,
                                          color: colors.primary,
                                        ),
                                        const SizedBox(width: 4),
                                        Text(
                                          course.ratingAvg?.toStringAsFixed(
                                                1,
                                              ) ??
                                              '—',
                                        ),
                                      ],
                                    ),
                                    onTap: () =>
                                        context.push('/courses/${course.id}'),
                                  ),
                              ],
                            ),
                          ),
                  ),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton(
                  onPressed: () => context.push('/courses'),
                  child: Text(l10n.coursesTitle),
                ),
              ),
              const SizedBox(height: 24),
              Container(
                padding: const EdgeInsets.all(24),
                decoration: BoxDecoration(
                  color: colors.primary.withValues(alpha: 0.06),
                  borderRadius: BorderRadius.circular(24),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfIconTile('calendar-days', size: 40),
                    const SizedBox(height: 16),
                    Text(l10n.campusPlanTitle, style: type.title2),
                    const SizedBox(height: 8),
                    Text(
                      l10n.campusPlanDescription,
                      style: type.small.copyWith(
                        height: 1.5,
                        color: colors.iconMuted,
                      ),
                    ),
                    const SizedBox(height: 24),
                    GfButton(
                      label: l10n.scheduleTitle,
                      expanded: true,
                      onPressed: () => context.push('/schedule'),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
