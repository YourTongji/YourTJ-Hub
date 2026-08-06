#!/usr/bin/env bash
# snapshot-db.sh — 用 SQLite online backup API 生成一致性快照。
# WAL 模式下源库在线运行也安全(项目自身 backupSQLite 同思路)。
# usage: snapshot-db.sh <source.db> <snapshot.db>
set -euo pipefail

SRC="${1:?usage: snapshot-db.sh <source.db> <snapshot.db>}"
DEST="${2:?usage: snapshot-db.sh <source.db> <snapshot.db>}"

if [ ! -f "$SRC" ]; then
  echo "snapshot-db: source not found: $SRC (skip)"
  exit 0
fi

mkdir -p "$(dirname "$DEST")"
TMP="$DEST.tmp.$$"
sqlite3 "$SRC" ".backup '$TMP'" || {
  echo "snapshot-db: backup failed for $SRC" >&2
  rm -f "$TMP"
  exit 1
}
mv -f "$TMP" "$DEST"
echo "snapshot-db: $SRC -> $DEST"
