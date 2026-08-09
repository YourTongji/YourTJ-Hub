import 'package:flutter/material.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

import '../asset_url.dart';

/// 帖子 markdown 渲染视图。
///
/// 基于 markdown_widget,渲染样式对齐 web 端 prose.css 语义:
/// - 引用:左 4px line 线 + base-200 底 + 弱化文字
/// - 代码块:base-200 底 + line 边框 + 圆角
/// - 行内代码:error 色 + base-200 底
/// - 表格:line 边框
/// - 图片:object-contain + max-height min(360, 70vh) + 圆角边框
/// 图片点击打开 [GfImageViewer] 全屏查看(web MarkdownImageViewer.vue 语义)。
class GfMarkdownView extends StatelessWidget {
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

  List<String> _extractImages() {
    // Local storage uploads intentionally return `/file/img/...`; keep both
    // relative and absolute destinations so the viewer mirrors the renderer.
    final RegExp re = RegExp(r'!\[[^\]]*\]\(([^)\s]+)\)');
    return re.allMatches(data).map((m) => m.group(1)!).toList(growable: false);
  }

  void _openViewer(BuildContext context, List<String> urls, int index) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          backgroundColor: const Color(0xFF000000),
          body: SafeArea(
            child: GfImageViewer(images: urls, initialIndex: index),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final List<String> sourceUrls = images ?? _extractImages();
    final List<String> resolvedUrls = sourceUrls
        .map(resolveApiAssetUrl)
        .toList(growable: false);

    // web prose.css 图片高度上限:min(360px, 70vh)。
    final double maxImageHeight = MediaQuery.of(context).size.height * 0.7 < 360
        ? MediaQuery.of(context).size.height * 0.7
        : 360;

    return MarkdownWidget(
      data: data,
      selectable: selectable,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      config: MarkdownConfig(
        configs: <WidgetConfig>[
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
            textStyle: const TextStyle(fontSize: 14, height: 1.5),
          ),
          // 行内代码:error 色 + base-200 底(prose.css code)。
          CodeConfig(
            style: TextStyle(
              color: colors.error,
              backgroundColor: colors.base200,
              fontSize: 13,
            ),
          ),
          // 表格:line 边框 + 紧凑 padding(prose.css table)。
          TableConfig(
            border: TableBorder.all(color: colors.line, width: borders.width),
            headerStyle: TextStyle(
              color: colors.baseContent,
              fontWeight: FontWeight.w600,
              fontSize: 14,
            ),
            bodyStyle: TextStyle(color: colors.baseContent, fontSize: 14),
            headPadding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
            bodyPadding: const EdgeInsets.fromLTRB(8, 6, 8, 6),
          ),
        ],
      ),
    );
  }
}
