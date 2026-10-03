"""Versioned release protocol. No credentials, GitHub calls or publication here."""
import hashlib
import json
import re
from pathlib import Path

SCHEMA = 2
REPOSITORY = "YourTongji/YourTJ-Hub"
FILES = {"web": "web.zh-CN.md", "android": "android.zh-CN.md",
         "ios-app-store": "ios.zh-Hans.txt", "ios-testflight": "testflight.en-US.txt"}
CHANGELOG = "changelog.json"
SHA = re.compile(r"[0-9a-f]{40}")
ID = re.compile(r"(?:web|mobile)-[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9]+)?(?:-store-[0-9]+)?")
VERSION = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
MAX_BUILD = 2099996000


class ReleaseError(ValueError):
    """An invariant or prerequisite failed; callers must not infer permission."""


def require(condition, message):
    if not condition:
        raise ReleaseError(message)


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"),
                                     ensure_ascii=False).encode()).hexdigest()


def channels_for(scope, destination="testflight"):
    require(scope in {"web", "android", "ios", "mobile"}, "Unknown release scope")
    require(destination in {"testflight", "app-store"}, "Unknown iOS destination")
    require(scope in {"ios", "mobile"} or destination == "testflight", "iOS destination requires an iOS scope")
    channels = ["web"] if scope == "web" else (["android"] if scope in {"android", "mobile"} else [])
    if scope in {"ios", "mobile"}:
        channels += ["ios-testflight"]
        if destination == "app-store":
            channels += ["ios-app-store"]
    return channels


def next_identity(product, bump, reservations, floor="1.0.0", floor_build=1):
    require(product in {"web", "mobile"} and bump in {"patch", "minor", "major"}, "Invalid version request")
    prefix = "v" if product == "web" else "mobile-v"
    versions, numbers = [(0, 0, 0) if product == "web" else tuple(map(int, floor.split(".")))], [floor_build]
    for item in reservations:
        if not item["tag"].startswith(prefix):
            continue
        version = item["tag"][len(prefix):]
        if not VERSION.fullmatch(version):
            continue
        versions.append(tuple(map(int, version.split("."))))
        if product == "mobile":
            require(type(item.get("buildNumber")) is int and 0 < item["buildNumber"] <= MAX_BUILD,
                    f"Unknown build number for {item['tag']}; reconcile its immutable tag first")
            numbers.append(item["buildNumber"])
    version = list(max(versions))
    position = {"major": 0, "minor": 1, "patch": 2}[bump]
    version[position] += 1
    version[position + 1:] = [0] * (2 - position)
    number = max(numbers) + 1 if product == "mobile" else None
    require(number is None or number <= MAX_BUILD, "Mobile build-number space exhausted")
    return ".".join(map(str, version)), number


def read_note(folder, name, channel, draft=False):
    path = Path(folder) / name
    require(path.is_file() and not path.is_symlink(), f"Missing or symlinked note: {name}")
    text = path.read_text(encoding="utf-8").strip()
    return validate_note_text(text, name, channel, draft)


def validate_note_text(text, name, channel, draft=False):
    """Shared shape rules for candidate validation and the bytes read by publishers."""
    require(text and "\x00" not in text and len(text) <= (4000 if channel.startswith("ios-") else 16000),
            f"Empty or oversized note: {name}")
    require(draft or "[DRAFT:" not in text, f"Unfinished draft: {name}")
    if channel.startswith("ios-"):
        require(not re.search(r"(?m)^\s{0,3}(?:#{1,6}\s|```)|\[[^\]]+\]\(|<[^>]+>", text),
                f"Apple notes must be plain text: {name}")
    return text


