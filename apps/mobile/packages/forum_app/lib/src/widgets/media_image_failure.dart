import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../app_config.dart';
import '../../l10n/app_localizations.dart';
import '../link_navigation.dart';

/// Why a remote image could not be displayed. The host turns this into copy, so
/// a user can tell "the site refused our request" from "we cannot decode this
/// format" instead of seeing one silent broken-image icon (issue 966 / 图裂).
enum GfMediaFailureKind { network, format, blocked, unknown }

/// Classifies the errors raised by `MediaRepository` and the image codecs.
///
/// Messages are the stable prefixes thrown in
/// `lib/src/storage/media_repository.dart`; anything else is a decode/parse
/// failure from Skia or flutter_svg.
GfMediaFailureKind classifyMediaFailure(Object error) {
  if (error is DioException) return GfMediaFailureKind.network;
  if (error is FormatException) {
    final String message = error.message;
    return message.startsWith('Unsupported image URL') ||
            message.startsWith('Untrusted image origin')
        ? GfMediaFailureKind.blocked
        : GfMediaFailureKind.format;
  }
  if (error is StateError) {
    final String message = error.message;
    if (message.startsWith('Image request failed') ||
        message.startsWith('Image too large') ||
        message.startsWith('Image exceeds download limit') ||
        message.startsWith('Media request deadline')) {
      return GfMediaFailureKind.network;
    }
    if (message.startsWith('Media request invalidated') ||
        message.startsWith('Media loading suspended') ||
        message.startsWith('Media SVG invalidated') ||
        message.startsWith('Invalid media redirect') ||
        message.startsWith('Unsupported media redirect')) {
      return GfMediaFailureKind.blocked;
    }
    return GfMediaFailureKind.format;
  }
  return GfMediaFailureKind.unknown;
}

/// Host fallback wired into `GfMediaScope.imageErrorBuilder`: the shared image
/// components ask for it, the host owns the copy and the actions.
///
/// [retry] re-resolves the failed image after dropping its decoded entry;
/// [url] is the resolved absolute URL, opened through the same confirm flow as
/// any other external link.
Widget mediaImageFailure(
  BuildContext context,
  Object error,
  VoidCallback retry,
  String url,
) {
  final AppLocalizations l10n = AppLocalizations.of(context);
  final GfColors colors = GfTheme.colorsOf(context);
  final GfMediaFailureKind kind = classifyMediaFailure(error);
  final String message = switch (kind) {
    GfMediaFailureKind.network => l10n.imageUnavailableNetwork,
    GfMediaFailureKind.format => l10n.imageUnavailableFormat,
    _ => l10n.imageUnavailable,
  };
  return Padding(
    padding: const EdgeInsets.all(12),
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        GfSymbol('image-off', color: colors.iconMuted, size: 28),
        const SizedBox(height: 8),
        Text(
          message,
          textAlign: TextAlign.center,
          style: TextStyle(
            color: colors.baseContent.withValues(alpha: 0.75),
            fontSize: 13,
            height: 1.35,
          ),
        ),
        const SizedBox(height: 4),
        Wrap(
          alignment: WrapAlignment.center,
          spacing: 4,
          children: <Widget>[
            TextButton(
              onPressed: retry,
              child: Text(l10n.imageRetry),
            ),
            TextButton(
              onPressed: () => LinkNavigation.open(
                context,
                url,
                baseUrl: AppConfig.apiBaseUrl,
              ),
              child: Text(l10n.imageOpenInBrowser),
            ),
          ],
        ),
        if (kDebugMode)
          Text(
            '$error',
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
            style: TextStyle(
              color: colors.iconMuted,
              fontSize: 11,
            ),
          ),
      ],
    ),
  );
}
