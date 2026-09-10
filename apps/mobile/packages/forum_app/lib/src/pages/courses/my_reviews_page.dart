import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../widgets/status_views.dart';
import 'course_common.dart';
import 'review_form_sheet.dart';
import 'review_delete_dialog.dart';

class MyCourseReviewsPage extends ConsumerStatefulWidget {
  const MyCourseReviewsPage({super.key});
  @override
  ConsumerState<MyCourseReviewsPage> createState() =>
      _MyCourseReviewsPageState();
}

class _MyCourseReviewsPageState extends ConsumerState<MyCourseReviewsPage> {
  List<OwnCourseReviewItem> _items = [];
  String _cursor = '';
  bool _loading = true;
  bool _loadingMore = false;
  Object? _error;
  int _sequence = 0;
  final _busy = <int>{};
  CourseRepository get _repository => ref.read(courseRepositoryProvider);

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool more = false}) async {
    if (more && (_loading || _loadingMore || _cursor.isEmpty)) return;
    final seq = more ? _sequence : ++_sequence;
    final epoch = ref.read(offlineCacheEpochProvider);
    setState(() {
      if (more) {
        _loadingMore = true;
      } else {
        _loading = true;
        _error = null;
      }
    });
    try {
      final page = await _repository.ownReviews(cursor: more ? _cursor : '');
      if (!mounted ||
          seq != _sequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      setState(() {
        final combined = [
          ...(more ? _items : <OwnCourseReviewItem>[]),
          ...page.list,
        ];
        _items = {
          for (final item in combined) item.review.id: item,
        }.values.toList();
        _cursor = page.nextCursor;
      });
    } catch (error) {
      if (!mounted ||
          seq != _sequence ||
          epoch != ref.read(offlineCacheEpochProvider)) {
        return;
      }
      if (more) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      } else {
        setState(() => _error = error);
      }
    } finally {
      if (mounted && seq == _sequence) {
        setState(() {
          _loading = false;
          _loadingMore = false;
        });
      }
    }
  }

  Future<void> _edit(OwnCourseReviewItem item) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    final updated = await showGfBottomSheet<ReviewPayload>(
      context,
      height: 600,
      keyboardAware: true,
      builder: (_) => CourseReviewFormSheet(
        pageContext: context,
        repository: _repository,
        offerings: const [],
        editing: item.review,
      ),
    );
    if (!mounted ||
        updated == null ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    _sequence++;
    setState(() {
      _loading = false;
      _loadingMore = false;
      _items = [
        for (final value in _items)
          if (value.review.id == updated.id)
            value.withReview(updated)
          else
            value,
      ];
    });
    showGfToast(
      context,
      CourseCopy(AppLocalizations.of(context)).updateSuccess,
    );
  }

  Future<void> _delete(OwnCourseReviewItem item) async {
    final epoch = ref.read(offlineCacheEpochProvider);
    if (!await confirmCourseReviewDeletion(context, item.review) ||
        !mounted ||
        epoch != ref.read(offlineCacheEpochProvider)) {
      return;
    }
    setState(() => _busy.add(item.review.id));
    try {
      await _repository.deleteReview(item.review.id);
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      _sequence++;
      setState(() {
        _loading = false;
        _loadingMore = false;
        _items.removeWhere((value) => value.review.id == item.review.id);
      });
      showGfToast(
        context,
        CourseCopy(AppLocalizations.of(context)).reviewDeleted,
      );
    } catch (error) {
      if (mounted && epoch == ref.read(offlineCacheEpochProvider)) {
        showGfToast(
          context,
          courseReviewError(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _busy.remove(item.review.id));
    }
  }

  Future<void> _open(OwnCourseReviewItem item) async {
    await context.push(
      '/courses/${item.courseId}?offeringId=${item.review.offeringId}&reviewId=${item.review.id}',
    );
    if (mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.myCourseReviewsTitle)),
      body: AppRefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: EdgeInsets.fromLTRB(
            16,
            8,
            16,
            24 + MediaQuery.paddingOf(context).bottom,
          ),
          children: [
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(32),
                child: Center(child: GfLoadingIndicator()),
              )
            else if (_error != null)
              GfErrorRetry(
                message: resolveErrorMessage(l10n, _error!),
                onRetry: _load,
              )
            else if (_items.isEmpty)
              GfEmpty(
                icon: Icons.rate_review_outlined,
                message: l10n.myCourseReviewsEmpty,
              )
            else
              for (final item in _items)
                Container(
                  key: ValueKey('own-review-${item.review.id}'),
                  margin: const EdgeInsets.only(bottom: 12),
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: colors.base100,
                    border: Border.all(color: colors.line),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        item.courseName.isEmpty
                            ? '#${item.courseId}'
                            : item.courseName,
                        style: type.heading,
                      ),
                      const SizedBox(height: 6),
                      Text(
                        '${item.courseCode} · ${item.review.createdAt.split('T').first}',
                        style: type.caption.copyWith(
                          color: colors.baseContent.withValues(alpha: .55),
                        ),
                      ),
                      const SizedBox(height: 10),
                      Row(
                        children: [
                          GfSymbol('star', size: 16, color: colors.warning),
                          const SizedBox(width: 6),
                          Text(
                            item.review.rating == null
                                ? '—'
                                : '${item.review.rating} / 5',
                            style: type.small,
                          ),
                          if (item.review.author.kind == 'anonymous') ...[
                            const SizedBox(width: 12),
                            Text(
                              CourseCopy(l10n).anonymousLabel,
                              style: type.caption,
                            ),
                          ],
                        ],
                      ),
                      const SizedBox(height: 12),
                      Text(
                        item.review.content,
                        maxLines: 5,
                        overflow: TextOverflow.ellipsis,
                        style: type.body,
                      ),
                      if (!item.canOpenCourse)
                        Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: Text(
                            item.hidden
                                ? l10n.myCourseReviewHidden
                                : l10n.myCourseReviewUnavailable,
                            style: type.caption.copyWith(
                              color: colors.baseContent.withValues(alpha: .55),
                            ),
                          ),
                        ),
                      const SizedBox(height: 8),
                      Wrap(
                        spacing: 4,
                        children: [
                          if (item.canOpenCourse)
                            TextButton(
                              onPressed: () => _open(item),
                              child: Text(l10n.myCourseReviewOpen),
                            ),
                          if (item.review.viewer.canEdit)
                            TextButton(
                              onPressed: _busy.contains(item.review.id)
                                  ? null
                                  : () => _edit(item),
                              child: Text(l10n.commonEdit),
                            ),
                          if (item.review.viewer.canDelete)
                            TextButton(
                              onPressed: _busy.contains(item.review.id)
                                  ? null
                                  : () => _delete(item),
                              style: TextButton.styleFrom(
                                foregroundColor: colors.error,
                              ),
                              child: Text(CourseCopy(l10n).delete),
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
            if (!_loading && _cursor.isNotEmpty)
              Center(
                child: _loadingMore
                    ? const GfLoadingIndicator()
                    : TextButton(
                        onPressed: () => _load(more: true),
                        child: Text(l10n.commonLoadMore),
                      ),
              ),
          ],
        ),
      ),
    );
  }
}
