"""Trusted release preparation and approval controller; source code is a separate checkout."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time
from model import (CHANGELOG, FILES, ID, SHA, REPOSITORY, ReleaseError, require, digest, channels_for,
                   next_identity, validate_candidate, validate_approval)
from github import GitHub, git, run
from collect import collect
from state import reservations, baselines, latest_receipt, record

ROOT = Path(__file__).resolve().parents[2]


def write_json(path, value):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    Path(path).write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding='utf-8')


def candidate_path(candidate_id):
    require(isinstance(candidate_id, str) and ID.fullmatch(candidate_id), "Invalid candidate ID")
    return f"releases/requests/{candidate_id}"


def load_candidate(candidate_id, ref="HEAD", destination=None, draft=False):
    prefix = candidate_path(candidate_id)
    paths = git("ls-tree", "-r", "--name-only", ref, "--", prefix).splitlines()
    require(paths and all(p.rsplit("/", 1)[0] == prefix for p in paths), "Missing/invalid candidate tree")
    folder = Path(destination) if destination else ROOT / prefix
    # Only approved, regular blob files are materialized; never follow symlinks.
    names = {p.rsplit('/', 1)[1] for p in paths}
    require(names <= {'manifest.json', 'evidence.json', 'operators.zh-CN.md', CHANGELOG, *FILES.values()},
            'Unknown candidate files')
    require(not folder.is_symlink(), 'Candidate directory must not be a symlink')
    folder.mkdir(parents=True, exist_ok=True)
    require(all(p.name in names and p.is_file() and not p.is_symlink() for p in folder.iterdir()),
            'Materialization directory contains unreviewed or unsafe files')
    for path in paths:
        mode = git("ls-tree", ref, "--", path).split()[0]
        require(mode == "100644", "Candidate files must be regular non-executable blobs")
        # git() trims command output; approved blobs must retain every byte, including whitespace.
        (folder / path.rsplit("/", 1)[1]).write_bytes(subprocess.check_output(['git', 'show', f'{ref}:{path}']))
    manifest = json.loads((folder / "manifest.json").read_text(encoding='utf-8'))
    require(manifest["candidateId"] == candidate_id, "Directory/manifest identity mismatch")
    validate_candidate(manifest, folder, draft)
    evidence = json.loads((folder / "evidence.json").read_text(encoding='utf-8'))
    require(evidence.get("schemaVersion") == 1 and evidence.get("sourceSha") == manifest["sourceSha"],
            "Evidence/source mismatch")
    return manifest, folder


def source_sha(source):
    git("fetch", "origin", "main", "--tags")
    source = "origin/main" if source == "main" else source
    require(source == "origin/main" or SHA.fullmatch(source), "Source must be main or a full main-history SHA")
    sha = git("rev-parse", f"{source}^{{commit}}")
    git("merge-base", "--is-ancestor", sha, "origin/main")
    return sha


def trusted_request(pr, github):
    """A release-looking fork branch never owns a preparation slot or identity."""
    head = pr.get('head') or {}
    return (head.get('repo') or {}).get('full_name', '').lower() == github.repository.lower()


def open_request(github, product):
    for pr in github.pages("pulls?state=open&base=main"):
        if not trusted_request(pr, github):
            continue
        branch = pr["head"]["ref"]
        if branch.startswith(f"codex/release/{product}-"):
            return {"number": pr["number"], "url": pr["html_url"], "candidateId": branch.split("/")[-1]}
    return None


def plan(scope, bump, source, destination, github, apple=None):
    channels = channels_for(scope, destination)
    product = "web" if scope == "web" else "mobile"
    pending = open_request(github, product)
    require(pending is None, f"An undecided {product} release request already exists: {pending}")
    sha = source_sha(source)
    tags = reservations()
    # Closed requests release the preparation slot but their version is not reused: a retained
    # branch may contain human edits or an uncertain attempt. Include all trusted identities.
    # REST head filtering requires an exact branch; a prefix query would miss reservations.
    for pr in github.pages("pulls?state=closed&base=main"):
        if not trusted_request(pr, github):
            continue
        branch = pr["head"]["ref"]
        match = re.fullmatch(r"codex/release/(web|mobile)-(\d+\.\d+\.\d+)(?:-([1-9]\d*))?", branch)
        if match:
            tags.append({"tag": ("v" if match[1] == "web" else "mobile-v") + match[2],
                         "buildNumber": int(match[3]) if match[3] else None})
    floor, floor_build = "1.0.0", 1
    if product == "mobile":
        pubspec = git("show", f"{sha}:apps/mobile/packages/forum_app/pubspec.yaml")
        match = re.search(r"(?m)^version:\s*(\d+\.\d+\.\d+)\+([1-9]\d*)\s*$", pubspec)
        require(match, "Mobile pubspec must define the version/build floor")
        floor, floor_build = match[1], int(match[2])
    version, number = next_identity(product, bump, tags, floor, floor_build)
    candidate_id = f"{product}-{version}" + (f"-{number}" if number else "")
    return {"schemaVersion": 2, "candidateId": candidate_id, "sourceSha": sha,
            "product": product, "version": version, "tag": ("v" if product == "web" else "mobile-v") + version,
            "buildNumber": number, "channels": channels, "operation": "release", "existingRelease": None,
            "baselines": baselines(github, channels, tags, apple), "notes": {c: FILES[c] for c in channels},
            "serverRequirement": None, "requiredDisclosures": []}


def prepare(manifest, github, folder):
    folder = Path(folder)
    folder.mkdir(parents=True, exist_ok=False)
    request, evidence = collect(manifest, github)
    # Mandatory disclosures are explicit reviewed policy, never inferred from keyword matches
    # in patches. A change mentioning "enabled" cannot establish collection behavior.
    request["requiredDisclosures"] = manifest["requiredDisclosures"]
    write_json(folder / "manifest.json", manifest)
    write_json(folder / "evidence.json", evidence)
    for filename in [*manifest["notes"].values()] + (["operators.zh-CN.md"] if manifest["product"] == "web" else []):
        (folder / filename).write_text("[DRAFT: human review required — complete from evidence.json]\n", encoding='utf-8')
    if manifest["product"] == "mobile":
        write_json(folder / CHANGELOG, {"schemaVersion": 1, "version": manifest["version"],
                   "buildNumber": manifest["buildNumber"], "highlights": [], "breaking": [],
                   "requiredActions": [], "evidence": {}, "testflightNotes": []})
    return request


def create_pr(manifest, folder, github):
    candidate_id = manifest["candidateId"]
    pending = open_request(github, manifest["product"])
    if pending:
        require(pending["candidateId"] == candidate_id, "Another pending candidate owns this product's preparation slot")
        return pending
    base = github.api("git/ref/heads/main")["object"]["sha"]
    # Use Git Data APIs, never push a main branch or execute candidate-supplied scripts.
    tree_entries = []
    for path in sorted(Path(folder).iterdir()):
        require(path.name in {"manifest.json", "evidence.json", "operators.zh-CN.md", CHANGELOG, *FILES.values()} and path.is_file() and not path.is_symlink(), "Invalid prepared file")
        blob = github.api("git/blobs", method="POST", data={"content": path.read_text(encoding='utf-8'), "encoding": "utf-8"})
        tree_entries.append({"path": candidate_path(candidate_id) + "/" + path.name, "mode": "100644", "type": "blob", "sha": blob["sha"]})
    base_tree = github.api(f"git/commits/{base}")["tree"]["sha"]
    tree = github.api("git/trees", method="POST", data={"base_tree": base_tree, "tree": tree_entries})
    commit = github.api("git/commits", method="POST", data={"message": f"chore: prepare {candidate_id}", "tree": tree["sha"], "parents": [base]})
    branch = f"codex/release/{candidate_id}"
    existing = github.api(f"git/ref/heads/{branch}", missing=True)
    if existing:
        raise ReleaseError("Preparation branch already exists; inspect it before retrying (human edits are never overwritten)")
    github.api("git/refs", method="POST", data={"ref": f"refs/heads/{branch}", "sha": commit["sha"]})
    previews = "\n\n".join(f"### {channel}\n\n{(Path(folder) / filename).read_text(encoding='utf-8')}" for channel, filename in manifest["notes"].items())
    body = (f"Release request `{candidate_id}`\n\nSource: `{manifest['sourceSha']}`\n\n"
            f"Targets: {', '.join(manifest['channels'])}\n\n"
            "A release maintainer must review the final head. For schema 2 mobile requests, edit changelog.json, attach evidence refs from evidence.json, then run `python3 scripts/release/workflow.py render-structured --candidate ID` to derive platform files; review TestFlight English separately. Verify platform scope, disclosures and server prerequisites, then approve and merge. Bots cannot approve. The publisher independently rechecks approval.\n\n"
            f"Baselines:\n```json\n{json.dumps(manifest['baselines'], indent=2)}\n```\n\n{previews}\n\n"
            "Evidence and uncertainties: `evidence.json`. Source, targets and note edits require fresh approval.")
    return github.api("pulls", method="POST", data={"title": f"chore: release {candidate_id}", "head": branch, "base": "main", "body": body})


def authorize(candidate_id, github, folder, require_merged=True):
    prs = github.pages(f"pulls?state=all&base=main&head={github.repository.split('/')[0]}:codex/release/{candidate_id}")
    prs = [p for p in prs if p.get("merged_at") or not require_merged]
    require(len(prs) == 1, "Expected exactly one reviewed Release PR; direct pushes cannot publish")
    pr = github.api(f"pulls/{prs[0]['number']}")
    require(pr["head"]["repo"]["full_name"].lower() == github.repository.lower(), "Release PR must belong to this repository")
    reviews = github.pages(f"pulls/{pr['number']}/reviews")
    binding = validate_approval(pr, reviews, github.files(pr["number"], pr["changed_files"]), candidate_id,
                                github.maintainers(reviews), require_merged)
    git("fetch", "origin", "main", f"refs/pull/{pr['number']}/head", "--tags")
    head = pr["head"]["sha"]
    manifest, folder = load_candidate(candidate_id, head, folder)
    git("merge-base", "--is-ancestor", manifest["sourceSha"], "origin/main")
    if require_merged:
        merge = pr["merge_commit_sha"]
        git("merge-base", "--is-ancestor", merge, "origin/main")
        require(not git("diff", "--name-only", head, merge, "--", candidate_path(candidate_id)), "Merged candidate differs from reviewed head")
    checks = github.pages(f"commits/{head}/check-runs", "check_runs")
    required = [c for c in checks if c["name"] == "ci-required" and c.get("app", {}).get("slug") == "github-actions"]
    require(required and max(required, key=lambda c: c["id"])["conclusion"] == "success", "Release PR needs successful ci-required on its final head")
    binding.update({"pr": pr["number"], "sourceSha": manifest["sourceSha"],
                    "notesDigests": {name: hashlib.sha256((folder / name).read_bytes()).hexdigest() for name in manifest['notes'].values()},
                    "changelogDigest": hashlib.sha256((folder / CHANGELOG).read_bytes()).hexdigest()
                    if manifest['product'] == 'mobile' and manifest['schemaVersion'] >= 2 else None,
                    "evidenceDigest": hashlib.sha256((folder / 'evidence.json').read_bytes()).hexdigest()
                    if manifest['product'] == 'mobile' and manifest['schemaVersion'] >= 2 else None,
                    "contentDigest": digest({p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(folder.iterdir())}),
                    "controllerSha": git("rev-parse", "HEAD")})
    dependency = manifest["serverRequirement"]
    if dependency:
        deployed = baselines(github, ["web"], reservations())["web"]["sourceSha"]
        git("merge-base", "--is-ancestor", dependency["sourceSha"], deployed)
    return manifest, binding


def verify_reservation(manifest, binding, github, existing=None):
    tag = manifest['tag']
    existing = existing or github.api(f'git/ref/tags/{tag}', missing=True)
    require(existing and existing['object']['type'] == 'tag', 'An approved annotated reservation is required')
    require(git('rev-parse', f'refs/tags/{tag}^{{commit}}') == manifest['sourceSha'], 'Existing tag has different source')
    annotation = github.api(f"git/tags/{existing['object']['sha']}")
    metadata = json.loads(annotation['message'])
    require(metadata.get('version') == manifest['version'] and metadata.get('buildNumber') == manifest['buildNumber'],
            'Tag version/build identity differs from approved request')
    if manifest['operation'] != 'promote-ios':
        require(metadata.get('candidateId') == manifest['candidateId'] and metadata.get('contentDigest') == binding['contentDigest'],
                'Existing tag belongs to another approval identity')
    else:
        require(metadata.get('candidateId') == manifest['existingRelease']['candidateId'], 'Promotion tag does not belong to original request')


def reserve(manifest, binding, github):
    tag = manifest["tag"]
    existing = github.api(f"git/ref/tags/{tag}", missing=True)
    if existing:
        verify_reservation(manifest, binding, github, existing)
        return
    require(manifest["operation"] == "release", "Promotion requires its original immutable tag")
    metadata = {"schema": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                "candidateId": manifest["candidateId"], "contentDigest": binding["contentDigest"]}
    annotated = github.api("git/tags", method="POST", data={"tag": tag, "message": json.dumps(metadata),
                            "object": manifest["sourceSha"], "type": "commit"})
    github.api("git/refs", method="POST", data={"ref": f"refs/tags/{tag}", "sha": annotated["sha"]})


def emit_outputs(manifest, binding):
    outputs = {"candidate": manifest["candidateId"], "sha": manifest["sourceSha"], "version": manifest["version"],
               "number": str(manifest["buildNumber"] or ""), "tag": manifest["tag"], "channels": json.dumps(manifest["channels"]),
               "approved_head": binding["approvedHead"], "binding": json.dumps(binding, separators=(",", ":")),
               "notes_digests": json.dumps(binding['notesDigests'], separators=(',', ':')),
               "changelog_digest": binding.get('changelogDigest') or '',
               "evidence_digest": binding.get('evidenceDigest') or '',
               "promotion": str(manifest["operation"] == "promote-ios").lower()}
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as out:
            for k, v in outputs.items():
                out.write(f"{k}={v}\n")
    return outputs
