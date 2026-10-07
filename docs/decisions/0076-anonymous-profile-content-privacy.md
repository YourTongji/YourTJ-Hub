# Independent anonymous profile content privacy

## Status

Accepted
Class: feature

## Context and Problem Statement

A persistent persona links its public topics and replies on one profile. Its owner needs control
of this aggregate history independently of the main profile. The existing main profile display
preference is local to a client and cannot suppress public server payloads or crawler HTML.

## Decision Drivers

- The owner controls aggregation across Web, native mobile and public server rendering.
- Anonymous and main-account presentation preferences remain independent.
- Disabling profile aggregation does not delete or moderate the underlying forum content.
- Existing public histories remain visible until an owner makes an explicit privacy choice.
- Publishing restrictions cannot prevent a live owner from reducing profile disclosure.

## Considered Options

- Reuse the main account's local display preference.
- Add an independent local preference only to the anonymous profile UI.
- Persist an independent persona preference and enforce it at the public profile boundary.

## Decision Outcome

Choose a server-persisted `show_content` preference on the persona, default true for new and
existing identities. The authenticated owner updates it through a required boolean without a
caller-supplied persona UID. Live frozen or restricted owners can change privacy; this does not
restore publishing. Closed accounts cannot update it.

A disabled preference omits topics, replies, counts and pagination from the public page payload
and crawler HTML for every viewer, including the owner. Name, avatar and the private owner-side
management action remain. Web and native mobile hide tabs and streams, and reload after changes;
profile responses are no-store. Original topics and replies keep ordinary forum visibility.
This extends the persona boundary in [0074](0074-six-character-persona-names.md) and does not alter
restricted administration in [0075](0075-restricted-anonymous-administration.md).

## Pros and Cons of the Options

- Reusing the main local preference is small but couples two identities and fails on other clients,
  direct page payloads and crawler rendering.
- An independent UI-only preference separates controls but still returns the hidden history.
- A persisted preference enforces one consistent public result and requires an additive migration.
  It controls aggregation only: existing external copies and ordinary forum access remain.

## Links

- [Product behavior](../product/anonymous-identity.md).
- [Operations and migration](../operations/anonymous-identity.md).
- [Public profile boundary](../../apps/gooseforum/app/http/controllers/forum/anonymous_profile.go).
- [Privacy request contract](../../packages/api-contract/paths/anonymous-identity.yaml).
