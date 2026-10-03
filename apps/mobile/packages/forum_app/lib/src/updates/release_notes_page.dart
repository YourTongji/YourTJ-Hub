import 'dart:io';

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../l10n/app_localizations.dart';
import 'android_release.dart';
import 'ios_store_release.dart';
import 'release_notes.dart';
import 'release_notes_view.dart';

class ReleaseNotesPage extends StatefulWidget {
  const ReleaseNotesPage({super.key});

  @override
  State<ReleaseNotesPage> createState() => _ReleaseNotesPageState();
}

class _ReleaseNotesPageState extends State<ReleaseNotesPage> {
  late final AndroidReleaseClient _client = AndroidReleaseClient();
  ReleaseNoteCatalog? _catalog;
  IosStoreListing? _appStore;
  bool _loading = true;
  // Public history follows the user's platform; beta instructions belong in TestFlight.
  String get _platform => Platform.isAndroid ? 'android' : 'ios-app-store';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool refresh = true}) async {
    if (!mounted) return;
    setState(() => _loading = true);
    try {
      final preferences = await SharedPreferences.getInstance();
      if (!mounted) return;
      _client.notesPreferences = preferences;
      if (refresh) {
        final cached = await _client.loadHistory(refresh: false);
        if (mounted && cached != null) setState(() => _catalog = cached);
      }
      final catalog = await _client.loadHistory(refresh: refresh);
      final store = Platform.isIOS ? await IosStoreListing.lookup() : null;
      if (mounted) {
        setState(() {
          _catalog = catalog;
          _appStore = store;
          _loading = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _openAppStore() async {
    final url = _appStore?.url;
    if (url != null) await launchUrl(url, mode: LaunchMode.externalApplication);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final entries = [
      for (final release
          in _catalog?.releases.reversed ?? const <ReleaseNoteVersion>[])
        if (release.channels.contains(_platform))
          (
            release,
            [
              ...release.highlights,
              ...release.breaking,
              ...release.requiredActions,
            ].where((note) => note.includesChannel(_platform)).toList(),
          ),
    ].where((entry) => entry.$2.isNotEmpty).toList();
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.releaseNotesHistory)),
      body: RefreshIndicator(
        onRefresh: () => _load(),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
          children: [
            if (Platform.isIOS && _appStore != null)
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton.icon(
                  onPressed: _openAppStore,
                  icon: const Icon(Icons.open_in_new),
                  label: Text(l10n.updateOpenAppStore),
                ),
              ),
            if (_loading && _catalog == null)
              const Center(
                child: Padding(
                  padding: EdgeInsets.all(32),
                  child: CircularProgressIndicator(),
                ),
              )
            else if (entries.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 32),
                child: Text(l10n.releaseNotesEmpty),
              )
            else
              for (final (index, entry) in entries.indexed)
                _ReleaseTimelineEntry(
                  release: entry.$1,
                  notes: entry.$2,
                  latest: index == 0,
                  last: index == entries.length - 1,
                ),
            if (_catalog != null && _hasIncompleteHistory)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: Text(l10n.releaseHistoryIncomplete),
              ),
            if (_loading && _catalog != null)
              const Padding(
                padding: EdgeInsets.all(12),
                child: LinearProgressIndicator(),
              ),
            if (!_loading && _catalog == null)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(l10n.releaseNotesHistoryError),
                    TextButton.icon(
                      onPressed: _load,
                      icon: const Icon(Icons.refresh),
                      label: Text(l10n.commonRetry),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }

  bool get _hasIncompleteHistory {
    final catalog = _catalog;
    if (catalog == null) return false;
    final coverage = catalog.channelCoverage[_platform];
    if (coverage == null) return true;
    return catalog.releases
        .where((release) => release.channels.contains(_platform))
        .any(
          (release) => !coverage.coveredBuilds.contains(release.buildNumber),
        );
  }
}

/// One version on the history timeline: a dot on a continuous hairline rail,
/// with the newest version marked in the primary color.
class _ReleaseTimelineEntry extends StatelessWidget {
  const _ReleaseTimelineEntry({
    required this.release,
    required this.notes,
    required this.latest,
    required this.last,
  });
  final ReleaseNoteVersion release;
  final List<ReleaseNote> notes;
  final bool latest;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            width: 12,
            child: Column(
              children: [
                // Centers the dot on the version line (16px × 1.5 leading).
                const SizedBox(height: 7),
                Container(
                  width: 10,
                  height: 10,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: latest ? colors.primary : colors.base100,
                    border: latest
                        ? null
                        : Border.all(color: colors.line, width: 2),
                  ),
                ),
                if (!last)
                  Expanded(
                    child: Padding(
                      padding: const EdgeInsets.only(top: 6),
                      child: Container(width: 1, color: colors.line),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Padding(
              padding: EdgeInsets.only(bottom: last ? 0 : 32),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text.rich(
                    TextSpan(
                      text: release.version,
                      style: type.heading,
                      children: [
                        TextSpan(
                          text: ' · ${release.buildNumber}',
                          style: type.caption.copyWith(
                            color: colors.baseContent.withValues(alpha: 0.6),
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  ReleaseNotesView(notes: notes),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
