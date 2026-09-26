import 'package:flutter/material.dart';
import '../../l10n/app_localizations.dart';

Future<bool> confirmDiscardEdit(BuildContext context) async {
  final l = AppLocalizations.of(context);
  return await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(l.settingsUnsavedTitle),
          content: Text(l.settingsUnsavedBody),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(l.settingsKeepEditing),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(l.settingsDiscardChanges),
            ),
          ],
        ),
      ) ??
      false;
}
