import 'dart:async';
import 'dart:math' as math;

import 'package:core/core.dart';
import 'package:flutter/gestures.dart' show PointerScrollEvent;
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart' show ScrollDirection;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../current_user.dart';
import '../../providers.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../navigation/reading_window.dart';
import '../../widgets/app_refresh_indicator.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/logo_motion_loader.dart';
import '../../widgets/status_views.dart';
import '../../widgets/campus_shortcuts.dart';
import 'campus_connection.dart';
import 'campus_course_picks.dart';
import 'campus_settings_page.dart';
import 'campus_data_views.dart';
import 'campus_helpers.dart';
import 'campus_lifecycle.dart';
import 'campus_message_page.dart';
import 'campus_state.dart';
import 'timetable/campus_timetable.dart' show campusTimetableHeaderHeight;
import 'timetable/timetable_grid.dart';
import 'timetable/timetable_layout.dart';
import 'timetable/timetable_memo_store.dart' show TimetablePrefs;

/// Only navigation choices survive the private response view. This object is
/// never serialized and contains no school response, credential or notice body.
class _CampusNavigation {
  String tab = 'today';
  final queries = <String, String>{};
  final offsets = <String, double>{};
  int? week;
  int? account;
  String? binding;
  bool bindingObserved = false;
  bool identityRejected = false;

  /// Last home-screen widget tap already handled; [clear] keeps it so an
  /// identity change cannot replay an old tap.
  int widgetEntry = 0;

  void clear() {
    tab = 'today';
    queries.clear();
    offsets.clear();
    week = null;
    binding = null;
    bindingObserved = false;
    identityRejected = false;
  }
}

class CampusPage extends ConsumerStatefulWidget {
  const CampusPage({super.key});
  @override
  ConsumerState<CampusPage> createState() => _CampusPageState();
}

class _CampusPageState extends ConsumerState<CampusPage> {
  _CampusNavigation _navigation = _CampusNavigation();

  @override
  Widget build(BuildContext context) {
    _CampusNavigation fresh() =>
        _CampusNavigation()
          ..widgetEntry = ref.read(scheduleWidgetEntryProvider);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      setState(() => _navigation = fresh());
    });
    final repository = ref.watch(campusRepositoryProvider);
    ref.listen(campusRepositoryProvider, (_, _) {
      _navigation = fresh();
    });
    return KeyedSubtree(
      key: ObjectKey(repository),
      child: _CampusAccount(navigation: _navigation),
    );
  }
}

class _CampusAccount extends ConsumerWidget {
  const _CampusAccount({required this.navigation});
  final _CampusNavigation navigation;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    Widget publicSurface(Widget content) => RootSurface(
      title: l.campusTitle,
      showComposeAction: false,
      body: (top, bottom) => ListView(
        padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16),
        children: [
          const CampusShortcuts(),
          const SizedBox(height: 24),
          content,
          const SizedBox(height: 24),
          const CampusCoursePicks(),
        ],
      ),
    );
    return ref
        .watch(currentUserProvider)
        .when(
          loading: () => publicSurface(const GfLoading()),
          error: (_, _) => publicSurface(
            GfErrorRetry(
              message: l.campusUnavailable,
              onRetry: () => ref.invalidate(currentUserProvider),
            ),
          ),
          data: (user) {
            if (navigation.account != user?.id) {
              navigation.clear();
              navigation.account = user?.id;
            }
            return user == null
                ? publicSurface(
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        GfText(
                          l.campusOfficialSubtitle,
                          style: GfTheme.typographyOf(context).title2,
                        ),
                        const SizedBox(height: 16),
                        GfText(l.campusPrivacy),
                        const SizedBox(height: 24),
                        GfButton(
                          label: l.authLoginTitle,
                          onPressed: () => context.push('/login'),
                        ),
                      ],
                    ),
                  )
                : _CampusWorkspace(
                    key: ValueKey(user.id),
                    navigation: navigation,
                  );
          },
        );
  }
}

class _CampusWorkspace extends ConsumerStatefulWidget {
  const _CampusWorkspace({super.key, required this.navigation});
  final _CampusNavigation navigation;
  @override
  ConsumerState<_CampusWorkspace> createState() => _CampusWorkspaceState();
}

