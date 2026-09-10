// 排课器（PK 课表）状态：v2 多方案 + 容忍式冲突（web useScheduleStore.ts 的
// 移动端移植）。持久化键与 web 完全一致（pk.plans / pk.activePlanId /
// pk.majorSelected / pk.weekView / pk.updateTime / goose:scheduleConfigCollapsed；
// 另有 pk.syncedAt 方案云同步时钟，issue #537），
// schema 与 web localStorage 可互相携带。派生数据（occupied / 网格 / 冲突 /
// 统计）一律不持久化，每次变更由纯函数重建。
//
// 冲突语义（容忍式，web 同款）：加课总是成功，冲突在入表后派生标注
// （课表 ⚠ / 统计计数），不再弹窗阻断。同基础课号换班 = 隐式替换，不报冲突。
// 移动端没有 web 的「保存课表」步骤：选班即 status=2 并立刻上表，刷新不丢。
import 'dart:async';
import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 配置区折叠记忆键（web CONFIG_COLLAPSED_STORAGE_KEY，原始 '1'/'0'）。
const String kScheduleConfigCollapsedKey = 'goose:scheduleConfigCollapsed';

/// 持久化键（web STORAGE_KEYS 同键）。
abstract final class ScheduleStorageKeys {
  static const String plans = 'pk.plans';
  static const String activePlanId = 'pk.activePlanId';
  static const String majorSelected = 'pk.majorSelected';
  static const String weekView = 'pk.weekView';
  static const String updateTime = 'pk.updateTime';

  /// 云端方案快照同步时钟（服务端 updatedAt 原文；issue #537）。
  static const String syncedAt = 'pk.syncedAt';
  static const String configCollapsed = kScheduleConfigCollapsedKey;
}

/// 课程班级状态（web COURSE_STATUS 同值；status 为前端持久化字段）。
abstract final class CourseStatus {
  static const int unselected = 0;
  static const int staged = 1;
  static const int selected = 2;

  static bool arranged(int status) => status == staged || status == selected;
}

/// 方案数量上限（web MAX_PLANS）。
const int kMaxPlans = 10;

/// 本地方案数据变更回调（模块级注入：schedule_sync 控制器在创建时绑定，
/// store 因此不直接依赖网络层，测试亦可替换）。
/// 仅本地写入路径触发；云端整包采用（applyingRemote 守卫）绝不触发。
void Function()? scheduleLocalPlansChanged;

/// 课程卡片配色槽位数（web courseColorSlots 8 槽；种子 hash 取模）。
const int kCourseColorSlotCount = 8;

/// 依据课程稳定标识（课号/课名）hash 出 1-based 配色槽位，同课同色。
int courseColorSlotFor(String seed) {
  var h = 0;
  for (var i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.codeUnitAt(i)) & 0xFFFFFFFF;
  }
  return (h % kCourseColorSlotCount) + 1;
}

/// 课表网格派生数据（不持久化，[ScheduleStoreNotifier] 变更时重建）。
class ScheduleGridData {
  const ScheduleGridData({
    required this.cellCourses,
    required this.cellSpans,
    required this.occupiedGrid,
    required this.rowHeights,
    required this.conflicts,
  });

  /// cellCourses[row][day]：该格课程（同节次簇合并后；被上方 rowspan 覆盖的
  /// 槽位为空数组）。
  final List<List<List<PkCourseOnTable>>> cellCourses;

  /// cellSpans[row][day]：该格纵向跨越行数（默认 1）。
  final List<List<int>> cellSpans;

  /// occupiedGrid[row][day]：该格是否已被上方跨行格占用。
  final List<List<bool>> occupiedGrid;

  /// 每行渲染高度（px；kInteractiveRowMetricsMobile 计算）。
  final List<int> rowHeights;

  /// deriveConflicts 全量冲突（key 为基础课号或 custom: 伪课号）。
  final Map<String, List<PkConflictItem>> conflicts;
}

/// 当前方案统计（web stats() 同口径）。
class ScheduleStats {
  const ScheduleStats({
    required this.courseCount,
    required this.totalCredit,
    required this.totalHours,
    required this.conflictCount,
  });

  final int courseCount;
  final double totalCredit;
  final int totalHours;
  final int conflictCount;
}

/// 加课结果（容忍式：added 恒 true；conflicts 为入表前与已占课程的冲突，
/// 同基础课号换班不计）。
class StageCourseResult {
  const StageCourseResult({required this.added, required this.conflicts});

  final bool added;
  final List<PkConflictItem> conflicts;
}

/// 排课器整体状态。
class ScheduleState {
  const ScheduleState({
    required this.majorSelected,
    required this.plans,
    required this.activePlanId,
    required this.weekView,
    required this.isConfigCollapsed,
    required this.updateTime,
    required this.latestUpdateTime,
    required this.occupied,
    required this.grid,
    required this.stats,
  });

  final PkMajorSelection majorSelected;
  final List<PkPlan> plans;
  final String activePlanId;
  final PkWeekView weekView;
  final bool isConfigCollapsed;

  /// 最近一次成功同步到本地数据的日期（YYYY-MM-DD；持久化 pk.updateTime）。
  final String updateTime;

  /// P11 查询到的服务器最新同步日期（会话内，不持久化）。
  final String latestUpdateTime;

  /// occupied 12×7 占用表（含 custom: 伪课号；冲突派生/周次判断同源）。
  final List<List<List<PkOccupyCell>>> occupied;

  /// 课表网格（按 weekView 过滤后重建）。
  final ScheduleGridData grid;

  final ScheduleStats stats;

  ScheduleState copyWith({
    PkMajorSelection? majorSelected,
    List<PkPlan>? plans,
    String? activePlanId,
    PkWeekView? weekView,
    bool? isConfigCollapsed,
    String? updateTime,
    String? latestUpdateTime,
    List<List<List<PkOccupyCell>>>? occupied,
    ScheduleGridData? grid,
    ScheduleStats? stats,
  }) {
    return ScheduleState(
      majorSelected: majorSelected ?? this.majorSelected,
      plans: plans ?? this.plans,
      activePlanId: activePlanId ?? this.activePlanId,
      weekView: weekView ?? this.weekView,
      isConfigCollapsed: isConfigCollapsed ?? this.isConfigCollapsed,
      updateTime: updateTime ?? this.updateTime,
      latestUpdateTime: latestUpdateTime ?? this.latestUpdateTime,
      occupied: occupied ?? this.occupied,
      grid: grid ?? this.grid,
      stats: stats ?? this.stats,
    );
  }
}

/// 默认方案名回退（l10n 由页面注入前使用；与 zh ARB schedulePlanN 一致）。
String _fallbackPlanName(int index) => '方案 $index';

