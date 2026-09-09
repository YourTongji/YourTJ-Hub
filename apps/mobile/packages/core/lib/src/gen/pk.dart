/// PK 排课器域契约镜像（对应 packages/api-contract 的 pk 域：14 个匿名
/// 公开操作 + 登录方案云同步 GET/PUT/DELETE /api/pk/plans，issue #537）。
///
/// 手写维护（wiki.dart 同风格），字段名与 `pk.yaml` / `components/schemas.yaml`
/// 的 Pk* 块逐一对应；统一 `{code,msg,data}` 信封由 `GfApiClient.getPk/postPk`
/// 解包后交本文件模型解析。所有列表字段后端保证非 null（contract required），
/// 解析侧仍做容错默认值。
library;

import '../schedule/pk_models.dart';

/// P1 GET /api/pk/calendars：学期条目（startDate/endDate 未配置时为 null）。
class PkCalendarItem {
  const PkCalendarItem({
    required this.calendarId,
    required this.calendarName,
    this.startDate,
    this.endDate,
  });

  final int calendarId;
  final String calendarName;
  final String? startDate; // YYYY-MM-DD 或 null
  final String? endDate;

  factory PkCalendarItem.fromJson(Map<String, dynamic> json) => PkCalendarItem(
    calendarId: (json['calendarId'] as num?)?.toInt() ?? 0,
    calendarName: json['calendarName'] as String? ?? '',
    startDate: json['startDate'] as String?,
    endDate: json['endDate'] as String?,
  );
}

/// P2 GET /api/pk/campuses：校区字典项。
class PkCampus {
  const PkCampus({required this.campusId, required this.campusName});

  final String campusId;
  final String campusName;

  factory PkCampus.fromJson(Map<String, dynamic> json) => PkCampus(
    campusId: json['campusId'] as String? ?? '',
    campusName: json['campusName'] as String? ?? '',
  );
}

/// P2 GET /api/pk/faculties：开课学院字典项。
class PkFaculty {
  const PkFaculty({required this.facultyId, required this.facultyName});

  final String facultyId;
  final String facultyName;

  factory PkFaculty.fromJson(Map<String, dynamic> json) => PkFaculty(
    facultyId: json['facultyId'] as String? ?? '',
    facultyName: json['facultyName'] as String? ?? '',
  );
}

/// P3 POST /api/pk/grades 响应。
class PkGradeList {
  const PkGradeList({required this.gradeList});

  final List<int> gradeList;

  factory PkGradeList.fromJson(Map<String, dynamic> json) => PkGradeList(
    gradeList: (json['gradeList'] as List<dynamic>? ?? const [])
        .map((e) => (e as num).toInt())
        .toList(),
  );
}

/// P4 POST /api/pk/majors 条目。
class PkMajor {
  const PkMajor({required this.code, required this.name});

  final String code;
  final String name;

  factory PkMajor.fromJson(Map<String, dynamic> json) => PkMajor(
    code: json['code'] as String? ?? '',
    name: json['name'] as String? ?? '',
  );
}

/// 教师（PkTeacherRef）。
class PkTeacherRef {
  const PkTeacherRef({required this.teacherCode, required this.teacherName});

  final String teacherCode;
  final String teacherName;

  factory PkTeacherRef.fromJson(Map<String, dynamic> json) => PkTeacherRef(
    teacherCode: json['teacherCode'] as String? ?? '',
    teacherName: json['teacherName'] as String? ?? '',
  );
}

/// 上课安排（PkArrangementInfo；可空字段为后端文本解析降级形态）。
class PkArrangementInfo {
  const PkArrangementInfo({
    required this.arrangementText,
    this.occupyDay,
    this.occupyTime,
    this.occupyWeek,
    this.occupyRoom,
    this.teacherAndCode,
  });

