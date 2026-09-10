import 'scheduler_web_tip.dart';
// 排课器主页面（/schedule 路由目标）：移动端双 tab（课表 / 选课）+ 方案条 +
// 学期·年级·专业配置行 + 数据过期同步 + 自定义占位 + PNG/CSV 导出。
//
// 设计对齐 web SchedulePage.vue 的移动端形态，采用 Gf 设计语言：8px 圆角
// （gf-radius-*）、GfTheme.colorsOf(context) token 配色（primary 选中 /
// error 冲突 / line 网格发丝线），light/dark 由 ui_kit token 自动切换。
// 课表网格为自绘（非 Table）：行高由 core pk_grid computeRowHeights +
// kInteractiveRowMetricsMobile 计算，天列用 Stack 以行高累计定位课程卡
// （跨节簇 rowspan 自然成立）；课程卡 = 课名(2 行省略) + 教室/教师 +
// 单双周徽标（formatDisplayWeeks 短式），冲突课程带 ⚠ 角标。
//
// 与 web 的差异（有意适配移动端）：
// - 选班即上表（无 web「保存课表」确认步骤；status=2 立即持久化）。
// - 导出只做 PNG 截图分享与 CSV（UTF-8 BOM）；XLS/XLSX 不做。
// - PNG 为网格截图分享（无 web 海报的文字排版画布）。
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:share_plus/share_plus.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../schedule/schedule_store.dart';
import '../../schedule/schedule_sync.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';

/// 教学班简表 → 域课程详情（status 由调用方/store 决定）。
PkCourseDetail detailFromBrief(PkCourseDetailBrief brief) {
  return PkCourseDetail(
    arrangementInfo: brief.arrangementInfo
        .map(
          (PkArrangementInfo a) => PkArrangement(
            arrangementText: a.arrangementText,
            occupyDay: a.occupyDay ?? 0,
            occupyTime: a.occupyTime ?? const <int>[],
            occupyWeek: a.occupyWeek ?? const <int>[],
            occupyRoom: a.occupyRoom ?? '',
            teacherAndCode: a.teacherAndCode ?? '',
          ),
        )
        .where(
          (PkArrangement a) =>
              a.occupyDay >= 1 && a.occupyDay <= 7 && a.occupyTime.isNotEmpty,
        )
        .toList(),
    campus: brief.campus,
    code: brief.code,
    teachingClassId: brief.teachingClassId,
    isExclusive: brief.isExclusive,
    status: 0,
    teachers: brief.teachers
        .map(
          (PkTeacherRef t) =>
              PkTeacher(teacherName: t.teacherName, teacherCode: t.teacherCode),
        )
        .toList(),
    teachingLanguage: brief.teachingLanguage,
  );
}

/// 主页面：排课器。
class SchedulePage extends ConsumerStatefulWidget {
  const SchedulePage({super.key});

  @override
  ConsumerState<SchedulePage> createState() => _SchedulePageState();
}

