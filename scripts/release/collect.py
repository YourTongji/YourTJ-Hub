"""Freeze net changes per actual distribution baseline, with complete file inventories."""
import hashlib
import json
import re
from model import require, SHA
from github import git


def applicable(path, channel):
    if channel == "operators":
        return path.startswith(("deploy/", "apps/gooseforum/app/migration/", "docs/operations/"))
    if channel == "web":
        return path.startswith(("apps/gooseforum/", "packages/api-contract/", "deploy/"))
    if not path.startswith("apps/mobile/"):
        return False
    if path.startswith("apps/mobile/store/"):
        return False
    if channel == "android" and "/ios/" in path:
        return False
    if channel.startswith("ios-") and "/android/" in path:
        return False
    return not path.endswith(".md")


def collect(manifest, github):
    source = manifest["sourceSha"]
    require(SHA.fullmatch(source), "Unresolved source")
    entries, inventories, uncertainties = [], {}, []
    # Net diffs avoid announcing a change that was reverted within the release range.
    for channel in manifest["channels"] + (["operators"] if manifest["product"] == "web" else []):
        baseline = manifest["baselines"]["web" if channel == "operators" else channel]
        base = baseline["sourceSha"]
        if base is None:
            base = git("hash-object", "-t", "tree", "--stdin")
            uncertainties.append(f"{channel}: first distribution has no prior baseline; review initial-release claims")
        else:
            git("merge-base", "--is-ancestor", base, source)
        # --no-renames exposes both sides of moves, including moves across platform boundaries.
        paths = git("diff", "--name-only", "--no-renames", "-z", base, source, "--").split("\0")
        paths = [p for p in paths if p]
        inventories[channel] = paths
        for path in paths:
            if not applicable(path, channel):
                continue
            patch = git("diff", "--no-ext-diff", "--no-renames", "--unified=5", base, source, "--", path)
            truncated = len(patch) > 24000
            if truncated:
                uncertainties.append(f"{channel}: {path} diff exceeds model excerpt; reviewer must inspect full source")
            entries.append({"id": hashlib.sha256(f"{channel}:{base}:{source}:{path}".encode()).hexdigest()[:24],
                            "channels": [channel], "title": path, "paths": [path],
                            "detail": patch[:24000] + ("\n[INCOMPLETE EXCERPT]" if truncated else "")})
    # Commit and PR metadata are supplemental evidence, not authority or proof of platform support.
    commit_lines = set()
    for baseline in manifest["baselines"].values():
        args = ["log", "--first-parent", "--format=%H %s", source]
        if baseline["sourceSha"]:
            args += ["--not", baseline["sourceSha"]]
        commit_lines.update(git(*args).splitlines())
    commits = "\n".join(sorted(commit_lines))
    prs = []
    for number in sorted(set(re.findall(r"#(\d+)\)?(?:\n|$)", commits))):
        pr = github.api(f"pulls/{number}")
        prs.append({"number": int(number), "title": pr["title"], "url": pr["html_url"], "body": (pr.get("body") or "")[:8000]})
    request = {"schemaVersion": 1, "repository": github.repository, "sourceSha": source,
               "channels": manifest["channels"] + (["operators"] if manifest["product"] == "web" else []),
               "baselines": manifest["baselines"], "evidence": entries,
               "requiredDisclosures": manifest["requiredDisclosures"]}
    return request, {"schemaVersion": 1, "sourceSha": source, "inventories": inventories,
                     "commits": commits.splitlines(), "pullRequests": prs, "uncertainties": uncertainties,
                     "evidence": entries}
