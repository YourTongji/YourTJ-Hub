# Bounded HTTP notification retries

## Status

Accepted
Class: bug-fix

## Context and Problem Statement

HTTP notification delivery was a single best-effort attempt. A transient receiver failure lost a
moderator reminder even though the site already had a durable task queue.

## Decision Drivers

- Retry transient HTTP notification failures without introducing another durable queue.
- Never persist callback URLs or channel secrets in retry task payloads.
- Re-check endpoint state and event subscription before each retry.
- Keep approval deduplication and the existing three-failure endpoint breaker effective.

## Considered Options

- Keep one delivery attempt and accept reminder loss during receiver outages.
- Add a dedicated notification outbox and dispatcher.
- Reuse the existing task queue with a bounded retry task.

## Decision Outcome

Chosen: reuse the existing task queue for bounded HTTP notification delivery retries.

- A failed HTTP notification releases its in-memory dedupe claim and enqueues a retry task containing
  the rendered safe body, event, channel, timestamp, dedupe key, expiry and a non-secret endpoint
  locator. URL-only endpoint locators are SHA-256 digests. The task never stores a callback URL or
  secret.
- The existing task worker provides bounded attempts. Each attempt reloads the current notification
  configuration and stops if the global switch, endpoint, subscription, or 24-hour delivery window
  no longer permits delivery. Successful delivery keeps the same endpoint-and-approval dedupe claim;
  failed retries release it. Each failed send continues to count toward automatic endpoint disabling
  after three consecutive failures.
- For endpoints without a stable ID, the retry locator is a digest of the current callback URL. Changing
  that URL invalidates pending retries instead of redirecting an old notification to the new destination.
- Enqueueing a retry remains best-effort: if the task queue cannot persist it, the original failed delivery
  has no retry and an error is logged. Guaranteeing persistence would require a durable enqueue/outbox path.
- Terminal retry task history is retained for seven days, then cleaned by the existing daily job.
- This decision supersedes the single-attempt delivery clause of [0064](0064-moderation-approval-notify-channels.md);
  the other notification channel, privacy, and signed confirmation decisions remain in force.

## Pros and Cons of the Options

- Keep one attempt: no queue work, but a transient receiver outage permanently loses the reminder.
- Dedicated outbox: clear separation, but adds another persistence model and dispatcher despite an
  existing queue that already supports retries and task cleanup.
- Existing task queue: reuses deployed retry and worker behavior with bounded scope; retry execution
  remains subject to the current process's notification configuration and in-memory dedupe lifetime.

## Links

- [Issue #1060](https://github.com/YourTongji/YourTJ-Hub/issues/1060)
- [0064](0064-moderation-approval-notify-channels.md)