class _SchedulePageState extends ConsumerState<SchedulePage>
    with WidgetsBindingObserver {
  bool _ready = false;
  bool _tabTimetable = false;
  bool _syncing = false;

  List<PkCalendarItem> _calendars = const <PkCalendarItem>[];
  List<SectionTime> _sectionOverrides = const <SectionTime>[];
  final GlobalKey _gridBoundaryKey = GlobalKey();

  // Capture the application controller; page exit flushes pending local edits.
  late final ScheduleSyncController _syncController;
  bool _showingSyncConflict = false;

  ScheduleState get _state => ref.read(scheduleStoreProvider);

  ScheduleStoreNotifier get _notifier =>
      ref.read(scheduleStoreProvider.notifier);

  @override
  void initState() {
    super.initState();
    _syncController = ref.read(scheduleSyncControllerProvider);
    _syncController.conflict.addListener(_onSyncConflict);
    WidgetsBinding.instance.addObserver(this);
    ref.read(scheduleStoreProvider.notifier).ready.then((_) {
      if (!mounted) return;
      setState(() => _ready = true);
      _loadSessionMeta();
      _syncPlansOnEnter();
    });
  }

  @override
  void dispose() {
    _syncController.conflict.removeListener(_onSyncConflict);
    unawaited(_syncController.flushPendingUpload());
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // paused 尽力冲刷未上行的本地方案（issue #537；best-effort）。
    if (state == AppLifecycleState.paused) {
      unawaited(ref.read(scheduleSyncControllerProvider).flushPendingUpload());
    }
  }

  /// 进页方案云同步对账（issue #537）：未登录零请求；冲突时弹窗二选一。
  Future<void> _syncPlansOnEnter() async {
    await _syncController.syncOnEnter();
  }

  void _onSyncConflict() {
    final snapshot = _syncController.conflict.value;
    if (!mounted || snapshot == null || _showingSyncConflict) return;
    _showingSyncConflict = true;
    unawaited(
      _showPlanSyncConflictDialog(snapshot).whenComplete(() {
        _showingSyncConflict = false;
      }),
    );
  }

  /// 冲突弹窗（一次性）：「使用云端」整包采用 / 「保留本地」立即上行。
  Future<void> _showPlanSyncConflictDialog(PkPlansSnapshot snapshot) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ScheduleSyncController sync = ref.read(
      scheduleSyncControllerProvider,
    );
    await showGfAlertDialog<void>(
      context,
      barrierDismissible: false,
      builder: (BuildContext dialogContext) => Padding(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 8),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text(
              l10n.scheduleSyncConflictTitle,
              style: GfTheme.typographyOf(dialogContext).heading,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.scheduleSyncConflictBody,
              style: GfTheme.typographyOf(dialogContext).body,
            ),
            const SizedBox(height: 18),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: <Widget>[
                GfButton(
                  label: l10n.scheduleSyncKeepLocal,
                  variant: GfButtonVariant.ghost,
                  onPressed: () {
                    Navigator.of(dialogContext).pop();
                    unawaited(sync.keepLocal());
                  },
                ),
                const SizedBox(width: 8),
                GfButton(
                  label: l10n.scheduleSyncUseCloud,
                  onPressed: () {
                    Navigator.of(dialogContext).pop();
                    unawaited(sync.adoptRemote(snapshot));
                  },
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    // 绑定 l10n 方案名（store 持久化恢复使用同文案）。
    final AppLocalizations l10n = AppLocalizations.of(context);
    ref.read(scheduleStoreProvider.notifier).setPlanNaming(l10n.schedulePlanN);
  }

  String _calendarName(int? calendarId) {
    for (final PkCalendarItem calendar in _calendars) {
      if (calendar.calendarId == calendarId) return calendar.calendarName;
    }
    return '';
  }

  /// 会话级元数据：学期字典 + 节次作息 + P11 最新同步日期（不持久化）。
  Future<void> _loadSessionMeta() async {
    try {
      final List<PkCalendarItem> calendars = await ref
          .read(pkRepositoryProvider)
          .calendars();
      if (mounted) setState(() => _calendars = calendars);
    } catch (_) {
      // 学期字典失败不阻塞主流程。
    }
    try {
      final SectionTimesPayload? payload = await ref
          .read(pkRepositoryProvider)
          .sectionTimes();
      if (mounted && payload != null) {
        setState(() {
          _sectionOverrides = payload.sectionTimes
              .map(
                (SectionTimeSetting setting) => SectionTime(
                  section: setting.section,
                  start: setting.start,
                  end: setting.end,
                ),
              )
              .toList();
        });
      }
    } catch (_) {
      // 作息表失败回退默认表。
    }
    try {
      final String? latest = await ref
          .read(pkRepositoryProvider)
          .latestUpdate();
      if (mounted && latest != null && latest.isNotEmpty) {
        _notifier.setLatestUpdateTime(latest);
      }
    } catch (_) {
      // P11 不可用时静默（web 同款：不提示过期）。
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ScheduleState state = ref.watch(scheduleStoreProvider);
    final GfColors colors = GfTheme.colorsOf(context);
    final bool outdated = _notifier.isDataOutdated;

    return Scaffold(
      backgroundColor: colors.base100,
      appBar: GfAppBar(
        title: Text(l10n.scheduleTitle),
        actions: <Widget>[
          GfIconButton(
            icon: Icons.event_available_outlined,
            size: 44,
            tooltip: l10n.scheduleAddCustomEvent,
            onPressed: _openCustomEventSheet,
          ),
          PopupMenuButton<_ExportAction>(
            tooltip: l10n.scheduleExportPng,
            icon: Icon(Icons.ios_share, size: 20, color: colors.iconMuted),
            color: colors.base100,
            onSelected: (_ExportAction action) {
              if (action == _ExportAction.png) {
                _exportPng();
              } else {
                _exportCsv();
              }
            },
            itemBuilder: (BuildContext menuContext) =>
                <PopupMenuEntry<_ExportAction>>[
                  PopupMenuItem<_ExportAction>(
                    value: _ExportAction.png,
                    child: Text(l10n.scheduleExportPng),
                  ),
                  PopupMenuItem<_ExportAction>(
                    value: _ExportAction.csv,
                    child: Text(l10n.scheduleExportCsv),
                  ),
                ],
          ),
        ],
      ),
      body: !_ready
          ? const GfScheduleSkeleton()
          : ListView(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 28),
              children: <Widget>[
                const SchedulerWebTip(),
                const SizedBox(height: 16),
                _PlanBar(notifier: _notifier, state: state),
                const SizedBox(height: 8),
                if (state.isConfigCollapsed)
                  _CollapsedConfigRow(
                    calendarName: _calendarName(state.majorSelected.calendarId),
                    state: state,
                    onTap: () => _notifier.setConfigCollapsed(false),
                  )
                else
                  _ConfigRow(
                    calendars: _calendars,
                    state: state,
                    notifier: _notifier,
                  ),
                const SizedBox(height: 8),
                if (outdated)
                  _DataOutdatedBanner(loading: _syncing, onTap: _syncLatest),
                if (outdated) const SizedBox(height: 8),
                _StatsFooter(state: state),
                const SizedBox(height: 8),
                _ScheduleTabs(
                  selectedTimetable: _tabTimetable,
                  onSelected: (bool timetable) =>
                      setState(() => _tabTimetable = timetable),
                ),
                const SizedBox(height: 8),
                if (_tabTimetable) ...[
                  Text(
                    l10n.schedulerPlanDisclaimer,
                    style: GfTheme.typographyOf(context).caption,
                  ),
                  const SizedBox(height: 12),
                  _TimetableTab(
                    boundaryKey: _gridBoundaryKey,
                    sectionOverrides: _sectionOverrides,
                  ),
                ] else
                  const _PickTab(),
              ],
            ),
    );
  }

  Future<void> _openCustomEventSheet() async {
    await showGfBottomSheet<void>(
      context,
      builder: (BuildContext sheetContext) => const _CustomEventSheet(),
    );
  }

  // ---- 同步最新 ----

  Future<void> _syncLatest() async {
    if (_syncing) return;
    _syncing = true;
    setState(() {});
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ScheduleState state = _state;
    try {
      final int calendarId = state.majorSelected.calendarId ?? 0;
      final List<String> majorCodes = <String>[];
      final List<String> otherCodes = <String>[];
      for (final PkPlan plan in state.plans) {
        for (final PkStagedCourse course in plan.stagedCourses) {
          final bool exclusive = course.courseDetail.any(
            (detail) => detail.isExclusive == true,
          );
          (exclusive ? majorCodes : otherCodes).add(course.courseCode);
        }
      }
      final int? grade = state.majorSelected.grade;
      final String? major = state.majorSelected.major;
      final (int, String)? majorInfo = grade != null && (major ?? '').isNotEmpty
          ? (grade, major!)
          : null;
      final Map<String, List<PkCourseDetailBrief>> briefs = await ref
          .read(pkRepositoryProvider)
          .courseInfoSync(
            calendarId: calendarId,
            majorCourseCodes: majorCodes,
            otherCourseCodes: otherCodes,
            majorInfo: majorInfo,
          );
      final Map<String, List<PkCourseDetail>> detailsByCode = briefs.map(
        (String code, List<PkCourseDetailBrief> list) =>
            MapEntry(code, list.map(detailFromBrief).toList()),
      );
      final ScheduleState synced = _notifier.syncLatest(
        detailsByCode,
        syncDate: _state.latestUpdateTime,
      );
      if (!mounted) return;
      showGfToast(context, l10n.scheduleSyncedTo(synced.updateTime));
    } catch (e) {
      if (!mounted) return;
      showGfToast(context, resolveErrorMessage(l10n, e), error: true);
    } finally {
      _syncing = false;
      if (mounted) setState(() {});
    }
  }

  // ---- 导出 ----

  Future<void> _exportPng() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final RenderRepaintBoundary? boundary =
        _gridBoundaryKey.currentContext?.findRenderObject()
            as RenderRepaintBoundary?;
    if (boundary == null) {
      showGfToast(context, l10n.commonEmpty, error: true);
      return;
    }
    try {
      final ui.Image image = await boundary.toImage(pixelRatio: 3);
      final ByteData? bytes = await image.toByteData(
        format: ui.ImageByteFormat.png,
      );
      if (bytes == null) throw StateError('PNG encode failed');
      final Directory dir = Directory.systemTemp;
      final File file = File('${dir.path}/yourtj-schedule.png');
      await file.writeAsBytes(bytes.buffer.asUint8List());
      await SharePlus.instance.share(
        ShareParams(
          files: <XFile>[XFile(file.path, mimeType: 'image/png')],
          subject: l10n.scheduleExportPng,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      showGfToast(context, '$e', error: true);
    }
  }

  Future<void> _exportCsv() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String csv = _buildCsv();
    if (csv.trim().isEmpty) {
      showGfToast(context, l10n.commonEmpty, error: true);
      return;
    }
    try {
      final Directory dir = Directory.systemTemp;
      final File file = File('${dir.path}/yourtj-schedule.csv');
      // 中文 Excel 兼容：UTF-8 BOM。
      await file.writeAsBytes(utf8.encode('\uFEFF$csv'));
      await SharePlus.instance.share(
        ShareParams(
          files: <XFile>[XFile(file.path, mimeType: 'text/csv')],
          subject: l10n.scheduleExportCsv,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      showGfToast(context, '$e', error: true);
    }
  }

  /// CSV 行：课程名,星期,开始节,结束节,教师,教室,周次（一个时间段一行）。
  String _buildCsv() {
    final PkPlan plan = _notifier.activePlan;
    final StringBuffer buffer = StringBuffer();
    for (final PkStagedCourse course in plan.stagedCourses) {
      for (final PkCourseDetail detail in course.courseDetail) {
        if (detail.status != CourseStatus.selected) continue;
        for (final PkArrangement arrangement in detail.arrangementInfo) {
          final List<int> times = arrangement.occupyTime;
          if (times.isEmpty ||
              arrangement.occupyDay < 1 ||
              arrangement.occupyDay > 7) {
            continue;
          }
          final int start = times.first;
          final int end = times.length == 1 ? start : times.last;
          buffer
            ..write(_csvCell(course.courseName))
            ..write(',')
            ..write('${arrangement.occupyDay}')
            ..write(',')
            ..write('$start')
            ..write(',')
            ..write('$end')
            ..write(',')
            ..write(_csvCell(_stripTeacherCode(arrangement.teacherAndCode)))
            ..write(',')
            ..write(_csvCell(arrangement.occupyRoom))
            ..write(',')
            ..write(_csvCell(_weeksText(arrangement.occupyWeek)))
            ..write('\n');
        }
      }
    }
    return buffer.toString();
  }

  static String _csvCell(String value) {
    if (value.contains(',') || value.contains('"') || value.contains('\n')) {
      return '"${value.replaceAll('"', '""')}"';
    }
    return value;
  }

  static String _stripTeacherCode(String teacherAndCode) {
    final List<String> names = teacherAndCode
        .split(RegExp(r'[,，、]'))
        .map((part) => part.replaceAll(RegExp(r'\([^)]*\)$'), '').trim())
        .where((part) => part.isNotEmpty)
        .toList();
    return names.join('、');
  }

  static String _weeksText(List<int> weeks) {
    final List<int> sorted = weeks.toSet().toList()..sort();
    if (sorted.isEmpty) return '';
    final StringBuffer buffer = StringBuffer();
    int runStart = sorted.first;
    int prev = sorted.first;
    for (int i = 1; i <= sorted.length; i++) {
      final int? current = i < sorted.length ? sorted[i] : null;
      if (current != null && current == prev + 1) {
        prev = current;
        continue;
      }
      if (buffer.isNotEmpty) buffer.write(',');
      buffer.write(runStart == prev ? '$runStart' : '$runStart-$prev');
      if (current != null) {
        runStart = current;
        prev = current;
      }
    }
    return buffer.toString();
  }
}

enum _ExportAction { png, csv }

/// 方案条：横向 chips（方案 N）+ 新建。
class _PlanBar extends ConsumerWidget {
  const _PlanBar({required this.notifier, required this.state});

  final ScheduleStoreNotifier notifier;
  final ScheduleState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Row(
      children: <Widget>[
        Expanded(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: <Widget>[
                for (final PkPlan plan in state.plans)
                  Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: _PlanChip(
                      label: plan.name,
                      active: plan.id == state.activePlanId,
                      onTap: () => notifier.switchPlan(plan.id),
                      onLongPress: () =>
                          _showPlanMenu(context, notifier, state, plan),
                    ),
                  ),
              ],
            ),
          ),
        ),
        GfIconButton(
          icon: Icons.add_circle_outline,
          size: 40,
          iconSize: 22,
          tooltip: l10n.schedulePlanNew,
          onPressed: state.plans.length >= kMaxPlans
              ? null
              : notifier.createPlan,
        ),
      ],
    );
  }
}

class _PlanChip extends StatelessWidget {
  const _PlanChip({
    required this.label,
    required this.active,
    required this.onTap,
    required this.onLongPress,
  });

  final String label;
  final bool active;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    return Material(
      color: active ? colors.primary : colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.selector),
        side: active ? BorderSide.none : BorderSide(color: colors.line),
      ),
      child: InkWell(
        onTap: onTap,
        onLongPress: onLongPress,
        borderRadius: BorderRadius.circular(radii.selector),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          child: Text(
            label,
            style: TextStyle(
              color: active ? colors.primaryContent : colors.baseContent,
              fontSize: 13,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

Future<void> _showPlanMenu(
  BuildContext context,
  ScheduleStoreNotifier notifier,
  ScheduleState state,
  PkPlan plan,
) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  await showGfBottomSheet<void>(
    context,
    builder: (BuildContext sheetContext) => SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Text(
              plan.name,
              style: GfTheme.typographyOf(sheetContext).title3,
            ),
          ),
          GfMenuItem(
            label: l10n.schedulePlanRename,
            icon: Icons.edit_outlined,
            onTap: () {
              Navigator.of(sheetContext).pop();
              _promptRename(context, notifier, plan);
            },
          ),
          GfMenuItem(
            label: l10n.schedulePlanClear,
            icon: Icons.delete_sweep_outlined,
            onTap: () {
              Navigator.of(sheetContext).pop();
              notifier.clearActivePlan();
            },
          ),
          GfMenuItem(
            label: l10n.schedulePlanDelete,
            icon: Icons.delete_outline,
            variant: GfMenuItemVariant.danger,
            onTap: () {
              Navigator.of(sheetContext).pop();
              _confirmDelete(context, notifier, plan);
            },
          ),
          const SizedBox(height: 4),
        ],
      ),
    ),
  );
}

