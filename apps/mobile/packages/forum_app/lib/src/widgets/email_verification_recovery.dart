import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../l10n/app_localizations.dart';

/// Opens current-account settings without replaying a write or replacing its page.
class EmailVerificationRecovery extends StatefulWidget {
  const EmailVerificationRecovery({
    this.error,
    this.preserveDraft,
    this.compact = false,
    super.key,
  });
  final ApiException? error;
  final bool compact;
  final Future<bool> Function()? preserveDraft;
  @override
  State<EmailVerificationRecovery> createState() =>
      _EmailVerificationRecoveryState();
}

class _EmailVerificationRecoveryState extends State<EmailVerificationRecovery> {
  bool _opening = false;
  Future<void> _open() async {
    if (_opening) {
      return;
    }
    setState(() => _opening = true);
    try {
      if (widget.preserveDraft != null && !await widget.preserveDraft!()) {
        return;
      }
      if (mounted) {
        await context.push('/settings/account');
      }
    } catch (_) {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).draftLocalSaveFailed,
          error: true,
        );
      }
    } finally {
      if (mounted) {
        setState(() => _opening = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.error?.messageCode != 'permission.emailRequired') {
      return const SizedBox.shrink();
    }
    final l = AppLocalizations.of(context);
    return Semantics(
      container: true,
      liveRegion: true,
      hint: widget.compact ? l.authWriteRecoveryHint : null,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          GfButton(
            label: l.authCheckEmailVerification,
            variant: GfButtonVariant.outline,
            loading: _opening,
            onPressed: _opening ? null : _open,
          ),
          if (!widget.compact) ...[
            const SizedBox(height: 8),
            GfText(l.authWriteRecoveryHint),
          ],
        ],
      ),
    );
  }
}
