import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:html/dom.dart' as html_dom;
import 'package:html/parser.dart' show parseFragment;

import '../l10n/app_localizations.dart';
import 'ascii_hostname.dart';
import 'registrable_domain.dart';

enum LinkRisk { normalExternal, suspicious, blocked }

/// 「标签与目标不符」的链接目标集合（Web linkRisk 的 Flutter 对等判定，
/// issue #1200 review）：链接可见文本本身是绝对 http(s) URL，且其 registrable
/// domain 与目标不同 → suspicious，始终弹确认层、不被白名单或会话信任豁免。
/// 富文本渲染器不把标签交给点击回调，因此由 Markdown 源文本预先算出目标集合，
/// 点击时按 URL 查表（与 Web 用渲染后 anchor 文本判定等价）。
Set<String> mislabeledLinkTargets(String markdown) {
  final Set<String> targets = <String>{};
  for (final UrlLabeledLink link in urlLabeledMarkdownLinks(markdown)) {
    if (_isMislabeledLink(link)) targets.add(link.href);
  }
  return targets;
}

/// 渲染后 HTML 片段的等价判定（Wiki、课评正文等 `GfHtmlContent` 链路，
/// issue #1200 review）：服务端 HTML 也不把标签交给点击回调，同样需要在
/// 点击前从源文本算出目标集合。
///
/// [base] 传与 `HtmlWidget.baseUrl` 相同的基准。集合同时收 href 原样与按
/// [base] 解析后的绝对形式：绝对 href 由 fwfh 原样交给回调，而 Wiki 侧还会经
/// `Uri.toString()` 归一化（主机小写、路径折叠、百分号解码），两种形式都要能
/// 命中；无 scheme 的 href（如协议相对 `//host/x`）两端的值都由 [base] 解析。
Set<String> mislabeledLinkTargetsInHtml(String html, {required Uri base}) {
  final Set<String> targets = <String>{};
  for (final UrlLabeledLink link in _urlLabeledHtmlLinks(html)) {
    if (_isMislabeledLink(link)) {
      targets.add(link.href);
      targets.add(base.resolveUri(Uri.parse(link.href)).toString());
    }
  }
  return targets;
}

/// 提取渲染后 HTML 里「可见文本本身是绝对 http(s) URL」的 `<a href>`。
List<UrlLabeledLink> _urlLabeledHtmlLinks(String html) {
  // 链接必须来自 `<a href>`；其余正文不必再解析一遍。
  if (!html.contains('<')) return const <UrlLabeledLink>[];
  final List<UrlLabeledLink> links = <UrlLabeledLink>[];
  for (final html_dom.Element anchor in parseFragment(
    html,
  ).querySelectorAll('a[href]')) {
    final String? href = anchor.attributes['href'];
    final String label = anchor.text.trim();
    if (href != null &&
        href.isNotEmpty &&
        label.isNotEmpty &&
        isAbsoluteHttpUrl(label)) {
      links.add(UrlLabeledLink(href: href, label: label));
    }
  }
  return links;
}

/// 链接的可见文本与目标跨 registrable domain（Web linkRisk 的核心判定，
/// Markdown 与 HTML 链路共用同一份 PSL 逻辑）。
bool _isMislabeledLink(UrlLabeledLink link) {
  final Uri? labelUri = Uri.tryParse(link.label);
  final Uri? targetUri = Uri.tryParse(link.href);
  if (labelUri == null || targetUri == null || targetUri.host.isEmpty) {
    return false;
  }
  return registrableDomain(asciiHostname(labelUri.host)) !=
      registrableDomain(asciiHostname(targetUri.host));
}

/// 归一化服务端下发的免确认白名单（issue #1200）：按 [asciiHostname] 统一为
/// 小写 ASCII（punycode）形式，丢弃空条目与单标签条目（如 localhost）——后者
/// 无法表达公开站点且会误伤。
Set<String> normalizeTrustedDomains(Iterable<String> domains) {
  final Set<String> normalized = <String>{};
  for (final String raw in domains) {
    final String value = asciiHostname(raw);
    if (value.isNotEmpty && value.contains('.')) normalized.add(value);
  }
  return normalized;
}

abstract final class LinkNavigation {
  static final Set<String> _trustedDomains = <String>{};
  // _policyDomains 管理员配置的免确认白名单（`GET /api/external-link-policy`）：
  // 条目覆盖其全部子域，与服务端 domainpolicy 的判定一致；Web 端由 SSR payload
  // 注入同一份清单。命中只豁免 normalExternal，suspicious/blocked 永不豁免。
  static Set<String> _policyDomains = <String>{};

  static void clearSessionTrust() => _trustedDomains.clear();

  static bool isTrusted(String domain) =>
      _trustedDomains.contains(trustDomainFor(domain));

  static void trustForSession(String domain) =>
      _trustedDomains.add(trustDomainFor(domain));

  // 归一化放在这里：写入方传 `Uri.host`（IDN 是百分号编码形式），读取方传
  // `asciiHostname` 后的 punycode，两者必须落到同一个键（issue #1200）。
  static String trustDomainFor(String hostname) =>
      registrableDomain(asciiHostname(hostname));

  /// 应用服务端下发的白名单（由 [ExternalLinkPolicyController] 启动时调用）。
  static void applyTrustedDomains(Iterable<String> domains) {
    _policyDomains = normalizeTrustedDomains(domains);
  }

  /// 命中白名单条目本身或其子域；后缀比较始终在点边界上，
  /// 因此 "github.com.evil.test" 与 "notgithub.com" 都不会命中。
  /// 入参先经 [asciiHostname]：IDN 链接在 Dart 里是百分号编码主机名，
  /// 不转换会与清单里的 punycode 条目永不相等（Web 端浏览器已给 punycode）。
  static bool isPolicyTrusted(String hostname) {
    String candidate = asciiHostname(hostname);
    while (candidate.contains('.')) {
      if (_policyDomains.contains(candidate)) return true;
      candidate = candidate.substring(candidate.indexOf('.') + 1);
    }
    return false;
  }

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
    // 会话信任与白名单都在 ASCII（punycode）形式上比较：PSL 数据与清单条目
    // 都是 A-label，IDN 链接经 asciiHostname 转换后才能命中（issue #1200）。
    final String domain = asciiHostname(uri.host);
    final String trustDomain = trustDomainFor(domain);
    // 白名单与会话信任都只豁免 normalExternal：suspicious/blocked 永不跳过弹层。
    if (risk != LinkRisk.normalExternal ||
        !(isTrusted(trustDomain) || isPolicyTrusted(domain))) {
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
              GfText(
                l10n.linkPreviewExternalTitle,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 8),
              GfText(l10n.linkPreviewExternalBody(uri.host)),
              const SizedBox(height: 12),
              GfSelectableText(
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
                  title: GfText(l10n.linkPreviewRememberDomain),
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
