# AI image and text moderation at publish time

## Status

Accepted
Class: architecture

## Context and Problem Statement

Topic and reply writes already pass a deterministic sensitive-word gate (`block` rejects, `review` stores
the content as pending, issue #975 audit). Images are not semantically checked: an uploaded image is
stored first and only later bound to content, so "normal text + violating image" passes. In addition,
pending content registered its images as `ACTIVE` usages, so a pending post's images were publicly
readable through `/file/img/*` before any human looked at them.

Jev (TypeSafe System One, also served by OpenRouter Decisions) returns typed probabilities for named
questions but does not read images. A cheap vision model can transcribe and describe images, but it must
not decide site policy, and third-party models may refuse or fail.

## Decision Drivers

- Keep the existing `block / pending / normal` state machine and review queue; no second workflow.
- Block or hold content inside the publish request; never publish first and retract later.
- Pending content's images must not be public earlier than its text.
- Model failure, refusal, malformed output, missing configuration or exceeded budgets must never allow.
- Site rules and thresholds belong to administrators and must be replayable without new model calls.
- Bound cost and latency; keep legacy uploads without usage rows readable.

## Considered Options

- A single "is this allowed?" call to a vision model.
- Vision evidence + one Jev `Choice` over categories and actions.
- Vision evidence + parallel per-policy Jev `Noul`s + severity `Score` + `review_needed` `Noul`,
  with the final action computed by a Go resolver.
- For pending images: a new `PENDING` usage status, or private-until-bound for every new upload.

## Decision Outcome

Use the third option. The vision model (OpenAI-compatible, configurable) only returns a strict, neutral
evidence JSON (OCR, scene, visible symbols, factual risk observations, `uncertain`, `refused`); text
inside images is data, never instructions. One Jev request per publish carries the visible text, the
evidence and the administrator-written policy text, asking one `Noul` per enabled policy (violations can
coexist, so no single `Choice` splits probability), a 0–3 severity `Score` and `review_needed`. The Go
resolver applies per-rule actions (`review` rules never auto-block), review/block thresholds, severity
escalation of already-triggered block rules only, and forces `review` whenever evidence is incomplete.

Failure semantics are fixed: any vision or Jev error, refusal, invalid JSON, invalid probability,
missing configuration, rate-limit guardrail or too many images yields `review`. Jev retries at most once
and only on 402/429/5xx/timeouts. External (non-site) images are never fetched server-side; they follow
the configured `review` or `block` policy.

Modes: `shadow` (default) evaluates asynchronously after the write and only records; `enforce` decides
synchronously and extends the HTTP write deadline only for requests that call models; `deferred`
(check after publishing) stores the content as pending without waiting, then decides in the background
through the same review implementation as the queue: `allow` publishes it silently, `block` rejects it and
notifies the author, `review` and every failure leave it in the review queue. A per-subject generation
counter plus a title/body comparison discard a background result once the author has edited again, and
disallowed external images are still rejected synchronously because no model call is needed. Jev sees at
most 300 title and 4,000 visible body characters; longer content is marked `text_truncated` evidence and
never auto-published, because the unsent part was not checked. In-flight
checks are process-local: a restart leaves the content pending for a reviewer (fail-closed). Every decision is
stored in `moderation_ai_decisions` with policy revision/hash, models, raw probabilities, evidence
status, final and applied action; review-queue outcomes and manual labels are written back, and the
admin replay endpoint recomputes a confusion matrix from stored probabilities. Thresholds move to
`enforce` only after shadow samples are labeled and replayed; demo thresholds are not used as-is.

Pending images use an explicit `PENDING` usage status (applies to sensitive-word review too).
`/file/img/*` returns 404 for anonymous readers and other users, serves the uploader, SiteManager
reviewers and moderators whose scope covers the owning topic's categories with `Cache-Control: private,
no-store`, and approval/unblock promotes the rows to `ACTIVE`. Usages are registered before the
background check starts, so an early automatic approval always finds them. The app requests images
anonymously and retries a 404 from the site's own `/file/img/` path once with the session, never across
redirects or to other hosts, and never stores the authenticated response on disk.
Private-until-bound for every upload was not chosen: it changes all new uploads, cannot distinguish
legacy normal uploads that have no owner row, and needs an owner preview path for every editor.

API keys use separate securestore purposes, are never returned, and can be cleared explicitly. User
images and text are sent to the configured providers during inference; deployments must restrict
providers and enable zero data retention at the provider account level.

### Consequences

- Good: no new state machine, deterministic resolver, fail-closed behavior, replayable thresholds, and
  the pending-image leak is closed for both AI and sensitive-word review.
- Bad: `enforce` adds vision + Jev latency to image publishes and failures increase review workload.
  `deferred` removes the wait but hides every new post from other readers until the check finishes, and
  a rejection reaches the author as a notification instead of an editor message.
- Scope: topics, replies and edits (including Agent/MCP writes through the same handlers). Avatars,
  course reviews, private messages and personal stickers keep their current checks; an image uploaded
  but never bound to content is still readable by URL until a later private-until-bound decision.

## Pros and Cons of the Options

- Single vision verdict: simplest, but the model invents policy, refusals look like verdicts and
  thresholds cannot be replayed.
- Jev `Choice`: one call, but coexisting violations dilute each other's probability.
- Parallel `Noul`s + Go resolver (chosen): auditable signals and site-owned actions; needs schema
  validation and a second model for images.
- `PENDING` usage status (chosen): incremental on the existing ACTIVE/RECOVERING/PURGED lifecycle and
  safe for legacy files; leaves the pre-bind upload window open.
- Private-until-bound: stronger, but changes every upload and needs a legacy-compatibility migration.

## Links

- Issue: https://github.com/YourTongji/YourTJ-Hub/issues/975
- [Image egress through the internal proxy](0009-image-egress-internal-proxy.md)
- [Deletion final-state retention](0021-deletion-final-state-data-retention.md)
- [Forum governance](../product/forum.md#governance-and-public-exports)
- [AI moderation runbook](../operations/deployment.md#ai-图文审查issue-975)
