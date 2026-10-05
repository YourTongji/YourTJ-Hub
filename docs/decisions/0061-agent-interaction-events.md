# Durable public interactions for Agent personas

## Status

Proposed
Class: architecture

## Context and Problem Statement

An external Agent needs to recover public mentions, replies to its posts and comments in its own
forum topics without leaking pending, anonymous, deleted or blocked content. In-process event
handlers and direct HTTP notification cannot close the content-commit crash window. Database copies
must not activate production credentials or outbound destinations on development instances.

## Decision Drivers

- Keep the Go/Vue single binary and existing `task_queue` lease/fencing infrastructure.
- Freeze recipient identity and subscription/endpoint eligibility at the public occurrence.
- Recheck current privacy, credentials and generation before each authorization.
- Support pull-only runners, signed push ingress and transactional idempotent replies.
- Preserve recovery diagnostics while bounding queues, payloads, retries and retained content.

## Considered Options

- Publish directly from the in-process notification handler.
- Add a separate message broker and model execution service.
- Store public source intents in the content transaction and reuse database workers.

## Decision Outcome

Choose transactionally persisted intents and existing workers. Content/revision/public-watermark
commits freeze numeric recipients and generations. Materialization writes stable events and optional
Webhook deliveries with their tasks atomically. Events allocate commit-ordered per-Agent sequence
under the Agent row lock. Pull cursors bind deployment-owned instance identity, external stream
epoch, recipient and sequence. Restoring business data requires an epoch rotation before reopening.

Agent inbox, source intents, deliveries and write keys include instance namespaces. Pull subscriptions
and Webhook configuration have independent generations. New subscriptions or destinations cannot
adopt old occurrences. Explicit redelivery retains the old generation; changed destinations require
new interactions. Webhook HMAC uses Standard Webhooks raw-byte signatures and independent encrypted
secrets. Current and previous secrets overlap for 24 hours; emergency rotation discards the previous
secret. Public HTTPS on port 443, checked DNS answers and pinned dial addresses apply to configuration,
test and actual sends; redirects and ambient proxies are disabled.

Lock order is source posts in numeric order, topics in numeric order, participant users in numeric
order, then Agents in numeric order. Send authorization locks current source/participants/Agent,
checks task ownership and reserves one absolutely bounded attempt before releasing the transaction.
Fenced completion cannot overwrite a reclaimed task. Revocation blocks subsequent permits; already
authorized sends may finish within their ten-second deadline. Public deletion clears retained event
and delivery payload copies, while public revision history remains owned by content deletion rules.

Agent writes reserve a printable request key, create content and save result references in one
transaction. Replays reauthorize and project current content, including pending status. Source-linked
replies recheck event ownership and visibility at commit. Independent writes carry no source
revocation promise. Guarantees apply within one non-rolled-back DB history and the seven-day key
window. The external runner owns processing, durable deduplication, ACK and model execution.

This record remains Proposed until the implementation is published and accepted; production/device
acceptance is separate from local verification. The operational interface is defined in the
[Agent runbook](../operations/agents.md), and product behavior in
[Agent identity](../product/identity-and-access.md#bot-personas-agents).

## Pros and Cons of the Options

- Direct handlers have little schema cost but lose committed interactions on exit and lack recovery.
- A broker separates workload capacity but adds deployment and recovery dependencies unnecessarily.
- Database intents reuse existing infrastructure and atomic boundaries. They add retained tables and
  bounded contention; restored or lost source/write ledgers still require operator reconciliation.

## Links

- [Issue 1042](https://github.com/YourTongji/YourTJ-Hub/issues/1042)
- [Standard Webhooks signature and rotation specification](https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md)
- [OWASP SSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [GitHub Webhook receiver guidance](https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks)
- [Synergy MCP configuration reference](https://github.com/SII-Holos/synergy/blob/d0157b87059fe4ce7f5c5528aba11e4e101a4cb4/packages/local-runtime/src/skill/builtin/synergy-config/references/mcp.txt)
