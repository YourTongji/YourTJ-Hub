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
from cycle_store import load as load_cycle
from cycle import current
from model import SHA

CONTROL_ACTIONS = ['start', 'admit', 'advance', 'qa', 'production-qa', 'extend', 'offer',
                   'accept', 'reassign', 'complete', 'cancel', 'abort', 'hotfix', 'backflow']


def parser():
    root = argparse.ArgumentParser(description=__doc__)
    sub = root.add_subparsers(dest="command", required=True)
    for name in ["status", "doctor", "plan", "prepare", "validate", "publish", "retry", "promote-ios", 'cycle', 'prerelease', 'control', 'rc', 'confirm', 'dev']:
        cmd = sub.add_parser(name)
        cmd.add_argument("--json", action="store_true", help="stable schemaVersion=1 JSON output")
        if name in {"plan", "prepare", "doctor", 'rc'}:
            cmd.add_argument("--scope", choices=["web", "android", "ios", "mobile"], default='mobile' if name == 'rc' else 'web')
        if name in {"plan", "prepare", 'rc'}:
            cmd.add_argument('--version', default='', help='Optional final X.Y.Z, including a replacement RC of an unpublished version')
            cmd.add_argument("--source", default="main", help="main or full SHA already in main history")
            cmd.add_argument("--bump", choices=["patch", "minor", "major"], default="patch")
            cmd.add_argument("--ios-destination", choices=["testflight", "app-store"], default='app-store' if name == 'rc' else 'testflight')
            cmd.add_argument("--apple-state", type=Path, help="read-only ASC discovery JSON for local planning")
        if name in {"status", "validate", "publish", "retry", "rc", 'confirm'}:
            cmd.add_argument("--candidate", required=name not in {'status', 'rc'})
        if name == 'dev':
            cmd.add_argument('--source', default='', help='Current dev SHA; empty resolves dev once')
        if name == "retry":
            cmd.add_argument("--channel", choices=["web", "android", "android-alias", "ios-testflight", "ios-app-store"], required=True)
        if name == "promote-ios":
            cmd.add_argument("--release", required=True, help="existing candidate ID")
            cmd.add_argument("--to", choices=["app-store"], default="app-store")
        if name in {'prerelease', 'control'}:
            cmd.add_argument('--cycle', default='')
            cmd.add_argument('--source', default='')
            cmd.add_argument('--issue', default='')
            cmd.add_argument('--channels', default='web,android,ios-testflight,ios-app-store')
        if name == 'prerelease':
            cmd.add_argument('--purpose', choices=['dev'], default='dev')
            cmd.add_argument('--artifacts', default='', help='Comma-separated web,android,ios; default all supported for purpose')
            cmd.add_argument('--source-review-bypass-reason', default='', help='Admin-only, audited source-review exception for this dev preview run')
        if name == 'control':
            cmd.add_argument('--action', choices=CONTROL_ACTIONS, required=True)
            cmd.add_argument('--release-run', default='', help='Successful formal run ID for retrying automatic completion')
            cmd.add_argument('--note', required=True)
            for field in ['pr', 'until', 'assignee', 'base']:
                cmd.add_argument('--' + field, default='')
        if name in {'prepare', 'rc'}:
            cmd.add_argument('--draft-notes', action='store_true', help='Opt in to AI note drafting; default opens an editable request immediately')
        if name in {"prepare", "publish", "retry", "promote-ios", 'prerelease', 'control', 'rc', 'confirm', 'dev'}:
            mode = cmd.add_mutually_exclusive_group()
            mode.add_argument("--apply", action="store_true", help="perform the user-authorized dispatch on main")
            mode.add_argument("--dry-run", action="store_true", help="show request; default, no remote writes")
    return root


