# Owner-scoped blocks and administrator-only private-message evidence

## Status
Proposed
Class: feature

## Context and Problem Statement

Forum content reports do not protect users from unwanted private contact. Reusing the public
moderator queue for private-message reports without additional access controls would disclose
private communications to category and global moderators.

## Decision Drivers

- Users need an immediate, reversible way to stop direct contact.
- Protection must also apply to Web and old clients through server enforcement.
- A report must disclose the selected received message, not grant access to a conversation.
- Evidence must retain the existing report lifecycle and remain actionable for administrators.

## Considered Options

- Hide messages locally and let all moderators inspect reported conversations.
- Store owner-scoped blocks and only the selected message's evidence, visible to administrators.
- Disable private messaging entirely.

## Decision Outcome

Use owner-scoped blocks with a maximum of 1000 entries. Either direction blocks new private
messages and new interaction notifications; delivery workers also recheck queued notifications.
Previously delivered notifications, public content and message history remain available for
context and reporting. The caller can list only their own blocks and can reverse them. Closing
either account removes the relationship. Block changes and message writes lock participant rows
in ID order; a send committed before a block may still finish its delivery.

Only a recipient can report an existing private message. The report contains at most 4000 runes
of that message, the author's numeric ID, a fixed reason and at most 300 runes of explanation.
It does not contain surrounding messages, attachment binaries, credentials or a conversation
access link. The UI explains the disclosure before submission.

Administrators see these reports in the existing moderation queue and may resolve/reject them
or use the author's profile for account moderation. Category and global moderators are excluded
before pagination, and direct status updates also require administrator permission. Private report
handling records its actor, time and outcome in the report row; it does not copy private evidence
or reporter identity into the broader moderator log. Evidence snapshots follow the existing
closed-report retention worker (180 days); open evidence remains while the case is open. Report
metadata and explanations retain the ordinary report lifecycle.

## Pros and Cons of the Options

- Local hiding is inexpensive but can be bypassed by another client and cannot prevent delivery.
  Conversation-wide moderation discloses messages that the user did not choose to report.
- Server blocks and bounded evidence protect all clients and minimize disclosure. They require
  relationship storage, transaction ordering, account cleanup and administrator handling.
- Disabling messaging removes this contact channel but removes an existing product capability.

## Links

- [Mobile product behavior](../product/mobile-experience.md)
- [Mobile release operations](../operations/mobile-releases.md)
- [Apple user-generated content guidelines](https://developer.apple.com/app-store/review/guidelines/#user-generated-content)
