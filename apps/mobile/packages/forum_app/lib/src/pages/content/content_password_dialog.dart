import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';

/// Reauthentication requested by the existing content deletion rate guard.
Future<String?> showContentPasswordDialog(BuildContext context) {
  final l10n = AppLocalizations.of(context);
  var enteredPassword = '';
  return showDialog<String>(
    context: context,
    animationStyle: GfMotion.dialogStyle(context),
    builder: (context) => AlertDialog(
      title: Text(l10n.contentPassword),
      scrollable: true,
      content: GfInput(
        labelText: l10n.contentPassword,
        onChanged: (value) => enteredPassword = value,
        obscureText: true,
        autofocus: true,
        autofillHints: const [AutofillHints.password],
        autocorrect: false,
        enableSuggestions: false,
        textInputAction: TextInputAction.done,
        onSubmitted: (value) => Navigator.pop(context, value),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.commonCancel),
        ),
        TextButton(
          onPressed: () => Navigator.pop(context, enteredPassword),
          child: Text(l10n.commonConfirm),
        ),
      ],
    ),
  );
}