def validate_candidate(manifest, folder, draft=False):
    required = {"schemaVersion", "candidateId", "sourceSha", "product", "version", "tag", "buildNumber",
                "channels", "operation", "existingRelease", "baselines", "notes", "serverRequirement", "requiredDisclosures"}
    require(isinstance(manifest, dict) and set(manifest) == required, "Unexpected or missing manifest fields")
    require(manifest["schemaVersion"] in {1, SCHEMA}, "Unsupported manifest schema")
    legacy_candidate = manifest["schemaVersion"] == 1
    require(isinstance(manifest["candidateId"], str) and ID.fullmatch(manifest["candidateId"]), "Invalid candidate ID")
    require(isinstance(manifest["sourceSha"], str) and SHA.fullmatch(manifest["sourceSha"]), "Source must be a full commit SHA")
    product, version, channels = manifest["product"], manifest["version"], manifest["channels"]
    require(product in {"web", "mobile"} and isinstance(version, str) and VERSION.fullmatch(version), "Invalid product/version")
    require(manifest["tag"] == ("v" if product == "web" else "mobile-v") + version, "Tag/version namespace mismatch")
    number = manifest["buildNumber"]
    require(number is None if product == "web" else type(number) is int and 0 < number <= MAX_BUILD, "Invalid build number")
    require(isinstance(channels, list) and channels and len(channels) == len(set(channels))
            and set(channels) <= FILES.keys(), "Invalid channels")
    require(channels == ["web"] if product == "web" else "web" not in channels, "Channel/product mismatch")
    stem = f"{product}-{version}" + (f"-{number}" if product == "mobile" else "")
    require(manifest["candidateId"] == stem if manifest["operation"] == "release" else
            manifest["candidateId"].startswith(stem + "-store-"), "Candidate ID does not match release identity")
    require(manifest["operation"] in {"release", "promote-ios"}, "Invalid operation")
    existing = manifest["existingRelease"]
    if manifest["operation"] == "promote-ios":
        require(channels == ["ios-app-store"] and isinstance(existing, dict)
                and set(existing) == {"candidateId", "buildId"}
                and ID.fullmatch(existing["candidateId"]) and re.fullmatch(r"[A-Za-z0-9-]+", existing["buildId"]),
                "Promotion requires an exact existing Apple build and only App Store")
    else:
        require(existing is None, "New release cannot replace an existing build")
    require(isinstance(manifest["baselines"], dict) and set(manifest["baselines"]) == set(channels), "Baseline/channel mismatch")
    for channel, baseline in manifest["baselines"].items():
        require(isinstance(baseline, dict) and set(baseline) <= {"tag", "sourceSha", "buildId", "state", "deploymentId"}
                and {"tag", "sourceSha"} <= set(baseline), f"Invalid baseline: {channel}")
        require((baseline["tag"] is None and baseline["sourceSha"] is None)
                or (isinstance(baseline["sourceSha"], str) and SHA.fullmatch(baseline["sourceSha"])
                    and isinstance(baseline["tag"], str) and re.fullmatch(r"(?:mobile-)?v\d+\.\d+\.\d+", baseline["tag"])),
                f"Unresolved baseline: {channel}")
    require(manifest["notes"] == {c: FILES[c] for c in channels}, "Each channel requires its own canonical note file")
    contents = {c: read_note(folder, name, c, draft) for c, name in manifest["notes"].items()}
    changelog_digest = None
    changelog = None
    if product == "mobile" and not legacy_candidate:
        changelog_path = Path(folder) / CHANGELOG
        require(changelog_path.is_file() and not changelog_path.is_symlink(), "Missing mobile changelog.json")
        changelog_raw = changelog_path.read_bytes()
        changelog_digest = hashlib.sha256(changelog_raw).hexdigest()
        changelog = json.loads(changelog_raw)
        validate_changelog(changelog, version, number, channels, draft=draft)
        evidence_path = Path(folder) / "evidence.json"
        require(evidence_path.is_file() and not evidence_path.is_symlink(), "Missing mobile evidence.json")
        evidence_raw = evidence_path.read_bytes()
        evidence_digest = hashlib.sha256(evidence_raw).hexdigest()
        evidence_record = json.loads(evidence_raw)
        require(evidence_record.get("schemaVersion") == 1 and evidence_record.get("sourceSha") == manifest["sourceSha"],
                "Evidence/source mismatch")
        evidence_items = {item.get("id"): item for item in evidence_record.get("evidence", [])
                          if isinstance(item, dict) and isinstance(item.get("id"), str)}
        for entry_id, references in changelog["evidence"].items():
            entry = next((e for group in ("highlights", "breaking", "requiredActions")
                          for e in changelog[group] if e["id"] == entry_id), None)
            testflight_note = next((item for item in changelog["testflightNotes"]
                                    if item["id"] == entry_id), None)
            evidence_platforms = set()
            for reference in references:
                item = evidence_items.get(reference)
                require(item is not None, f"Unknown evidence {reference} for changelog entry {entry_id}")
                evidence_platforms.update(p for c in item.get("channels", [])
                                          for p in ({"ios", c} if c.startswith("ios-") else {c}))
            if entry is not None:
                require(set(entry["platforms"]) <= evidence_platforms,
                        f"Evidence does not support every changelog platform for {entry_id}")
            if testflight_note is not None:
                require("ios-testflight" in evidence_platforms,
                        f"Evidence does not support TestFlight note {entry_id}")
        for item in changelog["testflightNotes"]:
            require(item["id"] in changelog["evidence"]
                    and set(item["evidenceIds"]) <= set(changelog["evidence"][item["id"]]),
                    f"TestFlight note {item['id']} must reference the same reviewed changelog evidence")
        if not draft:
            require(changelog["highlights"] or changelog["breaking"] or changelog["requiredActions"],
                    "A published mobile candidate needs at least one reviewed changelog entry")
            expected_android = render_changelog(changelog, "android")
            expected_ios = render_changelog(changelog, "ios-app-store")
            expected_testflight = render_changelog(changelog, "ios-testflight")
            if "android" in channels:
                require(contents["android"].strip() == expected_android.strip(),
                        "Android notes differ from the reviewed structured changelog; render them again")
            if "ios-app-store" in channels:
                require(contents["ios-app-store"].strip() == expected_ios.strip(),
                        "App Store notes differ from the reviewed structured changelog; render them again")
            if "ios-testflight" in channels:
                require(contents["ios-testflight"].strip() == expected_testflight.strip(),
                        "TestFlight notes differ from the reviewed structured changelog; render them again")
    else:
        evidence_digest = None
    if product == "web":
        contents["operators"] = read_note(folder, "operators.zh-CN.md", "operators", draft)
    dependency = manifest["serverRequirement"]
    require(dependency is None or (isinstance(dependency, dict) and set(dependency) == {"sourceSha", "reason"}
                                  and SHA.fullmatch(dependency["sourceSha"]) and dependency["reason"].strip()),
            "Server dependency must identify a deployed source SHA and reason")
    require(isinstance(manifest["requiredDisclosures"], list), "Invalid disclosures")
    for disclosure in manifest["requiredDisclosures"]:
        require(isinstance(disclosure, dict) and set(disclosure) == {"id", "channels", "text"}
                and isinstance(disclosure["id"], str) and disclosure["id"]
                and isinstance(disclosure["text"], str) and disclosure["text"]
                and disclosure["channels"] and set(disclosure["channels"]) <= set(channels), "Invalid disclosure")
        if not draft:
            for channel in disclosure["channels"]:
                require(disclosure["text"] in contents[channel], f"Missing disclosure {disclosure['id']} in {channel}")
    value = {"manifest": manifest, "notes": contents}
    if product == "mobile" and not legacy_candidate:
        value.update({"changelogDigest": changelog_digest, "evidenceDigest": evidence_digest})
    return digest(value)


