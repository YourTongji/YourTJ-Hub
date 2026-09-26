import 'package:flutter/widgets.dart';
import 'package:core/core.dart';

/// Localized copy shared by all sticker entry points. Kept together with the
/// feature until ARB ownership is consolidated; no server error text is shown.
class StickerStrings {
  StickerStrings(BuildContext context)
    : _locale = Localizations.localeOf(context).languageCode;
  final String _locale;
  String _s(String zh, String en, String de, String ja) => switch (_locale) {
    'zh' => zh,
    'de' => de,
    'ja' => ja,
    _ => en,
  };
  String get custom =>
      _s('自定义表情', 'Custom sticker', 'Eigener Sticker', 'カスタムステッカー');
  String displayLabel(StickerItemPayload item) =>
      !item.isOfficial &&
          (item.displayName.isEmpty || item.displayName == item.name)
      ? custom
      : item.label;
  String get licenses =>
      _s('来源与许可', 'Sources & licenses', 'Quellen & Lizenzen', '出典とライセンス');
  String get title =>
      _s('表情库', 'Sticker library', 'Stickerbibliothek', 'ステッカー');
  String get recent => _s('最近', 'Recent', 'Zuletzt', '最近');
  String get mine => _s('我的', 'Mine', 'Meine', 'マイ');
  String get official => _s('官方', 'Official', 'Offiziell', '公式');
  String get manage => _s('管理', 'Manage', 'Verwalten', '管理');
  String get add => _s('添加表情', 'Add sticker', 'Sticker hinzufügen', '追加');
  String get collect =>
      _s('收藏到我的表情', 'Save to my stickers', 'Zu meinen Stickern', 'マイステッカーに保存');
  String get saved =>
      _s('已添加到我的表情', 'Saved to your stickers', 'Sticker gespeichert', '保存しました');
  String get remove => _s('移除', 'Remove', 'Entfernen', '削除');
  String get rename => _s('重命名', 'Rename', 'Umbenennen', '名前を変更');
  String get name => _s('表情名称', 'Sticker name', 'Stickername', '名前');
  String get select => _s('选择', 'Select', 'Auswählen', '選択');
  String get done => _s('完成', 'Done', 'Fertig', '完了');
  String get retry => _s('重试', 'Retry', 'Erneut versuchen', '再試行');
  String get search => _s('搜索表情', 'Search stickers', 'Sticker suchen', '検索');
  String get all => _s('全部', 'All', 'Alle', 'すべて');
  String get emptyRecent => _s(
    '用过的表情会出现在这里',
    'Stickers you use appear here',
    'Verwendete Sticker erscheinen hier',
    '使ったステッカーがここに表示されます',
  );
  String get emptyMine => _s(
    '添加自己的表情，或长按别人分享的表情收藏',
    'Upload a sticker or hold a shared sticker to save it',
    'Sticker hochladen oder geteilte Sticker gedrückt halten',
    'アップロード、または共有ステッカーを長押しして保存',
  );
  String get emptyOfficial => _s(
    '暂无官方表情',
    'No official stickers yet',
    'Noch keine offiziellen Sticker',
    '公式ステッカーはまだありません',
  );
  String get noResults => _s(
    '没有找到匹配的表情',
    'No matching stickers',
    'Keine passenden Sticker',
    '見つかりませんでした',
  );
  String get failed => _s(
    '表情加载失败，请重试',
    'Could not load stickers. Try again.',
    'Sticker konnten nicht geladen werden.',
    '読み込めませんでした。再試行してください',
  );
  String get operationFailed => _s(
    '操作未完成，请重试',
    'Could not complete this action. Try again.',
    'Aktion fehlgeschlagen. Bitte erneut versuchen.',
    '操作できませんでした。再試行してください',
  );
  String get unsupported => _s(
    '当前站点尚不支持个人表情库，仍可使用官方表情',
    'This site does not support personal stickers yet. Official stickers are available.',
    'Diese Website unterstützt noch keine persönlichen Sticker. Offizielle Sticker sind verfügbar.',
    'このサイトはマイステッカーに未対応です。公式ステッカーは利用できます',
  );
  String get unavailable => _s(
    '表情暂不可用',
    'Sticker unavailable',
    'Sticker nicht verfügbar',
    'ステッカーを表示できません',
  );
  String get uploadHint => _s(
    '支持图片和 GIF，每张最多 4 MB。分享后其他人可以收藏。',
    'Images and GIFs up to 4 MB. People you share with can save them.',
    'Bilder und GIFs bis 4 MB. Geteilte Sticker können andere speichern.',
    '画像・GIF、最大 4 MB。共有した相手も保存できます',
  );
  String get tooLarge => _s(
    '请选择不超过 4 MB 的图片或 GIF',
    'Choose an image or GIF up to 4 MB',
    'Bild oder GIF bis 4 MB auswählen',
    '4 MB 以下の画像・GIFを選んでください',
  );
  String get reorderHint => _s(
    '拖动右侧手柄调整顺序；移除不影响已发送的表情',
    'Drag handles to reorder. Removing keeps sent stickers available.',
    'Mit Griffen sortieren. Gesendete Sticker bleiben erhalten.',
    'ハンドルをドラッグして並べ替え。削除しても送信済みは残ります',
  );
  String get limit => _s(
    '我的表情库已达上限（200 个）',
    'Your library is full (200 stickers)',
    'Bibliothek voll (200 Sticker)',
    '上限の 200 個に達しました',
  );
  String get uploadQuota => _s(
    '已达到个人表情上传数量上限（1000 张）',
    'Personal upload limit reached (1,000 stickers)',
    'Upload-Limit erreicht (1.000 Sticker)',
    'アップロード上限（1,000 個）に達しました',
  );
  String failure(Object? error, {bool loading = false}) {
    if (error is ApiException) {
      if (error.statusCode == 404 || error.statusCode == 405) {
        return unsupported;
      }
      switch (error.messageCode) {
        case 'sticker.libraryFull':
          return limit;
        case 'sticker.uploadQuota':
          return uploadQuota;
        case 'sticker.imageRequired':
          return tooLarge;
        case 'sticker.unavailable':
          return unavailable;
      }
    }
    return loading ? failed : operationFailed;
  }
}
