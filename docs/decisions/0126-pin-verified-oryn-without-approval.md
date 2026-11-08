# Pin the verified Oryn runtime and keep low-risk approval off

## Status

Superseded by [0127](0127-oryn-low-risk-approve-sensitive-prefixes.md)
Class: process

## Context and Problem Statement

YourTJ-Hub runs Oryn from an immutable checkout of the private runtime, then applies
[repository policy](../../.github/oryn/repositories.json) from the trusted default branch. Maintainers
need the independently verified multi-review dispatch in that runtime, and they need review lessons to
come from the decision and postmortem records that already exist. The same runtime can publish a
low-risk `APPROVE` when policy allows it. That publication must stay off. Live GitHub rulesets, App
permissions, required human reviews, and merge automation must stay as they are.

## Decision Drivers

- Pin one immutable commit that an independent reverify marked ready, not a floating branch.
- Keep `allowLowRiskApprove` false.
- Bind the existing required check to the GitHub App that produces it.
- Read lessons from `docs/decisions` and `docs/postmortems`. Do not add a lessons tree or another store.
- Leave `allowMerge`, protected merge paths, rulesets, App installations, and auto-merge unchanged.

## Considered Options

1. Pin the verified commit, set `allowLowRiskApprove` false, record `trustedCheckApps` for
   `ci-required` / `github-actions`, and set `lessonPaths` to the existing decision and postmortem
   directories.
2. Pin that commit and set `allowLowRiskApprove` true.
3. Keep the previous pin until the runtime commit is the repository default branch.
4. Reimplement multi-review inside Hub review skills.

## Decision Outcome

Choose option 1. The [setup action](../../.github/actions/setup-oryn/action.yml) checks out
`ad2b9847c01cba36f8a0abd5564df7144d47878e`. Policy for `YourTongji/YourTJ-Hub` sets
`allowLowRiskApprove` to false, keeps `requiredChecks` as `ci-required`, keeps `allowMerge` true, and
keeps `protectedMergePaths` as `releases/requests/`. It sets `trustedCheckApps` to the check name
`ci-required` and the App slug `github-actions`. That pair is the successful `ci-required` run on the
head of merged pull request 1591. It sets `lessonPaths` to `docs/decisions` and `docs/postmortems`,
which is also the runtime default, so the policy documents the reuse explicitly. It does not set
`sensitivePaths`.

GitHub rulesets, branch protection, the required reviewer count, CODEOWNERS, App installations,
secrets, and production deploy stay outside this policy. Auto-merge is not enabled. A bot `APPROVE`
does not replace a required human review or a ruleset. Confirmed regressions that already meet the
postmortem criteria use the existing five-section template. Ordinary bug fixes do not require a
postmortem.

## Pros and Cons of the Options

- Option 1 gives operators the verified dispatch runtime and an explicit closed approval default. It
  does not prove live model quality, a real GitHub dismissal, or App write rights.
- Option 2 would publish approvals while those live gaps are still unverified, and it would open a
  gate this change is required to leave closed.
- Option 3 leaves Hub on a runtime without the verified dispatch. The verified commit is not the
  runtime repository's default branch, and a floating `main` ref is not an acceptable pin.
- Option 4 copies review behavior into a second system and leaves the deployed runtime behind.

## Links

- [Operations guide](../operations/oryn.md)
- [Pinned setup action](../../.github/actions/setup-oryn/action.yml)
- [Repository policy](../../.github/oryn/repositories.json)
- [Postmortem template](../postmortems/README.md)
- [Repository-local Actions decision](0015-oryn-repository-local-actions.md)
- [Oryn Mini pull request 19](https://github.com/yzxoi/oryn-mini/pull/19)
- [Verified runtime commit](https://github.com/yzxoi/oryn-mini/commit/ad2b9847c01cba36f8a0abd5564df7144d47878e)
- [Successful ci-required run on merged pull request 1591](https://github.com/YourTongji/YourTJ-Hub/actions/runs/42234805308/job/127955969932)
- [Low-risk approval decision](0127-oryn-low-risk-approve-sensitive-prefixes.md)
