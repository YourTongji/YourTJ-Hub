import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../l10n/app_localizations.dart';
import '../../navigation/tab_scroll_registry.dart';
import '../../reading_preferences.dart';
import '../../widgets/markdown_view.dart';

/// "100%" reads as the adapted default rather than a number to compare.
String textSizeLabel(AppLocalizations l10n, double scale) {
  final int percent = (scale * 100).round();
  return percent == 100 ? l10n.textSizeDefault : l10n.textSizePercent(percent);
}

enum _TextSizeAxis { app, reading }

/// Text size settings with a live preview (adjust and see at once).
///
/// Two layered axes: the app size scales every text through the root
/// [GfAppTextScaler], and the reading size adjusts rich content bodies on top
/// of it. The preview is a small app screen built from the real components —
/// tab bar, feed row, post body, button and bottom navigation — and shows
/// what the slider being dragged affects: the app axis outlines the whole
/// screen, the reading axis outlines the post body and dims everything else.
/// The control panel keeps the default app size so the slider never moves
/// away from the finger.
class TextSizeSettingsPage extends ConsumerStatefulWidget {
  const TextSizeSettingsPage({super.key});

  @override
  ConsumerState<TextSizeSettingsPage> createState() =>
      _TextSizeSettingsPageState();
}

class _TextSizeSettingsPageState extends ConsumerState<TextSizeSettingsPage> {
  final GlobalKey _bodyKey = GlobalKey();
  _TextSizeAxis? _active;
  Timer? _release;

  void _activate(_TextSizeAxis axis) {
    _release?.cancel();
    setState(() => _active = axis);
    final BuildContext? body = _bodyKey.currentContext;
    if (axis == _TextSizeAxis.reading && body != null) {
      // Minimal scroll, so the pinned chrome around the body stays in view.
      Scrollable.ensureVisible(
        body,
        alignmentPolicy: ScrollPositionAlignmentPolicy.keepVisibleAtEnd,
        duration: GfMotion.duration(context, GfMotion.layout),
        curve: GfMotion.layoutCurve,
      );
    }
  }

  /// Keeps the highlight briefly after release so the result stays readable.
  void _deactivate() {
    _release?.cancel();
    _release = Timer(const Duration(milliseconds: 900), () {
      if (mounted) setState(() => _active = null);
    });
  }

