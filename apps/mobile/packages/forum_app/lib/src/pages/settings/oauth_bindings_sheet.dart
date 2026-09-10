import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';

Future<bool> _openBrowser(Uri uri) =>
    launchUrl(uri, mode: LaunchMode.externalApplication);

/// Provider authentication stays in the browser. No native credential is
/// transferred; the user signs into the matching Web account before connecting.
class OAuthBindingsSheet extends ConsumerStatefulWidget {
  const OAuthBindingsSheet({
    super.key,
    required this.username,
    required this.googleReady,
    this.openBrowser = _openBrowser,
  });
  final String username;
  final bool googleReady;
  final Future<bool> Function(Uri) openBrowser;
  @override
  ConsumerState<OAuthBindingsSheet> createState() => _OAuthBindingsSheetState();
}

class _OAuthBindingsSheetState extends ConsumerState<OAuthBindingsSheet>
    with WidgetsBindingObserver {
  AsyncValue<Map<String, OAuthBindingPayload>> _bindings = const AsyncLoading();
  bool _busy = false;
  bool _browserOpened = false;
  int _request = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _load();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && _browserOpened) {
      _browserOpened = false;
      _load();
    }
  }

  Future<void> _load() async {
    final request = ++_request;
    setState(() => _bindings = const AsyncLoading());
    try {
      final bindings = await ref
          .read(userRepositoryProvider)
          .getOAuthBindings();
      if (mounted && request == _request) {
        setState(() => _bindings = AsyncData(bindings));
      }
    } catch (error, stack) {
      if (mounted && request == _request) {
        setState(() => _bindings = AsyncError(error, stack));
      }
    }
  }

  Future<void> _connect() async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      final base = Uri.parse(ref.read(apiClientProvider).baseUrl);
      if (!['https', 'http'].contains(base.scheme) ||
          base.host.isEmpty ||
          base.userInfo.isNotEmpty) {
        throw StateError('Invalid site URL');
      }
      _browserOpened = true;
      if (!await widget.openBrowser(base.resolve('/settings?tab=binding'))) {
        _browserOpened = false;
        throw StateError('Could not open browser');
      }
    } catch (error) {
      _browserOpened = false;
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _unbind(String provider) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await ref.read(userRepositoryProvider).unbindOAuth(provider);
      if (mounted) await _load();
    } catch (error) {
      if (mounted) {
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(offlineCacheEpochProvider, (_, _) {
      // The sheet labels one account; never reuse it after a session switch.
      if (mounted) Navigator.pop(context);
    });
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.settingsOAuthBindings,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
                IconButton(
                  onPressed: _busy ? null : _load,
                  icon: const Icon(Icons.refresh),
                  tooltip: l10n.commonRefresh,
                ),
              ],
            ),
            const SizedBox(height: 12),
            _bindings.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (error, _) => GfErrorRetry(
                message: resolveErrorMessage(l10n, error),
                onRetry: _load,
              ),
              data: (bindings) => Column(
                children: [
                  for (final (key, name, supported) in [
                    ('github', 'GitHub', true),
                    ('google', 'Google', widget.googleReady),
                  ])
                    GfSettingRow(
                      icon: Icons.link,
                      title: name,
                      description: bindings[key]?.bound == true
                          ? l10n.settingsBound
                          : supported
                          ? l10n.settingsUnbound
                          : l10n.settingsOAuthUnavailable,
                      trailing: bindings[key]?.bound == true
                          ? TextButton(
                              onPressed: _busy ? null : () => _unbind(key),
                              child: Text(l10n.settingsUnbind),
                            )
                          : null,
                    ),
                  const SizedBox(height: 16),
                  Text(
                    l10n.settingsOAuthBrowserHint(widget.username),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 12),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton.icon(
                      onPressed: _busy ? null : _connect,
                      icon: const Icon(Icons.open_in_new),
                      label: Text(l10n.settingsOAuthOpenBrowser),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
