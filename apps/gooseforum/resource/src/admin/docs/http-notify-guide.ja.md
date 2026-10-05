## HTTP 通知の使い方

HTTP 通知は、選択したサイト内イベントが発生したときに、コールバック URL へ非同期で POST リクエストを送信します。この機能はベストエフォートです。送信に失敗しても、投稿、返信、登録、通報の処理には影響しません。

### 対応イベント

- `topic.published`
- `topic.updated`
- `comment.created`
- `user.signup`
- `moderation.report.created`

審査イベント（issue #1049、モデレーターの対応事項）：

- `moderation.review.topic.requested`：トピックが手動審査キューに入った（禁止語に一致、または AI 審査が判断できなかった・失敗した場合。自動チェック中は通知しません）
- `moderation.review.post.requested`：返信が手動審査キューに入った
- `moderation.report.topic.created`：トピックが通報された
- `moderation.report.post.created`：返信が通報された
- `moderation.report.chat_message.created`：チャットメッセージが通報された
- `moderation.report.course_review.created`：授業レビューが通報された

`moderation.report.created` は「すべての通報」をまとめた集約イベントとして残り、ペイロードは変わりません。同じ URL が集約イベントと個別の通報イベントの両方を購読している場合、各通報は個別イベントで 1 回だけ配信されます。先に公開して後から審査する AI 自動チェック（チェック中）は手動審査ではないため通知されず、AI 判定が人による確認を求めた時点で通知されます。授業レビューの審査待ち状態はまだ実装されていません（計画中）。現在は授業レビューの通報のみ通知されます。

### リクエスト形式

リクエスト本文は JSON です。Header にはイベント名、配信 ID、タイムスタンプ、署名が含まれます。

```http
POST /your-webhook HTTP/1.1
Content-Type: application/json
X-Goose-Event: topic.published
X-Goose-Delivery: 7c9f0b0fd4e2a111
X-Goose-Timestamp: 1710000000
X-Goose-Signature: sha256=...
```

```json
{
  "event": "topic.published",
  "timestamp": 1710000000,
  "data": {
    "baseUri": "http://localhost:5234",
    "topic": {
      "id": 123,
      "title": "Hello GooseForum",
      "url": "/p/post/123",
      "description": "Topic summary",
      "firstImageUrl": "",
      "userId": 1,
      "user": {
        "id": 1,
        "username": "alice",
        "nickname": "Alice",
        "displayName": "Alice",
        "avatarUrl": "/static/pic/1.webp",
        "url": "/u/1"
      },
      "categoryIds": [2],
      "categories": [
        { "id": 2, "name": "Announcements", "slug": "announcements" }
      ]
    },
    "user": {
      "id": 1,
      "username": "alice",
      "nickname": "Alice",
      "displayName": "Alice",
      "avatarUrl": "/static/pic/1.webp",
      "url": "/u/1"
    }
  }
}
```

`baseUri` は `topic.url`、`user.url`、`post.url` などのサイト内パスを絶対 URL にするために使えます。新しい連携では、重複するトップレベルのミラーフィールドではなく、`topic`、`user`、`post`、`reporter` などの構造化オブジェクトを優先して使ってください。コメントイベントには内容プレビュー、投稿者情報、post URL も含まれます。post への通報イベントには、通報対象の post 情報も含まれます。

### 署名検証

`X-Goose-Signature` は Secret、タイムスタンプ、元のリクエスト本文から計算されます。

```text
sha256 = HMAC_SHA256(secret, timestamp + "." + rawBody)
```

検証には元の body を使ってください。先に JSON を解析して再シリアライズしないでください。

### Node.js

```js
import crypto from 'node:crypto'

function verify(secret, timestamp, rawBody, signature) {
  const digest = crypto
    .createHmac('sha256', secret)
    .update(timestamp + '.' + rawBody)
    .digest('hex')

  return signature === 'sha256=' + digest
}
```

### Go

```go
func verify(secret string, timestamp string, rawBody []byte, signature string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(timestamp))
    mac.Write([]byte("."))
    mac.Write(rawBody)
    want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(want))
}
```

### 審査イベントのペイロード

