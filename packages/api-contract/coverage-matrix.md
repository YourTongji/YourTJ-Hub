# Route coverage report

This is the stable entry point, not a snapshot of the latest coverage counts. The route-by-route
matrix is generated from the current checkout into ignored `.tmp/coverage-matrix.md`; feature PRs
do not update this page or commit the report.

## Generate locally

From the repository root:

```bash
make contract-check
```

The command validates the contract, fixtures, generated TypeScript and route coverage. After it
succeeds, open `packages/api-contract/.tmp/coverage-matrix.md`. A failed coverage check removes the
previous report; do not use an old report after changing inputs without rerunning the check.

## Read CI reports

Open [CI / Verify](https://github.com/YourTongji/YourTJ-Hub/actions/workflows/ci.yml), select the run
for the source commit you need, and read the contract job summary or download the
`api-route-coverage` artifact. Both identify the checked-out source commit. Artifacts are retained
for seven days; use the local command on that checkout to reproduce expired reports. A run that
does not select the contract job does not publish a new coverage report.

## Sources and change boundaries

- [OpenAPI entry](openapi.yaml) and its domain files own API operations.
- [Route snapshot](fixtures/routes-snapshot.json) comes from `TestRoutesSnapshot` under the default
  server configuration; OIDC-enabled routes have separate tests.
- [Coverage exceptions](route-coverage.json) own explicit exclusions and known uncovered routes.
- [Coverage gate](scripts/check-route-coverage.mjs) rejects omissions, stale/dangling exceptions and
  contract operations with no registered route before generating a report.

Route snapshots, contracts, generated TypeScript, fixtures and explicit exceptions remain committed
and must accompany route changes. These inputs can still conflict: merge their actual changes and
regenerate the temporary report. Do not move report rows or counts into another shared tracked file.
