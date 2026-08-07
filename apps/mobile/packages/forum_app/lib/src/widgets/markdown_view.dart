import 'package:flutter/material.dart';
import 'package:markdown_widget/markdown_widget.dart';
import 'package:ui_kit/ui_kit.dart';

/// 帖子 markdown 渲染视图。
///
/// 基于 markdown_widget,对齐 web 端 markdown-it 渲染范围:
/// 标题/粗斜体/列表/引用/表格/任务列表/代码高亮/图片。
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
    final RegExp re = RegExp(r'!\[[^\]]*\]\((https?://[^)\s]+)\)');
    return re.allMatches(data).map((m) => m.group(1)!).toList(growable: false);
  }

  void _openViewer(BuildContext context, List<String> urls, int index) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          backgroundColor: Colors.black,
          body: SafeArea(child: GfImageViewer(images: urls, initialIndex: index)),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final List<String> urls = images ?? _extractImages();

    return MarkdownWidget(
      data: data,
      selectable: selectable,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      config: MarkdownConfig(
        configs: [
          ImgConfig(
            builder: (String url, Map<String, String> attributes) {
              return GestureDetector(
                onTap: () {
                  final int index = urls.indexOf(url);
                  _openViewer(context, urls, index < 0 ? 0 : index);
                },
                child: Image.network(
                  url,
                  fit: BoxFit.cover,
                  errorBuilder: (_, _, _) => const SizedBox(
                    height: 60,
                    child: Center(child: Icon(Icons.broken_image, color: Colors.grey)),
                  ),
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}
