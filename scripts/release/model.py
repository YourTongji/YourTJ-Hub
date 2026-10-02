"""Versioned release protocol. No credentials, GitHub calls or publication here."""
import hashlib
import json
import re
from pathlib import Path

SCHEMA = 1
REPOSITORY = "YourTongji/YourTJ-Hub"
FILES = {"web": "web.zh-CN.md", "android": "android.zh-CN.md",
         "ios-app-store": "ios.zh-Hans.txt", "ios-testflight": "testflight.en-US.txt"}
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
    require(manifest["schemaVersion"] == SCHEMA, "Unsupported manifest schema")
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
    return digest({"manifest": manifest, "notes": contents})


def validate_approval(pr, reviews, paths, candidate_id, maintainers, require_merged=True):
    require(ID.fullmatch(candidate_id), "Invalid candidate ID")
    require(pr.get("base", {}).get("ref") == "main", "Release PR must target main")
    require(pr.get("head", {}).get("ref") == f"codex/release/{candidate_id}", "Unexpected release PR branch")
    require(not require_merged or pr.get("merged") is True, "Release PR has not merged")
    prefix = f"releases/requests/{candidate_id}/"
    allowed = {"manifest.json", "evidence.json", "operators.zh-CN.md", *FILES.values()}
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
