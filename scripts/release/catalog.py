"""Build the public display-only catalog from reviewed candidates and successful receipts."""
from datetime import datetime, timezone
import hashlib
import json
import re
import tempfile
from pathlib import Path
from urllib.parse import urlsplit
from urllib.request import urlopen

from controller import load_candidate
from github import git, run
from model import CHANGELOG, ID, digest, require

CHANNELS = ("android", "ios-app-store", "ios-testflight")
CHANNEL_PLATFORMS = {
    "android": {"android"},
    "ios-app-store": {"ios", "ios-app-store"},
    "ios-testflight": {"ios", "ios-testflight"},
}
IOS_APP_ID = "6809457637"
MAX_LOOKUP_BYTES = 1024 * 1024


def candidate_paths():
    paths = git("ls-tree", "-r", "--name-only", "HEAD", "--", "releases/requests").splitlines()
    return {path.split("/")[2]: path.rsplit("/", 1)[0] for path in paths
            if path.startswith("releases/requests/") and path.endswith("/manifest.json")}


def public_app_store_version():
    try:
        with urlopen(f"https://itunes.apple.com/lookup?id={IOS_APP_ID}&country=cn", timeout=5) as response:
            final = response.geturl()
            final_url = urlsplit(final)
            if final_url.scheme != "https" or final_url.hostname != "itunes.apple.com" or final_url.port not in (None, 443):
                return None
            length = response.headers.get("content-length")
            if length and int(length) > MAX_LOOKUP_BYTES:
                return None
            raw = response.read(MAX_LOOKUP_BYTES + 1)
            if len(raw) > MAX_LOOKUP_BYTES:
                return None
            payload = json.loads(raw)
        rows = payload.get("results", [])
        result = rows[0] if len(rows) == 1 else {}
        version = result.get("version") if result.get("trackId") == int(IOS_APP_ID) and result.get("bundleId") == "tj.yourtj.forumApp" else None
        return version if isinstance(version, str) else None
    except (OSError, ValueError, KeyError, TypeError):
        return None


def successful_channels(github, store_version=None, verified_store_builds=()):
    result = {}
    for channel in CHANNELS:
        for deployment in github.pages(f"deployments?environment=release-{channel}"):
            payload = deployment.get("payload") or {}
            candidate_id = payload.get("candidateId") if isinstance(payload, dict) else None
            if not isinstance(candidate_id, str) or not ID.fullmatch(candidate_id):
                continue
            statuses = github.pages(f"deployments/{deployment['id']}/statuses")
            if not statuses or not isinstance(statuses[0], dict) or statuses[0].get("state") != "success":
                continue
            details = payload.get("details") if isinstance(payload, dict) else None
            availability = details.get("availability") if isinstance(details, dict) else None
            if channel == "android" and availability != "available":
                continue
            if channel == "ios-testflight" and availability != "APPROVED":
                continue
            if channel == "ios-app-store" and availability not in {"READY_FOR_SALE", "READY_FOR_DISTRIBUTION", "REPLACED_WITH_NEW_VERSION"}:
                # App Store Connect submission receipts are not proof of public availability.
                build = details.get("buildNumber") if isinstance(details, dict) else None
                build = int(build) if isinstance(build, (int, str)) and str(build).isdigit() else None
                identity = (payload.get("tag", "").removeprefix("mobile-v"), build)
                if identity not in verified_store_builds and (not store_version or identity[0] != store_version):
                    continue
            if channel == "android":
                assets = details.get("assets", {}) if isinstance(details, dict) else {}
                apks = [name for name in assets if name.endswith(".apk")]
                require(len(apks) == 3 and all(isinstance(assets[name], str)
                                               and re.fullmatch(r"[0-9a-f]{64}", assets[name]) for name in apks),
                        "Successful Android receipt has no verified APK digests")
            result.setdefault(candidate_id, {}).setdefault(channel, {"payload": payload, "sha": deployment.get("sha")})
    return result


def load_published_candidate(candidate_id, folder):
    manifest, candidate_folder = load_candidate(candidate_id, "HEAD", folder)
    require(manifest["product"] == "mobile" and manifest["candidateId"] == candidate_id,
            "Receipt points to a non-mobile or mismatched candidate")
    return manifest, candidate_folder