  final String arrangementText;
  final int? occupyDay; // 1-7
  final List<int>? occupyTime; // 节次集合
  final List<int>? occupyWeek; // 周次集合
  final String? occupyRoom;
  final String? teacherAndCode; // "张伟(T001)"

  factory PkArrangementInfo.fromJson(Map<String, dynamic> json) =>
      PkArrangementInfo(
        arrangementText: json['arrangementText'] as String? ?? '',
        occupyDay: (json['occupyDay'] as num?)?.toInt(),
        occupyTime: (json['occupyTime'] as List<dynamic>?)
            ?.map((e) => (e as num).toInt())
            .toList(),
        occupyWeek: (json['occupyWeek'] as List<dynamic>?)
            ?.map((e) => (e as num).toInt())
            .toList(),
        occupyRoom: json['occupyRoom'] as String?,
        teacherAndCode: json['teacherAndCode'] as String?,
      );
}

/// 教学班条目（P5 courses-by-major 的 courses 元素，= PkCourseClassItem）。
class PkCourseClassItem {
  const PkCourseClassItem({
    required this.code,
    required this.teachers,
    required this.campus,
    required this.teachingLanguage,
    required this.arrangementInfo,
    required this.isExclusive,
  });

  final String code;
  final List<PkTeacherRef> teachers;
  final String campus;
  final String teachingLanguage;
  final List<PkArrangementInfo> arrangementInfo;
  final bool isExclusive;

