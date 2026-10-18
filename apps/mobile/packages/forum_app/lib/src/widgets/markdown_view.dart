import 'dart:async';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:html/parser.dart' show parseFragment;
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../reading_preferences.dart';
import '../images/image_save.dart';
import '../link_navigation.dart';
import '../providers.dart';
import 'rich_content/gf_code_block.dart';
import 'rich_content/gf_table.dart';
import 'stickers/sticker_image.dart';
import 'stickers/resolved_sticker_content.dart';
import 'stickers/sticker_strings.dart';
import 'link_preview_card.dart';

// 既有引用方（页面与测试）仍从这里拿卡片实现，避免为一次拆分改所有 import。
export 'link_preview_card.dart' show GfLinkPreviewCard, LinkPreviewPlatformLabels;
import 'share/share_image_readiness.dart';

/// Body text comes from [GfRichContentTypography] so post bodies, publish
/// previews and the desktop-style fallbacks cannot drift apart. [userScale] is
/// the reader preference and stays at its 100% default for chrome that is not
/// rich content; rich content renderers pass the persisted value instead.
TextStyle readingBodyStyle(BuildContext context, [double userScale = 1]) =>
    GfRichContentTypography.of(context, userScale: userScale).body;

/// 帖子 markdown 渲染视图。
///
/// 基于 markdown_widget,渲染样式对齐 web 端 prose.css 语义:
/// - 引用:左 4px line 线 + base-200 底 + 弱化文字
/// - 代码块:base-200 底 + line 边框 + 圆角
/// - 行内代码:error 色 + base-200 底
/// - 表格:line 边框
/// - 图片:object-contain + max-height min(360, 70vh) + 圆角边框
/// 图片点击打开 [GfImageViewer] 全屏查看(web MarkdownImageViewer.vue 语义)。
///
/// 帖子 content 是 raw markdown(表情包 token 未展开,服务端只展开
/// renderedContent HTML 链路):渲染前把 `[:sticker:name:]` 重写为标准图片
/// 语法保留表情语义，独立预览并排除普通图片画廊;库未就绪时先用原文渲染,拉取完成
/// 后异步刷新,未知/停用 token 保持原文。
class GfMarkdownView extends ConsumerStatefulWidget {
  const GfMarkdownView({
    super.key,
    required this.data,
    this.images,
    this.mentions = const <PostMention>[],
    this.selectable = false,
    this.stickerSize = StickerImage.readingSize,
    this.screenshot = false,
    this.colors,
    this.compact = false,
  });

  final String data;
  final List<PostMention> mentions;

  /// 已知图片列表(取自 markdown 的图片引用);为 null 时从内容提取。
  final List<String>? images;

  final bool selectable;

  /// Draft previews opt into thumbnails; posts and replies use reading size.
  final double stickerSize;

  /// Uses stable raw Markdown links rather than deferred network link cards.
  final bool screenshot;

  /// Palette override for share cards; ordinary reading uses the app theme.
  final GfColors? colors;
  final bool compact;

  @override
  ConsumerState<GfMarkdownView> createState() => _GfMarkdownViewState();
}

class _GfMarkdownViewState extends ConsumerState<GfMarkdownView> {
  late Widget _markdownBody;
  Future<List<LinkPreviewPayload>>? _linkPreviews;
  String _linkPreviewSelectionSource = '';
  int _linkPreviewGeneration = 0;

  Map<String, String> _stickerUrls = const {};

  /// 标签与目标不符的链接目标（issue #1200 review）：按源文本记忆，避免每次
  /// 重建都重新解析 Markdown AST。
  String _mislabeledSource = '';
  Set<String> _mislabeledTargets = const <String>{};