/// 排课器 Notifier。
///
/// 使用方先调用 [initialize]（绑定方案名文案并异步从 SharedPreferences
/// 恢复），再 await [ready]；测试可在构造时注入 [planNameOf] 后直接 await
/// [ready]。持久化写入排入内部队列，[flush] 等待全部落盘（测试确定性用）。
class ScheduleStoreNotifier extends StateNotifier<ScheduleState> {
  ScheduleStoreNotifier({String Function(int index)? planNameOf})
    : _planNameOf = planNameOf ?? _fallbackPlanName,
      super(
        ScheduleState(
          majorSelected: PkMajorSelection(),
          plans: <PkPlan>[],
          activePlanId: '',
          weekView: PkWeekView(),
          isConfigCollapsed: false,
          updateTime: '',
          latestUpdateTime: '',
          occupied: createEmptyOccupied(),
          grid: _emptyGrid(),
          stats: const ScheduleStats(
            courseCount: 0,
            totalCredit: 0,
            totalHours: 0,
            conflictCount: 0,
          ),
        ),
      ) {
    _initial = _initialize();
  }

  String Function(int index) _planNameOf;
  SharedPreferences? _prefs;
  late final Future<void> _initial;
  Future<void> _writeQueue = Future<void>.value();
  int _planSeq = 0;
  int _eventSeq = 0;
  bool _applyingRemote = false;
  String _syncedAt = '';
  bool _syncDirty = false;
  int _syncSeq = 0;
  int _nowMs() => DateTime.now().millisecondsSinceEpoch;

  /// 初始化完成（恢复持久化状态后 resolve）。
  Future<void> get ready => _initial;

  /// 等待全部持久化写入完成（测试确定性）。
  Future<void> get flush => _writeQueue;

  /// 更换方案默认名文案生成器（页面在首次渲染前绑定 l10n）。
  void setPlanNaming(String Function(int index) planNameOf) {
    _planNameOf = planNameOf;
  }

  Future<void> _initialize() async {
    try {
      _prefs = await SharedPreferences.getInstance();
    } catch (_) {
      _prefs = null; // 存储不可用：保持内存态，写入静默跳过。
    }
    _loadFrom(_prefs);
    final String? syncedRaw = _prefs?.getString(ScheduleStorageKeys.syncedAt);
    if (syncedRaw != null && syncedRaw.isNotEmpty) _syncedAt = syncedRaw;
    _syncDirty = _prefs?.getString('pk.syncDirty') == '1';
    _rebuild();
  }

  // ---- 持久化 ----

  void _persist(String key, String value) {
    final SharedPreferences? prefs = _prefs;
    if (prefs == null) return;
    _writeQueue = _writeQueue.then((_) async {
      try {
        await prefs.setString(key, value);
      } catch (_) {
        // localStorage 不可用时的静默降级（web 同款）。
      }
    });
  }

  void _persistPlanData() {
    _persist(
      ScheduleStorageKeys.plans,
      jsonEncode(state.plans.map((p) => p.toJson()).toList()),
    );
    _persist(ScheduleStorageKeys.activePlanId, jsonEncode(state.activePlanId));
    // 云端同步钩子：本地变更（非整包采用路径）→ dirty + 防抖上行。
    if (!_applyingRemote) {
      scheduleLocalPlansChanged?.call();
    }
  }

  // ---- 加载 / 消毒 ----

  /// 从 SharedPreferences 恢复；损坏数据按 web sanitize* 语义丢弃/回退。
  /// v1 旧键迁移不做：本应用无排课器遗留数据。
  void _loadFrom(SharedPreferences? prefs) {
    PkMajorSelection majorSelected = PkMajorSelection();
    final String? majorRaw = prefs?.getString(
      ScheduleStorageKeys.majorSelected,
    );
    if (majorRaw != null) {
      final Object? decoded = _tryJsonDecode(majorRaw);
      if (decoded is Map<String, dynamic>) {
        majorSelected = _sanitizeMajorSelection(decoded);
      }
    }

    List<PkPlan> plans = <PkPlan>[];
    final String? plansRaw = prefs?.getString(ScheduleStorageKeys.plans);
    if (plansRaw != null) {
      final Object? decoded = _tryJsonDecode(plansRaw);
      if (decoded is List<dynamic>) {
        plans = decoded
            .map(_sanitizePlan)
            .where((plan) => plan != null)
            .cast<PkPlan>()
            .toList();
      }
    }
    if (plans.isEmpty) {
      plans = <PkPlan>[_createEmptyPlan(_planNameOf(1))];
    }
    _planSeq = plans.fold<int>(0, (max, plan) {
      final int index = _defaultNameIndexOf(plan.name);
      return index > max ? index : max;
    });

    String activePlanId = '';
    final String? activeRaw = prefs?.getString(
      ScheduleStorageKeys.activePlanId,
    );
    if (activeRaw != null) {
      final Object? decoded = _tryJsonDecode(activeRaw);
      if (decoded is String && plans.any((p) => p.id == decoded)) {
        activePlanId = decoded;
      }
    }
    if (activePlanId.isEmpty) activePlanId = plans.first.id;

    PkWeekView weekView = PkWeekView();
    final String? weekRaw = prefs?.getString(ScheduleStorageKeys.weekView);
    if (weekRaw != null) {
      final Object? decoded = _tryJsonDecode(weekRaw);
      if (decoded is Map<String, dynamic>) {
        weekView = _sanitizeWeekView(decoded) ?? PkWeekView();
      }
    }

    bool isConfigCollapsed = false;
    final String? collapsedRaw = prefs?.getString(
      ScheduleStorageKeys.configCollapsed,
    );
    if (collapsedRaw == '1') {
      isConfigCollapsed = true;
    } else if (collapsedRaw == '0') {
      isConfigCollapsed = false;
    }

    String updateTime = '';
    final String? updateRaw = prefs?.getString(ScheduleStorageKeys.updateTime);
    if (updateRaw != null) {
      final Object? decoded = _tryJsonDecode(updateRaw);
      if (decoded is String) updateTime = decoded;
    }

    state = state.copyWith(
      majorSelected: majorSelected,
      plans: plans,
      activePlanId: activePlanId,
      weekView: weekView,
      isConfigCollapsed: isConfigCollapsed,
      updateTime: updateTime,
    );
  }

  Object? _tryJsonDecode(String raw) {
    try {
      return jsonDecode(raw);
    } catch (_) {
      return null;
    }
  }