  factory PkCourseClassItem.fromJson(
    Map<String, dynamic> json,
  ) => PkCourseClassItem(
    code: json['code'] as String? ?? '',
    teachers: (json['teachers'] as List<dynamic>? ?? const [])
        .map((e) => PkTeacherRef.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    campus: json['campus'] as String? ?? '',
    teachingLanguage: json['teachingLanguage'] as String? ?? '',
    arrangementInfo: (json['arrangementInfo'] as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkArrangementInfo.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
    isExclusive: json['isExclusive'] as bool? ?? false,
  );
}

/// P5 POST /api/pk/courses-by-major 条目（专业计划内课程）。
class PkCourseByMajorItem {
  const PkCourseByMajorItem({
    required this.courseCode,
    required this.courseName,
    required this.faculty,
    required this.facultyI18n,
    required this.credit,
    required this.grade,
    required this.courseNature,
    required this.courses,
  });

  final String courseCode;
  final String courseName;
  final String faculty;
  final String facultyI18n;
  final double credit;
  final int grade;
  final List<String> courseNature;
  final List<PkCourseClassItem> courses;

  factory PkCourseByMajorItem.fromJson(Map<String, dynamic> json) =>
      PkCourseByMajorItem(
        courseCode: json['courseCode'] as String? ?? '',
        courseName: json['courseName'] as String? ?? '',
        faculty: json['faculty'] as String? ?? '',
        facultyI18n: json['facultyI18n'] as String? ?? '',
        credit: (json['credit'] as num?)?.toDouble() ?? 0,
        grade: (json['grade'] as num?)?.toInt() ?? 0,
        courseNature: (json['courseNature'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        courses: (json['courses'] as List<dynamic>? ?? const [])
            .map(
              (e) => PkCourseClassItem.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      );
}

/// P6 POST /api/pk/optional-types 条目。
class PkOptionalType {
  const PkOptionalType({
    required this.courseLabelId,
    required this.courseLabelName,
  });

  final int courseLabelId;
  final String courseLabelName;

  factory PkOptionalType.fromJson(Map<String, dynamic> json) => PkOptionalType(
    courseLabelId: (json['courseLabelId'] as num?)?.toInt() ?? 0,
    courseLabelName: json['courseLabelName'] as String? ?? '',
  );
}

/// P7 courses-by-nature 的课程条目（PkNatureCourseItem）。
class PkNatureCourseItem {
  const PkNatureCourseItem({
    required this.campus,
    required this.courseCode,
    required this.courseName,
    required this.faculty,
    required this.facultyI18n,
    required this.credit,
    required this.courseLabelName,
    required this.crossDiscipline,
  });

  final List<String> campus;
  final String courseCode;
  final String courseName;
  final String faculty;
  final String facultyI18n;
  final double credit;
  final String courseLabelName;
  final bool crossDiscipline;

  factory PkNatureCourseItem.fromJson(Map<String, dynamic> json) =>
      PkNatureCourseItem(
        campus: (json['campus'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        courseCode: json['courseCode'] as String? ?? '',
        courseName: json['courseName'] as String? ?? '',
        faculty: json['faculty'] as String? ?? '',
        facultyI18n: json['facultyI18n'] as String? ?? '',
        credit: (json['credit'] as num?)?.toDouble() ?? 0,
        courseLabelName: json['courseLabelName'] as String? ?? '',
        crossDiscipline: json['crossDiscipline'] as bool? ?? false,
      );
}

/// P7 POST /api/pk/courses-by-nature 条目（选修课类型分组）。
class PkCourseByNatureItem {
  const PkCourseByNatureItem({
    required this.courseLabelId,
    required this.courseLabelIds,
    required this.courseLabelName,
    required this.crossDiscipline,
    required this.courses,
  });

  final int courseLabelId;
  final List<int> courseLabelIds;
  final String courseLabelName;
  final bool crossDiscipline;
  final List<PkNatureCourseItem> courses;

  factory PkCourseByNatureItem.fromJson(Map<String, dynamic> json) =>
      PkCourseByNatureItem(
        courseLabelId: (json['courseLabelId'] as num?)?.toInt() ?? 0,
        courseLabelIds: (json['courseLabelIds'] as List<dynamic>? ?? const [])
            .map((e) => (e as num).toInt())
            .toList(),
        courseLabelName: json['courseLabelName'] as String? ?? '',
        crossDiscipline: json['crossDiscipline'] as bool? ?? false,
        courses: (json['courses'] as List<dynamic>? ?? const [])
            .map(
              (e) => PkNatureCourseItem.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      );
}

/// P8/P12 教学班详情简表（PkCourseDetailBrief）。
class PkCourseDetailBrief {
  const PkCourseDetailBrief({
    required this.code,
    required this.teachers,
    required this.campus,
    required this.teachingLanguage,
    required this.arrangementInfo,
    this.teachingClassId,
    this.isExclusive,
  });

  final String code;
  final List<PkTeacherRef> teachers;
  final String campus;
  final String teachingLanguage;
  final List<PkArrangementInfo> arrangementInfo;

  /// 一系统教学班 id（pk_course_detail.id），P13 直查键。
  final int? teachingClassId;

  /// 仅 course-info-sync 的 major 课程带该字段。
  final bool? isExclusive;

  factory PkCourseDetailBrief.fromJson(
    Map<String, dynamic> json,
  ) => PkCourseDetailBrief(
    code: json['code'] as String? ?? '',
    teachers: (json['teachers'] as List<dynamic>? ?? const [])
        .map((e) => PkTeacherRef.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    campus: json['campus'] as String? ?? '',
    teachingLanguage: json['teachingLanguage'] as String? ?? '',
    arrangementInfo: (json['arrangementInfo'] as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkArrangementInfo.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
    teachingClassId: (json['teachingClassId'] as num?)?.toInt(),
    isExclusive: json['isExclusive'] as bool?,
  );
}

/// P9 搜索结果课程条目（PkSearchCourseItem；campus_list 为兼容冗余字段）。
class PkSearchCourseItem {
  const PkSearchCourseItem({
    required this.courseCode,
    required this.courseName,
    required this.faculty,
    required this.facultyI18n,
    required this.courseNature,
    required this.campus,
    required this.campusList,
    required this.credit,
  });

  final String courseCode;
  final String courseName;
  final String faculty;
  final String facultyI18n;
  final List<String> courseNature;
  final List<String> campus;
  final List<String> campusList; // campus 兼容重复字段
  final double credit;

  factory PkSearchCourseItem.fromJson(Map<String, dynamic> json) =>
      PkSearchCourseItem(
        courseCode: json['courseCode'] as String? ?? '',
        courseName: json['courseName'] as String? ?? '',
        faculty: json['faculty'] as String? ?? '',
        facultyI18n: json['facultyI18n'] as String? ?? '',
        courseNature: (json['courseNature'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        campus: (json['campus'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        campusList: (json['campus_list'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        credit: (json['credit'] as num?)?.toDouble() ?? 0,
      );
}

/// P9 POST /api/pk/course-search 响应（LIMIT 100）。
class PkSearchResult {
  const PkSearchResult({required this.courses, required this.sizeLimit});

  final List<PkSearchCourseItem> courses;
  final int sizeLimit;

  factory PkSearchResult.fromJson(Map<String, dynamic> json) => PkSearchResult(
    courses: (json['courses'] as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkSearchCourseItem.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
    sizeLimit: (json['sizeLimit'] as num?)?.toInt() ?? 0,
  );
}

/// P10 POST /api/pk/courses-by-time 响应（auxiliaryReady=false 为降级 LIKE，正常 200）。
class PkCoursesByTimeResult {
  const PkCoursesByTimeResult({
    required this.auxiliaryReady,
    required this.courses,
  });

  final bool auxiliaryReady;
  final List<PkSearchCourseItem> courses;

  factory PkCoursesByTimeResult.fromJson(Map<String, dynamic> json) =>
      PkCoursesByTimeResult(
        auxiliaryReady: json['auxiliaryReady'] as bool? ?? false,
        courses: (json['courses'] as List<dynamic>? ?? const [])
            .map(
              (e) => PkSearchCourseItem.fromJson(
                Map<String, dynamic>.from(e as Map),
              ),
            )
            .toList(),
      );
}

/// P13 教学班级课评摘要项（PkReviewBriefClass，按 offering 聚合）。
class PkReviewBriefClass {
  const PkReviewBriefClass({
    required this.classCode,
    required this.offeringId,
    required this.teachers,
    this.ratingAvg,
    required this.reviewCount,
  });

  final String classCode;
  final int offeringId; // 供 /courses/:courseId?offeringId=:offeringId 聚焦
  final List<String> teachers;
  final double? ratingAvg;
  final int reviewCount;

  factory PkReviewBriefClass.fromJson(Map<String, dynamic> json) =>
      PkReviewBriefClass(
        classCode: json['classCode'] as String? ?? '',
        offeringId: (json['offeringId'] as num?)?.toInt() ?? 0,
        teachers: (json['teachers'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
        reviewCount: (json['reviewCount'] as num?)?.toInt() ?? 0,
      );
}

/// P13 GET /api/pk/course-review-brief 响应（未匹配时 courseId=0）。
class PkReviewBrief {
  const PkReviewBrief({
    required this.courseId,
    required this.courseCode,
    required this.courseName,
    required this.teacherName,
    this.ratingAvg,
    required this.reviewCount,
    this.ratingDistribution,
    required this.classes,
  });

  /// Hub 课程目录主键（/courses/:courseId 跳转用）；未匹配课评目录时为 0。
  final int courseId;
  final String courseCode;
  final String courseName;
  final String teacherName;
  final double? ratingAvg;
  final int reviewCount;

  /// 1-5 星各档计数（index 0 = 1 星）。
  final List<int>? ratingDistribution;
  final List<PkReviewBriefClass> classes;

  factory PkReviewBrief.fromJson(Map<String, dynamic> json) => PkReviewBrief(
    courseId: (json['courseId'] as num?)?.toInt() ?? 0,
    courseCode: json['courseCode'] as String? ?? '',
    courseName: json['courseName'] as String? ?? '',
    teacherName: json['teacherName'] as String? ?? '',
    ratingAvg: (json['ratingAvg'] as num?)?.toDouble(),
    reviewCount: (json['reviewCount'] as num?)?.toInt() ?? 0,
    ratingDistribution: (json['ratingDistribution'] as List<dynamic>?)
        ?.map((e) => (e as num).toInt())
        .toList(),
    classes: (json['classes'] as List<dynamic>? ?? const [])
        .map(
          (e) =>
              PkReviewBriefClass.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );
}

/// GET /api/pk/plans 的 data：云端方案快照（issue #537；云端无数据时整个
/// data 为 null，不出现本类）。plans 条目与本地持久化 schema 完全同构
/// （Route A 已与 web 逐字段对齐），因此直接复用 pk_models 领域模型解析，
/// 深度消毒仍由客户端加载路径（store.applyRemoteSnapshot）负责。
class PkPlansSnapshot {
  PkPlansSnapshot({
    required this.plans,
    required this.activePlanId,
    required this.majorSelected,
    required this.weekView,
    required this.updatedAt,
  });

  final List<PkPlan> plans;
  final String activePlanId;
  final PkMajorSelection majorSelected;
  final PkWeekView weekView;

  /// 服务端权威同步时钟（RFC3339Nano UTC）；客户端落为 pk.syncedAt。
  final String updatedAt;

  factory PkPlansSnapshot.fromJson(Map<String, dynamic> json) =>
      PkPlansSnapshot(
        plans: (json['plans'] as List<dynamic>? ?? const [])
            .map((e) => PkPlan.fromJson(Map<String, dynamic>.from(e as Map)))
            .toList(),
        activePlanId: json['activePlanId'] as String? ?? '',
        majorSelected: json['majorSelected'] is Map
            ? PkMajorSelection.fromJson(
                Map<String, dynamic>.from(json['majorSelected'] as Map),
              )
            : PkMajorSelection(),
        weekView: json['weekView'] is Map
            ? PkWeekView.fromJson(
                Map<String, dynamic>.from(json['weekView'] as Map),
              )
            : PkWeekView(),
        updatedAt: json['updatedAt'] as String? ?? '',
      );
}

/// PUT /api/pk/plans 请求体：快照四字段整体替换（服务端仅浅校验方案数
/// 1..10、id/name 非空、activePlanId 引用、载荷 ≤1MB）。
class PkPlanSnapshotPayload {
  PkPlanSnapshotPayload({
    required this.plans,
    required this.activePlanId,
    required this.majorSelected,
    required this.weekView,
    this.baseUpdatedAt,
  });

  /// Observed server revision; empty means create only when absent.
  final String? baseUpdatedAt;
  final List<PkPlan> plans;
  final String activePlanId;
  final PkMajorSelection majorSelected;
  final PkWeekView weekView;

  Map<String, dynamic> toJson() => <String, dynamic>{
    if (baseUpdatedAt != null) 'baseUpdatedAt': baseUpdatedAt,
    'plans': plans.map((plan) => plan.toJson()).toList(),
    'activePlanId': activePlanId,
    'majorSelected': majorSelected.toJson(),
    'weekView': weekView.toJson(),
  };
}

/// PUT /api/pk/plans 的 data：新的服务端同步时钟（客户端更新 pk.syncedAt）。
class PkPlansPutResult {
  const PkPlansPutResult({required this.updatedAt});

  final String updatedAt;

  factory PkPlansPutResult.fromJson(Map<String, dynamic> json) =>
      PkPlansPutResult(updatedAt: json['updatedAt'] as String? ?? '');
}

/// DELETE /api/pk/plans 的 data（幂等删除云端副本；本地数据不动）。
class PkPlansDeleteResult {
  const PkPlansDeleteResult({required this.deleted});

  final bool deleted;

  factory PkPlansDeleteResult.fromJson(Map<String, dynamic> json) =>
      PkPlansDeleteResult(deleted: json['deleted'] as bool? ?? false);
}
