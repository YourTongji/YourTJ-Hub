import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../l10n/app_localizations.dart';
import 'registrable_domain.dart';

enum LinkRisk { normalExternal, suspicious, blocked }

abstract final class LinkNavigation {
  static final Set<String> _trustedDomains = <String>{};

  static void clearSessionTrust() => _trustedDomains.clear();

  static bool isTrusted(String domain) =>
      _trustedDomains.contains(trustDomainFor(domain));

  static void trustForSession(String domain) =>
      _trustedDomains.add(trustDomainFor(domain));

  static String trustDomainFor(String hostname) => registrableDomain(hostname);

  static Future<void> open(
    BuildContext context,
    String rawUrl, {
    required String baseUrl,
    String? previewKind,
    LinkRisk risk = LinkRisk.normalExternal,
  }) async {
    final Uri? uri = Uri.tryParse(rawUrl.trim());
    if (uri == null || rawUrl.contains('\\')) return;

    final bool relativeInternal =
        !uri.hasScheme && !uri.hasAuthority && rawUrl.startsWith('/');
    final Uri? base = Uri.tryParse(baseUrl);
    final bool sameOrigin =
        uri.hasScheme &&
        base != null &&
        uri.scheme.toLowerCase() == base.scheme.toLowerCase() &&
        uri.host.toLowerCase() == base.host.toLowerCase() &&
        _effectivePort(uri) == _effectivePort(base);
    if (previewKind == 'internal' || relativeInternal || sameOrigin) {
      final String? route = _appRoute(uri);
      if (route != null && context.mounted) context.push(route);
      return;
    }

    if (!uri.hasScheme ||
        (uri.scheme != 'http' && uri.scheme != 'https') ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty) {
      return;
    }
    final String domain = uri.host.toLowerCase();
    final String trustDomain = trustDomainFor(domain);
    if (risk != LinkRisk.normalExternal || !isTrusted(trustDomain)) {
      final bool accepted = await _confirmExternal(context, uri, risk);
      if (!accepted || !context.mounted) return;
    }
    if (risk == LinkRisk.blocked) return;
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  static Future<bool> _confirmExternal(
    BuildContext context,
    Uri uri,
    LinkRisk risk,
  ) async {
    bool remember = false;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool? accepted = await showGfModal<bool>(
      context,
      barrierDismissible: false,
      builder: (BuildContext dialogContext) => StatefulBuilder(
        builder: (BuildContext context, StateSetter setState) => ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 448),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              Text(
                l10n.linkPreviewExternalTitle,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 8),
              Text(l10n.linkPreviewExternalBody(uri.host)),
              const SizedBox(height: 12),
              SelectableText(
                uri.toString(),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: GfTheme.colorsOf(
                    context,
                  ).baseContent.withValues(alpha: 0.7),
                ),
              ),
              const SizedBox(height: 12),
              if (risk == LinkRisk.normalExternal)
                CheckboxListTile(
                  value: remember,
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  title: Text(l10n.linkPreviewRememberDomain),
                  onChanged: (bool? value) =>
                      setState(() => remember = value ?? false),
                ),
              const SizedBox(height: 12),
              Wrap(
                alignment: WrapAlignment.end,
                spacing: 8,
                runSpacing: 8,
                children: <Widget>[
                  GfButton(
                    label: l10n.commonCancel,
                    variant: GfButtonVariant.secondary,
                    onPressed: () => Navigator.pop(dialogContext, false),
                  ),
                  if (risk != LinkRisk.blocked)
                    GfButton(
                      label: l10n.linkPreviewContinue,
                      onPressed: () => Navigator.pop(dialogContext, true),
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
    if (accepted == true && remember && risk == LinkRisk.normalExternal) {
      trustForSession(uri.host);
    }
    return accepted == true;
  }

  static String? _appRoute(Uri uri) {
    final List<String> segments = uri.pathSegments;
    if (segments.length >= 3 && segments[0] == 'p' && segments[1] == 'post') {
      final String postNo = segments.length >= 4
          ? '?postNo=${Uri.encodeQueryComponent(segments[3])}'
          : '';
      return '/p/${segments[2]}$postNo';
    }
    if (segments.length >= 2 && segments[0] == 'topics') {
      return '/p/${segments[1]}';
    }
    if (segments.length >= 2 &&
        (segments[0] == 'u' || segments[0] == 'courses')) {
      return '/${segments[0]}/${segments[1]}';
    }
    if (segments.isNotEmpty && segments[0] == 'wiki') {
      return uri.replace(scheme: '', host: '').toString();
    }
    return null;
  }

  static int _effectivePort(Uri uri) {
    if (uri.hasPort) return uri.port;
    return uri.scheme == 'https' ? 443 : 80;
  }
}