Future<void> _promptRename(
  BuildContext context,
  ScheduleStoreNotifier notifier,
  PkPlan plan,
) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final TextEditingController controller = TextEditingController(
    text: plan.name,
  );
  await showGfBottomSheet<void>(
    context,
    keyboardAware: true,
    builder: (BuildContext sheetContext) => SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text(
              l10n.schedulePlanRename,
              style: GfTheme.typographyOf(sheetContext).heading,
            ),
            const SizedBox(height: 10),
            GfInput(
              controller: controller,
              autofocus: true,
              onSubmitted: (String value) {
                notifier.renamePlan(plan.id, value);
                Navigator.of(sheetContext).pop();
              },
            ),
            const SizedBox(height: 12),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: <Widget>[
                GfButton(
                  label: l10n.commonCancel,
                  variant: GfButtonVariant.ghost,
                  onPressed: () => Navigator.of(sheetContext).pop(),
                ),
                const SizedBox(width: 8),
                GfButton(
                  label: l10n.commonSave,
                  onPressed: () {
                    notifier.renamePlan(plan.id, controller.text);
                    Navigator.of(sheetContext).pop();
                  },
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
}

Future<void> _confirmDelete(
  BuildContext context,
  ScheduleStoreNotifier notifier,
  PkPlan plan,
) async {
  final AppLocalizations l10n = AppLocalizations.of(context);
  await showGfAlertDialog<void>(
    context,
    builder: (BuildContext sheetContext) => Padding(
      padding: const EdgeInsets.fromLTRB(20, 20, 20, 8),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            l10n.schedulePlanDelete,
            style: GfTheme.typographyOf(sheetContext).heading,
          ),
          const SizedBox(height: 8),
          Text(plan.name, style: GfTheme.typographyOf(sheetContext).body),
          const SizedBox(height: 18),
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: <Widget>[
              GfButton(
                label: l10n.commonCancel,
                variant: GfButtonVariant.ghost,
                onPressed: () => Navigator.of(sheetContext).pop(),
              ),
              const SizedBox(width: 8),
              GfButton(
                label: l10n.schedulePlanDelete,
                variant: GfButtonVariant.danger,
                onPressed: () {
                  notifier.deletePlan(plan.id);
                  Navigator.of(sheetContext).pop();
                },
              ),
            ],
          ),
        ],
      ),
    ),
  );
}

/// 折叠态配置行：单行摘要，点击展开。
class _CollapsedConfigRow extends StatelessWidget {
  const _CollapsedConfigRow({
    required this.calendarName,
    required this.state,
    required this.onTap,
  });

  final String calendarName;
  final ScheduleState state;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final PkMajorSelection sel = state.majorSelected;
    final String major = (sel.majorName ?? '').isNotEmpty
        ? sel.majorName!
        : (sel.major ?? '');
    final List<String> parts = <String>[
      if (calendarName.isNotEmpty) calendarName,
      if (sel.grade != null) l10n.scheduleGradeYear('${sel.grade}'),
      if (major.isNotEmpty) major,
    ];
    final String summary = parts.isEmpty
        ? '${l10n.scheduleTerm} · ${l10n.scheduleGrade} · ${l10n.scheduleMajor}'
        : parts.join(' · ');
    return Material(
      color: colors.base100,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.box),
        side: BorderSide(color: colors.line),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(radii.box),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          child: Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  summary,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: colors.baseContent.withValues(alpha: 0.75),
                  ),
                ),
              ),
              Icon(Icons.expand_more, size: 20, color: colors.iconMuted),
            ],
          ),
        ),
      ),
    );
  }
}

/// 展开态配置行：学期 / 年级 / 专业三个选择器（底部弹层）。
class _ConfigRow extends ConsumerWidget {
  const _ConfigRow({
    required this.calendars,
    required this.state,
    required this.notifier,
  });

  final List<PkCalendarItem> calendars;
  final ScheduleState state;
  final ScheduleStoreNotifier notifier;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final PkMajorSelection sel = state.majorSelected;
    return Container(
      decoration: BoxDecoration(
        color: GfTheme.colorsOf(context).base100,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
        border: Border.all(color: GfTheme.colorsOf(context).line),
      ),
      child: Row(
        children: <Widget>[
          Expanded(
            child: _ConfigSelector(
              label: l10n.scheduleTerm,
              value: _calendarName(sel.calendarId),
              placeholder: l10n.scheduleTerm,
              onTap: () => _pickTerm(context, ref, l10n, sel),
            ),
          ),
          _SelectorDivider(),
          Expanded(
            child: _ConfigSelector(
              label: l10n.scheduleGrade,
              value: sel.grade == null
                  ? ''
                  : l10n.scheduleGradeYear('${sel.grade}'),
              placeholder: l10n.scheduleGrade,
              onTap: sel.calendarId == null
                  ? null
                  : () => _pickGrade(context, ref, l10n, sel),
            ),
          ),
          _SelectorDivider(),
          Expanded(
            child: _ConfigSelector(
              label: l10n.scheduleMajor,
              value: (sel.majorName ?? '').isNotEmpty
                  ? sel.majorName!
                  : (sel.major ?? ''),
              placeholder: l10n.scheduleMajor,
              onTap: sel.calendarId == null || sel.grade == null
                  ? null
                  : () => _pickMajor(context, ref, l10n, sel),
            ),
          ),
        ],
      ),
    );
  }

  String _calendarName(int? calendarId) {
    for (final PkCalendarItem calendar in calendars) {
      if (calendar.calendarId == calendarId) return calendar.calendarName;
    }
    return '';
  }

  Future<void> _pickTerm(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    PkMajorSelection sel,
  ) async {
    final List<PkCalendarItem> options = calendars.isNotEmpty
        ? calendars
        : await _loadCalendars(context, ref, l10n);
    if (options.isEmpty || !context.mounted) return;
    final PkCalendarItem? picked = await showGfBottomSheet<PkCalendarItem>(
      context,
      builder: (BuildContext sheetContext) => _ListPickerSheet<PkCalendarItem>(
        title: l10n.scheduleTerm,
        items: options,
        labelOf: (PkCalendarItem item) => item.calendarName,
        selectedOf: (PkCalendarItem item) => item.calendarId == sel.calendarId,
      ),
    );
    if (picked == null || !context.mounted) return;
    ref
        .read(scheduleStoreProvider.notifier)
        .setMajorSelection(PkMajorSelection(calendarId: picked.calendarId));
  }

  Future<List<PkCalendarItem>> _loadCalendars(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) async {
    try {
      return await ref.read(pkRepositoryProvider).calendars();
    } catch (e) {
      if (context.mounted) {
        showGfToast(context, resolveErrorMessage(l10n, e), error: true);
      }
      return const <PkCalendarItem>[];
    }
  }

  Future<void> _pickGrade(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    PkMajorSelection sel,
  ) async {
    final PkGradeList grades;
    try {
      grades = await ref
          .read(pkRepositoryProvider)
          .grades(calendarId: sel.calendarId!);
    } catch (e) {
      if (context.mounted) {
        showGfToast(context, resolveErrorMessage(l10n, e), error: true);
      }
      return;
    }
    if (!context.mounted) return;
    final int? picked = await showGfBottomSheet<int>(
      context,
      builder: (BuildContext sheetContext) => _ListPickerSheet<int>(
        title: l10n.scheduleGrade,
        items: grades.gradeList,
        labelOf: (int grade) => l10n.scheduleGradeYear('$grade'),
        selectedOf: (int grade) => grade == sel.grade,
      ),
    );
    if (picked == null || !context.mounted) return;
    ref
        .read(scheduleStoreProvider.notifier)
        .setMajorSelection(
          PkMajorSelection(calendarId: sel.calendarId, grade: picked),
        );
  }

  Future<void> _pickMajor(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    PkMajorSelection sel,
  ) async {
    final List<PkMajor> majors;
    try {
      majors = await ref
          .read(pkRepositoryProvider)
          .majors(grade: sel.grade!, calendarId: sel.calendarId!);
    } catch (e) {
      if (context.mounted) {
        showGfToast(context, resolveErrorMessage(l10n, e), error: true);
      }
      return;
    }
    if (!context.mounted) return;
    final PkMajor? picked = await showGfBottomSheet<PkMajor>(
      context,
      builder: (BuildContext sheetContext) => _ListPickerSheet<PkMajor>(
        title: l10n.scheduleMajor,
        items: majors,
        labelOf: (PkMajor major) => major.name,
        selectedOf: (PkMajor major) => major.code == sel.major,
      ),
    );
    if (picked == null || !context.mounted) return;
    ref
        .read(scheduleStoreProvider.notifier)
        .setMajorSelection(
          PkMajorSelection(
            calendarId: sel.calendarId,
            grade: sel.grade,
            major: picked.code,
            majorName: picked.name,
          ),
        );
  }
}

class _SelectorDivider extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 1,
      height: 28,
      color: GfTheme.colorsOf(context).line,
    );
  }
}

class _ConfigSelector extends StatelessWidget {
  const _ConfigSelector({
    required this.label,
    required this.value,
    required this.placeholder,
    required this.onTap,
  });

