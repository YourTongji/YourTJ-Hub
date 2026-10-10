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

class _ActivationEmailActionState extends ConsumerState<ActivationEmailAction> {
  CancelToken? _request;
  ApiException? _feedback;
  bool _failed = false;

  @override
  void dispose() {
    _request?.cancel();
    super.dispose();
  }

  Future<void> _resend() async {
    if (_request != null) return;
    final request = CancelToken();
    final epoch = ref.read(offlineCacheEpochProvider);
    bool active() => mounted && epoch == ref.read(offlineCacheEpochProvider);
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
      if (active()) setState(() => _request = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(offlineCacheEpochProvider, (_, _) {
      _request?.cancel();
      setState(() {
        _request = null;
        _feedback = null;
      });
    });
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfButton(
          label: l10n.authResendActivationEmail,
          variant: GfButtonVariant.outline,
          loading: _request != null,
          onPressed: _request != null ? null : _resend,
        ),
        if (_feedback != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Semantics(
              liveRegion: true,
              child: Text(
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
