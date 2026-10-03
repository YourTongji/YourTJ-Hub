import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;
import 'campus_repository_test.dart' show Storage;

/// 课评整链路的 method+path 审计：每个请求方法必须与
/// `apps/gooseforum/app/http/routes/route4api.go` 注册的方法一致。
/// 回归背景：updateReview 曾用 POST，而后端只注册了 PATCH，真机编辑课评
/// 直接落到未定义路由（route.notFound）。
void main() {
  late Dio dio;
  late List<RequestOptions> calls;

  setUp(() {
    dio = Dio(BaseOptions(baseUrl: 'http://test'));
    calls = <RequestOptions>[];
    dio.httpClientAdapter = MockAdapter((request) async {
      calls.add(request);
      return ResponseData(200, {'code': 0, 'result': _resultFor(request)});
    });
  });

  CourseRepository repository() => CourseRepository(
    GfApiClient(dio: dio, tokenStorage: Storage()),
  );

  FileRepository files() => FileRepository(
    GfApiClient(dio: dio, tokenStorage: Storage()),
  );

  test('course review chain uses the routes registered in route4api.go', () async {
    final repo = repository();

    await repo.detail(42);
    await repo.related(42);
    await repo.aiSummary(42);
    await repo.list();
    await repo.reviews(42);
    await repo.ownReviews();
    await repo.createReview(
      const CreateCourseReviewInput(
        offeringId: 901,
        rating: 5,
        content: 'good',
        isAnonymous: false,
      ),
    );
    await repo.updateReview(
      7,
      const UpdateCourseReviewInput(content: 'edited', isAnonymous: true),
    );
    await repo.deleteReview(7);
    await repo.markHelpful(7, on: true);
    await repo.markHelpful(7, on: false);
    await repo.markDislike(7, on: true);
    await repo.markDislike(7, on: false);
    await repo.reportReview(reviewId: 7, reason: 'spam');
    await repo.bookmark(courseId: 42, bookmarked: true);
    await files().uploadImage(bytes: const <int>[1, 2, 3], filename: 'a.png');

    expect(
      calls.map((call) => '${call.method} ${call.path}').toList(),
      <String>[
        'GET /api/forum/courses/42',
        'GET /api/forum/courses/42/related',
        'GET /api/forum/courses/42/summary',
        'GET /api/forum/courses',
        'GET /api/forum/courses/42/reviews',
        'GET /api/forum/my-course-reviews',
        'POST /api/forum/course-reviews',
        'PATCH /api/forum/course-reviews/7',
        'DELETE /api/forum/course-reviews/7',
        'PUT /api/forum/course-reviews/7/helpful',
        'DELETE /api/forum/course-reviews/7/helpful',
        'PUT /api/forum/course-reviews/7/dislike',
        'DELETE /api/forum/course-reviews/7/dislike',
        'POST /api/forum/course-reviews/7/reports',
        'POST /api/forum/courses/bookmark',
        'POST /file/img-upload',
      ],
    );
  });

  test('updateReview sends a PATCH body, not a POST', () async {
    await repository().updateReview(
      7,
      const UpdateCourseReviewInput(rating: 4, content: 'edited'),
    );

    expect(calls, hasLength(1));
    expect(calls.single.method, 'PATCH');
    expect(calls.single.path, '/api/forum/course-reviews/7');
    expect(calls.single.data, {
      'rating': 4,
      'content': 'edited',
      'isAnonymous': null,
    });
  });
}

Object? _resultFor(RequestOptions request) {
  final path = request.path;
  if (path.endsWith('/reviews')) {
    return {'list': <Object?>[], 'total': 0};
  }
  if (path.endsWith('/my-course-reviews')) {
    return {'list': <Object?>[]};
  }
  if (path.endsWith('/related')) {
    return {
      'teacherOtherCourses': <Object?>[],
      'sameCourseOtherTeachers': <Object?>[],
    };
  }
  if (path.endsWith('/summary')) {
    return {'status': 'none'};
  }
  if (path.contains('course-reviews')) {
    if (path.endsWith('/reports')) return true;
    return _reviewJson();
  }
  if (path == '/api/forum/courses') {
    return {
      'list': <Object?>[],
      'page': 1,
      'size': 20,
      'total': 0,
      'hasNext': false,
    };
  }
  if (path.startsWith('/api/forum/courses/')) {
    return {
      'id': 42,
      'primaryCode': '100001',
      'name': 'High Math',
      'department': 'Math',
      'creditX10': 50,
    };
  }
  if (path == '/file/img-upload') return '/file/img/a.png';
  return true;
}

Map<String, dynamic> _reviewJson() => jsonDecode(jsonEncode(<String, dynamic>{
  'id': 7,
  'offeringId': 901,
  'rating': 5,
  'content': 'good',
  'contentHtml': '<p>good</p>',
  'author': {'kind': 'member', 'label': 'alice'},
  'viewer': {
    'canEdit': true,
    'canDelete': true,
    'isHelpful': false,
    'isDisliked': false,
  },
  'helpfulCount': 0,
  'dislikeCount': 0,
  'createdAt': '2026-09-30T00:00:00Z',
  'updatedAt': '2026-09-30T00:00:00Z',
})) as Map<String, dynamic>;