  PkMajorSelection _sanitizeMajorSelection(Map<String, dynamic> json) {
    return PkMajorSelection(
      calendarId: json['calendarId'] is num
          ? (json['calendarId'] as num).toInt()
          : null,
      grade: json['grade'] is num ? (json['grade'] as num).toInt() : null,
      major: json['major'] is String ? json['major'] as String : null,
      majorName: json['majorName'] is String
          ? json['majorName'] as String
          : null,
    );
  }

  PkPlan? _sanitizePlan(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    final String id = raw['id'] is String ? raw['id'] as String : '';
    final String name = raw['name'] is String ? raw['name'] as String : '';
    final int createdAt = raw['createdAt'] is num
        ? (raw['createdAt'] as num).toInt()
        : 0;
    if (id.isEmpty) return null;
    return PkPlan(
      id: id,
      name: name.trim().isEmpty ? _planNameOf(1) : name.trim(),
      createdAt: createdAt != 0 ? createdAt : _nowMs(),
      stagedCourses: _asList(raw['stagedCourses'])
          .map(_sanitizeStagedCourse)
          .whereType<PkStagedCourse>()
          .where((course) => course.courseCode.isNotEmpty)
          .toList(),
      selectedCourses: _asList(
        raw['selectedCourses'],
      ).whereType<String>().where((code) => code.isNotEmpty).toList(),
      customEvents: _asList(raw['customEvents'])
          .map(_sanitizeCustomEvent)
          .whereType<PkCustomEvent>()
          .where((e) => e.day >= 1 && e.day <= 7 && e.sections.isNotEmpty)
          .toList(),
    );
  }

  PkStagedCourse? _sanitizeStagedCourse(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    final String courseCode = raw['courseCode'] is String
        ? raw['courseCode'] as String
        : '';
    if (courseCode.isEmpty) return null;
    return PkStagedCourse(
      courseCode: courseCode,
      courseName: raw['courseName'] is String
          ? raw['courseName'] as String
          : '',
      courseNameReserved: raw['courseNameReserved'] is String
          ? raw['courseNameReserved'] as String
          : '',
      credit: _normalizeCredit(raw['credit']),
      courseType: raw['courseType'] is String
          ? raw['courseType'] as String
          : '',
      courseNature: _normalizeStringList(raw['courseNature']),
      teacher: _sanitizeTeachers(raw['teacher']),
      status: raw['status'] is num ? (raw['status'] as num).toInt() : 0,
      courseDetail: _asList(raw['courseDetail'])
          .map(_sanitizeCourseDetail)
          .whereType<PkCourseDetail>()
          .where((detail) => detail.code.isNotEmpty)
          .toList(),
    );
  }

  PkCourseDetail? _sanitizeCourseDetail(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    final String code = raw['code'] is String ? raw['code'] as String : '';
    if (code.isEmpty) return null;
    // 消毒：非法占用安排（无有效星期 / 空节次）直接丢弃；周次夹到 1..16。
    final List<PkArrangement> arrangements = _asList(raw['arrangementInfo'])
        .map(_sanitizeArrangement)
        .whereType<PkArrangement>()
        .where(
          (a) =>
              a.occupyDay >= 1 && a.occupyDay <= 7 && a.occupyTime.isNotEmpty,
        )
        .toList();
    return PkCourseDetail(
      arrangementInfo: arrangements,
      campus: _normalizeStringList(raw['campus']).join('、'),
      code: code,
      teachingClassId: raw['teachingClassId'] is num
          ? (raw['teachingClassId'] as num).toInt()
          : null,
      isExclusive: raw['isExclusive'] is bool
          ? raw['isExclusive'] as bool
          : null,
      status: raw['status'] is num ? (raw['status'] as num).toInt() : 0,
      teachers: _sanitizeTeachers(raw['teachers']),
      teachingLanguage: raw['teachingLanguage'] is String
          ? raw['teachingLanguage'] as String
          : '',
    );
  }

  PkArrangement? _sanitizeArrangement(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    return PkArrangement(
      arrangementText: raw['arrangementText'] is String
          ? raw['arrangementText'] as String
          : '',
      occupyDay: raw['occupyDay'] is num
          ? (raw['occupyDay'] as num).toInt()
          : 0,
      occupyTime: _asList(raw['occupyTime'])
          .whereType<num>()
          .map((e) => e.toInt())
          .where((slot) => slot >= 1 && slot <= 12)
          .toList(),
      occupyWeek: _asList(raw['occupyWeek'])
          .whereType<num>()
          .map((e) => e.toInt())
          .map((week) => week < 1 ? 1 : (week > kMaxWeek ? kMaxWeek : week))
          .toList(),
      occupyRoom: raw['occupyRoom'] is String
          ? raw['occupyRoom'] as String
          : '',
      teacherAndCode: raw['teacherAndCode'] is String
          ? raw['teacherAndCode'] as String
          : '',
    );
  }

  PkCustomEvent? _sanitizeCustomEvent(dynamic raw) {
    if (raw is! Map<String, dynamic>) return null;
    return PkCustomEvent(
      id: raw['id'] is String && (raw['id'] as String).isNotEmpty
          ? raw['id'] as String
          : _genEventId(),
      label: raw['label'] is String ? raw['label'] as String : '',
      day: raw['day'] is num && (raw['day'] as num).toInt() >= 1
          ? (raw['day'] as num).toInt()
          : 0,
      sections: _sortedUnique(
        _asList(raw['sections'])
            .whereType<num>()
            .map((e) => e.toInt())
            .where((s) => s >= 1 && s <= 12),
      ),
      weeks: _sortedUnique(
        _asList(raw['weeks'])
            .whereType<num>()
            .map((e) => e.toInt())
            .map((w) => w < 1 ? 1 : (w > kMaxWeek ? kMaxWeek : w)),
      ),
    );
  }

  PkWeekView? _sanitizeWeekView(Map<String, dynamic> json) {
    final Object? rawWeek = json['week'];
    if (rawWeek is num) {
      final int week = rawWeek.toInt();
      return PkWeekView(
        week: (week >= 1 && week <= kMaxWeek) ? week : null,
        useCurrent: json['useCurrent'] == true,
      );
    }
    if (rawWeek == null) {
      return PkWeekView(week: null, useCurrent: json['useCurrent'] == true);
    }
    return null;
  }

  static List<dynamic> _asList(dynamic raw) =>
      raw is List<dynamic> ? raw : <dynamic>[];

  double _normalizeCredit(dynamic raw) {
    if (raw is num) return raw.toDouble();
    if (raw is String) return double.tryParse(raw.trim()) ?? 0;
    return 0;
  }

