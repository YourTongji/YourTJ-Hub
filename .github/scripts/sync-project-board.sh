#!/usr/bin/env bash
# shellcheck disable=SC2016  # GraphQL 模板里的 $var 是 GraphQL 变量，不是 shell 展开
#
# sync-project-board.sh — 把当前 issue / PR 幂等同步到 GitHub Projects v2 看板（RPD 状态机）。
#
# 事件上下文由调用方（workflow 或手动执行）通过环境变量传入，脚本不解析任何
# GitHub 特殊语法，因此可以在本地用 mock 的 gh 命令完整测试。
#
# 环境变量（均可由调用方覆盖）：
#   GH_TOKEN          必填。具备 read:project / write:project 权限的 token（PAT 或 GitHub App 签发）。
#   PROJECT_OWNER     项目属主，默认 YourTongji。
#   PROJECT_NUMBER    必填。看板项目号（看板 URL 末尾的数字）。
#   ITEM_URL          可选。要同步的 issue / PR URL；为空时由 ITEM_KIND + ITEM_NUMBER 拼装。
#   ITEM_NUMBER       可选。配合 ITEM_URL 缺失时使用（workflow_dispatch 手动同步）。
#   ITEM_KIND         issue | pr，默认 issue。
#   EVENT             opened | reopened | labeled | closed | ready_for_review | manual，默认 opened。
#   LABELS            逗号分隔的标签列表。
#   STATUS_FIELD      状态字段名，默认 Status。
#   TODO_VALUE        需求列，默认 Todo。
#   PLANNED_VALUE     计划列，默认 Planned。
#   IN_PROGRESS_VALUE 开发列，默认 In Progress。
#   DONE_VALUE        已完成列，默认 Done。
#   PLANNED_LABELS    命中即视为"计划中"的标签，默认 planning,planned,ready。
#   GITHUB_REPOSITORY Actions 自动注入的 owner/repo，用于拼装 URL。

set -euo pipefail

PROJECT_OWNER="${PROJECT_OWNER:-YourTongji}"
PROJECT_NUMBER="${PROJECT_NUMBER:-}"
ITEM_URL="${ITEM_URL:-}"
ITEM_NUMBER="${ITEM_NUMBER:-}"
ITEM_KIND="${ITEM_KIND:-issue}"
EVENT="${EVENT:-opened}"
LABELS="${LABELS:-}"
STATUS_FIELD="${STATUS_FIELD:-Status}"
TODO_VALUE="${TODO_VALUE:-Todo}"
PLANNED_VALUE="${PLANNED_VALUE:-Planned}"
IN_PROGRESS_VALUE="${IN_PROGRESS_VALUE:-In Progress}"
DONE_VALUE="${DONE_VALUE:-Done}"
PLANNED_LABELS="${PLANNED_LABELS:-planning,planned,ready}"

# 解析逗号分隔的标签列表（用 read -ra 精确切分，避免 word-splitting 拆坏含空格的标签）。
IFS=',' read -r -a LABEL_LIST <<<"$LABELS"

log()  { printf '[sync-project-board] %s\n' "$*"; }
fail() { printf '::error:: %s\n' "$*" >&2; exit 1; }
warn() { printf '::warning:: %s\n' "$*" >&2; }

# ---------------------------------------------------------------------------
# GraphQL 辅助：用最小查询与 Projects v2 交互。
# 不用 `gh project item-list / item-add / item-edit`：它们的响应会拉取条目
# 的全部字段值（含 reviewers 字段），组织项目下 token 权限不足时整个请求失败
# （GraphQL: Resource not accessible ... reviewers ...）。
# ---------------------------------------------------------------------------

