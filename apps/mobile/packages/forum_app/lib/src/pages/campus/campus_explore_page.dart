import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../widgets/campus_shortcuts.dart';
import '../../widgets/status_views.dart';

final campusCoursesProvider =
    FutureProvider.autoDispose<CourseListResultPayload>(
      (ref) => ref
          .watch(courseRepositoryProvider)
          .list(size: 3, onlyWithReviews: true),
    );

/// Existing public campus tools remain reachable outside the private overview.
class CampusExplorePage extends ConsumerWidget {
  const CampusExplorePage({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l.campusExplore)),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          const CampusShortcuts(),
          const SizedBox(height: 24),
          Text(
            l.campusCoursesTitle,
            style: GfTheme.typographyOf(context).title2,
          ),
          const SizedBox(height: 12),
          ref
              .watch(campusCoursesProvider)
              .when(
                loading: () => const GfLoading(),
                error: (_, _) => GfErrorRetry(
                  message: l.commonRetry,
                  onRetry: () => ref.invalidate(campusCoursesProvider),
                ),
                data: (result) => result.list.isEmpty
                    ? GfEmpty(message: l.campusCoursesEmpty)
                    : Column(
                        children: [
                          for (final course in result.list)
                            ListTile(
                              title: Text(course.name),
                              subtitle: Text(course.department),
                              trailing: Text(
                                course.ratingAvg?.toStringAsFixed(1) ?? '—',
                              ),
                              onTap: () =>
                                  context.push('/courses/${course.id}'),
                            ),
                        ],
                      ),
              ),
          TextButton(
            onPressed: () => context.push('/courses'),
            child: Text(l.coursesTitle),
          ),
          const SizedBox(height: 24),
          Text(l.campusPlanDescription),
          const SizedBox(height: 12),
          GfButton(
            label: l.scheduleTitle,
            onPressed: () => context.push('/schedule'),
          ),
        ],
      ),
    );
  }
}
