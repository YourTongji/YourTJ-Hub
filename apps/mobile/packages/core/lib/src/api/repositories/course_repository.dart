import '../../gen/own_course_reviews.dart';
import '../../gen/content_pages.dart';
import '../../gen/course_catalog.dart';
import '../../gen/course_review.dart';
import '../../gen/course_summary.dart';
import '../gf_api_client.dart';

/// 课程目录/详情/课评域（`/api/forum/courses*`、`/api/forum/course-reviews*`）。
///
/// forum 信封 `{code,result,messageCode,params}` 由 [GfApiClient.get/post]
/// 统一解包；公开读操作可选 JWT 个性化 viewer 态；我的课评与写操作需登录。
class CourseRepository {
  CourseRepository(this._client);

  final GfApiClient _client;

  static const _base = '/api/forum';

  /// 目录列表（滚动分页：hasNext=false 停止；筛选多值用逗号拼接，
  /// 契约为 repeated/comma-separated OR-joined）。
  Future<CourseListResultPayload> list({
    String? keyword,
    List<String> departments = const [],
    List<String> terms = const [],
    List<String> campuses = const [],
    List<String> instructors = const [],
    bool onlyWithReviews = false,
    String sortBy = '',
    int page = 1,
    int size = 20,
  }) => _client.get<CourseListResultPayload>(
    '$_base/courses',
    queryParameters: {
      if (keyword != null && keyword.isNotEmpty) 'keyword': keyword,
      if (departments.isNotEmpty) 'department': departments.join(','),
      if (terms.isNotEmpty) 'term': terms.join(','),
      if (campuses.isNotEmpty) 'campus': campuses.join(','),
      if (instructors.isNotEmpty) 'instructor': instructors.join(','),
      if (onlyWithReviews) 'onlyWithReviews': true,
      if (sortBy.isNotEmpty) 'sortBy': sortBy,
      'page': page,
      'size': size,
    },
    parser: (json) => CourseListResultPayload.fromJson(
      Map<String, dynamic>.from(json as Map),
    ),
  );

  /// 课程详情（含 offerings/沿革/评分分布；隐藏或不存在 → 404）。
  Future<CourseDetailPayload> detail(int courseId) =>
      _client.get<CourseDetailPayload>(
        '$_base/courses/$courseId',
        parser: (json) => CourseDetailPayload.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );

  /// 相关课程 + 沿革关系（教师他课/同课他师/lineage）。
  Future<CourseRelatedResult> related(int courseId) =>
      _client.get<CourseRelatedResult>(
        '$_base/courses/$courseId/related',
        parser: (json) => CourseRelatedResult.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );

  /// AI 课程总结（refresh=true 强制重评；check=true 只读预检不消耗限流）。
  Future<CourseAiSummaryResult> aiSummary(
    int courseId, {
    bool refresh = false,
    bool check = false,
  }) {
    return _client.get<CourseAiSummaryResult>(
      '$_base/courses/$courseId/summary',
      queryParameters: {if (refresh) 'refresh': true, if (check) 'check': true},
      parser: (json) => CourseAiSummaryResult.fromJson(
        Map<String, dynamic>.from(json as Map),
      ),
    );
  }

  /// 收藏/取消收藏（幂等；action 1=收藏 2=取消）。
  Future<bool> bookmark({
    required int courseId,
    required bool bookmarked,
  }) async {
    await _client.post<Object?>(
      '$_base/courses/bookmark',
      body: {'courseId': courseId, 'action': bookmarked ? 1 : 2},
    );
    return true;
  }

  /// 课评列表（cursor 分页，nextCursor 透传；offeringId 聚焦单教学班）。
  Future<ReviewListResult> reviews(
    int courseId, {
    int? offeringId,
    String? cursor,
    int pageSize = 20,
  }) => _client.get<ReviewListResult>(
    '$_base/courses/$courseId/reviews',
    queryParameters: {
      'offeringId': ?offeringId,
      if (cursor != null && cursor.isNotEmpty) 'cursor': cursor,
      'pageSize': pageSize,
    },
    parser: (json) =>
        ReviewListResult.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// Private, session-scoped review management across all courses.
  Future<OwnCourseReviewPage> ownReviews({
    String cursor = '',
    int pageSize = 20,
  }) => _client.get<OwnCourseReviewPage>(
    '$_base/my-course-reviews',
    queryParameters: {
      if (cursor.isNotEmpty) 'cursor': cursor,
      'pageSize': pageSize,
    },
    parser: (json) =>
        OwnCourseReviewPage.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 写课评。
  Future<ReviewPayload> createReview(CreateCourseReviewInput input) =>
      _client.post<ReviewPayload>(
        '$_base/course-reviews',
        body: input.toJson(),
        parser: (json) =>
            ReviewPayload.fromJson(Map<String, dynamic>.from(json as Map)),
      );

  /// 改课评。
  Future<ReviewPayload> updateReview(
    int reviewId,
    UpdateCourseReviewInput input,
  ) => _client.post<ReviewPayload>(
    '$_base/course-reviews/$reviewId',
    body: input.toJson(),
    parser: (json) =>
        ReviewPayload.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 删课评（幂等）。
  Future<bool> deleteReview(int reviewId) async {
    // DELETE 语义经 POST 兼容通道不可用；后端契约走 DELETE 方法。
    await _client.delete<Object?>('$_base/course-reviews/$reviewId');
    return true;
  }

  /// 有用/无用标记（幂等切换）。
  Future<bool> markHelpful(int reviewId, {required bool on}) =>
      _reviewAction('helpful', reviewId, on);

  /// 无用标记。
  Future<bool> markDislike(int reviewId, {required bool on}) =>
      _reviewAction('dislike', reviewId, on);

  Future<bool> _reviewAction(String kind, int reviewId, bool on) async {
    if (on) {
      await _client.put<Object?>('$_base/course-reviews/$reviewId/$kind');
    } else {
      await _client.delete<Object?>('$_base/course-reviews/$reviewId/$kind');
    }
    return true;
  }
}
