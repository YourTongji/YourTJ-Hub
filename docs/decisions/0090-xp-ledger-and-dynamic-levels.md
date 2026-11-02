# XP ledger and dynamic levels

## Status
Proposed
Class: architecture

## Context and Problem Statement

Issue [#1149](https://github.com/YourTongji/YourTJ-Hub/issues/1149) defines reversible XP awards and
levels derived from current XP; its confirmed daily base is +5 XP without an activity requirement.
The existing forum-local `user_points`,
`points_record`, and `users.prestige` already serve legacy points behavior. Reusing those tables for
XP would conflate distinct balances and reward semantics; storing only a mutable XP total would lose
the audit trail needed to explain awards and corrections. Persisting a level would also allow it to
drift from the XP balance after a reversal.

This proposal records the XP core implemented on PR
[#1150](https://github.com/YourTongji/YourTJ-Hub/pull/1150), which remains subject to review. It does
not claim that the implementation is deployed or that the product's unconfirmed reward proposals are
active.

## Decision Drivers

- Preserve an auditable history for every XP change, including partial and full corrections.
- Make repeated delivery of the same business source safe and detect conflicting reuse.
- Keep the ledger, balance projection, and positive-award quota updates transactionally consistent
  under concurrent requests.
- Reconstruct which effective rule version and key produced an award, and attribute human operations.
- Derive the displayed level from current XP and current account eligibility, including after XP falls.
- Keep XP separate from legacy points and leave an unimplemented Credit or Mall domain without a
  second XP authority.

## Considered Options

1. **Extend `user_points` and `points_record` for XP.** Reuse the existing account and source-key
   patterns, with a field or type distinguishing XP. This minimizes tables but mixes the legacy points
   path and XP policy, complicating independent changes and risking accidental changes to existing
   rewards.
2. **Maintain an XP balance in place.** Apply awards and corrections directly to one mutable balance.
   This is simple to read and write but cannot explain the balance, prove idempotency from history, or
   safely restore it after an error.
3. **Store XP events only and sum them on every read.** An append-only event history is auditable and
   avoids projection drift, but every balance and level lookup must aggregate an ever-growing ledger.
4. **Use an independent append-only XP ledger with a balance projection.** Store immutable XP entries
   and update a non-negative current-balance projection in the same transaction. This adds write and
   schema complexity, while keeping reads bounded and history reconstructible.
5. **Edit one active reward rule set in place.** This is easy to administer but erases the policy that
   explains past awards and makes concurrent publication hard to reason about. An alternative is to
   append immutable, effective-dated versions and reference the applied version from each award.
6. **Persist a current level on the account.** Reads are direct, but every award, correction, and
   eligibility change must keep another mutable value synchronized. Alternatively, derive level from
   current XP and eligibility when needed.

## Decision Outcome

Choose an XP-owned ledger and projections within the forum. Keep the legacy points and prestige tables
and award path unchanged. The XP models are `xp_balances`, `xp_entries`, `xp_rule_sets`,
`xp_reward_rules`, `xp_daily_usage`, `xp_daily_rule_usage`, and the mutable coordination table
`xp_rule_publication_locks`.

`xp_entries` is append-only. A globally unique `source_key` makes one source settle at most once;
reusing it for a different account, rule key, or operation returns a conflict. An entry stores its account,
actor (zero for a trusted system operation), signed delta, event time, source key, rule-set version and
rule key, and optional reversal reference and reason. ORM guards and database triggers reject updates
and deletes. `xp_balances.current_xp` starts at zero and has a non-negative database constraint.

Awarding runs in one database transaction: serialize access to the account balance, check the existing
source key and account eligibility, select the enabled rule version effective at the event time, enforce
the event/rule/day limits, append the entry, update the per-user/per-rule-key and per-user daily-usage
projections, and update the balance. Any failure rolls those writes back. The event's UTC timestamp is
stored; its Asia/Shanghai calendar date determines daily quota usage. The immutable entry references
the exact rule-set version and key used; the versioned rule records retain the reward and limit values.

Corrections are negative append-only entries that must reference a positive award for the same
account. Partial reversals are allowed, but cumulative reversals cannot exceed the original award and
the balance cannot become negative. A reversal inherits the original rule version and key, uses its own
idempotency source key, and does not refund any daily count or positive-XP quota. Human reversals and
rule publication read the actor from a context set by server-side `WithAuthenticatedActor`; the caller
must resolve the authenticated identity and authorize the operation before setting it. Actor identity
is not accepted in the service request payload. Trusted system awards or corrections use actor ID zero.

Reward policy is immutable after publication. `xp_rule_sets` are effective-dated and numbered in
sequence; `xp_reward_rules` are keyed within a version. Rule keys are trimmed and validated before
publication. A new disabled version pauses rewards without rewriting history. PostgreSQL and SQLite
publishers serialize through a coordination row; SQLite retries bounded `BUSY`/`LOCKED` failures.
Failed publication returns version zero and leaves no partial ruleset. Domain rejections are classified
as terminal by `IsTerminal`; unclassified database and transport failures remain retryable.

The only seeded reward is `daily_base`: +5 XP, at most once per day. Its rule set is in
Asia/Shanghai and records the event/rule limits; the optional total daily-positive-XP limit remains
unset. Issue #1149 lists other event rewards and a 65 XP daily cap as proposals pending confirmation;
they are not made active by this decision. Although the +5 base award does not require user activity,
this PR only seeds the rule: it does not run a daily settlement job.

Level is derived by `LevelForXP(currentXP, eligible)` and is not stored as authoritative account state:

- Lv0 茶苗: account is not eligible for XP.
- Lv1 绿茶: eligible account with 0–249 XP.
- Lv2 白茶: 250 XP.
- Lv3 黄茶: 1,750 XP.
- Lv4 青茶: 6,250 XP.
- Lv5 红茶: 15,000 XP.
- Lv6 黑茶: 30,000 XP.

Eligibility reuses the forum account record and security configuration: the account must be human,
not frozen or deleted, and verified when email verification is enabled. #1149 also lists Tongji SSO
in its eligibility context. The campus identity binding (MADR 0031/0032/0034) is an optional campus
workbench connection, not a participation gate: the forum has no existing "SSO-verified" account
state or site setting to reuse, and the registration-level controls (signup switch, allowed email
domains, activation) already shape which accounts exist. Requiring a live campus binding for XP
would be a new product rule, so this service adds no SSO-specific gate; whether XP should require
one is a product-requirements question for #1149, to be answered before any XP consumer depends on
this check. Registration time is not an input. An XP decrease therefore lowers the computed level
without a separate level-write operation.

Schema migration registers the XP models; app migration v35 seeds the initial rule after the v34
public-topic excerpt backfill. Existing accounts
without an XP balance have zero XP. No history is backfilled, and neither legacy points nor
`users.prestige` are converted or modified. There is no XP HTTP API, business-event subscription, daily settlement worker,
or Web/Flutter XP UI in this scope. Future trusted event adapters must map anonymous personas to the
real owning numeric `users.id`; the XP service does not issue persona wallets.

XP remains separate from future cross-platform Credit balances and Mall orders. Credit owns its
eventual settlement ledger; XP does not reuse or mirror that balance. For decorations, Hub owns the
definition, entitlement, and equipment state and a future Mall must use that owner boundary, as recorded
in [0082](0082-hub-decoration-entitlement-ownership.md). This decision creates no Credit or Mall
tables or flows.

### Consequences

Each award performs several writes in one transaction, and concurrent rewards for one account share a
serialization point. The ledger and daily-usage tables grow over time; this implementation has no
retention or archival policy. A future event-delivery layer must validate the business fact and resolve
its real account before calling the service; this core does not supply an outbox, replay worker, or
historical backfill. The separation preserves the legacy points path but requires future Credit and Mall
integrations to cross their owners' service boundaries instead of sharing XP storage.

## Pros and Cons of the Options

- **Independent ledger plus projection (chosen):** offers an auditable, replay-safe source of truth and
  bounded balance reads. It requires transactional projection maintenance, account-level write
  serialization, and additional daily-usage storage.
- **Reuse legacy points tables:** shares an existing source-key pattern but couples XP to a separate
  product balance and risks changing existing forum reward behavior; rejected to preserve ownership
  and compatibility boundaries.
- **Mutable balance only:** has the lowest storage and query cost but cannot explain or safely correct
  the current total; rejected because auditability and reversals are required.
- **Ledger-only aggregation:** gives a single durable source but makes every balance/level read scan or
  aggregate history; rejected for the routine read path. The ledger could support a future projection
  rebuild, but this PR does not include a rebuild job.
- **Mutable rule rows:** simplifies editing but cannot explain which policy an old reward used; rejected
  in favor of immutable effective-dated versions.
- **Persisted levels:** makes reads cheap but introduces a second mutable truth that can drift after
  reversals or eligibility changes; rejected in favor of deriving level from XP and eligibility.

## Links

- [Issue #1149: XP, Credit and Mall product requirements](https://github.com/YourTongji/YourTJ-Hub/issues/1149)
- [PR #1150: XP ledger and dynamic levels](https://github.com/YourTongji/YourTJ-Hub/pull/1150)
- [Points and cross-platform settlement](../product/credit-and-escrow.md)
- [0082: Hub owns decoration entitlements and equipment](0082-hub-decoration-entitlement-ownership.md)
- [XP data models at the implementation commit](https://github.com/YourTongji/YourTJ-Hub/blob/4172bc1576c32a567b5cfd47d516fedf86bde4c6/apps/gooseforum/app/models/forum/xp/models.go)
- [XP service and transactional rules at the implementation commit](https://github.com/YourTongji/YourTJ-Hub/blob/4172bc1576c32a567b5cfd47d516fedf86bde4c6/apps/gooseforum/app/service/xpservice/service.go)
- [Dynamic level calculation at the implementation commit](https://github.com/YourTongji/YourTJ-Hub/blob/4172bc1576c32a567b5cfd47d516fedf86bde4c6/apps/gooseforum/app/service/xpservice/levels.go)
