# Issue #94 — Lifecycle + Compliance Gaps Acceptance

Date: 2026-08-12
Worktree: `.worktrees/topic-delete-issue-94/apps/gooseforum`
Scope: evidence snapshot TTL, Terms/Privacy disclosure, network access logs retention

## Gap 1 — Evidence snapshot TTL auto-cleanup (180 days)

| Check | Status | Evidence |
| --- | --- | --- |
| `EvidenceSnapshotRetention = 180 * 24 * time.Hour` | Pass | `app/service/contentdeleteservice/content_delete_service.go` |
| `ExpireEvidenceSnapshotsBatch` calls clearer with cutoff | Pass | same file |
| `ClearExpiredEvidenceSnapshots` only closed reports with non-empty snapshot | Pass | `app/models/forum/reports/reports_rep.go` |
| Skip `LEGAL_HOLD` / `EVIDENCE_HOLD` topics | Pass | SQL `NOT EXISTS` + unit test |
| Open reports never cleared | Pass | unit test |
| Cron `8 3 * * *` | Pass | `app/console/job/cron.go` |
| Unit test | Pass | `go test ./app/models/forum/reports/` |

## Gap 2 — Terms + Privacy disclosure

| Check | Status | Evidence |
| --- | --- | --- |
| Terms: 30-day recovery + governance retention | Pass | `pageconfig/terms.json` |
| Privacy: 30-day recovery + governance + network logs ≥6 months | Pass | `pageconfig/privacy.json` |
| Public `/terms` and `/privacy` routes | Pass | `route4api.go` + `forum/terms.go` + `forum/privacy.go` |
| Vue + noscript templates | Pass | `TermsPage.vue`, `PrivacyPage.vue`, `*.gohtml` |
| Default embed load | Pass | `defaultconfig` + test assertions |
| Admin GET/SAVE privacy API | Pass | `adminController` + admin routes |
| Register form links Terms + Privacy | Pass | `LoginPage.vue` + `auth.privacyLink` locales |

## Gap 3 — `network_access_logs` ≥6 month retention

| Check | Status | Evidence |
| --- | --- | --- |
| Table/model `network_access_logs` | Pass | `app/models/forum/networkAccessLog/` |
| Migration AutoMigrate includes entity | Pass | `app/migration/migration.go` |
| Retention constant 183 days (≥6 months) | Pass | `network_access_log.go` |
| Middleware DB persist via `AccessLog` | Pass | `app/http/middleware/logger.go` (gated by `server.accessLog`) |
| Expire batch + cron `9 3 * * *` | Pass | `ExpireBefore` + `cron.go` |
| Unit test Record + ExpireBefore | Pass | `go test ./app/models/forum/networkAccessLog/` |

## Test commands run

```text
go build -o /tmp/gooseforum-issue94 ./
go test ./app/models/defaultconfig/ ./app/models/forum/reports/ ./app/models/forum/networkAccessLog/ ./app/service/contentdeleteservice/ ./app/migration/ -count=1
go test ./app/http/middleware/ ./app/http/controllers/forum/ ./app/http/controllers/api/ -count=1
go test ./app/models/forum/... ./app/service/contentdeleteservice/... ./app/migration/... ./app/console/... -count=1
```

All listed packages: **PASS** (or no test files). Build: **OK**.

## Remaining / non-blocking notes

1. **DB access log is behind `server.accessLog`** (template default `false`; local `config.toml` may set `true`). Ops must enable the flag for continuous network-log capture; retention job is safe no-op when empty.
2. **Admin UI for Privacy** mirrors API-only for now (Terms still has dedicated admin settings page). Operators can edit via API or seed defaults; optional follow-up to add admin settings kind `privacy`.
3. **Footer chrome** still config-driven; default site chrome does not hardcode Terms/Privacy links. Public routes + register disclosure cover the R14 disclosure path.
4. Untracked / dirty files for this gap set must be staged before commit (not done in this session).

## Verdict

**Issue #94 lifecycle + compliance gaps 1–3: ACCEPT.**
