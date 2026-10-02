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

/// 把服务端列表行与本地反应状态合并。
///
/// 列表请求在途期间可能发生反应切换（[currentVersions] 与 [versionsAtStart]
/// 不同），或请求发起时写入尚未提交（[busyAtStart] 包含该评价）；这两类评价的
/// 服务端返回值可能是旧快照，必须保留 [local] 中的 viewer 与计数，否则慢响应会
/// 回滚乐观更新。其余评价采用服务端值（内容/作者/编辑状态都以服务端为准）。
List<ReviewPayload> mergeCourseReviewReactions({
  required List<ReviewPayload> incoming,
  required List<ReviewPayload> local,
  required Map<int, int> versionsAtStart,
  required Map<int, int> currentVersions,
  required Set<int> busyAtStart,
}) {
  if (incoming.isEmpty) {
    return incoming;
  }
  final Map<int, ReviewPayload> localById = <int, ReviewPayload>{
    for (final ReviewPayload review in local) review.id: review,
  };
  return <ReviewPayload>[
    for (final ReviewPayload review in incoming)
      if (busyAtStart.contains(review.id) ||
          (currentVersions[review.id] ?? 0) !=
              (versionsAtStart[review.id] ?? 0))
        _keepLocalReaction(review, localById[review.id])
      else
        review,
  ];
}

ReviewPayload _keepLocalReaction(ReviewPayload server, ReviewPayload? local) {
  if (local == null) return server;
  return server.copyWith(
    viewer: server.viewer.copyWith(
      isHelpful: local.viewer.isHelpful,
      isDisliked: local.viewer.isDisliked,
    ),
    helpfulCount: local.helpfulCount,
    dislikeCount: local.dislikeCount,
  );
}