審査イベントの `data` は安全な要約です。タイトルと抜粋は制御文字を除去して切り詰められ、匿名コンテンツには投稿者情報が含まれません。チャットメッセージの通報は「モデレーター画面で処理してください」とだけ伝え、メッセージ本文や通報者は含みません。サイト内パスは `baseUri` と組み合わせて絶対 URL にできます。

```json
{
  "event": "moderation.report.post.created",
  "timestamp": 1710000000,
  "data": {
    "baseUri": "https://forum.example",
    "approval": {
      "id": "report:9012",
      "kind": "report",
      "targetType": "post",
      "targetId": 3456,
      "reportId": 9012,
      "topicId": 1201,
      "reason": "spam",
      "title": "期中复习资料汇总",
      "excerpt": "……",
      "anonymous": false,
      "author": { "id": 7, "displayName": "Alice", "url": "/u/7" },
      "categories": ["学习"],
      "createdAt": "2026-10-04T08:00:00Z",
      "targetUrl": "/p/post/1201#post-3456",
      "moderationUrl": "/moderation?tab=reports",
      "actions": [
        { "action": "ban", "url": "/moderation/action?token=…" },
        { "action": "dismiss", "url": "/moderation/action?token=…" }
      ]
    }
  }
}
```

`id` は安定した審査 ID で、重複排除のキーでもあります。同じ URL は 24 時間以内に同じ審査を 2 回受け取りません（プロセス内の記憶のため、再起動後に 1 回重複する可能性はありますが、取りこぼしはありません）。審査待ちコンテンツの `id` には提出された版の番号が含まれるため、編集後に再び手動審査に入った場合は新しい審査として通知されます。

手動審査の審査イベントには次の 3 つのフィールドもあります。`reason` は `sensitive_word`（禁止語に一致）または `ai`（AI 審査が判断できなかった・失敗した）、`version` は提出された版の番号、`edited` が `true` の場合は公開済みコンテンツの編集で、タイトルと抜粋は提出された版から取ります。

### チャネル種別

各コールバック URL ごとにチャネル種別を選べます：

- **汎用 Webhook（JSON）**：上記のリクエスト形式で、HTTP 2xx を成功とみなします。
- **Feishu グループボット**：Feishu のグループに「カスタムボット」を追加し、その Webhook URL を入力します。Feishu チャネルは審査イベントのみを配信し、対象・理由・抜粋・クイック操作ボタンを含むカードとして表示します。Feishu の Webhook URL は認証情報として扱われ、暗号化して保存され、保存後は表示されません。空のまま保存すると設定済みの URL を保持します。HTTP 200 でも `code` が 0 以外の応答は失敗とみなします。

Feishu ボットで「署名検証」を有効にしている場合は、そのシークレットを Secret に入力してください。リクエスト本文に `timestamp` と `sign` が追加されます：

```text
sign = base64(HMAC_SHA256(key = timestamp + "\n" + secret, message = ""))
```

### クイック操作のセキュリティモデル

カードや審査ペイロード内の操作リンクはサイト内の確認ページ `/moderation/action` を指し、直接何かを実行することはありません：

- リンク内の token はサイトが署名したもので、「このサイトがこの対象と操作のために発行したリンク」であることだけを証明します。24 時間で失効し、ログイン資格情報にはなりません。
- リンクを開くにはログインが必要です。確認ページはまず対象と現在の状態を読み取り専用で表示し、モデレーターが操作ボタン（「承認」「封禁」など）を押した後にだけ実行します。
- 実行時にはログイン中のアカウントのモデレーター権限とカテゴリ範囲を再確認し、モデレーター画面と同じ操作を使い、審査ログには実際の操作者が記録されます。
- 他の人がすでに処理した内容は重複して処理されません。操作は通知に含まれる版に紐づくため、通知後に作成者が編集した場合は実行されず、確認ページで最新版をモデレーター管理で確認するよう案内します。
- 確認ページはキャッシュされず、Referer を送信せず、token はアクセスログに記録されません。

### 失敗保護

同じコールバック URL への通知が連続 3 回失敗すると、システムはその URL を自動的に無効化し、「異常終了」として表示します。再度有効化して保存すると、失敗状態はクリアされます。
