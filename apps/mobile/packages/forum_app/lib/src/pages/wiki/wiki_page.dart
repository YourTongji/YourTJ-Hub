import 'package:flutter/material.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:core/core.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../format.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../widgets/status_views.dart';

/// 对 wiki 路径逐段做 URL 编码(与 web wiki-path.ts wikiHref 一致):
/// 路由参数按段解码、页面请求按段编码,中文/空格等字符不破坏路由匹配。
String encodeWikiPath(String path) =>
    path.split('/').map(Uri.encodeComponent).join('/');

/// 容错解码 wiki 锚点:`Uri.fragment` 保留 percent-encoded 态(裸 Unicode
/// 也会被 SDK 归一化为编码态),scrollToAnchor 需要解码后的真实标题 id;
/// `Uri.decodeComponent` 对非法转义(如 `%zz`)抛 ArgumentError,此处
/// 解码失败时原样返回,交由滚动找不到 id 时静默降级。
String decodeWikiAnchor(String anchor) {
  if (!anchor.contains('%')) return anchor;
  try {
    return Uri.decodeComponent(anchor);
  } on ArgumentError {
    return anchor;
  }
}

/// wiki 页面详情(web WikiPage.vue 的移动端形态)。
///
/// 正文经页面级数据通道返回的是**服务端渲染的 HTML**(goldmark →
/// `wiki_pages.rendered_html`,web 端 `v-html` 直接注入),因此用
/// [HtmlWidget] 渲染而非 markdown_widget;标题 id 由 goldmark headingid
/// 生成且与 `toc` 严格一致,目录跳转使用 HtmlWidget 的锚点滚动。
class WikiPage extends ConsumerStatefulWidget {
  const WikiPage({super.key, required this.wikiPath, this.initialAnchor = ''});

  /// 已解码的 wiki 路径(如 `guide/getting-started`),不含前导 `/wiki/`。
  final String wikiPath;
  final String initialAnchor;

  @override
  ConsumerState<WikiPage> createState() => _WikiPageState();
}

