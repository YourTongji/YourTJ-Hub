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
random 128-bit public UID, an independently generated fixed Beam avatar and a read-only `/a/:publicUid`
profile. It cannot log in, receive private messages, follow people, hold a separate wallet or gain
permissions. Numeric users remain the authentication, moderation, posting quota and credit subjects.
The anonymous profile links only its public forum topics and replies, with their visible counts.
Main profiles, Following, participation records and public author statistics exclude persona content.

`Current`: names are exact words from all eleven embedded THUOCL files at commit
`a30ce79d895d01ab5132a5c74c29703ff7efb4cc`. The pool contains 156,289 distinct words; identical
records are deduplicated. Length, characters, category, frequency and meaning do not filter this pool.
There is no nickname validation, truncation or generated suffix. Different personas can have the
same name; their UIDs, avatars and profiles distinguish them. A name confers no official badge.

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

## Writing and public attribution

`Current`: topic and reply composers choose `member` or `persona`. A fresh composition defaults to
member; a continuation in the owner's anonymous topic defaults to that persona. Draft recovery retains
the chosen identity with its body. Captcha, review and failed submissions retain that choice.
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

- [MADR 0066](../decisions/0066-persistent-anonymous-forum-persona.md) records the alternatives.
- [Issue 1068](https://github.com/YourTongji/YourTJ-Hub/issues/1068) owns research and acceptance evidence.
- [THUOCL provenance and license](../../apps/gooseforum/app/bundles/anonymousnames/README.md).
- [Contracts and data](../architecture/contracts-and-data.md#persistent-anonymous-personas).
