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


MERGED_PR = re.compile(r"^Merge pull request #(\d+) from (\S+)")
SQUASHED_PR = re.compile(r"\(#(\d+)\)$")
MAX_LINKED_PRS = 100


def pull_request_paths(source, base):
    """Feature PRs merged in (base, source], with the paths each one changed.

    Releases are cut from main, whose first-parent history only shows dev promotions; the
    feature PRs live on the merged side. Promotion merges from dev/main are not features.
    """
    args = ["log", "--format=%H%x00%P%x00%s", source] + ([f"^{base}"] if base else [])
    links = {}
    for line in git(*args).splitlines():
        sha, parents, subject = line.split("\0", 2)
        parents = parents.split()
        merged, squashed = MERGED_PR.match(subject), SQUASHED_PR.search(subject)
        if merged and len(parents) > 1:
            number, branch = int(merged[1]), merged[2]
            if branch.rsplit("/", 1)[-1] in {"dev", "main"}:
                continue
        elif squashed and len(parents) == 1:
            number, branch = int(squashed[1]), None
        else:
            continue
        paths = git("diff", "--name-only", "--no-renames", "-z", parents[0], sha, "--").split("\0")
        link = links.setdefault(number, {"branch": branch, "paths": set()})
        link["paths"].update(p for p in paths if p)
    return links


def collect(manifest, github):
    source = manifest["sourceSha"]
    require(SHA.fullmatch(source), "Unresolved source")
    entries, inventories, uncertainties = [], {}, []
    branches, evidence_prs = {}, {}
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
        channel_links = pull_request_paths(source, baseline["sourceSha"])
        for number, link in channel_links.items():
            branches.setdefault(number, link["branch"])
        for path in paths:
            if not applicable(path, channel):
                continue
            patch = git("diff", "--no-ext-diff", "--no-renames", "--unified=5", base, source, "--", path)
            truncated = len(patch) > 24000
            if truncated:
                uncertainties.append(f"{channel}: {path} diff exceeds model excerpt; reviewer must inspect full source")
            evidence_id = hashlib.sha256(f"{channel}:{base}:{source}:{path}".encode()).hexdigest()[:24]
            entries.append({"id": evidence_id,
                            "channels": [channel], "title": path, "paths": [path],
                            "detail": patch[:24000] + ("\n[INCOMPLETE EXCERPT]" if truncated else "")})
            linked = sorted(n for n, link in channel_links.items() if path in link["paths"])
            if linked:
                evidence_prs[evidence_id] = linked
    # Commit and PR metadata are supplemental evidence, not authority or proof of platform support.
    commit_lines = set()
    for baseline in manifest["baselines"].values():
        args = ["log", "--first-parent", "--format=%H %s", source]
        if baseline["sourceSha"]:
            args += ["--not", baseline["sourceSha"]]
        commit_lines.update(git(*args).splitlines())
    commits = "\n".join(sorted(commit_lines))
    numbers = {int(n) for n in re.findall(r"#(\d+)\)?(?:\n|$)", commits)}
    linked = sorted({n for refs in evidence_prs.values() for n in refs})
    if len(linked) > MAX_LINKED_PRS:
        uncertainties.append(f"{len(linked)} linked pull requests exceed the metadata limit; "
                             "classify the remaining entries from evidence")
    numbers.update(linked[:MAX_LINKED_PRS])
    prs = []
    for number in sorted(numbers):
        pr = github.api(f"pulls/{number}")
        prs.append({"number": number, "title": pr["title"], "url": pr["html_url"],
                    "branch": branches.get(number), "body": (pr.get("body") or "")[:8000]})
    # Reviewed manifest disclosures become citable evidence for required changelog entries.
    # They stay out of the model request, which already carries requiredDisclosures.
    disclosures = [{"id": "disclosure-" + hashlib.sha256(item["id"].encode()).hexdigest()[:16],
                    "channels": list(item["channels"]), "title": f"Required disclosure {item['id']}",
                    "disclosureId": item["id"], "detail": item["text"]}
                   for item in manifest["requiredDisclosures"]]
    request = {"schemaVersion": 1, "repository": github.repository, "sourceSha": source,
               "channels": manifest["channels"] + (["operators"] if manifest["product"] == "web" else []),
               "baselines": manifest["baselines"], "evidence": entries,
               "requiredDisclosures": manifest["requiredDisclosures"]}
    return request, {"schemaVersion": 1, "sourceSha": source, "inventories": inventories,
                     "commits": commits.splitlines(), "pullRequests": prs, "uncertainties": uncertainties,
                     "evidence": entries + disclosures, "evidencePullRequests": evidence_prs}