class _WikiPageState extends ConsumerState<WikiPage> {
  AsyncValue<WikiPageDetail> _detail = const AsyncValue.loading();
  final ScrollController _scrollController = ScrollController();
  final GlobalKey<HtmlWidgetState> _htmlKey = GlobalKey<HtmlWidgetState>();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final int epoch = ref.read(offlineCacheEpochProvider);
    setState(() => _detail = const AsyncValue.loading());
    try {
      final PagePayload payload = await ref
          .read(pageRepositoryProvider)
          .fetch('/wiki/${encodeWikiPath(widget.wikiPath)}');
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      final Object? pageJson = payload.props['page'];
      if (pageJson is! Map) {
        setState(
          () => _detail = AsyncValue.error(
            AppLocalizations.of(context).commonParseFailed,
            StackTrace.current,
          ),
        );
        return;
      }
      setState(
        () => _detail = AsyncValue.data(
          WikiPageDetail.fromJson(Map<String, dynamic>.from(pageJson)),
        ),
      );
      if (widget.initialAnchor.isNotEmpty) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) {
            _htmlKey.currentState?.scrollToAnchor(widget.initialAnchor);
          }
        });
      }
    } catch (e, st) {
      if (!mounted || epoch != ref.read(offlineCacheEpochProvider)) return;
      setState(() => _detail = AsyncValue.error(e, st));
    }
  }

  Future<void> _openToc(List<WikiTocItem> items) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final String? selected = await showGfBottomSheet<String>(
      context,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: <Widget>[
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: Text(
                l10n.wikiToc,
                style: GfTheme.typographyOf(
                  sheetContext,
                ).heading.copyWith(fontWeight: FontWeight.w700),
              ),
            ),
            for (final WikiTocItem item in items)
              InkWell(
                onTap: () => Navigator.pop(sheetContext, item.id),
                child: Padding(
                  padding: EdgeInsets.fromLTRB(
                    12 + ((item.level - 1) * 8).toDouble(),
                    10,
                    16,
                    10,
                  ),
                  child: Text(
                    item.text,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: GfTheme.typographyOf(sheetContext).small.copyWith(
                      color: GfTheme.colorsOf(
                        sheetContext,
                      ).baseContent.withValues(alpha: 0.7),
                      fontWeight: item.level <= 1
                          ? FontWeight.w600
                          : FontWeight.w400,
                    ),
                  ),
                ),
              ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
    if (!mounted || selected == null || selected.isEmpty) return;
    // 目录条目 id 与正文标题锚点一致(goldmark headingid)。
    _htmlKey.currentState?.scrollToAnchor(selected);
  }

  Future<void> _openEdit(String url) async {
    final Uri? uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      // 无可用浏览器/应用时静默;用户仍可在 GitHub 编辑入口外自行访问。
    }
  }

  /// 正文链接策略:站内 wiki 链接推入详情页;`#锚点` 交还 [HtmlWidget]
  /// 内部滚动;其余绝对链接交给系统浏览器。
  ///
  /// 服务端渲染的 href 处于 percent-encoded 态(issue #560):统一起
  /// [Uri.pathSegments](逐段解码)提取目标路径、[Uri.fragment](保留编码
  /// 态,须经 [decodeWikiAnchor] 解码)提取锚点,再经 [encodeWikiPath]
  /// 单出口编码推送。不得用 `Uri.path`——它保留编码态,再编码会把 `%`
  /// 变成 `%25`(二次编码)。
  Future<bool> _handleLinkTap(String url) async {
    final Uri? uri = Uri.tryParse(url);
    if (uri == null) return false;
    final bool isHttp =
        uri.hasScheme && (uri.scheme == 'http' || uri.scheme == 'https');
    // 站内判定:相对链接须 /wiki/ 前缀;绝对链接保留既有 path 前缀语义
    //(host 校验超出本修复范围)。`/wiki` 裸路径、`/wiki/` 空 target
    // 维持现状:不跳转。
    final List<String> segments = uri.pathSegments;
    final bool isInternalWiki =
        segments.isNotEmpty &&
        segments.first == 'wiki' &&
        segments.length > 1 &&
        (isHttp || url.startsWith('/wiki/'));
    if (isInternalWiki) {
      // decoded 域目标;锚点不卷入路径(%23)。`Uri.fragment` 保留编码态
      //(review P2),先解码再重新编码,避免二次编码 %25E7...。
      final String target = segments.sublist(1).join('/');
      if (context.mounted && target.isNotEmpty) {
        final String? fragment = uri.hasFragment ? uri.fragment : null;
        final String anchor = (fragment != null && fragment.isNotEmpty)
            ? '#${Uri.encodeComponent(decodeWikiAnchor(fragment))}'
            : '';
        context.push('/wiki/${encodeWikiPath(target)}$anchor');
      }
      return true;
    }
    if (isHttp) {
      try {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } catch (_) {
        // 外部链接打开失败不阻塞阅读。
      }
      return true;
    }
    // `#id` 页内锚点由 HtmlWidget 的 AnchorRegistry 处理。
    return false;
  }

  void _openImageViewer(String url) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          backgroundColor: const Color(0xFF000000),
          body: SafeArea(child: GfImageViewer(images: <String>[url])),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    // valueOrNull:AsyncValue.error 时直接读 .value 会重抛错误。
    final WikiPageDetail? page = _detail.valueOrNull;
    return Scaffold(
      appBar: GfAppBar(
        title: Text(
          page?.title ?? l10n.wikiTitle,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
      ),
      bottomNavigationBar: page == null
          ? null
          : SafeArea(
              top: false,
              child: Container(
                decoration: BoxDecoration(
                  border: Border(
                    top: BorderSide(color: GfTheme.colorsOf(context).line),
                  ),
                ),
                child: Row(
                  children: [
                    if (page.toc.isNotEmpty)
                      Expanded(
                        child: TextButton.icon(
                          icon: const GfSymbol('list', size: 18),
                          label: Text(l10n.wikiToc),
                          onPressed: () => _openToc(page.toc),
                        ),
                      ),
                    Expanded(
                      child: TextButton.icon(
                        icon: const GfSymbol('search', size: 18),
                        label: Text(l10n.commonSearch),
                        onPressed: () => context.push('/wiki/search'),
                      ),
                    ),
                    if (page.canEdit && page.editUrl.isNotEmpty)
                      Expanded(
                        child: TextButton.icon(
                          icon: const GfSymbol('external-link', size: 18),
                          label: Text(l10n.wikiEditOnGithub),
                          onPressed: () => _openEdit(page.editUrl),
                        ),
                      ),
                  ],
                ),
              ),
            ),
      body: _detail.when(
        loading: () => const _WikiPageSkeleton(),
        error: (Object e, StackTrace _) =>
            GfErrorRetry(message: resolveErrorMessage(l10n, e), onRetry: _load),
        data: (WikiPageDetail loaded) => _WikiProse(
          page: loaded,
          scrollController: _scrollController,
          htmlKey: _htmlKey,
          onLinkTap: _handleLinkTap,
          onImageTap: _openImageViewer,
        ),
      ),
    );
  }
}

/// 正文阅读面(web `.gf-card` + `.gf-prose-post` 的移动端形态):
/// 服务端渲染 HTML 配 Gf 排版,底部 meta 脚注。
class _WikiProse extends StatelessWidget {
  const _WikiProse({
    required this.page,
    required this.scrollController,
    required this.htmlKey,
    required this.onLinkTap,
    required this.onImageTap,
  });

  final WikiPageDetail page;
  final ScrollController scrollController;
  final GlobalKey<HtmlWidgetState> htmlKey;
  final Future<bool> Function(String url) onLinkTap;
  final void Function(String url) onImageTap;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography typography = GfTheme.typographyOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final String baseContent = _hex(colors.baseContent);
    final String line = _hex(colors.line);
    final String base200 = _hex(colors.base200);
    final Color faint = colors.baseContent.withValues(alpha: 0.45);