  final String label;
  final String value;
  final String placeholder;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool hasValue = value.isNotEmpty;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            Flexible(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  Text(
                    label,
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                      color: colors.iconMuted,
                    ),
                  ),
                  const SizedBox(height: 1),
                  Text(
                    hasValue ? value : placeholder,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: hasValue ? colors.baseContent : colors.iconMuted,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 2),
            Icon(Icons.arrow_drop_down, size: 18, color: colors.iconMuted),
          ],
        ),
      ),
    );
  }
}

/// 通用单项选择 sheet。
class _ListPickerSheet<T> extends StatelessWidget {
  const _ListPickerSheet({
    required this.title,
    required this.items,
    required this.labelOf,
    required this.selectedOf,
  });

  final String title;
  final List<T> items;
  final String Function(T item) labelOf;
  final bool Function(T item) selectedOf;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Text(title, style: GfTheme.typographyOf(context).title3),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              children: <Widget>[
                for (final T item in items)
                  ListTile(
                    dense: true,
                    title: Text(
                      labelOf(item),
                      style: TextStyle(fontSize: 14, color: colors.baseContent),
                    ),
                    trailing: selectedOf(item)
                        ? Icon(Icons.check, size: 18, color: colors.primary)
                        : null,
                    onTap: () => Navigator.of(context).pop(item),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 4),
        ],
      ),
    );
  }
}

/// 过期横幅：教务数据已更新，点击同步。
class _DataOutdatedBanner extends StatelessWidget {
  const _DataOutdatedBanner({required this.loading, required this.onTap});

