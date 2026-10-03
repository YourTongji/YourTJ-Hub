import 'package:ui_kit/ui_kit.dart';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../l10n/app_localizations.dart';
import 'android_release.dart';
import 'release_notes.dart';
import 'ios_store_release.dart';
import 'update_prompt_sheet.dart';

const updateChannel = MethodChannel('yourtj/app_updates');
const testFlightAppUrl = 'https://apps.apple.com/app/testflight/id899247664';
final appUpdateHostKey = GlobalKey<MobileUpdateHostState>();
bool get supportsApkUpdates => !kIsWeb && Platform.isAndroid;
bool get supportsMobileReleaseNotes =>
    !kIsWeb && (Platform.isAndroid || Platform.isIOS);

class MobileUpdateHost extends StatefulWidget {
  const MobileUpdateHost({
    super.key,
    required this.navigatorKey,
    required this.child,
  });
  final GlobalKey<NavigatorState> navigatorKey;
  final Widget child;

  @override
  State<MobileUpdateHost> createState() => MobileUpdateHostState();
}

class MobileUpdateHostState extends State<MobileUpdateHost>
    with WidgetsBindingObserver {
  bool _busy = false;
  final _client = AndroidReleaseClient();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    WidgetsBinding.instance.addPostFrameCallback((_) => check());
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) check();
  }

  Future<void> check({bool force = false}) async {
    if (!supportsMobileReleaseNotes || _busy || !mounted) return;
    _busy = true;
    try {
      final preferences = await SharedPreferences.getInstance();
      _client.notesPreferences = preferences;
      final now = DateTime.now().millisecondsSinceEpoch;
      final last = preferences.getInt('update.lastCheck') ?? 0;
      if (!force &&
          now >= last &&
          now - last < const Duration(hours: 6).inMilliseconds) {
        return;
      }
      await preferences.setInt('update.lastCheck', now);
      final info = await updateChannel.invokeMapMethod<String, dynamic>(
        'getInfo',
      );
      if (info == null) return;
      if (Platform.isIOS) {
        await _checkIos(info, preferences, force);
        return;
      }
      final release = await _client.check(
        (info['abis'] as List).cast<String>(),
        (info['buildNumber'] as num).toInt(),
      );
      if (!mounted) return;
      final context = widget.navigatorKey.currentContext;
      if (context == null || !context.mounted) return;
      if (release == null) {
        if (force) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(AppLocalizations.of(context).updateLatest)),
          );
        }
        return;
      }
      if (!force &&
          preferences.getInt('update.skippedBuild') == release.buildNumber) {
        return;
      }
      await showGfBottomSheet<void>(
        context,
        barrierDismissible: false,
        enableDrag: false,
        builder: (_) => _UpdateDialog(
          release: release,
          installedVersion: info['version'] as String?,
          installedBuild: (info['buildNumber'] as num).toInt(),
          client: _client,
          preferences: preferences,
        ),
      );
    } catch (_) {
      final context = widget.navigatorKey.currentContext;
      if (force && mounted && context != null && context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context).updateFailed)),
        );
      }
      // Offline and API rate limits never interrupt app startup or reading.
    } finally {
      _busy = false;
    }
  }

  Future<void> _checkIos(
    Map<String, dynamic> info,
    SharedPreferences preferences,
    bool force,
  ) async {
    final channel = info['channel'];
    final installedBuild = info['buildNumber'];
    if (installedBuild is! num) return;
    if (channel == 'ios-testflight') {
      final catalog = await _client.loadHistory();
      if (catalog == null) {
        if (force && mounted) {
          final context = widget.navigatorKey.currentContext;
          if (context != null && context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(AppLocalizations.of(context).updateFailed),
              ),
            );
          }
        }
        return;
      }
      final target = catalog.releases
          .where(
            (release) =>
                release.channels.contains('ios-testflight') &&
                release.buildNumber > installedBuild.toInt() &&
                catalog.channelCoverage['ios-testflight']?.coveredBuilds
                        .contains(release.buildNumber) ==
                    true,
          )
          .fold<ReleaseNoteVersion?>(
            null,
            (latest, release) =>
                latest == null || release.buildNumber > latest.buildNumber
                ? release
                : latest,
          );
      if (!mounted) return;
      final context = widget.navigatorKey.currentContext;
      if (context == null || !context.mounted) return;
      if (target == null) {
        if (force) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(AppLocalizations.of(context).updateLatest)),
          );
        }
        return;
      }
      if (!force &&
          preferences.getInt('update.skippedIosBuild') == target.buildNumber) {
        return;
      }
      await showGfBottomSheet<void>(
        context,
        barrierDismissible: false,
        enableDrag: false,
        builder: (_) => _IosUpdateDialog(
          testFlight: true,
          version: target.version,
          targetBuild: target.buildNumber,
          listing: null,
          installedVersion: info['version'] as String?,
          installedBuild: installedBuild.toInt(),
          client: _client,
          preferences: preferences,
        ),
      );
      return;
    }
    if (channel != 'ios-app-store') {
      if (force && mounted) {
        final context = widget.navigatorKey.currentContext;
        if (context != null && context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(AppLocalizations.of(context).updateChannelUnknown),
            ),
          );
        }
      }
      return;
    }
    final installedVersion = info['version'];
    if (installedVersion is! String) return;
    final listing = await IosStoreListing.lookup();
    if (!mounted) return;
    final context = widget.navigatorKey.currentContext;
    if (context == null || !context.mounted) return;
    if (listing == null) {
      if (force) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context).updateFailed)),
        );
      }
      return;
    }
    if (!listing.isNewerThan(installedVersion)) {
      if (force) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context).updateLatest)),
        );
      }
      return;
    }
    if (!force &&
        preferences.getString('update.skippedIosVersion') == listing.version) {
      return;
    }
    await showGfBottomSheet<void>(
      context,
      barrierDismissible: false,
      enableDrag: false,
      builder: (_) => _IosUpdateDialog(
        testFlight: false,
        version: listing.version,
        targetBuild: null,
        listing: listing,
        installedVersion: installedVersion,
        installedBuild: installedBuild.toInt(),
        client: _client,
        preferences: preferences,
      ),
    );
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

