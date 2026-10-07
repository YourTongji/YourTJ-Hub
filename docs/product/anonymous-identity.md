# Persistent anonymous forum identity

> Doc type: product spec (reference)
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-07

## Identity and naming

`Current`: each human account can create one persistent forum persona. It has a cryptographically
random 128-bit public UID, an independently generated fixed Beam avatar and a public `/a/:publicUid`
profile. It cannot log in, receive private messages, follow people, hold a separate wallet or gain
permissions. Numeric users remain the authentication, moderation, posting quota and credit subjects.
The anonymous profile links only its public forum topics and replies, with their visible counts.
Main profiles, Following, participation records and public author statistics exclude persona content.

`Current`: new names freely combine a two-Han-character action, a two-Han-character
scene/object, `的`, and a one-Han-character animal: 躲进云里的猫 or 抱着松果的熊.
The repository-authored component pool is embedded as `phrase6-v1`, with 187,776 distinct
six-character combinations. Components combine without grammar or semantic filtering, so
playful phrases such as 躲进松果的猫 are eligible. Confirmed names and already persisted
candidate batches retain their exact text across generator versions. Different personas can
have the same name; their UIDs, avatars and profiles distinguish them. A name confers no badge.

`Current`: Web Settings and the native mobile settings page expose ten distinct candidates per
batch, at most ten new batches per account per Shanghai calendar day. The initial batch counts.
Viewing and selecting already generated same-day batches costs nothing. Batches may repeat words
across draws. Database transactions serialize the shared cross-client quota and one-persona slot.
The client keeps its request key after an ambiguous failure; repeating it recovers the same batch.
Generation or persistence failure rolls the quota back. Midnight expires all previous-day candidates.
Confirmation accepts only the owner's persisted batch and an index from zero through nine.

`Current`: choosing a name requires explicit confirmation. It is locked until the next calendar
anniversary in Asia/Shanghai, preserving the time of day; February 29 maps to February 28 when needed.
The locked period prevents both new draws and changes. After the anniversary, selecting another word
keeps the UID, avatar, profile and historical attribution and starts another year. Selecting the same
name does not restart the lock. Nothing forces a name change. Dynamic author projections use the
current name; delivered notifications, screenshots and copied text are not recalled.

## Public profile and management entry

`Current`: Web member and persona profiles reuse the same cover/avatar header, responsive
action area, navigation tabs, statistics and forum topic list components. Native profiles
reuse the collapsing cover/navigation sliver, user card, edit button, animated tabs and topic
cards/content rows. Persona profiles show their own public topic/reply streams and counts;
they do not copy a main account's cover, badges, bio, relationships or activity.

`Current`: the signed-in user's ordinary profile menu includes the anonymous identity entry.
An existing persona opens its public profile; an unconfigured persona opens setup. The public
persona page determines ownership only from the viewer's private state. Its owner sees the
shared profile management button in the ordinary edit-action position, opening the shared
setup/management dialog or sheet. Other viewers receive no private binding or management action.
Web SVG avatars keep their original URL at comment, quote, list and profile sizes; no medium
raster derivative is requested for the public persona SVG endpoint.

## Settings and first use

`Current`: Web exposes a compact anonymous-identity row under Settings → Privacy, separate from
public-profile editing. It shows the current alias and availability or an unconfigured state. The
legacy `/settings#anonymous-identity` entry selects the Privacy tab; choosing a tab clears that
legacy anchor so reloading keeps the explicit tab selection. Native mobile retains its
account-settings entry, with a compact description separating persona management from
public-profile editing.

`Current`: Web topic, quick-publish and reply composers expose a compact identity menu. Choosing
setup opens the same accessible dialog as Settings without navigating away from the draft. Choosing
a candidate only previews it; explicit successful confirmation selects the persona in that composer
and closes the dialog. Cancelling or failed requests retain the draft and selected publishing identity.
A disabled persona can be managed but cannot be chosen for publishing.

`Current`: the dialog introduces the persona before drawing names, shows one candidate batch at a
time, and permits returning to previously generated same-day batches without spending quota. The
one-year lock, restricted audited reveal and post-closure retention are visible before confirmation;
full rules can be expanded. Reset time appears when the daily draw quota is exhausted. Failed draw
responses retain the request key, so retries recover the same batch.

