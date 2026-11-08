# Enable low-risk Oryn approval and name Hub sensitive prefixes

## Status

Accepted
Class: process

## Context and Problem Statement

[0126](0126-pin-verified-oryn-without-approval.md) pins Oryn at
`ad2b9847c01cba36f8a0abd5564df7144d47878e` and leaves `allowLowRiskApprove` false. That runtime can
publish `APPROVE` only when repository policy allows it, the review is low risk, and the changed
paths are not sensitive. Maintainers now want that publication for low-risk pull requests. Medium
and high risk, unknown risk, and sensitive paths must stay comments. The required team review,
rulesets, and auto-merge stay as they are.

## Decision Drivers

- Use the pinned runtime's `RepositoryPolicy` fields. Do not add a second approval store.
- Turn on `allowLowRiskApprove` only together with prefixes for Hub paths the matcher would otherwise
  miss.
- Keep `requiredChecks`, `trustedCheckApps`, `lessonPaths`, `allowMerge`, and `protectedMergePaths`.
- A change to this policy must itself be sensitive.
- Do not change GitHub rulesets, App installations, or merge automation.

## Considered Options

1. Set `allowLowRiskApprove` true and add Hub prefixes that `isSensitivePath` treats as
   `startsWith` matches, leaving paths the pinned matcher already marks sensitive out of that list.
2. Set `allowLowRiskApprove` true with an empty `sensitivePaths` list.
3. Keep `allowLowRiskApprove` false.

## Decision Outcome

Choose option 1. This supersedes the closed-approval outcome in
[0126](0126-pin-verified-oryn-without-approval.md). The pin, check binding, lesson paths, `allowMerge`,
and `protectedMergePaths` from that record stay.

`allowLowRiskApprove` is true. `sensitivePaths` lists Hub prefixes for Oryn policy and CI
(`.github/oryn/`, `.github/workflows/`, `.github/actions/`, `scripts/ci/`) and for auth, session, OIDC,
JWT, passkey, TOTP, and permission paths whose names do not split into the matcher's control tokens.
`releases/requests/` stays only in `protectedMergePaths`. The runtime checks `sensitivePaths` and then
`protectedMergePaths` with the same prefix rule. Schema migrations under
`apps/gooseforum/app/migration/` and the `deploy/` tree are already sensitive because their path
tokens match the pinned matcher, so this policy does not repeat them.

A bot `APPROVE` still does not replace the team review required by `dev-protect` and `main-protect`,
and it does not merge. Auto-merge stays off.

## Pros and Cons of the Options

- Option 1 lets a clean low-risk review become `APPROVE` while CamelCase auth paths and CI scripts
  stay comments. It adds a long prefix list because the matcher does not split camel case.
- Option 2 would approve changes to `scripts/ci/` and to OIDC, JWT, and session files that the
  current token split does not see.
- Option 3 avoids a wrong prefix, but it drops the approval maintainers have now authorized.

## Links

- [Closed-approval pin](0126-pin-verified-oryn-without-approval.md)
- [Operations guide](../operations/oryn.md)
- [Repository policy](../../.github/oryn/repositories.json)
- [Oryn Mini pull request 19](https://github.com/yzxoi/oryn-mini/pull/19)
- [Verified runtime commit](https://github.com/yzxoi/oryn-mini/commit/ad2b9847c01cba36f8a0abd5564df7144d47878e)
- [Merge commit](https://github.com/yzxoi/oryn-mini/commit/fa78f2e714b4662f89640e6e7e1594e411381df4)
