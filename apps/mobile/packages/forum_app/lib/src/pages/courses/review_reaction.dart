import 'package:core/core.dart';

enum CourseReviewReaction { helpful, dislike }

/// Applies a completed server reaction while keeping the two counts consistent.
ReviewPayload applyCourseReviewReaction(
  ReviewPayload review,
  CourseReviewReaction reaction, {
  required bool on,
}) {
  final bool helpful = reaction == CourseReviewReaction.helpful;
  final bool nextHelpful = helpful
      ? on
      : (on ? false : review.viewer.isHelpful);
  final bool nextDisliked = helpful
      ? (on ? false : review.viewer.isDisliked)
      : on;
  final int helpfulCount =
      review.helpfulCount +
      (nextHelpful ? 1 : 0) -
      (review.viewer.isHelpful ? 1 : 0);
  final int dislikeCount =
      review.dislikeCount +
      (nextDisliked ? 1 : 0) -
      (review.viewer.isDisliked ? 1 : 0);
  return review.copyWith(
    viewer: review.viewer.copyWith(
      isHelpful: nextHelpful,
      isDisliked: nextDisliked,
    ),
    helpfulCount: helpfulCount < 0 ? 0 : helpfulCount,
    dislikeCount: dislikeCount < 0 ? 0 : dislikeCount,
  );
}