    final Widget body;
    if (page.content.trim().isEmpty) {
      body = Padding(
        padding: const EdgeInsets.symmetric(vertical: 32),
        child: Center(
          child: Text(
            l10n.commonEmpty,
            style: typography.small.copyWith(
              color: colors.baseContent.withValues(alpha: 0.55),
            ),
          ),
        ),
      );
    } else {
      body = HtmlWidget(
        page.content,
        key: htmlKey,
        buildAsync: false,
        textStyle: typography.body.copyWith(color: colors.baseContent),
        customStylesBuilder: (element) {
          switch (element.localName) {
            case 'pre':
              return <String, String>{
                'background-color': base200,
                'color': baseContent,
                'padding': '12px',
                'border-radius': '${radii.box}px',
                'border': '1px solid $line',
                'margin': '10px 0',
                'font-size': '13px',
                'line-height': '1.55',
              };
            case 'blockquote':
              return <String, String>{
                'border-left': '3px solid $line',
                'background-color': _hex(colors.base200.withValues(alpha: 0.7)),
                'color': _hex(colors.baseContent.withValues(alpha: 0.75)),
                'padding': '6px 12px',
                'margin': '10px 0',
              };
            default:
              return null;
          }
        },
        customWidgetBuilder: (element) {
          if (element.localName != 'img') return null;
          final String? src = element.attributes['src'];
          // data: URI 交给 fwfh 内置 data-image 渲染,不当作网络图处理。
          if (src == null || src.isEmpty || src.startsWith('data:')) {
            return null;
          }
          final String resolved = resolveApiAssetUrl(src);
          final String? alt = element.attributes['alt'];
          return Container(
            margin: const EdgeInsets.symmetric(vertical: 8),
            alignment: Alignment.center,
            child: GestureDetector(
              onTap: () => onImageTap(resolved),
              child: Container(
                decoration: BoxDecoration(
                  border: Border.all(color: colors.line),
                  borderRadius: BorderRadius.circular(radii.box),
                ),
                clipBehavior: Clip.antiAlias,
                child: Image.network(
                  resolved,
                  fit: BoxFit.contain,
                  semanticLabel: alt,
                  errorBuilder: (_, _, _) => Padding(
                    padding: const EdgeInsets.all(24),
                    child: Icon(Icons.broken_image, color: faint),
                  ),
                ),
              ),
            ),
          );
        },
        onTapUrl: onLinkTap,
      );
    }

    return ListView(
      key: const Key('wiki-page-scroll'),
      controller: scrollController,
      padding: EdgeInsets.zero,
      children: <Widget>[
        GfPanel(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                body,
                const SizedBox(height: 14),
                GfDivider(),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 18,
                  runSpacing: 8,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: <Widget>[
                    _MetaItem(
                      icon: Icons.schedule,
                      text: formatDateTime(page.updatedAt),
                      faint: faint,
                    ),
                    _MetaItem(
                      icon: Icons.visibility_outlined,
                      text: l10n.wikiViewCount(page.viewCount),
                      faint: faint,
                    ),
                    _MetaItem(
                      icon: Icons.favorite_border,
                      text: formatNumber(page.likeCount),
                      faint: faint,
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  static String _hex(Color color) =>
      '#${color.toARGB32().toRadixString(16).padLeft(8, '0')}';
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({
    required this.icon,
    required this.text,
    required this.faint,
  });

  final IconData icon;
  final String text;
  final Color faint;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Icon(icon, size: 13, color: faint),
        const SizedBox(width: 4),
        Text(
          text,
          style: GfTheme.typographyOf(context).caption.copyWith(color: faint),
        ),
      ],
    );
  }
}

/// 详情加载骨架:标题行 + 正文段落块 + 脚注分隔线。
class _WikiPageSkeleton extends StatelessWidget {
  const _WikiPageSkeleton();

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView(
        physics: const NeverScrollableScrollPhysics(),
        children: const <Widget>[
          Padding(
            padding: EdgeInsets.fromLTRB(16, 18, 16, 14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                GfSkeleton(width: 200, height: 22, radius: 6),
                SizedBox(height: 16),
                GfSkeleton(height: 14, radius: 5),
                SizedBox(height: 8),
                GfSkeleton(height: 14, radius: 5),
                SizedBox(height: 8),
                GfSkeleton(width: 240, height: 14, radius: 5),
                SizedBox(height: 18),
                GfSkeleton(width: 140, height: 18, radius: 5),
                SizedBox(height: 12),
                GfSkeleton(height: 14, radius: 5),
                SizedBox(height: 8),
                GfSkeleton(height: 14, radius: 5),
                SizedBox(height: 8),
                GfSkeleton(width: 220, height: 14, radius: 5),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