def execute(args, github):
    if args.command in {'confirm', 'dev'}:
        if args.command == 'confirm':
            candidate_path(args.candidate)
            workflow, inputs = 'release-confirm.yml', {'candidate': args.candidate}
        else:
            require(not args.source or SHA.fullmatch(args.source), 'Use the current full dev SHA')
            workflow, inputs = 'dev-distribute.yml', {'source': args.source, 'request_run': ''}
        return github.dispatch(workflow, inputs) if args.apply else {'dryRun': True, 'workflow': workflow, 'ref': 'main', 'inputs': inputs}
    if args.command == 'rc':
        from model import ID
        if args.candidate:
            require(ID.fullmatch(args.candidate), 'Select a release candidate ID')
            inputs = {'candidate': args.candidate}
        else:
            destination = args.ios_destination if args.scope in {'ios', 'mobile'} else 'testflight'
            channels_for(args.scope, destination)
            inputs = {'scope': args.scope, 'source': args.source, 'bump': args.bump, 'version': args.version,
                      'ios_destination': destination, 'draft_notes': str(args.draft_notes).lower()}
        return github.dispatch('release-rc.yml', inputs) if args.apply else {
            'dryRun': True, 'workflow': 'release-rc.yml', 'ref': 'main', 'inputs': inputs}
    if args.command == 'cycle':
        return {'releaseCycles': load_cycle(github)}
    if args.command in {'prerelease', 'control'}:
        require(args.command != 'prerelease', 'Development builds follow dev automatically; use dev --apply only to retry current dev')
        fields = ['cycle', 'source', 'issue', 'channels']
        fields += ['purpose', 'artifacts', 'source_review_bypass_reason'] if args.command == 'prerelease' else ['action', 'note', 'pr', 'until', 'assignee', 'base', 'release_run']
        inputs = {key: getattr(args, key) for key in fields}
        require(not args.source or SHA.fullmatch(args.source), 'Use a full source SHA')
        if args.command == 'control' and args.action == 'complete':
            require(args.release_run.isdigit(), 'Completion needs a successful formal release run ID')
        if args.command == 'prerelease':
            from cycle import preview_artifacts
            preview_artifacts(args.artifacts, args.purpose)
            require(not args.source_review_bypass_reason or args.purpose == 'dev'
                    and 0 < len(args.source_review_bypass_reason.strip()) <= 1000,
                    'Source-review bypass is limited to dev previews and requires a reason (1-1000 characters)')
        if args.command == 'prerelease' and not args.cycle:
            require(args.source and args.issue and args.purpose == 'dev', 'A new prerelease needs source and tracking issue')
        workflow = 'release-prerelease.yml' if args.command == 'prerelease' else 'release-control.yml'
        return github.dispatch(workflow, inputs) if args.apply else {'dryRun': True, 'workflow': workflow, 'ref': 'main', 'inputs': inputs}
    if args.command == "doctor":
        run("gh", "auth", "status")
        repo = github.api(f"repos/{github.repository}")
        return {"repository": github.repository, "scope": args.scope, "defaultBranch": repo["default_branch"],
                "permission": repo.get("permissions"), "controllerAvailable": github.api("contents/scripts/release/cli.py?ref=main", missing=True) is not None,
                "appleStatus": "queried only by authenticated Actions discovery" if args.scope in {"ios", "mobile"} else "not_applicable",
                "nextActions": ["rc --scope <scope> --apply", "confirm --candidate <id> --apply"]}
    if args.command == "plan":
        apple = json.loads(args.apple_state.read_text(encoding='utf-8')) if args.apple_state else None
        source = args.source
        return {"state": "draft", "manifest": plan(args.scope, args.bump, source, args.ios_destination, github, apple,
                                                   version=args.version or None), "nextActions": ["rc --scope <scope> --apply"]}
    if args.command == "prepare":
        raise ReleaseError('Preparation is part of rc; use rc --scope <scope> --apply')
    if args.command == "promote-ios":
        candidate_path(args.release)
        inputs = {"scope": "ios", "bump": "patch", "source": "main", "ios_destination": "app-store", "existing_release": args.release}
        return github.dispatch("release-rc.yml", inputs) if args.apply else {
            "dryRun": True, "workflow": "release-rc.yml", "ref": "main", "inputs": inputs, "requiresConfirmation": True}
    if args.command == "status" and not args.candidate:
        prs = github.pages("pulls?state=open&base=main")
        return {"requests": [{"candidateId": p["head"]["ref"].split("/")[-1], "url": p["html_url"]} for p in prs if trusted_request(p, github) and p["head"]["ref"].startswith("codex/release/")],
                "nextActions": ["status --candidate <id>", "rc --apply"]}
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
        return {"candidateId": args.candidate, "valid": True, "contentDigest": content_digest,
                "approval": "requires exact RC confirmation" if manifest['schemaVersion'] == 4 else "requires current-head human GitHub review"}
    if args.command == "publish":
        # A reserved tag does not mean a publisher started. Once any channel has a start
        # receipt, resume original artifacts with Recover rather than rebuilding its identity.
        for channel in manifest['channels']:
            require(latest_receipt(github, args.candidate, channel) is None,
                    'Publication already started; use Recover for the original artifacts')
        inputs = {'candidate': args.candidate}
        return github.dispatch('release-publish.yml', inputs) if args.apply else {
            'dryRun': True, 'workflow': 'release-publish.yml', 'ref': 'main', 'inputs': inputs}
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
