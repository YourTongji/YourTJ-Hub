import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../current_user.dart';
import '../../widgets/status_views.dart';
import 'campus_connection.dart';
import 'campus_helpers.dart';
import 'campus_private_surface.dart';
import 'campus_state.dart';

/// Connection management is isolated from the five school-content tabs.
class CampusSettingsPage extends StatelessWidget {
  const CampusSettingsPage({super.key});
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: GfAppBar(
      title: GfText(AppLocalizations.of(context).campusSettings),
    ),
    body: CampusPrivateSurface(
      builder: (_) => ProviderScope(
        overrides: [
          campusControllerProvider.overrideWith(
            (ref) => createCampusController(ref, initialTab: 'connection'),
          ),
        ],
        child: const _ConnectionSettings(),
      ),
    ),
  );
}

class _ConnectionSettings extends ConsumerWidget {
  const _ConnectionSettings();
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final user = ref.watch(currentUserProvider);
    if (user.isLoading) return const GfLoading();
    if (user.valueOrNull == null) return GfEmpty(message: l.authLoginTitle);
    final state = ref.watch(campusControllerProvider);
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        if (state.loading)
          const GfLoading()
        else if (state.status == null)
          GfErrorRetry(
            message: campusError(l, state.error),
            onRetry: () =>
                ref.read(campusControllerProvider.notifier).refresh(),
          )
        else
          const CampusConnection(),
      ],
    );
  }
}
