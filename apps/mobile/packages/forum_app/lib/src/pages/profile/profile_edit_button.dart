import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';

class ProfileEditButton extends StatelessWidget {
  const ProfileEditButton({super.key, required this.onPressed, this.label});
  final VoidCallback onPressed;
  final String? label;
  @override
  Widget build(BuildContext context) => OutlinedButton(
    style: OutlinedButton.styleFrom(
      minimumSize: const Size(96, 44),
      shape: const StadiumBorder(),
      foregroundColor: GfTheme.colorsOf(context).baseContent,
      side: BorderSide(color: GfTheme.colorsOf(context).line),
      textStyle: Theme.of(context).textTheme.labelLarge?.copyWith(
        fontSize: 14,
        fontWeight: FontWeight.w700,
      ),
    ),
    onPressed: onPressed,
    child: Text(
      label ?? AppLocalizations.of(context).settingsEditProfile,
      textAlign: TextAlign.center,
    ),
  );
}
