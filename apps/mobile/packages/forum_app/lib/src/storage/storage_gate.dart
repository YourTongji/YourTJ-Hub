import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'storage_providers.dart';
import 'storage_strings.dart';

/// Recovered data is usable only after both startup recovery and any active
/// reset finish. A refreshing AsyncValue may carry old data; that is not ready.
final storageReadyProvider = Provider<bool>((ref) {
  final bootstrap = ref.watch(storageBootstrapProvider);
  final reset = ref.watch(storageResetStateProvider);
  return reset == null &&
      bootstrap.hasValue &&
      !bootstrap.isLoading &&
      !bootstrap.hasError;
});

/// Keeps routes and their side effects unmounted while destructive recovery is
/// incomplete. An overlay would still allow hidden pages to create new work.
class StorageGate extends ConsumerWidget {
  const StorageGate({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (ref.watch(storageReadyProvider)) return child;
    final bootstrap = ref.watch(storageBootstrapProvider);
    final reset = ref.watch(storageResetStateProvider);
    final loading = bootstrap.isLoading || reset?.isLoading == true;
    final failed = !loading && (bootstrap.hasError || reset?.hasError == true);
    final strings = StorageStrings(context);
    final colors = GfTheme.colorsOf(context);
    final typography = GfTheme.typographyOf(context);

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 440),
              child: Semantics(
                liveRegion: true,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (failed)
                      GfSymbol('circle-alert', size: 32, color: colors.error)
                    else
                      const GfLoadingIndicator(),
                    const SizedBox(height: 20),
                    Text(
                      failed
                          ? strings.recoveryBlocked
                          : reset != null
                          ? strings.resetting
                          : strings.preparing,
                      textAlign: TextAlign.center,
                      style: typography.title2,
                    ),
                    if (failed) ...[
                      const SizedBox(height: 12),
                      Text(
                        strings.recoveryRetryHint,
                        textAlign: TextAlign.center,
                        style: typography.body,
                      ),
                      const SizedBox(height: 24),
                      GfButton(
                        key: const Key('storage-gate-retry'),
                        label: strings.retry,
                        onPressed: () =>
                            ref.invalidate(storageBootstrapProvider),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
