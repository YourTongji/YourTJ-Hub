import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';

final campusCoursesProvider =
    FutureProvider.autoDispose<CourseListResultPayload>(
      (ref) => ref
          .watch(courseRepositoryProvider)
          .list(size: 3, onlyWithReviews: true),
    );

/// Public course recommendations shared by signed-in and guest landing views.
class CampusCoursePicks extends ConsumerWidget {
  const CampusCoursePicks({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfText(
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
                            title: GfText(course.name),
                            subtitle: GfText(course.department),
                            trailing: GfText(
                              course.ratingAvg?.toStringAsFixed(1) ?? '—',
                            ),
                            onTap: () => context.push('/courses/${course.id}'),
                          ),
                      ],
                    ),
            ),
        TextButton(
          onPressed: () => context.push('/courses'),
          child: GfText(l.coursesTitle),
        ),
      ],
    );
  }
}
