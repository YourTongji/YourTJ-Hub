import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../images/image_save.dart';
import '../providers.dart';

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
/// 语法复用图片渲染/点击查看链路;表情包库未就绪时先用原文渲染,拉取完成
/// 后异步刷新,未知/停用 token 保持原文。
class GfMarkdownView extends ConsumerStatefulWidget {
  const GfMarkdownView({
    super.key,
    required this.data,
    this.images,
    this.selectable = false,
  });

  final String data;

  /// 已知图片列表(取自 markdown 的图片引用);为 null 时从内容提取。
  final List<String>? images;

  final bool selectable;

  @override
  ConsumerState<GfMarkdownView> createState() => _GfMarkdownViewState();
}

class _GfMarkdownViewState extends ConsumerState<GfMarkdownView> {
  late Widget _markdownBody;

  /// 本条内容渲染时表情包库是否已就绪(决定要不要在库就绪后刷新)。
  bool _stickersResolved = false;

  @override
  void initState() {
    super.initState();
    _ensureStickersResolved();
  }

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
        !listEquals(oldWidget.images, widget.images)) {
      _ensureStickersResolved();
      _markdownBody = _buildMarkdownBody();
    }
  }

  /// 内容含 token 且表情包库未就绪时触发一次拉取,完成后刷新渲染。
  void _ensureStickersResolved() {
    if (_stickersResolved || !containsStickerToken(widget.data)) return;
    _stickersResolved = true;
    ref
        .read(stickerLibraryProvider)
        .load()
        .then((_) {
          if (mounted) setState(_rebuildMarkdownBody);
        })
        .catchError((Object _) {
          // 拉取失败保持原文渲染;库不缓存失败,下次重建(切换楼层等)重试。
          if (mounted) _stickersResolved = false;
        });
  }

  void _rebuildMarkdownBody() {
    _markdownBody = _buildMarkdownBody();
  }

  List<String> _extractImages(String data) {
    // Local storage uploads intentionally return `/file/img/...`; keep both
    // relative and absolute destinations so the viewer mirrors the renderer.
    final RegExp re = RegExp(r'!\[[^\]]*\]\(([^)\s]+)\)');
    return re.allMatches(data).map((m) => m.group(1)!).toList(growable: false);
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
    final String data = expandStickerTokens(
      widget.data,
      ref.read(stickerLibraryProvider).urlByName,
    );
    // 从展开后的内容提取图片引用,让贴纸图也进入点击查看的图片列表。
    final List<String> sourceUrls = widget.images ?? _extractImages(data);
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

    return MarkdownWidget(
      data: data,
      selectable: widget.selectable,
      shrinkWrap: true,
      markdownGenerator: MarkdownGenerator(
        linesMargin: const EdgeInsets.symmetric(vertical: 3),
      ),
      // Embedded in the page scroll view: never repeat its safe-area insets.
      padding: EdgeInsets.zero,
      physics: const NeverScrollableScrollPhysics(),
      config: MarkdownConfig(
        configs: <WidgetConfig>[
          PConfig(textStyle: readingBodyStyle(context)),
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
                          child: Icon(
                            Icons.broken_image,
                            color: colors.iconMuted,
                          ),
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
      ),
    );
  }

  @override
  Widget build(BuildContext context) => _markdownBody;
}
