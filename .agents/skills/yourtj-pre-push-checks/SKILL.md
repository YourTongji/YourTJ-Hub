---
name: yourtj-pre-push-checks
description: Use before pushing, force-pushing, marking ready for review, or claiming checks pass on a yourtj-hub branch. Selects the smallest tests and checks that cover the outgoing diff without reflexively running the full repository suite. Not for read-only analysis or implementation — route those to $yourtj-development.
---

# yourtj Pre-Push Checks

Use this skill to run the relevant local evidence once before pushing a yourtj-hub branch. Git hooks are
intentionally narrow: pre-commit checks staged whitespace and gofmt; pre-push runs `go vet` + `golangci-lint`
(incremental against `origin/dev`, full fallback) + `pnpm typecheck`. CI owns exhaustive coverage and the
platform matrix.

## Inspect the outgoing change

1. Confirm the checkout and branch:

```sh
git status --short --branch
git rev-parse --show-toplevel
```

2. Inspect the scope against the verified base:

```sh
git diff origin/dev...HEAD --stat
```

Use the live PR base when it differs from `origin/dev`; never guess or fetch a base. After a base merge or
rebase, rerun the diff and reassess which checks the combined scope invalidates.

## Select relevant evidence

Follow the evidence selection table in `$yourtj-development` §5: backend → focused `go vet`/`go test`
(package-level, `-run` filter when practical); model/migration → the mandatory PG migration gate; frontend →
`pnpm typecheck` + affected component tests; contract → `make contract-check`; docs → `git diff --check` +
link/status-word check; cross-cutting or CI diagnosis → full `make test` / `make build`.

There is no universal local baseline beyond the hooks. Add broader checks only for surfaces the diff actually
reaches; do not manually repeat a passing check merely because a commit or push follows.

## Run the hooks

`make hooks` installs lefthook for the current worktree. The pre-push hook runs automatically on push — do
not run typecheck manually immediately before pushing solely to duplicate it. To exercise the hook set
manually:

```sh
lefthook run pre-push
```

## Handle failures

If a relevant check fails before an ordinary push, stop and fix or explain the blocker. Do not push and hope
CI differs. For a failure that looks environment-specific, record the exact command, failing test, and
platform mismatch, confirm the relevant non-platform evidence, and prefer fixing cross-platform
nondeterminism. Bypass a local hook only when the user explicitly asks, and report exactly what failed and
why CI is expected to differ.

## Push procedure

1. Run the selected relevant checks once.
2. Commit normally; inspect any files changed by the pre-commit fixer before continuing.
3. Push normally so the pre-push hook runs.
4. Verify the remote ref matches local `HEAD`:

```sh
git rev-parse HEAD origin/$(git branch --show-current)
```

5. Inspect remote CI after the push: `gh pr checks`. Report pending checks as pending; inspect failures
   before attributing them to the branch or the environment.

## Force-push discipline

Standalone and stacked PR branches may rebase, including after review. Before a history rewrite, fetch the
current remote branch and record its exact OID, then publish with
`--force-with-lease=<branch>:<observed-oid>` so a concurrent update aborts the push. Raw `--force` is never
allowed. After any rewritten push, fetch the live heads again and re-audit unresolved review threads,
approvals, mergeability, and checks — commit hashes and inline-comment anchors from before the rewrite are
not current evidence.
