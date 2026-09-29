import 'package:flutter/widgets.dart';

class ScheduleStorageStrings {
  ScheduleStorageStrings(BuildContext context)
    : locale = Localizations.localeOf(context).languageCode;
  final String locale;
  String s(String zh, String en, String de, String ja) => switch (locale) {
    'zh' => zh,
    'de' => de,
    'ja' => ja,
    _ => en,
  };
  String get found => s(
    '发现本机旧版本排课数据',
    'Legacy schedule data found',
    'Alte lokale Stundenpläne gefunden',
    '旧バージョンの履修計画があります',
  );
  String get review => s(
    '确认归属并恢复',
    'Review and restore',
    'Zuordnung prüfen und wiederherstellen',
    '所有者を確認して復元',
  );
  String get explanation => s(
    '旧数据未记录来源站点。仅在确认这些方案属于下列站点和账号后恢复。恢复会保留当前方案；重复标识或超过 10 个方案时保留原数据，请先导出。旧同步版本不会用于新站点。',
    'The old data has no source site. Restore only if these plans belong to the site and account below. Existing plans stay. Conflicting IDs or more than 10 plans prevent restoration; export first. Old sync revisions are not used on this site.',
    'Die alten Daten enthalten keine Quellseite. Nur wiederherstellen, wenn sie zur folgenden Seite und zum Konto gehören. Bestehende Pläne bleiben. Doppelte IDs oder über 10 Pläne verhindern die Wiederherstellung; zuerst exportieren. Alte Synchronisationsstände werden nicht übernommen.',
    '旧データに元のサイトの記録がありません。以下のサイトとアカウントの計画であることを確認してください。現在の計画は保持します。ID の競合や 10 件超過時は復元せず元データを保持します。先にエクスポートできます。旧同期リビジョンは引き継ぎません。',
  );
  String summary(int count) => s(
    '$count 个方案；未确认前保留在本机，不自动同步。',
    '$count plans; retained locally without automatic synchronization until confirmed.',
    '$count Pläne; bleiben bis zur Bestätigung lokal und werden nicht automatisch synchronisiert.',
    '計画 $count 件。確認前は端末に保持し、自動同期しません。',
  );
  String ownerMismatch(int owner) => s(
    '旧记录属于账号 ID $owner，请切换到该账号后恢复。',
    'The old record belongs to account ID $owner. Switch to that account to restore.',
    'Die Daten gehören zu Konto-ID $owner. Zum Wiederherstellen zu diesem Konto wechseln.',
    '旧データのアカウント ID は $owner です。そのアカウントに切り替えてください。',
  );
  String get guest => s('游客', 'Guest', 'Gast', 'ゲスト');
  String get cancel =>
      s('保留，暂不恢复', 'Keep for later', 'Für später behalten', '後で復元');
  String get restore => s('确认恢复', 'Restore', 'Wiederherstellen', '復元');
  String get restored => s(
    '排课方案已恢复',
    'Schedule plans restored',
    'Stundenpläne wiederhergestellt',
    '履修計画を復元しました',
  );
  String get export => s(
    '导出原始数据',
    'Export original data',
    'Originaldaten exportieren',
    '元データをエクスポート',
  );
  String get exportArchive => s(
    '导出已保留的旧版本原始备份',
    'Export preserved legacy backup',
    'Gesicherte alte Originaldaten exportieren',
    '保持した旧バージョンのバックアップをエクスポート',
  );
  String get exportHint => s(
    '原始数据可能包含个人课程信息，仅在你选择分享目标后发送。',
    'Original data may contain personal course information. It is shared only with a destination you choose.',
    'Originaldaten können persönliche Kursdaten enthalten. Sie werden nur an ein selbst gewähltes Ziel geteilt.',
    '元データには個人の授業情報が含まれる場合があります。選択した共有先にのみ送信します。',
  );
  String get failed => s(
    '未能完成操作，原始数据已保留。',
    'Could not complete the action. Original data is preserved.',
    'Aktion fehlgeschlagen. Die Originaldaten bleiben erhalten.',
    '操作を完了できません。元データは保持されています。',
  );
  String get storageError => s(
    '本机保存未完成，请保留此页面并重试；已保存的数据不会被覆盖。',
    'Local saving is incomplete. Keep this page open and retry; previously saved data is preserved.',
    'Lokales Speichern unvollständig. Seite geöffnet lassen und erneut versuchen; gespeicherte Daten bleiben erhalten.',
    '端末への保存が未完了です。この画面を開いたまま再試行してください。保存済みデータは保持されています。',
  );
  String get retry => s(
    '重试读取本机数据',
    'Retry local storage',
    'Lokalen Speicher erneut lesen',
    '端末データを再読み込み',
  );
}
