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
import '../../updates/release_notes_page.dart';
import '../../navigation/reading_window.dart';

enum SiteInfoKind { links, sponsors, terms, privacy }

extension SiteInfoLabel on SiteInfoKind {
  String label(AppLocalizations l10n) => switch (this) {
    SiteInfoKind.links => l10n.siteInfoLinks,
    SiteInfoKind.sponsors => l10n.siteInfoSponsors,
    SiteInfoKind.terms => l10n.siteInfoTerms,
    SiteInfoKind.privacy => l10n.siteInfoPrivacy,
  };
  String get symbol => switch (this) {
    SiteInfoKind.links => 'link',
    SiteInfoKind.sponsors => 'heart',
    SiteInfoKind.terms => 'file-text',
    SiteInfoKind.privacy => 'shield-check',
  };
}

class SiteInfoIndexPage extends StatelessWidget {
  const SiteInfoIndexPage({super.key});

  @override
  Widget build(BuildContext context) => const _SiteInfoWorkspace();
}

class SiteInfoPage extends StatelessWidget {
  const SiteInfoPage({super.key, required this.kind});
  final SiteInfoKind kind;

  @override
  Widget build(BuildContext context) =>
      _SiteInfoWorkspace(key: ValueKey(kind), initialKind: kind);
}

class _SiteInfoWorkspace extends StatefulWidget {
  const _SiteInfoWorkspace({super.key, this.initialKind});
  final SiteInfoKind? initialKind;

  @override
  State<_SiteInfoWorkspace> createState() => _SiteInfoWorkspaceState();
}

class _SiteInfoWorkspaceState extends State<_SiteInfoWorkspace> {
  late SiteInfoKind? _kind = widget.initialKind;
  late bool _hasSelection = widget.initialKind != null;

  void _select(SiteInfoKind? kind) => setState(() {
    _kind = kind;
    _hasSelection = true;
  });

  Widget _directory(bool wide) {
    final l10n = AppLocalizations.of(context);
    Widget entry({
      Key? key,
      required String symbol,
      required String label,
      required VoidCallback onTap,
      bool selected = false,
    }) => Semantics(
      selected: wide && selected,
      child: Material(
        color: wide && selected
            ? GfTheme.colorsOf(context).primary.withValues(alpha: .08)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(12),
        clipBehavior: Clip.antiAlias,
        child: GfSettingRow(
          key: key,
          symbol: symbol,
          title: label,
          trailing: wide ? const SizedBox.shrink() : null,
          onTap: onTap,
        ),
      ),
    );
    return ListView(
      key: const ValueKey('site-info-directory'),
      padding: const EdgeInsets.fromLTRB(12, 20, 12, 24),
      children: [
        if (supportsMobileReleaseNotes) ...[
          entry(
            symbol: 'download',
            label: l10n.updateCheck,
            onTap: () => appUpdateHostKey.currentState?.check(force: true),
          ),
          entry(
            symbol: 'history',
            label: l10n.releaseNotesHistory,
            onTap: () => Navigator.of(context).push<void>(
              MaterialPageRoute(
                builder: (_) =>
                    const ReadingPageWindow(child: ReleaseNotesPage()),
              ),
            ),
          ),
        ],
        for (final kind in SiteInfoKind.values)
          entry(
            key: ValueKey('site-info-kind-${kind.name}'),
            symbol: kind.symbol,
            label: kind.label(l10n),
            selected: _kind == kind,
            onTap: () => wide ? _select(kind) : context.push('/${kind.name}'),
          ),
        entry(
          key: const ValueKey('visitor-analytics-disclosure'),
          symbol: 'monitor',
          label: l10n.siteInfoAnalytics,
          selected: _kind == null,
          onTap: () => wide
              ? _select(null)
              : Navigator.of(context).push<void>(
                  MaterialPageRoute(
                    builder: (_) => const ReadingPageWindow(
                      child: _VisitorAnalyticsDisclosurePage(),
                    ),
                  ),
                ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final scale = (MediaQuery.textScalerOf(context).scale(16) / 16).clamp(
      1.0,
      double.infinity,
    );
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth >= 900 * scale;
        final showIndex = !wide && !_hasSelection;
        final title = showIndex
            ? l10n.siteInfoTitle
            : _kind?.label(l10n) ?? l10n.siteInfoAnalytics;
        return Scaffold(
          appBar: GfAppBar(
            title: Text(title),
            leading: !wide && _hasSelection && widget.initialKind == null
                ? GfIconButton(
                    symbol: 'arrow-left',
                    tooltip: l10n.commonBack,
                    onPressed: () => setState(() => _hasSelection = false),
                  )
                : null,
          ),
          body: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (wide) ...[
                SizedBox(width: 264 * scale, child: _directory(true)),
                VerticalDivider(
                  width: 1,
                  thickness: 1,
                  color: GfTheme.colorsOf(context).line,
                ),
              ],
              Expanded(
                key: const ValueKey('site-info-detail-slot'),
                child: showIndex
                    ? _directory(false)
                    : Align(
                        alignment: Alignment.topCenter,
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 720),
                          child: SizedBox.expand(
                            key: const ValueKey('site-info-content'),
                            child: _kind == null
                                ? const _VisitorAnalyticsDisclosureContent()
                                : _SiteInfoContent(
                                    key: ValueKey(_kind),
                                    kind: _kind!,
                                    onOpenKind: wide ? _select : null,
                                  ),
                          ),
                        ),
                      ),
              ),
            ],
          ),
        );
      },
    );
  }
}

/// Bundled disclosure stays readable when site policies are disabled or offline.
class _VisitorAnalyticsDisclosurePage extends StatelessWidget {
  const _VisitorAnalyticsDisclosurePage();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.siteInfoAnalytics)),
      body: const _VisitorAnalyticsDisclosureContent(),
    );
  }
}

class _VisitorAnalyticsDisclosureContent extends StatelessWidget {
  const _VisitorAnalyticsDisclosureContent();

  @override
  Widget build(BuildContext context) => SafeArea(
    child: SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 20, 20, 40),
      child: Text(
        AppLocalizations.of(context).siteInfoAnalyticsDescription,
        style: GfTheme.typographyOf(context).body,
      ),
    ),
  );
}

/// Public Web information uses the existing sanitized page payload channel.
class _SiteInfoContent extends ConsumerStatefulWidget {
  const _SiteInfoContent({super.key, required this.kind, this.onOpenKind});
  final SiteInfoKind kind;
  final ValueChanged<SiteInfoKind>? onOpenKind;

  @override
  ConsumerState<_SiteInfoContent> createState() => _SiteInfoContentState();
}

class _SiteInfoContentState extends ConsumerState<_SiteInfoContent> {
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
      final kind = SiteInfoKind.values.firstWhere(
        (kind) => uri.path == '/${kind.name}',
      );
      if (widget.onOpenKind case final onOpenKind?) {
        onOpenKind(kind);
      } else {
        context.push(uri.path);
      }
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
    trailing: url.isEmpty ? null : const GfSymbol('external-link', size: 18),
    onTap: url.isEmpty ? null : () => _open(url),
  );

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final type = GfTheme.typographyOf(context);
    final body = _content.when(
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
    );
    return body;
  }
}
