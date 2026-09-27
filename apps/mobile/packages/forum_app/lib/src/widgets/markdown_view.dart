import 'dart:async';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../images/image_save.dart';
import '../link_navigation.dart';
import '../providers.dart';
import 'stickers/sticker_image.dart';
import 'stickers/resolved_sticker_content.dart';

/// Shared prose scale for reading, writing and preview.
TextStyle readingBodyStyle(BuildContext context) =>
    GfTheme.typographyOf(context).body.copyWith(fontSize: 18, height: 1.55);

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
/// 语法保留表情语义，使用紧凑 renderer 并排除灯箱;库未就绪时先用原文渲染,拉取完成
/// 后异步刷新,未知/停用 token 保持原文。
class GfMarkdownView extends ConsumerStatefulWidget {
  const GfMarkdownView({
    super.key,
    required this.data,
    this.images,
    this.mentions = const <PostMention>[],
    this.selectable = false,
  });

  final String data;
  final List<PostMention> mentions;

  /// 已知图片列表(取自 markdown 的图片引用);为 null 时从内容提取。
  final List<String>? images;

  final bool selectable;

  @override
  ConsumerState<GfMarkdownView> createState() => _GfMarkdownViewState();
}

class _GfMarkdownViewState extends ConsumerState<GfMarkdownView> {
  late Widget _markdownBody;
  Future<List<LinkPreviewPayload>>? _linkPreviews;

  Map<String, String> _stickerUrls = const {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _markdownBody = _buildMarkdownBody();
  }

  @override
  void didUpdateWidget(covariant GfMarkdownView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.data != widget.data ||
        oldWidget.selectable != widget.selectable ||
        !listEquals(oldWidget.images, widget.images) ||
        !listEquals(oldWidget.mentions, widget.mentions)) {
      if (oldWidget.data != widget.data) _linkPreviews = null;
      _markdownBody = _buildMarkdownBody();
    }
  }

  List<String> _extractImages(String data) {
    // Local storage uploads intentionally return `/file/img/...`; keep both
    // relative and absolute destinations so the viewer mirrors the renderer.
    final RegExp re = RegExp(r'!\[([^\]]*)\]\(([^)\s]+)\)');
    return re.allMatches(data).map((m) => m.group(2)!).toList(growable: false);
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

  Widget _buildMarkdownBody() {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
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
    final String data = expandStickerTokens(
      expandPostMentions(widget.data, widget.mentions),
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
        PConfig(textStyle: readingBodyStyle(context)),
        LinkConfig(
          style: TextStyle(
            color: colors.primary,
            decoration: TextDecoration.underline,
          ),
          onTap: (String url) {
            unawaited(
              LinkNavigation.open(
                context,
                url,
                baseUrl: ref.read(apiClientProvider).baseUrl,
              ),
            );
          },
        ),
        H1Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontSize: 28, fontWeight: FontWeight.w700, height: 1.3),
        ),
        H2Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontSize: 24, fontWeight: FontWeight.w700, height: 1.35),
        ),
        H3Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontSize: 21, fontWeight: FontWeight.w600, height: 1.4),
        ),
        H4Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontSize: 19, fontWeight: FontWeight.w600),
        ),
        H5Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontWeight: FontWeight.w600),
        ),
        H6Config(
          style: readingBodyStyle(
            context,
          ).copyWith(fontWeight: FontWeight.w600, color: colors.iconMuted),
        ),
        // 图片:contain + 高度约束 + 圆角边框(prose.css img)。
        ImgConfig(
          builder: (String url, Map<String, String> attributes) {
            final sticker = stickerImages[url];
            if (sticker != null) {
              return StickerImage(name: sticker.name, url: sticker.url);
            }
            final String resolvedUrl = resolveApiAssetUrl(url);
            return GestureDetector(
              onTap: () {
                final int index = sourceUrls.indexOf(url);
                _openViewer(
                  context,
                  resolvedUrls.isEmpty ? [resolvedUrl] : resolvedUrls,
                  index < 0 ? 0 : index,
                );
              },
              onLongPress: () async {
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
                  child: Image.network(
                    resolvedUrl,
                    fit: BoxFit.contain,
                    cacheWidth: imageCacheWidth,
                    errorBuilder: (_, _, _) => SizedBox(
                      height: 60,
                      child: Center(
                        child: GfSymbol('image-off', color: colors.iconMuted),
                      ),
                    ),
                  ),
                ),
              ),
            );
          },
        ),
        // 引用:左线 + 底(prose.css blockquote)。
        BlockquoteConfig(
          sideColor: colors.line,
          textColor: colors.baseContent.withValues(alpha: 0.75),
          sideWith: 4,
          padding: const EdgeInsets.fromLTRB(12, 4, 12, 4),
          margin: const EdgeInsets.symmetric(vertical: 8),
        ),
        // 代码块:base-200 底 + line 边框 + 圆角(prose.css pre)。
        PreConfig(
          decoration: BoxDecoration(
            color: colors.base200,
            border: Border.all(color: colors.line, width: borders.width),
            borderRadius: BorderRadius.circular(8),
          ),
          padding: const EdgeInsets.all(12),
          margin: const EdgeInsets.symmetric(vertical: 8),
          textStyle: TextStyle(
            fontSize: 16,
            height: 1.5,
            color: colors.baseContent,
          ),
        ),
        // 行内代码:error 色 + base-200 底(prose.css code)。
        CodeConfig(
          style: TextStyle(
            color: colors.error,
            backgroundColor: colors.base200,
            fontSize: 16,
          ),
        ),
        // 表格:line 边框 + 紧凑 padding(prose.css table)。
        TableConfig(
          border: TableBorder.all(color: colors.line, width: borders.width),
          headerStyle: TextStyle(
            color: colors.baseContent,
            fontWeight: FontWeight.w600,
            fontSize: 17,
          ),
          bodyStyle: TextStyle(
            color: colors.baseContent,
            fontSize: 17,
            height: 1.5,
          ),
          headPadding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
          bodyPadding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
        ),
      ],
    );

    Widget buildMarkdown(String source) => MarkdownWidget(
      data: source,
      selectable: widget.selectable,
      shrinkWrap: true,
      markdownGenerator: MarkdownGenerator(
        linesMargin: const EdgeInsets.symmetric(vertical: 3),
      ),
      // Embedded in the page scroll view: never repeat its safe-area insets.
      padding: EdgeInsets.zero,
      physics: const NeverScrollableScrollPhysics(),
      config: config,
    );

    final List<LinkPreviewMarkdownBlock> blocks = splitLinkPreviewMarkdown(
      data,
    );
    final List<String> previewUrls = blocks
        .where((LinkPreviewMarkdownBlock block) => block.isPreview)
        .map((LinkPreviewMarkdownBlock block) => block.url!)
        .toList(growable: false);
    if (previewUrls.isEmpty) return buildMarkdown(data);

    Future<List<LinkPreviewPayload>> loadPreviews() => _linkPreviews ??= ref
        .read(linkPreviewRepositoryProvider)
        .resolve(previewUrls);

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        for (final LinkPreviewMarkdownBlock block in blocks)
          if (block.isPreview)
            _DeferredLinkPreview(
              url: block.url!,
              fallback: buildMarkdown(block.markdown),
              load: loadPreviews,
            )
          else
            buildMarkdown(block.markdown),
      ],
    );
  }

  @override
  Widget build(BuildContext context) => ResolvedStickerContent(
    content: widget.data,
    builder: (urls) {
      if (!mapEquals(_stickerUrls, urls)) {
        _stickerUrls = urls;
        _markdownBody = _buildMarkdownBody();
      }
      return _markdownBody;
    },
  );
}

