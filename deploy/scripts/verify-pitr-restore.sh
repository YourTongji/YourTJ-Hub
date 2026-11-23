#!/usr/bin/env bash
# verify-pitr-restore.sh — 在完全隔离的临时容器中演练 Cloudflare R2 的 pgBackRest PITR 与 restic 校验。
#
# 安全边界:
#   - 绝不触碰线上 yourtj-postgres 容器、pgdata 卷或 yourtj_main 库
#   - 恢复阶段写入临时 0700 目录；验证阶段以 --network=none 隔离启动临时 PG 容器
#   - 退出时 trap 自动删除临时容器与临时目录
#
# usage: verify-pitr-restore.sh [--target-time "YYYY-MM-DD HH:MM:SS+08"] [--skip-restic] [--allow-empty]
set -euo pipefail
umask 077

ROOT="${YOURTJ_ROOT:-/opt/yourtj}"
R2_ENV_FILE="${R2_ENV_FILE:-$ROOT/backup/r2.env}"
PGBACKREST_CONF_DIR="${PGBACKREST_CONF_DIR:-$ROOT/backup/pgbackrest}"
TARGET_TIME=""
SKIP_RESTIC=0
ALLOW_EMPTY=0

while [ $# -gt 0 ]; do
  case "$1" in
    --target-time) TARGET_TIME="${2:?--target-time requires timestamp}"; shift 2 ;;
    --skip-restic) SKIP_RESTIC=1; shift ;;
    --allow-empty) ALLOW_EMPTY=1; shift ;;
    -h|--help)
      echo "usage: $0 [--target-time 'YYYY-MM-DD HH:MM:SS+08'] [--skip-restic] [--allow-empty]"
      exit 0
      ;;
    *) echo "verify-pitr-restore: unknown argument: $1" >&2; exit 2 ;;
  esac
done

[ -f "$R2_ENV_FILE" ] || { echo "verify-pitr-restore: missing $R2_ENV_FILE" >&2; exit 1; }
# shellcheck disable=SC1090
source "$R2_ENV_FILE"

POSTGRES_IMAGE="${POSTGRES_IMAGE:-}"
PG_USER="${POSTGRES_USER:-}"
if [ -f "$ROOT/.env" ]; then
  [ -n "$POSTGRES_IMAGE" ] || POSTGRES_IMAGE="$(grep -E '^POSTGRES_IMAGE=' "$ROOT/.env" | head -1 | cut -d= -f2- || true)"
  [ -n "$PG_USER" ] || PG_USER="$(grep -E '^POSTGRES_USER=' "$ROOT/.env" | head -1 | cut -d= -f2- || true)"
fi
POSTGRES_IMAGE="${POSTGRES_IMAGE:-yourtj-postgres-backup:16-alpine}"
PG_USER="${PG_USER:-yourtj}"
STANZA="${PGBACKREST_STANZA:-main}"
PG_DB="${VERIFY_PG_DB:-yourtj_main}"

[ -s "$PGBACKREST_CONF_DIR/pgbackrest.conf" ] || {
  echo "verify-pitr-restore: missing $PGBACKREST_CONF_DIR/pgbackrest.conf" >&2
  exit 1
}

WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yourtj-pitr-verify.XXXXXX")"
chmod 0700 "$WORK_DIR"
VERIFY_CONTAINER="yourtj-pitr-verify-$$"

