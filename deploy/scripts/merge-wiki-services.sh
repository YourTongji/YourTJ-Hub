#!/usr/bin/env bash
# merge-wiki-services.sh — 把旧 compose 中未迁移的 wiki 服务块并入新 compose。
# 目的: 存量服务器(PR #219 前初始化)的 compose 仍定义 wiki-main/wiki-dev,
#       直接换成新 compose 后 deploy.sh 的 --remove-orphans 会把 wiki 容器
#       带下线。本脚本在 main 部署更新 compose 时保留这些服务块, 直到
#       wiki 内容迁移完成后再手动移除(见 docs/operations/deployment.md)。
# usage: merge-wiki-services.sh <old-compose> <new-compose>
set -euo pipefail

OLD="${1:?usage: merge-wiki-services.sh <old-compose> <new-compose>}"
NEW="${2:?usage: merge-wiki-services.sh <old-compose> <new-compose>}"

[ -f "$OLD" ] || { echo "merge-wiki: $OLD not found, skip"; exit 0; }
[ -f "$NEW" ] || { echo "merge-wiki: $NEW not found, skip"; exit 0; }

# 新 compose 已含 wiki 服务则无需合并
if grep -qE '^\s{2}wiki-(main|dev):' "$NEW"; then
  echo "merge-wiki: new compose already has wiki services, skip"
  exit 0
fi

# 旧 compose 无 wiki 服务也无需合并
if ! grep -qE '^\s{2}wiki-(main|dev):' "$OLD"; then
  echo "merge-wiki: old compose has no wiki services, skip"
  exit 0
fi

python3 - "$OLD" "$NEW" <<'PYEOF'
import re, sys
old_path, new_path = sys.argv[1], sys.argv[2]
old = open(old_path).read()
new = open(new_path).read()

blocks = []
for svc in ('wiki-main', 'wiki-dev'):
    m = re.search(rf'^(\s{{2}}{svc}:\n(?:\s{{4,}}.*\n|\s*\n)*)', old, re.M)
    if m:
        blocks.append(m.group(1))

if blocks and 'wiki-' not in new:
    new = re.sub(r'^(services:\n)', r'\1' + ''.join(blocks), new, count=1, flags=re.M)
    open(new_path, 'w').write(new)
    print(f'merge-wiki: merged {len(blocks)} legacy wiki service block(s) into {new_path}')
else:
    print('merge-wiki: nothing to merge')
PYEOF
