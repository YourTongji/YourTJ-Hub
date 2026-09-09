# Oryn repository maintenance

> Doc type: operations guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-09

Implementation status: **Current** for the configured integration. Live run results are recorded in
Actions; a green preflight proves App access and planning, while an executed review proves model access.
Repair publication additionally requires sandboxed application validation and an independent review.

## Execution and policy

[Oryn workflow](../../.github/workflows/oryn.yml) runs on this repository's Actions runners. The trusted
[setup action](../../.github/actions/setup-oryn/action.yml) loads
[Oryn Mini at an immutable commit](https://github.com/yzxoi/oryn-mini/tree/e36e678fae9a4d3e173750d9ce583c9140235b10)
and applies [this repository's policy](../../.github/oryn/repositories.json). Oryn's public Synergy core
creates a fresh temporary home per invocation; no server, database or reusable model history is deployed.
GitHub comments contain bounded queue receipts; Actions artifacts expire after seven days.

Manual runs use the selected workflow commit. Issue/PR events load the current default branch (`dev`),
never the contributor's PR head as executable workflow or operator policy. The model sees target source
as evidence. The model is `glm-5.3-flash` through the official Zhipu Coding Plan endpoint, with
`reasoning_effort=max`, thinking enabled, image input, and a configured 1,000,000-token context window.
The context setting is not a one-million-token capacity benchmark. The default task budget is 1800
seconds, manually overridable within 10–3000 seconds.

All Oryn policy capabilities are enabled: reviews, triage, source-backed Mermaid diagrams, emoji labels,
questions, stop/resume, repair, adoption, rebase, clusters, automatic small-bug implementation, close and
merge. A scan selects at most 20 eligible items; later scans pick up the remaining items after cooldown
receipts exclude completed work. Two model jobs run concurrently. Close/merge still require a current
maintainer command and live evidence; merge additionally requires independent approval and successful
`ci-backend`, `ci-frontend`, and `ci-contract` checks. Generated fixes remain draft PRs for human review.

The trusted [validation entry](../../.github/oryn/validate.mjs) is installed outside the candidate
checkout and mounted read-only in Oryn's credential-free bubblewrap sandbox. It compares the working
tree with the job's exact base SHA, including staged and untracked changes. Every repair runs governance
checks, then affected Go vet/test/build, PostgreSQL migration/model tests, Vue typecheck/test/i18n/build,
API contract checks or Flutter analysis/tests. Unknown executable paths select all domains. Linux
runners install Go, Node/pnpm and Flutter before repair execution; PostgreSQL uses a disposable local
service. Flutter's writable SDK cache lives in the temporary validation home. Validation failures or
budget exhaustion prevent patch publication; browser layout tests and other full CI suites remain
required wherever existing repository CI selects them. Oryn cannot repair its own `.github` policy.

## Register and install the App

Create a GitHub App under the operator's account, for example `oryn-yourtj` if available.
Use this repository URL as its homepage. Disable webhooks: native Actions events supply intake;
no callback URL, OAuth user authorization, webhook receiver or client secret is needed.
Choose installation on **Any account** because the runtime and target belong to different owners.
This allows cross-account installation; it does not publish private repository source or require a
Marketplace listing.

Repository permissions:

| Permission | App grant | Use |
| --- | --- | --- |
| Metadata | Read | Repository metadata and authority checks |
| Contents | Read and write | Read source; supports the separately gated repair publisher |
| Issues | Read and write | Issue reports, labels and operational receipts |
| Pull requests | Read and write | PR context, reports and separately gated draft publication |
| Actions | Read | Exact-head failure steps and bounded logs |
| Checks | Read | CI results |
| Commit statuses | Read | Commit status context |

No organization permissions, Administration, Secrets, or Workflows write permission is needed.
Install with **Only select repositories** in both accounts:

- `YourTongji`: select only `YourTJ-Hub`.
- `yzxoi`: select only `oryn-mini`, currently private. Each setup job requests a separate Contents-read
  token limited to this source repository; it never uses a target token to fetch the runtime.

Generate a private key. Copy the **Client ID** and store the PEM as a secret, not a file in Git or chat.
Installation tokens are minted independently per job, narrowed to the repository and required
permissions, and revoked at job completion. The model execution job's target token is read-only;
its core child has no GitHub token. Tokens and private keys are never passed in artifacts or job outputs.
The maximum task budget is 50 minutes; the target token is minted after runtime installation so its
one-hour lifetime covers the task. The publisher receives a new token.

## Repository configuration

In YourTJ-Hub → Settings → Secrets and variables → Actions, add:

| Kind | Name | Value |
| --- | --- | --- |
| Variable | `ORYN_APP_CLIENT_ID` | The App Client ID (`Iv...`), not its installation ID |
| Secret | `ORYN_APP_PRIVATE_KEY` | Entire generated PEM private key |
| Secret | `ORYN_GLM_API_KEY` | Zhipu Coding Plan API key; another repository's secret is not inherited |

Optional overrides: `ORYN_MODEL` (default `oryn/glm-5.3-flash`), `ORYN_BASE_URL`
(default `https://open.bigmodel.cn/api/coding/paas/v4`), `ORYN_TASK_TIMEOUT_SECONDS` (default 1800),
`ORYN_REQUEST_TIMEOUT_SECONDS` (core adapter default 300). The workflow fixes reasoning at `max`.
The bot login is derived from the App token action's slug output; no manual login variable is needed.

Leave automation variables unset during preflight. From Actions → Oryn → Run workflow, retain
`preflight=true` and optionally select an item number. The job validates source access, installs the
locked runtime dependencies and plans from live target context. It does not call the model, write a
receipt, comment, label, patch, close or merge. Inspect the `plan` artifact; any authentication/planning
failure must be fixed before activation. This preflight verifies read access, not successful publication
or model inference. Then run one selected item with `preflight=false`, `publish=false` to inspect a
real model report, followed by an explicitly selected published review when ready.

Automatic triggers require the workflow on the default branch. Enable only the desired variables:

| Variable | `true` enables |
| --- | --- |
| `ORYN_EVENT_ENABLED` | Native Issue/PR/comment event execution |
| `ORYN_EVENT_PUBLISH` | Publication for those enabled events |
| `ORYN_ENABLED` | A scan every six hours |
| `ORYN_SCHEDULE_PUBLISH` | Publication for scheduled scans |

Manual publication defaults off; the deployment enables all four automatic execution/publication variables. `@oryn-mini review`, `@oryn-mini ask …`, `@oryn-mini stop` and
`@oryn-mini resume` use Oryn's fixed command prefix even when the App has a different display name.
`@oryn-mini repair`, `implement`, `assist`, `rebase`, `cluster`, `close` and `merge` are also available to authorized maintainers.
The operator controls token grants and policy; text in issues/PRs cannot enable repair or merge.
Disable the event/schedule execution variables to stop new automatic work; use the item's stop command
for an active task. Preserve receipts when rotating keys or upgrading the runtime.

## Maintenance and verification

Update the pinned Oryn commit in the setup action through a reviewed PR; rerun a manual preflight and
selected report after changing runtime or credentials. Review local entry changes with:

```sh
actionlint -ignore 'unexpected key "queue" for "concurrency" section' .github/workflows/oryn.yml
node --test scripts/test-oryn-validation.mjs
node scripts/run-gates.mjs
git diff --check
```

The ignore covers only an older actionlint version's lack of native queued-concurrency support; do not
suppress other errors. Local checks do not establish live App authentication or model success.

References: [GitHub App registration](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app),
[pinned token action and permission inputs](https://github.com/actions/create-github-app-token/tree/bcd2ba49218906704ab6c1aa796996da409d3eb1),
[decision 0015](../decisions/0015-oryn-repository-local-actions.md).
