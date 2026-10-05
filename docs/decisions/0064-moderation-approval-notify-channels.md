# Moderator approval notifications through channel adapters and a signed confirmation page

## Status

Accepted
Class: feature

## Context and Problem Statement

Moderators only learned about pending review items and new reports by opening the workbench. The
existing HTTP notification module had one `moderation.report.created` event whose payload could
expose the real author of anonymous posts, did not cover manual review or chat and course review
reports, and only spoke one generic JSON format. Teams want approval reminders in a Feishu group,
with buttons that shorten the path to an action, without turning a chat message into a credential
that can moderate content by itself.

## Decision Drivers

- Never leak anonymous authors, chat content or reporter identity through a third-party channel.
- A forwarded or leaked notification must not be able to change content on its own.
- Reuse the workbench commands, permission scopes and moderation log; no second moderation path.
- Keep existing generic webhook consumers working without payload changes.
- Make adding another chat channel a local change, not a change to every event producer.

## Considered Options

- Extend the generic JSON payload only and let operators build their own Feishu relay.
- Channel adapters inside the HTTP notification module, with card buttons that open a signed,
  login-gated confirmation page.
- Feishu interactive cards with callback buttons that execute actions from the Feishu app.
- One-click links whose token alone performs the action.

## Decision Outcome

Chosen: channel adapters plus a signed confirmation page.

- Event producers publish domain events after commit. The HTTP notification service turns each
  event into a safe approval summary (sanitized, truncated text; no anonymous author; chat reports
  carry no text, note, reporter or actions) and lets each endpoint's channel encode it. Channels are
  `generic` (unchanged envelope, X-Goose headers, 2xx success), `feishu` (card schema 2.0,
  optional Feishu signature, HTTP 200 with non-zero `code` counts as a failure) and `astrbot`
  (plain text for every event, posted to the unmodified push_lite plugin's `/send` API with
  a Bearer token and a per-endpoint target session `umo`; only `status: queued` counts as success,
  so delivery failures inside AstrBot stay invisible). Endpoint URLs must name a host.
- Admins can send a test delivery to one endpoint before saving it: generic endpoints get a signed
  `webhook.test` envelope, Feishu gets a sample card, AstrBot gets the sample as text. Tests ignore the switches and subscriptions and
  never count toward automatic disabling. Each endpoint can also be saved on its own.
- Feishu webhook URLs are credentials: encrypted with a purpose-scoped key, never echoed, kept when
  saved empty, and scrubbed from delivery errors.
- Specific events (`moderation.review.{topic,post}.requested`,
  `moderation.report.{topic,post,chat_message,course_review}.created`) sit beside the legacy
  aggregate event. An endpoint subscribed to both receives each report once. Deliveries are
  deduplicated per endpoint and approval ID for 24 hours in process memory; a restart can repeat a
  reminder once. Each delivery is a single best-effort attempt with no automatic retry or durable
  outbox, so a receiver outage loses that notification (it still counts toward automatic
  disabling); a failed delivery releases its dedupe slot so a later publish of the same approval
  is delivered. Pending content IDs include the submitted revision ID. The aggregate
  `moderation.report.created` event now also fires for chat message and course review reports.
- Card buttons open `/moderation/action?token=…`. The token is an HMAC over subject, ID, action,
  revision ID and a 24-hour expiry, derived from the site signing key. It proves origin only.
  The page requires login, previews read-only, and executes only on an explicit POST that re-checks
  the current session's moderator scope, re-validates state (report still open via compare-and-set,
  the same revision still pending and latest) and reuses the workbench commands, so the moderation log
  records the real actor. The page sends no-store and no-referrer headers; access logs redact the
  token.
- Review approvals follow the versioned pipeline of decision 0061: every route to human review
  (sensitive word, AI uncertain or failed) passes through the publication service handoff, which
  enqueues a content-effects task in the same transaction; the task publishes the event only while
  that revision is still the latest pending one. Automatic checking does not notify, and a repeated
  handoff of the same revision enqueues nothing. Author-facing notices stay as in decision 0062.
- Course review pending state and chat report quick actions stay out of scope.

## Pros and Cons of the Options

- Generic payload only: no new surface, but every team rebuilds signing and card rendering, and the
  privacy rules would still have to be fixed in the payload.
- Adapters plus confirmation page: one safe summary for all channels and no new moderation path;
  costs an extra click and a login on the moderator's device.
- Feishu callback buttons: one-tap moderation, but they require a Feishu app with a public callback
  endpoint, map Feishu identities to site accounts, and move authorization into a third party.
- Token-only links: fastest, but any forwarded message becomes a bearer credential that can ban or
  approve content without a session.

## Links

- Feishu custom bot (signature, response codes): https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- Feishu card button behaviors: https://open.feishu.cn/document/feishu-cards/card-components/interactive-components/button
- AstrBot push_lite plugin (`/send` API): https://github.com/Raven95676/astrbot_plugin_push_lite
- Issue: https://github.com/YourTongji/YourTJ-Hub/issues/1049
- Related: [0056](0056-ai-image-text-moderation.md), [0061](0061-versioned-background-moderation.md), [0062](0062-moderation-notification-feedback.md)