class _UpdateDialog extends StatefulWidget {
  const _UpdateDialog({
    required this.release,
    required this.installedVersion,
    required this.installedBuild,
    required this.client,
    required this.preferences,
  });
  final AndroidRelease release;
  final String? installedVersion;
  final int installedBuild;
  final AndroidReleaseClient client;
  final SharedPreferences preferences;

  @override
  State<_UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<_UpdateDialog> {
  late AndroidRelease _release = widget.release;
  CancelToken? _cancel;
  bool _working = false;
  bool _failed = false;
  bool _needsPermission = false;
  int _received = 0;
  File? _apk;

  @override
  void initState() {
    super.initState();
    _loadNotes();
  }

  Future<void> _loadNotes() async {
    try {
      final cached = await widget.client.withNotes(
        widget.release,
        widget.installedBuild,
        refresh: false,
      );
      if (mounted) setState(() => _release = cached);
      final refreshed = await widget.client.withNotes(
        widget.release,
        widget.installedBuild,
      );
      if (mounted) setState(() => _release = refreshed);
    } catch (_) {
      // Display-only metadata cannot block or interrupt the update prompt.
    }
  }

  @override
  void dispose() {
    _cancel?.cancel();
    super.dispose();
  }

  Future<void> _download() async {
    setState(() {
      _working = true;
      _failed = false;
      _received = 0;
    });
    final cancel = _cancel = CancelToken();
    try {
      final temporary = await getTemporaryDirectory();
      final file = await widget.client.download(
        _release,
        Directory('${temporary.path}/updates'),
        cancel: cancel,
        onProgress: (received, _) {
          if (mounted) setState(() => _received = received);
        },
      );
      if (mounted) setState(() => _apk = file);
    } catch (_) {
      if (mounted && !cancel.isCancelled) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _working = false);
    }
  }

