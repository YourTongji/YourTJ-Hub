import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../l10n/app_localizations.dart';
import '../app_config.dart';

const previewStatusUrl =
    'https://github.com/YourTongji/YourTJ-Hub/releases/tag/dev-latest';
const _previewIndexUrl =
    'https://github.com/YourTongji/YourTJ-Hub/releases/download/dev-latest/dev-status.json';

enum PreviewValidity { current, building, stale, unknown }

/// Informational only: validity follows the source, never binary identity or QA.
PreviewValidity previewValidity(
  Object? data, {
  required String source,
  required String purpose,
  required String artifact,
  required DateTime now,
}) {
  if (data is Map && data['schema'] == 2) {
    final latest = data['source'];
    final platforms = data['platforms'];
    if (purpose != 'dev' ||
        data['updatedAt'] is! int ||
        latest is! String ||
        !RegExp(r'^[0-9a-f]{40}$').hasMatch(latest) ||
        !RegExp(r'^[0-9a-f]{40}$').hasMatch(source) ||
        platforms is! Map) {
      return PreviewValidity.unknown;
    }
    final ready = platforms[artifact];
    if (ready is! Map ||
        ready['source'] is! String ||
        !RegExp(r'^[0-9a-f]{40}$').hasMatch(ready['source'] as String)) {
      return PreviewValidity.unknown;
    }
    if (ready['source'] != source) return PreviewValidity.stale;
    return latest == source
        ? PreviewValidity.current
        : PreviewValidity.building;
  }
  if (data is! Map ||
      data['schema'] != 1 ||
      data['updatedAt'] is! int ||
      !data.containsKey('active')) {
    return PreviewValidity.unknown;
  }
  final seconds = now.millisecondsSinceEpoch ~/ 1000;
  final age = seconds - (data['updatedAt'] as int);
  if (age < -300 || age > const Duration(hours: 26).inSeconds) {
    return PreviewValidity.unknown;
  }
  final active = data['active'];
  if (active == null) return PreviewValidity.stale;
  if (active is! Map ||
      active['source'] is! String ||
      active['phase'] is! String ||
      active['until'] is! int ||
      active['artifacts'] is! Map ||
      !RegExp(r'^[0-9a-f]{40}$').hasMatch(source)) {
    return PreviewValidity.unknown;
  }
  final artifacts = (active['artifacts'] as Map)[purpose];
  return active['source'] == source &&
          active['phase'] == 'testing' &&
          (active['until'] as int) > seconds &&
          artifacts is List &&
          artifacts.contains(artifact)
      ? PreviewValidity.current
      : PreviewValidity.stale;
}

class PreviewStatusHost extends StatefulWidget {
  const PreviewStatusHost({super.key, required this.child, this.fetchStatus});
  final Widget child;
  final Future<Object?> Function()? fetchStatus;

  @override
  State<PreviewStatusHost> createState() => _PreviewStatusHostState();
}

class _PreviewStatusHostState extends State<PreviewStatusHost>
    with WidgetsBindingObserver {
  PreviewValidity _validity = PreviewValidity.unknown;
  final _client = Dio(
    BaseOptions(
      connectTimeout: const Duration(seconds: 8),
      receiveTimeout: const Duration(seconds: 8),
      responseType: ResponseType.plain,
    ),
  );
  Timer? _timer;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _check();
    _timer = Timer.periodic(const Duration(minutes: 5), (_) => _check());
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) _check();
  }

  Future<Object?> _fetch() async {
    // A separate public client carries no forum credentials or user identity.
    final cancel = CancelToken();
    final response = await _client.get<String>(
      _previewIndexUrl,
      queryParameters: {
        'checked': DateTime.now().millisecondsSinceEpoch ~/ 300000,
      },
      cancelToken: cancel,
      onReceiveProgress: (received, total) {
        if (received > 65536 || total > 65536) {
          cancel.cancel('Status too large');
        }
      },
    );
    return jsonDecode(response.data ?? '');
  }

  Future<void> _check() async {
    if (_busy || !mounted) return;
    _busy = true;
    var validity = PreviewValidity.unknown;
    try {
      validity = previewValidity(
        await (widget.fetchStatus ?? _fetch)(),
        source: AppConfig.sourceSha,
        purpose: AppConfig.environment,
        artifact: defaultTargetPlatform == TargetPlatform.iOS
            ? 'ios'
            : 'android',
        now: DateTime.now(),
      );
    } catch (_) {
      // Offline or stale status is unknown, never a pass and never an app lockout.
    } finally {
      _busy = false;
      if (mounted) setState(() => _validity = validity);
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    _client.close(force: true);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_validity == PreviewValidity.current) return widget.child;
    final l10n = AppLocalizations.of(context);
    return Column(
      children: [
        Material(
          color: Theme.of(context).colorScheme.secondaryContainer,
          child: SafeArea(
            bottom: false,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              child: Row(
                children: [
                  Expanded(
                    child: GfText(
                      _validity == PreviewValidity.building
                          ? l10n.previewBuildPending
                          : _validity == PreviewValidity.stale
                          ? l10n.previewStale
                          : l10n.previewStatusUnknown,
                    ),
                  ),
                  TextButton(
                    onPressed: () async {
                      try {
                        await launchUrl(
                          Uri.parse(previewStatusUrl),
                          mode: LaunchMode.externalApplication,
                        );
                      } catch (_) {
                        // Keep the notice and app available when no browser can open.
                      }
                    },
                    child: GfText(l10n.previewStatusLink),
                  ),
                ],
              ),
            ),
          ),
        ),
        Expanded(child: widget.child),
      ],
    );
  }
}
