import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/courses/review_reaction.dart';

ReviewPayload _review({
  bool helpful = false,
  bool disliked = false,
  int? helpfulCount,
  int? dislikeCount,
}) =>
    ReviewPayload(
      id: 1,
      offeringId: 2,
      rating: 4,
      content: 'Good',
      contentHtml: '<p>Good</p>',
      author: const ReviewAuthorPayload(kind: 'member', label: 'Reader'),
      viewer: ReviewViewerPayload(
        canEdit: false,
        canDelete: false,
        isHelpful: helpful,
        isDisliked: disliked,
      ),
      helpfulCount: helpfulCount ?? (helpful ? 1 : 0),
      dislikeCount: dislikeCount ?? (disliked ? 1 : 0),
      createdAt: '2026-09-30T00:00:00Z',
      updatedAt: '2026-09-30T00:00:00Z',
    );

void main() {
  test(
    'switching review reaction clears the opposite count and viewer flag',
    () {
      final next = applyCourseReviewReaction(
        _review(disliked: true),
        CourseReviewReaction.helpful,
        on: true,
      );

      expect(next.viewer.isHelpful, isTrue);
      expect(next.viewer.isDisliked, isFalse);
      expect(next.helpfulCount, 1);
      expect(next.dislikeCount, 0);
    },
  );

  test('removing a selected reaction never makes its count negative', () {
    final next = applyCourseReviewReaction(
      _review(helpful: true).copyWith(helpfulCount: 0),
      CourseReviewReaction.helpful,
      on: false,
    );

    expect(next.viewer.isHelpful, isFalse);
    expect(next.helpfulCount, 0);
  });

  test('switching helpful to dislike clears helpful and changes both counts', () {
    final next = applyCourseReviewReaction(
      _review(helpful: true),
      CourseReviewReaction.dislike,
      on: true,
    );

    expect(next.viewer.isHelpful, isFalse);
    expect(next.viewer.isDisliked, isTrue);
    expect(next.helpfulCount, 0);
    expect(next.dislikeCount, 1);
  });

  test('removing dislike clears its flag without enabling helpful', () {
    final next = applyCourseReviewReaction(
      _review(disliked: true),
      CourseReviewReaction.dislike,
      on: false,
    );

    expect(next.viewer.isHelpful, isFalse);
    expect(next.viewer.isDisliked, isFalse);
    expect(next.dislikeCount, 0);
  });

  test('both unselected baseline can turn each reaction on and back off', () {
    final helpful = applyCourseReviewReaction(
      _review(),
      CourseReviewReaction.helpful,
      on: true,
    );
    final unhelpful = applyCourseReviewReaction(
      helpful,
      CourseReviewReaction.helpful,
      on: false,
    );
    final disliked = applyCourseReviewReaction(
      _review(),
      CourseReviewReaction.dislike,
      on: true,
    );
    final notDisliked = applyCourseReviewReaction(
      disliked,
      CourseReviewReaction.dislike,
      on: false,
    );

    expect(unhelpful.viewer.isHelpful, isFalse);
    expect(unhelpful.helpfulCount, 0);
    expect(notDisliked.viewer.isDisliked, isFalse);
    expect(notDisliked.dislikeCount, 0);
  });

  test('both count underflows are clamped', () {
    final noHelpful = applyCourseReviewReaction(
      _review(helpful: true, helpfulCount: 0),
      CourseReviewReaction.helpful,
      on: false,
    );
    final noDislikes = applyCourseReviewReaction(
      _review(disliked: true, dislikeCount: 0),
      CourseReviewReaction.dislike,
      on: false,
    );

    expect(noHelpful.helpfulCount, 0);
    expect(noDislikes.dislikeCount, 0);
  });

  test('a reaction toggled after the request started survives a stale row', () {
    final ReviewPayload server = _review();
    final ReviewPayload local = applyCourseReviewReaction(
      server,
      CourseReviewReaction.helpful,
      on: true,
    );

    final merged = mergeCourseReviewReactions(
      incoming: <ReviewPayload>[server],
      local: <ReviewPayload>[local],
      // 请求发起时还没有任何本地反应版本，响应到达时已经切换到 v1。
      versionsAtStart: const <int, int>{},
      currentVersions: <int, int>{1: 1},
      busyAtStart: const <int>{},
    );

    expect(merged.single.viewer.isHelpful, isTrue);
    expect(merged.single.viewer.isDisliked, isFalse);
    expect(merged.single.helpfulCount, 1);
    expect(merged.single.dislikeCount, 0);
  });

  test('a request started while the write is pending keeps local state', () {
    final ReviewPayload server = _review(helpful: true);
    final ReviewPayload local = applyCourseReviewReaction(
      _review(),
      CourseReviewReaction.dislike,
      on: true,
    );

    final merged = mergeCourseReviewReactions(
      incoming: <ReviewPayload>[server],
      local: <ReviewPayload>[local],
      versionsAtStart: <int, int>{1: 1},
      currentVersions: <int, int>{1: 1},
      busyAtStart: const <int>{1},
    );

    expect(merged.single.viewer.isHelpful, isFalse);
    expect(merged.single.viewer.isDisliked, isTrue);
    expect(merged.single.helpfulCount, 0);
    expect(merged.single.dislikeCount, 1);
  });

  test('fresh server rows win when no local reaction changed', () {
    final ReviewPayload server = _review(helpful: true, helpfulCount: 5);
    final ReviewPayload local = _review(helpful: true, helpfulCount: 1);

    final merged = mergeCourseReviewReactions(
      incoming: <ReviewPayload>[server],
      local: <ReviewPayload>[local],
      versionsAtStart: <int, int>{1: 4},
      currentVersions: <int, int>{1: 4},
      busyAtStart: const <int>{},
    );

    expect(merged.single.helpfulCount, 5);
  });

  test('server rows win when the local reaction version did not change', () {
    final ReviewPayload server = _review(helpful: false, helpfulCount: 2);
    final ReviewPayload local = applyCourseReviewReaction(
      _review(helpful: true, helpfulCount: 9),
      CourseReviewReaction.helpful,
      on: false,
    );

    final merged = mergeCourseReviewReactions(
      incoming: <ReviewPayload>[server],
      local: <ReviewPayload>[local],
      versionsAtStart: <int, int>{1: 1},
      currentVersions: <int, int>{1: 1},
      busyAtStart: const <int>{},
    );

    expect(merged.single.helpfulCount, 2);
    expect(merged.single.viewer.isHelpful, isFalse);
  });
}
