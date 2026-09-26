import '../../../l10n/app_localizations.dart';

/// User agents are protocol metadata, not names. Surface only recognizable
/// platform/browser labels; never echo arbitrary long client strings.
String sessionDeviceLabel(
  String userAgent,
  AppLocalizations l, {
  required bool isCurrent,
}) {
  final ua = userAgent.toLowerCase();
  final platform = ua.contains('iphone')
      ? 'iPhone'
      : ua.contains('ipad')
      ? 'iPad'
      : ua.contains('android')
      ? 'Android'
      : ua.contains('windows')
      ? 'Windows'
      : ua.contains('macintosh') || ua.contains('mac os')
      ? 'Mac'
      : ua.contains('linux')
      ? 'Linux'
      : null;
  final browser = ua.contains('edg/')
      ? 'Edge'
      : ua.contains('firefox/') || ua.contains('fxios/')
      ? 'Firefox'
      : ua.contains('chrome/') || ua.contains('crios/')
      ? 'Chrome'
      : ua.contains('safari/')
      ? 'Safari'
      : null;
  final parts = [?platform, ?browser];
  return parts.isEmpty
      ? (isCurrent ? l.settingsDeviceCurrent : l.settingsDeviceUnknown)
      : parts.join(' · ');
}