def candidate_content_digest(folder):
    return digest({path.name: hashlib.sha256(path.read_bytes()).hexdigest()
                   for path in sorted(folder.iterdir())})


def newer_store_correction(first, second):
    """Choose reviewed copy by main merge order, never by a publisher's retry time."""
    identity = ("version", "buildNumber", "tag", "sourceSha")
    require(all(first[key] == second[key] for key in identity), "Store correction changes binary identity")
    original = f"mobile-{first['version']}-{first['buildNumber']}"
    build_ids = set()
    for manifest in (first, second):
        if manifest.get("operation") == "release":
            require(manifest["candidateId"] == original, "Store correction has an unrelated original")
        else:
            existing = manifest.get("existingRelease") or {}
            require(manifest.get("operation") == "promote-ios" and existing.get("candidateId") == original,
                    "Store correction must promote the same original candidate")
            build_ids.add(existing.get("buildId"))
    require(len(build_ids) == 1, "Store corrections must use the same Apple build")
    paths = {f"releases/requests/{m['candidateId']}/manifest.json": m["candidateId"] for m in (first, second)}
    history = git("log", "--first-parent", "--format=", "--name-only", "HEAD", "--", *paths).splitlines()
    newest = next((paths[path] for path in history if path in paths), None)
    require(newest is not None, "Store correction has no merged candidate history")
    return newest


def build_catalog(github, published_at=None, verified_store_builds=()):
    paths = candidate_paths()
    receipts = successful_channels(github, public_app_store_version(), verified_store_builds)
    by_build = {}
    for candidate_id, receipt_channels in receipts.items():
        channels = set(receipt_channels)
        require(candidate_id in paths, "Successful mobile receipt has no merged reviewed release request")
        with tempfile.TemporaryDirectory() as temporary:
            manifest, folder = load_published_candidate(candidate_id, Path(temporary))
            require(set(channels) <= set(manifest["channels"]), "Receipt includes an unapproved release channel")
            # Every successful deployment is tied to the immutable candidate bytes by its binding.
            for channel in sorted(channels):
                receipt = receipt_channels[channel]
                payload = receipt["payload"]
                require(payload.get("candidateId") == candidate_id and payload.get("channel") == channel,
                        "Successful receipt identity/channel mismatch")
                binding = payload.get("binding") or {}
                require(payload.get("tag") == manifest["tag"]
                        and receipt.get("sha") == manifest["sourceSha"]
                        and binding.get("sourceSha") == manifest["sourceSha"]
                        and binding.get("contentDigest") == candidate_content_digest(folder),
                        "Successful receipt does not match merged reviewed candidate bytes")
                if manifest.get("operation") == "promote-ios":
                    require((payload.get("details") or {}).get("buildId") == manifest["existingRelease"]["buildId"],
                            "Promotion receipt does not match the reviewed Apple build")
            record = by_build.setdefault((manifest["version"], manifest["buildNumber"]),
                                         {"version": manifest["version"], "buildNumber": manifest["buildNumber"],
                                          "channelCandidates": {}})
            for channel in channels:
                existing = record["channelCandidates"].get(channel)
                if existing is not None and existing[0]["candidateId"] != manifest["candidateId"]:
                    require(channel == "ios-app-store", "Two successful release candidates claim the same build/channel")
                    if newer_store_correction(existing[0], manifest) != manifest["candidateId"]:
                        continue
                record["channelCandidates"][channel] = (manifest, folder / CHANGELOG if manifest["schemaVersion"] >= 2 else None)
                # Keep the temp directory alive until the generated catalog is written below.
                # The caller receives only copied JSON values, so copy the structured source now.
                changelog_path = record["channelCandidates"][channel][1]
                record["channelCandidates"][channel] = (manifest, json.loads(changelog_path.read_text(encoding="utf-8"))
                                                         if changelog_path and changelog_path.is_file() else None)

    releases = []
    for (version, build_number), record in sorted(by_build.items(), key=lambda item: item[0][1], reverse=True):
        channel_candidates = record["channelCandidates"]
        structured_channels = {channel for channel, (_, changelog) in channel_candidates.items() if changelog is not None}
        output = {"version": version, "buildNumber": build_number,
                  "channels": sorted(structured_channels), "highlights": [], "breaking": [],
                  "requiredActions": [], "testflightNotes": []}
        for channel, (_, changelog) in sorted(channel_candidates.items()):
            if changelog is None:
                continue
            allowed = CHANNEL_PLATFORMS[channel]
            for group in ("highlights", "breaking", "requiredActions"):
                for entry in changelog[group]:
                    platforms = sorted({channel} if set(entry["platforms"]) & allowed else set())
                    if not platforms:
                        continue
                    public = {key: entry[key] for key in ("id", "title", "summary", "kind")}
                    public["platforms"] = platforms
                    existing = next((item for item in output[group] if item["id"] == public["id"]), None)
                    if existing:
                        require(all(existing[key] == public[key] for key in ("title", "summary", "kind")),
                                f"Conflicting platform text for shared changelog ID {public['id']}")
                        existing["platforms"] = sorted(set(existing["platforms"]) | set(platforms))
                    else:
                        output[group].append(public)
            if channel == "ios-testflight":
                output["testflightNotes"].extend({"id": item["id"], "text": item["text"]}
                                                for item in changelog["testflightNotes"])
        if structured_channels:
            releases.append(output)

    all_builds = {(version, build): record for (version, build), record in by_build.items()}
    coverage = {}
    for channel in CHANNELS:
        published_builds = sorted((build for (_, build), record in all_builds.items()
                                  if channel in record["channelCandidates"]))
        covered = []
        for build in reversed(published_builds):
            record = next(value for (_, candidate_build), value in all_builds.items() if candidate_build == build)
            _, changelog = record["channelCandidates"][channel]
            if changelog is None:
                break
            covered.append(build)
        covered.reverse()
        if covered:
            # This is an inclusive installed-build floor, not an entry-range bound. Without a
            # known older public build, the first structured build is the floor; earlier history is unknown.
            previous = [build for build in published_builds if build < covered[0]]
            coverage[channel] = {"completeFromBuild": max(previous) if previous else covered[0],
                                 "throughBuild": covered[-1], "coveredBuilds": covered}
    return {"schemaVersion": 1, "historyCoverage": {"source": "github-release-receipts",
            "publishedAt": published_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            "byChannel": coverage}, "releases": releases}


