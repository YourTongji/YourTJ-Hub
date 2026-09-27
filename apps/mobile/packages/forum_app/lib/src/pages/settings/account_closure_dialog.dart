import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';

class AccountClosureDialog extends StatefulWidget {
  const AccountClosureDialog({super.key});
  @override
  State<AccountClosureDialog> createState() => _AccountClosureDialogState();
}

class _AccountClosureDialogState extends State<AccountClosureDialog> {
  final _password = TextEditingController();
  String _mode = 'anonymize';
  @override
  void dispose() {
    _password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AlertDialog(
      title: Text(l10n.settingsCloseAccount),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.settingsCloseAccountWarning),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              initialValue: _mode,
              isExpanded: true,
              itemHeight: null,
              borderRadius: BorderRadius.circular(16),
              icon: const GfSymbol('chevron-down', size: 20),
              items: [
                DropdownMenuItem(
                  value: 'anonymize',
                  child: Text(l10n.settingsCloseKeepContent),
                ),
                DropdownMenuItem(
                  value: 'delete',
                  child: Text(l10n.settingsCloseDeleteContent),
                ),
              ],
              onChanged: (value) {
                if (value != null) setState(() => _mode = value);
              },
            ),
            const SizedBox(height: 16),
            GfInput(
              controller: _password,
              obscureText: true,
              enableSuggestions: false,
              autocorrect: false,
              autofillHints: const [AutofillHints.password],
              textInputAction: TextInputAction.done,
              onChanged: (_) => setState(() {}),
              labelText: l10n.authPassword,
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.commonCancel),
        ),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
          onPressed: _password.text.isEmpty
              ? null
              : () => Navigator.pop(context, (
                  mode: _mode,
                  password: _password.text,
                )),
          child: Text(l10n.settingsCloseAccount),
        ),
      ],
    );
  }
}