  final bool loading;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    return Material(
      color: colors.warning.withValues(alpha: 0.10),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.box),
        side: BorderSide(color: colors.warning.withValues(alpha: 0.35)),
      ),
      child: InkWell(
        onTap: loading ? null : onTap,
        borderRadius: BorderRadius.circular(radii.box),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          child: Row(
            children: <Widget>[
              Icon(Icons.sync_problem, size: 18, color: colors.warning),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.scheduleDataOutdated,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: colors.baseContent.withValues(alpha: 0.8),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              if (loading)
                const GfLoadingIndicator(small: true)
              else
                Text(
                  l10n.scheduleSyncLatest,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                    color: colors.primary,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 主 tab：课表 / 选课。
class _ScheduleTabs extends StatelessWidget {
  const _ScheduleTabs({
    required this.selectedTimetable,
    required this.onSelected,
  });

  final bool selectedTimetable;
  final ValueChanged<bool> onSelected;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return GfTabBar(
      tabs: <GfTab>[
        GfTab(label: l10n.scheduleTabTimetable, value: true),
        GfTab(label: l10n.scheduleTabPick, value: false),
      ],
      selected: selectedTimetable,
      onSelected: (Object value) => onSelected(value as bool),
    );
  }
}

/// 课表 tab（周次过滤 + 自绘网格）。
class _TimetableTab extends ConsumerWidget {
  const _TimetableTab({
    required this.boundaryKey,
    required this.sectionOverrides,
  });

  final GlobalKey boundaryKey;
  final List<SectionTime> sectionOverrides;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ScheduleState state = ref.watch(scheduleStoreProvider);
    final ScheduleGridData grid = state.grid;
    final GfColors colors = GfTheme.colorsOf(context);

    final int maxRows = maxRowsForCalendar(state.majorSelected.calendarId);
    final int rows = grid.cellCourses.length;
    final List<SectionTime> times = sectionTimesFor(
      maxRows,
      sectionOverrides.isEmpty ? null : sectionOverrides,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: _WeekFilter(
            week: state.weekView.week,
            onChanged: (int? week) => ref
                .read(scheduleStoreProvider.notifier)
                .setWeekView(PkWeekView(week: week, useCurrent: false)),
          ),
        ),
        RepaintBoundary(
          key: boundaryKey,
          child: Container(
            decoration: BoxDecoration(
              color: colors.base100,
              borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
              border: Border.all(color: colors.line),
            ),
            clipBehavior: Clip.antiAlias,
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: SizedBox(
                width: _kTimeColumnWidth + 7 * _kDayColumnWidth,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    _DayHeaderRow(
                      timeColumnWidth: _kTimeColumnWidth,
                      dayColumnWidth: _kDayColumnWidth,
                    ),
                    SizedBox(
                      height: grid.rowHeights.fold<double>(
                        0,
                        (sum, h) => sum + h,
                      ),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: <Widget>[
                          SizedBox(
                            width: _kTimeColumnWidth,
                            child: Column(
                              children: <Widget>[
                                for (int row = 0; row < rows; row++)
                                  SizedBox(
                                    height: grid.rowHeights[row].toDouble(),
                                    child: _TimeCell(row: row, times: times),
                                  ),
                              ],
                            ),
                          ),
                          for (int day = 0; day < 7; day++)
                            SizedBox(
                              width: _kDayColumnWidth,
                              child: _DayColumn(
                                day: day,
                                grid: grid,
                                conflicts: state.grid.conflicts,
                                onTapEmptyCell: (int section) =>
                                    _openCellPicker(
                                      context,
                                      ref,
                                      day + 1,
                                      section,
                                    ),
                                onTapCourse: (PkCourseOnTable course) =>
                                    _openCourseDetail(context, ref, course),
                              ),
                            ),
                        ],
                      ),
                    ),
                    if (state.stats.courseCount == 0)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 10),
                        child: Text(
                          l10n.commonEmpty,
                          textAlign: TextAlign.center,
                          style: TextStyle(
                            fontSize: 12,
                            color: colors.iconMuted,
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }

  static const double _kTimeColumnWidth = 44;
  static const double _kDayColumnWidth = 62;

  Future<void> _openCellPicker(
    BuildContext context,
    WidgetRef ref,
    int day,
    int section,
  ) async {
    final ScheduleStoreNotifier notifier = ref.read(
      scheduleStoreProvider.notifier,
    );
    final ScheduleState state = ref.read(scheduleStoreProvider);
    if (!notifier.isMajorSelected) return;
    final PkCoursesByTimeResult result;
    try {
      result = await ref
          .read(pkRepositoryProvider)
          .coursesByTime(
            calendarId: state.majorSelected.calendarId!,
            day: day,
            section: section,
          );
    } catch (e) {
      if (context.mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), e),
          error: true,
        );
      }
      return;
    }
    if (!context.mounted) return;
    await showGfBottomSheet<void>(
      context,
      builder: (BuildContext sheetContext) =>
          _CellPickerSheet(day: day, section: section, result: result),
    );
  }

  Future<void> _openCourseDetail(
    BuildContext context,
    WidgetRef ref,
    PkCourseOnTable course,
  ) async {
    if (course.code.startsWith(kCustomEventCodePrefix)) return;
    await showGfBottomSheet<void>(
      context,
      builder: (BuildContext sheetContext) =>
          _CourseDetailSheet(course: course),
    );
  }
}

/// 天头部（周一..周日）。
class _DayHeaderRow extends StatelessWidget {
  const _DayHeaderRow({
    required this.timeColumnWidth,
    required this.dayColumnWidth,
  });

  final double timeColumnWidth;
  final double dayColumnWidth;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final List<String> days = <String>[
      l10n.scheduleDayMon,
      l10n.scheduleDayTue,
      l10n.scheduleDayWed,
      l10n.scheduleDayThu,
      l10n.scheduleDayFri,
      l10n.scheduleDaySat,
      l10n.scheduleDaySun,
    ];
    return Container(
      height: 32,
      decoration: BoxDecoration(
        color: colors.base200.withValues(alpha: 0.7),
        border: Border(bottom: BorderSide(color: colors.line)),
      ),
      child: Row(
        children: <Widget>[
          SizedBox(width: timeColumnWidth),
          for (final String day in days)
            SizedBox(
              width: dayColumnWidth,
              child: Center(
                child: Text(
                  day,
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: colors.baseContent.withValues(alpha: 0.7),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _TimeCell extends StatelessWidget {
  const _TimeCell({required this.row, required this.times});

  final int row;
  final List<SectionTime> times;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final SectionTime? time = row < times.length ? times[row] : null;
    final String? partKey = dayPartKeyForRow(row + 1, times);
    final String? partLabel = switch (partKey) {
      'morning' => l10n.scheduleMorning,
      'afternoon' => l10n.scheduleAfternoon,
      'evening' => l10n.scheduleEvening,
      _ => null,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 1, vertical: 3),
      decoration: BoxDecoration(
        border: Border(
          right: BorderSide(color: colors.line),
          bottom: BorderSide(color: colors.line),
        ),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          if (partLabel != null)
            Text(
              partLabel,
              style: TextStyle(
                fontSize: 8,
                height: 1.1,
                fontWeight: FontWeight.w700,
                color: colors.primary.withValues(alpha: 0.8),
              ),
            ),
          Text(
            '${row + 1}',
            style: TextStyle(
              fontSize: 11,
              height: 1.1,
              fontWeight: FontWeight.w700,
              color: colors.baseContent.withValues(alpha: 0.7),
            ),
          ),
          if (time != null)
            Text(
              '${time.start}\n${time.end}',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 7,
                height: 1.1,
                color: colors.baseContent.withValues(alpha: 0.45),
              ),
            ),
        ],
      ),
    );
  }
}

/// 单天列：Stack 分层 —— 行发丝线底 + 空格点击层 + 课程卡（行高累计定位）。
class _DayColumn extends StatelessWidget {
  const _DayColumn({
    required this.day,
    required this.grid,
    required this.conflicts,
    required this.onTapEmptyCell,
    required this.onTapCourse,
  });

  final int day;
  final ScheduleGridData grid;
  final Map<String, List<PkConflictItem>> conflicts;
  final void Function(int section) onTapEmptyCell;
  final void Function(PkCourseOnTable course) onTapCourse;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final int rows = grid.rowHeights.length;
    final List<Widget> children = <Widget>[];

    double yOf(int row) {
      double total = 0;
      for (int i = 0; i < row; i++) {
        total += grid.rowHeights[i].toDouble();
      }
      return total;
    }

    // 行背景（发丝线）。
    for (int row = 0; row < rows; row++) {
      children.add(
        Positioned(
          top: yOf(row),
          left: 0,
          right: 0,
          height: grid.rowHeights[row].toDouble(),
          child: IgnorePointer(
            child: Container(
              decoration: BoxDecoration(
                border: Border(
                  right: BorderSide(color: colors.line),
                  bottom: BorderSide(color: colors.line),
                ),
              ),
            ),
          ),
        ),
      );
    }

    // 单元格：课程卡 or 空格点击层（跨行簇只渲染锚点格）。
    for (int row = 0; row < rows; row++) {
      final List<PkCourseOnTable> courses = grid.cellCourses[row][day];
      final bool covered = grid.occupiedGrid[row][day];
      if (covered) continue;
      if (courses.isNotEmpty) {
        final int span = grid.cellSpans[row][day] < 1
            ? 1
            : grid.cellSpans[row][day];
        double height = 0;
        for (int i = 0; i < span && row + i < rows; i++) {
          height += grid.rowHeights[row + i].toDouble();
        }
        children.add(
          Positioned(
            top: yOf(row),
            left: 1,
            right: 1,
            height: height,
            child: _CourseCell(
              courses: courses,
              conflicts: conflicts,
              onTapCourse: onTapCourse,
            ),
          ),
        );
      } else {
        children.add(
          Positioned(
            top: yOf(row),
            left: 0,
            right: 0,
            height: grid.rowHeights[row].toDouble(),
            child: GestureDetector(
              onTap: () => onTapEmptyCell(row + 1),
              behavior: HitTestBehavior.opaque,
              child: const SizedBox.expand(),
            ),
          ),
        );
      }
    }

    return Stack(children: children);
  }
}

class _CourseCell extends StatelessWidget {
  const _CourseCell({
    required this.courses,
    required this.conflicts,
    required this.onTapCourse,
  });

  final List<PkCourseOnTable> courses;
  final Map<String, List<PkConflictItem>> conflicts;
  final void Function(PkCourseOnTable course) onTapCourse;

  @override
  Widget build(BuildContext context) {
    if (courses.length == 1) {
      return _CourseCard(
        course: courses.first,
        conflicts: conflicts,
        onTap: () => onTapCourse(courses.first),
      );
    }
    return Column(
      children: <Widget>[
        for (int i = 0; i < courses.length; i++)
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(top: 1, bottom: 1),
              child: _CourseCard(
                course: courses[i],
                conflicts: conflicts,
                onTap: () => onTapCourse(courses[i]),
              ),
            ),
          ),
      ],
    );
  }
}

/// 课程卡（课表格内）。
class _CourseCard extends StatelessWidget {
  const _CourseCard({
    required this.course,
    required this.conflicts,
    required this.onTap,
  });

  final PkCourseOnTable course;
  final Map<String, List<PkConflictItem>> conflicts;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool custom = course.code.startsWith(kCustomEventCodePrefix);
    final bool conflicted = _isConflicted(course);
    final Color fill = custom ? colors.base200 : _slotColor(context, course);
    final String name = course.courseName.isNotEmpty
        ? course.courseName
        : course.code;
    final String teacher = compactTeacherName(teacherNameOf(course), 2);
    final String weeksText = formatDisplayWeeks(
      course.occupyWeek,
      (PkWeekParity parity) => parity == PkWeekParity.odd
          ? l10n.scheduleParityOdd
          : l10n.scheduleParityEven,
      (String range) => l10n.scheduleWeeksN(range),
    );
    final String room = course.occupyRoom ?? '';

    return Container(
      decoration: BoxDecoration(
        color: fill,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(
          color: custom ? colors.line : colors.primary.withValues(alpha: 0.22),
        ),
      ),
      clipBehavior: Clip.antiAlias,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          child: Stack(
            children: <Widget>[
              Padding(
                padding: const EdgeInsets.fromLTRB(5, 3, 5, 3),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: <Widget>[
                    Text(
                      name,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.15,
                        fontWeight: FontWeight.w600,
                        color: colors.baseContent,
                      ),
                    ),
                    if (!custom && (room.isNotEmpty || teacher.isNotEmpty))
                      Padding(
                        padding: const EdgeInsets.only(top: 1),
                        child: Text(
                          room.isNotEmpty ? room : teacher,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 8,
                            height: 1.1,
                            color: colors.baseContent.withValues(alpha: 0.55),
                          ),
                        ),
                      ),
                    if (weeksText.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 1),
                        child: Text(
                          weeksText,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 8,
                            height: 1.1,
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
              if (conflicted && !custom)
                Positioned(
                  right: 2,
                  top: 2,
                  child: Container(
                    width: 13,
                    height: 13,
                    decoration: BoxDecoration(
                      color: colors.error.withValues(alpha: 0.15),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: colors.error.withValues(alpha: 0.5),
                      ),
                    ),
                    child: Icon(
                      Icons.warning_amber_rounded,
                      size: 9,
                      color: colors.error,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  /// 冲突判定：deriveConflicts 以基础课号/custom 伪课号为键。
  bool _isConflicted(PkCourseOnTable course) {
    final String base = course.code.startsWith(kCustomEventCodePrefix)
        ? course.code
        : getCourseBaseCode(course.code);
    return (conflicts[base] ?? const <PkConflictItem>[]).isNotEmpty;
  }

  Color _slotColor(BuildContext context, PkCourseOnTable course) {
    final int slot = courseColorSlotFor(
      course.code.isNotEmpty ? course.code : course.courseName,
    );
    final bool dark = Theme.of(context).brightness == Brightness.dark;
    const List<Color> light = <Color>[
      Color(0xFFE8EEFF),
      Color(0xFFDCF2FE),
      Color(0xFFDDF3E7),
      Color(0xFFFDF0D5),
      Color(0xFFFFE9E7),
      Color(0xFFF1E8FB),
      Color(0xFFE0F2FE),
      Color(0xFFFFECF1),
    ];
    const List<Color> darkColors = <Color>[
      Color(0xFF1E2A4A),
      Color(0xFF12303F),
      Color(0xFF123527),
      Color(0xFF3A2F10),
      Color(0xFF482029),
      Color(0xFF2E1D40),
      Color(0xFF12303F),
      Color(0xFF402030),
    ];
    return (dark ? darkColors : light)[(slot - 1) % 8];
  }
}

/// 周次过滤控件。
class _WeekFilter extends StatelessWidget {
  const _WeekFilter({required this.week, required this.onChanged});

  final int? week;
  final ValueChanged<int?> onChanged;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    return Container(
      padding: const EdgeInsets.only(left: 12, right: 4),
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).field),
        border: Border.all(color: colors.line),
      ),
      child: Row(
        children: <Widget>[
          Icon(Icons.date_range_outlined, size: 16, color: colors.iconMuted),
          const SizedBox(width: 8),
          Expanded(
            child: DropdownButtonHideUnderline(
              child: DropdownButton<int?>(
                value: week,
                isExpanded: true,
                isDense: true,
                dropdownColor: colors.base100,
                items: <DropdownMenuItem<int?>>[
                  DropdownMenuItem<int?>(
                    value: null,
                    child: Text(
                      l10n.scheduleWeekAll,
                      style: TextStyle(fontSize: 13, color: colors.baseContent),
                    ),
                  ),
                  for (int w = 1; w <= kMaxWeek; w++)
                    DropdownMenuItem<int?>(
                      value: w,
                      child: Text(
                        l10n.scheduleWeekN(w),
                        style: TextStyle(
                          fontSize: 13,
                          color: colors.baseContent,
                        ),
                      ),
                    ),
                ],
                onChanged: (int? value) => onChanged(value),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// 底部统计卡（当前方案）。
class _StatsFooter extends StatelessWidget {
  const _StatsFooter({required this.state});

  final ScheduleState state;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final ScheduleStats stats = state.stats;
    final String creditText = stats.totalCredit.toStringAsFixed(
      stats.totalCredit % 1 == 0 ? 0 : 1,
    );
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 14),
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
        border: Border.all(color: colors.line),
      ),
      child: Row(
        children: <Widget>[
          _StatItem(
            value: l10n.scheduleStatsCourses(stats.courseCount),
            label: '',
          ),
          _StatDivider(),
          _StatItem(value: creditText, label: l10n.scheduleStatsCredits),
          _StatDivider(),
          _StatItem(
            value: '${stats.totalHours}',
            label: l10n.scheduleStatsHours,
          ),
          _StatDivider(),
          _StatItem(
            value: '${stats.conflictCount}',
            label: l10n.scheduleStatsConflicts,
            highlight: stats.conflictCount > 0,
          ),
        ],
      ),
    );
  }
}

class _StatItem extends StatelessWidget {
  const _StatItem({
    required this.value,
    required this.label,
    this.highlight = false,
  });

  final String value;
  final String label;
  final bool highlight;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Expanded(
      child: Column(
        children: <Widget>[
          Text(
            value,
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w700,
              color: highlight ? colors.error : colors.baseContent,
            ),
          ),
          if (label.isNotEmpty)
            Text(
              label,
              style: TextStyle(
                fontSize: 10,
                color: colors.baseContent.withValues(alpha: 0.5),
              ),
            ),
        ],
      ),
    );
  }
}

class _StatDivider extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 1,
      height: 24,
      color: GfTheme.colorsOf(context).line,
    );
  }
}

/// 选课 tab：必修 / 选修 / 搜索。
class _PickTab extends ConsumerStatefulWidget {
  const _PickTab();

  @override
  ConsumerState<_PickTab> createState() => _PickTabState();
}

class _PickTabState extends ConsumerState<_PickTab> {
  int _segment = 0;
  String _query = '';
  Timer? _debounce;
  bool _loading = false;
  String? _error;

  List<PkCourseByMajorItem> _compulsory = const <PkCourseByMajorItem>[];
  List<PkCourseByNatureItem> _optionalGroups = const <PkCourseByNatureItem>[];
  List<PkSearchCourseItem> _searchResults = const <PkSearchCourseItem>[];

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(_loadCurrentSegment);
  }

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  Future<void> _loadCurrentSegment() async {
    final ScheduleState state = ref.read(scheduleStoreProvider);
    if (!_hasMajorSelection(state)) {
      setState(() => _loading = false);
      return;
    }
    if (_loading) return;
    setState(() => _loading = true);
    try {
      if (_segment == 0) {
        final List<PkCourseByMajorItem> courses = await ref
            .read(pkRepositoryProvider)
            .coursesByMajor(
              grade: state.majorSelected.grade!,
              code: state.majorSelected.major!,
              calendarId: state.majorSelected.calendarId!,
            );
        if (!mounted) return;
        setState(() {
          _compulsory = courses;
          _loading = false;
          _error = null;
        });
      } else if (_segment == 1) {
        final List<PkOptionalType> types = await ref
            .read(pkRepositoryProvider)
            .optionalTypes(calendarId: state.majorSelected.calendarId!);
        final List<PkCourseByNatureItem> groups = await ref
            .read(pkRepositoryProvider)
            .coursesByNature(
              calendarId: state.majorSelected.calendarId!,
              ids: types.map((PkOptionalType t) => t.courseLabelId).toList(),
            );
        if (!mounted) return;
        setState(() {
          _optionalGroups = groups;
          _loading = false;
          _error = null;
        });
      } else {
        await _runSearch(_query);
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = resolveErrorMessage(AppLocalizations.of(context), e);
      });
    }
  }

  bool _hasMajorSelection(ScheduleState state) {
    return state.majorSelected.calendarId != null &&
        state.majorSelected.grade != null &&
        state.majorSelected.major != null &&
        state.majorSelected.major!.isNotEmpty;
  }

  Future<void> _runSearch(String query) async {
    final ScheduleState state = ref.read(scheduleStoreProvider);
    if (!_hasMajorSelection(state)) return;
    if (query.trim().isEmpty) {
      setState(() {
        _searchResults = const <PkSearchCourseItem>[];
        _loading = false;
        _error = null;
      });
      return;
    }
    setState(() => _loading = true);
    try {
      final PkSearchResult result = await ref
          .read(pkRepositoryProvider)
          .searchCourses(
            calendarId: state.majorSelected.calendarId!,
            courseName: query.trim(),
          );
      if (!mounted) return;
      setState(() {
        _searchResults = result.courses;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = resolveErrorMessage(AppLocalizations.of(context), e);
      });
    }
  }

  void _switchSegment(int segment) {
    if (_segment == segment) return;
    setState(() => _segment = segment);
    _loadCurrentSegment();
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ScheduleState state = ref.watch(scheduleStoreProvider);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        GfTabBar(
          tabs: [
            GfTab(label: l10n.scheduleCompulsory, value: 0),
            GfTab(label: l10n.scheduleOptional, value: 1),
            GfTab(label: l10n.commonSearch, value: 2),
          ],
          selected: _segment,
          onSelected: (value) => _switchSegment(value as int),
        ),
        const SizedBox(height: 6),
        if (!_hasMajorSelection(state))
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 40),
            child: GfEmpty(
              message: l10n.schedulePickClass,
              description:
                  '${l10n.scheduleTerm} · ${l10n.scheduleGrade} · ${l10n.scheduleMajor}',
            ),
          )
        else if (_loading)
          const Padding(
            padding: EdgeInsets.all(28),
            child: Center(child: GfLoadingIndicator()),
          )
        else if (_error != null)
          GfErrorRetry(message: _error!, onRetry: _loadCurrentSegment)
        else if (_segment == 0)
          _CourseList(
            emptyHint: l10n.commonEmpty,
            children: <Widget>[
              for (final PkCourseByMajorItem course in _compulsory)
                _CourseRow(
                  key: ValueKey<String>('comp-${course.courseCode}'),
                  name: course.courseName,
                  code: course.courseCode,
                  credit: course.credit,
                  faculty: course.faculty,
                  teacherText: _majorTeacherText(course),
                  onTap: () => _openClasses(
                    context,
                    course.courseCode,
                    courseName: course.courseName,
                  ),
                ),
            ],
          )
        else if (_segment == 1)
          _CourseList(
            emptyHint: l10n.commonEmpty,
            children: <Widget>[
              for (final PkCourseByNatureItem group
                  in _optionalGroups) ...<Widget>[
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 10, 12, 4),
                  child: Text(
                    group.courseLabelName,
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w700,
                      color: GfTheme.colorsOf(
                        context,
                      ).baseContent.withValues(alpha: 0.7),
                    ),
                  ),
                ),
                for (final PkNatureCourseItem course in group.courses)
                  _CourseRow(
                    key: ValueKey<String>('opt-${course.courseCode}'),
                    name: course.courseName,
                    code: course.courseCode,
                    credit: course.credit,
                    faculty: course.faculty,
                    teacherText: '',
                    onTap: () => _openClasses(
                      context,
                      course.courseCode,
                      courseName: course.courseName,
                    ),
                  ),
              ],
            ],
          )
        else
          _SearchPane(
            query: _query,
            results: _searchResults,
            onQueryChanged: (String value) {
              setState(() => _query = value);
              _debounce?.cancel();
              _debounce = Timer(const Duration(milliseconds: 350), () {
                _runSearch(value);
              });
            },
            onPick: (PkSearchCourseItem course) => _pickSearchCourse(course),
          ),
      ],
    );
  }

  String _majorTeacherText(PkCourseByMajorItem course) {
    final List<String> names = <String>[];
    for (final PkCourseClassItem item in course.courses) {
      for (final PkTeacherRef teacher in item.teachers) {
        if (teacher.teacherName.isNotEmpty) names.add(teacher.teacherName);
      }
    }
    return names.toSet().join('、');
  }

  Future<void> _openClasses(
    BuildContext context,
    String courseCode, {
    String? courseName,
  }) async {
    final ScheduleState state = ref.read(scheduleStoreProvider);
    final int calendarId = state.majorSelected.calendarId!;
    final List<PkCourseDetailBrief> briefs;
    try {
      briefs = await ref
          .read(pkRepositoryProvider)
          .courseDetails(calendarId: calendarId, courseCode: courseCode);
    } catch (e) {
      if (context.mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), e),
          error: true,
        );
      }
      return;
    }
    if (!context.mounted) return;
    await showGfBottomSheet<void>(
      context,
      builder: (BuildContext sheetContext) =>
          _ClassListSheet(briefs: briefs, courseName: courseName ?? courseCode),
    );
  }

  Future<void> _pickSearchCourse(PkSearchCourseItem course) async {
    final ScheduleState state = ref.read(scheduleStoreProvider);
    final int calendarId = state.majorSelected.calendarId!;
    final List<PkCourseDetailBrief> briefs;
    try {
      briefs = await ref
          .read(pkRepositoryProvider)
          .courseDetails(calendarId: calendarId, courseCode: course.courseCode);
    } catch (e) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), e),
          error: true,
        );
      }
      return;
    }
    if (!mounted || briefs.isEmpty) return;
    await showGfBottomSheet<void>(
      context,
      builder: (BuildContext sheetContext) =>
          _ClassListSheet(briefs: briefs, courseName: course.courseName),
    );
  }
}

