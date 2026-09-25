import 'dart:async';
import 'dart:math' as math;
import 'package:core/core.dart';
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
import '../../widgets/status_views.dart';
import '../../widgets/campus_shortcuts.dart';
import 'campus_connection.dart';
import 'campus_data_views.dart';
import 'campus_helpers.dart';
import 'campus_message_page.dart';
import 'campus_private_surface.dart';
import 'campus_state.dart';

/// Only navigation choices survive the disposable private view. This object is
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

class _CampusPageState extends ConsumerState<CampusPage>
    with WidgetsBindingObserver {
  _CampusNavigation _navigation = _CampusNavigation();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) {
      // Replace the object so a disposing child cannot refill the next session.
      setState(() => _navigation = _CampusNavigation());
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, _) {
      setState(() => _navigation = _CampusNavigation());
    });
    final repository = ref.watch(campusRepositoryProvider);
    ref.listen(campusRepositoryProvider, (_, _) {
      _navigation = _CampusNavigation();
    });
    return CampusPrivateSurface(
      key: ObjectKey(repository),
      builder: (_) => _CampusAccount(navigation: _navigation),
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

class _CampusWorkspaceState extends ConsumerState<_CampusWorkspace> {
  _CampusNavigation get _navigation => widget.navigation;
  String get _tab => _navigation.tab;
  String get _query => _navigation.queries[_tab] ?? '';
  bool _restoreScroll = true;
  int _wish = math.Random().nextInt(4);
  final _search = TextEditingController();
  final _scroll = GfScrollToTopController();
  late final GfTabScrollRegistry _registry;
  Timer? _clock;
  List<SectionTime> _times = [];
  @override
  void initState() {
    super.initState();
    _search.text = _query;
    // A notice may have kept the shared connection alive while this tab was
    // hidden. Verify the binding before reusing any foreground cache.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        unawaited(ref.read(campusControllerProvider.notifier).enterTab(_tab));
      }
    });
    _registry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.campus, _scroll);
    _clock = Timer.periodic(const Duration(minutes: 1), (_) {
      if (mounted) {
        setState(() {});
        unawaited(ref.read(campusControllerProvider.notifier).refreshVisible());
      }
    });
    unawaited(_loadTimes());
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
    _search.dispose();
    _registry.unregister(GfShellDestination.campus, _scroll);
    super.dispose();
  }

  void _select(String tab) {
    if (tab == _tab) return;
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _navigation.tab = tab;
      _search.text = _query;
      _restoreScroll = true;
    });
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
          icon: const Icon(Icons.refresh, size: 18),
          label: Text(l.commonRefresh),
        ),
      ],
    );
    final content = data == null
        ? (loading
              ? const Padding(padding: EdgeInsets.all(20), child: GfLoading())
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
                icon: const Icon(Icons.refresh, size: 18),
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

  Widget _messages(CampusDataset data, {bool recent = false}) {
    final l = AppLocalizations.of(context);
    final sorted = [...data.messages]
      ..sort((a, b) => b.publishedAt.compareTo(a.publishedAt));
    final messages = recent
        ? sorted.take(5)
        : sorted.where(
            (m) => '${m.title} ${m.publisher}'.toLowerCase().contains(
              _query.toLowerCase(),
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

  Widget _today(CampusViewState state) {
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
            icon: const Icon(Icons.refresh, size: 16),
            label: Text(l.campusAnotherWish),
          ),
        ),
        const SizedBox(height: 20),
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

  Widget _academics(CampusViewState state) {
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
                _searchField(),
                CampusRecords(data: data, query: _query, titleColumn: 1),
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

  Widget _searchField() => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: GfSearchField(
      controller: _search,
      hintText: AppLocalizations.of(context).commonSearch,
      clearLabel: MaterialLocalizations.of(context).deleteButtonTooltip,
      onChanged: (value) => setState(() => _navigation.queries[_tab] = value),
    ),
  );
  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    final binding = state.status?.binding;
    final identityRejected =
        state.needsAuthorization || isCampusIdentityError(state.error);
    if (!state.loading && (state.status != null || identityRejected)) {
      if ((_navigation.bindingObserved &&
              _navigation.binding != binding?.revision) ||
          (!_navigation.identityRejected && identityRejected)) {
        _navigation.clear();
        _search.clear();
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
    Widget content;
    if (state.loading) {
      content = const GfLoading();
    } else if (state.status == null) {
      content = GfErrorRetry(
        message: campusError(l, state.error),
        onRetry: () => ref.read(campusControllerProvider.notifier).refresh(),
      );
    } else if (state.status?.binding == null || _tab == 'connection') {
      content = const CampusConnection();
    } else {
      content = Column(
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
            switch (_tab) {
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
              'academics' => _academics(state),
              'messages' => Column(
                children: [
                  _searchField(),
                  _dataset(state, 'messages', _messages),
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
    return RootSurface(
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
      body: (top, bottom) => GfScrollToTop(
        controller: _scroll,
        showButton: false,
        semanticLabel: l.commonBackToTop,
        builder: (_, controller) {
          final tab = _tab;
          final navigation = _navigation;
          // A first visit already starts at zero. Settle that immediately:
          // waiting for missing datasets could later undo a manual refresh/scroll.
          final ready =
              (navigation.offsets[tab] ?? 0) <= 0 ||
              (!state.loading &&
                  !state.refreshing &&
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
          return NotificationListener<ScrollNotification>(
            onNotification: (notification) {
              if (notification.depth != 0 || tab != _tab) return false;
              if ((notification is ScrollStartNotification &&
                      notification.dragDetails != null) ||
                  (notification is UserScrollNotification &&
                      notification.direction != ScrollDirection.idle)) {
                // Explicit reading intent supersedes a saved position, including
                // while a section is still waiting for its private data.
                _restoreScroll = false;
              }
              if (notification is ScrollUpdateNotification && !_restoreScroll) {
                navigation.offsets[tab] = notification.metrics.pixels;
              }
              return false;
            },
            child: AppRefreshIndicator(
              edgeOffset: top,
              onRefresh: () =>
                  ref.read(campusControllerProvider.notifier).refresh(),
              child: ListView(
                controller: controller,
                physics: const AlwaysScrollableScrollPhysics(),
                padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16),
                children: [
                  if (_tab == 'today') ...[
                    const CampusShortcuts(),
                    const SizedBox(height: 24),
                  ],
                  content,
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
