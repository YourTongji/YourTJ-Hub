# Product Vision & Principles

> Doc type: product baseline
>
> Status: Active
>
> Owner: Product owner, Platform maintainers
>
> Last verified: 2026-08-07

yourtj is a community platform for Tongji campus members. The forum is the core public discussion space;
unified auth (built-in OIDC Provider), search (Meilisearch), and cross-platform points (credit, `Planned`) are shared identity,
infrastructure, and settlement subdomains. The product goal is to accumulate campus information, build
trusted discussion, and share an identity across the forum, courses, reviews and campus services.
Cross-platform points settlement remains `Planned`.

## User value

- Students use one account for the forum, courses, Wiki and private campus workspace.
- Forum content has clear boards, context, and a recoverable governance process; it does not degrade
  into an unstructured short-content stream.
- Site content is uniformly searchable (Chinese-friendly), and information accumulates long-term
  instead of drowning in a timeline.
- Contributions earn forum-local points; cross-platform settlement is `Planned`. Points are not a rechargeable currency.
- Mobile (iOS/Android) and Web share the same API and experience semantics.

## Product boundaries

### Current positioning

- `Current`: monorepo skeleton; the forum runs (three-mode rendering + JSON API, single binary).
- `Current`: PostgreSQL is the deployment default; SQLite remains the local development/test default;
  search shape via Meilisearch (optional, event-synced index).
- `Current`: unified auth integration (built-in OIDC Provider; forum users are the identity source).
- `Planned`: cross-platform credit settlement; forum-local ledger and delivery limits are described
  in the [points specification](credit-and-escrow.md).

### Explicitly out of scope

- Campus email / student ID are not exposed as public social identity (auth is separate from public
  identity, per the YourTJ principle).
- No points top-up, withdrawal, fiat exchange, or unrestricted transfers.
- Admins cannot impersonate users by rewriting their content through normal edit endpoints.
- No algorithm feeds, group chat, or complex ad targeting before the social graph, privacy, and
  governance foundations are stable.
- No nginx/CDN split deployment (single binary is a deliberate choice).

## Product principles

1. The forum `users` table is the identity source; the built-in OIDC Provider authenticates first-party clients against it; the forum JWT is a session credential, not identity truth.
2. User IDs must be numeric (uint64) — credit's `GetID()` only accepts numeric sub; UUID collapses all
   users to 0.
3. The chosen DB is the business fact source; search, cache, counters, and feeds are rebuildable projections.
4. For OpenAPI-covered operations, contracts center on `packages/api-contract/openapi.yaml`; Web
   TypeScript types are generated artifacts, while mobile type generation remains Planned.
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

## Decisions and implementation boundaries

Database, search and identity choices are documented in the [system architecture](../architecture/system-overview.md)
and [decision log](../decisions/README.md). They are not pending product selections.

`Decision needed`: MFA for GitHub/Google/Tongji and built-in OIDC login paths; see
[identity and access](identity-and-access.md).
`Decision needed`: whether anonymous visitor visibility is declared per board; its search and privacy
semantics must be specified before changing access rules.
`Planned`: cross-platform credit integration; its target merchant model does not imply a deployed service.

## Product health metrics

Metrics must serve community quality, not "longer dwell time at all costs". Suggested tracking: weekly
actives/registration conversion, post-to-reply ratio, report resolution time, search hit rate, points
reconciliation variance after cross-platform settlement exists. These are proposed measurements, not
claims that the corresponding dashboards or data pipelines are implemented.