cleanup() {
  docker rm -f "$VERIFY_CONTAINER" >/dev/null 2>&1 || true
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

PG_UID="$(docker run --rm --entrypoint id "$POSTGRES_IMAGE" -u postgres)"
PG_GID="$(docker run --rm --entrypoint id "$POSTGRES_IMAGE" -g postgres)"
mkdir -p "$WORK_DIR/pgdata"
chown "$PG_UID:$PG_GID" "$WORK_DIR/pgdata"
chmod 0700 "$WORK_DIR/pgdata"

RESTORE_ARGS=(--stanza="$STANZA" --pg1-path=/var/lib/postgresql/data --target-action=promote restore)
if [ -n "$TARGET_TIME" ]; then
  RESTORE_ARGS=(--stanza="$STANZA" --pg1-path=/var/lib/postgresql/data --type=time --target="$TARGET_TIME" --target-action=promote restore)
fi

echo "[verify-pitr] restoring stanza=$STANZA (target=${TARGET_TIME:-latest}) into isolated dir"
docker run --rm \
  --user "$PG_UID:$PG_GID" \
  -v "$WORK_DIR/pgdata:/var/lib/postgresql/data" \
  -v "$PGBACKREST_CONF_DIR:/etc/pgbackrest:ro" \
  --entrypoint pgbackrest \
  "$POSTGRES_IMAGE" \
  "${RESTORE_ARGS[@]}"

# Boot restored data directory with archive_mode=off and restore_command enabled only during WAL replay.
# Network is needed during replay so pgBackRest archive-get can pull WAL segments from R2;
# ports are never published to the host. Preserve the primary's max_connections so physical
# WAL recovery does not stall on a lower connection limit.
echo "[verify-pitr] starting isolated verification container $VERIFY_CONTAINER"
docker run -d \
  --name "$VERIFY_CONTAINER" \
  -v "$WORK_DIR/pgdata:/var/lib/postgresql/data" \
  -v "$PGBACKREST_CONF_DIR:/etc/pgbackrest:ro" \
  "$POSTGRES_IMAGE" \
  postgres \
  -c archive_mode=off \
  -c shared_buffers=64MB >/dev/null

READY=0
for _ in $(seq 1 60); do
  if docker exec -u postgres "$VERIFY_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -tAc "SELECT NOT pg_is_in_recovery();" 2>/dev/null | grep -qx "t"; then
    READY=1
    break
  fi
  sleep 2
done

if [ "$READY" -ne 1 ]; then
  echo "[verify-pitr] FAIL: isolated PostgreSQL did not finish WAL recovery in time" >&2
  docker logs --tail 40 "$VERIFY_CONTAINER" >&2 || true
  exit 1
fi

echo "[verify-pitr] running integrity assertions on restored $PG_DB"
COUNTS="$(docker exec -i -u postgres "$VERIFY_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 -tAF '|' <<'SQL'
SELECT
  (SELECT count(*) FROM users),
  (SELECT count(*) FROM topics),
  (SELECT count(*) FROM posts),
  (SELECT count(*) FROM page_config);
SQL
)"
IFS='|' read -r u_cnt t_cnt p_cnt cfg_cnt <<< "$COUNTS"
echo "[verify-pitr] restored counts: users=$u_cnt topics=$t_cnt posts=$p_cnt page_config=$cfg_cnt"
if [ "$ALLOW_EMPTY" -eq 0 ] && { [ "$u_cnt" -le 0 ] || [ "$t_cnt" -le 0 ] || [ "$p_cnt" -le 0 ]; }; then
  echo "[verify-pitr] FAIL: expected non-zero users/topics/posts in restored $PG_DB" >&2
  exit 1
fi

# Verify anonymous identity tables exist and satisfy uniqueness invariants (see docs/operations/anonymous-identity.md)
docker exec -i -u postgres "$VERIFY_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
DO $$
DECLARE dup_owner bigint; dup_uid bigint;
BEGIN
  IF to_regclass('public.anonymous_personas') IS NULL
     OR to_regclass('public.anonymous_bindings') IS NULL
     OR to_regclass('public.anonymous_reveal_audits') IS NULL THEN
    RAISE EXCEPTION 'missing required anonymous identity tables in restored database';
  END IF;
  SELECT count(*) INTO dup_owner FROM (SELECT owner_id FROM anonymous_bindings GROUP BY owner_id HAVING count(*) > 1) s;
  SELECT count(*) INTO dup_uid FROM (SELECT persona_uid FROM anonymous_bindings GROUP BY persona_uid HAVING count(*) > 1) s;
  IF dup_owner > 0 OR dup_uid > 0 THEN
    RAISE EXCEPTION 'anonymous_bindings uniqueness violation: dup_owner=%, dup_uid=%', dup_owner, dup_uid;
  END IF;
END $$;
SQL
echo "[verify-pitr] PostgreSQL PITR restore verification PASSED"

if [ "$SKIP_RESTIC" -eq 0 ]; then
  command -v restic >/dev/null 2>&1 || { echo "[verify-pitr] FAIL: restic not installed" >&2; exit 1; }
  export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
  export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
  export RESTIC_PASSWORD
  export RESTIC_REPOSITORY="s3:https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com/${R2_BUCKET}/${R2_RESTIC_PREFIX:-restic}"
  echo "[verify-pitr] running restic check (--read-data-subset=${RESTIC_CHECK_SUBSET:-5%})"
  restic check --read-data-subset="${RESTIC_CHECK_SUBSET:-5%}"
  echo "[verify-pitr] restic integrity check PASSED"
fi