  List<String> _normalizeStringList(dynamic raw) {
    if (raw is String) {
      final String trimmed = raw.trim();
      if (trimmed.isEmpty ||
          trimmed == '[]' ||
          trimmed == '[""]' ||
          trimmed == 'null') {
        return <String>[];
      }
      if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
        return _normalizeStringList(_tryJsonDecode(trimmed));
      }
      return trimmed
          .split(',')
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();
    }
    if (raw is List<dynamic>) {
      return raw
          .expand((item) => _normalizeStringList(item))
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toList();
    }
    return <String>[];
  }

  List<PkTeacher> _sanitizeTeachers(dynamic raw) {
    return _asList(raw)
        .map((item) {
          if (item is! Map<String, dynamic>) return null;
          return PkTeacher(
            teacherName: item['teacherName'] is String
                ? item['teacherName'] as String
                : '',
            teacherCode: item['teacherCode'] is String
                ? item['teacherCode'] as String
                : '',
          );
        })
        .whereType<PkTeacher>()
        .toList();
  }

  List<int> _sortedUnique(Iterable<int> values) =>
      values.toSet().toList()..sort();

  /// 「方案 N」默认名序号（自定义名/旧数据不匹配返回 0）。
  int _defaultNameIndexOf(String name) {
    for (int n = 1; n <= kMaxPlans * 2; n++) {
      if (name == _planNameOf(n)) return n;
    }
    return 0;
  }

  // ---- 方案/事件 id ----

  PkPlan _createEmptyPlan(String name) {
    return PkPlan(
      id: 'plan_${_nowMs().toRadixString(36)}_${(_planSeq++).toRadixString(36)}',
      name: name,
      createdAt: _nowMs(),
      stagedCourses: <PkStagedCourse>[],
      selectedCourses: <String>[],
      customEvents: <PkCustomEvent>[],
    );
  }

  String _genEventId() =>
      'evt_${_nowMs().toRadixString(36)}_${(_eventSeq++).toRadixString(36)}';

  String _nextDefaultPlanName(List<PkPlan> plans) {
    int max = 0;
    for (final plan in plans) {
      final int index = _defaultNameIndexOf(plan.name);
      if (index > max) max = index;
    }
    return _planNameOf(max + 1);
  }

  // ---- 查询 ----

  PkPlan? _findPlan(String id) {
    for (final plan in state.plans) {
      if (plan.id == id) return plan;
    }
    return null;
  }

  /// 当前激活方案。
  PkPlan get activePlan => _findPlan(state.activePlanId) ?? state.plans.first;

  /// 学期/年级/专业是否已选齐（web isMajorSelected）。
  bool get isMajorSelected =>
      state.majorSelected.calendarId != null &&
      state.majorSelected.grade != null &&
      state.majorSelected.major != null &&
      state.majorSelected.major!.isNotEmpty;

  /// 派生课表最大行数：新 11 节 / 旧 12 节。
  int get maxRows => maxRowsForCalendar(state.majorSelected.calendarId);

  // ---- 公开操作 ----

  /// 学期/年级/专业选择变更：持久化三元组并清空全部方案的课程与占位
  /// （防跨学期污染，web 语义）。
  void setMajorSelection(PkMajorSelection selection) {
    final PkMajorSelection sanitized = PkMajorSelection(
      calendarId: selection.calendarId,
      grade: selection.grade,
      major: selection.major,
      majorName: selection.majorName,
    );
    state = state.copyWith(majorSelected: sanitized);
    _persist(ScheduleStorageKeys.majorSelected, jsonEncode(sanitized.toJson()));
    clearPlansData();
  }

  void setConfigCollapsed(bool collapsed) {
    if (state.isConfigCollapsed == collapsed) return;
    state = state.copyWith(isConfigCollapsed: collapsed);
    _persist(ScheduleStorageKeys.configCollapsed, collapsed ? '1' : '0');
  }

  void setWeekView(PkWeekView view) {
    final PkWeekView sanitized =
        _sanitizeWeekView({'week': view.week, 'useCurrent': view.useCurrent}) ??
        PkWeekView();
    if (state.weekView.week == sanitized.week) return;
    state = state.copyWith(weekView: sanitized);
    _persist(
      ScheduleStorageKeys.weekView,
      jsonEncode({'week': sanitized.week, 'useCurrent': sanitized.useCurrent}),
    );
    // weekView 属云同步快照字段：本地变更同样通知上行钩子。
    if (!_applyingRemote) {
      scheduleLocalPlansChanged?.call();
    }
    _rebuild();
  }

  /// 记录本地同步日期（web setUpdateTime：首次进入无课程时也推进时间）。
  void setUpdateTime(String date) {
    if (state.updateTime == date) return;
    state = state.copyWith(updateTime: date);
    _persist(ScheduleStorageKeys.updateTime, jsonEncode(date));
  }

  void setLatestUpdateTime(String date) {
    state = state.copyWith(latestUpdateTime: date);
  }

  // ---- 云端方案同步（issue #537）----

  /// 本地方案是否为空壳（仅一个方案且无课程/已选/占位；云同步 localEmpty）。
  bool get isLocalEmpty {
    if (state.plans.length != 1) return false;
    final PkPlan plan = state.plans.first;
    return plan.stagedCourses.isEmpty &&
        plan.selectedCourses.isEmpty &&
        plan.customEvents.isEmpty;
  }

  /// 服务端同步时钟（PUT 成功 / 整包采用后推进；持久化 pk.syncedAt）。
  String get syncedAt => _syncedAt;

  bool get syncDirty => _syncDirty;
  int? get syncOwner => int.tryParse(_prefs?.getString('pk.syncOwner') ?? '');

  Future<bool> setSyncOwner(int id) async {
    await flush;
    try {
      return await _prefs?.setString('pk.syncOwner', '$id') ?? false;
    } catch (_) {
      return false;
    }
  }

  void markSyncDirty() {
    _syncDirty = true;
    _syncSeq++;
    _persist('pk.syncDirty', '1');
  }

  /// Persist a complete local snapshot before advancing its synchronization clock.
  Future<bool> markSyncedAt(String updatedAt) {
    final seq = _syncSeq;
    final payload = buildSnapshotPayload().toJson();
    var saved = false;
    _writeQueue = _writeQueue.then((_) async {
      final prefs = _prefs;
      if (prefs == null) return;
      try {
        for (final key in [
          'plans',
          'activePlanId',
          'majorSelected',
          'weekView',
        ]) {
          if (!await prefs.setString('pk.$key', jsonEncode(payload[key]))) {
            return;
          }
        }
        if (seq != _syncSeq) return;
        if (!await prefs.setString(ScheduleStorageKeys.syncedAt, updatedAt)) {
          return;
        }
        if (!await prefs.setString('pk.syncDirty', '0')) return;
        if (seq != _syncSeq) return;
        _syncedAt = updatedAt;
        _syncDirty = false;
        saved = true;
      } catch (_) {
        /* A failed write leaves reconciliation pending. */
      }
    });
    return _writeQueue.then((_) => saved);
  }

  /// Snapshot upload with an optional observed server revision.
  PkPlanSnapshotPayload buildSnapshotPayload({String? baseUpdatedAt}) =>
      PkPlanSnapshotPayload(
        plans: state.plans,
        activePlanId: state.activePlanId,
        majorSelected: state.majorSelected,
        weekView: state.weekView,
        baseUpdatedAt: baseUpdatedAt,
      );

  /// 云端快照整包采用：applyingRemote 守卫下原子替换四字段并重建派生，
  /// 绝不触发本地变更钩子（防回灌），也不走 [setMajorSelection] 的
  /// 「换专业清空」语义。深度消毒与本地加载路径同源（_sanitize*）。
  void applyRemoteSnapshot(PkPlansSnapshot snapshot) {
    _applyingRemote = true;
    try {
      List<PkPlan> plans = snapshot.plans
          .map((plan) => _sanitizePlan(plan.toJson()))
          .whereType<PkPlan>()
          .toList();
      if (plans.isEmpty) {
        plans = <PkPlan>[_createEmptyPlan(_planNameOf(1))];
      }
      _planSeq = plans.fold<int>(0, (max, plan) {
        final int index = _defaultNameIndexOf(plan.name);
        return index > max ? index : max;
      });
      String activePlanId = snapshot.activePlanId;
      if (!plans.any((plan) => plan.id == activePlanId)) {
        activePlanId = plans.first.id;
      }
      final PkWeekView weekView =
          _sanitizeWeekView({
            'week': snapshot.weekView.week,
            'useCurrent': snapshot.weekView.useCurrent,
          }) ??
          PkWeekView();
      state = state.copyWith(
        majorSelected: snapshot.majorSelected,
        plans: plans,
        activePlanId: activePlanId,
        weekView: weekView,
      );
      _persistPlanData();
      _persist(
        ScheduleStorageKeys.majorSelected,
        jsonEncode(snapshot.majorSelected.toJson()),
      );
      _persist(
        ScheduleStorageKeys.weekView,
        jsonEncode({'week': weekView.week, 'useCurrent': weekView.useCurrent}),
      );

      _rebuild();
    } finally {
      _applyingRemote = false;
    }
  }

  /// 方案是否已过期（P11 > 本地 updateTime；页面据此展示陈旧横幅）。
  bool get isDataOutdated {
    final String latest = state.latestUpdateTime;
    if (latest.isEmpty || state.updateTime.isEmpty) return false;
    return latest != state.updateTime;
  }

  /// 同步最新（P12 结果应用，web applySyncToAllPlans）：各方案按课号命中
  /// 替换课程详情并保留每班排课状态；updateTime 推进到 [syncDate]。
  /// 返回同步后的状态（页面用于读取新 updateTime）。
  ScheduleState syncLatest(
    Map<String, List<PkCourseDetail>> detailsByCode, {
    String? syncDate,
  }) {
    final List<PkPlan> plans = state.plans.map((plan) {
      return _applySyncToPlan(plan, detailsByCode);
    }).toList();
    final String effectiveSyncDate = syncDate ?? state.latestUpdateTime;
    final String nextUpdateTime = effectiveSyncDate.isNotEmpty
        ? effectiveSyncDate
        : state.updateTime;
    state = state.copyWith(
      plans: plans,
      updateTime: nextUpdateTime,
      latestUpdateTime: nextUpdateTime,
    );
    _persistPlanData();
    _persist(ScheduleStorageKeys.updateTime, jsonEncode(nextUpdateTime));
    _rebuild();
    return state;
  }

  PkPlan _applySyncToPlan(
    PkPlan plan,
    Map<String, List<PkCourseDetail>> detailsByCode,
  ) {
    bool changed = false;
    final List<PkStagedCourse> staged = plan.stagedCourses.map((course) {
      final List<PkCourseDetail>? fresh = detailsByCode[course.courseCode];
      if (fresh == null || fresh.isEmpty) return course;
      changed = true;
      final List<PkCourseDetail> merged = fresh.map((detail) {
        int? status;
        for (final old in course.courseDetail) {
          if (old.code == detail.code) {
            status = old.status;
            break;
          }
        }
        return _detailWithStatus(detail, status ?? CourseStatus.unselected);
      }).toList();
      return PkStagedCourse(
        courseCode: course.courseCode,
        courseName: course.courseName,
        courseNameReserved: course.courseNameReserved,
        credit: course.credit,
        courseType: course.courseType,
        courseNature: course.courseNature,
        teacher: course.teacher,
        status: course.status,
        courseDetail: merged,
      );
    }).toList();
    if (!changed) return plan;
    return PkPlan(
      id: plan.id,
      name: plan.name,
      createdAt: plan.createdAt,
      stagedCourses: staged,
      selectedCourses: plan.selectedCourses,
      customEvents: plan.customEvents,
    );
  }

  // ---- 方案 CRUD ----

  /// 新建方案并激活；达到上限返回 null（web 同款）。
  PkPlan? createPlan() {
    if (state.plans.length >= kMaxPlans) return null;
    final PkPlan plan = _createEmptyPlan(_nextDefaultPlanName(state.plans));
    state = state.copyWith(
      plans: <PkPlan>[...state.plans, plan],
      activePlanId: plan.id,
    );
    _persistPlanData();
    _rebuild();
    return plan;
  }

  void switchPlan(String planId) {
    if (state.activePlanId == planId) return;
    if (_findPlan(planId) == null) return;
    state = state.copyWith(activePlanId: planId);
    _persist(ScheduleStorageKeys.activePlanId, jsonEncode(planId));
    // activePlanId 属云同步快照字段：本地切换方案也通知上行钩子。
    if (!_applyingRemote) {
      scheduleLocalPlansChanged?.call();
    }
    _rebuild();
  }

  void renamePlan(String planId, String name) {
    final String trimmed = name.trim();
    if (trimmed.isEmpty) return;
    final PkPlan? target = _findPlan(planId);
    if (target == null || target.name == trimmed) return;
    state = state.copyWith(
      plans: state.plans
          .map(
            (plan) => plan.id == planId
                ? PkPlan(
                    id: plan.id,
                    name: trimmed,
                    createdAt: plan.createdAt,
                    stagedCourses: plan.stagedCourses,
                    selectedCourses: plan.selectedCourses,
                    customEvents: plan.customEvents,
                  )
                : plan,
          )
          .toList(),
    );
    _persistPlanData();
  }

  /// 删除方案；删除最后一个时自动新建空方案（名称回到「方案 1」）。
  void deletePlan(String planId) {
    final List<PkPlan> remaining = state.plans
        .where((p) => p.id != planId)
        .toList();
    if (remaining.isEmpty) {
      _planSeq = 0;
      final PkPlan fresh = _createEmptyPlan(_planNameOf(1));
      state = state.copyWith(plans: <PkPlan>[fresh], activePlanId: fresh.id);
    } else {
      state = state.copyWith(
        plans: remaining,
        activePlanId: state.activePlanId == planId
            ? remaining.first.id
            : state.activePlanId,
      );
    }
    _persistPlanData();
    _rebuild();
  }

  /// 清空当前方案（保留方案壳）。
  void clearActivePlan() {
    _replaceActivePlan(
      staged: <PkStagedCourse>[],
      selected: <String>[],
      customEvents: <PkCustomEvent>[],
    );
  }

  /// 清空全部方案的课程与占位（学期/年级/专业变更时调用）。
  void clearPlansData() {
    final List<PkPlan> cleared = state.plans.map((plan) {
      return PkPlan(
        id: plan.id,
        name: plan.name,
        createdAt: plan.createdAt,
        stagedCourses: <PkStagedCourse>[],
        selectedCourses: <String>[],
        customEvents: <PkCustomEvent>[],
      );
    }).toList();
    state = state.copyWith(plans: cleared);
    _persistPlanData();
    _rebuild();
  }

  // ---- 课程操作 ----

  /// 退掉整门课（入参可为基础课号或班级课号；web removeCourseFromSchedule）。
  void removeCourse(String courseCodeOrClassCode) {
    final PkPlan active = activePlan;
    final String input = courseCodeOrClassCode.trim();
    final bool isBase = active.stagedCourses.any(
      (course) => course.courseCode == input,
    );
    final String base = isBase ? input : getCourseBaseCode(input);
    _replaceActivePlan(
      staged: active.stagedCourses
          .where((course) => course.courseCode != base)
          .toList(),
      selected: active.selectedCourses
          .where((code) => !isClassOfCourse(code, base))
          .toList(),
    );
  }

  /// 选班（移动端主入口）：容忍式加课，总是成功；同基础课号旧班先移除
  /// （换班不报冲突）。班级以 status=2 直接上表（移动端无「保存课表」步骤）。
  StageCourseResult selectClass(PkCourseDetail detail, String courseName) {
    return _insertClass(detail, courseName, CourseStatus.selected);
  }

  /// 备选加课（web appendToTimeTable 语义：status=1 仅入列表，不上表）。
  /// 保留给需要「先备选后统一上表」的流程；当前移动端 UI 用 [selectClass]。
  StageCourseResult stageClass(PkCourseDetail detail, String courseName) {
    return _insertClass(detail, courseName, CourseStatus.staged);
  }

  StageCourseResult _insertClass(
    PkCourseDetail detail,
    String courseName,
    int status,
  ) {
    final PkPlan active = activePlan;
    final String base = getCourseBaseCode(detail.code);

    // 同基础课号旧班：换班 = 先把旧班从占用/已选中摘除，再判冲突。
    PkCourseDetail? oldSelected;
    for (final course in active.stagedCourses) {
      if (course.courseCode != base) continue;
      for (final old in course.courseDetail) {
        if (old.code == detail.code) continue;
        if (CourseStatus.arranged(old.status ?? CourseStatus.unselected)) {
          oldSelected = old;
          break;
        }
      }
    }

    final List<PkConflictItem> conflicts = oldSelected == null
        ? findConflicts(
            detail,
            state.occupied,
          ).where((c) => !isSameCourse(c.code, detail.code)).toList()
        : <PkConflictItem>[];

    List<String> selected = <String>[
      ...active.selectedCourses.where(
        (code) => !isSameCourse(code, detail.code),
      ),
    ];

    final List<PkStagedCourse> staged = <PkStagedCourse>[];
    PkStagedCourse? shell;
    for (final course in active.stagedCourses) {
      if (course.courseCode != base) {
        staged.add(course);
        continue;
      }
      final List<PkCourseDetail> details = <PkCourseDetail>[];
      for (final old in course.courseDetail) {
        if (isSameCourse(old.code, detail.code)) continue;
        if (oldSelected != null && old.code == oldSelected.code) {
          // 旧班被替换：已保存的移出已选列表，状态归零。
          if (old.status == CourseStatus.selected) {
            selected = selected.where((code) => code != old.code).toList();
          }
          details.add(_detailWithStatus(old, CourseStatus.unselected));
          continue;
        }
        details.add(old);
      }
      details.add(_detailWithStatus(detail, status));
      shell = PkStagedCourse(
        courseCode: course.courseCode,
        courseName: course.courseName,
        courseNameReserved: course.courseNameReserved,
        credit: course.credit,
        courseType: course.courseType,
        courseNature: course.courseNature,
        teacher: detail.teachers,
        status: _shellStatusOf(details),
        courseDetail: details,
      );
      staged.add(shell);
    }
    if (shell == null) {
      shell = PkStagedCourse(
        courseCode: base,
        courseName: courseName,
        courseNameReserved: courseName,
        credit: 0,
        courseType: '',
        courseNature: null,
        teacher: detail.teachers,
        status: status,
        courseDetail: <PkCourseDetail>[_detailWithStatus(detail, status)],
      );
      staged.add(shell);
    }

    if (status == CourseStatus.selected) {
      selected = <String>[...selected, detail.code];
    }

    _replaceActivePlan(staged: staged, selected: selected);
    return StageCourseResult(added: true, conflicts: conflicts);
  }

  /// 移除一个已选/备选班级（班级级反选；课程仍保留在备选池）。
  void deselectClass(String classCode) {
    final PkPlan active = activePlan;
    final String base = getCourseBaseCode(classCode);
    final List<PkStagedCourse> staged = active.stagedCourses.map((course) {
      if (course.courseCode != base) return course;
      final List<PkCourseDetail> details = course.courseDetail
          .where((detail) => detail.code != classCode)
          .map((detail) => detail)
          .toList();
      return PkStagedCourse(
        courseCode: course.courseCode,
        courseName: course.courseName,
        courseNameReserved: course.courseNameReserved,
        credit: course.credit,
        courseType: course.courseType,
        courseNature: course.courseNature,
        teacher: course.teacher,
        status: _shellStatusOf(details),
        courseDetail: details,
      );
    }).toList();
    _replaceActivePlan(
      staged: staged,
      selected: active.selectedCourses
          .where((code) => code != classCode)
          .toList(),
    );
  }

  /// 清空一门课的已选班级（web clearStagedCourseClass：所有 1/2 重置，课程保留）。
  void clearStagedCourseClass(String courseCode) {
    final PkPlan active = activePlan;
    final List<PkStagedCourse> staged = active.stagedCourses.map((course) {
      if (course.courseCode != courseCode) return course;
      final List<PkCourseDetail> details = course.courseDetail
          .map(
            (detail) =>
                CourseStatus.arranged(detail.status ?? CourseStatus.unselected)
                ? _detailWithStatus(detail, CourseStatus.unselected)
                : detail,
          )
          .toList();
      return PkStagedCourse(
        courseCode: course.courseCode,
        courseName: course.courseName,
        courseNameReserved: course.courseNameReserved,
        credit: course.credit,
        courseType: course.courseType,
        courseNature: course.courseNature,
        teacher: course.teacher,
        status: 0,
        courseDetail: details,
      );
    }).toList();
    _replaceActivePlan(
      staged: staged,
      selected: active.selectedCourses
          .where((code) => !isClassOfCourse(code, courseCode))
          .toList(),
    );
  }

  /// 壳课程状态推导：有已选(2) → 2；否则有待选(1) → 1；否则 0。
  static int _shellStatusOf(List<PkCourseDetail> details) {
    int status = CourseStatus.unselected;
    for (final detail in details) {
      if (detail.status == CourseStatus.selected) return CourseStatus.selected;
      if (detail.status == CourseStatus.staged &&
          status < CourseStatus.staged) {
        status = CourseStatus.staged;
      }
    }
    return status;
  }

  // ---- 自定义占位 ----

  /// 添加自定义占位事件；day/sections/weeks 非法返回 null。
  PkCustomEvent? addCustomEvent({
    required String label,
    required int day,
    required List<int> sections,
    required List<int> weeks,
  }) {
    final PkCustomEvent event = PkCustomEvent(
      id: _genEventId(),
      label: label.trim().isEmpty ? '有事' : label.trim(),
      day: day,
      sections: _sortedUnique(sections.where((s) => s >= 1 && s <= 12)),
      weeks: _sortedUnique(weeks.where((w) => w >= 1 && w <= kMaxWeek)),
    );
    if (event.day < 1 ||
        event.day > 7 ||
        event.sections.isEmpty ||
        event.weeks.isEmpty) {
      return null;
    }
    final PkPlan active = activePlan;
    _replaceActivePlan(
      customEvents: <PkCustomEvent>[...active.customEvents, event],
    );
    return event;
  }

  void removeCustomEvent(String id) {
    final PkPlan active = activePlan;
    _replaceActivePlan(
      customEvents: active.customEvents
          .where((event) => event.id != id)
          .toList(),
    );
  }

  // ---- 内部 ----

  void _replaceActivePlan({
    List<PkStagedCourse>? staged,
    List<String>? selected,
    List<PkCustomEvent>? customEvents,
  }) {
    final PkPlan active = activePlan;
    state = state.copyWith(
      plans: state.plans
          .map(
            (plan) => plan.id == active.id
                ? PkPlan(
                    id: plan.id,
                    name: plan.name,
                    createdAt: plan.createdAt,
                    stagedCourses: staged ?? plan.stagedCourses,
                    selectedCourses: selected ?? plan.selectedCourses,
                    customEvents: customEvents ?? plan.customEvents,
                  )
                : plan,
          )
          .toList(),
    );
    _persistPlanData();
    _rebuild();
  }

  PkCourseDetail _detailWithStatus(PkCourseDetail detail, int status) {
    return PkCourseDetail(
      arrangementInfo: detail.arrangementInfo,
      campus: detail.campus,
      code: detail.code,
      teachingClassId: detail.teachingClassId,
      isExclusive: detail.isExclusive,
      status: status,
      teachers: detail.teachers,
      teachingLanguage: detail.teachingLanguage,
    );
  }

  /// 重建派生（occupied/grid/stats）。派生永不落盘。
  void _rebuild() {
    final PkPlan active = activePlan;
    final int rows = maxRows;
    final (List<PkCourseOnTable> rows2, List<List<List<PkOccupyCell>>> occ) =
        _deriveOccupiedAndRows(active);
    List<PkCourseOnTable> table = rows2;
    final List<List<List<PkOccupyCell>>> occupied = occ;

    // 周次过滤（网格按当前周次视图过滤，occupied 全量用于冲突派生）。
    final int? week = state.weekView.week;
    if (week != null) {
      table = table
          .where((course) => (course.occupyWeek ?? <int>[]).contains(week))
          .toList();
    }

    final ScheduleGridData grid = buildScheduleGridData(
      table,
      occupied,
      maxRows: rows,
    );

    final Map<String, List<PkConflictItem>> conflicts = deriveConflicts(
      occupied,
    );
    final ScheduleStats stats = computeScheduleStats(active, conflicts);

    state = state.copyWith(
      occupied: occupied,
      grid: ScheduleGridData(
        cellCourses: grid.cellCourses,
        cellSpans: grid.cellSpans,
        occupiedGrid: grid.occupiedGrid,
        rowHeights: grid.rowHeights,
        conflicts: conflicts,
      ),
      stats: stats,
    );
  }
}

