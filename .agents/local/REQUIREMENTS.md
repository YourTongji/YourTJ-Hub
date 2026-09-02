# YourTJ-Hub Local AI Agent Requirements

> This file maintains local AI agent context and requirements for the YourTJ-Hub project.
> It is gitignored and will NOT be pushed to remote - for local AI agents only.

## Project Overview

**YourTJ Hub** is a community platform for Tongji University campus members. It's a course choosing forum that evolved from GooseForum, built with a single-binary deployment approach.

### Core Identity
- **Brand**: yourtj (distinct from archived YourTJ-Platform)
- **Target Users**: Tongji University students and campus members
- **Purpose**: Accumulate campus information, build trusted discussions, shared identity across services
- **Future Services**: Course selection, course reviews, etc.

### Tech Stack

| Domain | Technology |
|--------|------------|
| Backend | Go 1.26, Gin, GORM, Cobra |
| Web Frontend | Vue 3, TypeScript, Vite, Tailwind CSS, GoHTML |
| Mobile | Flutter, Dart, Melos, Riverpod |
| Database | PostgreSQL (production), SQLite (development) |
| Search | Meilisearch (optional, event-driven index) |
| Auth | Built-in OIDC Provider, GitHub OAuth, JWT, TOTP 2FA |
| Deployment | Single binary via `go:embed`, Docker Compose |

### Key Architecture Decisions

1. **Single Binary**: Vue build artifacts + GoHTML templates embedded in Go executable via `go:embed`
2. **Database is Truth**: Search, cache, counters are rebuildable projections
3. **Numeric User IDs**: Must be uint64 (credit system requirement)
4. **Identity Source**: Forum `users` table is the identity source
5. **OIDC Provider**: Built-in, issues numeric `sub` = users.id

## Repository Structure

```
apps/
  gooseforum/       Main forum (Go + Vue, single binary)
    app/           Go backend layers
      bundles/     Utilities
      models/      GORM models
      service/     Business logic
      http/        Controllers (API + GoHTML)
    resource/      Vue 3 frontend + GoHTML templates
  mobile/          Flutter mobile app (Partial)
packages/
  api-contract/    OpenAPI specs, fixtures, generated types
services/
  search/          Meilisearch config
  credit/          Points system (Phase 2, not implemented)
deploy/            Docker, environment configs
docs/              Product, architecture, development docs
.agents/           AI agent skills and local requirements
```

## Branch Naming Convention

- `dev` - Main development branch (create branches FROM here, PRs TO here)
- `main` - Production branch (merges via PR only)
- `feat/<topic>` - New features
- `fix/<topic>` - Bug fixes
- `docs/<topic>` - Documentation changes

**Never develop directly on `main` or `dev`!**

## Development Workflow

### Standard Process
1. Read AGENTS.md and relevant docs
2. Create branch from `origin/dev`
3. Assess change impact (backend/web/contract/migration/auth/search/deploy/docs)
4. Implement with proper layer boundaries
5. Run relevant tests and checks
6. Update documentation if needed
7. Commit/push/PR only when explicitly authorized

### Layer Boundaries
- **Business Logic**: `app/service`
- **Data Access**: `app/models`
- **HTTP Layer**: `app/http/controllers`

**No foreign SQL against other domains' tables!**

## Testing Requirements

### Backend Tests
```bash
cd apps/gooseforum
go vet ./...
go test ./...
```

### Migration Tests (PostgreSQL required)
```bash
YOURTJ_TEST_PG_URL="host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres sslmode=disable" \
go test ./app/migration/ -run 'TestSchema' -v
```

### Web Tests
```bash
cd apps/gooseforum/resource
pnpm typecheck
pnpm test
pnpm build
```

### Full Build
```bash
make build
# Output: bin/yourtj-hub
```

## Implementation Status Words

- `Current`: Fully implemented and verified
- `Partial`: Some layers done, must state what's missing
- `Planned`: Target formed but not delivered
- `Decision needed`: Awaiting owner decision, must not implement

## Current Feature Status

| Domain | Status |
|--------|--------|
| Forum | Current |
| Identity & Security | Partial |
| Search | Partial |
| Data & Files | Current |
| Content Governance | Current |
| Mobile | Partial |
| API Contract | Partial |
| Points (Credit) | Planned |

## Hard Constraints

1. Never commit `config.toml` (contains signing keys)
2. User IDs must be numeric (uint64)
3. No MySQL support (PostgreSQL/SQLite only)
4. Deployment is single binary (no nginx/CDN split)
5. Admin permissions are capability-based (UI hiding ≠ authorization)
6. New features require documentation updates

## Key Commands

```bash
# Start backend (default: http://localhost:5234)
make server

# Start frontend dev server (http://localhost:3010)
make web

# Start all dependencies (PostgreSQL, Meilisearch, etc.)
make dev

# Run tests
make test

# Build production binary
make build

# Pre-push hooks
make hooks
```

## Documentation Map

| Topic | Location |
|-------|----------|
| Project Overview | README.md |
| Agent Guidelines | AGENTS.md |
| Product Vision | docs/product/vision-and-principles.md |
| Current State | docs/product/current-state.md |
| System Architecture | docs/architecture/system-overview.md |
| Local Development | docs/development/local-development.md |
| Testing Strategy | docs/development/testing.md |
| PR Guidelines | docs/development/pull-requests.md |

## OpenAPI Contract Rules

- Contract center: `packages/api-contract/openapi.yaml`
- TypeScript types: generated in `resource/packages/client/src/contracts/`
- Dart types (Planned): `apps/mobile/packages/core/lib/src/gen/`
- Every `/api` route must be covered or in exclusion list
- Contract changes ship in same PR as implementation

## Git Discipline

- Stage only files this task owns
- Leave unrelated dirty/untracked files alone
- Commit messages: `feat:` / `fix:` / `docs:` / `refactor:` / `chore:`
- Never push to protected branches
- PR requires CI passing

## Common Gotchas

1. **Module Path**: After merging upstream, rewrite imports from `github.com/leancodebox/GooseForum` to `github.com/YourTongji/YourTJ-Hub/apps/gooseforum`

2. **MySQL Types Forbidden**: `bigint unsigned`, `datetime`, `tinyint` break PostgreSQL

3. **Design Tokens**: Changes to `resource/src/styles/tokens.css` require updating `apps/mobile/packages/ui_kit/lib/src/theme/tokens.json` in same commit

4. **Research Files**: Put in `research/` directory, not in git

5. **Config**: Never commit `config.toml` - it's gitignored for a reason!

## AI Agent Skills

Available in `.agents/skills/`:
- `$yourtj-development` - Main development workflow
- `$yourtj-pre-push-checks` - Pre-push verification
- `$yourtj-code-review` - Code review process
- `$yourtj-simplifications` - Code simplification
- `$yourtj-doc-standards` - Documentation standards
- `$forum-ai-readable-content` - Forum content access

## Notes for AI Agents

1. Always read AGENTS.md before making changes
2. Use the `$yourtj-development` skill for implementation work
3. Verify with actual commands, not just "it should work"
4. Keep contract, implementation, tests, and docs in sync
5. Follow existing code patterns and naming conventions
6. Ask for clarification if requirements are ambiguous
7. Never bypass the numeric user ID constraint

---

> Last updated: 2026-09-02
> This file is for local AI agent use only and will not be pushed to remote.