# 幂等查重：按条目 URL 在项目中查找 item id（分页遍历，只取 content.url / id）。
graphql_item_id() {
  local url="$1" cursor="" resp proj id q
  q='query($login: String!, $number: Int!, $after: String) {
    organization(login: $login) {
      projectV2(number: $number) {
        items(first: 100, after: $after) {
          pageInfo { hasNextPage endCursor }
          nodes {
            id
            content {
              ... on Issue { url }
              ... on PullRequest { url }
            }
          }
        }
      }
    }
    user(login: $login) {
      projectV2(number: $number) {
        items(first: 100, after: $after) {
          pageInfo { hasNextPage endCursor }
          nodes {
            id
            content {
              ... on Issue { url }
              ... on PullRequest { url }
            }
          }
        }
      }
    }
  }'
  while :; do
    # bash 3.2 兼容：不用空数组展开（set -u 下会 unbound variable），改用 set -- 条件组参。
    if [ -n "$cursor" ]; then
      set -- -f "query=$q" -f "login=$PROJECT_OWNER" -F "number=$PROJECT_NUMBER" -f "after=$cursor"
    else
      set -- -f "query=$q" -f "login=$PROJECT_OWNER" -F "number=$PROJECT_NUMBER"
    fi
    resp="$(gh api graphql "$@")" \
      || fail "无法读取项目条目：请确认 GH_TOKEN 具备 read:project 权限、项目号正确"
    proj="$(jq -r '(.data.organization.projectV2 // .data.user.projectV2) // empty' <<<"$resp")"
    [ -n "$proj" ] || fail "无法定位项目 $PROJECT_OWNER/projects/$PROJECT_NUMBER：请确认 PROJECT_OWNER / PROJECT_NUMBER 正确"
    id="$(jq -r --arg url "$url" '[.items.nodes[]? | select(.content.url == $url) | .id][0] // empty' <<<"$proj")"
    if [ -n "$id" ]; then
      printf '%s' "$id"
      return 0
    fi
    [ "$(jq -r '.items.pageInfo.hasNextPage' <<<"$proj")" = "true" ] || break
    cursor="$(jq -r '.items.pageInfo.endCursor' <<<"$proj")"
  done
  return 0
}

# 把 issue / PR URL 解析为全局节点 id。
graphql_content_id() {
  gh api graphql \
    -f query='query($url: URI!) {
      resource(url: $url) {
        ... on Issue { id }
        ... on PullRequest { id }
      }
    }' \
    -f url="$ITEM_URL" 2>/dev/null \
    | jq -r '.data.resource.id // empty'
}

# 添加条目（返回新 item id）。
graphql_add_item() {
  local content_id
  content_id="$(graphql_content_id)" \
    || fail "无法解析条目 URL（仅支持 issue / PR）：$ITEM_URL"
  [ -n "$content_id" ] || fail "无法解析条目 URL（仅支持 issue / PR）：$ITEM_URL"
  gh api graphql \
    -f query='mutation($projectId: ID!, $contentId: ID!) {
      addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
        item { id }
      }
    }' \
    -f projectId="$PROJECT_ID" -f contentId="$content_id" 2>/dev/null \
    | jq -r '.data.addProjectV2ItemById.item.id // empty'
}

# ---------------------------------------------------------------------------
# 0) 必填校验
# ---------------------------------------------------------------------------
[ -n "${GH_TOKEN:-}" ] || fail "GH_TOKEN 未设置：请先在仓库配置具备项目读写权限的 secret（见 docs/development/project-board.md）"
[ -n "$PROJECT_NUMBER" ] || fail "PROJECT_NUMBER 未设置：请在仓库变量或 workflow env 中配置看板项目号"

if [ -z "$ITEM_URL" ]; then
  [ -n "$ITEM_NUMBER" ] || fail "ITEM_URL 与 ITEM_NUMBER 均未设置：无法定位要同步的 issue/PR"
  kind_path="issues"
  [ "$ITEM_KIND" = "pr" ] && kind_path="pull"
  ITEM_URL="https://github.com/${GITHUB_REPOSITORY:-$PROJECT_OWNER/YourTJ-Hub}/${kind_path}/${ITEM_NUMBER}"
fi
log "目标条目: $ITEM_URL (kind=$ITEM_KIND, event=$EVENT)"

# ---------------------------------------------------------------------------
# 1) 定位项目
# ---------------------------------------------------------------------------
project_json="$(gh project view "$PROJECT_NUMBER" --owner "$PROJECT_OWNER" --format json 2>/dev/null)" \
  || fail "无法访问项目 $PROJECT_OWNER/projects/$PROJECT_NUMBER：请确认 GH_TOKEN 具备 read:project 权限、项目号正确"
PROJECT_ID="$(jq -r '.id' <<<"$project_json")"
[ -n "$PROJECT_ID" ] && [ "$PROJECT_ID" != "null" ] || fail "项目解析失败（$PROJECT_OWNER/projects/$PROJECT_NUMBER）"
log "已定位项目 $PROJECT_OWNER/projects/$PROJECT_NUMBER ($PROJECT_ID)"