def previous_store_builds(github, release):
    assets = {item.get("name"): item for item in release.get("assets", [])}
    if "releases.json" not in assets:
        return set()
    with tempfile.TemporaryDirectory() as temporary:
        run("gh", "release", "download", "mobile-notes", "--repo", github.repository,
            "--pattern", "releases.json", "--dir", temporary)
        path = Path(temporary) / "releases.json"
        require(path.is_file() and path.stat().st_size <= 1024 * 1024, "Previous release catalog is missing or oversized")
        previous = json.loads(path.read_text(encoding="utf-8"))
    require(previous.get("schemaVersion") == 1 and isinstance(previous.get("releases"), list)
            and isinstance(previous.get("historyCoverage"), dict), "Previous release catalog is invalid")
    store = previous["historyCoverage"].get("byChannel", {}).get("ios-app-store", {})
    covered = set(store.get("coveredBuilds", []))
    return {(release["version"], release["buildNumber"]) for release in previous["releases"]
            if "ios-app-store" in release.get("channels", []) and release.get("buildNumber") in covered}


def publish_catalog(github):
    release = github.api("releases/tags/mobile-notes", missing=True)
    verified_store = previous_store_builds(github, release) if release else set()
    catalog = build_catalog(github, verified_store_builds=verified_store)
    content = json.dumps(catalog, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n"
    with tempfile.TemporaryDirectory() as temporary:
        path = Path(temporary) / "releases.json"
        path.write_text(content, encoding="utf-8")
        if release is None:
            github.api("releases", method="POST", data={"tag_name": "mobile-notes", "name": "Mobile release notes",
                        "body": "Display-only catalog derived from reviewed release candidates and successful channel receipts.",
                        "draft": False, "prerelease": True})
        run("gh", "release", "upload", "mobile-notes", str(path), "--repo", github.repository, "--clobber")
    return catalog
