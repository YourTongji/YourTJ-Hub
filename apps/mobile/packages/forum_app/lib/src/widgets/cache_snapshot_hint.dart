import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../l10n/app_localizations.dart';

/// A device snapshot never implies that the server is reachable or permits edits.
class CacheSnapshotHint extends StatelessWidget {
  const CacheSnapshotHint({
    super.key,
    this.savedAt,
    this.refreshing = false,
    this.cleared = false,
    this.onRetry,
  });

  final DateTime? savedAt;
  final bool refreshing;
  final bool cleared;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final language = Localizations.localeOf(context).languageCode;
    final labels = switch (language) {
      'en' => (
        'Local copy',
        'Refreshing…',
        'Could not refresh · Read only',
        'Cache cleared. Reload to fetch content.',
        'Reload',
      ),
      'ja' => (
        '端末内のコピー',
        '更新中…',
        '更新できませんでした・閲覧のみ',
        'キャッシュを削除しました。再取得してください。',
        '再取得',
      ),
      'de' => (
        'Lokale Kopie',
        'Wird aktualisiert…',
        'Aktualisierung fehlgeschlagen · Nur lesen',
        'Cache geleert. Inhalte erneut laden.',
        'Neu laden',
      ),
      _ => ('本机副本', '正在刷新…', '暂时无法刷新 · 仅供阅读', '缓存已清除，请重新获取内容。', '重新获取'),
    };
    final at = savedAt?.toLocal();
    final time = at == null
        ? ''
        : ' · ${MaterialLocalizations.of(context).formatCompactDate(at)} ${MaterialLocalizations.of(context).formatTimeOfDay(TimeOfDay.fromDateTime(at))}';
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      liveRegion: true,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(
          children: [
            Expanded(
              child: Text(
                cleared
                    ? labels.$4
                    : '${labels.$1}$time\n${refreshing ? labels.$2 : labels.$3}',
                style: TextStyle(fontSize: 12, color: colors.iconMuted),
              ),
            ),
            if (!refreshing && onRetry != null)
              TextButton(
                onPressed: onRetry,
                child: Text(
                  cleared
                      ? labels.$5
                      : AppLocalizations.of(context).commonRetry,
                ),
              ),
          ],
        ),
      ),
    );
  }
}
