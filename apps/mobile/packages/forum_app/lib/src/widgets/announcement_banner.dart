import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_widget_from_html_core/flutter_widget_from_html_core.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';

import '../providers.dart';

/// Published Web announcements, with natural height for HTML and large text.
class AnnouncementBanner extends ConsumerStatefulWidget {
  const AnnouncementBanner({super.key, required this.announcement});
  final AnnouncementPayload announcement;

  @override
  ConsumerState<AnnouncementBanner> createState() => _AnnouncementBannerState();
}

class _AnnouncementBannerState extends ConsumerState<AnnouncementBanner> {
  Timer? _timer;
  int _current = 0;

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
        // Respect screen readers, reduced motion and manual selection.
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

  @override
  Widget build(BuildContext context) {
    final items = _items;
    if (items.isEmpty) return const SizedBox.shrink();
    final item = items[_current];
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: colors.primary.withValues(alpha: 0.05),
        border: Border(
          bottom: BorderSide(color: colors.primary.withValues(alpha: 0.15)),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 3),
            child: ExcludeSemantics(
              child: GfSymbol('bell', size: 18, color: colors.primary),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (item.title.trim().isNotEmpty)
                  Text(item.title.trim(), style: type.bodyStrong),
                if (item.title.trim().isNotEmpty && item.html.trim().isNotEmpty)
                  const SizedBox(height: 4),
                if (item.html.trim().isNotEmpty)
                  HtmlWidget(
                    item.html,
                    baseUrl: Uri.parse(ref.read(apiClientProvider).baseUrl),
                    textStyle: type.body,
                    customStylesBuilder: (element) =>
                        element.localName == 'p' ? {'margin': '0 0 4px'} : null,
                    onTapUrl: _open,
                  ),
                if (items.length > 1)
                  Wrap(
                    children: [
                      for (var i = 0; i < items.length; i++)
                        Semantics(
                          selected: i == _current,
                          child: TextButton(
                            onPressed: () {
                              setState(() => _current = i);
                              _restart();
                            },
                            child: Text('${i + 1} / ${items.length}'),
                          ),
                        ),
                    ],
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
