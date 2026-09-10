import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../widgets/app_refresh_indicator.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import '../../updates/update_host.dart';

enum SiteInfoKind { links, sponsors, terms, privacy }

extension SiteInfoLabel on SiteInfoKind {
  String label(AppLocalizations l10n) => switch (this) {
    SiteInfoKind.links => l10n.siteInfoLinks,
    SiteInfoKind.sponsors => l10n.siteInfoSponsors,
    SiteInfoKind.terms => l10n.siteInfoTerms,
    SiteInfoKind.privacy => l10n.siteInfoPrivacy,
  };
  IconData get icon => switch (this) {
    SiteInfoKind.links => Icons.link,
    SiteInfoKind.sponsors => Icons.favorite_border,
    SiteInfoKind.terms => Icons.description_outlined,
    SiteInfoKind.privacy => Icons.privacy_tip_outlined,
  };
}

class SiteInfoIndexPage extends StatelessWidget {
  const SiteInfoIndexPage({super.key});
  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.siteInfoTitle)),
      body: ListView(
        children: [
          if (supportsApkUpdates)
            ListTile(
              leading: const GfSymbol('download'),
              title: Text(l10n.updateCheck),
              onTap: () => appUpdateHostKey.currentState?.check(force: true),
            ),
          for (final kind in SiteInfoKind.values)
            ListTile(
              leading: Icon(kind.icon),
              title: Text(kind.label(l10n)),
              trailing: const Icon(Icons.chevron_right),
              onTap: () => context.push('/${kind.name}'),
            ),
        ],
      ),
    );
  }
}

/// Public Web information, read through the same page payload channel as the
/// forum. Policy HTML is server-sanitized and rendered without a JavaScript VM.
class SiteInfoPage extends ConsumerStatefulWidget {
  const SiteInfoPage({super.key, required this.kind});
  final SiteInfoKind kind;
  @override
  ConsumerState<SiteInfoPage> createState() => _SiteInfoPageState();
}

class _SiteInfoPageState extends ConsumerState<SiteInfoPage> {
  AsyncValue<Object> _content = const AsyncValue.loading();
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _content = const AsyncValue.loading());
    final result = await AsyncValue.guard<Object>(() async {
      final page = await ref
          .read(pageRepositoryProvider)
          .fetch('/${widget.kind.name}');
      final Object? props = switch (widget.kind) {
        SiteInfoKind.links => parsePageProps<LinksPageProps>(page),
        SiteInfoKind.sponsors => parsePageProps<SponsorsPageProps>(page),
        SiteInfoKind.terms ||
        SiteInfoKind.privacy => parsePageProps<TermsPageProps>(page),
      };
      if (props == null) {
        throw const FormatException('Invalid site information payload');
      }
      return props;
    });
    if (mounted) setState(() => _content = result);
  }

  Future<bool> _open(String link) async {
    final origin = Uri.parse(ref.read(apiClientProvider).baseUrl);
    if (link.startsWith('#')) return false;
    final parsed = Uri.tryParse(link);
    if (parsed == null) return true;
    final uri = origin.resolveUri(parsed);
    if (uri.userInfo.isNotEmpty ||
        !['https', 'http', 'mailto'].contains(uri.scheme)) {
      return true;
    }
    if (uri.hasAuthority &&
        uri.origin == origin.origin &&
        SiteInfoKind.values.any((kind) => uri.path == '/${kind.name}')) {
      context.push(uri.path);
      return true;
    }
    try {
      if (!await launchUrl(uri, mode: LaunchMode.externalApplication)) {
        throw StateError('Could not open link');
      }
    } catch (_) {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).commonLoadFailed,
          error: true,
        );
      }
    }
    return true;
  }

  Widget _link({
    required String title,
    required String description,
    required String url,
    String image = '',
  }) => ListTile(
    contentPadding: const EdgeInsets.symmetric(vertical: 8),
    leading: image.isEmpty ? null : GfAvatar(src: resolveApiAssetUrl(image)),
    title: Text(title),
    subtitle: description.isEmpty ? null : Text(description),
    trailing: url.isEmpty ? null : const Icon(Icons.open_in_new, size: 18),
    onTap: url.isEmpty ? null : () => _open(url),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final type = GfTheme.typographyOf(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(widget.kind.label(l10n))),
      body: _content.when(
        loading: () => const GfLoading(),
        error: (_, _) =>
            GfErrorRetry(message: l10n.commonLoadFailed, onRetry: _load),
        data: (content) => AppRefreshIndicator(
          onRefresh: _load,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(20, 20, 20, 40),
            children: switch (content) {
              LinksPageProps props => [
                if (props.totalCount == 0) Text(l10n.siteInfoEmpty),
                for (final group in props.groups) ...[
                  Text(
                    '${group.emoji} ${group.name}'.trim(),
                    style: type.heading,
                  ),
                  for (final link in group.links)
                    _link(
                      title: link.name,
                      description: link.desc,
                      url: link.url,
                      image: link.logoUrl,
                    ),
                  const SizedBox(height: 24),
                ],
              ],
              SponsorsPageProps props => [
                Text(props.content.title, style: type.heading),
                const SizedBox(height: 12),
                Text(props.content.description),
                const SizedBox(height: 24),
                for (final section in props.sections) ...[
                  Text(section.label, style: type.heading),
                  for (final sponsor in section.sponsors)
                    _link(
                      title: sponsor.name,
                      description: sponsor.message,
                      url: sponsor.link,
                      image: sponsor.avatarUrl,
                    ),
                  const SizedBox(height: 24),
                ],
                for (final rule in props.rules)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(rule.content),
                  ),
                _link(
                  title: props.contact.title,
                  description: props.contact.description,
                  url: '',
                ),
                if (props.contact.buttonLink.isNotEmpty)
                  FilledButton(
                    onPressed: () => _open(props.contact.buttonLink),
                    child: Text(props.contact.buttonText),
                  ),
              ],
              TermsPageProps props => [
                if (!props.enabled || props.contentHtml.isEmpty)
                  Text(l10n.siteInfoEmpty)
                else
                  HtmlWidget(
                    props.contentHtml,
                    baseUrl: Uri.parse(ref.read(apiClientProvider).baseUrl),
                    onTapUrl: _open,
                    textStyle: type.body,
                  ),
              ],
              _ => [Text(l10n.commonLoadFailed)],
            },
          ),
        ),
      ),
    );
  }
}
