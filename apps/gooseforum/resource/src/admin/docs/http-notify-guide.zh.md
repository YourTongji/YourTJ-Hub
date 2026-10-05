## HTTP 通知使用说明

HTTP 通知会在选中的站内事件发生后，异步向回调地址发送 POST 请求。这个功能是尽力通知，失败不会影响用户发帖、回复或注册。

### 支持的事件

- `topic.published`
- `topic.updated`
- `comment.created`
- `user.signup`
- `moderation.report.created`

审批事件（issue #1049，版主待办）：

- `moderation.review.topic.requested`：话题进入人工审核队列（命中敏感词，或 AI 审查存疑、审查失败转人工；自动检查期间不通知）
- `moderation.review.post.requested`：回复进入人工审核队列
- `moderation.report.topic.created`：话题被举报
- `moderation.report.post.created`：回复被举报
- `moderation.report.chat_message.created`：聊天消息被举报
- `moderation.report.course_review.created`：课评被举报

`moderation.report.created` 保留为“全部举报”聚合事件，载荷格式不变。**范围变化**：以前只有话题和回复举报会触发它，现在聊天消息和课评举报也会触发，`targetType` 会出现 `chat_message` 和 `course_review`；已订阅它的接收方请确认能处理这两种取值。同一个地址同时订阅聚合事件和具体举报事件时，每条举报只投递一次，优先使用具体事件。先发后审的 AI 自动检查（检查中）不算人工待审，不会触发通知；AI 结论为人工复核时才发送。课评的待审状态尚未实现（规划中），目前只有课评举报会通知。

### 请求格式

请求体固定为 JSON，Header 会带上事件名、投递 ID、时间戳和签名。

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

`baseUri` 可用于把 `topic.url`、`user.url`、`post.url` 这类站内路径拼成完整 URL；新接入建议优先使用 `topic`、`user`、`post`、`reporter` 等结构化对象，避免重复解析顶层镜像字段。评论事件会额外包含内容预览、评论人信息和 post URL；举报 post 时也会包含被举报 post 信息。

### 签名校验

`X-Goose-Signature` 由 Secret、时间戳和原始请求体计算得出。

```text
sha256 = HMAC_SHA256(secret, timestamp + "." + rawBody)
```

验签时请使用原始 body，不要先解析再重新序列化。

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

### 审批事件载荷

审批事件的 `data` 是一份安全摘要：标题与摘要已去除控制字符并截断，匿名内容不含作者身份，聊天消息举报只提示“请到版主工作台处理”，不含消息正文和举报人。站内路径可与 `baseUri` 拼成完整 URL。

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

`id` 是稳定的审批标识，也是去重键：同一个地址在 24 小时内不会重复收到同一审批（进程内记忆：重启后或多实例部署时可能重复提醒一次）。投递失败（网络错误、对方返回非成功）不占用去重名额，同一审批之后若被再次发布仍会投递。待审内容的 `id` 包含送审版本号，作者改稿后再次进入人工审核会作为新审批通知。

人工审核审批还有三个字段：`reason` 为 `sensitive_word`（命中敏感词）或 `ai`（AI 审查存疑或失败）；`version` 是送审版本号；`edited` 为 `true` 表示这是已公开内容的改稿，标题与摘要取自送审版本。

### 通道类型

每个回调地址可以选择通道类型：

- **通用 Webhook（JSON）**：上文的请求格式，HTTP 2xx 视为成功。
- **飞书群机器人**：在飞书群中添加“自定义机器人”，把 webhook 地址填入 URL。飞书通道只投递审批事件，以卡片形式展示目标、原因、摘要和快捷操作按钮。飞书 webhook 地址等同于凭据：服务端加密存储，保存后不再回显，留空保存会保留已配置的地址。飞书返回 HTTP 200 但 `code` 不为 0 时视为失败。

如果在飞书机器人中开启了“签名校验”，把签名密钥填入 Secret，系统会在请求体中加入 `timestamp` 与 `sign`：

```text
sign = base64(HMAC_SHA256(key = timestamp + "\n" + secret, message = ""))
```

### AstrBot 机器人

选择「AstrBot 机器人（push_lite 插件）」可以通过 AstrBot 把通知推到 QQ 群、私聊等会话，插件不需要改动：