/// 空网格（未初始化/空状态占位）。
ScheduleGridData _emptyGrid() {
  return ScheduleGridData(
    cellCourses: List.generate(
      12,
      (_) => List.generate(7, (_) => <PkCourseOnTable>[]),
    ),
    cellSpans: List.generate(12, (_) => List.filled(7, 1)),
    occupiedGrid: List.generate(12, (_) => List.filled(7, false)),
    rowHeights: List.filled(12, kInteractiveRowMetricsMobile.baseH),
    conflicts: <String, List<PkConflictItem>>{},
  );
}

/// 从方案派生 occupied + 平铺课表行：
/// 仅 status==2 的班级进课表（1 只显示在列表）；custom 事件逐节次成行，
/// 以 `custom:<id>` 伪课号入 occupied。
(List<PkCourseOnTable>, List<List<List<PkOccupyCell>>>) _deriveOccupiedAndRows(
  PkPlan plan,
) {
  List<List<List<PkOccupyCell>>> occupied = createEmptyOccupied();
  final List<PkCourseOnTable> rows = <PkCourseOnTable>[];

  for (final course in plan.stagedCourses) {
    final String displayName = course.courseNameReserved.isNotEmpty
        ? course.courseNameReserved
        : course.courseName;
    for (final detail in course.courseDetail) {
      if (detail.status != CourseStatus.selected) continue;
      for (final arrangement in detail.arrangementInfo) {
        rows.add(
          PkCourseOnTable(
            showText: [
              arrangement.teacherAndCode,
              '$displayName(${detail.code})',
              arrangement.arrangementText,
            ].where((s) => s.isNotEmpty).join(' '),
            courseName: displayName,
            code: detail.code,
            occupyTime: List<int>.from(arrangement.occupyTime),
            occupyDay: arrangement.occupyDay,
            occupyWeek: List<int>.from(arrangement.occupyWeek),
            teacherAndCode: arrangement.teacherAndCode,
            arrangementText: arrangement.arrangementText,
            occupyRoom: arrangement.occupyRoom,
          ),
        );
      }
      occupied = insertOccupied(
        occupied,
        detail.arrangementInfo,
        detail.code,
        displayName,
      );
    }
  }

  for (final event in plan.customEvents) {
    final String code = '$kCustomEventCodePrefix${event.id}';
    for (final section in event.sections) {
      rows.add(
        PkCourseOnTable(
          showText: event.label,
          courseName: event.label,
          code: code,
          occupyTime: <int>[section],
          occupyDay: event.day,
          occupyWeek: List<int>.from(event.weeks),
        ),
      );
    }
    occupied = insertOccupied(
      occupied,
      <PkArrangement>[
        PkArrangement(
          occupyDay: event.day,
          occupyTime: List<int>.from(event.sections),
          occupyWeek: List<int>.from(event.weeks),
        ),
      ],
      code,
      event.label,
    );
  }
  return (rows, occupied);
}

