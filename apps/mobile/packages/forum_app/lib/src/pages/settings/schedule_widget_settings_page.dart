import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../app_config.dart';
import '../../campus_widget/schedule_widget_projection.dart';
import '../../current_user.dart';
import '../../offline/campus_snapshot_store.dart';
import '../../providers.dart';

class ScheduleWidgetSettingsPage extends ConsumerStatefulWidget {
  const ScheduleWidgetSettingsPage({super.key});

  @override
  ConsumerState<ScheduleWidgetSettingsPage> createState() =>
      _ScheduleWidgetSettingsPageState();
}

class _ScheduleWidgetSettingsPageState
    extends ConsumerState<ScheduleWidgetSettingsPage> {
  static const _privacyKey = 'schedule_widget_privacy_ack';
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _ackPrivacy());
  }

  Future<bool> _ackPrivacy() async {
    final preferences = await SharedPreferences.getInstance();
    if (preferences.getBool(_privacyKey) == true || !mounted) return true;
    final l = AppLocalizations.of(context);
    final accepted = await showGfAlertDialog<bool>(
      context,
      builder: (context) => GfAlertDialog(
        title: Text(l.scheduleWidgetSettingsTitle),
        content: Text(l.scheduleWidgetPrivacyDescription),
        actions: [
          GfButton(
            label: l.commonCancel,
            variant: GfButtonVariant.ghost,
            onPressed: () => Navigator.pop(context, false),
          ),
          GfButton(
            label: l.commonConfirm,
            onPressed: () => Navigator.pop(context, true),
          ),
        ],
      ),
    );
    if (accepted == true) await preferences.setBool(_privacyKey, true);
    return accepted == true;
  }

  Future<void> _refresh() async {
    if (_busy || !await _ackPrivacy()) return;
    setState(() => _busy = true);
    try {
      final user = await ref.read(currentUserProvider.future);
      if (user == null) throw StateError('No signed-in account');
      final scope = CampusCacheScope(
        site: Uri.parse(
          AppConfig.apiBaseUrl.isNotEmpty
              ? AppConfig.apiBaseUrl
              : GfApiClient.defaultBaseUrl,
        ).origin,
        accountId: user.id,
      );
      final snapshot = await ref.read(campusSnapshotStoreProvider).read(scope);
      if (snapshot == null) throw StateError('No offline campus snapshot');
      CampusCalendarRules? calendarRules;
      try {
        calendarRules =
            (await ref.read(campusRepositoryProvider).calendarRules()).rules;
      } catch (_) {
        // Keep today's authoritative data and leave unresolved future days unknown.
      }
      await ref
          .read(scheduleWidgetBridgeProvider)
          .write(
            ScheduleWidgetProjection.fromSnapshot(
              snapshot,
              scope,
              calendarRules: calendarRules,
            ),
          );
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).scheduleWidgetRefresh,
        );
      }
    } catch (error) {
      if (mounted) showGfToast(context, '$error', error: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _clear() async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await ref.read(scheduleWidgetBridgeProvider).clear();
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).scheduleWidgetCleared,
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l.scheduleWidgetSettingsTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          GfCard(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Text(l.scheduleWidgetPrivacyDescription),
            ),
          ),
          const SizedBox(height: 12),
          GfSettingRow(
            symbol: 'refresh-cw',
            title: l.scheduleWidgetRefresh,
            onTap: _busy ? null : _refresh,
          ),
          GfSettingRow(
            symbol: 'trash-2',
            title: l.scheduleWidgetClear,
            onTap: _busy ? null : _clear,
          ),
          const SizedBox(height: 24),
          Text(
            l.scheduleWidgetDiagnostics,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          Text(l.scheduleWidgetDiagnosticsDescription),
        ],
      ),
    );
  }
}
