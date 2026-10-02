import 'dart:isolate';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';

import '../../images/image_save.dart';
import '../../../l10n/app_localizations.dart';
import 'share_image_theme.dart';
import 'share_image_readiness.dart';

export 'share_image_theme.dart';

const double shareImageLogicalWidth = 375;
// ponytail: captures are 3x for retina clarity; keep the 4096 tile and 48 MiB
// ceilings — raise them only if a real card is rejected on device.
const double _shareImagePixelRatio = 3;
const int _maxTilePixels = 4096;
const int _maxImageBytes = 48 * 1024 * 1024;
// Serialize image capture so concurrent previews cannot compete for memory.
bool _captureInProgress = false;

Future<void> showShareImagePreview(
  BuildContext context, {
  required String fileName,
  required Widget Function(ShareImageTheme theme) cardBuilder,
}) async {
  final l10n = AppLocalizations.of(context);
  await showGfBottomSheet<void>(
    context,
    barrierDismissible: false,
    enableDrag: false,
    height: MediaQuery.sizeOf(context).height * .9,
    builder: (_) => _ShareImagePreview(
      fileName: _safePngFileName(fileName),
      cardBuilder: cardBuilder,
      themes: ShareImageTheme.all(l10n),
    ),
  );
}

String _safePngFileName(String value) {
  final cleaned = value.replaceAll(RegExp(r'[^A-Za-z0-9._-]'), '_');
  final stem = cleaned.replaceFirst(
    RegExp(r'\.png$', caseSensitive: false),
    '',
  );
  return '${stem.isEmpty ? 'yourtj-share' : stem}.png';
}

class ShareImageCard extends StatelessWidget {
  const ShareImageCard({
    super.key,
    required this.theme,
    required this.child,
    this.footerTrailing,
    this.canvasColor,
  });

  final ShareImageTheme theme;
  final Widget child;

  /// Optional canvas behind [child] (defaults to the theme's `base100`); cards
  /// that float their own surface use the deeper `base200`.
  final Color? canvasColor;

  /// Optional right-aligned footer content (site/link) next to the brand mark.
  final Widget? footerTrailing;

