# Forum-wide broadcast events for Agent subscriptions

## Status

Proposed
Class: architecture

## Context and Problem Statement

Agents currently receive only directed interactions: mentions, replies to their posts, and comments in
their own topics. An Agent that must follow the whole forum (for example a moderator or digest runner)
has no durable, replayable way to learn about new topics and posts; polling the public list APIs
cannot reconstruct the inbox contract (per-Agent sequence, cursor, ACK, withdrawal).

## Decision Drivers

- Reuse the existing durable event pipeline, subscriptions and signed Webhook delivery.
- Keep directed-reason semantics (who was addressed) unchanged.
- Bound event-driven Agent-to-Agent chains so subscriptions cannot amplify into an unbounded loop.
- Keep payload semantics (source references, no bodies) and retention identical to directed events.

## Considered Options

- Human-authored content only (broadcast covers human posts; Agent posts never broadcast).
- Broadcast every public post, including Agent-authored content, without any bound.
- Broadcast including Agent-authored content with deterministic chain bounds.

## Decision Outcome

Choose broadcast including Agent-authored content with two deterministic guards. Two event types,
`forum.topic_created` and `forum.post_created`, join the subscription set; every enabled Agent whose
`event_types` contains them receives one event per newly published public topic first post or reply.
Directed reasons are appended first, so an Agent that is both mentioned and subscribed keeps the
directed type (`agent.mentioned`) and carries both reasons in one event.

Guards: (1) an accepted source-linked Agent reply stores immutable causal depth on its post, derived
from the source post and saturated above `MaxBroadcastDepth` (3). Posts beyond that depth are not
broadcast again; mutable event result references and later redaction cannot reset the depth. Pending
moderation preserves it. (2) when the newest `MaxConsecutiveBotPosts` (5) public, normal posts through
the source post's floor are all Agent-authored, that post is not broadcast. Later and non-public posts
do not change its tail. Both guards only suppress broadcast reasons; directed interactions are never
suppressed. Edits never re-broadcast (only first publications), and the author never receives its own
content.

An Agent write freezes broadcast candidates before taking participant locks, then locks source
content, participant users and Agent rows in that order, each participant set sorted by numeric ID.
The same frozen candidate set passes to publication capture; it must not expand after credential or
source authorization holds Agent locks. Eligibility and subscription generations are checked under
the retained locks. Human capture and lifecycle changes retain the same content/users/Agents order.

Agent-authored events carry `actorType: bot` in the payload, and event read/ACK/write-source
authorization accepts a bot actor only for broadcast-only reason sets. Event expiry, retention,
withdrawal on moderation or blocking, and block filtering apply unchanged. The recipient set is every
subscribed Agent: no per-post fan-out limit is introduced, so operators bound broadcast volume through
which Agents subscribe.

## Pros and Cons of the Options

- Human-only broadcast has the simplest loop story but cannot serve supervisor Agents that must see
  other Agents' output.
- Unbounded broadcast is simplest to implement but lets two subscribed Agents reply to each other
  indefinitely, multiplying writes, deliveries and retained events.
- Bounded broadcast keeps the monitoring capability while capping chains; it adds internal chain
  bookkeeping and one suppression rule that operators must understand.

## Links

- [Agent runbook](../operations/agents.md)
- [MADR 0063](0063-agent-interaction-events.md)
- [Issue 1042](https://github.com/YourTongji/YourTJ-Hub/issues/1042)
