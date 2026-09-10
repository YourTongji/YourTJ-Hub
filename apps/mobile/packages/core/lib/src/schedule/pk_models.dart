/// 排课器领域模型（对齐 web `resource/src/site/types/pk.ts`）。
///
/// 这里是前端持久化/算法域类型：字段名与 JSON 键同 web localStorage
/// (`pk.plans` 等) 完全一致，保证 schema 可携带。API 线上类型在
/// `../gen/pk.dart`（freezed 镜像），两者按 web 的分层各自独立。
library;

/// 一次上课安排（域形态，字段非空收敛；wire 形态见 gen/pk.dart 的
/// PkArrangementInfo，可空字段在此收敛为默认值）。
class PkArrangement {
  PkArrangement({
    this.arrangementText = '',
    required this.occupyDay,
    required this.occupyTime,
    required this.occupyWeek,
    this.occupyRoom = '',
    this.teacherAndCode = '',
  });

  final String arrangementText;
  final int occupyDay; // 1-7
  final List<int> occupyTime; // 1-12
  final List<int> occupyWeek; // 1-16
  final String occupyRoom;
  final String teacherAndCode; // "教师名(工号)"，多教师逗号分隔

  factory PkArrangement.fromJson(Map<String, dynamic> json) => PkArrangement(
    arrangementText: (json['arrangementText'] as String?) ?? '',
    occupyDay: (json['occupyDay'] as num?)?.toInt() ?? 0,
    occupyTime: intList(json['occupyTime']),
    occupyWeek: intList(json['occupyWeek']),
    occupyRoom: (json['occupyRoom'] as String?) ?? '',
    teacherAndCode: (json['teacherAndCode'] as String?) ?? '',
  );

  Map<String, dynamic> toJson() => {
    'arrangementText': arrangementText,
    'occupyDay': occupyDay,
    'occupyTime': occupyTime,
    'occupyWeek': occupyWeek,
    'occupyRoom': occupyRoom,
    'teacherAndCode': teacherAndCode,
  };
}

class PkTeacher {
  PkTeacher({required this.teacherName, required this.teacherCode});

  final String teacherName;
  final String teacherCode;

  factory PkTeacher.fromJson(Map<String, dynamic> json) => PkTeacher(
    teacherName: (json['teacherName'] as String?) ?? '',
    teacherCode: (json['teacherCode'] as String?) ?? '',
  );

  Map<String, dynamic> toJson() => {
    'teacherName': teacherName,
    'teacherCode': teacherCode,
  };
}

/// 课程详情 = 一个教学班（一门课的一个授课实例）。
class PkCourseDetail {
  PkCourseDetail({
    required this.arrangementInfo,
    this.campus = '',
    required this.code,
    this.teachingClassId,
    this.isExclusive,
    this.status,
    required this.teachers,
    this.teachingLanguage = '',
  });

  final List<PkArrangement> arrangementInfo;
  final String campus; // 多校区用「、」拼接
  final String code; // 教学班课号（含班号后缀，如 "122004.01"）
  final int? teachingClassId; // pk_course_detail.id：P13 直查键
  final bool? isExclusive; // 仅专业课/必修为 true（P5 语义）
  final int? status; // 0 未选 / 1 备选 / 2 已选（前端持久化字段，非 API 契约）
  final List<PkTeacher> teachers;
  final String teachingLanguage;

