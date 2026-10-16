import 'package:ui_kit/ui_kit.dart';
import 'package:flutter/material.dart';
import '../../l10n/app_localizations.dart';

Future<bool> confirmDiscardEdit(BuildContext context) async {
  final l = AppLocalizations.of(context);
  return await showGfAlertDialog<bool>(
        context,
        builder: (context) => GfAlertDialog(
          title: GfText(l.settingsUnsavedTitle),
          content: GfText(l.settingsUnsavedBody),
          actions: [
            GfDialogAction(
              onPressed: () => Navigator.pop(context, false),
              child: GfText(l.settingsKeepEditing),
            ),
            GfDialogAction(
              destructive: true,
              onPressed: () => Navigator.pop(context, true),
              child: GfText(l.settingsDiscardChanges),
            ),
          ],
        ),
      ) ??
      false;
}
