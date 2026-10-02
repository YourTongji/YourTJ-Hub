# Branches, Commits & Pull Requests

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-10

## Branches

- `dev` is the main development line: create `feat/<topic>` / `fix/<topic>` / `docs/<topic>` from
  `origin/dev`, open PRs against `dev`. CI builds and auto-deploys `dev` to the test instance.
- `main` contains reviewed production source. Promote dev with an ordinary PR; release preparation
  never merges it implicitly. [Release requests](../operations/releases.md) are the explicit branch
  exception: `codex/release/<id>` starts from main and targets main with only its release metadata.
  Final-head human review and merge trigger approved platform publication. Never develop directly
  on main or dev.
- The dev instance syncs a consistent snapshot of the main database on each deploy (see
  `docs/operations/deployment.md`), so DB migrations are rehearsed on dev before reaching main.
- Prefer worktrees (`git worktree add`) for parallel tasks; do not mix branches in one checkout.

## Commits

- Conventional Commits: `feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`.
- Stage only files this task owns; leave unrelated dirty/untracked files alone.
- Never push to protected branches; releases go through PR + CI.

## Pull Requests

- PR description states: motivation, behavior change, verification (commands + results),
  documentation/contract impact, known gaps.
- Contract-changing PRs must include generated-output diffs and fixture updates.
- Do not merge your own PR (unless a solo repo and explicitly allowed); at least one review.

## Forbidden

- No `push --force` to shared branches; never change git config.
- No local paths, secrets, logs, or internal addresses in commits/PRs/comments.
- No merging production deployments or external changes (unless the user explicitly asks).
