#!/usr/bin/env python3
"""Internal trusted Actions commands. Use cli.py for the public interface."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile
from model import ID, FILES, require, validate_candidate, validate_approval, digest
from github import GitHub, git
from controller import (plan, prepare, create_pr, authorize, reserve, emit_outputs, write_json,
                        candidate_path, load_candidate, verify_reservation, notes_banner)
from state import reservations, baselines, latest_receipt, record
from notes import draft_notes, render, render_structured
from catalog import publish_catalog


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=["prepare", "create-pr", "draft-notes", "render", "render-structured", "publish-catalog", "authorize", "reserve", "receipt", "validate-pr", "review-check", "discover", "start", "verify-apple", "verify-publication", "verify-deploy"])
    parser.add_argument("--candidate", default=os.environ.get("CANDIDATE"))
    parser.add_argument("--folder", type=Path, default=Path(".release-approved"))
    parser.add_argument("--pr", type=int)
    parser.add_argument("--apple-state", type=Path)
    parser.add_argument("--replace-structured", action="store_true")
    args = parser.parse_args()
    github = GitHub()
    if args.command == "publish-catalog":
        catalog = publish_catalog(github)
        print(json.dumps({"schemaVersion": catalog["schemaVersion"], "releaseCount": len(catalog["releases"])}))
        return
    if args.command == "render-structured":
        require(args.candidate, "Structured rendering requires --candidate")
        folder = Path(candidate_path(args.candidate))
        manifest = json.loads((folder / "manifest.json").read_text(encoding="utf-8"))
        require(manifest.get("candidateId") == args.candidate and manifest.get("schemaVersion") == 2
                and manifest.get("product") == "mobile",
                "Structured rendering applies to schema-2 mobile candidates")
        render_structured(manifest, folder, replace=args.replace_structured)
        return
    if args.command == "prepare":
        apple = json.loads(args.apple_state.read_text(encoding='utf-8')) if args.apple_state else None
        existing_id = os.environ.get("EXISTING_RELEASE")
        if existing_id:
            original, _ = authorize(existing_id, github, args.folder / "original")
            receipt = latest_receipt(github, existing_id, "ios-testflight")
            require(receipt and receipt["status"] and receipt["status"]["state"] == "success", "Promotion requires successful original TestFlight receipt")
            build_id = receipt["deployment"]["payload"]["details"]["buildId"]
            found = (apple or {}).get('validBuilds', {}).get(build_id)
            require(found and found['version'] == original['version'] and str(found['buildNumber']) == str(original['buildNumber']),
                    'Discover the exact valid original Apple build before promotion')
            manifest = original | {"candidateId": original["candidateId"] + f"-store-{os.environ['GITHUB_RUN_ID']}",
                         "operation": "promote-ios", "existingRelease": {"candidateId": existing_id, "buildId": build_id},
                         "channels": ["ios-app-store"], "notes": {"ios-app-store": FILES["ios-app-store"]},
                         "baselines": baselines(github, ["ios-app-store"], reservations(), apple), "requiredDisclosures": []}
        else:
            manifest = plan(os.environ["SCOPE"], os.environ["BUMP"], os.environ.get("SOURCE", "main"),
                            os.environ.get("IOS_DESTINATION", "testflight"), github, apple)
        request = prepare(manifest, github, args.folder / "candidate")
        write_json(args.folder / "oryn-input.json", request)
        with open(os.environ["GITHUB_OUTPUT"], "a") as out:
            out.write(f"candidate={manifest['candidateId']}\n")
        return
    if args.command == "draft-notes":
        folder = args.folder / "candidate"
        model_input = (args.folder / "oryn-input.json").resolve()
        output = (args.folder / "oryn-output.json").resolve()

        def run_oryn(timeout):
            # Let Oryn terminate its detached worker before the outer retry deadline.
            # Later attempts may have much less budget than the configured first attempt.
            env = {**os.environ, "ORYN_TASK_TIMEOUT_SECONDS": str(max(10, int(timeout) - 60))}
            subprocess.run(["bun", "script/release-notes.ts", str(model_input), str(output)],
                           cwd=".oryn-runtime", check=True, timeout=timeout, env=env)
        status = draft_notes(json.loads((folder / "manifest.json").read_text(encoding='utf-8')), folder,
                             model_input.read_bytes(), output, run_oryn,
                             attempts=int(os.environ.get("ORYN_NOTES_ATTEMPTS", "3")),
                             budget=float(os.environ.get("ORYN_NOTES_BUDGET_SECONDS", "2400")),
                             task_timeout=float(os.environ.get("ORYN_TASK_TIMEOUT_SECONDS", "1800")) + 60)
        write_json(args.folder / "notes-status.json", status)
        print(json.dumps(status, ensure_ascii=False))
        require(status["status"] == "complete", f"Oryn notes {status['status']}; missing: {', '.join(status['missing'])}")
        return
    if args.command == "render":
        folder = args.folder / "candidate"
        render(json.loads((folder / "manifest.json").read_text(encoding='utf-8')), folder,
               json.loads((args.folder / "oryn-output.json").read_text(encoding='utf-8')), (args.folder / "oryn-input.json").read_bytes())
        return
    if args.command == "create-pr":
        folder = args.folder / "candidate"
        status_path = args.folder / "notes-status.json"
        status = json.loads(status_path.read_text(encoding='utf-8')) if status_path.is_file() else None
        manifest = json.loads((folder / "manifest.json").read_text(encoding='utf-8'))
        result = create_pr(manifest, folder, github, status)
        print(json.dumps(result))
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as out:
            out.write(f"Release request: {result.get('html_url', result.get('url'))}\n\n{notes_banner(manifest, status)}\n\n"
                      "Human review is required before publishing.\n")
        return
    if args.command == "discover":
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding='utf-8'))
        head = os.environ["GITHUB_SHA"]
        paths = git("diff", "--name-only", "--no-renames", event["before"], head, "--", "releases/requests/").splitlines()
        ids = sorted({p.split("/")[2] for p in paths})
        require(len(ids) <= 1, "Publish one candidate per release-data merge")
        with open(os.environ["GITHUB_OUTPUT"], "a") as out:
            out.write("candidate=" + (ids[0] if ids else "") + "\n")
        return
    if args.command in {"review-check", "validate-pr"}:
        event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text(encoding='utf-8'))
        pr_number = args.pr or event.get("pull_request", {}).get("number")
        if args.command == "validate-pr":
            # Read-only PR CI; actual authorization comes from the trusted target controller.
            paths = git("diff", "--name-only", "--no-renames", event.get("pull_request", {}).get("base", {}).get("sha", "HEAD^"), "HEAD", "--").splitlines()
            ids = {p.split("/")[2] for p in paths if p.startswith("releases/requests/") and len(p.split("/")) > 3}
            for candidate_id in ids:
                folder = Path(candidate_path(candidate_id))
                validate_candidate(json.loads((folder / "manifest.json").read_text(encoding='utf-8')), folder)
            if pr_number and event["pull_request"]["head"]["ref"].startswith("codex/release/"):
                require(len(ids) == 1 and all(p.startswith(candidate_path(next(iter(ids))) + "/") for p in paths), "Release PR includes unrelated changes")
            return
        require(pr_number, "Review check requires a PR")
        pr = github.api(f"pulls/{pr_number}")
        paths = github.files(pr_number, pr["changed_files"])
        ids = {p.split("/")[2] for p in paths if p.startswith("releases/requests/") and len(p.split("/")) > 3}
        conclusion, summary = "success", "No release request changes; approval gate does not apply."
        try:
            if ids:
                require(len(ids) == 1, "One release request per PR")
                reviews = github.pages(f"pulls/{pr_number}/reviews")
                validate_approval(pr, reviews, paths, next(iter(ids)), github.maintainers(reviews), False)
                summary = "An eligible human approved the current release request head."
        except ValueError as error:
            conclusion, summary = "failure", str(error)
        github.api("check-runs", method="POST", data={"name": "release-authorized", "head_sha": pr["head"]["sha"],
                   "status": "completed", "conclusion": conclusion, "output": {"title": "Release human review", "summary": summary}})
        return
    manifest, binding = authorize(args.candidate, github, args.folder)
    if args.command == "authorize":
        if os.environ.get('INITIAL_PUBLICATION') == 'true':
            for channel in manifest['channels']:
                require(latest_receipt(github, args.candidate, channel) is None,
                        'Publication already started; use Recover for the original artifacts')
        selected = os.environ.get('REQUESTED_CHANNEL', '')
        require(not selected or (selected.removesuffix('-alias') if selected == 'android-alias' else selected)
                in manifest['channels'], 'Requested channel is outside the approved request')
        emit_outputs(manifest, binding)
    elif args.command in {'verify-publication', 'verify-deploy'}:
        # Run again after a potentially long native build: revoked reviews, changed notes or
        # a moved reservation must stop the next external publication operation.
        verify_reservation(manifest, binding, github)
        if args.command == 'verify-deploy':
            receipt = latest_receipt(github, args.candidate, 'web')
            require(receipt and manifest['channels'] == ['web'], 'No approved web image receipt')
            payload = receipt['deployment']['payload']
            require(payload['binding']['contentDigest'] == binding['contentDigest'], 'Image belongs to different approved content')
            details = payload['details']
            require(os.environ['IMAGE_REF'] == details['image'] and os.environ['BINARY_SHA'] == details['binarySha256']
                    and os.environ['SOURCE_SHA'] == manifest['sourceSha'] and os.environ['VERSION'] == manifest['version'],
                    'Deployment inputs differ from the approved image/binary identity')
    elif args.command == 'verify-apple':
        require(args.apple_state, 'Authenticated current Apple state is required')
        apple = json.loads(args.apple_state.read_text(encoding='utf-8'))
        for channel in json.loads(os.environ['APPLE_CHANNELS']):
            require(channel in manifest['channels'], 'Unapproved Apple channel')
            live = baselines(github, [channel], reservations(), apple)[channel]
            # A previous uncertain attempt may already have distributed this exact candidate.
            same = live.get('tag') == manifest['tag'] and live.get('sourceSha') == manifest['sourceSha']
            require(same or all(live.get(k) == manifest['baselines'][channel].get(k) for k in ('tag', 'sourceSha', 'buildId')),
                    'Apple distribution advanced while awaiting review; refresh the request')
    elif args.command == "reserve":
        # Reconcile baselines immediately before the first public side effect. A same-candidate
        # completed channel is a retry, not a new baseline. Apple is discovered in its publisher.
        for channel in [c for c in manifest["channels"] if not c.startswith("ios-")]:
            prior = latest_receipt(github, args.candidate, channel)
            if prior is None:
                live = baselines(github, [channel], reservations())[channel]
                require(all(live.get(k) == manifest["baselines"][channel].get(k) for k in ("tag", "sourceSha")), "Distribution advanced while awaiting review; refresh the request")
        reserve(manifest, binding, github)
    elif args.command == "start":
        channel = os.environ["CHANNEL"]
        require(channel in manifest["channels"], "Channel is outside the approved request")
        if not channel.startswith('ios-'):
            # Reserve can precede the platform queue. Recheck inside the publisher lock as well,
            # including recovery, so an older candidate cannot overwrite a newer distribution.
            live = baselines(github, [channel], reservations())[channel]
            same = live.get('tag') == manifest['tag'] and live.get('sourceSha') == manifest['sourceSha']
            require(same or all(live.get(k) == manifest['baselines'][channel].get(k) for k in ('tag', 'sourceSha')),
                    'Distribution advanced while queued; prepare a refreshed request')
        prior = latest_receipt(github, args.candidate, channel)
        recover = os.environ.get("RECOVER") == "true"
        if prior:
            require(recover, "Execution already started; use Recover to reuse its original artifacts")
            previous = prior["deployment"]["payload"]
            require(previous["binding"]["contentDigest"] == binding["contentDigest"], "Recovery approval/content changed")
            build_run = previous.get('details', {}).get('buildRunId')
            require(build_run, 'Original build receipt is missing buildRunId; reconcile before recovery')
        else:
            require(not recover or manifest["operation"] == "promote-ios", "No original build receipt for recovery")
            build_run = os.environ["GITHUB_RUN_ID"]
        prior_details = prior["deployment"]["payload"]["details"] if prior else {}
        record(github, manifest, binding, channel, "in_progress", prior_details | {"buildRunId": build_run, "runId": os.environ["GITHUB_RUN_ID"], "availability": "executing"})
        with open(os.environ["GITHUB_OUTPUT"], "a") as out:
            out.write("binary=" + str(prior_details.get("binarySha256", "")) + "\n")
            out.write("image=" + str(prior_details.get("image", "")) + "\n")
            out.write(f"build_run_id={build_run}\n")
            out.write(f"existing_build_id={manifest['existingRelease']['buildId'] if manifest['existingRelease'] else ''}\n")
    elif args.command == "receipt":
        channel = os.environ["CHANNEL"]
        details_file = Path(os.environ.get("RECEIPT_DETAILS", ".release-result.json"))
        details = json.loads(details_file.read_text(encoding='utf-8')) if details_file.exists() else {}
        details = details.get(channel, {}) if set(details).intersection(FILES) else details
        previous = latest_receipt(github, args.candidate, channel)
        require(previous, "Cannot finish an execution that never started")
        require(previous['deployment']['payload'].get('binding', {}).get('contentDigest') == binding['contentDigest'],
                'Receipt approval/content changed')
        details = previous["deployment"]["payload"]["details"] | {k: v for k, v in details.items() if v != ""}
        details.update({"runId": os.environ["GITHUB_RUN_ID"], "attempt": os.environ["GITHUB_RUN_ATTEMPT"]})
        record(github, manifest, binding, channel, os.environ["RELEASE_STATE"], details)


if __name__ == "__main__":
    main()
