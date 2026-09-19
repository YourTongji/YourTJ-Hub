import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import 'campus_helpers.dart';
import 'campus_calendar_rules.dart';
import 'campus_state.dart';

class CampusCalendarExportButton extends ConsumerStatefulWidget {
  const CampusCalendarExportButton({super.key, required this.enabled});
  final bool enabled;
  @override
  ConsumerState<CampusCalendarExportButton> createState() =>
      _CampusCalendarExportButtonState();
}

class _CampusCalendarExportButtonState
    extends ConsumerState<CampusCalendarExportButton> {
  bool _busy = false;
  bool _applyAdjustments = true;
  String? _error;

  Future<void> _export(BuildContext anchor) async {
    if (_busy) return;
    final l = AppLocalizations.of(context);
    final revision = ref
        .read(campusControllerProvider)
        .status
        ?.binding
        ?.revision;
    setState(() {
      _busy = true;
      _error = null;
    });
    Directory? temporary;
    bool active() =>
        mounted &&
        ref.read(campusControllerProvider).status?.binding?.revision ==
            revision;
    try {
      final result = await ref
          .read(campusControllerProvider.notifier)
          .exportCalendar(applyAdjustments: _applyAdjustments);
      if (result == null || !active()) return;
      final root = await getTemporaryDirectory();
      if (!active()) return;
      temporary = await root.createTemp('yourtj-campus-calendar-');
      // Fixed local name: upstream text cannot become a filesystem path.
      final file = File('${temporary.path}/yourtj-courses.ics');
      await file.writeAsString(result.content, flush: true);
      if (!active() || !anchor.mounted) return;
      final box = anchor.findRenderObject() as RenderBox;
      await SharePlus.instance.share(
        ShareParams(
          files: [XFile(file.path, mimeType: 'text/calendar')],
          subject: l.campusExportCalendar,
          sharePositionOrigin: box.localToGlobal(Offset.zero) & box.size,
        ),
      );
    } catch (e) {
      if (active()) setState(() => _error = campusError(l, e));
    } finally {
      // Only the explicitly exported copy is shared. Our staging file is removed
      // on completion/cancel/failure; recipient/OS share copies are user-owned.
      if (temporary != null) {
        try {
          await temporary.delete(recursive: true);
        } on FileSystemException {
          /* OS temporary storage cleanup remains the fallback. */
        }
      }
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SwitchListTile.adaptive(
          contentPadding: EdgeInsets.zero,
          title: Text(l.campusApplyAdjustments),
          subtitle: Text(l.campusApplyAdjustmentsHint),
          value: _applyAdjustments,
          onChanged: _busy
              ? null
              : (value) => setState(() => _applyAdjustments = value),
        ),
        Builder(
          builder: (anchor) => GfButton(
            label: _busy ? l.campusExportingCalendar : l.campusExportCalendar,
            variant: GfButtonVariant.secondary,
            onPressed:
                widget.enabled &&
                    !_busy &&
                    !state.busy &&
                    !state.needsAuthorization
                ? () => _export(anchor)
                : null,
          ),
        ),
        const CampusCalendarRulesView(),
        const SizedBox(height: 8),
        Text(
          l.campusExportCalendarHint,
          style: GfTheme.typographyOf(context).caption,
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Semantics(
              liveRegion: true,
              child: Text(
                _error!,
                style: TextStyle(color: GfTheme.colorsOf(context).error),
              ),
            ),
          ),
      ],
    );
  }
}
