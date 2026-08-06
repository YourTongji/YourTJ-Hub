# Product Vision & Principles

> Doc type: product baseline
>
> Status: Active
>
> Owner: Product owner, Platform maintainers
>
> Last verified: 2026-08-06

yourtj is a community platform for Tongji campus members. The forum is the core public discussion space;
unified auth (Casdoor), search (Meilisearch), and points (credit, phase 2) are shared identity,
infrastructure, and settlement subdomains. The product goal is to accumulate campus information, build
trusted discussion, and let users share one identity and one points system between the forum and future
services (course selection, course reviews, etc.).

## User value

- Students use one account for the forum and all future campus services — no repeated registration.
- Forum content has clear boards, context, and a recoverable governance process; it does not degrade
  into an unstructured short-content stream.
- Site content is uniformly searchable (Chinese-friendly), and information accumulates long-term
  instead of drowning in a timeline.
- Contributions earn points that settle across platforms, but points are not a rechargeable currency.
- Mobile (iOS/Android) and Web share the same API and experience semantics.

## Product boundaries

### Current positioning

- `Current`: monorepo skeleton; the forum runs (three-mode rendering + JSON API, single binary).
- `Current`: database selection (PostgreSQL main-db support landed, issue #11; SQLite default retained);
  search shape via Meilisearch (optional, event-synced index). `Planned`: unified auth integration.
- Points (credit) are explicitly **phase 2**; not claimed usable in UI or marketing now.

### Explicitly out of scope

- Campus email / student ID are not exposed as public social identity (auth is separate from public
  identity, per the YourTJ principle).
- No points top-up, withdrawal, fiat exchange, or unrestricted transfers.
- Admins cannot impersonate users by rewriting their content through normal edit endpoints.
- No algorithm feeds, group chat, or complex ad targeting before the social graph, privacy, and
  governance foundations are stable.
- No nginx/CDN split deployment (single binary is a deliberate choice).

## Product principles

1. Auth's only source is Casdoor (once integrated); the forum JWT is a session credential, not identity truth.
2. User IDs must be numeric (uint64) — credit's `GetID()` only accepts numeric sub; UUID collapses all
   users to 0.
3. The chosen DB is the business fact source; search, cache, counters, and feeds are rebuildable projections.
4. Contracts center on `packages/api-contract/openapi.yaml` (once the pipeline exists); Web/mobile types
   are generated artifacts.
5. Deployment is a single binary (go:embed), source separated by directory, deployment merged.
6. All user media is managed via platform-controlled asset ids, states, and reference relations; no
   arbitrary external links as persistent facts.
7. Admin permissions are judged by capability; hiding a button in the frontend is not authorization.
8. Governance prefers reversibility: reasons, audit, user notification, and appealability are required.
9. Business lifecycles use explicit state machines; not ambiguous boolean combinations.
10. Critical side effects must be idempotent, retryable, observable; no unsupervised fire-and-forget.
11. Notification events, delivery channels, and user preferences are three independent models.
12. New data must answer purpose, visibility, retention, export, and deletion before persisting.

## Actors

| Actor | Main capabilities | Hard boundaries |
|---|---|---|
| Anonymous visitor | Read pages policy allows to be public | Cannot create content or relations |
| Campus member | Profile, post, comment, interact, follow, DM | Constrained by state, trust level, privacy, rate limits |
| Moderator | Content moderation, limited user actions, audit read | Cannot manage same/higher roles, cannot read arbitrary DMs |
| Admin | Platform config, user roles, structure & ops | Still bounded by reason, audit, and compliance red lines |
| System/service | Projections, scheduling, notifications, governance automation | Uses a distinct actor kind; never impersonates a human |

## Decisions pending product owner

| Decision | Recommended default | Impact if undecided |
|---|---|---|
| Database | PostgreSQL 15+ | Migration, queries, deployment |
| Search shape | Meilisearch standalone | Topology, index sync, Chinese tokenization |
| Anonymous visibility | Board-declared visibility | Search index, SEO, privacy settings |
| Login method | Casdoor unified login (OIDC), numeric ID | credit & mobile exchange integration |
| Points source | credit merchant distribution (forum event-driven) | Settlement model, anti-abuse, audit |

Undecided items may be researched but must not be irreversibly decided by a local UI or migration.

## Product health metrics

Metrics must serve community quality, not "longer dwell time at all costs". Suggested tracking: weekly
actives/registration conversion, post-to-reply ratio, report resolution time, search hit rate, points
reconciliation variance (phase 2). Concrete metrics are defined and recorded here when the core API ships.
