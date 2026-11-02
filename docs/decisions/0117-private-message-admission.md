# Recipient policy for new private conversations

## Status

Proposed
Class: feature

## Context and Problem Statement

Blocks stop known unwanted contact, but recipients also need to limit first contact.
A conversation row can be created before a message is sent, so row existence cannot
establish a prior relationship. The maintainer explicitly chose everyone, mutual
followers or no new conversations, with real-message history exempt and blocks first.

## Decision Drivers

- The recipient controls new contact across all clients and send types.
- Restricting new contact preserves conversations with committed messages.
- Saving a restriction cannot race a send through a missing preference row.
- Denial must not disclose the recipient's setting or block relationship.

## Considered Options

- Client-only filters or a separate permission check before sending.
- A recipient policy enforced inside the existing message transaction.
- Permission requests and a separate accepted-conversation state machine.

## Decision Outcome

Store `chatMessagePolicy` in the existing owner-only privacy settings, defaulting to
`everyone` for missing rows and upgraded accounts. Check it in the message transaction
under the same ordered participant locks used by blocks. Policy updates lock the
owner before the settings row. Use committed message existence in either direction,
not empty conversation/config rows, for the history exemption. Mutual requires both
active follows. Bots and privileged users have no exemption. Blocks still win.

All direct, media, reply and forward writes use the same check. A committed idempotent
retry acknowledges an earlier delivery; it neither writes again nor bypasses the
identity/content conflict checks. All denied first contacts use the same public error.
The setting is separate from profile aggregate audiences and does not erase history.

## Pros and Cons of the Options

- Client filters are easy to bypass; a separate preflight races policy changes.
- The existing transaction provides one rule and a clear commit order without a new
  conversation state. It adds a settings read and, for restricted first contact, an
  indexed history/follow lookup. Saving policy uses the existing account row lock.
- Permission requests offer a richer inbox workflow but add pending states and user
  actions outside the requested three choices, so are not selected.

## Links

- [Issue #1431](https://github.com/YourTongji/YourTJ-Hub/issues/1431)
- [Existing block and message-evidence decision](0040-user-blocks-and-private-message-reports.md)
- [Forum permissions](../product/forum.md#private-message-permissions)
- [Owner settings contract](../../packages/api-contract/paths/user-account.yaml)
