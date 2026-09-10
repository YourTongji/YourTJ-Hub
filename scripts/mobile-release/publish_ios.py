#!/usr/bin/env python3
"""Upload once, prepare reviewed metadata, submit TestFlight and App Store.

Run only from the protected mobile release job. A processing timeout is resumed
by querying the existing upload, never by blindly re-uploading the same version.
"""
import json
import os
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[2]
APP_ID = "6809457637"
GROUP_ID = "25249074-709f-45d7-a2f6-950136ecf636"
REVIEW_NOTES = (
    "YourTJ is a campus course-selection community. Guests can browse courses, reviews, "
    "community posts and campus Wiki. Use account/password login with the supplied account. "
    "A visual captcha may be required. User-generated posts, comments and messaging are "
    "supported, with reporting and account settings in-app. The scheduler only plans courses; "
    "it does not enroll students. Full scheduler: https://f.yourtj.de/schedule ."
)
REVIEW_FIELDS = {"firstName": "contact-first-name", "lastName": "contact-last-name", "email": "contact-email",
                        "phone": "contact-phone", "username": "demo-account-name", "password": "demo-account-password"}
ACCEPTED_STATES = {"WAITING_FOR_REVIEW", "IN_REVIEW", "PENDING_DEVELOPER_RELEASE",
                   "PENDING_APPLE_RELEASE", "READY_FOR_DISTRIBUTION", "READY_FOR_SALE", "PROCESSING_FOR_APP_STORE"}


def asc(*args, allow_missing=False):
    result = subprocess.run([os.environ.get("ASC_BIN", "asc"), *args],
                            capture_output=True, text=True)
    if result.returncode:
        # ASC 5 reports an absent beta review with exit 4 and a domain message,
        # without an HTTP 404. This is expected before a build's first submission.
        absent_beta_review = (args[:3] == ("builds", "beta-app-review-submission", "view")
                              and result.returncode == 4
                              and "no beta app review submission found for build" in result.stderr)
        if allow_missing and ("404" in result.stderr or "NOT_FOUND" in result.stderr or absent_beta_review):
            return {"data": None}
        # Never echo argv: review fields include private credentials.
        raise RuntimeError(f"ASC {args[0]} {args[1]} failed (exit {result.returncode}); inspect the corresponding App Store Connect status")
    return json.loads(result.stdout) if result.stdout.strip() else {}


def resource(result):
    value = result.get("data", result)
    # Some ASC single-app lookups return a one-element JSON:API collection.
    if isinstance(value, list):
        if len(value) > 1:
            raise ValueError("Ambiguous ASC resource collection")
        return value[0] if value else {}
    return value or {}


def find_build(version, number):
    data = asc("builds", "list", "--app", APP_ID, "--version", version,
               "--build-number", number, "--processing-state", "all")["data"]
    if len(data) > 1:
        raise ValueError("Ambiguous build identity")
    return data[0] if data else None


def wait_for_build(version, number):
    deadline = time.monotonic() + 1800
    while time.monotonic() < deadline:
        build = find_build(version, number)
        if build:
            state = build["attributes"]["processingState"]
            if state == "VALID":
                return build["id"]
            if state in {"FAILED", "INVALID"}:
                raise ValueError(f"Apple rejected build processing: {state}")
        time.sleep(30)
    raise TimeoutError("Apple is still processing this upload; rerun to resume")