/// 选课列表容器。
class _CourseList extends StatelessWidget {
  const _CourseList({required this.emptyHint, required this.children});

  final String emptyHint;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    if (children.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 36),
        child: GfEmpty(message: emptyHint),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }
}

/// 课程行。
class _CourseRow extends StatelessWidget {
  const _CourseRow({
    super.key,
    required this.name,
    required this.code,
    required this.credit,
    required this.faculty,
    required this.teacherText,
    required this.onTap,
  });

  final String name;
  final String code;
  final double credit;
  final String faculty;
  final String teacherText;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final String creditText = credit.toStringAsFixed(credit % 1 == 0 ? 0 : 1);
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 11),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(color: colors.line.withValues(alpha: 0.7)),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    <String>[
                      code,
                      '$creditText ${l10n.scheduleStatsCredits}',
                      if (faculty.isNotEmpty) faculty,
                      if (teacherText.isNotEmpty) teacherText,
                    ].join(' · '),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 12,
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Icon(
                Icons.chevron_right,
                size: 18,
                color: colors.iconMuted,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 搜索面板。
class _SearchPane extends StatelessWidget {
  const _SearchPane({
    required this.query,
    required this.results,
    required this.onQueryChanged,
    required this.onPick,
  });

  final String query;
  final List<PkSearchCourseItem> results;
  final ValueChanged<String> onQueryChanged;
  final void Function(PkSearchCourseItem course) onPick;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        GfSearchField(
          hintText: l10n.scheduleSearchHint,
          clearLabel: l10n.courseCopyClearSearch,
          onChanged: onQueryChanged,
        ),
        const SizedBox(height: 6),
        if (results.isEmpty && query.trim().isNotEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 32),
            child: GfEmpty(message: l10n.commonEmpty),
          )
        else
          for (final PkSearchCourseItem course in results)
            _SearchRow(course: course, onTap: () => onPick(course)),
      ],
    );
  }
}

