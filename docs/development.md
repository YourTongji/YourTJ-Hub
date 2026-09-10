# Development

Contributor setup, the standard workflow, and the executable process live in
[docs/development/README.md](development/README.md); local environment,
service addresses, and runtime configuration live in
[docs/development/local-development.md](development/local-development.md). This file only orients
newcomers and names the runtime prerequisites.

## Prerequisites

- Git
- Go 1.26+ (backend), Node.js 24 + pnpm 11 (web), Flutter (mobile workspace via melos), and
  Docker Compose for local PostgreSQL / Meilisearch when needed. Version details and service setup:
  [local-development.md](development/local-development.md).

## Daily workflow

1. Follow [docs/development/README.md](development/README.md): branch from `origin/dev`, keep
   implementation, contract, tests, and docs in sync, and honor the `AGENTS.md` hard constraints.
2. Run the governance gates early — they are fast and catch doc/manifest drift before review:
   ```sh
   node scripts/run-gates.mjs
   ```
3. Run the verification relevant to the change (root [`AGENTS.md`](../AGENTS.md) §4 owns the command
   list: `make test` for the default suite, focused `go test`/`pnpm test` during iteration, and the
   PostgreSQL migration tests for any model/migration change).

Commit messages and PR discipline follow
[docs/development/pull-requests.md](development/pull-requests.md).

## Source attribution

When an implementation is materially derived from a paper, article, community post, benchmark, research report, or copied/adapted code, preserve that provenance at the closest stable repository location:

- For a local algorithm, formula, constant, workaround, or behavior, add a nearby `Source:` comment with a descriptive title, a stable URL or DOI, and what the implementation derived from it.
- For a cross-cutting design, link the implementation entry point to the owning decision record and list the external sources in that record's `## Links` section.
- For generated, vendored, copied, or adapted material, retain the source header or metadata and satisfy the applicable copyright, license, and NOTICE requirements; a citation does not replace license compliance.
- For mutable community pages, include the relevant version or section and an access date when it helps future readers recover the cited evidence.

Routine language idioms and standard-library usage do not need citations. A pull request, issue, prompt, or chat transcript may supplement repository provenance but is never its only home because it can become detached from the implementation.

Example:

```text
// Source: "Exponential Backoff and Jitter" — https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
// Derived: full jitter prevents synchronized retries.
```

## Working tree

Governance files tracked in `.repo-seed/manifest.json` (CLAUDE.md, docs/AGENTS.md, this file,
`scripts/` gates, `.agents/skills/repo-*`, `.repo-seed/`) are owned by the repo-seed upgrade
channel. Hand-edit them with intent; the manifest records their hash and a re-run preserves your
edits. The repository's own docs center (`docs/`), root `AGENTS.md`, `CONTRIBUTING.md`, and repo
skills remain the authoritative surfaces for project content.