  /// Reader preference the cached body was built with; the persisted setting
  /// must reach every renderer branch, including this cached one.
  double _contentScale = 1;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _contentScale = ref.read(contentFontScaleProvider);
    _markdownBody = _buildMarkdownBody(_contentScale);
  }

  @override
  void didUpdateWidget(covariant GfMarkdownView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.data != widget.data ||
        oldWidget.selectable != widget.selectable ||
        oldWidget.stickerSize != widget.stickerSize ||
        oldWidget.screenshot != widget.screenshot ||
        oldWidget.colors != widget.colors ||
        oldWidget.compact != widget.compact ||
        !listEquals(oldWidget.images, widget.images) ||
        !listEquals(oldWidget.mentions, widget.mentions)) {
      if (oldWidget.data != widget.data) {
        _linkPreviews = null;
        _linkPreviewSelectionSource = '';
        _linkPreviewGeneration++;
        _mislabeledSource = '';
        _mislabeledTargets = const <String>{};
      }
      _contentScale = ref.read(contentFontScaleProvider);
      _markdownBody = _buildMarkdownBody(_contentScale);
    }
  }

  List<String> _extractImages(String data) {
    // Local storage uploads intentionally return `/file/img/...`; keep both
    // relative and absolute destinations so the viewer mirrors the renderer.
    final RegExp re = RegExp(r'!\[([^\]]*)\]\(([^)\s]+)\)');
    return re.allMatches(data).map((m) => m.group(2)!).toList(growable: false);
  }

  Future<List<LinkPreviewPayload>> _loadLinkPreviews(
    List<String> urls,
    int generation,
  ) async {
    final previews = await ref
        .read(linkPreviewRepositoryProvider)
        .resolve(urls);
    if (!mounted || generation != _linkPreviewGeneration) return previews;

    // These are the text values GfLinkPreviewCard can expose through GfText.
    // Reserve them in the shared selection registry before the FutureBuilder
    // mounts the card, so late PUA characters cannot collide with an Emoji
    // placeholder allocated by an earlier Markdown block.
    final AppLocalizations? l10n = Localizations.of<AppLocalizations>(
      context,
      AppLocalizations,
    );
    final String source = previews
        .expand(
          (preview) => <String>[
            preview.siteName ?? '',
            preview.displayHost ?? '',
            preview.title ?? '',
            preview.description ?? '',
            if (preview.campus) l10n?.linkPreviewCampusFallbackTitle ?? '',
            if (preview.campus)
              l10n?.linkPreviewCampusFallbackDescription ?? '',
            // 平台卡片的来源行、类型徽标与作者行也都是可见文本，同样要预先登记。
            ...LinkPreviewPlatformLabels.of(l10n, preview).visible,
          ],
        )
        .join('\n');
    if (source == _linkPreviewSelectionSource) return previews;

    setState(() {
      _linkPreviewSelectionSource = source;
      _markdownBody = _buildMarkdownBody(_contentScale);
    });
    // GfSelectionArea must receive the expanded reservation before
    // FutureBuilder exposes the returned preview widgets to the selection
    // scope. Otherwise a descendant's late source registration can make its
    // Emoji placeholder ambiguous and disable copying.
    await WidgetsBinding.instance.endOfFrame;
    return previews;
  }

  void _openViewer(BuildContext context, List<String> urls, int index) {
    Navigator.of(context, rootNavigator: true).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          backgroundColor: const Color(0xFF000000),
          body: SafeArea(
            child: GfImageViewer(
              images: urls,
              initialIndex: index,
              onSaveImage: (String url) => saveImageFromUrl(context, url),
              saveImageLabel: AppLocalizations.of(context).imageSave,
              onShareImage: (String url) => shareImageFromUrl(context, url),
              shareImageLabel: AppLocalizations.of(context).topicShare,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildMarkdownBody(double userScale) {
    final GfColors colors = widget.colors ?? GfTheme.colorsOf(context);
    final readiness = widget.screenshot
        ? ShareImageReadiness.maybeOf(context)
        : null;
    final GfBorders borders = GfTheme.bordersOf(context);
    // One profile drives every rich-content branch, including link previews.
    final GfRichContentTypography profile = GfRichContentTypography.of(
      context,
      userScale: userScale,
      // Compact callers (table embeds) and screenshot exports share the
      // course-review baseline so exports stay within capture bounds.
      compact: widget.compact || widget.screenshot,
    );
    // Only token expansion receives these private image sources. Image alt,
    // titles and even a matching public asset URL remain ordinary user input.
    final stickerImages = <String, ({String name, String url})>{};
    final stickerSources = <String, String>{};
    if (_stickerUrls.isNotEmpty && containsStickerToken(widget.data)) {
      final random = Random.secure();
      final nonce = List.generate(
        4,
        (_) => random.nextInt(1 << 32).toRadixString(16).padLeft(8, '0'),
      ).join();
      for (final entry in _stickerUrls.entries) {
        final source = 'gf-sticker-render:$nonce/${stickerImages.length}';
        stickerImages[source] = (name: entry.key, url: entry.value);
        stickerSources[entry.key] = source;
      }
    }
    final String screenshotSource = widget.screenshot
        ? widget.data.replaceAllMapped(stickerTokenPattern, (match) {
            final name = match.group(1)!;
            return _stickerUrls.containsKey(name)
                ? match[0]!
                : '[${StickerStrings(context).unavailable}]';
          })
        : widget.data;
    final String data = expandStickerTokens(
      expandPostMentions(screenshotSource, widget.mentions),
      stickerSources,
    );
    // Expressions never belong to a photo gallery, including supplied lists.
    final ordinary = _extractImages(
      data,
    ).where((source) => !stickerImages.containsKey(source)).toList();
    final List<String> sourceUrls = widget.images == null
        ? ordinary
        : widget.images!.where(ordinary.contains).toList();
    final List<String> resolvedUrls = sourceUrls
        .map(resolveApiAssetUrl)
        .toList(growable: false);
    final MediaQueryData media = MediaQuery.of(context);

    // web prose.css 图片高度上限:min(360px, 70vh)。
    final double maxImageHeight = media.size.height * 0.7 < 360
        ? media.size.height * 0.7
        : 360;
    final int imageCacheWidth = (media.size.width * media.devicePixelRatio)
        .round();

    final MarkdownConfig config = MarkdownConfig(
      configs: <WidgetConfig>[
        PConfig(textStyle: profile.body),
        // 列表缩进与行距同属 profile,避免各链路各写一套数字。
        ListConfig(
          marginLeft: profile.listIndent,
          marginBottom: profile.listSpacing,
        ),
        LinkConfig(
          style: TextStyle(
            color: colors.primary,
            decoration: TextDecoration.underline,
          ),
          onTap: (String url) {
            // 标签与目标不符的链接按 suspicious 处理（对齐 Web linkRisk）：
            // 白名单与会话信任都不豁免，始终弹确认层（issue #1200）。
            // 按 widget.data 而非展开后的 data 记忆：贴纸 token 展开会插入每次
            // 重建都变的随机 source，用它做键会让缓存永远失效。
            if (_mislabeledSource != widget.data) {
              _mislabeledSource = widget.data;
              _mislabeledTargets = mislabeledLinkTargets(widget.data);
            }
            unawaited(
              LinkNavigation.open(
                context,
                url,
                baseUrl: ref.read(apiClientProvider).baseUrl,
                risk: _mislabeledTargets.contains(url)
                    ? LinkRisk.suspicious
                    : LinkRisk.normalExternal,
              ),
            );
          },
        ),
        H1Config(style: profile.h1),
        H2Config(style: profile.h2),
        H3Config(style: profile.h3),
        H4Config(style: profile.h4),
        H5Config(style: profile.h5),
        H6Config(style: profile.h6), // 图片:contain + 高度约束 + 圆角边框(prose.css img)。
        ImgConfig(
          builder: (String url, Map<String, String> attributes) {
            final sticker = stickerImages[url];
            if (sticker != null) {
              return StickerImage(
                name: sticker.name,
                url: sticker.url,
                size: widget.stickerSize,
              );
            }
            final String resolvedUrl = resolveApiAssetUrl(url);
            readiness?.begin(resolvedUrl);
            var hasFrame = false;
            return GestureDetector(
              onTap: widget.screenshot
                  ? null
                  : () {
                      final int index = sourceUrls.indexOf(url);
                      _openViewer(
                        context,
                        resolvedUrls.isEmpty ? [resolvedUrl] : resolvedUrls,
                        index < 0 ? 0 : index,
                      );
                    },
              onLongPress: widget.screenshot
                  ? null
                  : () async {
                      final bool save = await showGfImageSaveSheet(
                        context,
                        saveImageLabel: AppLocalizations.of(context).imageSave,
                      );
                      if (!mounted || !save) return;
                      await saveImageFromUrl(context, resolvedUrl);
                    },
              child: ConstrainedBox(
                constraints: BoxConstraints(maxHeight: maxImageHeight),
                child: Container(
                  decoration: BoxDecoration(
                    border: Border.all(
                      color: colors.line,
                      width: borders.width,
                    ),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  clipBehavior: Clip.antiAlias,
                  child: GfNetworkImage(
                    resolvedUrl,
                    fit: BoxFit.contain,
                    cacheWidth: imageCacheWidth,
                    frameBuilder: (_, child, frame, _) {
                      hasFrame = frame != null;
                      if (frame != null) readiness?.finish(resolvedUrl);
                      return child;
                    },
                    loadingBuilder: (_, child, progress) {
                      if (readiness?.isFrozen(resolvedUrl) == true ||
                          (widget.screenshot && !hasFrame)) {
                        return SizedBox(
                          height: 80,
                          child: Center(
                            child: GfSymbol(
                              'image-off',
                              color: colors.iconMuted,
                            ),
                          ),
                        );
                      }
                      return child;
                    },
                    // 失败时交给宿主兜底（可重试/在浏览器打开），readiness 仍按原语义收口。
                    onImageError: (_) => readiness?.finish(resolvedUrl),
                  ),
                ),
              ),
            );
          },
        ),
        // 引用:左线 + 底(prose.css blockquote)。
        BlockquoteConfig(
          sideColor: colors.line,
          textColor: profile.quote.color ?? colors.baseContent,
          sideWith: profile.quoteSide,
          padding: profile.quotePadding,
          margin: EdgeInsets.symmetric(vertical: profile.blockSpacing),
        ),
        // 代码块:base-200 底 + line 边框 + 圆角(prose.css pre)。
        // 代码块整体交给共享组件:自带底/边框/语言标签/复制/横向滚动,
        // 与 Wiki、课评 HTML 路径是同一个 widget。
        PreConfig(
          builder: (String code, String language) =>
              GfCodeBlock(profile: profile, code: code, language: language),
        ),
        // 行内代码:error 色 + base-200 底(prose.css code)。
        CodeConfig(
          style: profile.inlineCode.copyWith(backgroundColor: colors.base200),
        ),
        // 表格:line 边框 + 紧凑 padding(prose.css table)。
        // 表格自然列宽,超宽时在自身区域内横滚;整页不出现横向滚动。
        TableConfig(
          border: TableBorder.all(color: colors.line, width: borders.width),
          headerStyle: profile.tableHeader,
          bodyStyle: profile.tableBody,
          headPadding: profile.tableCellPadding,
          bodyPadding: profile.tableCellPadding,
          wrapper: (Widget child) => GfTableViewport(child: child),
        ),
      ],
    );

    Widget buildMarkdown(String source) => MarkdownWidget(
      data: source,
      // The shared selection area below owns selection across every Markdown
      // block; GfText joins its registrar and Twemoji restoration scope.
      selectable: false,
      shrinkWrap: true,
      markdownGenerator: MarkdownGenerator(
        linesMargin: EdgeInsets.symmetric(vertical: profile.paragraphSpacing),
        richTextBuilder: (span) => GfText.rich(span),
      ),
      // Embedded in the page scroll view: never repeat its safe-area insets.
      padding: EdgeInsets.zero,
      physics: const NeverScrollableScrollPhysics(),
      config: config,
    );

    Widget withMarkdownSelection(Widget child) {
      if (!widget.selectable || widget.screenshot) return child;

      // Include decoded numeric entities so every code unit a rendered leaf
      // can expose is reserved before Twemoji placeholder allocation. Avoid
      // parsing Markdown without entity syntax on the common path.
      final String entityText = data.contains('&#')
          ? parseFragment(data).text ?? ''
          : '';
      return GfSelectionArea(
        selectionSource: '$data$entityText$_linkPreviewSelectionSource',
        child: child,
      );
    }

    final List<LinkPreviewMarkdownBlock> blocks = splitLinkPreviewMarkdown(
      data,
    );
    final List<String> previewUrls = blocks
        .where((LinkPreviewMarkdownBlock block) => block.isPreview)
        .map((LinkPreviewMarkdownBlock block) => block.url!)
        .toList(growable: false);
    if (previewUrls.isEmpty || widget.screenshot) {
      return withMarkdownSelection(buildMarkdown(data));
    }

    Future<List<LinkPreviewPayload>> loadPreviews() => _linkPreviews ??=
        _loadLinkPreviews(previewUrls, _linkPreviewGeneration);

    return withMarkdownSelection(
      Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          for (final LinkPreviewMarkdownBlock block in blocks)
            if (block.isPreview)
              _DeferredLinkPreview(
                url: block.url!,
                sourceRevision: _linkPreviewGeneration,
                fallback: buildMarkdown(block.markdown),
                load: loadPreviews,
              )
            else
              buildMarkdown(block.markdown),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) => ResolvedStickerContent(
    content: widget.data,
    resolve: !widget.screenshot,
    builder: (urls) {
      final double contentScale = ref.watch(contentFontScaleProvider);
      if (!mapEquals(_stickerUrls, urls) || contentScale != _contentScale) {
        _stickerUrls = urls;
        _contentScale = contentScale;
        _markdownBody = _buildMarkdownBody(contentScale);
      }
      return _markdownBody;
    },
  );
}

class _DeferredLinkPreview extends StatefulWidget {
  const _DeferredLinkPreview({
    required this.url,
    required this.sourceRevision,
    required this.fallback,
    required this.load,
  });

  final String url;
  final int sourceRevision;
  final Widget fallback;
  final Future<List<LinkPreviewPayload>> Function() load;

  @override
  State<_DeferredLinkPreview> createState() => _DeferredLinkPreviewState();
}

class _DeferredLinkPreviewState extends State<_DeferredLinkPreview> {
  Future<List<LinkPreviewPayload>>? _future;
  Timer? _retry;
  ScrollPosition? _scrollPosition;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadWhenVisible());
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final ScrollPosition? nextPosition = Scrollable.maybeOf(context)?.position;
    if (identical(nextPosition, _scrollPosition)) return;
    _scrollPosition?.removeListener(_loadWhenVisible);
    _scrollPosition = nextPosition;
    _scrollPosition?.addListener(_loadWhenVisible);
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadWhenVisible());
  }

  @override
  void didUpdateWidget(covariant _DeferredLinkPreview oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url == widget.url &&
        oldWidget.sourceRevision == widget.sourceRevision) {
      return;
    }
    _retry?.cancel();
    _future = null;
    WidgetsBinding.instance.addPostFrameCallback((_) => _loadWhenVisible());
  }

  void _loadWhenVisible() {
    if (!mounted || _future != null) return;
    final RenderObject? renderObject = context.findRenderObject();
    if (renderObject is! RenderBox ||
        !renderObject.attached ||
        !renderObject.hasSize) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _loadWhenVisible());
      return;
    }
    final double top = renderObject.localToGlobal(Offset.zero).dy;
    final double bottom = top + renderObject.size.height;
    final double viewportHeight = MediaQuery.sizeOf(context).height;
    if (bottom < -160 || top > viewportHeight + 160) return;
    if (Scrollable.recommendDeferredLoadingForContext(context)) {
      _retry = Timer(const Duration(milliseconds: 160), _loadWhenVisible);
      return;
    }
    final future = widget.load();
    setState(() {
      _future = future;
    });
  }

  @override
  void dispose() {
    _retry?.cancel();
    _scrollPosition?.removeListener(_loadWhenVisible);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final Future<List<LinkPreviewPayload>>? future = _future;
    if (future == null) return widget.fallback;
    return FutureBuilder<List<LinkPreviewPayload>>(
      future: future,
      builder:
          (
            BuildContext context,
            AsyncSnapshot<List<LinkPreviewPayload>> snapshot,
          ) {
            LinkPreviewPayload? preview;
            for (final LinkPreviewPayload item
                in snapshot.data ?? const <LinkPreviewPayload>[]) {
              if (item.requestedUrl == widget.url && item.isReady) {
                preview = item;
                break;
              }
            }
            return preview == null
                ? widget.fallback
                : GfLinkPreviewCard(preview: preview);
          },
    );
  }
}