class _SearchRow extends StatelessWidget {
  const _SearchRow({required this.course, required this.onTap});

  final PkSearchCourseItem course;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final String creditText = course.credit.toStringAsFixed(
      course.credit % 1 == 0 ? 0 : 1,
    );
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 11),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(color: colors.line.withValues(alpha: 0.7)),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text(
              course.courseName,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 15,
                fontWeight: FontWeight.w600,
                color: colors.baseContent,
              ),
            ),
            const SizedBox(height: 2),
            Text(
              <String>[
                course.courseCode,
                '$creditText ${l10n.scheduleStatsCredits}',
                if (course.faculty.isNotEmpty) course.faculty,
              ].join(' · '),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 12,
                color: colors.baseContent.withValues(alpha: 0.55),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 教学班列表 sheet（点选即上表；stay-open 可连续加同课时段）。
class _ClassListSheet extends ConsumerWidget {
  const _ClassListSheet({required this.briefs, required this.courseName});

  final List<PkCourseDetailBrief> briefs;
  final String courseName;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final ScheduleState state = ref.watch(scheduleStoreProvider);
    final PkPlan active = state.plans.firstWhere(
      (PkPlan plan) => plan.id == state.activePlanId,
      orElse: () => state.plans.first,
    );
    final ScheduleStoreNotifier notifier = ref.read(
      scheduleStoreProvider.notifier,
    );
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Text(
              l10n.schedulePickClass,
              style: GfTheme.typographyOf(context).title3,
            ),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              children: <Widget>[
                if (briefs.isEmpty)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 28),
                    child: Center(
                      child: Text(
                        l10n.commonEmpty,
                        style: TextStyle(fontSize: 13, color: colors.iconMuted),
                      ),
                    ),
                  ),
                for (final PkCourseDetailBrief brief in briefs)
                  _ClassRow(
                    brief: brief,
                    selected: active.selectedCourses.contains(brief.code),
                    conflicts: findClassConflicts(
                      detailFromBrief(brief),
                      state.occupied,
                    ),
                    onTap: () {
                      final StageCourseResult result = notifier.selectClass(
                        detailFromBrief(brief),
                        courseName,
                      );
                      if (result.conflicts.isNotEmpty) {
                        showGfToast(context, l10n.scheduleConflictBadge);
                      } else {
                        showGfToast(context, l10n.scheduleSelected);
                      }
                    },
                  ),
              ],
            ),
          ),
          const SizedBox(height: 4),
        ],
      ),
    );
  }
}

class _ClassRow extends StatelessWidget {
  const _ClassRow({
    required this.brief,
    required this.selected,
    required this.conflicts,
    required this.onTap,
  });

  final PkCourseDetailBrief brief;
  final bool selected;
  final List<PkConflictItem> conflicts;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final bool hasConflict = !selected && conflicts.isNotEmpty;
    final List<String> teachers = brief.teachers
        .map((PkTeacherRef t) => t.teacherName)
        .where((String s) => s.isNotEmpty)
        .toList();
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        decoration: BoxDecoration(
          color: hasConflict ? colors.error.withValues(alpha: 0.04) : null,
          border: Border(
            left: hasConflict
                ? BorderSide(
                    color: colors.error.withValues(alpha: 0.8),
                    width: 3,
                  )
                : BorderSide.none,
            bottom: BorderSide(color: colors.line.withValues(alpha: 0.7)),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Flexible(
                        child: Text(
                          brief.code,
                          style: TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w700,
                            color: colors.baseContent,
                          ),
                        ),
                      ),
                      if (selected) ...<Widget>[
                        const SizedBox(width: 6),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 1,
                          ),
                          decoration: BoxDecoration(
                            color: colors.primary.withValues(alpha: 0.12),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(
                            l10n.scheduleSelected,
                            style: TextStyle(
                              fontSize: 9,
                              fontWeight: FontWeight.w700,
                              color: colors.primary,
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                  if (hasConflict) ...<Widget>[
                    const SizedBox(height: 4),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 7,
                        vertical: 3,
                      ),
                      decoration: BoxDecoration(
                        color: colors.error.withValues(alpha: 0.10),
                        borderRadius: BorderRadius.circular(6),
                        border: Border.all(
                          color: colors.error.withValues(alpha: 0.25),
                        ),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: <Widget>[
                          Icon(
                            Icons.warning_amber_rounded,
                            size: 13,
                            color: colors.error,
                          ),
                          const SizedBox(width: 4),
                          Flexible(
                            child: Text(
                              conflicts.length == 1
                                  ? l10n.scheduleConflictWith(
                                      conflicts.first.courseName.isNotEmpty
                                          ? conflicts.first.courseName
                                          : conflicts.first.code,
                                    )
                                  : l10n.scheduleConflictsWith(
                                      conflicts.first.courseName.isNotEmpty
                                          ? conflicts.first.courseName
                                          : conflicts.first.code,
                                      conflicts.length,
                                    ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.w600,
                                color: colors.error,
                              ),
                            ),
                          ),
                          const SizedBox(width: 6),
                          Text(
                            l10n.scheduleConflictCanAdd,
                            style: TextStyle(
                              fontSize: 9,
                              fontWeight: FontWeight.w700,
                              color: colors.error.withValues(alpha: 0.85),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                  const SizedBox(height: 3),
                  Text(
                    _timeSummary(context, brief),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 11,
                      height: 1.3,
                      color: hasConflict
                          ? colors.error.withValues(alpha: 0.85)
                          : colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                  if (teachers.isNotEmpty)
                    Text(
                      teachers.join('、'),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11,
                        color: colors.baseContent.withValues(alpha: 0.55),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Icon(
              selected ? Icons.check_circle : Icons.add_circle_outline,
              size: 20,
              color: selected
                  ? colors.primary
                  : (hasConflict ? colors.error : colors.primary),
            ),
          ],
        ),
      ),
    );
  }

  String _timeSummary(BuildContext context, PkCourseDetailBrief brief) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final List<String> days = <String>[
      l10n.scheduleDayMon,
      l10n.scheduleDayTue,
      l10n.scheduleDayWed,
      l10n.scheduleDayThu,
      l10n.scheduleDayFri,
      l10n.scheduleDaySat,
      l10n.scheduleDaySun,
    ];
    final List<String> parts = <String>[];
    for (final PkArrangementInfo a in brief.arrangementInfo) {
      final int? day = a.occupyDay;
      final List<int> times = a.occupyTime ?? const <int>[];
      if (day == null || day < 1 || day > 7 || times.isEmpty) continue;
      final String span = times.length == 1
          ? '${times.first}'
          : '${times.first}-${times.last}';
      final String weeks = formatWeeksText(a.occupyWeek);
      parts.add(
        [
          days[day - 1],
          l10n.schedulePeriodRange(span),
          if (weeks.isNotEmpty) l10n.scheduleWeeksN(weeks),
        ].join(' · '),
      );
    }
    return parts.take(3).join('; ');
  }
}

/// 空格选课 sheet（该时段全校课程）。
class _CellPickerSheet extends ConsumerWidget {
  const _CellPickerSheet({
    required this.day,
    required this.section,
    required this.result,
  });

  final int day;
  final int section;
  final PkCoursesByTimeResult result;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final List<String> days = <String>[
      l10n.scheduleDayMon,
      l10n.scheduleDayTue,
      l10n.scheduleDayWed,
      l10n.scheduleDayThu,
      l10n.scheduleDayFri,
      l10n.scheduleDaySat,
      l10n.scheduleDaySun,
    ];
    final ScheduleState state = ref.watch(scheduleStoreProvider);
    final ScheduleStoreNotifier notifier = ref.read(
      scheduleStoreProvider.notifier,
    );
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Text(
              '${days[day - 1]} · ${l10n.schedulePeriodRange('$section')}',
              style: GfTheme.typographyOf(context).title3,
            ),
          ),
          if (!result.auxiliaryReady)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 6),
              child: GfStatusMessage(message: l10n.scheduleDegraded),
            ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              children: <Widget>[
                if (result.courses.isEmpty)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 28),
                    child: Center(
                      child: Text(
                        l10n.commonEmpty,
                        style: TextStyle(fontSize: 13, color: colors.iconMuted),
                      ),
                    ),
                  ),
                for (final PkSearchCourseItem course in result.courses)
                  _CellCourseRow(
                    course: course,
                    onTap: () => _pick(context, ref, state, notifier, course),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 4),
        ],
      ),
    );
  }

  Future<void> _pick(
    BuildContext context,
    WidgetRef ref,
    ScheduleState state,
    ScheduleStoreNotifier notifier,
    PkSearchCourseItem course,
  ) async {
    final List<PkCourseDetailBrief> briefs;
    try {
      briefs = await ref
          .read(pkRepositoryProvider)
          .courseDetails(
            calendarId: state.majorSelected.calendarId!,
            courseCode: course.courseCode,
          );
    } catch (e) {
      if (context.mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), e),
          error: true,
        );
      }
      return;
    }
    if (!context.mounted) return;
    // 命中该时段的班级优先。
    PkCourseDetailBrief? target;
    for (final PkCourseDetailBrief brief in briefs) {
      final bool hit = brief.arrangementInfo.any(
        (PkArrangementInfo a) =>
            a.occupyDay == day && (a.occupyTime ?? <int>[]).contains(section),
      );
      if (hit) {
        target = brief;
        break;
      }
    }
    target ??= briefs.isNotEmpty ? briefs.first : null;
    if (target == null) return;
    final StageCourseResult addResult = notifier.selectClass(
      detailFromBrief(target),
      course.courseName,
    );
    if (!context.mounted) return;
    final AppLocalizations l10n = AppLocalizations.of(context);
    if (addResult.conflicts.isNotEmpty) {
      showGfToast(context, l10n.scheduleConflictBadge);
    } else {
      showGfToast(context, l10n.scheduleSelected);
    }
    Navigator.of(context).pop();
  }
}

