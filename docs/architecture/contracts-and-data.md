# Contracts, Data & Derived Projections

> Doc type: architecture
>
> Status: Active (contract pipeline `Partial`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-08

## Contract status

The contract capability is **Partial**. The controlled OpenAPI 3.1 entry point is
`packages/api-contract/openapi.yaml`; it currently covers these operations only:

- `POST /api/login`;
- `POST /api/auth/oidc/exchange`;
- `POST /api/forum/topics/write`.

The first coverage intentionally describes the current legacy wire behavior. A business failure commonly
uses HTTP `200` with `{ "code": 1, "result": null, "messageCode": ... }`; consumers must inspect the
JSON envelope rather than treating every 2xx status as application success. Middleware-owned failures
such as unauthenticated access, frozen or unresolvable authenticated accounts, and rate limiting remain
HTTP `401`, `403`, and `429` with the same failure envelope. Topic write's current permissive `UpButterReq`
wrapper reports malformed or incomplete JSON as
an HTTP `200` validation failure, not a guaranteed `400`.

## Contract pipeline (Partial)

```
Go controller, route wrapper, and Gin middleware (current behavior)
   │  manually maintained operation descriptions
   ▼
packages/api-contract/openapi.yaml + paths/ + components/
   │  Redocly lint and bundle             │  openapi-typescript
   ▼                                      ▼
packages/api-contract/fixtures/      @gooseforum/client/openapi types
   │                                      │
   └──── route-level httptest assertions ──┴──── CI generated-output no-diff gate
```

- **Go behavior is the fact source**: controllers, route wrappers, middleware, and their `httptest`
  coverage establish the behavior being documented.
- **OpenAPI is the controlled protocol source**: each covered operation documents that behavior in a
  reviewable, consumable format. It is split into entry point, paths, and schemas but linted and bundled
  as one contract.
- **Generated Web types are artifacts**: `@gooseforum/client/openapi` exports only the generated OpenAPI
  types under `src/gen/`; it does not replace the existing hand-written page payload contracts or create
  a request client. CI regenerates the types and rejects an uncommitted diff.
- **Fixtures are representative wire samples**: committed JSON fixtures capture stable envelope shape,
  message codes, and rate-limit metadata. The route-level Go tests exercise real Gin route chains and
  assert the actual status, envelope, result shape, and `Retry-After` behavior against those fixtures.
- **Mobile/Dart generation is Planned**: no Dart generator or generated mobile artifact is maintained by
  this repository yet.

Breaking-change comparison is not a current gate. The `dev` base before this first coverage contains no
stable operations to compare, so a snapshot baseline would be redundant and misleading. Enable a
base-versus-head bundled-spec breaking gate in a separate change only when `dev` has stable operation
coverage on both sides of the comparison; that gate must compare the PR base and head contracts rather
than a hand-maintained duplicate baseline.

## Data model

- Migrations: upstream `app/migration` (Go migrations, run at startup/CLI); SQLite default, MySQL and
  PostgreSQL (main db, issue #11) supported; the file db stays SQLite.
- State machines: business lifecycles use explicit state machines (e.g. topic:
  draft/published/archived/deleted), not ambiguous boolean combinations (product principle 9).
- Soft/hard delete policy is decided with the database migration decision; record in the note.
- Search index sync is event-driven: topic publish/update/delete events keep Meilisearch documents in
  sync; the index is a rebuildable projection (`rebuild-search-index` CLI), not the only truth.

## Task queue & background workers

- `task_queue` rows carry a `type` string; workers poll by **type prefix** so task types never leak
  across handlers:
  - `email.*` (activation/reset_password; legacy `activation` / `reset_password` rows are whitelisted)
  - `export` (data export)
  - `file-migrate` (BLOB → object storage migration)
- Export and migration tasks update `task_json` with progress payloads (`processed/total/errorCount`,
  cursor `lastId`) so the admin panel can render live progress and resume after restarts.
- Export files land in `data/export/` and are retained 7 days (daily cron cleanup).

## Config-driven features (pageConfig)

New page config types added with this admin backlog work (all persisted in `page_config`, cached in
`hotdataserve`, editable from the admin panel):

| pageType | Struct | Purpose |
|---|---|---|
| `storageSettings` | `pageConfig.StorageSettings` | provider local/s3, endpoint/bucket/region, bucket lookup (auto/dns/path), credentials, optional public URL prefix |
| `termsOfService` | `pageConfig.TermsOfServiceConfig` | markdown ToS rendered at `/terms` |
| `securitySettings` (extended) | `pageConfig.SecurityAndRegistration` | + reservedUsernames / bannedUsernames / sensitiveWords / sensitiveAction (block\|review) |

Object storage addressing notes: Alibaba OSS and Tencent COS (buckets created after 2024-01-01)
require virtual-hosted style — use `bucketLookup: dns` with an explicit region; MinIO/R2 accept
`auto`/`path`. Endpoint may include a scheme; it is stripped before building the minio-go client
(`Secure` is derived from the scheme).

## Derived projections

| Projection | Source | Rebuildable |
|---|---|---|
| Search index | Meilisearch | ✅ full rebuild (rebuild-search-index CLI) |
| Counters (replies/likes) | DB aggregate or cache | ✅ recompute |
| Hot lists / feeds | derived queries | ✅ |
| Notification read/unread | user pointer table | ✅ |

Principle: projections must be rebuildable from the fact source; never treat a projection as the only truth.

## Contract change discipline

- For an OpenAPI-covered operation, backend behavior, `openapi.yaml`, generated TypeScript output, and
  representative fixtures/route-level contract tests ship in the same PR.
- Operations outside the current coverage remain manually synchronized with their consumers until they
  are added to the controlled contract. `@gooseforum/client` must stay in sync with Go structs.
- The mobile client mirrors contracts into `apps/mobile/packages/core/lib/src/gen/*.dart` (a
  generated-artifact placeholder until the OpenAPI pipeline lands). Backend/TS contract changes that
  affect the mobile surface must update the Dart mirrors in the same PR; fixture contract tests
  (`core/test/fixtures`) back runtime deserialization.
- Dart generation and full-route coverage are Planned; do not claim they are current contract gates.
- Docs status words updated in step (docs/README.md).
