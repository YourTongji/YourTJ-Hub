import 'package:flutter/widgets.dart';
import '../offline/drift_cache.dart';

class StorageStrings {
  StorageStrings(BuildContext context)
    : locale = Localizations.localeOf(context).languageCode;
  final String locale;
  String s(String zh, String en, String de, String ja) => switch (locale) {
    'zh' => zh,
    'de' => de,
    'ja' => ja,
    _ => en,
  };
  String get preparing => s(
    '正在准备本机数据…',
    'Preparing local data…',
    'Lokale Daten werden vorbereitet…',
    '端末データを準備しています…',
  );
  String get resetting => s(
    '正在重置本机数据…',
    'Resetting local data…',
    'Lokale Daten werden zurückgesetzt…',
    '端末データをリセットしています…',
  );
  String get recoveryBlocked => s(
    '本机数据处理尚未完成',
    'Local data is not ready',
    'Lokale Daten sind noch nicht bereit',
    '端末データの処理が完了していません',
  );
  String get recoveryRetryHint => s(
    '请重试以完成本机存储检查和未完成的操作，之后即可继续使用 App。',
    'Retry to finish checking local storage and any unfinished operations before continuing.',
    'Erneut versuchen, um den lokalen Speicher zu prüfen und ausstehende Vorgänge abzuschließen. Danach können Sie fortfahren.',
    '端末ストレージの確認と未完了の処理を再試行してください。完了後にアプリを利用できます。',
  );
  String get cache =>
      s('可清理缓存', 'Rebuildable cache', 'Löschbarer Cache', '削除できるキャッシュ');
  String category(CacheCategory c) => switch (c) {
    CacheCategory.forum => s('论坛阅读', 'Forum reading', 'Forum-Inhalte', 'フォーラム'),
    CacheCategory.chat => s(
      '已同步聊天',
      'Synced chats',
      'Synchronisierte Chats',
      '同期済みチャット',
    ),
    CacheCategory.media => s(
      '图片与 GIF',
      'Images & GIFs',
      'Bilder & GIFs',
      '画像と GIF',
    ),
    CacheCategory.campus => s(
      '校园与桌面课表',
      'Campus & timetable widget',
      'Campus & Stundenplan-Widget',
      'キャンパスと時間割ウィジェット',
    ),
  };
  String get preserve => s(
    '清理此 App 所有站点和账号的所选本机副本，之后可联网重新获取。登录、草稿、未发送私信和排课方案都会保留。',
    'Clears selected local copies for all sites and accounts in this app; they can be fetched again online. Your login, drafts, unsent messages and schedule plans stay on this device.',
    'Löscht ausgewählte lokale Kopien aller Websites und Konten dieser App. Sie können online erneut geladen werden. Anmeldung, Entwürfe, ungesendete Nachrichten und Stundenpläne bleiben erhalten.',
    'このアプリの全サイト・アカウントの選択したコピーを削除します。オンラインで再取得できます。ログイン、下書き、未送信メッセージ、履修計画は保持されます。',
  );
  String get campusHint => s(
    '同时移除离线校园副本和桌面课表，下次需手动刷新校园数据。',
    'Also removes the offline campus copy and widget. Refresh campus data manually afterwards.',
    'Löscht auch die Offline-Campuskopie und das Widget. Campus-Daten danach manuell aktualisieren.',
    'オフラインのキャンパス情報とウィジェットも削除します。後で手動更新してください。',
  );
  String get accounting => s(
    '分类大小为估算值，可能与实际占用不同。缓存按默认 256 MiB 预算自动管理。',
    'Category sizes are estimates and may differ from actual storage used. Cache is managed automatically with a default 256 MiB budget.',
    'Kategoriegrößen sind Schätzungen und können vom tatsächlichen Speicherverbrauch abweichen. Der Cache wird automatisch mit einem Standardbudget von 256 MiB verwaltet.',
    '分類のサイズは推定値で、実際の使用量と異なる場合があります。キャッシュは標準の容量目安 256 MiB で自動管理されます。',
  );
  String get total => s(
    '本机缓存占用',
    'Cache on this device',
    'Cache auf diesem Gerät',
    'この端末のキャッシュ',
  );
  String get clear => s(
    '清理所选缓存',
    'Clear selected cache',
    'Ausgewählten Cache löschen',
    '選択したキャッシュを削除',
  );
  String get clearConfirm => s(
    '清理这些本机副本？',
    'Clear these local copies?',
    'Diese lokalen Kopien löschen?',
    'ローカルコピーを削除しますか？',
  );
  String get cleared => s(
    '所选缓存已清理',
    'Selected cache cleared',
    'Ausgewählter Cache gelöscht',
    '選択したキャッシュを削除しました',
  );
  String get failed => s(
    '部分清理未完成，已保留重试记录。请重试；重启后也会继续处理。',
    'Some cleanup is unfinished. Retry now; the app also retries after restarting.',
    'Ein Teil konnte nicht gelöscht werden. Erneut versuchen; die App versucht es auch nach einem Neustart.',
    '一部の削除が未完了です。再試行するか、アプリ再起動後に再処理します。',
  );
  String get unavailable => s(
    '暂时无法读取存储信息，请重试。',
    'Storage information is unavailable. Please retry.',
    'Speicherinformationen sind derzeit nicht verfügbar. Bitte erneut versuchen.',
    'ストレージ情報を読み取れません。再試行してください。',
  );
  String get retry => s('重试', 'Retry', 'Erneut versuchen', '再試行');
  String get localWork =>
      s('本机内容', 'Work on this device', 'Lokale Inhalte', 'この端末の作業');
  String workSummary(int drafts, int chats, int plans, int dirty) => s(
    '$drafts 份草稿 · $chats 份未发送私信 · $plans 个排课方案（$dirty 个未同步）',
    '$drafts drafts · $chats unsent messages · $plans schedule plans ($dirty unsynced)',
    '$drafts Entwürfe · $chats ungesendete Nachrichten · $plans Stundenpläne ($dirty nicht synchronisiert)',
    '下書き $drafts 件・未送信 $chats 件・履修計画 $plans 件（未同期 $dirty 件）',
  );
  String get excluded => s(
    '这些内容不参与缓存回收。请先保存到云端或导出重要作品。',
    'These are excluded from cache eviction. Save important work to the cloud or export it first.',
    'Diese Inhalte werden nicht als Cache gelöscht. Wichtige Arbeit zuerst online speichern oder exportieren.',
    'これらはキャッシュ削除の対象外です。大切な作業は先にクラウド保存またはエクスポートしてください。',
  );
  String get drafts => s('查看草稿箱', 'Open drafts', 'Entwürfe öffnen', '下書きを開く');
  String get plans =>
      s('查看排课方案', 'Open schedule plans', 'Stundenpläne öffnen', '履修計画を開く');
  String recovery(int count) => s(
    '$count 份旧排课数据待确认归属，请在排课页恢复。',
    '$count legacy schedules need an owner. Restore them from the schedule page.',
    '$count alte Stundenpläne benötigen eine Zuordnung. Auf der Stundenplanseite wiederherstellen.',
    '旧履修計画 $count 件の所有者を確認してください。履修計画ページで復元できます。',
  );
  String get reset => s(
    '重置本机数据',
    'Reset local data',
    'Lokale Daten zurücksetzen',
    '端末データをリセット',
  );
  String get resetHint => s(
    '退出登录并删除本机缓存、偏好、草稿及排课方案。云端内容和学校绑定不变。',
    'Signs out and removes local caches, preferences, drafts and schedule plans. Cloud content and school bindings stay unchanged.',
    'Meldet ab und löscht lokale Caches, Einstellungen, Entwürfe und Stundenpläne. Online-Inhalte und Schulverknüpfungen bleiben erhalten.',
    'ログアウトし、端末のキャッシュ、設定、下書き、履修計画を削除します。クラウドの内容と学校連携は変更しません。',
  );
  String get resetConfirm => s(
    '这些本机作品将永久删除，未同步内容无法从云端恢复。确认重置？',
    'Local work will be permanently deleted. Unsynced content cannot be recovered from the cloud. Reset now?',
    'Lokale Arbeit wird dauerhaft gelöscht. Nicht synchronisierte Inhalte sind nicht aus der Cloud wiederherstellbar. Zurücksetzen?',
    'ローカルの作業は完全に削除されます。未同期の内容はクラウドから復元できません。リセットしますか？',
  );
  String get resetDone => s(
    '本机数据已重置',
    'Local data reset',
    'Lokale Daten zurückgesetzt',
    '端末データをリセットしました',
  );
  String get cancel => s('取消', 'Cancel', 'Abbrechen', 'キャンセル');
  String get confirm => s('确认', 'Confirm', 'Bestätigen', '確認');
  String get calculating => s(
    '正在读取存储占用…',
    'Reading storage usage…',
    'Speicherbelegung wird gelesen…',
    'ストレージ使用量を読み込み中…',
  );
}

String storageSize(int bytes) => bytes < 1024
    ? '$bytes B'
    : bytes < 1024 * 1024
    ? '${(bytes / 1024).toStringAsFixed(1)} KiB'
    : '${(bytes / 1024 / 1024).toStringAsFixed(1)} MiB';