class _CellCourseRow extends StatelessWidget {
  const _CellCourseRow({required this.course, required this.onTap});

  final PkSearchCourseItem course;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 11),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(color: colors.line.withValues(alpha: 0.7)),
          ),
        ),
        child: Row(
          children: <Widget>[
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    course.courseName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                      color: colors.baseContent,
                    ),
                  ),
                  Text(
                    '${course.courseCode} · ${course.faculty}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 11,
                      color: colors.baseContent.withValues(alpha: 0.55),
                    ),
                  ),
                ],
              ),
            ),
            Icon(Icons.add_circle_outline, size: 20, color: colors.primary),
          ],
        ),
      ),
    );
  }
}

/// 课程详情 sheet（课表课程点击；含 P13 课评摘要 + 退课）。
class _CourseDetailSheet extends ConsumerWidget {
  const _CourseDetailSheet({required this.course});

  final PkCourseOnTable course;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final ScheduleStoreNotifier notifier = ref.read(
      scheduleStoreProvider.notifier,
    );
    final String teacher = teacherNameOf(course);
    final String room = course.occupyRoom ?? '';
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      Text(
                        course.courseName,
                        style: GfTheme.typographyOf(context).title3,
                      ),
                      Text(
                        course.code,
                        style: TextStyle(
                          fontSize: 12,
                          color: colors.baseContent.withValues(alpha: 0.55),
                        ),
                      ),
                    ],
                  ),
                ),
                GfIconButton(
                  icon: Icons.close,
                  size: 36,
                  iconSize: 18,
                  tooltip: l10n.commonClose,
                  onPressed: () => Navigator.of(context).pop(),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              <String>[
                if (teacher.isNotEmpty) teacher,
                if (room.isNotEmpty) room,
              ].join(' · '),
              style: TextStyle(
                fontSize: 13,
                color: colors.baseContent.withValues(alpha: 0.7),
              ),
            ),
            const SizedBox(height: 6),
            if ((course.arrangementText ?? '').isNotEmpty)
              Text(
                course.arrangementText!,
                style: TextStyle(
                  fontSize: 12,
                  height: 1.4,
                  color: colors.baseContent.withValues(alpha: 0.55),
                ),
              ),
            const SizedBox(height: 12),
            _ReviewBriefPanel(course: course),
            const SizedBox(height: 12),
            GfButton(
              label: l10n.scheduleRemoveCourse,
              variant: GfButtonVariant.outline,
              expanded: true,
              onPressed: () {
                notifier.deselectClass(course.code);
                Navigator.of(context).pop();
              },
            ),
          ],
        ),
      ),
    );
  }
}

/// P13 课评摘要（异步加载；courseId==0 → scheduleNoReviewData）。
class _ReviewBriefPanel extends ConsumerStatefulWidget {
  const _ReviewBriefPanel({required this.course});

  final PkCourseOnTable course;

  @override
  ConsumerState<_ReviewBriefPanel> createState() => _ReviewBriefPanelState();
}

class _ReviewBriefPanelState extends ConsumerState<_ReviewBriefPanel> {
  bool _loading = true;
  PkReviewBrief? _brief;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final ScheduleState state = ref.read(scheduleStoreProvider);
    final PkPlan active = state.plans.firstWhere(
      (PkPlan plan) => plan.id == state.activePlanId,
      orElse: () => state.plans.first,
    );
    int? teachingClassId;
    for (final PkStagedCourse staged in active.stagedCourses) {
      if (staged.courseCode != getCourseBaseCode(widget.course.code)) continue;
      for (final PkCourseDetail detail in staged.courseDetail) {
        if (detail.code == widget.course.code) {
          teachingClassId = detail.teachingClassId;
          break;
        }
      }
    }
    try {
      final PkReviewBrief brief = await ref
          .read(pkRepositoryProvider)
          .courseReviewBrief(
            courseCode: getCourseBaseCode(widget.course.code),
            teacherName: teacherNameOf(widget.course),
            calendarId: state.majorSelected.calendarId,
            teachingClassId: teachingClassId,
          );
      if (!mounted) return;
      setState(() {
        _brief = brief;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final PkReviewBrief? brief = _brief;
    final String text;
    if (_loading) {
      text = l10n.scheduleNoReviewData;
    } else if (brief == null || brief.courseId == 0 || brief.reviewCount == 0) {
      text = l10n.scheduleNoReviewData;
    } else {
      final String avg = brief.ratingAvg == null
          ? ''
          : '${brief.ratingAvg!.toStringAsFixed(1)} ★';
      text = <String>[
        avg,
        l10n.coursesRatingCount(brief.reviewCount),
      ].where((String s) => s.isNotEmpty).join(' · ');
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 12,
          color: colors.baseContent.withValues(alpha: 0.65),
        ),
      ),
    );
  }
}

/// 自定义占位事件 sheet：星期 / 节次 / 周次 多选。
class _CustomEventSheet extends ConsumerStatefulWidget {
  const _CustomEventSheet();

  @override
  ConsumerState<_CustomEventSheet> createState() => _CustomEventSheetState();
}

class _CustomEventSheetState extends ConsumerState<_CustomEventSheet> {
  int _day = 1;
  final Set<int> _sections = <int>{};
  final Set<int> _weeks = <int>{1};

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final List<String> dayLabels = <String>[
      l10n.scheduleDayMon,
      l10n.scheduleDayTue,
      l10n.scheduleDayWed,
      l10n.scheduleDayThu,
      l10n.scheduleDayFri,
      l10n.scheduleDaySat,
      l10n.scheduleDaySun,
    ];
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.72,
        ),
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                l10n.scheduleAddCustomEvent,
                style: GfTheme.typographyOf(context).title3,
              ),
              const SizedBox(height: 12),
              Text('${l10n.scheduleTerm}：', style: _groupLabelStyle(colors)),
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: <Widget>[
                  for (int i = 0; i < 7; i++)
                    _ChoiceChip(
                      label: dayLabels[i],
                      selected: _day == i + 1,
                      onTap: () => setState(() => _day = i + 1),
                    ),
                ],
              ),
              const SizedBox(height: 12),
              Text(l10n.schedulePeriods, style: _groupLabelStyle(colors)),
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: <Widget>[
                  for (int section = 1; section <= 12; section++)
                    _ChoiceChip(
                      label: '$section',
                      selected: _sections.contains(section),
                      onTap: () => setState(() {
                        if (!_sections.remove(section)) {
                          _sections.add(section);
                        }
                      }),
                    ),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: <Widget>[
                  Text(
                    l10n.scheduleWeeksLabel,
                    style: _groupLabelStyle(colors),
                  ),
                  const SizedBox(width: 8),
                  Text(
                    _weeksTextPreview(),
                    style: TextStyle(
                      fontSize: 11,
                      color: colors.baseContent.withValues(alpha: 0.5),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: <Widget>[
                  for (int week = 1; week <= kMaxWeek; week++)
                    _ChoiceChip(
                      label: '$week',
                      selected: _weeks.contains(week),
                      onTap: () => setState(() {
                        if (!_weeks.remove(week)) {
                          _weeks.add(week);
                        }
                      }),
                    ),
                ],
              ),
              const SizedBox(height: 16),
              Row(
                children: <Widget>[
                  Expanded(
                    child: GfButton(
                      label: l10n.commonCancel,
                      variant: GfButtonVariant.ghost,
                      onPressed: () => Navigator.of(context).pop(),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: GfButton(
                      label: l10n.commonSave,
                      onPressed: _sections.isEmpty
                          ? null
                          : () {
                              ref
                                  .read(scheduleStoreProvider.notifier)
                                  .addCustomEvent(
                                    label: l10n.scheduleCustomEventLabel,
                                    day: _day,
                                    sections: _sections.toList(),
                                    weeks: _weeks.toList(),
                                  );
                              Navigator.of(context).pop();
                            },
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  TextStyle _groupLabelStyle(GfColors colors) => TextStyle(
    fontSize: 12,
    fontWeight: FontWeight.w600,
    color: colors.baseContent.withValues(alpha: 0.7),
  );

  String _weeksTextPreview() {
    if (_weeks.isEmpty) return '';
    final List<int> sorted = _weeks.toList()..sort();
    return AppLocalizations.of(
      context,
    ).scheduleWeeksN('${sorted.first}-${sorted.last}');
  }
}

class _ChoiceChip extends StatelessWidget {
  const _ChoiceChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: selected ? colors.primary : colors.base100,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: selected ? colors.primary : colors.line),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: selected ? colors.primaryContent : colors.baseContent,
          ),
        ),
      ),
    );
  }
}

/// 排课器加载骨架（等待 store 恢复持久化）。
class GfScheduleSkeleton extends StatelessWidget {
  const GfScheduleSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return ListView(
      physics: const NeverScrollableScrollPhysics(),
      padding: const EdgeInsets.all(12),
      children: const <Widget>[
        GfSkeleton(height: 40, radius: 8),
        SizedBox(height: 10),
        GfSkeleton(height: 34, radius: 8),
        SizedBox(height: 10),
        GfSkeleton(height: 220, radius: 8),
      ],
    );
  }
}
