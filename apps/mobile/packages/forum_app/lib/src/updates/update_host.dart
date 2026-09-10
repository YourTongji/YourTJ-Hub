import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../l10n/app_localizations.dart';
import 'android_release.dart';

const updateChannel = MethodChannel('yourtj/app_updates');
final appUpdateHostKey = GlobalKey<MobileUpdateHostState>();
bool get supportsApkUpdates => !kIsWeb && Platform.isAndroid;

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
    if (!supportsApkUpdates || _busy || !mounted) return;
    _busy = true;
    try {
      final preferences = await SharedPreferences.getInstance();
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
      await showDialog<void>(
        context: context,
        barrierDismissible: false,
        builder: (_) => _UpdateDialog(
          release: release,
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

  @override
  Widget build(BuildContext context) => widget.child;
}

class _UpdateDialog extends StatefulWidget {
  const _UpdateDialog({
    required this.release,
    required this.client,
    required this.preferences,
  });
  final AndroidRelease release;
  final AndroidReleaseClient client;
  final SharedPreferences preferences;

  @override
  State<_UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<_UpdateDialog> {
  CancelToken? _cancel;
  bool _working = false;
  bool _failed = false;
  bool _needsPermission = false;
  int _received = 0;
  File? _apk;

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
        widget.release,
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
    return AlertDialog(
      title: Text('${l10n.updateAvailable} ${widget.release.version}'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('${(widget.release.size / 1024 / 1024).toStringAsFixed(1)} MB'),
          const SizedBox(height: 12),
          if (_working) ...[
            Text(
              _received == 0 ? l10n.updatePreparing : l10n.updateDownloading,
            ),
            const SizedBox(height: 12),
            LinearProgressIndicator(
              value: _received == 0 ? null : _received / widget.release.size,
            ),
          ] else if (_failed)
            Text(l10n.updateFailed)
          else if (_needsPermission)
            Text(l10n.updatePermission)
          else if (_apk != null)
            Text(l10n.updateReady),
        ],
      ),
      actions: [
        if (!_working && _apk == null)
          TextButton(
            onPressed: () async {
              await widget.preferences.setInt(
                'update.skippedBuild',
                widget.release.buildNumber,
              );
              if (context.mounted) Navigator.pop(context);
            },
            child: Text(l10n.updateSkip),
          ),
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(_working ? l10n.commonCancel : l10n.updateLater),
        ),
        if (!_working)
          FilledButton(
            onPressed: _apk == null ? _download : _install,
            child: Text(
              _apk != null
                  ? l10n.updateInstall
                  : _failed
                  ? l10n.updateRetry
                  : l10n.updateDownload,
            ),
          ),
      ],
    );
  }
}
