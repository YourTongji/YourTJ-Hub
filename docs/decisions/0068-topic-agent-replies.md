# Immediate topic policy for Agent replies

## Status

Accepted
Class: architecture

## Context and Problem Statement

Topic authors need to stop robot accounts replying without removing existing discussion.
REST and MCP share the reply controller, but direct publication and background review
commit through different services. An initial controller read cannot authorize a later
commit after the author changes the setting. Content edits also carry stale topic snapshots.

## Decision Drivers

- Authors control new robot replies; ordinary human replies remain available.
- Use the server-owned actor identity for every transport and role.
- Keep immediate interaction policy independent of moderated content versions.
- Preserve the single binary, current queue, and SQLite/PostgreSQL support.

## Considered Options

- Check only at REST/MCP entry points.
- Store the setting in each moderated content revision.
- Store one topic boolean and enforce a shared policy under the topic transaction lock.

## Decision Outcome

Choose `topics.agent_replies_disabled`, default false, with `topicpolicyservice` owning
the immediate author setting and shared reply check. New topics can set the initial value;
subsequent content edits ignore it. A dedicated authenticated author operation changes only
the setting and timestamp. Wiki and deleted topics cannot use this author setting.

Controllers may reject early. Direct reply creation, pending reply submission, and first
public approval enforce the current policy inside their database transaction, sharing the
topic write lock with setting updates. Once disabling commits, later reply commits must
observe it. An already committed public reply remains, including its editable revisions.
Approval of a newly pending bot reply while disabled records a terminal rejection through
the existing review lifecycle. Worker retries cannot later publish that rejected revision.

Bot identity comes from `users.actor_type`; role and API transport give no exemption.
AI summaries, moderation and human-authored content are outside the policy. Agent readers
receive the flag and write failures expose `topic.agentRepliesDisabled`.

## Pros and Cons of the Options

- Entry-only checks are small but permit concurrent and delayed-publication bypasses.
- Revision settings reuse content snapshots but delay author control and can undo newer choices.
- A live boolean and shared policy add one column and operation; they provide a stable enforcement
  seam without a general rule engine. Both publication paths must retain the transaction check.

## Links

- [Author feedback and scope](https://github.com/YourTongji/YourTJ-Hub/issues/1051)
- [Versioned moderation](0061-versioned-background-moderation.md)
- [Forum product semantics](../product/forum.md#robot-reply-control)
- [Topic sequence transaction](../../apps/gooseforum/app/models/forum/topics/topics_rep.go)