1. 在 AstrBot 中安装 [astrbot_plugin_push_lite](https://github.com/Raven95676/astrbot_plugin_push_lite)。插件默认在 `9966` 端口提供接口，这个端口和 AstrBot 管理面板的端口不同。
2. URL 填插件的接口地址，例如 `http://astrbot.example.com:9966/send`。只填到端口时会自动补上 `/send`。论坛服务器必须能访问这个地址。
3. 在要接收通知的会话里发送 `/sid`，把返回的 SID（例如 `aiocqhttp:GroupMessage:123456`）填入「目标会话 SID」。
4. 把插件配置里的 API token 填入「API token」。它和 Secret 一样加密存储、不再回显。

系统向插件发送 `POST /send`，请求头为 `Authorization: Bearer <API token>`，请求体为：

```json
{ "content": "【新举报 · 回复】垃圾广告\n编号：举报 #12 · 回复 #34\n…", "umo": "aiocqhttp:GroupMessage:123456", "message_type": "text" }
```

AstrBot 通道可以订阅所有事件，统一以纯文本发送：审核和举报消息包含编号、标题、摘要、作者、分类和处理链接，其他事件包含标题、关键字段和站内链接。插件返回 `{"status": "queued"}` 才算成功；令牌错误（403）或缺少字段（400）时会显示插件给出的原因。插件收到消息后排队转发，转发到聊天平台失败时论坛无法得知，请查看 AstrBot 日志。

### 测试发送与单独保存

每个回调地址展开后都有「测试发送」和「保存此地址」：

- **测试发送**用表单里当前的配置（可以还没保存）立即发送一条测试消息，并在按钮旁显示结果。URL 或 Secret 留空时沿用这个地址已保存的值（仅限通道类型没改；改了通道需要重新填写）。测试不受总开关、启用状态和订阅事件影响，也不计入失败次数。测试请求由论坛服务器发出，和正式通知一样不限制目标地址（包括内网地址），并会把失败原因显示给管理员；只有站点管理员能使用。
- 通用 Webhook 收到 `webhook.test` 事件，同样带 `X-Goose-*` 请求头和签名，可以用来验证签名代码：

```json
{
  "event": "webhook.test",
  "timestamp": 1710000000,
  "data": {
    "baseUri": "https://forum.example",
    "endpointId": "…",
    "endpointName": "审核群",
    "message": "This is a test delivery from the HTTP notification settings."
  }
}
```

- 飞书群收到一张带「测试」标签的示例审批卡片，样式与真实通知相同；卡片上的按钮只会打开版主工作台。
- AstrBot 会话收到一条以「【测试】」开头的示例举报文本，链接只会打开版主工作台。
- URL 没有主机名（例如 `http:///send`、`http://:9966/send`）时不会发送，结果提示 `url is missing a host`。
- **保存此地址**只保存当前这一个地址，其他地址未保存的修改不会一起提交。保存成功后该地址会自动收起。

### 快捷审批的安全模型

卡片和审批载荷中的操作链接指向站内确认页 `/moderation/action`，不会直接执行任何操作：

- 链接中的 token 由站点签名，只能证明“这条链接由本站为该目标和操作签发”，24 小时后过期，不能作为登录凭据。
- 打开链接必须先登录；确认页先只读展示目标和当前状态，版主点击操作按钮（如“通过”“封禁”）后才执行。
- 执行时按当前登录账号重新校验版主权限和分区范围，复用版主工作台同一套操作，审核日志记录真实操作者。
- 内容已被他人处理时不会重复执行；操作绑定通知中的送审版本，作者在通知发出后改稿时不会执行，确认页会提示到工作台查看最新版本。
- 确认页不缓存、不发送 Referer，访问日志中不记录 token。

### 失败保护

每条通知只发送一次，不会自动重试：接收方暂时不可用时，这条通知就会丢失，所有事件（包括审批和举报提醒）都是如此。失败会计入失败次数，最近一次错误显示在设置页。通知只是提醒，待处理事项以版主工作台为准。

同一个回调地址连续 3 次通知失败后，系统会自动关闭该地址，并标记为异常终止。重新启用并保存后会清空失败状态。
