/// 图片相对宽度（docs/decisions/0131-relative-image-width.md）。
///
/// Markdown 图片地址末尾的 `#w=<预设>` 片段表示按正文宽度的百分比显示，窄屏上不小于
/// [imageMinRenderedWidth]。与 Web `runtime/image-width.ts`、prose.css 同一规则，共用
/// `packages/api-contract/fixtures/markdown-image-width.json`。
library;

import 'package:markdown/markdown.dart' as md;

const List<int> imageWidthPresets = <int>[25, 33, 50, 66, 75, 100];

/// 渲染下限（逻辑像素）。
const double imageMinRenderedWidth = 160;

final RegExp _widthFragment = RegExp(r'#w=(\d{2,3})$');
final RegExp _ownFragment = RegExp(r'^#w=\d*$');

/// 地址里的宽度预设；没有或不是预设值时为 null。
int? imageWidthOf(String src) {
  final RegExpMatch? match = _widthFragment.firstMatch(src);
  final int? value = match == null ? null : int.tryParse(match.group(1)!);
  return value != null && imageWidthPresets.contains(value) ? value : null;
}

/// 已带其他片段（如 SVG 精灵 `#icon`）的地址不改动。
bool canSetImageWidth(String src) {
  final int hash = src.indexOf('#');
  return hash < 0 || _ownFragment.hasMatch(src.substring(hash));
}

/// 写入或清除宽度（null = 按图片原始尺寸自适应）。
String withImageWidth(String src, int? width) {
  if (!canSetImageWidth(src)) return src;
  final int hash = src.indexOf('#');
  final String base = hash < 0 ? src : src.substring(0, hash);
  return width == null ? base : '$base#w=$width';
}

/// 拖拽得到的宽度比例吸附到最近的预设（≥0.9 视为铺满）。
int snapImageWidth(double ratio) {
  if (ratio >= 0.9) return 100;
  int best = imageWidthPresets.first;
  for (final int preset in imageWidthPresets) {
    if ((preset / 100 - ratio).abs() < (best / 100 - ratio).abs()) best = preset;
  }
  return best;
}

/// 只由图片（及空白）组成且至少一张带宽度的段落，返回各图片属性；其余返回 null。
/// 端上据此把段落排成居中换行的图片行，对应 Web prose.css 的并排规则。
List<Map<String, String>>? sizedImageRow(md.Element paragraph) {
  final List<Map<String, String>> images = <Map<String, String>>[];
  for (final md.Node node in paragraph.children ?? const <md.Node>[]) {
    if (node is md.Element && node.tag == 'img') {
      images.add(node.attributes);
    } else if (node is! md.Text || node.text.trim().isNotEmpty) {
      return null;
    }
  }
  return images.any((a) => imageWidthOf(a['src'] ?? '') != null)
      ? images
      : null;
}