def validate_changelog(value, version, build_number, channels, draft=False):
    require(isinstance(value, dict) and set(value) == {"schemaVersion", "version", "buildNumber", "highlights", "breaking", "requiredActions", "evidence", "testflightNotes"},
            "Invalid structured changelog fields")
    require(value["schemaVersion"] == 1 and value["version"] == version and value["buildNumber"] == build_number,
            "Structured changelog identity differs from release candidate")
    allowed_platforms = {"android", "ios", "ios-testflight", "ios-app-store"}
    supported = {p for channel in channels for p in ({"ios", channel} if channel.startswith("ios-") else {channel})}
    ids = set()
    for group in ("highlights", "breaking", "requiredActions"):
        require(isinstance(value[group], list), f"Invalid changelog {group}")
        for entry in value[group]:
            require(isinstance(entry, dict) and set(entry) == {"id", "title", "summary", "platforms", "kind"},
                    "Invalid changelog entry fields")
            require(isinstance(entry["id"], str) and re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", entry["id"])
                    and entry["id"] not in ids, "Changelog entry IDs must be unique stable slugs")
            ids.add(entry["id"])
            require(entry["kind"] in {"feature", "improvement", "fix", "security"}, "Invalid changelog kind")
            require(isinstance(entry["title"], str) and entry["title"].strip() and len(entry["title"]) <= 100
                    and isinstance(entry["summary"], str) and entry["summary"].strip() and len(entry["summary"]) <= 400,
                    "Empty or oversized changelog text")
            require(draft or ("[DRAFT:" not in entry["title"] and "[DRAFT:" not in entry["summary"]),
                    "Unfinished structured changelog entry")
            require(isinstance(entry["platforms"], list) and entry["platforms"]
                    and len(entry["platforms"]) == len(set(entry["platforms"]))
                    and set(entry["platforms"]) <= allowed_platforms
                    and set(entry["platforms"]) <= supported, "Invalid or unpublished changelog platforms")
    testflight = value["testflightNotes"]
    require(isinstance(testflight, list) and ("ios-testflight" not in channels or draft or testflight),
            "TestFlight release requires structured testing notes")
    for item in testflight:
        require(isinstance(item, dict) and set(item) == {"id", "text", "evidenceIds"}
                and isinstance(item["id"], str) and re.fullmatch(r"[a-z0-9][a-z0-9-]{0,79}", item["id"])
                and isinstance(item["text"], str) and item["text"].strip() and len(item["text"]) <= 500
                and isinstance(item["evidenceIds"], list) and item["evidenceIds"]
                and len(item["evidenceIds"]) == len(set(item["evidenceIds"])), "Invalid TestFlight testing note")
        require(draft or "[DRAFT:" not in item["text"], "Unfinished TestFlight note")
    require(len({item["id"] for item in testflight}) == len(testflight), "Duplicate TestFlight note IDs")
    evidence = value["evidence"]
    evidence_ids = ids | {item["id"] for item in testflight}
    require(isinstance(evidence, dict) and set(evidence) == evidence_ids, "Each changelog entry needs evidence references")
    for entry_id in evidence_ids:
        references = evidence[entry_id]
        require(isinstance(references, list) and references and all(isinstance(ref, str) and ref for ref in references),
                f"Missing source evidence for changelog entry {entry_id}")
    for item in testflight:
        require(set(item["evidenceIds"]) <= set(evidence[item["id"]]),
                f"TestFlight note {item['id']} must use its reviewed evidence references")
    require(len({item["id"] for item in testflight}) == len(testflight), "Duplicate TestFlight note IDs")