/// 从（周次过滤后的）平铺课表行构建网格：
/// 同天按节次区间聚类（相交/包含同格）→ consolidate 同班多段 → spans/覆盖表
/// → computeRowHeights 行高。occup 仅用于 deriveConflicts（全量冲突标注）。
ScheduleGridData buildScheduleGridData(
  List<PkCourseOnTable> table,
  List<List<List<PkOccupyCell>>> occupied, {
  required int maxRows,
}) {
  final List<List<PkCourseOnTable>> byDay = List.generate(
    7,
    (_) => <PkCourseOnTable>[],
  );
  for (final course in table) {
    if (course.occupyTime.isEmpty) continue;
    if (course.occupyDay < 1 || course.occupyDay > 7) continue;
    if (course.occupyTime.any((s) => s < 1 || s > maxRows)) continue;
    byDay[course.occupyDay - 1].add(course);
  }

  final List<List<List<PkCourseOnTable>>> cellCourses = List.generate(
    maxRows,
    (_) => List.generate(7, (_) => <PkCourseOnTable>[]),
  );
  final List<List<int>> cellSpans = List.generate(
    maxRows,
    (_) => List.filled(7, 1),
  );
  final List<List<bool>> covered = List.generate(
    maxRows,
    (_) => List.filled(7, false),
  );

  for (int day = 0; day < 7; day++) {
    final List<PkDayCluster<PkCourseOnTable>> clusters = clusterBySections(
      byDay[day],
      (course) => course.occupyTime,
    );
    for (final cluster in clusters) {
      final List<PkConsolidatableCourse> consolidated =
          consolidateSameClassArrangements(
            cluster.items
                .map(
                  (course) => PkConsolidatableCourse(
                    code: course.code,
                    courseName: course.courseName,
                    occupyDay: course.occupyDay,
                    occupyTime: List<int>.from(course.occupyTime),
                    occupyWeek: course.occupyWeek == null
                        ? null
                        : List<int>.from(course.occupyWeek!),
                    teacherAndCode: course.teacherAndCode,
                    arrangementText: course.arrangementText,
                    occupyRoom: course.occupyRoom,
                    showText: course.showText,
                  ),
                )
                .toList(),
          );
      final int row = cluster.start - 1;
      final int span = cluster.end - row;
      cellCourses[row][day] = consolidated
          .map(
            (c) => PkCourseOnTable(
              showText: c.showText ?? '',
              courseName: c.courseName,
              code: c.code,
              occupyTime: List<int>.from(c.occupyTime),
              occupyDay: c.occupyDay,
              occupyWeek: c.occupyWeek == null
                  ? null
                  : List<int>.from(c.occupyWeek!),
              teacherAndCode: c.teacherAndCode,
              arrangementText: c.arrangementText,
              occupyRoom: c.occupyRoom,
            ),
          )
          .toList();
      cellSpans[row][day] = span;
      for (int r = row + 1; r < row + span && r < maxRows; r++) {
        covered[r][day] = true;
      }
    }
  }

  final GridLayout layout = GridLayout(
    cellCourses: cellCourses,
    cellSpans: cellSpans,
    occupiedGrid: covered,
  );
  final List<int> rowHeights = computeRowHeights(
    layout,
    kInteractiveRowMetricsMobile,
  );
  final Map<String, List<PkConflictItem>> conflicts = deriveConflicts(occupied);

  return ScheduleGridData(
    cellCourses: cellCourses,
    cellSpans: cellSpans,
    occupiedGrid: covered,
    rowHeights: rowHeights,
    conflicts: conflicts,
  );
}

