import 'package:flutter/foundation.dart';

/// Selects the platform-specific passkey capability probe without invoking any
/// platform API on unsupported targets.
Future<bool> resolvePasskeyAvailability({
  required bool isWeb,
  required TargetPlatform platform,
  required Future<bool> Function() web,
  required Future<bool> Function() android,
  required Future<bool> Function() ios,
}) {
  if (isWeb) return web();
  return switch (platform) {
    TargetPlatform.android => android(),
    TargetPlatform.iOS => ios(),
    _ => Future<bool>.value(false),
  };
}
