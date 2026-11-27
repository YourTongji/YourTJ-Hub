import 'dart:collection';

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
  CourseRepository(
    this._client, {
    this.cacheTtl = Duration.zero,
    this.maxCacheEntries = 64,
    DateTime Function()? now,
  }) : _now = now ?? DateTime.now;

  final GfApiClient _client;
  final Duration cacheTtl;
  final int maxCacheEntries;
  final DateTime Function() _now;

  static const _base = '/api/forum';

  final LinkedHashMap<int, ({CourseDetailPayload data, DateTime fetchedAt})>
  _detailCache =
      LinkedHashMap<int, ({CourseDetailPayload data, DateTime fetchedAt})>();
  final LinkedHashMap<int, ({CourseRelatedResult data, DateTime fetchedAt})>
  _relatedCache =
      LinkedHashMap<int, ({CourseRelatedResult data, DateTime fetchedAt})>();

  void clearCache() {
    _detailCache.clear();
    _relatedCache.clear();
  }

  T? _readFresh<T>(
    LinkedHashMap<int, ({T data, DateTime fetchedAt})> cache,
    int key,
  ) {
    if (cacheTtl <= Duration.zero) return null;
    final entry = cache.remove(key);
    if (entry == null) return null;
    if (_now().difference(entry.fetchedAt) >= cacheTtl) return null;
    cache[key] = entry;
    return entry.data;
  }

  void _store<T>(
    LinkedHashMap<int, ({T data, DateTime fetchedAt})> cache,
    int key,
    T data,
  ) {
    if (cacheTtl <= Duration.zero) return;
    cache.remove(key);
    if (cache.length >= maxCacheEntries) {
      cache.remove(cache.keys.first);
    }
    cache[key] = (data: data, fetchedAt: _now());
  }

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
  Future<CourseDetailPayload> detail(int courseId) async {
    final cached = _readFresh(_detailCache, courseId);
    if (cached != null) return cached;
    final result = await _client.get<CourseDetailPayload>(
      '$_base/courses/$courseId',
      parser: (json) =>
          CourseDetailPayload.fromJson(Map<String, dynamic>.from(json as Map)),
    );
    _store(_detailCache, courseId, result);
    return result;
  }

  /// 相关课程 + 沿革关系（教师他课/同课他师/lineage）。
  Future<CourseRelatedResult> related(int courseId) async {
    final cached = _readFresh(_relatedCache, courseId);
    if (cached != null) return cached;
    final result = await _client.get<CourseRelatedResult>(
      '$_base/courses/$courseId/related',
      parser: (json) =>
          CourseRelatedResult.fromJson(Map<String, dynamic>.from(json as Map)),
    );
    _store(_relatedCache, courseId, result);
    return result;
  }

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
    _detailCache.remove(courseId);
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
  Future<ReviewPayload> createReview(CreateCourseReviewInput input) async {
    final result = await _client.post<ReviewPayload>(
      '$_base/course-reviews',
      body: input.toJson(),
      parser: (json) =>
          ReviewPayload.fromJson(Map<String, dynamic>.from(json as Map)),
    );
    _detailCache.clear();
    return result;
  }

  /// 改课评（部分更新语义，后端只注册 PATCH；见 route4api.go）。
  Future<ReviewPayload> updateReview(
    int reviewId,
    UpdateCourseReviewInput input,
  ) async {
    final result = await _client.patch<ReviewPayload>(
      '$_base/course-reviews/$reviewId',
      body: input.toJson(),
      parser: (json) =>
          ReviewPayload.fromJson(Map<String, dynamic>.from(json as Map)),
    );
    _detailCache.clear();
    return result;
  }

  /// 删课评（幂等）。
  Future<bool> deleteReview(int reviewId) async {
    // DELETE 语义经 POST 兼容通道不可用；后端契约走 DELETE 方法。
    await _client.delete<Object?>('$_base/course-reviews/$reviewId');
    _detailCache.clear();
    return true;
  }

  /// 有用/无用标记（幂等切换）。
  Future<bool> markHelpful(int reviewId, {required bool on}) =>
      _reviewAction('helpful', reviewId, on);

  /// 无用标记。
  Future<bool> markDislike(int reviewId, {required bool on}) =>
      _reviewAction('dislike', reviewId, on);

  /// 举报课评（说明按服务端口径 trim，空说明省略；移动端限制 300 rune）。
  Future<bool> reportReview({
    required int reviewId,
    required String reason,
    String note = '',
  }) async {
    final trimmedNote = note.trim();
    await _client.post<Object?>(
      '$_base/course-reviews/$reviewId/reports',
      body: {'reason': reason, if (trimmedNote.isNotEmpty) 'note': trimmedNote},
    );
    return true;
  }

  Future<bool> _reviewAction(String kind, int reviewId, bool on) async {
    if (on) {
      await _client.put<Object?>('$_base/course-reviews/$reviewId/$kind');
    } else {
      await _client.delete<Object?>('$_base/course-reviews/$reviewId/$kind');
    }
    return true;
  }
}
