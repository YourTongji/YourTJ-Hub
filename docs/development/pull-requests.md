# Branches, Commits & Pull Requests

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## Branches

- Never develop directly on `main`; create `feat/<topic>` / `fix/<topic>` / `docs/<topic>` from `origin/main`.
- Prefer worktrees (`git worktree add`) for parallel tasks; do not mix branches in one checkout.

## Commits

- Conventional Commits: `feat:` / `fix:` / `docs:` / `refactor:` / `chore:` / `test:`.
- Stage only files this task owns; leave unrelated dirty/untracked files alone.
- Never push to protected branches; releases go through PR + CI.
- Agent commits must carry the footer:
  `Co-authored-by: synergy-agent <299070056+synergy-agent@users.noreply.github.com>`

## Pull Requests

- PR description states: motivation, behavior change, verification (commands + results),
  documentation/contract impact, known gaps.
- Contract-changing PRs must include generated-output diffs and fixture updates.
- Do not merge your own PR (unless a solo repo and explicitly allowed); at least one review.

## Forbidden

- No `push --force` to shared branches; never change git config.
- No local paths, secrets, logs, or internal addresses in commits/PRs/comments.
- No merging production deployments or external changes (unless the user explicitly asks).