def main():
    os.environ["ASC_TELEMETRY_DISABLED"] = "1"
    os.environ.setdefault("ASC_TIMEOUT", "90s")
    version = os.environ["MOBILE_VERSION"]
    number = os.environ["MOBILE_BUILD_NUMBER"]
    target = os.environ.get("IOS_PUBLISH_TARGET", "both")
    if target not in {"both", "testflight"}:
        raise ValueError("Invalid iOS publish target; choose both or testflight")
    if find_build(version, number) is None:
        if os.environ.get("IOS_EXISTING_BUILD_ONLY") == "true":
            raise ValueError("iOS recovery requires an existing uploaded build with the exact version/build number")
        ipa = os.environ["IOS_IPA_PATH"]
        uploads = asc("builds", "uploads", "list", "--app", APP_ID,
                      "--cf-bundle-short-version", version, "--cf-bundle-version", number, "--paginate")["data"]
        matching = [u for u in uploads if u["attributes"].get("cfBundleShortVersionString") == version
                    and u["attributes"].get("cfBundleVersion") == number]
        if not matching:
            asc("builds", "upload", "--app", APP_ID, "--ipa", ipa,
                "--version", version, "--build-number", number)
        elif not any(u["attributes"].get("uploadedDate") for u in matching):
            raise ValueError("An incomplete upload reservation exists; inspect ASC build uploads before retrying")
    build_id = wait_for_build(version, number)
    asc("builds", "update", "--build-id", build_id, "--uses-non-exempt-encryption=false")

    beta = resource(asc("builds", "beta-app-review-submission", "view", "--build-id", build_id, allow_missing=True))
    if beta.get("attributes", {}).get("betaReviewState") not in {"WAITING_FOR_REVIEW", "IN_REVIEW", "APPROVED"}:
        contact = json.loads(os.environ["IOS_REVIEW_JSON"])
        beta_details = resource(asc("testflight", "review", "view", "--app", APP_ID))
        args = ["testflight", "review", "edit", "--id", beta_details["id"]]
        for field, flag in REVIEW_FIELDS.items():
            args.extend([f"--{flag}", contact[field]])
        asc(*args, "--demo-account-required=true", "--notes", REVIEW_NOTES)
        asc("publish", "testflight", "--app", APP_ID, "--build-id", build_id, "--group", GROUP_ID,
            "--test-notes", "Test course search and reviews, scheduler, Wiki, community and account features.",
            "--locale", "en-US", "--submit", "--confirm")
    print(f"TestFlight build: {build_id}", flush=True)
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a") as output:
            output.write(f"TestFlight: {version} ({number}), build `{build_id}`.\n\n")
    if target == "testflight":
        message = "App Store submission skipped by request; the existing review queue is unchanged."
        print(message)
        if summary:
            with open(summary, "a") as output:
                output.write(message + "\n")
        return

    versions = asc("versions", "list", "--app", APP_ID, "--platform", "IOS", "--include", "build", "--paginate")["data"]
    matching = [v for v in versions if v["attributes"]["versionString"] == version]
    if len(matching) > 1:
        raise ValueError("Ambiguous App Store version identity")
    if matching:
        current = matching[0]
        state = current["attributes"].get("appStoreState")
        attached = current.get("relationships", {}).get("build", {}).get("data") or {}
        if state in ACCEPTED_STATES:
            if attached.get("id") != build_id:
                raise ValueError("This App Store version already uses a different build")
            print(f"App Store version already submitted: {version} ({state})")
            return
        version_id = current["id"]
    else:
        released = {"READY_FOR_SALE", "READY_FOR_DISTRIBUTION", "REPLACED_WITH_NEW_VERSION",
                    "REMOVED_FROM_SALE", "DEVELOPER_REMOVED_FROM_SALE"}
        for other in versions:
            attributes = other["attributes"]
            state = attributes.get("appStoreState")
            if state not in released:
                raise ValueError(
                    f"App Store {attributes['versionString']} is {state}; it blocks creation of {version}. "
                    "Preserve its review queue using testflight-only recovery, or resolve the existing version in ASC."
                )
        created = resource(asc("versions", "create", "--app", APP_ID, "--version", version,
                               "--platform", "IOS", "--release-type", "AFTER_APPROVAL",
                               "--copyright", "2026 YourTJ Open Source"))
        version_id = created["id"]

    localizations = {l["attributes"]["locale"]: l["id"] for l in
                     asc("localizations", "list", "--version", version_id, "--paginate")["data"]}
    fields = {"description": "description", "keywords": "keywords", "marketingUrl": "marketing-url",
              "supportUrl": "support-url", "whatsNew": "whats-new"}
    store = Path(os.environ.get("IOS_STORE_PATH", str(ROOT / "apps/mobile/store")))
    for folder in sorted(store.iterdir()):
        if not folder.is_dir() or not (folder / "metadata.json").exists():
            continue
        locale = folder.name
        values = json.loads((folder / "metadata.json").read_text())
        if locale not in localizations:
            created = resource(asc("localizations", "create", "--version", version_id, "--locale", locale))
            localizations[locale] = created["id"]
        args = ["localizations", "update", "--id", localizations[locale]]
        for key, flag in fields.items():
            if key in values:
                args.extend([f"--{flag}", values[key]])
        asc(*args)
        for kind, display in [("iphone", "IPHONE_69"), ("ipad", "IPAD_PRO_3GEN_129")]:
            asc("screenshots", "upload", "--version-localization", localizations[locale],
                "--path", str(folder / kind), "--device-type", display, "--skip-existing")

    contact = json.loads(os.environ["IOS_REVIEW_JSON"])
    details = resource(asc("review", "details-for-version", "--version-id", version_id, allow_missing=True))
    args = ["review", "details-update", "--id", details["id"]] if details.get("id") else [
        "review", "details-create", "--version-id", version_id]
    for field, flag in REVIEW_FIELDS.items():
        args.extend([f"--{flag}", contact[field]])
    asc(*args, "--demo-account-required=true", "--notes", REVIEW_NOTES)
    asc("versions", "attach-build", "--version-id", version_id, "--build-id", build_id)
    asc("validate", "--app", APP_ID, "--version", version, "--strict")
    submitted = asc("review", "submit", "--app", APP_ID, "--version-id", version_id,
                    "--build-id", build_id, "--confirm")
    print(json.dumps({"version": version, "buildId": build_id, "submissionId": submitted.get("submissionId")}))


if __name__ == "__main__":
    main()
