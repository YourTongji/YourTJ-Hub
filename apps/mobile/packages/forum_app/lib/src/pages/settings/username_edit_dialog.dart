import 'package:flutter/material.dart';

import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';

/// Keep server validation alongside the input so a rejected name can be fixed
/// without reopening the form or losing the user's text.
class UsernameEditDialog extends StatefulWidget {
  const UsernameEditDialog({
    super.key,
    required this.username,
    required this.onSave,
  });

  final String username;
  final Future<void> Function(String) onSave;

  @override
  State<UsernameEditDialog> createState() => _UsernameEditDialogState();
}

class _UsernameEditDialogState extends State<UsernameEditDialog> {
  late final _controller = TextEditingController(text: widget.username);
  bool _saving = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_saving) return;
    final l10n = AppLocalizations.of(context);
    final name = _controller.text.trim();
    if (name.isEmpty) {
      setState(() => _error = l10n.settingsFillComplete);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave(name);
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) setState(() => _error = resolveErrorMessage(l10n, error));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_saving,
      child: AlertDialog(
        title: Text(l10n.authUsername),
        scrollable: true,
        content: TextField(
          key: const Key('settings-username-input'),
          controller: _controller,
          enabled: !_saving,
          autocorrect: false,
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => _save(),
          decoration: InputDecoration(
            labelText: l10n.authUsername,
            helperText: l10n.settingsUsernameHint,
            helperMaxLines: 3,
            errorText: _error,
            errorMaxLines: 4,
          ),
        ),
        actions: [
          TextButton(
            onPressed: _saving ? null : () => Navigator.pop(context),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: _saving ? null : _save,
            child: Text(_saving ? l10n.commonLoading : l10n.commonSave),
          ),
        ],
      ),
    );
  }
}
