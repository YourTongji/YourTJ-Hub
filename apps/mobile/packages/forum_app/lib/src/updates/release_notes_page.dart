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
  const ReleaseNotesPage({super.key, this.initialIosChannel = 'ios-app-store'});

  final String initialIosChannel;

  @override
  State<ReleaseNotesPage> createState() => _ReleaseNotesPageState();
}

class _ReleaseNotesPageState extends State<ReleaseNotesPage> {
  late final AndroidReleaseClient _client = AndroidReleaseClient();
  ReleaseNoteCatalog? _catalog;
  IosStoreListing? _appStore;
  bool _loading = true;
  late String _iosChannel = widget.initialIosChannel;
  String get _platform => Platform.isAndroid ? 'android' : _iosChannel;

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
    final releases =
        _catalog?.releases.reversed.toList() ?? const <ReleaseNoteVersion>[];
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.releaseNotesHistory)),
      body: RefreshIndicator(
        onRefresh: () => _load(),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
          children: [
            if (Platform.isIOS)
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(
                    value: 'ios-app-store',
                    label: Text('App Store'),
                  ),
                  ButtonSegment(
                    value: 'ios-testflight',
                    label: Text('TestFlight'),
                  ),
                ],
                selected: {_iosChannel},
                onSelectionChanged: (value) =>
                    setState(() => _iosChannel = value.first),
              ),
            if (Platform.isIOS &&
                _iosChannel == 'ios-app-store' &&
                _appStore != null)
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
            else if (releases.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 32),
                child: Text(l10n.releaseNotesEmpty),
              )
            else
              for (final release in releases)
                _ReleaseHistoryCard(release: release, platform: _platform),
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

class _ReleaseHistoryCard extends StatelessWidget {
  const _ReleaseHistoryCard({required this.release, required this.platform});
  final ReleaseNoteVersion release;
  final String platform;

  @override
  Widget build(BuildContext context) {
    if (!release.channels.contains(platform)) return const SizedBox.shrink();
    final notes = [
      ...release.highlights,
      ...release.breaking,
      ...release.requiredActions,
      ...release.testflightNotes,
    ].where((note) => note.includesChannel(platform)).toList();
    if (notes.isEmpty) return const SizedBox.shrink();
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '${release.version} · ${release.buildNumber}',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            ReleaseNotesView(notes: notes),
          ],
        ),
      ),
    );
  }
}
