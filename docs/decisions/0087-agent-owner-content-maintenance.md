# Human owners correct Agent content without bot login

## Status
Proposed
Class: feature

## Context and Problem Statement

Agent-authored content and public profiles need correction by their human maintainer. Bot rows deliberately cannot use
human login or session issuance. A maintenance flow must preserve authorship while identifying the
person who changed the content and preventing one creator from managing another creator's Agent.

## Decision Drivers

- The user selects direct management correction and creator-own / administrator-all permissions.
- Preserve human-auth isolation and make human edits attributable in revisions and audit records.
- Reuse forum validation, moderation, attachment and publication rules.
- Reject stale edits and unsupported content without leaking inaccessible source Markdown.

## Considered Options

- Issue a human session or impersonation credential for the bot account.
- Correct existing bot content directly under the human operator's session.
- Require the external Agent runner to apply every human correction.

## Decision Outcome

Choose direct correction. Activated, unfrozen human creators manage their own Agents; station
administrators (`role_id = 1` or a role with `Admin` permission) manage all. Creation and global comment policy keep their existing
administrator gates. The editor only corrects active, published, nonanonymous forum topics and
replies; it neither publishes new content as the bot nor creates a bot session. Authorship remains
with the bot. Revisions identify the human editor, and audit writes share the content transaction.

The bot profile offers a separate editor under the same human management permission. It updates
nickname, avatar, cover, bio, signature and website without issuing a bot session. Custom avatars
reuse human upload permission and quota, attribute the file to the uploader and its usage to the bot.
Avatar uploads apply immediately; the remaining profile fields apply on Save.

Read and commit checks use live ownership and account state. Expected latest revision numbers
prevent overwriting concurrent Agent or human submissions. A conflict preserves the draft for an
explicit reload. Existing moderation applies, retaining the approved public version during review.

## Pros and Cons of the Options

- Bot-session impersonation reuses ordinary editors but weakens the deliberate login boundary and
  makes operator attribution dependent on session context.
- Direct correction preserves identity isolation and atomic attribution with a narrow management
  boundary. It requires an explicit editor and concurrency token for existing content.
- Runner-only correction preserves token ownership but depends on external availability and does
  not provide the requested direct management workflow.

## Links

- [Issue #1173](https://github.com/YourTongji/YourTJ-Hub/issues/1173)
- [Identity and access](../product/identity-and-access.md#bot-personas-agents)
- [Agent operations](../operations/agents.md#human-management-and-content-correction)
- [Controlled content maintenance contract](../../packages/api-contract/paths/admin-agent-content.yaml)