  factory PkCourseDetail.fromJson(Map<String, dynamic> json) => PkCourseDetail(
    arrangementInfo: (json['arrangementInfo'] as List<dynamic>? ?? const [])
        .map((e) => PkArrangement.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    campus: (json['campus'] as String?) ?? '',
    code: (json['code'] as String?) ?? '',
    teachingClassId: (json['teachingClassId'] as num?)?.toInt(),
    isExclusive: json['isExclusive'] as bool?,
    status: (json['status'] as num?)?.toInt(),
    teachers: (json['teachers'] as List<dynamic>? ?? const [])
        .map((e) => PkTeacher.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    teachingLanguage: (json['teachingLanguage'] as String?) ?? '',
  );

  Map<String, dynamic> toJson() => {
    'arrangementInfo': arrangementInfo.map((e) => e.toJson()).toList(),
    'campus': campus,
    'code': code,
    if (teachingClassId != null) 'teachingClassId': teachingClassId,
    if (isExclusive != null) 'isExclusive': isExclusive,
    if (status != null) 'status': status,
    'teachers': teachers.map((e) => e.toJson()).toList(),
    'teachingLanguage': teachingLanguage,
  };
}

/// 备选 / 已选课程。
class PkStagedCourse {
  PkStagedCourse({
    required this.courseCode,
    required this.courseName,
    this.courseNameReserved = '',
    this.credit = 0,
    this.courseType = '',
    this.courseNature,
    required this.teacher,
    this.status = 0,
    required this.courseDetail,
  });

  final String courseCode; // 基础课号（无班号）
  final String courseName;
  final String courseNameReserved; // 保留课程名（导出用）
  final double credit; // x.0 / x.5
  final String courseType; // '必' | '选' | '跨' | '查'
  final List<String>? courseNature;
  final List<PkTeacher> teacher;
  final int status; // 0 未选 / 1 备选 / 2 已选
  final List<PkCourseDetail> courseDetail;

  factory PkStagedCourse.fromJson(Map<String, dynamic> json) => PkStagedCourse(
    courseCode: (json['courseCode'] as String?) ?? '',
    courseName: (json['courseName'] as String?) ?? '',
    courseNameReserved: (json['courseNameReserved'] as String?) ?? '',
    credit: (json['credit'] as num?)?.toDouble() ?? 0,
    courseType: (json['courseType'] as String?) ?? '',
    courseNature: (json['courseNature'] as List<dynamic>?)
        ?.map((e) => e.toString())
        .toList(),
    teacher: (json['teacher'] as List<dynamic>? ?? const [])
        .map((e) => PkTeacher.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
    status: (json['status'] as num?)?.toInt() ?? 0,
    courseDetail: (json['courseDetail'] as List<dynamic>? ?? const [])
        .map(
          (e) => PkCourseDetail.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );

  Map<String, dynamic> toJson() => {
    'courseCode': courseCode,
    'courseName': courseName,
    'courseNameReserved': courseNameReserved,
    'credit': credit,
    'courseType': courseType,
    if (courseNature != null) 'courseNature': courseNature,
    'teacher': teacher.map((e) => e.toJson()).toList(),
    'status': status,
    'courseDetail': courseDetail.map((e) => e.toJson()).toList(),
  };
}

/// 自定义占位事件：用户手工标注的不可用时段（如「有事」）。
class PkCustomEvent {
  PkCustomEvent({
    required this.id,
    required this.label,
    required this.day,
    required this.sections,
    required this.weeks,
  });

  final String id;
  final String label;
  final int day; // 1-7
  final List<int> sections; // 1-12
  final List<int> weeks;

  factory PkCustomEvent.fromJson(Map<String, dynamic> json) => PkCustomEvent(
    id: (json['id'] as String?) ?? '',
    label: (json['label'] as String?) ?? '',
    day: (json['day'] as num?)?.toInt() ?? 0,
    sections: intList(json['sections']),
    weeks: intList(json['weeks']),
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'label': label,
    'day': day,
    'sections': sections,
    'weeks': weeks,
  };
}

/// 排课方案（v2）：每套方案独立持有已选/备选课程与自定义占位。
class PkPlan {
  PkPlan({
    required this.id,
    required this.name,
    required this.createdAt,
    required this.stagedCourses,
    required this.selectedCourses,
    required this.customEvents,
  });

  final String id;
  final String name;
  final int createdAt; // ms epoch
  final List<PkStagedCourse> stagedCourses;
  final List<String> selectedCourses; // 已选班级课号（含班号）
  final List<PkCustomEvent> customEvents;

  factory PkPlan.fromJson(Map<String, dynamic> json) => PkPlan(
    id: (json['id'] as String?) ?? '',
    name: (json['name'] as String?) ?? '',
    createdAt: (json['createdAt'] as num?)?.toInt() ?? 0,
    stagedCourses: (json['stagedCourses'] as List<dynamic>? ?? const [])
        .map(
          (e) => PkStagedCourse.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
    selectedCourses: (json['selectedCourses'] as List<dynamic>? ?? const [])
        .map((e) => e.toString())
        .toList(),
    customEvents: (json['customEvents'] as List<dynamic>? ?? const [])
        .map((e) => PkCustomEvent.fromJson(Map<String, dynamic>.from(e as Map)))
        .toList(),
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'createdAt': createdAt,
    'stagedCourses': stagedCourses.map((e) => e.toJson()).toList(),
    'selectedCourses': selectedCourses,
    'customEvents': customEvents.map((e) => e.toJson()).toList(),
  };
}

/// 学期→年级→专业 选择三元组。
class PkMajorSelection {
  PkMajorSelection({this.calendarId, this.grade, this.major, this.majorName});

  final int? calendarId;
  final int? grade;
  final String? major;
  final String? majorName;

  factory PkMajorSelection.fromJson(Map<String, dynamic> json) =>
      PkMajorSelection(
        calendarId: (json['calendarId'] as num?)?.toInt(),
        grade: (json['grade'] as num?)?.toInt(),
        major: json['major'] as String?,
        majorName: json['majorName'] as String?,
      );

  Map<String, dynamic> toJson() => {
    if (calendarId != null) 'calendarId': calendarId,
    if (grade != null) 'grade': grade,
    if (major != null) 'major': major,
    if (majorName != null) 'majorName': majorName,
  };
}

/// 周次视图状态：week 为 null 表示「全部周次」堆叠视图。
class PkWeekView {
  PkWeekView({this.week, this.useCurrent = false});

  final int? week;
  final bool useCurrent;

  factory PkWeekView.fromJson(Map<String, dynamic> json) => PkWeekView(
    week: (json['week'] as num?)?.toInt(),
    useCurrent: json['useCurrent'] as bool? ?? false,
  );

  Map<String, dynamic> toJson() => {'week': week, 'useCurrent': useCurrent};
}

/// occupied 12×7 三维数组的单元（一个格子里的一个课程）。
class PkOccupyCell {
  PkOccupyCell({
    required this.code,
    required this.courseName,
    required this.occupyWeek,
  });

  final String code;
  final String courseName;
  final List<int> occupyWeek;
}

/// 课表条目（平铺，渲染时转网格）。v2：结构化字段随重建写入。
class PkCourseOnTable {
  PkCourseOnTable({
    required this.showText,
    required this.courseName,
    required this.code,
    required this.occupyTime,
    required this.occupyDay,
    this.occupyWeek,
    this.teacherAndCode,
    this.arrangementText,
    this.occupyRoom,
  });

  final String showText;
  final String courseName;
  final String code;
  final List<int> occupyTime;
  final int occupyDay;
  final List<int>? occupyWeek; // 单周视图过滤用，容忍缺失
  final String? teacherAndCode;
  final String? arrangementText;
  final String? occupyRoom;
}

List<int> intList(dynamic value) => (value as List<dynamic>? ?? const [])
    .map((e) => (e as num).toInt())
    .toList();
