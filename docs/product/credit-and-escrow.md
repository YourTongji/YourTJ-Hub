# Points & Cross-Platform Settlement

> Doc type: product spec
>
> Status: Active (implementation `Partial`: forum-local ledger mechanics are `Current`, durable reward delivery is missing, and cross-platform credit is `Planned`)
>
> Owner: Product owner, Platform maintainers
>
> Last verified: 2026-08-06

## Positioning

Points (credit, linux-do) is a **cross-platform settlement center**, not a forum side feature. The forum,
Web, mobile, and future campus services (course selection, reviews, etc.) are all **merchants/consumers**
of the points system, sharing the forum identity (numeric users.id).

- The target cross-platform ledger's only source = credit (PostgreSQL): balances, transactions, transfers, red packets, orders,
  and merchant settlement all live in credit.
- After integration, the forum produces points **events** (post, reply, interact), settled via credit's
  merchant distribution API.
- credit ships a merchant model: API Key + signature (MD5/Ed25519), `/pay/distribute` for issuing,
  `/pay/submit.php` for collecting, transfers, leaderboards, gamification tasks.
- Points are a closed-loop virtual entitlement from contribution — **not a rechargeable, withdrawable,
  or freely transferable currency**.

## Key constraints (confirmed from source)

- credit is an OAuth2/OIDC **client**: it validates that the id_token `sub` is a numeric uint64.
  → The numeric-ID constraint comes from here (see identity-and-access.md).
- credit deployment: PostgreSQL 18+ / Redis 6+ / Go 1.26, `api`+`scheduler`+`worker` processes +
  Next.js frontend; it upserts a copy of users into its own DB (by IdP ID) — it is not a read-only proxy.
- Ban semantics: credit checks `active` at login; the forum server propagates its frozen/ban state.

## Integration shape (phase-2 draft)

```
Forum built-in OIDC Provider (numeric sub = users.id)
   ├── apps/gooseforum (forum)──┐
   ├── apps/mobile ─────────────┼──→ credit (points ledger)
   └── future campus services ──┘        ↑
                                  merchant distribution API (API Key + signature)
```

- Forum post/reply rewards → server calls `POST /pay/distribute` as a merchant → user `BalanceAdd`.
- User-to-user transfers → credit's own `POST /api/v1/payment/transfer`.
- In-service spending (badges/pins) → merchant creates an order → user pays → merchant receives.
- Reconciliation: credit read-only reconcile + per-wallet drift metrics (per YourTJ points ops experience).

## Current boundary

- **Forum-local ledger mechanics are `Current`**: accepted topic/reply events update the local
  `user_points` balance and visible `users.prestige`; source keys make rewards idempotent, reply deletion
  reverses its reward in the same transaction, and migration v14 reconstructs missing legacy balance rows.
- **Forum-local delivery is `Partial`**: reward events use the in-memory event bus, so process loss after
  content persistence can lose a reward; no durable outbox or reconciliation job currently repairs it.
- **Cross-platform credit is `Planned`**: services/credit holds only deployment config and README; it is
  not deployed or wired, and the local ledger is not presented as the future settlement source.
- No top-up/withdrawal/fiat exchange/free transfers.
- Cross-platform settlement events are designed after the forum business stabilizes, to avoid early coupling.
