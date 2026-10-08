# Restricted anonymous identity administration

## Status

Accepted
Class: feature

## Context and Problem Statement

Per-post identity controls mix content moderation with private account relationships. Operators
cannot find all personas or govern an identity without locating an existing post. The maintained
persona model already separates public attribution, explicit reveal permissions and private audits.
A central admin list must preserve those boundaries while exposing approved owner relationships.

## Decision Drivers

- Central account governance follows the existing user-management workflow.
- Main-account relationships stay private outside explicitly authorized administration.
- Each disclosure and publishing restriction has a committed private audit and nonempty reason.
- Topic/reply interfaces serve content interactions on Web and native mobile.

## Considered Options

- Retain per-post reveal/governance controls.
- Add owner bindings to the ordinary user-management list for every Admin.
- Add dedicated restricted anonymous administration with explicit reveal and management grants.

## Decision Outcome

Choose dedicated restricted administration. The table/card list searches, filters and paginates
personas with their owners, including retained closed accounts. Loading each page requires user
management and an explicit reveal grant, with a viewing reason and a committed audit for each
returned mapping. Empty pages still record the access. Permission or audit failure releases no data.
No seeds, mapping export, ordinary operation-log entries or browser persistence are introduced.

Governance uses the persona UID and a separate action reason, with the existing atomic owner/persona
publishing restriction. Restore preserves independent account freezes and self-disabled status.
Web and native topic/reply surfaces omit the private control; compatible legacy APIs remain.
The owner’s profile/settings/composer identity controls continue to serve self-management.

This extends the private-audit boundary of [0066](0066-persistent-anonymous-forum-persona.md) and
does not change the naming decision in [0074](0074-six-character-persona-names.md).

## Pros and Cons of the Options

- Per-post controls preserve category-scoped governance, but clutter content UI and cannot browse
  or govern identities without posts.
- Ordinary user-list mappings minimize navigation, but imply private disclosure through Admin's
  wildcard and expose relationships during unrelated user-management tasks.
- Dedicated restricted administration centralizes the approved workflow and audits each disclosure,
  but requires an explicit grant and viewing reason and adds bounded audit rows for list reads.

## Links

- [Persona product behavior](../product/anonymous-identity.md).
- [Permissions, private evidence and logging](../operations/anonymous-identity.md).
- [shadcn-vue data tables](https://www.shadcn-vue.com/docs/components/data-table.html): structured filtering and pagination; existing repository components remain the implementation foundation.
- [OWASP logging guidance](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html): separate audit purposes and exclude sensitive data from ordinary logs. Sources checked 2026-10-07.
