import '../../gen/pk.dart';
import '../../gen/schedule_settings.dart';
import '../api_error.dart';
import '../gf_api_client.dart';

/// PK 排课器域（14 个匿名公开操作 + section-times + 登录方案云同步
/// GET/PUT/DELETE `/api/pk/plans`，issue #537）。
///
/// 全部走 [GfApiClient.getPk]/[postPk]/[putPk]/[deletePk] 的
/// `{code,msg,data}` 信封分支；公开操作匿名只读（服务端统一
/// RateLimitCourseCatalog），plans 三端点需登录（读写受独立 pk.plans 配额，
/// 401 由客户端统一触发会话失效）。
class PkRepository {
  PkRepository(this._client);

  final GfApiClient _client;

  static const _base = '/api/pk';

  /// P1 学期列表（最近 8 个；startDate/endDate 未配置为 null）。
  Future<List<PkCalendarItem>> calendars() =>
      _client.getPk<List<PkCalendarItem>>(
        '$_base/calendars',
        parser: (json) => (json as List<dynamic>? ?? const [])
            .map(
              (e) =>
                  PkCalendarItem.fromJson(Map<String, dynamic>.from(e as Map)),
            )
            .toList(),
      );

  /// P2 校区字典。
  Future<List<PkCampus>> campuses() => _client.getPk<List<PkCampus>>(
    '$_base/campuses',
    parser: (json) => (json as List<dynamic>? ?? const [])
        .map((e) => PkCampus.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
  );

  /// P2 开课学院字典。
  Future<List<PkFaculty>> faculties() => _client.getPk<List<PkFaculty>>(
    '$_base/faculties',
    parser: (json) => (json as List<dynamic>? ?? const [])
        .map((e) => PkFaculty.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
  );

  /// P3 某学期可选年级（降序）。
  Future<PkGradeList> grades({required int calendarId}) =>
      _client.postPk<PkGradeList>(
        '$_base/grades',
        body: {'calendarId': calendarId},
        parser: (json) =>
            PkGradeList.fromJson(Map<String, dynamic>.from(json as Map)),
      );

  /// P4 年级→专业列表（按 code 排序）。
  Future<List<PkMajor>> majors({required int grade, int? calendarId}) =>
      _client.postPk<List<PkMajor>>(
        '$_base/majors',
        body: {'grade': grade, 'calendarId': ?calendarId},
        parser: (json) => (json as List<dynamic>? ?? const [])
            .map((e) => PkMajor.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
      );

  /// P5 专业计划内课程（含教学班详情）。
  Future<List<PkCourseByMajorItem>> coursesByMajor({
    required int grade,
    required String code,
    required int calendarId,
  }) => _client.postPk<List<PkCourseByMajorItem>>(
    '$_base/courses-by-major',
    body: {'grade': grade, 'code': code, 'calendarId': calendarId},
    parser: (json) => (json as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkCourseByMajorItem.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );

  /// P6 选修课类型。
  Future<List<PkOptionalType>> optionalTypes({required int calendarId}) =>
      _client.postPk<List<PkOptionalType>>(
        '$_base/optional-types',
        body: {'calendarId': calendarId},
        parser: (json) => (json as List<dynamic>? ?? const [])
            .map(
              (e) =>
                  PkOptionalType.fromJson(Map<String, dynamic>.from(e as Map)),
            )
            .toList(),
      );

  /// P7 按选修类型批量取课程。
  Future<List<PkCourseByNatureItem>> coursesByNature({
    required int calendarId,
    required List<int> ids,
  }) => _client.postPk<List<PkCourseByNatureItem>>(
    '$_base/courses-by-nature',
    body: {'calendarId': calendarId, 'ids': ids},
    parser: (json) => (json as List<dynamic>? ?? const [])
        .map(
          (e) => PkCourseByNatureItem.fromJson(
            Map<String, dynamic>.from(e as Map),
          ),
        )
        .toList(),
  );

  /// P8 单课教学班详情。
  Future<List<PkCourseDetailBrief>> courseDetails({
    required int calendarId,
    required String courseCode,
  }) => _client.postPk<List<PkCourseDetailBrief>>(
    '$_base/course-details',
    body: {'calendarId': calendarId, 'courseCode': courseCode},
    parser: (json) => (json as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkCourseDetailBrief.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );

  /// P8 批量教学班详情（{courseCode: [brief]} 字典）。
  Future<Map<String, List<PkCourseDetailBrief>>> courseDetailsBatch({
    required int calendarId,
    required List<String> courseCodes,
  }) => _client.postPk<Map<String, List<PkCourseDetailBrief>>>(
    '$_base/course-details',
    body: {'calendarId': calendarId, 'courseCodes': courseCodes},
    parser: (json) => (json as Map<String, dynamic>? ?? const {}).map(
      (code, list) => MapEntry(
        code,
        (list as List<dynamic>? ?? const [])
            .map(
              (e) => PkCourseDetailBrief.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      ),
    ),
  );

  /// P9 课程搜索（服务端 LIMIT 100）。
  Future<PkSearchResult> searchCourses({
    required int calendarId,
    String? courseName,
    String? courseCode,
    String? teacherCode,
    String? teacherName,
    String? campus,
    String? faculty,
  }) => _client.postPk<PkSearchResult>(
    '$_base/course-search',
    body: {
      'calendarId': calendarId,
      if (courseName != null && courseName.isNotEmpty) 'courseName': courseName,
      if (courseCode != null && courseCode.isNotEmpty) 'courseCode': courseCode,
      if (teacherCode != null && teacherCode.isNotEmpty)
        'teacherCode': teacherCode,
      if (teacherName != null && teacherName.isNotEmpty)
        'teacherName': teacherName,
      if (campus != null && campus.isNotEmpty) 'campus': campus,
      if (faculty != null && faculty.isNotEmpty) 'faculty': faculty,
    },
    parser: (json) =>
        PkSearchResult.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// P10 按天+大节查课程（auxiliaryReady=false 为降级 LIKE，正常响应）。
  Future<PkCoursesByTimeResult> coursesByTime({
    required int calendarId,
    required int day,
    required int section,
  }) => _client.postPk<PkCoursesByTimeResult>(
    '$_base/courses-by-time',
    body: {'calendarId': calendarId, 'day': day, 'section': section},
    parser: (json) =>
        PkCoursesByTimeResult.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// P11 最近一系统同步日期（YYYY-MM-DD 或 null）。
  Future<String?> latestUpdate() => _client.getPk<String?>(
    '$_base/latest-update',
    parser: (json) => json is String? ? json : json.toString(),
  );

  /// P12 「同步最新」：全方案并集的课程码 → 教学班详情字典。
  Future<Map<String, List<PkCourseDetailBrief>>> courseInfoSync({
    required int calendarId,
    List<String> majorCourseCodes = const [],
    List<String> otherCourseCodes = const [],
    (int grade, String code)? majorInfo,
  }) => _client.postPk<Map<String, List<PkCourseDetailBrief>>>(
    '$_base/course-info-sync',
    body: {
      'calendarId': calendarId,
      if (majorCourseCodes.isNotEmpty) 'majorCourseCodes': majorCourseCodes,
      if (otherCourseCodes.isNotEmpty) 'otherCourseCodes': otherCourseCodes,
      if (majorInfo != null)
        'majorInfo': {'grade': majorInfo.$1, 'code': majorInfo.$2},
    },
    parser: (json) => (json as Map<String, dynamic>? ?? const {}).map(
      (code, list) => MapEntry(
        code,
        (list as List<dynamic>? ?? const [])
            .map(
              (e) => PkCourseDetailBrief.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      ),
    ),
  );

  /// P13 课评摘要（teachingClassId 直查优先；未匹配 courseId=0）。
  Future<PkReviewBrief> courseReviewBrief({
    required String courseCode,
    String? teacherName,
    int? calendarId,
    int? teachingClassId,
  }) => _client.getPk<PkReviewBrief>(
    '$_base/course-review-brief',
    queryParameters: {
      'courseCode': courseCode,
      if (teacherName != null && teacherName.isNotEmpty)
        'teacherName': teacherName,
      'calendarId': ?calendarId,
      'teachingClassId': ?teachingClassId,
    },
    parser: (json) =>
        PkReviewBrief.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 节次作息表（服务端未配置返回内置默认表；端点由后端契约新增）。
  Future<SectionTimesPayload?> sectionTimes() async {
    try {
      return await _client.getPk<SectionTimesPayload>(
        '$_base/section-times',
        parser: (json) => SectionTimesPayload.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );
    } on ApiException {
      // 旧后端未部署该端点（404）时回退内置默认表。
      return null;
    }
  }

  /// 云端方案快照读取（登录；data=null 表示云端尚无快照）。
  Future<PkPlansSnapshot?> getPlans() => _client.getPk<PkPlansSnapshot?>(
    '$_base/plans',
    parser: (json) => json == null
        ? null
        : PkPlansSnapshot.fromJson(Map<String, dynamic>.from(json as Map)),
  );

  /// 云端方案快照整体替换（登录）；返回新的服务端同步时钟 updatedAt。
  Future<PkPlansPutResult> putPlans(PkPlanSnapshotPayload payload) =>
      _client.putPk<PkPlansPutResult>(
        '$_base/plans',
        body: payload.toJson(),
        parser: (json) =>
            PkPlansPutResult.fromJson(Map<String, dynamic>.from(json as Map)),
      );

  /// 删除云端快照（登录、幂等；本地数据不动）。
  Future<PkPlansDeleteResult> deletePlans() =>
      _client.deletePk<PkPlansDeleteResult>(
        '$_base/plans',
        parser: (json) => PkPlansDeleteResult.fromJson(
          Map<String, dynamic>.from(json as Map),
        ),
      );
}