  @override
  Widget build(BuildContext context) {
    final current = MediaQuery.of(context);
    return MediaQuery(
      data: current.copyWith(
        size: const Size(shareImageLogicalWidth, 1024),
        devicePixelRatio: _shareImagePixelRatio,
        textScaler: TextScaler.linear(1),
      ),
      child: Theme(
        data: gfThemeData(theme.brightness, overrides: theme.colors),
        child: SizedBox(
          width: shareImageLogicalWidth,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: canvasColor ?? theme.colors.base100,
            ),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  child,
                  const SizedBox(height: 18),
                  Row(
                    children: [
                      Image.asset(
                        'assets/splash/brand_mark.png',
                        width: 22,
                        height: 22,
                        fit: BoxFit.contain,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        'YourTJ',
                        style: TextStyle(
                          color: theme.colors.iconMuted,
                          fontWeight: FontWeight.w700,
                          fontSize: 13,
                        ),
                      ),
                      if (footerTrailing != null) ...<Widget>[
                        const Spacer(),
                        footerTrailing!,
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Network images outside GfMarkdownView (for example a course image) that
/// must settle before a share card is captured.
class ShareImageNetworkImage extends StatelessWidget {
  const ShareImageNetworkImage(
    this.url, {
    super.key,
    this.width,
    this.height,
    this.fit,
    this.cacheWidth,
    this.errorBuilder,
  });
  final String url;
  final double? width;
  final double? height;
  final BoxFit? fit;
  final int? cacheWidth;
  final WidgetBuilder? errorBuilder;

  @override
  Widget build(BuildContext context) {
    final tracker = ShareImageReadiness.maybeOf(context);
    tracker?.begin(url);
    var hasFrame = false;
    return GfNetworkImage(
      url,
      width: width,
      height: height,
      fit: fit,
      cacheWidth: cacheWidth ?? ((width ?? shareImageLogicalWidth) * 2).round(),
      frameBuilder: (_, child, frame, _) {
        hasFrame = frame != null;
        if (frame != null) tracker?.finish(url);
        return child;
      },
      loadingBuilder: (_, child, progress) {
        if (tracker?.isFrozen(url) == true || !hasFrame) {
          return SizedBox(
            height: height ?? 80,
            width: width,
            child: const Center(child: GfSymbol('image-off')),
          );
        }
        return progress == null
            ? child
            : SizedBox(
                height: height ?? 80,
                width: width,
                child: const Center(child: GfSymbol('image-off')),
              );
      },
      errorBuilder: (context, _, _) {
        tracker?.finish(url);
        return errorBuilder?.call(context) ?? const SizedBox.shrink();
      },
    );
  }
}

class _ShareImagePreview extends StatefulWidget {
  const _ShareImagePreview({
    required this.fileName,
    required this.cardBuilder,
    required this.themes,
  });
  final String fileName;
  final Widget Function(ShareImageTheme) cardBuilder;
  final List<ShareImageTheme> themes;
  @override
  State<_ShareImagePreview> createState() => _ShareImagePreviewState();
}

class _ShareImagePreviewState extends State<_ShareImagePreview> {
  final _boundaryKey = GlobalKey();
  final _readiness = ShareImageReadinessTracker();
  int _selected = 0;
  bool _working = false;

  @override
  void dispose() {
    _readiness.dispose();
    super.dispose();
  }

  Future<void> _act(bool save) async {
    if (_working || _captureInProgress) return;
    setState(() => _working = true);
    try {
      await precacheImage(
        const AssetImage('assets/splash/brand_mark.png'),
        context,
      );
      if (!mounted) return;
      _readiness.reset();
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted) return;
      await _readiness.wait();
      await WidgetsBinding.instance.endOfFrame;
      if (!mounted) return;
      final bytes = await _capture(_boundaryKey);
      if (!mounted) return;
      if (save) {
        await saveImageBytes(context, bytes, widget.fileName);
      } else {
        await shareImageBytes(context, bytes, widget.fileName);
      }
    } on _ShareImageTooLongException {
      if (mounted) {
        showGfToast(
          context,
          AppLocalizations.of(context).shareImageTooLong,
          error: true,
        );
      }
    } catch (error) {
      debugPrint('Share image capture failed (${error.runtimeType}).');
      if (mounted) {
        showGfToast(
          context,
          save
              ? AppLocalizations.of(context).imageSaveFailed
              : AppLocalizations.of(context).shareImageFailed,
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => _working = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = widget.themes[_selected];
    return PopScope(
      canPop: !_working,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 18, 12, 8),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.shareImageTitle,
                    style: GfTheme.typographyOf(context).title2,
                  ),
                ),
                IconButton(
                  tooltip: l10n.commonClose,
                  onPressed: _working ? null : () => Navigator.pop(context),
                  icon: const GfSymbol('x'),
                ),
              ],
            ),
          ),
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Center(
                child: SizedBox(
                  width: (MediaQuery.sizeOf(context).width - 32).clamp(
                    1.0,
                    shareImageLogicalWidth,
                  ),
                  child: FittedBox(
                    fit: BoxFit.contain,
                    alignment: Alignment.topCenter,
                    child: ShareImageReadiness(
                      tracker: _readiness,
                      child: RepaintBoundary(
                        key: _boundaryKey,
                        child: widget.cardBuilder(theme),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 0),
            child: Align(
              alignment: AlignmentDirectional.centerStart,
              child: Text(
                l10n.shareImageTheme,
                style: GfTheme.typographyOf(context).caption,
              ),
            ),
          ),
          // 色卡：圆形对角拼色（卡片底色 | 主题主色），选中时外圈主色描边；
          // 五个刚好铺满一行，放不下时横向滚动。
          LayoutBuilder(
            builder: (context, constraints) => SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(12, 6, 12, 2),
              child: ConstrainedBox(
                constraints: BoxConstraints(
                  minWidth: math.max(0, constraints.maxWidth - 24),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: [
                    for (int index = 0; index < widget.themes.length; index++)
                      _ThemeSwatch(
                        key: ValueKey<String>(
                          'share-theme-${widget.themes[index].id}',
                        ),
                        theme: widget.themes[index],
                        selected: _selected == index,
                        onTap: _working
                            ? null
                            : () {
                                if (_selected == index) return;
                                _readiness.reset();
                                setState(() => _selected = index);
                              },
                      ),
                  ],
                ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
            child: MediaQuery.textScalerOf(context).scale(14) > 18
                ? Column(
                    children: [
                      SizedBox(
                        width: double.infinity,
                        child: OutlinedButton.icon(
                          onPressed: _working ? null : () => _act(true),
                          icon: const GfSymbol('download'),
                          label: Text(l10n.imageSave),
                        ),
                      ),
                      SizedBox(
                        width: double.infinity,
                        child: FilledButton.icon(
                          onPressed: _working ? null : () => _act(false),
                          icon: _working
                              ? const SizedBox.square(
                                  dimension: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const GfSymbol('share-2'),
                          label: Text(
                            _working
                                ? l10n.shareImageGenerating
                                : l10n.topicShare,
                          ),
                        ),
                      ),
                    ],
                  )
                : Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: _working ? null : () => _act(true),
                          icon: const GfSymbol('download'),
                          label: Text(l10n.imageSave),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton.icon(
                          onPressed: _working ? null : () => _act(false),
                          icon: _working
                              ? const SizedBox.square(
                                  dimension: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const GfSymbol('share-2'),
                          label: Text(
                            _working
                                ? l10n.shareImageGenerating
                                : l10n.topicShare,
                          ),
                        ),
                      ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }
}

Future<Uint8List> _capture(
  GlobalKey key, {
  int maxImageBytes = _maxImageBytes,
  int maxTilePixels = _maxTilePixels,
}) async {
  if (_captureInProgress) throw StateError('capture already in progress');
  _captureInProgress = true;
  try {
    final render = key.currentContext?.findRenderObject();
    if (render is! RenderRepaintBoundary ||
        !render.attached ||
        !render.hasSize) {
      throw StateError('share card is not laid out');
    }
    final width = (render.size.width * _shareImagePixelRatio).ceil();
    final height = (render.size.height * _shareImagePixelRatio).ceil();
    if (width <= 0 || height <= 0 || width * height * 4 > maxImageBytes) {
      throw _ShareImageTooLongException();
    }
    final tiles = <_ShareImageTileData>[];
    if (height <= maxTilePixels) {
      final image = await render.toImage(pixelRatio: _shareImagePixelRatio);
      try {
        final data = await image.toByteData(format: ui.ImageByteFormat.rawRgba);
        if (data == null) throw StateError('could not encode captured image');
        tiles.add(
          _ShareImageTileData(
            image.height,
            data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes),
          ),
        );
      } finally {
        image.dispose();
      }
    } else {
      // Bounded captures are exposed only on OffsetLayer; RepaintBoundary owns it.
      // ignore: invalid_use_of_protected_member
      final layer = render.layer;
      if (layer is! OffsetLayer) {
        throw StateError('share card layer unavailable');
      }
      final logicalTileHeight = maxTilePixels / _shareImagePixelRatio;
      for (double top = 0; top < render.size.height; top += logicalTileHeight) {
        final tileHeight = (render.size.height - top)
            .clamp(0.0, logicalTileHeight)
            .toDouble();
        final image = await layer.toImage(
          Rect.fromLTWH(0, top, render.size.width, tileHeight),
          pixelRatio: _shareImagePixelRatio,
        );
        try {
          final data = await image.toByteData(
            format: ui.ImageByteFormat.rawRgba,
          );
          if (data == null) throw StateError('could not read captured pixels');
          tiles.add(
            _ShareImageTileData(
              image.height,
              data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes),
            ),
          );
        } finally {
          image.dispose();
        }
      }
    }
    final stitchedHeight = tiles.fold<int>(
      0,
      (height, tile) => height + tile.height,
    );
    if (width * stitchedHeight * 4 > maxImageBytes) {
      throw _ShareImageTooLongException();
    }
    return await compute(
      _stitchShareImage,
      _ShareImageStitchInput(width, stitchedHeight, tiles),
    );
  } finally {
    _captureInProgress = false;
  }
}

class _ShareImageTileData {
  _ShareImageTileData(this.height, Uint8List pixels)
    : pixels = kIsWeb ? pixels : null,
      transferablePixels = kIsWeb
          ? null
          : TransferableTypedData.fromList([pixels]);
  final int height;
  final Uint8List? pixels;
  final TransferableTypedData? transferablePixels;

  Uint8List materialize() =>
      pixels ?? transferablePixels!.materialize().asUint8List();
}

class _ShareImageStitchInput {
  const _ShareImageStitchInput(this.width, this.height, this.tiles);
  final int width;
  final int height;
  final List<_ShareImageTileData> tiles;
}

Uint8List _stitchShareImage(_ShareImageStitchInput input) {
  if (input.tiles.length == 1) {
    final pixels = input.tiles.single.materialize();
    final image = img.Image.fromBytes(
      width: input.width,
      height: input.height,
      bytes: pixels.buffer,
      bytesOffset: pixels.offsetInBytes,
      numChannels: 4,
      order: img.ChannelOrder.rgba,
    );
    return Uint8List.fromList(img.encodePng(image));
  }
  final stitched = img.Image(
    width: input.width,
    height: input.height,
    numChannels: 4,
  );
  var consumedHeight = 0;
  for (final tileData in input.tiles) {
    final pixels = tileData.materialize();
    final tile = img.Image.fromBytes(
      width: input.width,
      height: tileData.height,
      bytes: pixels.buffer,
      bytesOffset: pixels.offsetInBytes,
      numChannels: 4,
      order: img.ChannelOrder.rgba,
    );
    img.compositeImage(stitched, tile, dstX: 0, dstY: consumedHeight);
    consumedHeight += tileData.height;
  }
  return Uint8List.fromList(img.encodePng(stitched));
}

@visibleForTesting
Future<Uint8List> captureShareImageForTesting(
  GlobalKey key, {
  int maxImageBytes = _maxImageBytes,
  int maxTilePixels = _maxTilePixels,
}) => _capture(key, maxImageBytes: maxImageBytes, maxTilePixels: maxTilePixels);

class _ShareImageTooLongException implements Exception {}

/// 主题色卡：32dp 圆形，左上为卡片底色、右下为主题主色（硬分界，不是渐变），
/// 选中时 2dp 主色外圈 + 2dp 间隙；标签在下方。命中区 64×64。
class _ThemeSwatch extends StatelessWidget {
  const _ThemeSwatch({
    super.key,
    required this.theme,
    required this.selected,
    required this.onTap,
  });

  final ShareImageTheme theme;
  final bool selected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final Duration duration = GfMotion.duration(context, GfMotion.selection);
    return Semantics(
      container: true,
      button: true,
      selected: selected,
      enabled: onTap != null,
      label: theme.label,
      onTap: onTap,
      excludeSemantics: true,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
        child: SizedBox(
          width: 64,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                AnimatedContainer(
                  duration: duration,
                  curve: GfMotion.enterCurve,
                  width: 40,
                  height: 40,
                  padding: const EdgeInsets.all(2),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: selected ? colors.primary : Colors.transparent,
                      width: 2,
                    ),
                  ),
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      border: Border.all(color: colors.line),
                      gradient: LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [
                          theme.colors.base100,
                          theme.colors.base100,
                          theme.accent,
                          theme.accent,
                        ],
                        stops: const [0, .5, .5, 1],
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  theme.label,
                  maxLines: 1,
                  style: type.meta.copyWith(
                    fontSize: 11,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    color: selected
                        ? colors.primary
                        : colors.baseContent.withValues(alpha: 0.7),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
