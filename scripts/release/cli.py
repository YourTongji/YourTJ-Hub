#!/usr/bin/env python3
"""Portable release client. Git + Python + gh; no signing keys or private runtime required."""
import argparse
import json
from pathlib import Path
import sys
import tempfile
from model import ReleaseError, channels_for, validate_candidate, require
from github import GitHub, git, run
from controller import plan, candidate_path, load_candidate, trusted_request
from state import latest_receipt


def parser():
    root = argparse.ArgumentParser(description=__doc__)
    sub = root.add_subparsers(dest="command", required=True)
    for name in ["status", "doctor", "plan", "prepare", "validate", "retry", "promote-ios"]:
        cmd = sub.add_parser(name)
        cmd.add_argument("--json", action="store_true", help="stable schemaVersion=1 JSON output")
        if name in {"plan", "prepare", "doctor"}:
            cmd.add_argument("--scope", choices=["web", "android", "ios", "mobile"], default="web")
        if name in {"plan", "prepare"}:
            cmd.add_argument("--source", default="main", help="main or full SHA already in main history")
            cmd.add_argument("--bump", choices=["patch", "minor", "major"], default="patch")
            cmd.add_argument("--ios-destination", choices=["testflight", "app-store"], default="testflight")
            cmd.add_argument("--apple-state", type=Path, help="read-only ASC discovery JSON for local planning")
        if name in {"status", "validate", "retry"}:
            cmd.add_argument("--candidate", required=name != "status")
        if name == "retry":
            cmd.add_argument("--channel", choices=["web", "android", "android-alias", "ios-testflight", "ios-app-store"], required=True)
        if name == "promote-ios":
            cmd.add_argument("--release", required=True, help="existing candidate ID")
            cmd.add_argument("--to", choices=["app-store"], default="app-store")
        if name in {"prepare", "retry", "promote-ios"}:
            mode = cmd.add_mutually_exclusive_group()
            mode.add_argument("--apply", action="store_true", help="perform the user-authorized dispatch on main")
            mode.add_argument("--dry-run", action="store_true", help="show request; default, no remote writes")
    return root


def execute(args, github):
    if args.command == "doctor":
        run("gh", "auth", "status")
        repo = github.api(f"repos/{github.repository}")
        return {"repository": github.repository, "scope": args.scope, "defaultBranch": repo["default_branch"],
                "permission": repo.get("permissions"), "controllerAvailable": github.api("contents/scripts/release/cli.py?ref=main", missing=True) is not None,
                "appleStatus": "queried only by authenticated Actions discovery" if args.scope in {"ios", "mobile"} else "not_applicable",
                "nextActions": ["plan", "prepare --apply"]}
    if args.command == "plan":
        apple = json.loads(args.apple_state.read_text(encoding='utf-8')) if args.apple_state else None
        return {"state": "draft", "manifest": plan(args.scope, args.bump, args.source, args.ios_destination, github, apple), "nextActions": ["prepare --apply"]}
    if args.command == "prepare":
        channels_for(args.scope, args.ios_destination)
        inputs = {"scope": args.scope, "bump": args.bump, "source": args.source, "ios_destination": args.ios_destination}
        return github.dispatch("release-prepare.yml", inputs) if args.apply else {"dryRun": True, "workflow": "release-prepare.yml", "ref": "main", "inputs": inputs}
    if args.command == "promote-ios":
        candidate_path(args.release)
        inputs = {"scope": "ios", "bump": "patch", "source": "main", "ios_destination": "app-store", "existing_release": args.release}
        return github.dispatch("release-prepare.yml", inputs) if args.apply else {"dryRun": True, "inputs": inputs, "requiresFreshHumanReview": True}
    if args.command == "status" and not args.candidate:
        prs = github.pages("pulls?state=open&base=main")
        return {"requests": [{"candidateId": p["head"]["ref"].split("/")[-1], "url": p["html_url"]} for p in prs if trusted_request(p, github) and p["head"]["ref"].startswith("codex/release/")],
                "nextActions": ["status --candidate <id>", "prepare --apply"]}
    folder = Path(candidate_path(args.candidate))
    if not folder.exists():
        # Agents normally work on dev, while release data belongs to main/its Release PR.
        # Inspect an isolated copy of the remote request without switching their checkout.
        prs = github.pages(f"pulls?state=all&base=main&head={github.repository.split('/')[0]}:codex/release/{args.candidate}")
        require(len(prs) == 1, 'Expected exactly one remote Release PR for this candidate')
        pr = github.api(f"pulls/{prs[0]['number']}")
        require(pr['head']['repo']['full_name'].lower() == github.repository.lower(), 'Candidate belongs to another repository')
        git('fetch', 'origin', f"refs/pull/{pr['number']}/head")
        with tempfile.TemporaryDirectory() as temporary:
            load_candidate(args.candidate, pr['head']['sha'], temporary, draft=args.command == 'status')
            return inspect_candidate(args, github, Path(temporary))
    return inspect_candidate(args, github, folder)


def inspect_candidate(args, github, folder):
    manifest = json.loads((folder / "manifest.json").read_text(encoding='utf-8'))
    require(manifest['candidateId'] == args.candidate, 'Candidate directory and manifest identity differ')
    content_digest = validate_candidate(manifest, folder, draft=args.command == "status")
    if args.command == "validate":
        return {"candidateId": args.candidate, "valid": True, "contentDigest": content_digest, "approval": "must be checked against live GitHub reviews"}
    if args.command == "retry":
        channel = "android" if args.channel == "android-alias" else args.channel
        require(channel in manifest["channels"], "Recovery cannot widen the approved targets")
        receipt = latest_receipt(github, args.candidate, channel)
        require(receipt is not None, "No execution receipt; use initial approved publication, not a guessed retry")
        inputs = {"candidate": args.candidate, "channel": args.channel}
        return github.dispatch("release-recover.yml", inputs) if args.apply else {"dryRun": True, "inputs": inputs, "receipt": receipt}
    channels = {c: latest_receipt(github, args.candidate, c) for c in manifest["channels"]}
    return {"candidateId": args.candidate, "sourceSha": manifest["sourceSha"], "channels": channels,
            "appleLiveStatus": "unknown; recorded submission is not proof of current store availability",
            "nextActions": ["validate", "retry --channel <approved-channel> --dry-run"]}


def main():
    args = parser().parse_args()
    try:
        result = {"schemaVersion": 1, "ok": True, **execute(args, GitHub())}
    except (ReleaseError, OSError, KeyError, json.JSONDecodeError) as error:
        result = {"schemaVersion": 1, "ok": False, "state": "unknown_or_blocked", "error": str(error)}
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 1
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
