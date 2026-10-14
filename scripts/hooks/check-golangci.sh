#!/usr/bin/env bash
# golangci-lint pre-push gate: lint changes relative to the current PR base.
# If GitHub CLI confirms no PR exists, use this repository's documented default target.
# A missing/unfetchable base is an error; never silently omit lint or substitute a stale ref.
set -euo pipefail

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found — run: brew install golangci-lint" >&2
  exit 1
fi

# Git injects a relative GIT_DIR while running hooks. Drop it so Git can rediscover
# the worktree root after this script changes directories.
unset GIT_DIR

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

base_repo=""
base_branch=""
pr_number=""
pr_info=""
pr_url=""
gh_output=""

# Allow a contributor to state a non-default target explicitly when GitHub CLI cannot
# identify a PR, or before the PR has been opened. Format: owner/repo:branch.
parse_lint_base_override() {
  if [[ "$YOURTJ_LINT_BASE" =~ ^([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+):(.+)$ ]]; then
    base_repo=${BASH_REMATCH[1]}
    base_branch=${BASH_REMATCH[2]}
    return 0
  else
    echo "Invalid YOURTJ_LINT_BASE '$YOURTJ_LINT_BASE' — expected owner/repo:branch" >&2
    return 1
  fi
}

if command -v gh >/dev/null 2>&1; then
  if pr_info=$(gh pr view --json number,url --jq '[.number, .url] | @tsv' 2>&1); then
    IFS=$'\t' read -r pr_number pr_url <<< "$pr_info"
    if [[ ! "$pr_number" =~ ^[0-9]+$ ]]; then
      echo "GitHub CLI returned an invalid pull request number: $pr_number" >&2
      exit 1
    fi
    if [[ ! "$pr_url" =~ ^https://github\.com/([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)/pull/${pr_number}/?$ ]]; then
      echo "GitHub CLI returned an unrecognized pull request URL: $pr_url" >&2
      exit 1
    fi
    base_repo=${BASH_REMATCH[1]}/${BASH_REMATCH[2]}
    if gh_output=$(gh api "repos/${base_repo}/pulls/${pr_number}" --jq '.base.repo.full_name + "\t" + .base.ref' 2>&1); then
      IFS=$'\t' read -r base_repo base_branch <<< "$gh_output"
      if [[ -z "$base_repo" || -z "$base_branch" ]]; then
        echo "GitHub CLI returned an incomplete pull request base: $gh_output" >&2
        exit 1
      fi
    else
      gh_status=$?
      echo "Unable to read pull request #${pr_number}'s base from GitHub: $gh_output" >&2
      echo "Restore GitHub API access and retry; the actual PR base could not be verified, so lint was not run." >&2
      exit "$gh_status"
    fi
  else
    gh_status=$?
    if [[ "$pr_info" == *"no pull requests found"* ]]; then
      if [[ -n "${YOURTJ_LINT_BASE:-}" ]]; then
        parse_lint_base_override || exit 1
      else
        base_repo="YourTongji/YourTJ-Hub"
        base_branch="dev"
      fi
    elif [[ -n "${YOURTJ_LINT_BASE:-}" ]]; then
      echo "GitHub CLI lookup failed; using the explicitly configured lint base." >&2
      parse_lint_base_override || exit 1
    else
      echo "Unable to resolve this branch's pull request base with 'gh pr view': $pr_info" >&2
      echo "Fix GitHub CLI access or set YOURTJ_LINT_BASE=owner/repo:branch, then retry." >&2
      exit "$gh_status"
    fi
  fi
else
  if [[ -n "${YOURTJ_LINT_BASE:-}" ]]; then
    parse_lint_base_override || exit 1
  else
    echo "Cannot resolve the actual PR base because GitHub CLI is unavailable." >&2
    echo "Set YOURTJ_LINT_BASE=owner/repo:branch, then retry. Lint was not run." >&2
    exit 1
  fi
fi

if [[ ! "$base_repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || ! git check-ref-format --branch "$base_branch" >/dev/null 2>&1; then
  echo "Invalid lint base '$base_repo:$base_branch'" >&2
  exit 1
fi

base_repo_lower=$(printf '%s' "$base_repo" | tr '[:upper:]' '[:lower:]')
github_repo_from_remote_url() {
  local remote_url=$1 authority host_port host path host_lower

  case "$remote_url" in
    https://*|ssh://*)
      remote_url=${remote_url#*://}
      [[ "$remote_url" == */* ]] || return 1
      authority=${remote_url%%/*}
      path=${remote_url#*/}
      host_port=${authority##*@}
      host=${host_port%%:*}
      ;;
    *://*)
      return 1
      ;;
    *:*)
      authority=${remote_url%%:*}
      host=${authority##*@}
      path=${remote_url#*:}
      ;;
    *)
      return 1
      ;;
  esac

  host_lower=$(printf '%s' "$host" | tr '[:upper:]' '[:lower:]')
  [[ "$host_lower" == "github.com" ]] || return 1

  path=${path%%\?*}
  path=${path%%\#*}
  path=${path%/}
  path=${path%.git}
  [[ "$path" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || return 1
  printf '%s' "$path"
}

base_remote=""
while IFS= read -r remote; do
  [[ -n "$remote" ]] || continue
  remote_url=$(git config --get "remote.${remote}.url" 2>/dev/null || true)
  [[ -n "$remote_url" ]] || continue

  remote_repo=$(github_repo_from_remote_url "$remote_url") || continue
  remote_repo_lower=$(printf '%s' "$remote_repo" | tr '[:upper:]' '[:lower:]')
  if [[ "$remote_repo_lower" == "$base_repo_lower" ]]; then
    base_remote=$remote
    break
  fi
done < <(git remote)

# Fetch the exact base branch on every run. FETCH_HEAD is scoped to this worktree,
# so it can refresh a missing/stale base without sharing a synthetic ref with sibling worktrees.
base_ref=FETCH_HEAD
base_url=https://github.com/${base_repo}.git
fetch_source=${base_remote:-$base_url}
if ! git fetch --no-tags --quiet "$fetch_source" "+refs/heads/${base_branch}"; then
  if [[ "$fetch_source" != "$base_url" ]]; then
    echo "Fetch via ${fetch_source} failed; retrying the same base over HTTPS." >&2
    if ! git fetch --no-tags --quiet "$base_url" "+refs/heads/${base_branch}"; then
      echo "Unable to fetch lint base ${base_repo}:${base_branch} via ${fetch_source} or HTTPS." >&2
      echo "Check the repository, branch, and fetch access; set YOURTJ_LINT_BASE if the target was misidentified. Lint was not run." >&2
      exit 1
    fi
  else
    echo "Unable to fetch lint base ${base_repo}:${base_branch} from ${fetch_source}." >&2
    echo "Check the repository, branch, and fetch access; set YOURTJ_LINT_BASE if the target was misidentified. Lint was not run." >&2
    exit 1
  fi
fi

if ! base_sha=$(git rev-parse --verify "${base_ref}^{commit}" 2>/dev/null); then
  echo "Fetched lint base ${base_repo}:${base_branch}, but its commit could not be resolved. Lint was not run." >&2
  exit 1
fi

echo "golangci-lint baseline: ${base_repo}:${base_branch} (${base_sha})"
cd "$repo_root/apps/gooseforum"
exec golangci-lint run "--new-from-rev=${base_sha}"