class _DeferredLinkPreview extends StatefulWidget {
  const _DeferredLinkPreview({
    required this.url,
    required this.fallback,
    required this.load,
  });

  final String url;
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
    if (oldWidget.url == widget.url) return;
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

class GfLinkPreviewCard extends ConsumerStatefulWidget {
  const GfLinkPreviewCard({super.key, required this.preview});

  final LinkPreviewPayload preview;

  @override
  ConsumerState<GfLinkPreviewCard> createState() => _GfLinkPreviewCardState();
}

class _GfLinkPreviewCardState extends ConsumerState<GfLinkPreviewCard> {
  /// Web 端同一套档位：<480 不显示描述（小卡化最缺的正是垂直空间），
  /// 480–639 一行，≥640 两行。
  static int descriptionLinesFor(double cardWidth) =>
      cardWidth < 480 ? 0 : (cardWidth < 640 ? 1 : 2);

  bool _coverFailed = false;

  @override
  Widget build(BuildContext context) {
    final LinkPreviewPayload preview = widget.preview;
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final String? coverUrl =
        !_coverFailed && preview.imageUrl?.isNotEmpty == true
        ? resolveApiAssetUrl(preview.imageUrl!)
        : null;
    final String? faviconUrl = preview.faviconUrl?.isNotEmpty == true
        ? resolveApiAssetUrl(preview.faviconUrl!)
        : null;
    final String sourceLabel = preview.siteName?.trim().isNotEmpty == true
        ? preview.siteName!.trim()
        : (preview.displayHost?.trim() ?? '');
    // #733 要求离开前能看到真实 host；与来源行是同一个字符串时不再重复占一行。
    final String rawHost = preview.displayHost?.trim() ?? '';
    final String hostLabel = rawHost.toLowerCase() == sourceLabel.toLowerCase()
        ? ''
        : rawHost;
    final AppLocalizations? l10n = Localizations.of<AppLocalizations>(
      context,
      AppLocalizations,
    );
    // 校园网卡片由服务端按部署配置本地渲染：配置没给名字时 title / description
    // 为空，兜底文案必须由客户端按语言出（服务端不留任何中文，issue #729）。
    final String title = preview.title?.trim().isNotEmpty == true
        ? preview.title!.trim()
        : (preview.campus ? (l10n?.linkPreviewCampusFallbackTitle ?? '') : '');
    final String description = preview.description?.trim().isNotEmpty == true
        ? preview.description!.trim()
        : (preview.campus
              ? (l10n?.linkPreviewCampusFallbackDescription ?? '')
              : '');

    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final bool railCover = constraints.maxWidth >= 640;
        final int descriptionLines = descriptionLinesFor(constraints.maxWidth);
        // 封面走 Stack + Positioned，刻意不参与卡片高度计算：竖版封面若留在
        // 文档流里，会按自身比例把整张卡片撑高（Web 端实测 164px → 362px）。
        // <640px 是右上角 56px 方形缩略图；≥640px 是保留 16px 内边距的
        // 168px 居中缩略图，宽屏使用 contain 完整显示，避免裁掉封面两端。
        final double coverInset = railCover ? 16 : 14;
        final double coverWidth = railCover ? 168 : 56;
        final double bodyRightPadding = coverUrl == null
            ? 14
            : (railCover ? coverWidth + 16 : 14 + coverWidth + 12);

        final Widget body = Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Row(
              children: <Widget>[
                if (faviconUrl != null) ...<Widget>[
                  ClipRRect(
                    borderRadius: BorderRadius.circular(4),
                    child: Image.network(
                      faviconUrl,
                      width: 16,
                      height: 16,
                      cacheWidth: (32 * MediaQuery.devicePixelRatioOf(context))
                          .round(),
                      cacheHeight: (32 * MediaQuery.devicePixelRatioOf(context))
                          .round(),
                      fit: BoxFit.contain,
                      errorBuilder: (_, _, _) => const SizedBox.shrink(),
                    ),
                  ),
                  const SizedBox(width: 8),
                ],
                Expanded(
                  child: Text(
                    sourceLabel,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: colors.baseContent.withValues(alpha: 0.6),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              title,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.titleSmall?.copyWith(
                color: colors.baseContent,
                fontWeight: FontWeight.w600,
                height: 1.35,
              ),
            ),
            if (description.isNotEmpty && descriptionLines > 0) ...<Widget>[
              const SizedBox(height: 12),
              Text(
                description,
                maxLines: descriptionLines,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  fontSize: 13,
                  color: colors.baseContent.withValues(alpha: 0.65),
                  height: 1.4,
                ),
              ),
            ],
            if (hostLabel.isNotEmpty) ...<Widget>[
              const SizedBox(height: 12),
              Text(
                hostLabel,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  fontSize: 11,
                  color: colors.baseContent.withValues(alpha: 0.5),
                ),
              ),
            ],
          ],
        );