  @override
  void dispose() {
    _release?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final double appScale = ref.watch(appFontScaleProvider);
    final double readingScale = ref.watch(contentFontScaleProvider);
    return Scaffold(
      appBar: GfAppBar(title: Text(l10n.textSizeTitle)),
      body: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) => Column(
          children: <Widget>[
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
                child: _TextSizePreview(active: _active, bodyKey: _bodyKey),
              ),
            ),
            // Large system text can make the panel tall on short screens; it
            // scrolls inside its own cap instead of overflowing.
            ConstrainedBox(
              constraints: BoxConstraints(
                maxHeight: constraints.maxHeight * .6,
              ),
              child: MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(textScaler: GfAppTextScaler.defaultOf(context)),
                child: _TextSizeControls(
                  appScale: appScale,
                  readingScale: readingScale,
                  onAxisStart: _activate,
                  onAxisEnd: _deactivate,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A small app screen from the real components. Everything follows the app
/// size; only the Markdown body adds the reading size on top.
///
/// The tab bar and bottom navigation are pinned and only the content between
/// them scrolls, so interface text stays in view while a slider moves. Short
/// windows (large system text on small phones) scroll the whole screen
/// instead of squeezing that middle region away.
class _TextSizePreview extends StatelessWidget {
  const _TextSizePreview({required this.active, required this.bodyKey});

  final _TextSizeAxis? active;
  final GlobalKey bodyKey;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final GfRadii radii = GfTheme.radiiOf(context);
    final GfBorders borders = GfTheme.bordersOf(context);
    final Duration duration = GfMotion.duration(context, GfMotion.layout);
    final TextStyle meta = type.caption.copyWith(color: colors.iconMuted);
    final bool focusApp = active == _TextSizeAxis.app;
    final bool focusReading = active == _TextSizeAxis.reading;

    // Chrome outside the post body: dimmed while only the body changes.
    Widget chrome(Widget child) => AnimatedOpacity(
      opacity: focusReading ? .35 : 1,
      duration: duration,
      child: child,
    );

    Widget action(String symbol, String count) => Padding(
      padding: const EdgeInsetsDirectional.only(end: 20),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          GfSymbol(symbol, size: 18, color: colors.iconMuted),
          const SizedBox(width: 6),
          Text(count, style: meta),
        ],
      ),
    );

    final Widget post = Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          // Constant padding and border width, so highlighting never moves
          // the layout.
          AnimatedContainer(
            key: bodyKey,
            duration: duration,
            padding: const EdgeInsets.fromLTRB(10, 4, 10, 4),
            decoration: BoxDecoration(
              color: focusReading
                  ? colors.primary.withValues(alpha: .06)
                  : colors.primary.withValues(alpha: 0),
              borderRadius: BorderRadius.circular(radii.field),
              border: Border.all(
                color: focusReading
                    ? colors.primary
                    : colors.primary.withValues(alpha: 0),
                width: 1.5,
              ),
            ),
            child: GfMarkdownView(
              key: const ValueKey('text-size-preview-body'),
              data: l10n.textSizePreviewBody,
            ),
          ),
          const SizedBox(height: 8),
          chrome(
            // The button moves to its own line when large text leaves no
            // room beside the counts.
            Wrap(
              runSpacing: 8,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: <Widget>[
                action('heart', '128'),
                action('message-circle', '32'),
                IgnorePointer(
                  child: ExcludeSemantics(
                    child: GfButton(
                      label: l10n.topicReply,
                      size: GfButtonSize.small,
                      onPressed: () {},
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );

    final Widget tabs = chrome(
      IgnorePointer(
        child: ExcludeSemantics(
          child: GfTabBar(
            key: const ValueKey('text-size-preview-tabs'),
            tabs: <GfTab>[
              GfTab(label: l10n.sortLatest, value: 0),
              GfTab(label: l10n.sortFollowing, value: 1),
              GfTab(label: l10n.sortHot, value: 2),
            ],
            selected: 0,
            onSelected: (_) {},
          ),
        ),
      ),
    );
    final Widget row = chrome(
      GfTopicRow(
        key: const ValueKey('text-size-preview-row'),
        title: l10n.textSizePreviewTitle,
        description: l10n.textSizePreviewExcerpt,
        categories: <GfTopicCategory>[
          GfTopicCategory(
            name: l10n.textSizePreviewCategory,
            color: colors.primary,
          ),
        ],
        participantAvatarUrls: const <String>[],
        activityText: l10n.textSizePreviewTime,
        replyCount: 32,
        pinnedLabel: '',
        home: true,
      ),
    );
    final Widget navigation = chrome(
      IgnorePointer(
        child: ExcludeSemantics(
          // The real bar pads for the device inset; the preview sits
          // mid-page.
          child: MediaQuery.removePadding(
            context: context,
            removeBottom: true,
            child: GfBottomNavigation(
              key: const ValueKey('text-size-preview-navigation'),
              currentIndex: 0,
              onSelected: (_) {},
              showLabels: shellNavigationShowsLabels,
              items: <GfBottomNavigationItem>[
                for (final GfShellDestination destination
                    in GfShellDestination.values)
                  GfBottomNavigationItem(
                    symbol: destination.symbol,
                    selectedSymbol: '${destination.symbol}-filled',
                    label: destination.label(l10n),
                  ),
              ],
            ),
          ),
        ),
      ),
    );

    Widget frame(Widget child) => AnimatedContainer(
      key: const ValueKey('text-size-preview'),
      duration: duration,
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: colors.base100,
        borderRadius: BorderRadius.circular(radii.box),
        border: Border.all(
          color: focusApp ? colors.primary : colors.line,
          width: focusApp ? 1.5 : borders.width,
        ),
      ),
      child: child,
    );

    return Semantics(
      container: true,
      label: l10n.textSizePreview,
      child: LayoutBuilder(
        builder: (BuildContext context, BoxConstraints constraints) {
          if (constraints.maxHeight < 360) {
            return Scrollbar(
              child: ListView(
                key: const ValueKey('text-size-preview-scroll'),
                padding: EdgeInsets.zero,
                children: <Widget>[
                  frame(
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: <Widget>[tabs, row, post, navigation],
                    ),
                  ),
                ],
              ),
            );
          }
          return frame(
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: <Widget>[
                tabs,
                Expanded(
                  // Soft edges show that enlarged content continues under
                  // the pinned bars instead of colliding with them.
                  child: ShaderMask(
                    blendMode: BlendMode.dstIn,
                    shaderCallback: (Rect bounds) => const LinearGradient(
                      begin: Alignment.topCenter,
                      end: Alignment.bottomCenter,
                      colors: <Color>[
                        Color(0x00000000),
                        Color(0xFF000000),
                        Color(0xFF000000),
                        Color(0x00000000),
                      ],
                      stops: <double>[0, .04, .92, 1],
                    ).createShader(bounds),
                    child: Scrollbar(
                      child: ListView(
                        key: const ValueKey('text-size-preview-scroll'),
                        padding: const EdgeInsets.only(bottom: 16),
                        children: <Widget>[row, post],
                      ),
                    ),
                  ),
                ),
                navigation,
              ],
            ),
          );
        },
      ),
    );
  }
}

class _TextSizeControls extends ConsumerWidget {
  const _TextSizeControls({
    required this.appScale,
    required this.readingScale,
    required this.onAxisStart,
    required this.onAxisEnd,
  });

  final double appScale;
  final double readingScale;
  final ValueChanged<_TextSizeAxis> onAxisStart;
  final VoidCallback onAxisEnd;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final bool isDefault =
        (appScale * 100).round() == 100 && (readingScale * 100).round() == 100;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: colors.base100,
        border: Border(top: BorderSide(color: colors.line)),
      ),
      child: SafeArea(
        top: false,
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              _TextSizeSlider(
                axis: _TextSizeAxis.app,
                title: l10n.textSizeGlobal,
                description: l10n.textSizeGlobalDesc,
                value: appScale,
                min: GfAppTextScaler.minUserScale,
                max: GfAppTextScaler.maxUserScale,
                notifier: ref.read(appFontScaleProvider.notifier),
                onStart: onAxisStart,
                onEnd: onAxisEnd,
              ),
              const SizedBox(height: 8),
              _TextSizeSlider(
                axis: _TextSizeAxis.reading,
                title: l10n.textSizeReading,
                description: l10n.textSizeReadingDesc,
                value: readingScale,
                min: GfRichContentTypography.minUserScale,
                max: GfRichContentTypography.maxUserScale,
                notifier: ref.read(contentFontScaleProvider.notifier),
                onStart: onAxisStart,
                onEnd: onAxisEnd,
              ),
              // Last in the panel so it never shifts the sliders above it.
              Align(
                alignment: AlignmentDirectional.centerEnd,
                child: TextButton.icon(
                  key: const ValueKey('text-size-reset'),
                  onPressed: isDefault
                      ? null
                      : () {
                          ref
                              .read(appFontScaleProvider.notifier)
                              .resetToDefault();
                          ref
                              .read(contentFontScaleProvider.notifier)
                              .resetToDefault();
                        },
                  icon: const GfSymbol('undo-2', size: 18),
                  label: Text(l10n.textSizeReset),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// One labelled `Aa — Aa` slider in 10% steps.
class _TextSizeSlider extends StatelessWidget {
  const _TextSizeSlider({
    required this.axis,
    required this.title,
    required this.description,
    required this.value,
    required this.min,
    required this.max,
    required this.notifier,
    required this.onStart,
    required this.onEnd,
  });

  final _TextSizeAxis axis;
  final String title;
  final String description;
  final double value;
  final double min;
  final double max;
  final FontScaleNotifier notifier;
  final ValueChanged<_TextSizeAxis> onStart;
  final VoidCallback onEnd;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final GfColors colors = GfTheme.colorsOf(context);
    final GfTypography type = GfTheme.typographyOf(context);
    final String label = textSizeLabel(l10n, value);
    final TextStyle muted = type.caption.copyWith(color: colors.iconMuted);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: <Widget>[
            Expanded(child: Text(title, style: type.bodyStrong)),
            Text(
              label,
              key: ValueKey('text-size-${axis.name}-label'),
              style: type.small.copyWith(
                color: colors.primary,
                fontWeight: FontWeight.w600,
                fontFeatures: const <FontFeature>[FontFeature.tabularFigures()],
              ),
            ),
          ],
        ),
        const SizedBox(height: 2),
        Text(description, style: muted),
        Row(
          children: <Widget>[
            ExcludeSemantics(
              child: Text('Aa', style: muted.copyWith(fontSize: 13)),
            ),
            Expanded(
              child: Semantics(
                label: title,
                child: Slider(
                  key: ValueKey('text-size-${axis.name}'),
                  value: value.clamp(min, max),
                  min: min,
                  max: max,
                  divisions: ((max - min) * 10).round(),
                  semanticFormatterCallback: (double v) =>
                      textSizeLabel(l10n, v),
                  onChangeStart: (double _) => onStart(axis),
                  // Divisions yield values like 1.0000000002; keep whole
                  // percents so the stored value matches the label.
                  onChanged: (double v) =>
                      notifier.setScale((v * 100).round() / 100),
                  onChangeEnd: (double _) {
                    notifier.persistScale();
                    onEnd();
                  },
                ),
              ),
            ),
            ExcludeSemantics(
              child: Text(
                'Aa',
                style: type.title2.copyWith(color: colors.iconMuted),
              ),
            ),
          ],
        ),
      ],
    );
  }
}
