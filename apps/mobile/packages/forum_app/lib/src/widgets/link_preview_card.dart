import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';

import '../../l10n/app_localizations.dart';
import '../asset_url.dart';
import '../link_navigation.dart';
import '../providers.dart';

/// 外部链接预览卡片（服务端 `/api/link-previews/resolve` 的渲染端）。
///
/// 服务端只给标识与原文：平台名、资源类型徽标与作者称呼都由客户端按语言渲染
/// （issue #1201，见 MADR 0094）。卡片与正文渲染分开，便于单独测试与复用。
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
    final AppLocalizations? l10n = Localizations.of<AppLocalizations>(
      context,
      AppLocalizations,
    );
    final LinkPreviewPlatformLabels platformLabels =
        LinkPreviewPlatformLabels.of(l10n, preview);
    // 来源名优先平台名（客户端按语言渲染），其次页面站点名，再回退 host。
    final String sourceLabel = platformLabels.platform.isNotEmpty
        ? platformLabels.platform
        : (preview.siteName?.trim().isNotEmpty == true
              ? preview.siteName!.trim()
              : (preview.displayHost?.trim() ?? ''));
    // #733 要求离开前能看到真实 host；与来源行是同一个字符串时不再重复占一行。
    final String rawHost = preview.displayHost?.trim() ?? '';
    final String hostLabel = rawHost.toLowerCase() == sourceLabel.toLowerCase()
        ? ''
        : rawHost;
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
                    child: GfNetworkImage(
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
                  child: Row(
                    children: <Widget>[
                      Flexible(
                        child: GfText(
                          sourceLabel,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(
                                color: colors.baseContent.withValues(alpha: 0.6),
                              ),
                        ),
                      ),
                      // 资源类型徽标只在严格识别出平台资源时出现（服务端
                      // platform + resourceType），通用卡片上不会有它。
                      if (platformLabels.resourceType.isNotEmpty) ...<Widget>[
                        const SizedBox(width: 6),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 2,
                          ),
                          decoration: BoxDecoration(
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(
                              color: colors.line,
                              width: borders.width,
                            ),
                          ),
                          child: GfText(
                            platformLabels.resourceType,
                            style: Theme.of(context).textTheme.bodySmall
                                ?.copyWith(
                                  fontSize: 11,
                                  height: 1.2,
                                  color: colors.baseContent.withValues(
                                    alpha: 0.6,
                                  ),
                                ),
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            GfText(
              title,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.titleSmall?.copyWith(
                color: colors.baseContent,
                fontWeight: FontWeight.w600,
                height: 1.35,
              ),
            ),
            // 作者/艺术家缺失就整行省略：服务端从不合成字段，客户端也不留空标签。
            if (platformLabels.author.isNotEmpty) ...<Widget>[
              const SizedBox(height: 4),
              GfText(
                platformLabels.author,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  fontSize: 12,
                  color: colors.baseContent.withValues(alpha: 0.6),
                ),
              ),
            ],
            if (description.isNotEmpty && descriptionLines > 0) ...<Widget>[
              const SizedBox(height: 12),
              GfText(
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
              GfText(
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
            if (platformLabels.author.isNotEmpty) platformLabels.author,
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
                          child: GfNetworkImage(
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

/// 平台卡片的三段可见文案（平台名 / 资源类型 / 作者行）。
///
/// 服务端只给平台标识与作者原文，展示名与称呼由客户端按语言渲染；卡片与选择注
/// 册表必须共用同一份取值逻辑，否则注册表会漏登记一段真实渲染出来的文本，复制
/// 与 Emoji 占位符归属随之错位。
class LinkPreviewPlatformLabels {
  const LinkPreviewPlatformLabels({
    required this.platform,
    required this.resourceType,
    required this.author,
  });

  factory LinkPreviewPlatformLabels.of(
    AppLocalizations? l10n,
    LinkPreviewPayload preview,
  ) {
    final String author = preview.author?.trim() ?? '';
    return LinkPreviewPlatformLabels(
      platform: switch (preview.platform) {
        'bilibili' => l10n?.linkPreviewPlatformBilibili ?? '',
        'apple_music' => l10n?.linkPreviewPlatformAppleMusic ?? '',
        _ => '',
      },
      resourceType: switch (preview.resourceType) {
        'video' => l10n?.linkPreviewResourceTypeVideo ?? '',
        'song' => l10n?.linkPreviewResourceTypeSong ?? '',
        'album' => l10n?.linkPreviewResourceTypeAlbum ?? '',
        _ => '',
      },
      author: author.isEmpty
          ? ''
          : (preview.platform == 'apple_music'
                ? (l10n?.linkPreviewAuthorArtist(author) ?? '')
                : (l10n?.linkPreviewAuthorUploader(author) ?? '')),
    );
  }

  final String platform;
  final String resourceType;
  final String author;

  /// 卡片上会真实渲染的文本，供选择注册表预先登记。
  List<String> get visible => <String>[
    platform,
    resourceType,
    author,
  ].where((String value) => value.isNotEmpty).toList(growable: false);
}
