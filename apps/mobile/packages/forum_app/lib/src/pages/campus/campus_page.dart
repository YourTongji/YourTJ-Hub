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

final campusCoursesProvider =
    FutureProvider.autoDispose<CourseListResultPayload>(
      (ref) => ref
          .watch(courseRepositoryProvider)
          .list(size: 3, onlyWithReviews: true),
    );

/// Campus tools and real course previews; no official personal timetable.
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
            padding: EdgeInsets.fromLTRB(16, top + 16, 16, bottom),
            children: [
              InkWell(
                onTap: () => context.push('/courses'),
                borderRadius: BorderRadius.circular(8),
                child: Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: colors.base200,
                    borderRadius: BorderRadius.circular(8),
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
              Row(
                children: [
                  for (final entry in [
                    ('graduation-cap', l10n.coursesTitle, '/courses'),
                    ('calendar-days', l10n.scheduleTitle, '/schedule'),
                    ('book-open', l10n.wikiTitle, '/wiki'),
                  ])
                    Expanded(
                      child: InkWell(
                        onTap: () => context.push(entry.$3),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          child: Column(
                            children: [
                              GfSymbol(entry.$1, size: 28),
                              const SizedBox(height: 12),
                              Text(
                                entry.$2,
                                textAlign: TextAlign.center,
                                style: type.small,
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 16),
              const Divider(),
              const SizedBox(height: 24),
              Text(l10n.campusExploreCourses, style: type.title2),
              const SizedBox(height: 12),
              ref
                  .watch(campusCoursesProvider)
                  .when(
                    loading: () => const LinearProgressIndicator(),
                    error: (_, _) => TextButton(
                      onPressed: () => ref.invalidate(campusCoursesProvider),
                      child: Text(l10n.commonRetry),
                    ),
                    data: (result) => Column(
                      children: [
                        for (final course in result.list)
                          ListTile(
                            contentPadding: EdgeInsets.zero,
                            title: Text(course.name, style: type.heading),
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
                                  course.ratingAvg?.toStringAsFixed(1) ?? '—',
                                ),
                              ],
                            ),
                            onTap: () => context.push('/courses/${course.id}'),
                          ),
                      ],
                    ),
                  ),
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton(
                  onPressed: () => context.push('/courses'),
                  child: Text(l10n.coursesTitle),
                ),
              ),
              const SizedBox(height: 16),
              const Divider(),
              const SizedBox(height: 24),
              Text(l10n.campusPlanTitle, style: type.heading),
              const SizedBox(height: 8),
              Text(
                l10n.campusPlanDescription,
                style: type.body.copyWith(color: colors.iconMuted),
              ),
              const SizedBox(height: 16),
              Align(
                alignment: Alignment.centerLeft,
                child: OutlinedButton(
                  onPressed: () => context.push('/schedule'),
                  child: Text(l10n.scheduleTitle),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
