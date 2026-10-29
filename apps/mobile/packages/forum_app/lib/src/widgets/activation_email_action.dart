import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';
import '../providers.dart';
import '../server_messages.dart';

class ActivationEmailAction extends ConsumerStatefulWidget {
  const ActivationEmailAction({super.key});
  @override
  ConsumerState<ActivationEmailAction> createState() =>
      _ActivationEmailActionState();
}

class _ActivationEmailActionState extends ConsumerState<ActivationEmailAction>
    with WidgetsBindingObserver {
  CancelToken? _request;
  ApiException? _feedback;
  ActivationStatus? _status;
  bool _failed = false;
  bool _statusFailed = false;
  int _generation = 0;
  int _seconds = 0;
  Timer? _timer;
  final _elapsed = Stopwatch();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    scheduleMicrotask(_refresh);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && _request == null) _refresh();
  }

  @override
  void dispose() {
    _generation++;
    _request?.cancel();
    _timer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  Future<void> _refresh() async {
    if (!mounted) return;
    final generation = ++_generation;
    final epoch = ref.read(offlineCacheEpochProvider);
    bool active() =>
        mounted &&
        generation == _generation &&
        epoch == ref.read(offlineCacheEpochProvider);
    _request?.cancel();
    _timer?.cancel();
    final request = CancelToken();
    setState(() {
      _request = request;
      _status = null;
    });
    try {
      final status = await ref
          .read(authRepositoryProvider)
          .getActivationStatus(cancelToken: request);
      if (!active()) return;
      setState(() {
        _status = status;
        _seconds = status.retryAfterSeconds;
        if (_statusFailed) _feedback = null;
        _statusFailed = false;
      });
      _elapsed
        ..reset()
        ..start();
      if (_seconds > 0) {
        _timer = Timer.periodic(const Duration(seconds: 1), (_) {
          if (!active()) return;
          setState(
            () => _seconds =
                (status.retryAfterSeconds - _elapsed.elapsed.inSeconds).clamp(
                  0,
                  status.retryAfterSeconds,
                ),
          );
          if (_seconds == 0) {
            _timer?.cancel();
            _refresh();
          }
        });
      }
    } catch (error) {
      if (!active()) return;
      setState(() {
        _statusFailed = true;
        _failed = true;
        _feedback = error is ApiException
            ? error
            : const ApiException(
                fallbackMessage: '',
                messageCode: 'auth.activation.resendFailed',
              );
      });
    } finally {
      if (active()) setState(() => _request = null);
    }
  }

  Future<void> _resend() async {
    if (_request != null || _status?.state != 'ready') return;
    final request = CancelToken();
    final epoch = ref.read(offlineCacheEpochProvider);
    final generation = _generation;
    bool active() =>
        mounted &&
        generation == _generation &&
        epoch == ref.read(offlineCacheEpochProvider);
    setState(() {
      _request = request;
      _feedback = null;
    });
    try {
      final response = await ref
          .read(authRepositoryProvider)
          .resendActivationEmail(cancelToken: request);
      if (!active()) return;
      setState(() {
        _failed = false;
        _feedback = ApiException(
          fallbackMessage: '',
          messageCode: response.messageCode ?? 'auth.activation.resendSuccess',
          params: response.params,
        );
      });
    } catch (error) {
      if (!active()) return;
      setState(() {
        _failed = true;
        _feedback = error is ApiException
            ? error
            : const ApiException(
                fallbackMessage: '',
                messageCode: 'auth.activation.resendFailed',
              );
      });
    } finally {
      if (active()) await _refresh();
    }
  }

  String _stateMessage(AppLocalizations l) => switch (_status?.state) {
    'cooldown' => l.activationStatusCooldown(_seconds),
    'daily_limit' => l.activationStatusDailyLimit,
    'disabled' => l.activationStatusDisabled,
    'verified' => l.activationStatusVerified,
    'frozen' => l.activationStatusFrozen,
    'mail_unavailable' => l.activationStatusMailUnavailable,
    'missing_email' => l.activationStatusMissingEmail,
    _ => '',
  };

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, _) {
      _request?.cancel();
      _timer?.cancel();
      setState(() {
        _feedback = null;
        _statusFailed = false;
        _status = null;
      });
      _refresh();
    });
    final l10n = AppLocalizations.of(context);
    final status = _status;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (status != null) ...[
          if (status.targetEmail.isNotEmpty)
            GfText(l10n.activationStatusTarget(status.targetEmail)),
          GfText(
            l10n.activationStatusRemaining(
              status.remainingToday,
              status.dailyLimit,
            ),
          ),
          if (_stateMessage(l10n).isNotEmpty) GfText(_stateMessage(l10n)),
          const SizedBox(height: 8),
        ],
        GfButton(
          label: l10n.authResendActivationEmail,
          variant: GfButtonVariant.outline,
          loading: _request != null,
          onPressed: _request != null || status?.state != 'ready'
              ? null
              : _resend,
        ),
        GfButton(
          label: l10n.activationStatusRefresh,
          variant: GfButtonVariant.ghost,
          onPressed: _request != null ? null : _refresh,
        ),
        if (_feedback != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Semantics(
              liveRegion: true,
              child: GfText(
                resolveErrorMessage(l10n, _feedback!),
                style: _failed
                    ? TextStyle(color: GfTheme.colorsOf(context).error)
                    : null,
              ),
            ),
          ),
      ],
    );
  }
}
