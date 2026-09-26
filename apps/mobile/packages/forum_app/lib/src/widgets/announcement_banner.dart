import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../l10n/app_localizations.dart';
import '../providers.dart';

/// 首页公告栏：轻量信息条，保留展开、折叠与多公告导航。
class AnnouncementBanner extends ConsumerStatefulWidget {
  const AnnouncementBanner({
    super.key,
    required this.announcement,
    this.collapsed,
    this.onCollapsedChanged,
  });

  final AnnouncementPayload announcement;

  /// Optional shared state supplied by the host feed. When omitted, the
  /// banner keeps its own state for standalone uses and tests.
  final bool? collapsed;
  final ValueChanged<bool>? onCollapsedChanged;

  @override
  ConsumerState<AnnouncementBanner> createState() => _AnnouncementBannerState();
}

class _AnnouncementBannerState extends ConsumerState<AnnouncementBanner>
    with AutomaticKeepAliveClientMixin<AnnouncementBanner> {
  Timer? _timer;
  int _current = 0;
  bool _isCollapsed = false;

  bool get _collapsed => widget.collapsed ?? _isCollapsed;

  void _setCollapsed(bool value) {
    if (widget.collapsed != null) {
      widget.onCollapsedChanged?.call(value);
      return;
    }
    setState(() => _isCollapsed = value);
  }

  @override
  bool get wantKeepAlive => true;

  List<AnnouncementItemPayload> get _items {
    if (!widget.announcement.enabled) return const [];
    final items =
        (widget.announcement.items ?? const <AnnouncementItemPayload>[])
            .where(
              (item) =>
                  item.title.trim().isNotEmpty || item.html.trim().isNotEmpty,
            )
            .toList();
    if (items.isEmpty && widget.announcement.html.trim().isNotEmpty) {
      items.add(
        AnnouncementItemPayload(
          id: 'legacy',
          title: '',
          html: widget.announcement.html,
        ),
      );
    }
    return items;
  }

  @override
  void initState() {
    super.initState();
    _restart();
  }

  void _restart() {
    _timer?.cancel();
    if (_items.length > 1) {
      _timer = Timer.periodic(const Duration(seconds: 5), (_) {
        if (!mounted ||
            MediaQuery.accessibleNavigationOf(context) ||
            MediaQuery.disableAnimationsOf(context)) {
          return;
        }
        setState(() => _current = (_current + 1) % _items.length);
      });
    }
  }

  @override
  void didUpdateWidget(covariant AnnouncementBanner oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.announcement != widget.announcement) {
      _current = 0;
      _restart();
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<bool> _open(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null || !{'https', 'http'}.contains(uri.scheme)) return true;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      // A missing system browser must not break the feed.
    }
    return true;
  }

  String _snippetFor(AnnouncementItemPayload item) {
    final title = item.title.trim();
    final bodyText = _stripHtml(item.html);
    if (title.isNotEmpty && bodyText.isNotEmpty) {
      return '$title：$bodyText';
    }
    if (title.isNotEmpty) return title;
    return bodyText;
  }

  static String _stripHtml(String html) {
    if (html.isEmpty) return '';
    return html
        .replaceAll(RegExp(r'<[^>]*>'), ' ')
        .replaceAll(RegExp(r'\s+'), ' ')
        .trim();
  }

  @override
  Widget build(BuildContext context) {
    // The banner lives inside a lazy ListView. Keep its interaction state
    // alive while it is scrolled out of the viewport.
    super.build(context);
    final items = _items;
    if (items.isEmpty) return const SizedBox.shrink();
    if (_current >= items.length) _current = 0;
    final item = items[_current];
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final l10n = AppLocalizations.of(context);

    return Container(
      margin: const EdgeInsets.fromLTRB(16, 6, 16, 6),
      decoration: BoxDecoration(
        color: colors.base200,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colors.line),
      ),
      child: GestureDetector(
        onHorizontalDragEnd: (details) {
          if (items.length <= 1) return;
          final velocity = details.primaryVelocity ?? 0;
          if (velocity < -120) {
            setState(() => _current = (_current + 1) % items.length);
            _restart();
          } else if (velocity > 120) {
            setState(
              () => _current = (_current - 1 + items.length) % items.length,
            );
            _restart();
          }
        },
        child: AnimatedSize(
          duration: const Duration(milliseconds: 220),
          curve: Curves.easeInOutCubic,
          alignment: Alignment.topCenter,
          child: _collapsed
              ? _buildCollapsed(context, item, colors, l10n)
              : _buildExpanded(context, item, items, colors, type, l10n),
        ),
      ),
    );
  }

  Widget _buildCollapsed(
    BuildContext context,
    AnnouncementItemPayload item,
    GfColors colors,
    AppLocalizations l10n,
  ) {
    final snippet = _snippetFor(item);
    return InkWell(
      key: const Key('announcement-collapsed-tap'),
      onTap: () => _setCollapsed(false),
      borderRadius: BorderRadius.circular(12),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
        child: Row(
          children: [
            Container(
              width: 22,
              height: 22,
              decoration: BoxDecoration(
                color: colors.primary.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(6),
              ),
              alignment: Alignment.center,
              child: ExcludeSemantics(
                child: GfSymbol('bell', size: 12, color: colors.primary),
              ),
            ),
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1.5),
              decoration: BoxDecoration(
                color: colors.primary.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(
                l10n.announcementLabel,
                style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  color: colors.primary,
                  letterSpacing: 0.2,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                snippet,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: colors.baseContent.withValues(alpha: 0.82),
                ),
              ),
            ),
            const SizedBox(width: 6),
            Semantics(
              label: l10n.announcementExpand,
              button: true,
              child: Padding(
                padding: const EdgeInsets.all(2),
                child: GfSymbol(
                  'chevron-down',
                  size: 13,
                  color: colors.baseContent.withValues(alpha: 0.45),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildExpanded(
    BuildContext context,
    AnnouncementItemPayload item,
    List<AnnouncementItemPayload> items,
    GfColors colors,
    GfTypography type,
    AppLocalizations l10n,
  ) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(13, 11, 13, 12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              color: colors.primary.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(8),
            ),
            alignment: Alignment.center,
            child: ExcludeSemantics(
              child: GfSymbol('bell', size: 14, color: colors.primary),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 6,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: colors.primary.withValues(alpha: 0.1),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        l10n.announcementLabel,
                        style: TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.w700,
                          color: colors.primary,
                          letterSpacing: 0.2,
                        ),
                      ),
                    ),
                    if (item.title.trim().isNotEmpty) ...[
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          item.title.trim(),
                          style: type.bodyStrong.copyWith(
                            fontSize: 13,
                            fontWeight: FontWeight.w700,
                            letterSpacing: -0.1,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ] else
                      const Spacer(),
                    Semantics(
                      button: true,
                      label: l10n.announcementCollapse,
                      child: InkWell(
                        key: const Key('announcement-collapse-btn'),
                        onTap: () => _setCollapsed(true),
                        borderRadius: BorderRadius.circular(6),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 3,
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(
                                l10n.announcementCollapseAction,
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w500,
                                  color: colors.baseContent.withValues(
                                    alpha: 0.5,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 2),
                              RotatedBox(
                                quarterTurns: 2,
                                child: GfSymbol(
                                  'chevron-down',
                                  size: 11,
                                  color: colors.baseContent.withValues(
                                    alpha: 0.5,
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 6),
                if (item.html.trim().isNotEmpty)
                  HtmlWidget(
                    item.html,
                    baseUrl: Uri.parse(ref.read(apiClientProvider).baseUrl),
                    textStyle: type.body.copyWith(
                      fontSize: 12.5,
                      height: 1.45,
                      color: colors.baseContent.withValues(alpha: 0.85),
                    ),
                    customStylesBuilder: (element) =>
                        element.localName == 'p' ? {'margin': '0 0 4px'} : null,
                    onTapUrl: _open,
                  ),
                if (items.length > 1) ...[
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      for (var i = 0; i < items.length; i++)
                        Semantics(
                          selected: i == _current,
                          label: l10n.announcementItem(i + 1),
                          child: GestureDetector(
                            key: Key('announcement-dot-$i'),
                            onTap: () {
                              setState(() => _current = i);
                              _restart();
                            },
                            behavior: HitTestBehavior.opaque,
                            child: Padding(
                              padding: const EdgeInsets.only(
                                right: 6,
                                top: 3,
                                bottom: 3,
                              ),
                              child: AnimatedContainer(
                                duration: const Duration(milliseconds: 250),
                                curve: Curves.easeInOutCubic,
                                width: i == _current ? 16.0 : 5.0,
                                height: 4.5,
                                decoration: BoxDecoration(
                                  color: i == _current
                                      ? colors.primary
                                      : colors.baseContent.withValues(
                                          alpha: 0.22,
                                        ),
                                  borderRadius: BorderRadius.circular(2.5),
                                ),
                              ),
                            ),
                          ),
                        ),
                      const Spacer(),
                      InkWell(
                        key: const Key('announcement-prev-btn'),
                        onTap: () {
                          setState(
                            () => _current =
                                (_current - 1 + items.length) % items.length,
                          );
                          _restart();
                        },
                        borderRadius: BorderRadius.circular(4),
                        child: Padding(
                          padding: const EdgeInsets.all(3),
                          child: GfSymbol(
                            'chevron-left',
                            size: 13,
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ),
                      const SizedBox(width: 4),
                      InkWell(
                        key: const Key('announcement-next-btn'),
                        onTap: () {
                          setState(
                            () => _current = (_current + 1) % items.length,
                          );
                          _restart();
                        },
                        borderRadius: BorderRadius.circular(4),
                        child: Padding(
                          padding: const EdgeInsets.all(3),
                          child: GfSymbol(
                            'chevron-right',
                            size: 13,
                            color: colors.baseContent.withValues(alpha: 0.5),
                          ),
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}