        return Semantics(
          link: true,
          label: <String>[
            title,
            if (sourceLabel.isNotEmpty) sourceLabel,
          ].join(', '),
          hint: preview.kind == 'external'
              ? l10n?.linkPreviewExternalTitle
              : null,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Material(
              color: colors.base100,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(radii.box),
                side: BorderSide(color: colors.line, width: borders.width),
              ),
              clipBehavior: Clip.antiAlias,
              child: InkWell(
                onTap: () => LinkNavigation.open(
                  context,
                  preview.url!,
                  baseUrl: ref.read(apiClientProvider).baseUrl,
                  previewKind: preview.kind,
                ),
                child: Stack(
                  children: <Widget>[
                    ConstrainedBox(
                      // 缩略图绝对定位不参与高度计算，短卡片要自己留够 56 + 上下 14。
                      constraints: BoxConstraints(
                        minHeight: coverUrl != null && !railCover
                            ? coverWidth + 28
                            : 0,
                      ),
                      child: Padding(
                        padding: EdgeInsets.fromLTRB(
                          14,
                          14,
                          bodyRightPadding,
                          14,
                        ),
                        child: body,
                      ),
                    ),
                    if (coverUrl != null)
                      Positioned(
                        top: coverInset,
                        right: coverInset,
                        bottom: railCover ? coverInset : null,
                        width: coverWidth,
                        height: railCover ? null : coverWidth,
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(6),
                          child: Image.network(
                            coverUrl,
                            fit: railCover ? BoxFit.contain : BoxFit.cover,
                            cacheWidth:
                                (coverWidth *
                                        MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                            cacheHeight:
                                (coverWidth *
                                        MediaQuery.devicePixelRatioOf(context))
                                    .round(),
                            errorBuilder: (_, _, _) {
                              // 封面挂了就把让位空间一起收回，不留死白。
                              WidgetsBinding.instance.addPostFrameCallback((_) {
                                if (mounted && !_coverFailed) {
                                  setState(() => _coverFailed = true);
                                }
                              });
                              return const SizedBox.shrink();
                            },
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}
