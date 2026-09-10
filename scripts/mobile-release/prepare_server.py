#!/usr/bin/env python3
"""Promote dev through PR CI and merge, then tag that exact production commit."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time


REPOSITORY = "YourTongji/YourTJ-Hub"
# These workflows run for every PR. Never interpret missing checks as success,
# even when the repository has no required status checks configured.
REQUIRED_WORKFLOWS = {
    ".github/workflows/ci-backend.yml",
    ".github/workflows/ci-frontend.yml",
    ".github/workflows/ci-contract.yml",
    ".github/workflows/ci-govulncheck.yml",
}


def next_version(tags, bump):
    versions = [tuple(map(int, match.groups())) for tag in tags
                if (match := re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)", tag))]
    major, minor, patch = max(versions, default=(0, 0, 0))
    if bump == "patch": patch += 1
    elif bump == "minor": minor, patch = minor + 1, 0
    elif bump == "major": major, minor, patch = major + 1, 0, 0
    else: raise ValueError("Unknown server version increment")
    return f"v{major}.{minor}.{patch}"


def read(*args):
    return subprocess.check_output(args, text=True).strip()


def api(path, *args):
    return json.loads(read("gh", "api", f"repos/{REPOSITORY}/{path}", *args))


def ci_ready(pr, source_sha, since):
    pages = api(f"actions/runs?event=pull_request&branch=dev&head_sha={source_sha}&per_page=100",
                "--paginate", "--slurp")
    latest = {}
    for page in pages:
        for run in page["workflow_runs"]:
            path = run["path"]
            if (not path.startswith(".github/workflows/ci-")
                    or run["event"] != "pull_request"
                    or run["head_sha"] != source_sha
                    or run["head_branch"] != "dev"
                    or run["head_repository"]["full_name"] != REPOSITORY
                    or run["created_at"] < since):
                continue
            # GitHub can return an empty association for same-repository PRs.
            # In that case branch, SHA and PR/base freshness still constrain the run.
            if run["pull_requests"] and not any(
                    item["number"] == pr["number"] for item in run["pull_requests"]):
                continue
            if path not in latest or run["id"] > latest[path]["id"]:
                latest[path] = run
    for run in latest.values():
        if run["status"] == "completed" and run["conclusion"] != "success":
            raise RuntimeError(f"Release CI {run['conclusion']}: {run['html_url']}")
    missing = REQUIRED_WORKFLOWS - latest.keys()
    pending = [path for path, run in latest.items() if run["status"] != "completed"]
    if missing or pending:
        print("Waiting for PR CI: " + ", ".join(sorted(missing | set(pending))), flush=True)
        return False
    # Wait for whole workflows, including dependent race/PG jobs, not just their
    # first green job. Path-filtered CI participates whenever it has a run.
    return True


def validate_pr(pr, source_sha, base_sha):
    if (pr["head"]["ref"] != "dev" or pr["base"]["ref"] != "main"
            or pr["head"]["repo"]["full_name"] != REPOSITORY
            or pr["base"]["repo"]["full_name"] != REPOSITORY):
        raise RuntimeError("Release PR must promote this repository's dev to main")
    if pr["head"]["sha"] != source_sha or pr["base"]["sha"] != base_sha:
        raise RuntimeError("dev or main changed during release; start a new release for the new snapshot")
    if pr["state"] != "open" and not pr["merged"]:
        raise RuntimeError("Release PR was closed without merging")
    if pr["draft"]:
        raise RuntimeError("Release PR is a draft; mark it ready before releasing")
    if not pr["merged"] and pr["mergeable_state"] == "dirty":
        raise RuntimeError("Release PR has merge conflicts")


def wait_for_merge(number, source_sha, base_sha, timeout=3600):
    deadline = time.monotonic() + timeout
    base_date = api(f"commits/{base_sha}")["commit"]["committer"]["date"]
    while time.monotonic() < deadline:
        pr = api(f"pulls/{number}")
        validate_pr(pr, source_sha, base_sha)
        # Do not reuse dev push checks or PR checks older than this PR/base.
        if ci_ready(pr, source_sha, max(pr["created_at"], base_date)):
            # Re-read after CI polling so changes during the API calls cannot
            # silently select a different head. The merge API also locks the SHA.
            pr = api(f"pulls/{number}")
            validate_pr(pr, source_sha, base_sha)
            if pr["merged"]:
                return pr["merge_commit_sha"]
            if pr["mergeable"] and pr["mergeable_state"] == "clean":
                merged = api(f"pulls/{number}/merge", "--method", "PUT",
                             "-f", f"sha={source_sha}", "-f", "merge_method=merge")
                if not merged.get("merged"):
                    raise RuntimeError(f"PR merge refused: {merged.get('message')}")
                return merged["sha"]
            print(f"Waiting for PR #{number} merge requirements: {pr['mergeable_state']}", flush=True)
        time.sleep(15)
    raise TimeoutError(f"Timed out waiting for PR #{number} CI/merge requirements; no release tag created")


def ensure_promoted():
    source_sha = read("git", "rev-parse", "origin/dev")
    base_sha = read("git", "rev-parse", "origin/main")
    if (os.environ.get("GITHUB_REF") == "refs/heads/dev"
            and os.environ["GITHUB_SHA"] != source_sha):
        raise RuntimeError("dev changed since this workflow was dispatched; start a new release")
    # Content equivalence accepts squash/rebase merges without requiring matching ancestry.
    comparison = subprocess.run(["git", "diff", "--quiet", base_sha, source_sha]).returncode
    if comparison not in (0, 1):
        raise RuntimeError("Could not compare main and dev")
    if comparison == 1:
        prs = json.loads(read("gh", "pr", "list", "--base", "main", "--head", "dev", "--state", "open", "--json", "number,url"))
        if prs:
            number = prs[0]["number"]
            print("Continuing release PR: " + prs[0]["url"], flush=True)
        else:
            with tempfile.TemporaryDirectory() as folder:
                body = Path(folder) / "body.md"
                body.write_text(f"Promote dev snapshot `{source_sha}` to production. The server release workflow waits for PR CI, merges this PR, then tags, publishes and deploys that commit in the same run. Branch protection still applies. Mobile releases use their separate mobile-vX.Y.Z tags.\n")
                url = read("gh", "pr", "create", "--base", "main", "--head", "dev", "--title", "chore: promote dev to main", "--body-file", str(body))
                print(url, flush=True)
                number = int(url.rstrip("/").rsplit("/", 1)[1])
        sha = wait_for_merge(number, source_sha, base_sha)
        subprocess.run(["git", "fetch", "origin", "main"], check=True)
        # A concurrent main update can race the merge API (which locks only the
        # head). Never publish a merge containing a base we did not validate.
        if read("git", "show", "-s", "--format=%P", sha).split() != [base_sha, source_sha]:
            raise RuntimeError("Merged PR has unexpected parents; refusing to publish")
        return sha
    return base_sha


def main():
    subprocess.run(["git", "fetch", "origin", "dev", "main", "--tags"], check=True)
    sha = ensure_promoted()
    tag = next_version(read("git", "tag", "--list").splitlines(), os.environ["BUMP"])
    # Ref creation through the API avoids putting a PAT in Git remote URLs.
    existing_tags = read("git", "tag", "--points-at", sha).splitlines()
    if any(re.fullmatch(r"v\d+\.\d+\.\d+", item) for item in existing_tags):
        raise ValueError("This main commit already has a server release tag; resume its failed run instead")
    read("gh", "api", "--method", "POST", f"repos/{REPOSITORY}/git/refs",
         "-f", f"ref=refs/tags/{tag}", "-f", f"sha={sha}")
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        output.write(f"tag={tag}\n")
    print(f"Tagged reviewed main commit {sha}: {tag}")


if __name__ == "__main__":
    main()
