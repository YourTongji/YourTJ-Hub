// PK 方案云同步（issue #537）契约镜像与仓库测试：
// - 快照类：契约 fixture（pk-plans-get-success.json）解析、快照→载荷
//   toJson→领域模型往返、PUT/DELETE 结果容错解析。
// - 仓库：getPlans（云端空 data=null）、putPlans（PUT 四字段整体替换 +
//   回传 updatedAt、PK 业务失败信封 400）、deletePlans（幂等删除）、
//   401 统一抛 UnauthorizedException。
import 'dart:convert';
import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'api_client_test.dart' show MockAdapter, ResponseData;

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'tok-plan-sync';

  @override
  Future<void> write(String token) async {}

  @override
  Future<void> clear() async {}
}

Map<String, dynamic> _loadFixture() =>
    jsonDecode(
          File(
            '../../../../packages/api-contract/fixtures/pk-plans-get-success.json',
          ).readAsStringSync(),
        )
        as Map<String, dynamic>;

void main() {
  group('PkPlansSnapshot / PkPlanSnapshotPayload', () {
    test('契约 fixture 解析（课程/教师/占位/周次视图/同步时钟）', () {
      final PkPlansSnapshot snapshot = PkPlansSnapshot.fromJson(
        _loadFixture()['data'] as Map<String, dynamic>,
      );
      expect(snapshot.plans, hasLength(1));
      final PkPlan plan = snapshot.plans.first;
      expect(plan.id, 'plan_a');
      expect(plan.name, '方案 1');
      expect(plan.createdAt, 1725000000000);
      expect(plan.selectedCourses, <String>['122004.01']);
      final PkStagedCourse course = plan.stagedCourses.single;
      expect(course.courseCode, '122004');
      expect(course.courseName, '数据结构');
      expect(course.credit, 4);
      expect(course.status, 2);
      final PkCourseDetail detail = course.courseDetail.single;
      expect(detail.teachingClassId, 445566);
      expect(detail.isExclusive, isTrue);
      expect(detail.status, 2);
      final PkArrangement arrangement = detail.arrangementInfo.single;
      expect(arrangement.occupyTime, <int>[3, 4]);
      expect(arrangement.occupyWeek, List<int>.generate(8, (int i) => i + 1));
      expect(arrangement.occupyRoom, '同德楼A201');
      expect(arrangement.teacherAndCode, '张伟(T001)');
      final PkCustomEvent event = plan.customEvents.single;
      expect(event.id, 'evt_1');
      expect(event.day, 6);
      expect(event.sections, <int>[1, 2]);
      expect(event.weeks, <int>[1, 3, 5]);
      expect(snapshot.activePlanId, 'plan_a');
      expect(snapshot.majorSelected.calendarId, 121);
      expect(snapshot.majorSelected.grade, 2024);
      expect(snapshot.majorSelected.major, '080601');
      expect(snapshot.weekView.week, 5);
      expect(snapshot.weekView.useCurrent, isTrue);
      expect(snapshot.updatedAt, '2026-09-08T07:00:00Z');
    });

    test('快照 → 载荷 toJson → 领域模型往返一致', () {
      final PkPlansSnapshot snapshot = PkPlansSnapshot.fromJson(
        _loadFixture()['data'] as Map<String, dynamic>,
      );
      final Map<String, dynamic> payload = PkPlanSnapshotPayload(
        plans: snapshot.plans,
        activePlanId: snapshot.activePlanId,
        majorSelected: snapshot.majorSelected,
        weekView: snapshot.weekView,
      ).toJson();
      // 载荷只含四字段整体替换（不含 updatedAt / configCollapsed）。
      expect(
        payload.keys,
        unorderedEquals(<String>[
          'plans',
          'activePlanId',
          'majorSelected',
          'weekView',
        ]),
      );
      expect(payload['activePlanId'], 'plan_a');
      expect(
        (payload['plans'] as List<dynamic>)
            .map(
              (dynamic e) =>
                  PkPlan.fromJson(Map<String, dynamic>.from(e as Map)),
            )
            .single
            .id,
        'plan_a',
      );
      final Map<String, dynamic> major =
          payload['majorSelected'] as Map<String, dynamic>;
      expect(major['calendarId'], 121);
      expect(major['grade'], 2024);
      expect(major['major'], '080601');
      expect(major['majorName'], '计算机科学与技术');
      final Map<String, dynamic> weekView =
          payload['weekView'] as Map<String, dynamic>;
      expect(weekView['week'], 5);
      expect(weekView['useCurrent'], isTrue);
    });

    test('PUT/DELETE 结果容错解析', () {
      expect(
        PkPlansPutResult.fromJson(<String, dynamic>{
          'updatedAt': '2026-09-08T08:30:00.123456789Z',
        }).updatedAt,
        '2026-09-08T08:30:00.123456789Z',
      );
      expect(PkPlansPutResult.fromJson(<String, dynamic>{}).updatedAt, '');
      expect(
        PkPlansDeleteResult.fromJson(<String, dynamic>{
          'deleted': true,
        }).deleted,
        isTrue,
      );
      expect(
        PkPlansDeleteResult.fromJson(<String, dynamic>{}).deleted,
        isFalse,
      );
    });
  });

  group('PkRepository plans 三端点', () {
    late Dio dio;
    late PkRepository repository;

    setUp(() {
      dio = Dio(BaseOptions(baseUrl: 'http://test'));
      repository = PkRepository(GfApiClient(dio: dio, tokenStorage: _Tokens()));
    });

    tearDown(() {
      dio.close();
    });

    test('getPlans：云端空（data=null）→ null', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/pk/plans');
        expect(request.method, 'GET');
        return ResponseData(200, <String, dynamic>{
          'code': 0,
          'msg': '查询成功',
          'data': null,
        });
      });
      expect(await repository.getPlans(), isNull);
    });

    test('getPlans：契约 fixture → 快照（updatedAt 为服务端同步时钟）', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(200, _loadFixture());
      });
      final PkPlansSnapshot? snapshot = await repository.getPlans();
      expect(snapshot, isNotNull);
      expect(snapshot!.updatedAt, '2026-09-08T07:00:00Z');
      expect(snapshot.plans.single.id, 'plan_a');
    });

    test('getPlans：401 → UnauthorizedException（会话失效统一出口）', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(401, <String, dynamic>{'code': 1, 'msg': '未登录'});
      });
      await expectLater(
        repository.getPlans(),
        throwsA(isA<UnauthorizedException>()),
      );
    });

    test('putPlans：PUT /api/pk/plans 四字段整体替换并回传 updatedAt', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/pk/plans');
        expect(request.method, 'PUT');
        // dio 在 adapter 之前已把 Map 序列化为 String。
        final dynamic raw = request.data;
        final Map<String, dynamic> body = raw is String
            ? jsonDecode(raw) as Map<String, dynamic>
            : Map<String, dynamic>.from(raw as Map);
        expect(
          body.keys,
          unorderedEquals(<String>[
            'plans',
            'activePlanId',
            'majorSelected',
            'weekView',
          ]),
        );
        expect((body['plans'] as List<dynamic>), hasLength(1));
        return ResponseData(200, <String, dynamic>{
          'code': 0,
          'msg': '更新成功',
          'data': <String, dynamic>{'updatedAt': '2026-09-08T08:30:00Z'},
        });
      });
      final PkPlansSnapshot snapshot = PkPlansSnapshot.fromJson(
        _loadFixture()['data'] as Map<String, dynamic>,
      );
      final PkPlansPutResult result = await repository.putPlans(
        PkPlanSnapshotPayload(
          plans: snapshot.plans,
          activePlanId: snapshot.activePlanId,
          majorSelected: snapshot.majorSelected,
          weekView: snapshot.weekView,
        ),
      );
      expect(result.updatedAt, '2026-09-08T08:30:00Z');
    });

    test('putPlans：结构校验失败（PK 信封 code=400）→ ApiException 400', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        return ResponseData(400, <String, dynamic>{
          'code': 400,
          'msg': '方案数超过上限',
        });
      });
      await expectLater(
        repository.putPlans(
          PkPlanSnapshotPayload(
            plans: <PkPlan>[],
            activePlanId: '',
            majorSelected: PkMajorSelection(),
            weekView: PkWeekView(),
          ),
        ),
        throwsA(
          isA<ApiException>().having(
            (ApiException e) => e.statusCode,
            'statusCode',
            400,
          ),
        ),
      );
    });

    test('deletePlans：幂等删除 data={deleted:true}', () async {
      dio.httpClientAdapter = MockAdapter((request) async {
        expect(request.path, '/api/pk/plans');
        expect(request.method, 'DELETE');
        return ResponseData(200, <String, dynamic>{
          'code': 0,
          'msg': '删除成功',
          'data': <String, dynamic>{'deleted': true},
        });
      });
      expect((await repository.deletePlans()).deleted, isTrue);
    });
  });
}
