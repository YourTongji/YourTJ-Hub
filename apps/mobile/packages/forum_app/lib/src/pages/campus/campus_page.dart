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
import '../../widgets/app_refresh_indicator.dart';
import '../../widgets/root_surface.dart';
import '../../widgets/logo_motion_loader.dart';
import '../../widgets/status_views.dart';
import '../../widgets/campus_shortcuts.dart';
import 'campus_connection.dart';
import 'campus_data_views.dart';
import 'campus_helpers.dart';
import 'campus_message_page.dart';
import 'campus_state.dart';

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
    ref.listen(offlineCacheEpochProvider, (_, _) {
      setState(() => _navigation = _CampusNavigation());
    });
    final repository = ref.watch(campusRepositoryProvider);
    ref.listen(campusRepositoryProvider, (_, _) {
      _navigation = _CampusNavigation();
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
      actions: [
        IconButton(
          tooltip: l.campusExplore,
          icon: const GfSymbol('graduation-cap'),
          onPressed: () => context.push('/campus/explore'),
        ),
      ],
      body: (top, bottom) => ListView(
        padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16),
        children: [
          const CampusShortcuts(),
          const SizedBox(height: 24),
          content,
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
                        Text(
                          l.campusOfficialSubtitle,
                          style: GfTheme.typographyOf(context).title2,
                        ),
                        const SizedBox(height: 16),
                        Text(l.campusPrivacy),
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
    _foreground = lifecycle == null || lifecycle == AppLifecycleState.resumed;
    WidgetsBinding.instance.addObserver(this);
    _searchFor(_tab);
    // A notice may have kept the controller alive while this page was hidden.
    // Verify its binding and refresh private data when the page returns.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _visible == true) {
        unawaited(ref.read(campusControllerProvider.notifier).enterTab(_tab));
      }
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
    if (state != AppLifecycleState.resumed && _visible == true) {
      _restoreScroll = true;
      ref.read(campusControllerProvider.notifier).suspend();
    }
    _foreground = state == AppLifecycleState.resumed;
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
        if (mounted && _visible == true) {
          unawaited(ref.read(campusControllerProvider.notifier).enterTab(_tab));
        }
      });
    }
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

  void _select(String tab, {bool fromSwipe = false}) {
    if (tab == _tab) return;
    FocusManager.instance.primaryFocus?.unfocus();
    _swipeLoadingTab = fromSwipe ? tab : null;
    setState(() {
      _navigation.tab = tab;
      _searchFor(tab).text = _queryFor(tab);
      _restoreScroll = true;
    });
    _registry.register(GfShellDestination.campus, _scrollFor(tab));
    unawaited(ref.read(campusControllerProvider.notifier).loadTab(tab));
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
            Text(title, style: GfTheme.typographyOf(context).title2),
            ?action,
          ],
        ),
        const SizedBox(height: 12),
        child,
      ],
    ),
  );
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
        Text(l.campusDataNeedsRefresh),
        TextButton.icon(
          onPressed: loading
              ? null
              : () => ref.read(campusControllerProvider.notifier).refresh(),
          icon: const GfSymbol('refresh-cw', size: 18),
          label: Text(l.commonRefresh),
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
        ? Text(l.campusUnavailable)
        : data.status == 'empty' && key != 'today'
        ? Text(l.campusNoData)
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
                    : Text(
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
            Text(l.campusSnapshotOffline)
          else if (state.error != null || state.errors.isNotEmpty)
            Text(l.campusSnapshotRefreshFailed)
          else if (stale)
            Text(l.campusSnapshotStale),
        ],
      ),
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
    if (messages.isEmpty) return Text(l.campusNoNotices);
    return Column(
      children: [
        for (final m in messages)
          GfCard(
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => CampusMessagePage(id: m.id)),
            ),
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  m.title,
                  style: GfTheme.typographyOf(context).heading,
                  maxLines: recent ? 2 : null,
                  overflow: recent ? TextOverflow.ellipsis : null,
                ),
                const SizedBox(height: 8),
                Text(
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
        Text(
          '${week == null ? l.campusWeekUnknown : l.scheduleWeekN(week)} · ${DateFormat.MMMEd(l.localeName).format(clock)}',
          style: GfTheme.typographyOf(
            context,
          ).small.copyWith(color: GfTheme.colorsOf(context).iconMuted),
        ),
        const SizedBox(height: 12),
        Text(
          '${campusGreeting(l, now)}${name.isEmpty ? '' : '，$name'}',
          style: GfTheme.typographyOf(context).title1,
        ),
        const SizedBox(height: 12),
        Text(wishes[_wish], style: GfTheme.typographyOf(context).body),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: () => setState(
              () => _wish = (_wish + 1 + math.Random().nextInt(3)) % 4,
            ),
            icon: const GfSymbol('refresh-cw', size: 16),
            label: Text(l.campusAnotherWish),
          ),
        ),
        const SizedBox(height: 20),
      ],
    );
  }

  Widget _today(CampusViewState state) {
    final l = AppLocalizations.of(context);
    final now = DateTime.now();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _todayHeader(state),
        _section(
          l.campusTodayCourses,
          _dataset(state, 'today', (data) {
            final day = data.teachingDay;
            if (day == null || day.date != campusDateKey(now)) {
              return Text(l.campusDataNeedsRefresh);
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
                    child: Text(notice),
                  ),
                if (today.isEmpty) Text(l.campusNoClasses),
                for (final e in today)
                  GfCard(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(
                          '${times[e.start - 1].start}–${times[e.end - 1].end} · ${l.schedulePeriodRange('${e.start}–${e.end}')}',
                          style: GfTheme.typographyOf(context).caption,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          e.name,
                          style: GfTheme.typographyOf(context).heading,
                        ),
                        const SizedBox(height: 6),
                        Text(
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
        _section(
          l.campusMessages,
          _dataset(state, 'messages', (data) => _messages(data, recent: true)),
          action: TextButton(
            onPressed: () => _select('messages'),
            child: Text(l.campusAllNotices),
          ),
        ),
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
                  Text(l.campusCreditProgress),
                  const SizedBox(height: 8),
                  LinearProgressIndicator(
                    value: (done / required).clamp(0, 1),
                    minHeight: 8,
                    borderRadius: BorderRadius.circular(4),
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
                ? Text(l.campusNoData)
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
      'connection': l.campusConnection,
    };
    final tabKeys = labels.keys.toList(growable: false);
    bool isPending(String tab) {
      if (!privateVisible ||
          tab == 'connection' ||
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

    Widget contentFor(String tab) {
      if (isPending(tab) && tab == _swipeLoadingTab) {
        return const LogoMotionLoader(showMessage: true);
      }
      if (!privateVisible || state.loading || withholdData) {
        return _CampusRefreshSkeleton(
          tab: tab,
          week: _navigation.week ?? 1,
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
      if (state.status?.binding == null || tab == 'connection') {
        return const CampusConnection();
      }
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _snapshotNotice(state),
          if (state.refreshing)
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
                  week:
                      (int.tryParse(
                                campusMetric(state.data['calendar'], '教学周'),
                              ) ??
                              1)
                          .clamp(1, 53),
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
                child: ListView(
                  key: PageStorageKey<String>('campus-list-$tab'),
                  controller: controller,
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16),
                  children: [
                    if (tab == 'today') ...[
                      const CampusShortcuts(),
                      const SizedBox(height: 24),
                    ],
                    contentFor(tab),
                  ],
                ),
              ),
            ),
          );
        },
      );
    }

    return RootSurface(
      swipeTabIndex: tabKeys.indexOf(_tab),
      swipeTabCount: tabKeys.length,
      onSwipeTabChanged: (index) => _select(tabKeys[index], fromSwipe: true),
      swipePageKey: (index) => tabKeys[index],
      swipePageBuilder: (index, top, bottom) {
        final tab = tabKeys[index];
        return scrollPage(tab, top, bottom, swipePreview: true);
      },
      title: l.campusTitle,
      showComposeAction: false,
      actions: [
        IconButton(
          tooltip: l.campusExplore,
          icon: const GfSymbol('graduation-cap'),
          onPressed: () => context.push('/campus/explore'),
        ),
      ],
      toolbarHeight: GfTabBar.heightFor(context),
      toolbar: GfTabBar(
        tabs: [
          for (final e in labels.entries) GfTab(label: e.value, value: e.key),
        ],
        selected: _tab,
        onSelected: (value) => _select(value as String),
      ),
      body: (top, bottom) => scrollPage(_tab, top, bottom),
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
  });

  final String tab;
  final int week;
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
            Text(title, style: GfTheme.typographyOf(context).title2),
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

  Widget _timetable(AppLocalizations l) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Align(
        alignment: Alignment.centerLeft,
        child: _shine(_bone(width: 144, height: 44, radius: 22)),
      ),
      const SizedBox(height: 12),
      Wrap(
        alignment: WrapAlignment.center,
        crossAxisAlignment: WrapCrossAlignment.center,
        spacing: 8,
        children: [
          _bone(width: 40, height: 40, radius: 20),
          Text(
            l.scheduleWeekN(widget.week),
            style: GfTheme.typographyOf(context).heading,
          ),
          _bone(width: 40, height: 40, radius: 20),
          TextButton(onPressed: null, child: Text(l.scheduleCurrentWeek)),
        ],
      ),
      const SizedBox(height: 12),
      _timetableGrid(l),
    ],
  );

  Widget _timetableGrid(AppLocalizations l) => LayoutBuilder(
    builder: (context, constraints) {
      final scaler = MediaQuery.textScalerOf(context);
      final scale = [
        9.5,
        10.0,
        11.0,
        12.0,
      ].map((size) => scaler.scale(size) / size).fold(1.0, math.max);
      final width = constraints.maxWidth;
      final timeWidth = math.min(56 * scale, width * .4);
      final available = width - timeWidth - 2;
      final dayWidth = math.max(76 * scale, available / 7);
      final rowHeight = 64 * scale;
      final colors = GfTheme.colorsOf(context);
      final days = [
        l.scheduleDayMon,
        l.scheduleDayTue,
        l.scheduleDayWed,
        l.scheduleDayThu,
        l.scheduleDayFri,
        l.scheduleDaySat,
        l.scheduleDaySun,
      ];
      Widget header(String label, double width) => Container(
        width: width,
        height: 36 * scale,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: colors.base200.withValues(alpha: .7),
          border: Border(
            right: BorderSide(color: colors.line),
            bottom: BorderSide(color: colors.line),
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            // Text applies the ambient scaler; only grid geometry scales here.
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: colors.baseContent,
          ),
        ),
      );
      Widget timeCell(int row) => Container(
        width: timeWidth,
        height: rowHeight,
        padding: const EdgeInsets.symmetric(horizontal: 1, vertical: 3),
        decoration: BoxDecoration(
          border: Border(
            right: BorderSide(color: colors.line),
            bottom: BorderSide(color: colors.line),
          ),
        ),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              '$row',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w700,
                color: colors.baseContent.withValues(alpha: .7),
              ),
            ),
            SizedBox(height: 2 * scale),
            _shine(
              Column(
                children: [
                  _bone(width: 28 * scale, height: 8 * scale, radius: 3),
                  SizedBox(height: 2 * scale),
                  _bone(width: 28 * scale, height: 8 * scale, radius: 3),
                ],
              ),
            ),
          ],
        ),
      );
      Widget dayColumn(int day) => SizedBox(
        width: dayWidth,
        height: rowHeight * 12,
        child: Stack(
          children: [
            Column(
              children: [
                for (var row = 0; row < 12; row++)
                  Container(
                    width: dayWidth,
                    height: rowHeight,
                    decoration: BoxDecoration(
                      border: Border(
                        right: BorderSide(color: colors.line),
                        bottom: BorderSide(color: colors.line),
                      ),
                    ),
                  ),
              ],
            ),
            for (final (courseDay, row, span) in const [
              (1, 1, 2),
              (3, 4, 2),
              (5, 7, 2),
            ])
              if (day == courseDay)
                Positioned(
                  top: row * rowHeight + 4,
                  left: 4,
                  right: 4,
                  height: span * rowHeight - 8,
                  child: _shine(_bone(radius: 8)),
                ),
          ],
        ),
      );
      final scrolls = dayWidth * 7 > available + 1;
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            decoration: BoxDecoration(
              color: colors.base100,
              borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
              border: Border.all(color: colors.line),
            ),
            clipBehavior: Clip.antiAlias,
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(
                  width: timeWidth,
                  child: Column(
                    children: [
                      header(l.scheduleTimeAxis, timeWidth),
                      for (var row = 1; row <= 12; row++) timeCell(row),
                    ],
                  ),
                ),
                Expanded(
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: SizedBox(
                      width: dayWidth * 7,
                      child: Column(
                        children: [
                          Row(
                            children: [
                              for (final day in days) header(day, dayWidth),
                            ],
                          ),
                          Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              for (var day = 0; day < 7; day++) dayColumn(day),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
          if (scrolls)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                l.scheduleGridScrollHint,
                style: GfTheme.typographyOf(context).caption,
              ),
            ),
        ],
      );
    },
  );

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
            child: Text(l.campusAllNotices),
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
        Text(l.campusConnection, style: GfTheme.typographyOf(context).title2),
        const SizedBox(height: 12),
        Text(l.campusPrivacy),
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
