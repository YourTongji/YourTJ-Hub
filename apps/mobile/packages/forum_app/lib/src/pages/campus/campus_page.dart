import 'dart:async';
import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/material.dart';
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
import 'campus_connection.dart';
import 'campus_data_views.dart';
import 'campus_helpers.dart';
import 'campus_message_page.dart';
import 'campus_private_surface.dart';
import 'campus_state.dart';

class CampusPage extends StatelessWidget {
  const CampusPage({super.key});
  @override
  Widget build(BuildContext context) =>
      CampusPrivateSurface(builder: (_) => const _CampusAccount());
}

class _CampusAccount extends ConsumerWidget {
  const _CampusAccount();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    return ref
        .watch(currentUserProvider)
        .when(
          loading: () => const Center(child: GfLoading()),
          error: (_, _) => GfErrorRetry(
            message: l.campusUnavailable,
            onRetry: () => ref.invalidate(currentUserProvider),
          ),
          data: (user) => user == null
              ? RootSurface(
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
                    padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom),
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
              : _CampusWorkspace(key: ValueKey(user.id)),
        );
  }
}

class _CampusWorkspace extends ConsumerStatefulWidget {
  const _CampusWorkspace({super.key});
  @override
  ConsumerState<_CampusWorkspace> createState() => _CampusWorkspaceState();
}

class _CampusWorkspaceState extends ConsumerState<_CampusWorkspace> {
  String _tab = 'today';
  String _query = '';
  int _wish = math.Random().nextInt(4);
  final _search = TextEditingController();
  final _scroll = GfScrollToTopController();
  late final GfTabScrollRegistry _registry;
  Timer? _clock;
  List<SectionTime> _times = [];
  @override
  void initState() {
    super.initState();
    // A notice may have kept the shared connection alive while this tab was
    // hidden. Re-entering the overview must request its own datasets again.
    unawaited(ref.read(campusControllerProvider.notifier).loadTab('today'));
    _registry = ref.read(tabScrollRegistryProvider)
      ..register(GfShellDestination.campus, _scroll);
    _clock = Timer.periodic(const Duration(minutes: 1), (_) {
      if (mounted) setState(() {});
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
    setState(() {
      _tab = tab;
      _query = '';
      _search.clear();
    });
    unawaited(ref.read(campusControllerProvider.notifier).loadTab(tab));
    _scroll.scrollToTop();
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
    if (error != null) {
      return GfErrorRetry(
        message: campusError(l, error),
        onRetry: () => ref.read(campusControllerProvider.notifier).load(key),
      );
    }
    if (data == null) {
      return const Padding(padding: EdgeInsets.all(20), child: GfLoading());
    }
    if (data.status != 'ready' && data.status != 'empty') {
      return Text(l.campusUnavailable);
    }
    if (data.status == 'empty') return Text(l.campusNoData);
    return ready(data);
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
          l.campusMessages,
          _dataset(state, 'messages', (data) => _messages(data, recent: true)),
          action: TextButton(
            onPressed: () => _select('messages'),
            child: Text(l.campusAllNotices),
          ),
        ),
        _section(
          l.campusTodayCourses,
          _dataset(state, 'timetable', (data) {
            if (week == null) return Text(l.campusWeekUnknown);
            final today =
                campusCoursesForWeek(
                    data.events,
                    week,
                  ).where((e) => e.day == clock.weekday).toList()
                  ..sort((a, b) => a.start.compareTo(b.start));
            if (today.isEmpty) return Text(l.campusNoClasses);
            final times = sectionTimesFor(
              data.events.any((e) => e.end == 12) ? 12 : 11,
              _times.isEmpty ? null : _times,
            );
            return Column(
              children: [
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
      onChanged: (value) => setState(() => _query = value),
    ),
  );
  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
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
        builder: (_, controller) => AppRefreshIndicator(
          edgeOffset: top,
          onRefresh: () =>
              ref.read(campusControllerProvider.notifier).refresh(),
          child: ListView(
            controller: controller,
            physics: const AlwaysScrollableScrollPhysics(),
            padding: EdgeInsets.fromLTRB(20, top + 24, 20, bottom + 16),
            children: [content],
          ),
        ),
      ),
    );
  }
}
