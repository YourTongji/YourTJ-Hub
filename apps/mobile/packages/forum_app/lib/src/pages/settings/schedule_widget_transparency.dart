import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../campus_widget/schedule_widget_bridge.dart';
import '../../providers.dart';
import '../../server_messages.dart';

/// Appearance and privacy for the home-screen widget live on the same page.
class ScheduleWidgetTransparency extends ConsumerStatefulWidget {
  const ScheduleWidgetTransparency({super.key});
  @override
  ConsumerState<ScheduleWidgetTransparency> createState() =>
      _ScheduleWidgetTransparencyState();
}

class _ScheduleWidgetTransparencyState
    extends ConsumerState<ScheduleWidgetTransparency> {
  int _value = ScheduleWidgetBridge.defaultTransparencyPercent;
  int _saved = ScheduleWidgetBridge.defaultTransparencyPercent;
  bool _loaded = false;
  int _revision = 0;
  Future<void> _writes = Future.value();
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final value = await ref
          .read(scheduleWidgetBridgeProvider)
          .readTransparency();
      if (mounted) {
        setState(() {
          _value = value;
          _saved = value;
        });
      }
    } catch (_) {
      // Missing platform preferences must not prevent using the settings page.
    } finally {
      if (mounted) setState(() => _loaded = true);
    }
  }

  Future<void> _save(int value) {
    final revision = ++_revision;
    final bridge = ref.read(scheduleWidgetBridgeProvider);
    // Keep native writes in gesture order, including work already submitted
    // when the page closes. Every success becomes the rollback baseline, but
    // only the newest gesture may change the visible preview after a failure.
    return _writes = _writes.then((_) async {
      try {
        await bridge.setTransparency(value);
        _saved = value;
      } catch (error) {
        if (!mounted || revision != _revision) return;
        setState(() => _value = _saved);
        showGfToast(
          context,
          resolveErrorMessage(AppLocalizations.of(context), error),
          error: true,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l.scheduleWidgetTransparencyTitle,
            style: GfTheme.typographyOf(context).bodyStrong,
          ),
          Slider(
            value: _value.toDouble(),
            min: ScheduleWidgetBridge.minTransparencyPercent.toDouble(),
            max: ScheduleWidgetBridge.maxTransparencyPercent.toDouble(),
            divisions:
                ScheduleWidgetBridge.maxTransparencyPercent -
                ScheduleWidgetBridge.minTransparencyPercent,
            label: '$_value%',
            semanticFormatterCallback: (value) =>
                '${l.scheduleWidgetTransparencyTitle}, ${value.round()}%',
            onChanged: !_loaded
                ? null
                : (value) => setState(() {
                    _revision++;
                    _value = value.round();
                  }),
            onChangeEnd: !_loaded ? null : (value) => _save(value.round()),
          ),
          Text(
            '$_value% · ${l.scheduleWidgetTransparencyDescription}',
            style: GfTheme.typographyOf(
              context,
            ).caption.copyWith(color: GfTheme.colorsOf(context).iconMuted),
          ),
        ],
      ),
    );
  }
}