# ---------------------------------------------------------------------------
# 2) 幂等添加：已存在则复用 item id，否则添加
# ---------------------------------------------------------------------------
existing_item="$(graphql_item_id "$ITEM_URL")"

if [ -n "$existing_item" ]; then
  ITEM_ID="$existing_item"
  ITEM_REUSED=1
  log "条目已在项目中，复用 item $ITEM_ID"
else
  ITEM_ID="$(graphql_add_item)"
  [ -n "$ITEM_ID" ] || fail "无法添加条目：请确认 GH_TOKEN 具备 write:project 权限、条目 URL 正确"
  ITEM_REUSED=0
  log "已添加条目 → item $ITEM_ID"
fi

# ---------------------------------------------------------------------------
# 3) 状态字段发现（仅单选项字段有意义）
# ---------------------------------------------------------------------------
field="$(gh project field-list "$PROJECT_NUMBER" --owner "$PROJECT_OWNER" --format json \
  | jq -c --arg name "$STATUS_FIELD" '.fields[] | select(.name == $name)' 2>/dev/null || true)"

if [ -z "$field" ]; then
  warn "未找到状态字段 \"$STATUS_FIELD\"，仅完成添加、未设置状态（可用 STATUS_FIELD 变量覆盖）"
  exit 0
fi

# ---------------------------------------------------------------------------
# 4) RPD 状态机：目标列计算
# ---------------------------------------------------------------------------
target_value="$TODO_VALUE"
skip_status=false
if [ "$EVENT" = "closed" ]; then
  target_value="$DONE_VALUE"
elif [ "$EVENT" = "labeled" ] && [ "$ITEM_KIND" = "issue" ]; then
  # labeled 事件：仅命中计划标签时更新为 Planned；未命中则保持看板现状
  # （新入板条目仍落 Todo，已存在条目不覆盖人工挪动的状态）。
  if [ "${#LABEL_LIST[@]}" -gt 0 ]; then
    for label in "${LABEL_LIST[@]}"; do
      if [[ ",$PLANNED_LABELS," == *",$label,"* ]]; then
        target_value="$PLANNED_VALUE"
        break
      fi
    done
  fi
  if [ "$target_value" = "$TODO_VALUE" ] && [ "$ITEM_REUSED" = "1" ]; then
    log "labeled 事件未命中计划标签（labels=[$LABELS]），条目已存在，保持看板现状"
    skip_status=true
  fi
elif [ "$ITEM_KIND" = "pr" ] && [[ "$EVENT" =~ ^(opened|reopened|ready_for_review|labeled|manual)$ ]]; then
  target_value="$IN_PROGRESS_VALUE"
elif [ "$ITEM_KIND" = "issue" ]; then
  if [ "${#LABEL_LIST[@]}" -gt 0 ]; then
    for label in "${LABEL_LIST[@]}"; do
      if [[ ",$PLANNED_LABELS," == *",$label,"* ]]; then
        target_value="$PLANNED_VALUE"
        break
      fi
    done
  fi
fi

if [ "$skip_status" = "true" ]; then
  exit 0
fi


option_id="$(jq -r --arg v "$target_value" '[((.options // [])[]) | select(.name == $v) | .id][0] // empty' <<<"$field")"
if [ -z "$option_id" ]; then
  warn "字段 \"$STATUS_FIELD\" 缺少选项 \"$target_value\"，跳过状态设置（可用 TODO_VALUE/PLANNED_VALUE/IN_PROGRESS_VALUE/DONE_VALUE 变量调整）"
  exit 0
fi

# ---------------------------------------------------------------------------
# 5) 设置状态字段
# ---------------------------------------------------------------------------
field_id="$(jq -r '.id' <<<"$field")"
gh api graphql \
  -f query='mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
    updateProjectV2ItemFieldValue(input: {
      projectId: $projectId
      itemId: $itemId
      fieldId: $fieldId
      value: { singleSelectOptionId: $optionId }
    }) { projectV2Item { id } }
  }' \
  -f projectId="$PROJECT_ID" -f itemId="$ITEM_ID" \
  -f fieldId="$field_id" -f optionId="$option_id" >/dev/null \
  || fail "无法设置状态字段：请确认 GH_TOKEN 具备 write:project 权限"
log "已设置状态 → \"$target_value\""
