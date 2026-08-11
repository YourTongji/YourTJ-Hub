# Dev Server Disk Investigation (2026-08-11)

Host: `20.205.27.178` (Azure Ubuntu, hostname `YourTJ`)  
SSH: port **5522** (intermittent reachability from WSL observed)  
Scope: read-only diagnosis; no prune/delete performed.

## Snapshot

| Item | Value |
| --- | --- |
| Root disk | 29G total, **24G used (84%)**, 4.7G free |
| Inodes | 5% used (not an inode problem) |
| Memory | 3.8G total, load ~0.3 |
| Uptime | 32 days |

## Space breakdown (host)

| Path | Size | Notes |
| --- | --- | --- |
| `/` | 24G | root filesystem only data disk |
| `/var/lib/containerd` | **18G** | primary consumer (Docker via containerd snapshotter) |
| → overlayfs snapshots | 14G | image/layer snapshots |
| → content blobs | 3.5G | content store |
| `/usr` | 2.2G | OS packages |
| `/opt` | 725M | 1Panel 457M + yourtj 149M + preview 109M |
| `/var/log` | 149M | journal ~125M; site access logs few MB |
| `/opt/yourtj` | 149M | build/snapshots/storage; **not** the disk hog |
| Docker volumes | 81MB | `yourtj_pgdata`, `yourtj_meilidata` only |

## Docker inventory

```
TYPE            TOTAL   ACTIVE   SIZE      RECLAIMABLE
Images          78      7        12.43GB   8.599GB (69%)
Containers      7       5        1.5MB     negligible
Local Volumes   2       2        81MB      0B dangling
Build Cache     77      0        8.882GB   8.882GB
```

Notes:

- `Images` + `Build Cache` in `docker system df` partially **overlap** on disk; real on-disk bulk is `/var/lib/containerd` ≈ 18G.
- **No dangling volumes** (`docker volume ls -f dangling=true` empty).
- Only two named volumes, both in use → **orphan volume hypothesis rejected**.

### yourtj-hub image accumulation

- **72 tags** for `yourtj-hub` (64 `dev-*`, 7 `main-*`, 1 `prev`)
- **71 unique image IDs** (almost every deploy creates a new layer set)
- Active tags:
  - `MAIN_TAG=main-9f5a34adc25225de5bdcd019f13190e1582d72e9`
  - `DEV_TAG=dev-94a5e53b69beb04375fe0b161202bdd30c54bbcc`
- Oldest retained: `2026-08-06` (first day of container deploys)
- Pattern: every CI deploy tags `dev-<sha>` / `main-<sha>` and **never deletes** older tags

### Root cause in deploy path

`/opt/yourtj/scripts/deploy.sh` (and repo `deploy/scripts/deploy.sh`):

1. Tags previous image as `yourtj-hub:prev`
2. `docker build -t yourtj-hub:<new-sha-tag>`
3. compose up new tag
4. **No** `docker image prune` / retention keep-N / age-based rmi

CI (`.github/workflows/deploy-dev.yml` / `deploy-main.yml`) calls:

```text
deploy.sh dev|main /tmp/yourtj-hub dev|main-${{ github.sha }} <port>
```

High-frequency **dev** deploys therefore leave ~150MB-class image history + BuildKit cache layers (~130MB intermediate layers seen in `docker builder du`) forever on a **30G** disk.

## QPS / traffic hypothesis

**Rejected as disk-full cause.**

Evidence from `/opt/1panel/www/sites/forum.yourtj.de/log/access.log`:

| Metric | Value |
| --- | --- |
| Log size | 5.8M / ~22k lines (multi-day) |
| Current hour | ~142 req / 37 min → **avg ~0.06 QPS** |
| Peak minute sample | ~103 req/min → **~1.7 QPS** |
| Peak hour in log | 1880 req/hour → **~0.5 QPS** |
| `unread-status` share | **58.5%** of logged forum requests (polling, not disk growth) |
| Host load | 0.04–0.31 |
| Established connections | low single-digit / tens |

Logs themselves are small; traffic is not filling the disk.

## Other hypotheses

| Hypothesis | Verdict | Evidence |
| --- | --- | --- |
| Framework QPS very high | **No** | Access logs + load average |
| Orphan Docker volumes | **No** | 0 dangling; 2 active volumes 81MB |
| Host deploy garbage under `/opt` | **Minor** | `/opt/yourtj` 149M, preview 109M |
| Application storage bloat | **No** | main/dev storage ~5–8M each |
| System journals | **Minor** | ~124M |
| **Cover-deploy image + build-cache never reclaimed** | **Yes (primary)** | 72 yourtj-hub tags, 8.9G build cache, 18G containerd |

## Safe reclaim plan (NOT executed)

Conservative order (keeps current main/dev + `prev` rollback):

1. **Build cache**  
   `docker builder prune -af`  
   Expected: up to ~8.9G reclaimable (some layers shared with images; actual free space gain may be lower until images pruned).

2. **Unused yourtj-hub tags** (keep current MAIN_TAG, DEV_TAG, `prev`, and optionally last N)  
   Example keep-3 policy sketch:

   ```bash
   # list candidates then rmi; do not delete tags referenced by running containers
   docker images yourtj-hub --format '{{.Repository}}:{{.Tag}}'
   ```

3. **Stopped 1Panel extras** (optional)  
   Exited: `1Panel-casdoor-*`, `1Panel-mysql-*` + unused `mysql:8.4.11` (~1.1G), extra casdoor tag.

4. **Generic**  
   `docker image prune -a` is aggressive; prefer explicit keep-list for yourtj-hub.

Expected free space after careful prune: **roughly +8–15G** (enough to drop well below 70% on 29G disk), depending on shared layers.

## Hardening recommendations

1. Extend `deploy.sh` after successful health check:
   - keep last **N** tags per instance (e.g. N=3: current, prev, prev-1)
   - `docker image rm` older `yourtj-hub:dev-*` / `main-*`
   - optional `docker builder prune -f --filter until=72h`
2. Cron weekly: `docker system df` + alert if root use% > 80
3. Medium-term: larger data disk **or** external registry + pull-only deploys (avoid accumulating local build history on 30G root)
4. Polling (`/api/forum/unread-status`) is traffic noise, not disk; optional later optimization only

## Worktree

- Path: `/root/project/hub/.worktrees/dev-server-disk-investigation`
- Branch: `audit/dev-server-disk-investigation` (from `dev`)
- This report only; no production changes applied.

## Collected at

2026-08-11 ~18:30–18:40 CST via SSH `yourtj@20.205.27.178:5522`