class _CampusWorkspaceState extends ConsumerState<_CampusWorkspace>
    with WidgetsBindingObserver {
  _CampusNavigation get _navigation => widget.navigation;
  String get _tab => _navigation.tab;
  String _queryFor(String tab) => _navigation.queries[tab] ?? '';
  bool _restoreScroll = true;
  int _wish = math.Random().nextInt(4);
  final _searches = <String, TextEditingController>{};
  final _scrolls = <String, GfScrollToTopController>{};
  final _scrollKeys = <String, GlobalKey>{};
  GfScrollToTopController _scrollFor(String tab) =>
      _scrolls.putIfAbsent(tab, GfScrollToTopController.new);
  GlobalKey _scrollKeyFor(String tab) => _scrollKeys.putIfAbsent(
    tab,
    () => GlobalKey(debugLabel: 'campus-scroll-$tab'),
  );
  TextEditingController _searchFor(String tab) => _searches.putIfAbsent(
    tab,
    () => TextEditingController(text: _queryFor(tab)),
  );
  GfScrollToTopController get _scroll => _scrollFor(_tab);
  late final GfTabScrollRegistry _registry;
  Timer? _clock;
  bool _foreground = true;
  bool? _visible;
  String? _swipeLoadingTab;
  List<SectionTime> _times = [];
  @override
  void initState() {
    super.initState();
    final lifecycle = WidgetsBinding.instance.lifecycleState;
    _foreground = isCampusForeground(lifecycle);
    WidgetsBinding.instance.addObserver(this);
    _searchFor(_tab);
    // A notice may have kept the controller alive while this page was hidden.
    // Verify its binding and refresh private data when the page returns.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _visible == true) _enter();
    });
    _registry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.campus, _scroll);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncVisibility();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // 仅 paused/hidden/detached 视为被覆盖（suspend + 丢弃隐私子树）；
    // inactive（通知栏/弹窗瞬时遮挡）保持现状，不再整页重载。
    final foreground = isCampusForeground(state);
    if (!foreground && _visible == true) {
      _restoreScroll = true;
      ref.read(campusControllerProvider.notifier).suspend();
    }
    _foreground = foreground;
    _syncVisibility();
    if (mounted) setState(() {});
  }

  void _syncVisibility() {
    final visible = _foreground && TickerMode.valuesOf(context).enabled;
    if (_visible == visible) return;
    final wasVisible = _visible;
    _visible = visible;
    if (!visible) {
      _swipeLoadingTab = null;
      _clock?.cancel();
      _clock = null;
      if (wasVisible == true) _restoreScroll = true;
      return;
    }
    _loadTimes();
    _clock = Timer.periodic(const Duration(minutes: 1), (_) {
      if (mounted && _visible == true) {
        setState(() {});
        unawaited(ref.read(campusControllerProvider.notifier).refreshVisible());
      }
    });
    if (wasVisible == false) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && _visible == true) _enter();
      });
    }
  }

  /// Page entry. A new home-screen widget tap opens the timetable from the
  /// device snapshot and refreshes behind it; other entries keep their tab.
  void _enter() {
    final controller = ref.read(campusControllerProvider.notifier);
    final entry = ref.read(scheduleWidgetEntryProvider);
    if (entry == _navigation.widgetEntry) {
      unawaited(controller.enterTab(_tab));
      return;
    }
    _navigation.widgetEntry = entry;
    _select('timetable', load: false);
    unawaited(controller.enterFromWidget('timetable'));
  }

  Future<void> _loadTimes() async {
    try {
      final value = await ref.read(pkRepositoryProvider).sectionTimes();
      if (mounted && value != null) {
        setState(
          () => _times = value.sectionTimes
              .map(
                (t) =>
                    SectionTime(section: t.section, start: t.start, end: t.end),
              )
              .toList(),
        );
      }
    } catch (_) {
      /* The planner's built-in section times are the fallback. */
    }
  }

  @override
  void dispose() {
    _clock?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    for (final search in _searches.values) {
      search.dispose();
    }
    _registry.unregister(GfShellDestination.campus, _scroll);
    super.dispose();
  }

  void _select(String tab, {bool fromSwipe = false, bool load = true}) {
    if (tab == _tab) return;
    FocusManager.instance.primaryFocus?.unfocus();
    _swipeLoadingTab = fromSwipe ? tab : null;
    setState(() {
      _navigation.tab = tab;
      _searchFor(tab).text = _queryFor(tab);
      _restoreScroll = true;
    });
    _registry.register(GfShellDestination.campus, _scrollFor(tab));
    if (load) {
      unawaited(ref.read(campusControllerProvider.notifier).loadTab(tab));
    }
  }

  Widget _section(String title, Widget child, {Widget? action}) => Padding(
    padding: const EdgeInsets.only(bottom: 28),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Wrap(
          alignment: WrapAlignment.spaceBetween,
          crossAxisAlignment: WrapCrossAlignment.center,
          spacing: 8,
          children: [
            GfText(title, style: GfTheme.typographyOf(context).title2),
            ?action,
          ],
        ),
        const SizedBox(height: 12),
        child,
      ],
    ),
  );

  /// The teaching week, advanced by the Mondays passed since a previewed
  /// snapshot was saved so an earlier week never shows as current.
  int _snapshotWeek(CampusViewState state) {
    final calendar = state.data['calendar'];
    final week = int.tryParse(campusMetric(calendar, '教学周')) ?? 1;
    final snapshot = state.snapshot;
    if (snapshot == null || !identical(calendar, snapshot.data['calendar'])) {
      return week;
    }
    DateTime monday(DateTime instant) {
      final day = campusNow(instant);
      return DateTime.utc(day.year, day.month, day.day - (day.weekday - 1));
    }

    final weeks =
        monday(
          DateTime.now(),
        ).difference(monday(snapshot.committedAt)).inDays ~/
        7;
    return week + math.max(0, weeks);
  }

  Widget _dataset(
    CampusViewState state,
    String key,
    Widget Function(CampusDataset) ready,
  ) {
    final l = AppLocalizations.of(context);
    final data = state.data[key];
    final error = state.errors[key];
    final loading = state.refreshing || state.fetching.contains(key);
    final usable =
        data != null && const {'ready', 'empty'}.contains(data.status);
    Widget refreshPrompt() => Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        GfText(l.campusDataNeedsRefresh),
        TextButton.icon(
          onPressed: loading
              ? null
              : () => ref.read(campusControllerProvider.notifier).refresh(),
          icon: const GfSymbol('refresh-cw', size: 18),
          label: GfText(l.commonRefresh),
        ),
      ],
    );
    final content = data == null
        ? (loading
              ? const ExcludeSemantics(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      GfSkeleton(height: 48, radius: 10),
                      SizedBox(height: 10),
                      GfSkeleton(height: 40, radius: 10),
                    ],
                  ),
                )
              : refreshPrompt())
        : !usable
        ? GfText(l.campusUnavailable)
        : data.status == 'empty' && key != 'today'
        ? GfText(l.campusNoData)
        : ready(data);
    if (error == null) return content;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfErrorRetry(
          message: campusError(l, error),
          onRetry: () => ref.read(campusControllerProvider.notifier).retry(key),
        ),
        // Explicitly invalid rules must not present a stale teaching-day result.
        if (usable &&
            !(error is ApiException &&
                const {
                  'campus.rulesUnavailable',
                  'campus.rulesInvalid',
                }.contains(error.messageCode)))
          content,
      ],
    );
  }

  Widget _snapshotNotice(CampusViewState state) {
    final l = AppLocalizations.of(context);
    final snapshot = state.snapshot;
    final stale =
        snapshot != null &&
        (DateTime.now().difference(snapshot.committedAt) >
                const Duration(days: 1) ||
            snapshot.data['today']?.teachingDay?.date !=
                campusDateKey(DateTime.now()));
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: snapshot == null
                    ? const SizedBox.shrink()
                    : GfText(
                        l.campusSnapshotUpdated(
                          DateFormat.yMd(
                            l.localeName,
                          ).add_Hm().format(snapshot.committedAt.toLocal()),
                        ),
                        style: GfTheme.typographyOf(context).caption.copyWith(
                          color: GfTheme.colorsOf(context).iconMuted,
                        ),
                      ),
              ),
              const SizedBox(width: 8),
              IconButton(
                tooltip: l.commonRefresh,
                onPressed: state.refreshing || state.busy
                    ? null
                    : () =>
                          ref.read(campusControllerProvider.notifier).refresh(),
                style: IconButton.styleFrom(
                  foregroundColor: GfTheme.colorsOf(context).primary,
                  minimumSize: const Size(44, 44),
                  padding: EdgeInsets.zero,
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                icon: const GfSymbol('refresh-cw', size: 18),
              ),
            ],
          ),
          if (state.errors.containsKey('status') && snapshot != null)
            GfText(l.campusSnapshotOffline)
          else if (state.error != null || state.errors.isNotEmpty)
            GfText(l.campusSnapshotRefreshFailed)
          else if (stale)
            GfText(l.campusSnapshotStale),
        ],
      ),
    );
  }

  /// The freshness facts of [_snapshotNotice], condensed for one line.
  CampusTimetableStatus? _timetableStatus(CampusViewState state) {
    final l = AppLocalizations.of(context);
    final snapshot = state.snapshot;
    final stale =
        snapshot != null &&
        (DateTime.now().difference(snapshot.committedAt) >
                const Duration(days: 1) ||
            snapshot.data['today']?.teachingDay?.date !=
                campusDateKey(DateTime.now()));
    final (String, String)? warning =
        state.errors.containsKey('status') && snapshot != null
        ? (l.campusSnapshotOffline, l.timetableShowingSnapshot)
        : state.error != null || state.errors.isNotEmpty
        ? (l.campusSnapshotRefreshFailed, l.timetablePartlyUpdated)
        : stale
        ? (l.campusSnapshotStale, l.timetableMayBeOutdated)
        : null;
    if (warning == null && snapshot == null) return null;
    final updated = snapshot == null
        ? null
        : l.campusSnapshotUpdated(
            DateFormat.yMd(
              l.localeName,
            ).add_Hm().format(snapshot.committedAt.toLocal()),
          );
    return CampusTimetableStatus(
      warning?.$2 ??
          l.timetableUpdatedAt(
            DateFormat.Md(
              l.localeName,
            ).add_Hm().format(snapshot!.committedAt.toLocal()),
          ),
      detail: [?updated, ?warning?.$1].join('\n'),
      warning: warning != null,
    );
  }

  Widget _messages(
    CampusDataset data, {
    bool recent = false,
    String tab = 'messages',
  }) {
    final l = AppLocalizations.of(context);
    final query = _queryFor(tab);
    final sorted = [...data.messages]
      ..sort((a, b) => b.publishedAt.compareTo(a.publishedAt));
    final messages = recent
        ? sorted.take(5)
        : sorted.where(
            (m) => '${m.title} ${m.publisher}'.toLowerCase().contains(
              query.toLowerCase(),
            ),
          );
    if (messages.isEmpty) return GfText(l.campusNoNotices);
    return Column(
      children: [
        for (final m in messages)
          GfCard(
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) =>
                    ReadingPageWindow(child: CampusMessagePage(id: m.id)),
              ),
            ),
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                GfText(
                  m.title,
                  style: GfTheme.typographyOf(context).heading,
                  maxLines: recent ? 2 : null,
                  overflow: recent ? TextOverflow.ellipsis : null,
                ),
                const SizedBox(height: 8),
                GfText(
                  [
                    m.publisher,
                    m.publishedAt,
                  ].where((s) => s.isNotEmpty).join(' · '),
                  style: GfTheme.typographyOf(context).caption,
                ),
              ],
            ),
          ),
      ],
    );
  }

  Widget _todayHeader(CampusViewState state) {
    final l = AppLocalizations.of(context);
    final now = DateTime.now();
    final clock = campusNow(now);
    final name = campusMetric(state.data['profile'], '姓名');
    final week = int.tryParse(campusMetric(state.data['calendar'], '教学周'));
    final wishes = [l.campusWish1, l.campusWish2, l.campusWish3, l.campusWish4];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfText(
          '${week == null ? l.campusWeekUnknown : l.scheduleWeekN(week)} · ${DateFormat.MMMEd(l.localeName).format(clock)}',
          style: GfTheme.typographyOf(
            context,
          ).small.copyWith(color: GfTheme.colorsOf(context).iconMuted),
        ),
        const SizedBox(height: 12),
        GfText(
          '${campusGreeting(l, now)}${name.isEmpty ? '' : '，$name'}',
          style: GfTheme.typographyOf(context).title1,
        ),
        const SizedBox(height: 12),
        GfText(wishes[_wish], style: GfTheme.typographyOf(context).body),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: () => setState(
              () => _wish = (_wish + 1 + math.Random().nextInt(3)) % 4,
            ),
            icon: const GfSymbol('refresh-cw', size: 16),
            label: GfText(l.campusAnotherWish),
          ),
        ),
        const SizedBox(height: 20),
      ],
    );
  }

  Widget _today(CampusViewState state) {
    final l = AppLocalizations.of(context);
    final now = DateTime.now();
    final courses = KeyedSubtree(
      key: const Key('campus-today-courses'),
      child: _section(
        l.campusTodayCourses,
        _dataset(state, 'today', (data) {
          final day = data.teachingDay;
          if (day == null || day.date != campusDateKey(now)) {
            return GfText(l.campusDataNeedsRefresh);
          }
          final today = data.events;
          final notice = switch (day.kind) {
            'makeup' => l.campusTodayMakeup(day.label, day.sourceDate),
            'holiday' => l.campusTodayHoliday(day.label),
            'moved' => l.campusTodayMoved(day.label),
            _ => '',
          };
          final times = sectionTimesFor(
            day.sectionCount,
            _times.isEmpty ? null : _times,
          );
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (notice.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: GfText(notice),
                ),
              if (today.isEmpty) GfText(l.campusNoClasses),
              for (final e in today)
                GfCard(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      GfText(
                        '${times[e.start - 1].start}–${times[e.end - 1].end} · ${l.schedulePeriodRange('${e.start}–${e.end}')}',
                        style: GfTheme.typographyOf(context).caption,
                      ),
                      const SizedBox(height: 8),
                      GfText(
                        e.name,
                        style: GfTheme.typographyOf(context).heading,
                      ),
                      const SizedBox(height: 6),
                      GfText(
                        [
                          e.room,
                          e.campus,
                          e.teacher,
                        ].where((s) => s.isNotEmpty).join(' · '),
                      ),
                    ],
                  ),
                ),
            ],
          );
        }),
      ),
    );
    final notices = KeyedSubtree(
      key: const Key('campus-today-notices'),
      child: _section(
        l.campusMessages,
        _dataset(state, 'messages', (data) => _messages(data, recent: true)),
        action: TextButton(
          onPressed: () => _select('messages'),
          child: GfText(l.campusAllNotices),
        ),
      ),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _todayHeader(state),
        LayoutBuilder(
          builder: (context, constraints) {
            final scale = (MediaQuery.textScalerOf(context).scale(16) / 16)
                .clamp(1.0, double.infinity);
            if (constraints.maxWidth < 720 * scale) {
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [courses, notices],
              );
            }
            return Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(flex: 3, child: courses),
                const SizedBox(width: 32),
                Expanded(flex: 2, child: notices),
              ],
            );
          },
        ),
        const SizedBox(height: 24),
        const CampusCoursePicks(),
      ],
    );
  }

  Widget _academics(CampusViewState state, String tab) {
    final l = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _section(
          l.campusAcademics,
          _dataset(state, 'summary', (data) {
            final done = double.tryParse(campusMetric(data, '已修学分'));
            final required = double.tryParse(campusMetric(data, '要求学分'));
            return Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                CampusMetrics(data: data),
                if (done != null &&
                    done.isFinite &&
                    done >= 0 &&
                    required != null &&
                    required.isFinite &&
                    required > 0) ...[
                  const SizedBox(height: 20),
                  GfText(l.campusCreditProgress),
                  const SizedBox(height: 8),
                  LinearProgressIndicator(
                    value: (done / required).clamp(0, 1),
                    minHeight: 8,
                    borderRadius: BorderRadius.circular(GfRadius.xs),
                  ),
                ],
              ],
            );
          }),
        ),
        _section(
          l.campusGradeTrend,
          _dataset(
            state,
            'grades',
            (data) => data.series.isEmpty
                ? GfText(l.campusNoData)
                : CampusChart(points: data.series, maximum: 5),
          ),
        ),
        _section(
          l.campusCourses,
          _dataset(
            state,
            'grades',
            (data) => Column(
              children: [
                if (data.metrics.isNotEmpty) CampusMetrics(data: data),
                const SizedBox(height: 12),
                _searchField(tab),
                CampusRecords(
                  data: data,
                  query: _queryFor(tab),
                  titleColumn: 1,
                ),
              ],
            ),
          ),
        ),
        _section(
          l.campusCet,
          _dataset(
            state,
            'cet',
            (data) => Column(
              children: [
                CampusChart(points: data.series, maximum: 710),
                CampusRecords(data: data, titleColumn: 1),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _searchField(String tab) => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: GfSearchField(
      controller: _searchFor(tab),
      hintText: AppLocalizations.of(context).commonSearch,
      clearLabel: MaterialLocalizations.of(context).deleteButtonTooltip,
      onChanged: (value) => setState(() => _navigation.queries[tab] = value),
    ),
  );
  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    ref.listen(scheduleWidgetEntryProvider, (_, _) {
      if (_visible == true) _enter();
    });
    final privateVisible = _visible ?? false;
    final state = privateVisible
        ? ref.watch(campusControllerProvider)
        : const CampusViewState();
    final withholdData =
        privateVisible &&
        ref.read(campusControllerProvider.notifier).withholdDataUntilFresh;
    final binding = state.status?.binding;
    final identityRejected =
        state.needsAuthorization || isCampusIdentityError(state.error);
    if (!state.loading && (state.status != null || identityRejected)) {
      if ((_navigation.bindingObserved &&
              _navigation.binding != binding?.revision) ||
          (!_navigation.identityRejected && identityRejected)) {
        _navigation.clear();
        for (final search in _searches.values) {
          search.clear();
        }
        _restoreScroll = true;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) {
            unawaited(
              ref.read(campusControllerProvider.notifier).loadTab(_tab),
            );
          }
        });
      }
      _navigation.binding = binding?.revision;
      _navigation.bindingObserved = true;
      _navigation.identityRejected = identityRejected;
    }
    final labels = {
      'today': l.campusToday,
      'timetable': l.campusTimetable,
      'academics': l.campusAcademics,
      'messages': l.campusMessages,
      'calendars': l.campusCalendars,
    };
    final tabKeys = labels.keys.toList(growable: false);
    final layoutScale = (MediaQuery.textScalerOf(context).scale(16) / 16).clamp(
      1.0,
      double.infinity,
    );
    final hasDirectory = MediaQuery.sizeOf(context).width >= 900 * layoutScale;
    bool isPending(String tab) {
      if (!privateVisible ||
          (state.status != null && state.status?.binding == null)) {
        return false;
      }
      if (state.loading || withholdData) return true;
      return (campusTabKeys[tab] ?? const <String>[]).any((key) {
        return state.data[key] == null &&
            state.fetching.contains(key) &&
            !state.errors.containsKey(key);
      });
    }

    if (_swipeLoadingTab != null && !isPending(_swipeLoadingTab!)) {
      _swipeLoadingTab = null;
    }

    Widget contentFor(String tab, {double? fitHeight}) {
      if (isPending(tab) && tab == _swipeLoadingTab) {
        return const LogoMotionLoader(showMessage: true);
      }
      if (!privateVisible || state.loading || withholdData) {
        return _CampusRefreshSkeleton(
          tab: tab,
          week: _navigation.week ?? 1,
          timetableHeight: fitHeight,
          todayHeader: tab == 'today'
              ? _todayHeader(const CampusViewState())
              : null,
          onSelectMessages: () => _select('messages'),
          animate: privateVisible && _foreground,
        );
      }
      if (isPending(tab)) {
        return _CampusRefreshSkeleton(
          tab: tab,
          week: _navigation.week ?? 1,
          timetableHeight: fitHeight,
          todayHeader: tab == 'today'
              ? _todayHeader(const CampusViewState())
              : null,
          onSelectMessages: () => _select('messages'),
          animate: privateVisible && _foreground,
        );
      }
      if (state.status == null) {
        return GfErrorRetry(
          message: campusError(l, state.error),
          onRetry: () => ref.read(campusControllerProvider.notifier).refresh(),
        );
      }
      if (state.status?.binding == null) {
        return const CampusConnection(compact: true);
      }
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // The timetable folds freshness and progress into its header so
          // the week keeps the screen.
          if (tab != 'timetable') _snapshotNotice(state),
          if (state.refreshing && tab != 'timetable')
            Padding(
              padding: const EdgeInsets.only(bottom: 16),
              child: LinearProgressIndicator(semanticsLabel: l.commonLoading),
            ),
          if (state.status?.candidate != null || state.needsAuthorization)
            const Padding(
              padding: EdgeInsets.only(bottom: 24),
              child: CampusConnection(compact: true),
            ),
          if (!state.needsAuthorization)
            switch (tab) {
              'today' => _today(state),
              'timetable' => _dataset(
                state,
                'timetable',
                (data) => CampusWeekTimetable(
                  data: data,
                  selectedWeek: _navigation.week,
                  onWeekChanged: (week) => _navigation.week = week,
                  week: _snapshotWeek(state).clamp(1, 53),
                  maxWeek: math
                      .max(
                        int.tryParse(
                              campusMetric(state.data['calendar'], '学期周数'),
                            ) ??
                            20,
                        data.events
                            .expand((e) => e.weeks)
                            .fold<int>(1, math.max),
                      )
                      .clamp(1, 53),
                  times: _times,
                  height: fitHeight,
                  status: _timetableStatus(state),
                  refreshing: state.refreshing,
                  onRefresh: state.refreshing || state.busy
                      ? null
                      : () => ref
                            .read(campusControllerProvider.notifier)
                            .refresh(),
                ),
              ),
              'academics' => _academics(state, tab),
              'messages' => Column(
                children: [
                  _searchField(tab),
                  _dataset(
                    state,
                    'messages',
                    (data) => _messages(data, tab: tab),
                  ),
                ],
              ),
              'calendars' => Column(
                children: [
                  _section(
                    l.scheduleCurrentWeek,
                    _dataset(
                      state,
                      'calendar',
                      (data) => CampusMetrics(data: data),
                    ),
                  ),
                  _section(
                    l.campusCalendars,
                    _dataset(
                      state,
                      'terms',
                      (data) => CampusRecords(data: data),
                    ),
                  ),
                ],
              ),
              _ => const SizedBox.shrink(),
            },
        ],
      );
    }

    Widget scrollPage(
      String tab,
      double top,
      double bottom, {
      bool swipePreview = false,
    }) {
      final showSwipeLoader =
          isPending(tab) && (swipePreview || tab == _swipeLoadingTab);
      return GfScrollToTop(
        key: _scrollKeyFor(tab),
        controller: _scrollFor(tab),
        showButton: false,
        semanticLabel: l.commonBackToTop,
        builder: (_, controller) {
          if (showSwipeLoader) {
            return const LogoMotionLoader(showMessage: true);
          }
          final navigation = _navigation;
          final ready =
              (navigation.offsets[tab] ?? 0) <= 0 ||
              (!state.loading &&
                  !withholdData &&
                  (campusTabKeys[tab] ?? []).every(
                    (key) =>
                        !state.fetching.contains(key) &&
                        const {
                          'ready',
                          'empty',
                        }.contains(state.data[key]?.status),
                  ));
          if (_restoreScroll && ready) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (!mounted ||
                  !identical(navigation, _navigation) ||
                  tab != _tab ||
                  !controller.hasClients ||
                  !_restoreScroll) {
                return;
              }
              final position = controller.position;
              controller.jumpTo(
                (navigation.offsets[tab] ?? 0).clamp(
                  position.minScrollExtent,
                  position.maxScrollExtent,
                ),
              );
              _restoreScroll = false;
            });
          }
          return Listener(
            onPointerSignal: (signal) {
              if (signal is PointerScrollEvent &&
                  tab == _tab &&
                  controller.hasClients) {
                _restoreScroll = false;
                navigation.offsets[tab] = controller.position.pixels;
              }
            },
            child: NotificationListener<ScrollNotification>(
              onNotification: (notification) {
                if (notification.depth != 0 || tab != _tab) return false;
                if ((notification is ScrollStartNotification &&
                        notification.dragDetails != null) ||
                    (notification is UserScrollNotification &&
                        notification.direction != ScrollDirection.idle)) {
                  _restoreScroll = false;
                  navigation.offsets[tab] = notification.metrics.pixels;
                }
                if (notification is ScrollUpdateNotification &&
                    !_restoreScroll) {
                  navigation.offsets[tab] = notification.metrics.pixels;
                }
                return false;
              },
              child: AppRefreshIndicator(
                edgeOffset: top,
                onRefresh: () =>
                    ref.read(campusControllerProvider.notifier).refresh(),
                child: LayoutBuilder(
                  builder: (context, viewport) {
                    // The timetable fits one screen: tighter page margins and
                    // the remaining viewport height handed to the grid.
                    final timetable = tab == 'timetable';
                    final inline = timetable && viewport.maxWidth < 600
                        ? 12.0
                        : 20.0;
                    final padding = timetable
                        ? EdgeInsets.fromLTRB(
                            inline,
                            top + 12,
                            inline,
                            bottom + 12,
                          )
                        : EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16);
                    return ListView(
                      key: PageStorageKey<String>('campus-list-$tab'),
                      controller: controller,
                      physics: const AlwaysScrollableScrollPhysics(),
                      padding: padding,
                      children: [
                        if (tab == 'today') ...[
                          const CampusShortcuts(),
                          const SizedBox(height: 24),
                        ],
                        contentFor(
                          tab,
                          fitHeight: timetable && viewport.hasBoundedHeight
                              ? viewport.maxHeight - padding.vertical
                              : null,
                        ),
                        if (tab == 'today' &&
                            state.status?.binding == null) ...[
                          const SizedBox(height: 24),
                          const CampusCoursePicks(),
                        ],
                      ],
                    );
                  },
                ),
              ),
            ),
          );
        },
      );
    }

    return RootSurface(
      swipeTabIndex: hasDirectory ? null : tabKeys.indexOf(_tab),
      swipeTabCount: hasDirectory ? 0 : tabKeys.length,
      onSwipeTabChanged: (index) => _select(tabKeys[index], fromSwipe: true),
      swipePageKey: (index) => tabKeys[index],
      swipePageBuilder: (index, top, bottom) {
        final tab = tabKeys[index];
        return scrollPage(tab, top, bottom, swipePreview: true);
      },
      statusBarScrollToTopController: _scroll,
      title: l.campusTitle,
      showComposeAction: false,
      actions: [
        IconButton(
          tooltip: l.campusSettings,
          icon: const GfSymbol('settings'),
          onPressed: () => Navigator.of(context).push(
            MaterialPageRoute<void>(builder: (_) => const CampusSettingsPage()),
          ),
        ),
      ],
      toolbarHeight: hasDirectory ? 0 : GfTabBar.heightFor(context),
      toolbar: hasDirectory
          ? null
          : GfTabBar(
              tabs: [
                for (final e in labels.entries)
                  GfTab(label: e.value, value: e.key),
              ],
              selected: _tab,
              onSelected: (value) => _select(value as String),
            ),
      body: (top, bottom) => Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (hasDirectory) ...[
            SizedBox(
              key: const Key('campus-module-directory'),
              width: 200 * layoutScale,
              child: ListView(
                padding: EdgeInsets.fromLTRB(12, top + 24, 12, bottom + 16),
                children: [
                  for (final entry in labels.entries)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Semantics(
                        selected: _tab == entry.key,
                        child: Material(
                          color: _tab == entry.key
                              ? GfTheme.colorsOf(
                                  context,
                                ).primary.withValues(alpha: .08)
                              : Colors.transparent,
                          borderRadius: BorderRadius.circular(GfRadius.md),
                          clipBehavior: Clip.antiAlias,
                          child: GfSettingRow(
                            key: Key('campus-module-${entry.key}'),
                            symbol: switch (entry.key) {
                              'today' => 'sun',
                              'timetable' => 'calendar-days',
                              'academics' => 'book-open',
                              'messages' => 'bell',
                              'calendars' => 'calendar-days',
                              _ => 'link',
                            },
                            title: entry.value,
                            trailing: const SizedBox.shrink(),
                            onTap: () => _select(entry.key),
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
            VerticalDivider(
              width: 1,
              thickness: 1,
              color: GfTheme.colorsOf(context).line,
            ),
          ],
          Expanded(child: scrollPage(_tab, top, bottom)),
        ],
      ),
    );
  }
}

/// Clears private response widgets during refresh while preserving the
/// tab's real headings, controls and data-independent layout.
class _CampusRefreshSkeleton extends StatefulWidget {
  const _CampusRefreshSkeleton({
    required this.tab,
    required this.week,
    required this.animate,
    required this.onSelectMessages,
    this.todayHeader,
    this.timetableHeight,
  });

  final String tab;
  final int week;
  final double? timetableHeight;
  final bool animate;
  final Widget? todayHeader;
  final VoidCallback onSelectMessages;

  @override
  State<_CampusRefreshSkeleton> createState() => _CampusRefreshSkeletonState();
}

class _CampusRefreshSkeletonState extends State<_CampusRefreshSkeleton>
    with SingleTickerProviderStateMixin {
  late final AnimationController _shimmer = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1500),
  );
  bool _reducedMotion = false;

  bool get _shouldAnimate => widget.animate && !_reducedMotion;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncMotion();
  }

  @override
  void didUpdateWidget(_CampusRefreshSkeleton oldWidget) {
    super.didUpdateWidget(oldWidget);
    _syncMotion();
  }

  void _syncMotion() {
    _reducedMotion = GfMotion.reducedOf(context);
    if (_shouldAnimate) {
      if (!_shimmer.isAnimating) _shimmer.repeat();
    } else {
      _shimmer.stop();
      _shimmer.value = 0;
    }
  }

  @override
  void dispose() {
    _shimmer.dispose();
    super.dispose();
  }

  Widget _bone({double? width, double height = 16, double radius = 6}) =>
      GfSkeleton(width: width, height: height, radius: radius);

  Widget _shine(Widget child) {
    if (!_shouldAnimate) return child;
    final colors = GfTheme.colorsOf(context);
    return AnimatedBuilder(
      animation: _shimmer,
      child: child,
      builder: (context, child) => ShaderMask(
        blendMode: BlendMode.srcATop,
        shaderCallback: (bounds) {
          final start = -2.4 + _shimmer.value * 4.8;
          return LinearGradient(
            begin: Alignment(start, 0),
            end: Alignment(start + 1.4, 0),
            colors: [colors.base300, colors.base200, colors.base300],
          ).createShader(bounds);
        },
        child: child,
      ),
    );
  }

  Widget _section(String title, Widget child, {Widget? action}) => Padding(
    padding: const EdgeInsets.only(bottom: 28),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Wrap(
          alignment: WrapAlignment.spaceBetween,
          crossAxisAlignment: WrapCrossAlignment.center,
          spacing: 8,
          children: [
            GfText(title, style: GfTheme.typographyOf(context).title2),
            ?action,
          ],
        ),
        const SizedBox(height: 12),
        child,
      ],
    ),
  );

  Widget _courseCard() => GfCard(
    padding: const EdgeInsets.symmetric(vertical: 16),
    child: _shine(
      Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _bone(width: 176, height: 12, radius: 5),
          const SizedBox(height: 8),
          _bone(width: 232, height: 18, radius: 5),
          const SizedBox(height: 6),
          _bone(width: 196, height: 14, radius: 5),
        ],
      ),
    ),
  );

  Widget _messageCard({bool recent = false}) => GfCard(
    padding: const EdgeInsets.symmetric(vertical: 16),
    child: _shine(
      Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _bone(height: 18, radius: 5),
          if (recent) ...[
            const SizedBox(height: 6),
            Align(
              alignment: Alignment.centerLeft,
              child: _bone(width: 176, height: 18, radius: 5),
            ),
          ],
          const SizedBox(height: 8),
          _bone(width: 144, height: 12, radius: 5),
        ],
      ),
    ),
  );

  Widget _metrics() => LayoutBuilder(
    builder: (context, constraints) {
      final wide =
          constraints.maxWidth >= 360 &&
          MediaQuery.textScalerOf(context).scale(16) < 25;
      final width = wide
          ? (constraints.maxWidth - 12) / 2
          : constraints.maxWidth;
      return Wrap(
        spacing: 12,
        runSpacing: 12,
        children: [
          for (var i = 0; i < 4; i++)
            SizedBox(
              width: width,
              child: GfCard(
                emphasized: true,
                padding: const EdgeInsets.all(16),
                child: _shine(
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _bone(width: 84 + (i % 2) * 24, height: 12, radius: 5),
                      const SizedBox(height: 8),
                      _bone(width: 56, height: 22, radius: 5),
                    ],
                  ),
                ),
              ),
            ),
        ],
      );
    },
  );

  Widget _chart() => Column(
    children: [
      for (var i = 0; i < 3; i++)
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: _shine(
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _bone(width: 160 + i * 20, height: 14, radius: 5),
                const SizedBox(height: 8),
                _bone(height: 8, radius: 4),
              ],
            ),
          ),
        ),
    ],
  );

  Widget _records() => Column(
    children: [
      for (var i = 0; i < 3; i++)
        GfCard(
          padding: const EdgeInsets.symmetric(vertical: 16),
          child: _shine(
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _bone(width: 176 + (i % 2) * 36, height: 18, radius: 5),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 16,
                  runSpacing: 6,
                  children: [
                    _bone(width: 88, height: 12, radius: 5),
                    _bone(width: 104, height: 12, radius: 5),
                    _bone(width: 64, height: 12, radius: 5),
                  ],
                ),
              ],
            ),
          ),
        ),
    ],
  );

  Widget _search() => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: _shine(_bone(height: 48, radius: 28)),
  );

  Widget _snapshot() => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: Row(
      children: [
        Expanded(child: _shine(_bone(width: 168, height: 12, radius: 5))),
        const SizedBox(width: 8),
        _shine(_bone(width: 44, height: 44, radius: 22)),
      ],
    ),
  );

  /// Mirrors [CampusWeekTimetable]: the same header rhythm and the real grid
  /// geometry in placeholder mode, so loading never shifts the layout.
  Widget _timetable(AppLocalizations l) {
    final header = campusTimetableHeaderHeight(timetableTextScale(context));
    final height = widget.timetableHeight;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SizedBox(
          height: header - 8,
          child: Column(
            children: [
              Expanded(
                flex: 44,
                child: Row(
                  children: [
                    const SizedBox(width: 2),
                    _bone(width: 40, height: 40, radius: 20),
                    const SizedBox(width: 10),
                    Flexible(
                      child: FittedBox(
                        fit: BoxFit.scaleDown,
                        child: GfText(
                          l.scheduleWeekN(widget.week),
                          style: GfTheme.typographyOf(context).title3,
                        ),
                      ),
                    ),
                    const SizedBox(width: 10),
                    _bone(width: 40, height: 40, radius: 20),
                    const Spacer(),
                    _shine(_bone(width: 40, height: 40, radius: 20)),
                  ],
                ),
              ),
              Expanded(
                flex: 26,
                child: Align(
                  alignment: AlignmentDirectional.centerStart,
                  child: Padding(
                    padding: const EdgeInsetsDirectional.only(start: 12),
                    child: _shine(_bone(width: 168, height: 10, radius: 5)),
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        TimetableGrid(
          layout: buildTimetableWeek(
            events: const [],
            week: widget.week,
            blocks: const [],
            prefs: const TimetablePrefs(),
          ),
          times: sectionTimesFor(11, null),
          week: widget.week,
          height: height == null ? null : math.max(0.0, height - header),
          placeholder: _shine,
        ),
      ],
    );
  }

  Widget _body(AppLocalizations l) => switch (widget.tab) {
    'today' => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ?widget.todayHeader,
        _section(
          l.campusTodayCourses,
          Column(children: [_courseCard(), _courseCard()]),
        ),
        _section(
          l.campusMessages,
          Column(
            children: [_messageCard(recent: true), _messageCard(recent: true)],
          ),
          action: TextButton(
            onPressed: widget.onSelectMessages,
            child: GfText(l.campusAllNotices),
          ),
        ),
      ],
    ),
    'timetable' => _timetable(l),
    'academics' => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _section(l.campusAcademics, _metrics()),
        _section(l.campusGradeTrend, _chart()),
        _section(
          l.campusCourses,
          Column(children: [_metrics(), _search(), _records()]),
        ),
        _section(l.campusCet, Column(children: [_chart(), _records()])),
      ],
    ),
    'messages' => Column(
      children: [_search(), for (var i = 0; i < 4; i++) _messageCard()],
    ),
    'calendars' => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _section(l.scheduleCurrentWeek, _metrics()),
        _section(l.campusCalendars, _records()),
      ],
    ),
    _ => Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfText(l.campusConnection, style: GfTheme.typographyOf(context).title2),
        const SizedBox(height: 12),
        GfText(l.campusPrivacy),
        const SizedBox(height: 16),
        _shine(_bone(width: 176, height: 24, radius: 5)),
        const SizedBox(height: 20),
        _shine(_bone(height: 48, radius: 12)),
      ],
    ),
  };

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    return Semantics(
      container: true,
      liveRegion: true,
      label: l.commonLoading,
      child: ExcludeSemantics(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [if (widget.tab != 'connection') _snapshot(), _body(l)],
        ),
      ),
    );
  }
}