`Current`: native topic and reply composers open a shared bottom sheet from their identity menu,
keeping the active writing route and draft mounted. Account settings reuses the same content on a
page. Both show one batch at a time, permit returning to earlier same-day batches and preview a
candidate before explicit confirmation. The one-year lock and confirmation button remain in the
bottom action area while explanatory text scrolls. Successful setup selects the returned persona
in the invoking composer; cancellation, failure and late responses after a session change do not.
Self-disabled identities retain a reactivation action; governance restrictions remain unavailable.
Opening the persona profile from the sheet leaves the writing field unfocused; returning through
normal cancellation or successful confirmation restores its focus.
The sheet follows the native component foundation's safe-area, keyboard-inset and width boundaries,
using [Flutter's modal-sheet semantics](https://api.flutter.dev/flutter/material/showModalBottomSheet.html).

The contextual entry and progressive disclosure follow
[NN/g guidance on contextual help](https://www.nngroup.com/articles/onboarding-tutorials/).

## Writing and public attribution

`Current`: topic and reply composers choose `member` or `persona`. A fresh composition defaults to
member; a continuation in the owner's anonymous topic defaults to that persona. Draft recovery retains
the chosen identity with its body. Captcha, review and failed submissions retain that choice.
A session change clears the previous composer's choice before loading the new account's draft.
An unavailable persona causes failure instead of silently switching to member. Published authors
cannot be changed by editing their body. Existing server drafts retain the original attribution.

`Current`: composer identity controls stay on one row at large text sizes. A long selected name
remains complete in the control's accessible text and name tooltip; settings and the public profile
display the full word. Loading failures offer retry without changing the selected identity or
expanding the identity row into the editor, mention candidates or sticker panel.

`Current`: topic lists project persona authors again at the final public payload boundary. Private
author IDs, names and avatars cannot pass through even if an upstream transform is omitted or the
public persona row is missing; member replies still retain their public attribution.
Personalized feeds do not recall persona content through its owner's follows or author affinity,
learn main-author affinity from persona topics, or label a persona as a followed main account.
Public category, freshness and quality recall remain available; account-level eligibility and
anti-abuse checks continue to use the private owner.

`Current`: persona authors expose `kind`, `publicUid`, name, avatar and profile URL, with numeric
author ID zero. Anonymous authors do not carry main-account badges, personal notes or profile fields.
Missing persona data yields an anonymous placeholder rather than a main-account lookup. Topic and
post payloads, replies and quotes, participants, revisions, last editor, SSR, JSON-LD, public exports,
Agent reads, notifications and webhooks follow this boundary. Avatar and attachment references contain
no numeric uploader ID; ordinary administrative file listings omit uploader identity.
Moderation approval summaries mark persona content anonymous and omit the private author, just as
they do for legacy anonymous replies.

`Current`: persona reply, mention and watched-topic notifications use the persona author and link.
Anonymous content's likes carry counts without a public liker identity; public main-account like
history excludes those targets. Eligibility and private blocking still use numeric account IDs.
One account cannot gain another vote or like by switching a composer identity, and persona content
cannot be liked by its own owner. Public main-account activity and badges exclude persona publishing.

`Current`: Web authenticated layouts load neither Umami analytics/session replay nor configured
injected scripts, because composer identity choices and audited reveal results are private. Native
analytics' public-screen allowlist excludes identity settings and persona profiles. Persona settings
and audit APIs return `private, no-store`; persona public views do not update main-account activity.

## Governance and retention

`Current`: readers and ordinary moderators cannot obtain the owner binding. Moderators can restrict
an anonymous author's account through a post in their category scope, with a required reason. The
private transaction disables the persona and the owner's writing together and records a governance
audit. Both member and persona topic/reply publishing are blocked. Restoration uses the same
moderator action and preserves independent account freezes. Self-disable and self-enable
retain the UID, name lock and occupied slot and cannot clear a governance restriction.

`Current`: reveal requires an explicit `anonymous.identity.reveal` permission (ID 7), a nonempty
reason and a committed restricted audit. Admin is not a wildcard for this permission. Audit failure
returns no identity. The response contains only persona UID, owner ID and username. The legacy Wiki
reveal endpoint rejects persistent personas. Revealed data is not copied into ordinary moderation
logs, exports, user cards or public caches.

`Current`: account closure blocks subsequent persona writes and retains the private binding, public
persona, name lock and restricted reveal/governance audits indefinitely. Historical content retains
the persona attribution and its normal deletion/visibility rules. This approved retention policy is
separate from the content recovery window. Operations and backup access are defined in
[anonymous identity operations](../operations/anonymous-identity.md).

These identities hide the main-account association, while deliberately linking one persona's own
history. Content, timing and external copies can still let readers infer a person. Course-review
anonymous display and legacy Wiki per-post anonymous replies remain separate and are not migrated.

## Sources

- [MADR 0074](../decisions/0074-six-character-persona-names.md) owns the current naming policy and carries forward the persona boundaries from [0066](../decisions/0066-persistent-anonymous-forum-persona.md).
- [Issue 1068](https://github.com/YourTongji/YourTJ-Hub/issues/1068) owns research and acceptance evidence.
- [Six-character name component source](../../apps/gooseforum/app/bundles/anonymousnames/README.md).
- [Contracts and data](../architecture/contracts-and-data.md#persistent-anonymous-personas).