/// 统计（web stats() 同口径）：已排班级（1 或 2）计入门数与学分；
/// 学时 = Σ 节数×周数；冲突 = 非 custom 的冲突基础课号数。
ScheduleStats computeScheduleStats(
  PkPlan plan,
  Map<String, List<PkConflictItem>> conflicts,
) {
  int courseCount = 0;
  double totalCredit = 0;
  int totalHours = 0;
  for (final course in plan.stagedCourses) {
    final List<PkCourseDetail> arranged = course.courseDetail
        .where(
          (d) => CourseStatus.arranged(d.status ?? CourseStatus.unselected),
        )
        .toList();
    if (arranged.isEmpty) continue;
    courseCount += 1;
    totalCredit += course.credit;
    for (final detail in arranged) {
      for (final arrangement in detail.arrangementInfo) {
        totalHours +=
            arrangement.occupyTime.length * arrangement.occupyWeek.length;
      }
    }
  }
  final int conflictCount = conflicts.keys
      .where((key) => !key.startsWith(kCustomEventCodePrefix))
      .length;
  return ScheduleStats(
    courseCount: courseCount,
    totalCredit: totalCredit,
    totalHours: totalHours,
    conflictCount: conflictCount,
  );
}

/// 排课器 Notifier Provider（页面初始化时 await notifier.ready）。
final StateNotifierProvider<ScheduleStoreNotifier, ScheduleState>
scheduleStoreProvider =
    StateNotifierProvider<ScheduleStoreNotifier, ScheduleState>(
      (ref) => ScheduleStoreNotifier(),
    );

/// 当前激活方案（跟随 [scheduleStoreProvider] 的派生 Provider）。
final Provider<PkPlan> activePlanProvider = Provider<PkPlan>((ref) {
  final ScheduleState state = ref.watch(scheduleStoreProvider);
  for (final plan in state.plans) {
    if (plan.id == state.activePlanId) return plan;
  }
  return state.plans.first;
});