  Future<void> _install() async {
    try {
      final opened = await updateChannel.invokeMethod<bool>('install', {
        'path': _apk!.path,
      });
      if (!mounted) return;
      if (opened == true) {
        Navigator.pop(context);
      } else {
        setState(() => _needsPermission = true);
      }
    } catch (_) {
      if (mounted) {
        setState(() {
          _apk = null;
          _failed = true;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    Future<void> skip() async {
      await widget.preferences.setInt(
        'update.skippedBuild',
        _release.buildNumber,
      );
      if (context.mounted) Navigator.pop(context);
    }

    return UpdatePromptSheet(
      version: _release.version,
      installedVersion: widget.installedVersion,
      sizeBytes: _release.size,
      notes: _release.notes,
      historyComplete: _release.hasCompleteHistory,
      working: _working,
      failed: _failed,
      needsPermission: _needsPermission,
      ready: _apk != null,
      receivedBytes: _received,
      failureText: l10n.updateIncomplete,
      primaryLabel: _needsPermission
          ? l10n.updateOpenPermissionSettings
          : _apk != null
          ? l10n.updateInstall
          : _failed
          ? l10n.updateRetry
          : l10n.updateDownload,
      onPrimary: _apk == null ? _download : _install,
      // Cancelling keeps the prompt open; the download task ends without error.
      onCancel: () => _cancel?.cancel(),
      onLater: () => Navigator.pop(context),
      onSkip: _apk == null ? skip : null,
    );
  }
}

class _IosUpdateDialog extends StatefulWidget {
  const _IosUpdateDialog({
    required this.testFlight,
    required this.version,
    required this.targetBuild,
    required this.listing,
    required this.installedVersion,
    required this.installedBuild,
    required this.client,
    required this.preferences,
  });
  final bool testFlight;
  final String version;
  final int? targetBuild;
  final IosStoreListing? listing;
  final String? installedVersion;
  final int installedBuild;
  final AndroidReleaseClient client;
  final SharedPreferences preferences;

  @override
  State<_IosUpdateDialog> createState() => _IosUpdateDialogState();
}

class _IosUpdateDialogState extends State<_IosUpdateDialog> {
  List<ReleaseNote> _notes = const [];
  bool _historyComplete = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _loadNotes();
  }

  Future<void> _loadNotes() async {
    try {
      final cached = await widget.client.loadHistory(refresh: false);
      if (mounted && cached != null) _apply(cached);
      final refreshed = await widget.client.loadHistory();
      if (mounted && refreshed != null) _apply(refreshed);
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    }
  }

  void _apply(ReleaseNoteCatalog catalog) {
    final channel = widget.testFlight ? 'ios-testflight' : 'ios-app-store';
    final matches = catalog.releases.where((release) {
      if (!release.channels.contains(channel) ||
          catalog.channelCoverage[channel]?.coveredBuilds.contains(
                release.buildNumber,
              ) !=
              true) {
        return false;
      }
      return widget.testFlight
          ? release.buildNumber == widget.targetBuild
          : release.version == widget.version;
    });
    final target = matches.isEmpty
        ? null
        : matches.reduce((a, b) => a.buildNumber > b.buildNumber ? a : b);
    if (target == null) {
      setState(() {
        _notes = const [];
        _historyComplete = false;
      });
      return;
    }
    setState(() {
      _notes = catalog.notesForUpdate(
        installedBuild: widget.installedBuild,
        targetBuild: target.buildNumber,
        platform: channel,
        channel: channel,
      );
      _historyComplete = catalog.hasCompleteRange(
        installedBuild: widget.installedBuild,
        targetBuild: target.buildNumber,
        channel: channel,
      );
    });
  }

  Future<void> _skip() async {
    if (widget.testFlight) {
      await widget.preferences.setInt(
        'update.skippedIosBuild',
        widget.targetBuild!,
      );
    } else {
      await widget.preferences.setString(
        'update.skippedIosVersion',
        widget.version,
      );
    }
    if (mounted) Navigator.pop(context);
  }

  Future<void> _openStore() async {
    try {
      final destination = widget.testFlight
          ? Uri.parse(testFlightAppUrl)
          : widget.listing!.url;
      if (!await launchUrl(destination, mode: LaunchMode.externalApplication)) {
        throw StateError('App Store is unavailable');
      }
      if (mounted) Navigator.pop(context);
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return UpdatePromptSheet(
      version: widget.version,
      installedVersion: widget.installedVersion,
      notes: _notes,
      historyComplete: _historyComplete,
      failed: _failed,
      channelNote: widget.testFlight ? l10n.updateTestFlightInstructions : null,
      primaryLabel: widget.testFlight
          ? l10n.updateOpenTestFlight
          : l10n.updateOpenAppStore,
      onPrimary: _openStore,
      onLater: () => Navigator.pop(context),
      onSkip: _skip,
    );
  }
}