def render_changelog(changelog, platform):
    if platform == "ios-testflight":
        return "\n".join(item["text"] for item in changelog["testflightNotes"])
    entries = [entry for group in ("breaking", "requiredActions", "highlights") for entry in changelog[group]
               if platform in entry["platforms"] or (platform.startswith("ios-") and "ios" in entry["platforms"])]
    if platform == "android":
        groups = [(label, [entry for entry in entries if entry in changelog[group]])
                  for group, label in (("breaking", "重要变更"), ("requiredActions", "需要注意"), ("highlights", "本次更新"))]
        return "\n\n".join(f"### {label}\n\n" + "\n".join(f"- **{e['title']}**：{e['summary']}" for e in rows)
                            for label, rows in groups if rows)
    return "\n".join(f"{e['title']}：{e['summary']}" for e in entries)


def validate_approval(pr, reviews, paths, candidate_id, maintainers, require_merged=True):
    require(ID.fullmatch(candidate_id), "Invalid candidate ID")
    require(pr.get("base", {}).get("ref") == "main", "Release PR must target main")
    require(pr.get("head", {}).get("ref") == f"codex/release/{candidate_id}", "Unexpected release PR branch")
    require(not require_merged or pr.get("merged") is True, "Release PR has not merged")
    prefix = f"releases/requests/{candidate_id}/"
    allowed = {"manifest.json", "evidence.json", "operators.zh-CN.md", CHANGELOG, *FILES.values()}
    require(paths and all(p.startswith(prefix) and p[len(prefix):] in allowed for p in paths),
            "Release PR may change only this candidate's release data")
    latest = {}
    for review in sorted(reviews, key=lambda r: r["id"]):
        if review["state"] != "COMMENTED":
            latest[review["user"]["login"]] = review
    require(not any(r["state"] == "CHANGES_REQUESTED" for login, r in latest.items() if login in maintainers),
            "A release maintainer requested changes")
    valid = [r for login, r in latest.items() if login in maintainers and r["user"]["type"] == "User"
             and not login.endswith("[bot]") and r["state"] == "APPROVED" and r["commit_id"] == pr["head"]["sha"]]
    require(valid, "Final PR head needs an eligible human's current APPROVED review")
    return {"approvedHead": pr["head"]["sha"], "mergeSha": pr.get("merge_commit_sha"),
            "reviewIds": [r["id"] for r in valid], "reviewers": [r["user"]["login"] for r in valid]}
