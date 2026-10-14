# Server-enforced profile section privacy

## Status

Accepted
Class: feature

## Context and Problem Statement

A member profile aggregates topics, activity, liked topics and both follow lists. Members asked to
limit who can browse these aggregates. The previous Web switches lived only in browser storage, so
the server, other devices, the native app and crawler HTML ignored them while the UI implied they
worked. The forum still needs public topics and replies to stay discoverable.

## Decision Drivers

- One owner choice applies to Web HTML, page JSON and the native app.
- Hidden sections leak nothing through rows, counts, cursors or mixed timeline entries.
- Existing accounts keep today's public profiles until their owner chooses otherwise.
- Copy must not promise more privacy than the model delivers.
- Account-level privacy (private accounts, follow approval, message and mention permissions) needs
  separate product decisions and must not block profile controls.

## Considered Options

- Keep the browser-local switches and only fix their copy.
- Hide whole profiles behind a private-account mode with follow approval (model B).
- Persist per-section audiences and enforce them where profile aggregates are built (model A).

## Decision Outcome

Choose model A. `user_privacy_settings` stores one row per member with an audience for
`profileTopics`, `profileActivity`, `profileLikes`, `followingList` and `followersList`; a missing
row means everyone. Audiences are everyone, signed-in members, followers, mutual follows and only
the owner; likes and follow lists offer the subsets that keep their meaning. The owner reads and
patches the settings through `/api/user/privacy-settings`; partial updates validate every field
before one transaction writes.

`profileprivacyservice.Evaluate` resolves each section for the viewer before any profile query
runs. The owner always sees everything; blocks and storage errors hide the section (fail closed).
Hidden sections return no rows or cursor, and the timeline filters actions in SQL so posts, likes
and follows obey their own sections. Profile responses are `private, no-store` and carry
`visibility` per section, with the audience shown only to the owner. Web and the app explain hidden
sections from the owner's, another member's and a guest's point of view. Legacy browser switches
are offered for an explicit one-time migration instead of being applied silently.

Topics, replies, search, user cards, counts and moderation tools are unchanged; this controls
profile aggregation only.

## Pros and Cons of the Options

- Browser-local switches are cheap but protect nothing and mislead members.
- A private-account mode protects more but changes following, messaging, notifications and
  moderation contracts that are still undecided.
- Per-section audiences protect the aggregates members care about with one additive table, at the
  cost of leaving individual public posts and counts visible, which the UI states explicitly.

## Links

- [Issue #1124 plan and review](https://github.com/YourTongji/YourTJ-Hub/issues/1124).
- [Product behavior](../product/identity-and-access.md#profile-visibility).
- [Mobile behavior](../product/mobile-experience.md#profile-and-privacy).
- [Evaluator](../../apps/gooseforum/app/service/profileprivacyservice/profileprivacyservice.go).
- [Settings contract](../../packages/api-contract/paths/user-account.yaml).
