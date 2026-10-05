## HTTP notification guide

HTTP notifications send asynchronous POST requests to callback URLs when selected site events happen. This is best-effort delivery; failures never block posting, replying, registration, or reports.

### Supported events

- `topic.published`
- `topic.updated`
- `comment.created`
- `user.signup`
- `moderation.report.created`

Approval events (issue #1049, moderator to-dos):

- `moderation.review.topic.requested`: a topic entered the manual review queue (a sensitive word matched, or AI review was uncertain or failed; nothing is sent while the automatic check is running)
- `moderation.review.post.requested`: a reply entered the manual review queue
- `moderation.report.topic.created`: a topic was reported
- `moderation.report.post.created`: a reply was reported
- `moderation.report.chat_message.created`: a chat message was reported
- `moderation.report.course_review.created`: a course review was reported

`moderation.report.created` remains the aggregate "all reports" event with an unchanged payload. If one URL subscribes to both the aggregate event and a specific report event, each report is delivered once, using the specific event. The publish-first AI check (still checking) is not manual review and sends nothing; a notification is sent only when the AI verdict asks for human review. A pending state for course reviews is not implemented yet (planned); only course review reports notify today.

### Request format

The request body is JSON. Headers include the event name, delivery ID, timestamp, and signature.

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

`baseUri` can be used to turn site paths such as `topic.url`, `user.url`, and `post.url` into absolute URLs. New integrations should prefer structured objects such as `topic`, `user`, `post`, and `reporter`, avoiding duplicate top-level mirror fields. Comment events also include the content preview, commenter profile, and post URL; report events for posts include the reported post details.

### Signature verification

`X-Goose-Signature` is calculated from the Secret, timestamp, and raw request body.

```text
sha256 = HMAC_SHA256(secret, timestamp + "." + rawBody)
```

Use the raw body for verification. Do not parse and serialize it again first.

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

### Approval event payload

The `data` of an approval event is a safe summary: titles and excerpts have control characters removed and are truncated, anonymous content carries no author identity, and chat message reports only say "handle it in the moderator workspace" without the message text or the reporter. Site paths can be joined with `baseUri` into absolute URLs.

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

`id` is a stable approval identifier and the deduplication key: the same URL does not receive the same approval twice within 24 hours (kept in process memory; after a restart one duplicate reminder is possible, never a missed one). For pending content the `id` includes the submitted revision number, so content that is edited and sent to manual review again is notified as a new approval.

Manual review approvals carry three more fields: `reason` is `sensitive_word` (a sensitive word matched) or `ai` (AI review was uncertain or failed); `version` is the submitted revision number; `edited` is `true` when the revision edits already published content, and the title and excerpt come from the submitted revision.

### Channel types

Each callback URL can choose a channel type:

- **Generic webhook (JSON)**: the request format above; any HTTP 2xx counts as success.
- **Feishu group bot**: add a "custom bot" to a Feishu group and enter its webhook URL. The Feishu channel only delivers approval events, rendered as a card with the target, reason, excerpt, and quick-action buttons. A Feishu webhook URL is a credential: it is stored encrypted, never shown after saving, and saving with an empty value keeps the configured URL. A Feishu response with HTTP 200 but a non-zero `code` counts as a failure.

If "signature verification" is enabled on the Feishu bot, enter its secret as the Secret; the system then adds `timestamp` and `sign` to the request body:

```text
sign = base64(HMAC_SHA256(key = timestamp + "\n" + secret, message = ""))
```

### Test delivery and saving one URL

Each callback URL has **Send test** and **Save this URL** at the bottom of its panel:

- **Send test** immediately sends one test message using the current form values (they do not need to be saved first) and shows the result next to the buttons. An empty URL or Secret reuses the value already saved for this URL. Tests ignore the master switch, the enabled state, and the subscribed events, and they never count toward the failure counter.
- Generic webhooks receive a `webhook.test` event with the usual `X-Goose-*` headers and signature, so you can check your signature verification:

```json
{
  "event": "webhook.test",
  "timestamp": 1710000000,
  "data": {
    "baseUri": "https://forum.example",
    "endpointId": "…",
    "endpointName": "Moderators",
    "message": "This is a test delivery from the HTTP notification settings."
  }
}
```

- Feishu groups receive a sample approval card marked "测试" (test) that looks exactly like a real notification; its buttons only open the moderator workspace.
- **Save this URL** saves only this callback URL; unsaved changes to other URLs are not submitted. The panel collapses after a successful save.

### Quick-action security model

Action links in cards and approval payloads point to the on-site confirmation page `/moderation/action` and never execute anything directly:

- The token in the link is signed by the site. It only proves "this site issued this link for this target and action", expires after 24 hours, and is not a login credential.
- Opening the link requires signing in. The confirmation page first shows the target and its current state read-only; the action runs only after the moderator clicks the action button (such as "Approve" or "Block").
- On execution, moderator permissions and category scope are re-checked for the signed-in account, the same commands as the moderator workspace are reused, and the moderation log records the real actor.
- Content already handled by someone else is not handled again; actions are bound to the revision in the notification, so nothing runs if the author edits the content after the notification, and the page points to the workspace for the latest version.
- The confirmation page is not cached, sends no Referer, and its token is not written to access logs.

### Failure protection

If the same callback URL fails 3 times in a row, the system disables that URL and marks it abnormally stopped. Re-enable and save it to clear the failure state.
