# Contracts, Data & Derived Projections

> Doc type: architecture
>
> Status: Active (contract pipeline `Partial`, not built)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## Contract status

- Upstream GooseForum has **no swagger annotations, no openapi.yaml**; JSON is serialized directly
  from Go structs.
- The frontend `@gooseforum/client` package (resource/packages/client) is hand-written TS contracts,
  manually kept in sync with Go structs.
- `packages/api-contract/openapi.yaml` is currently a placeholder (`paths: {}`).

## Contract pipeline (planned)

```
apps/gooseforum (Go structs + swag annotations, or manually maintained first)
   │  gen script
   ▼
packages/api-contract/openapi.yaml      ← CI checks: generated output has no diff
   │  gen-ts.sh                 │  gen-dart.sh
   ▼                           ▼
apps/gooseforum/resource/packages/client  apps/mobile/packages/core/lib/src/gen/*.dart
```

- **Backend is the fact source**: Go structs → openapi.yaml.
- **Web/mobile are generated artifacts**: types never silently drift; CI checks "generated output has
  no diff".
- **Fixture contract tests**: real API response samples; deserialization tests back runtime behavior.
- Transition path: since upstream has no annotations, use go-json-schema or progressive swag
  annotation (annotate each API as you touch it).

## Data model

- Migrations: upstream `app/migration` (Go migrations, run at startup/CLI); SQLite default, MySQL and
  PostgreSQL (main db, issue #11) supported; the file db stays SQLite.
- State machines: business lifecycles use explicit state machines (e.g. topic:
  draft/published/archived/deleted), not ambiguous boolean combinations (product principle 9).
- Soft/hard delete policy is decided with the database migration decision; record in the note.
- Search index sync is event-driven: topic publish/update/delete events keep Meilisearch documents in
  sync; the index is a rebuildable projection (`rebuild-search-index` CLI), not the only truth.

## Derived projections

| Projection | Source | Rebuildable |
|---|---|---|
| Search index | Meilisearch | ✅ full rebuild (rebuild-search-index CLI) |
| Counters (replies/likes) | DB aggregate or cache | ✅ recompute |
| Hot lists / feeds | derived queries | ✅ |
| Notification read/unread | user pointer table | ✅ |

Principle: projections must be rebuildable from the fact source; never treat a projection as the only truth.

## Contract change discipline

- Backend field changes ship in the same PR: Go struct → openapi.yaml → TS/Dart generated output →
  fixture (once the pipeline exists).
- Until the pipeline exists, at least keep `@gooseforum/client` manually in sync with Go structs and
  note it in the PR.
- Docs status words updated in step (docs/README.md).
