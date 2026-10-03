"""Render platform notes only from validated evidence and reviewed structured facts."""
import hashlib
import json
import os
import re
from pathlib import Path
import shutil
import tempfile
import time

from model import CHANGELOG, FILES, ReleaseError, require, validate_candidate, render_changelog


def render(manifest, folder, response, model_input, replace=False):
    require(isinstance(response, dict), "Invalid Oryn result")
    require(response.get("schemaVersion") == 1 and response.get("promptVersion") == 1, "Unsupported Oryn result")
    require(response.get("inputSha256") == hashlib.sha256(model_input).hexdigest(), "Oryn input digest mismatch")
    request = json.loads(model_input)
    result = response["output"]
    require(isinstance(result, dict) and set(result) == {"schemaVersion", "entries", "uncertainties"}
            and result["schemaVersion"] == 1 and isinstance(result["entries"], list)
            and isinstance(result["uncertainties"], list), "Invalid notes output")
    if manifest["product"] == "mobile" and manifest["schemaVersion"] >= 2:
        _draft_changelog(manifest, folder, request, result, response, replace)
        return
    _render_legacy(manifest, folder, request, result, response, replace)


MIN_ATTEMPT_SECONDS = 60


def draft_notes(manifest, folder, model_input, output_path, run_oryn, attempts=3, budget=2400.0,
                task_timeout=1860.0, sleep=time.sleep, clock=time.monotonic):
    """Retry Oryn until its draft passes rendering and covers every channel.

    Each attempt renders into a scratch copy, so a malformed or partial result never touches the
    request. The best valid partial draft is kept for human completion; publishing still requires
    reviewed, complete notes.
    """
    folder, output_path = Path(folder), Path(output_path)
    deadline = clock() + budget
    history, best, stopped = [], None, None
    for attempt in range(1, attempts + 1):
        remaining = deadline - clock()
        if remaining < MIN_ATTEMPT_SECONDS:
            stopped = "Draft time budget exhausted"
            break
        output_path.unlink(missing_ok=True)
        try:
            run_oryn(min(task_timeout, remaining))
            response = json.loads(output_path.read_text(encoding="utf-8"))
            with tempfile.TemporaryDirectory(dir=folder.parent) as temporary:
                trial = Path(temporary) / folder.name
                shutil.copytree(folder, trial)
                render(manifest, trial, response, model_input)
                missing = draft_gaps(manifest, trial)
        except Exception as error:  # Any provider, transport or format failure is retryable.
            history.append({"attempt": attempt, "error": f"{type(error).__name__}: {error}"[:300]})
        else:
            history.append({"attempt": attempt, "error": None, "missing": missing})
            if best is None or len(missing) < len(best[1]):
                best = (response, missing)
            if not missing:
                break
        if attempt < attempts:
            sleep(max(0, min(30 * attempt, deadline - clock() - MIN_ATTEMPT_SECONDS)))
    if best is not None:
        render(manifest, folder, best[0], model_input)
    missing = draft_gaps(manifest, folder)
    status = "complete" if not missing else "partial" if best is not None else "failed"
    return {"status": status, "missing": missing, "attempts": history, "stopped": stopped}


def draft_gaps(manifest, folder):
    """User-facing channels whose platform note is still an unfinished draft."""
    return [channel for channel, filename in manifest["notes"].items()
            if "[DRAFT:" in (Path(folder) / filename).read_text(encoding="utf-8")]


ORYN_FIELDS = {"channel", "text", "evidenceIds"}
# Optional structured fields a newer Oryn release-notes task may return; validated before use.
ORYN_OPTIONAL = {"kind", "title", "summary", "required"}
KINDS = ("security", "feature", "fix", "improvement")
TYPE_KINDS = {"feat": "feature", "feature": "feature", "fix": "fix", "bugfix": "fix", "hotfix": "fix",
              "security": "security", "sec": "security"}
CONVENTIONAL = re.compile(r"^(\w+)(?:\([^)]*\))?(!)?:")
SECURITY_WORDS = re.compile(r"security|vulnerab|cve-|安全|漏洞", re.I)
CHANNEL_SUFFIX = {"android": "", "ios-app-store": "-ios", "ios-testflight": "-beta"}


def _pr_type(pr):
    """Change type from a conventional PR title, else its branch prefix (owner/feat/topic)."""
    match = CONVENTIONAL.match(pr.get("title") or "")
    kind = match[1].lower() if match else None
    branch = (pr.get("branch") or "").split("/")
    if kind is None and len(branch) >= 3:
        kind = branch[1].lower()
    if SECURITY_WORDS.search(pr.get("title") or ""):
        kind = "security"
    return TYPE_KINDS.get(kind, "improvement"), bool(match and match[2])


def _split_text(text):
    """Short title plus the rest of the sentence, without repeating the title in the summary."""
    for pattern in (r"^(.{2,24}?)[：:]\s*(\S.*)$", r"^(.{2,24}?)[。！？!?；;\n]\s*(\S.*)$"):
        match = re.match(pattern, text, re.S)
        if match:
            return match[1].strip(), match[2].strip()
    clause = re.split(r"[，,。！？!?；;\n]", text, maxsplit=1)[0].strip()
    if 2 <= len(clause) <= 24:
        return clause, text
    return text[:23].rstrip() + "…", text


def _stable_id(base, used):
    candidate, n = base, 2
    while candidate in used:
        candidate, n = f"{base}-{n}", n + 1
    used.add(candidate)
    return candidate


def _validate_oryn_metadata(item):
    require("kind" not in item or item["kind"] in KINDS, "Invalid Oryn kind")
    for field, limit in (("title", 100), ("summary", 400)):
        require(field not in item or isinstance(item[field], str) and 0 < len(item[field].strip()) <= limit,
                f"Invalid Oryn {field}")
    require("required" not in item or isinstance(item["required"], bool), "Invalid Oryn required flag")


def _draft_changelog(manifest, folder, request, result, response, replace=False):
    folder = Path(folder)
    changelog_path = folder / CHANGELOG
    require(replace or not changelog_path.exists() or _is_initial_changelog_scaffold(changelog_path, folder, manifest),
            "Preserving existing structured changelog; explicit regeneration requires new review")
    for filename in manifest["notes"].values():
        path = folder / filename
        require(not path.is_symlink(), f"Cannot render through a symlink: {filename}")
        require(replace or not path.exists() or "[DRAFT:" in path.read_text(encoding="utf-8"),
                f"Preserving human edits in {filename}; explicit regeneration requires new review")
    evidence = {item["id"]: item for item in request["evidence"]}
    entries = {channel: [] for channel in manifest["channels"]}
    for item in result["entries"]:
        require(isinstance(item, dict) and ORYN_FIELDS <= set(item) <= ORYN_FIELDS | ORYN_OPTIONAL,
                "Invalid Oryn changelog draft")
        channel = item["channel"]
        require(channel in entries and isinstance(item["text"], str) and item["text"].strip()
                and len(item["text"].strip()) <= (500 if channel == "ios-testflight" else 400)
                and isinstance(item["evidenceIds"], list) and item["evidenceIds"], "Invalid Oryn changelog draft")
        require(all(reference in evidence and channel in evidence[reference]["channels"]
                    for reference in item["evidenceIds"]), "Unknown or cross-platform changelog evidence")
        _validate_oryn_metadata(item)
        entries[channel].append(item)

    evidence_path = folder / "evidence.json"
    record = json.loads(evidence_path.read_text(encoding="utf-8"))
    prs = {pr["number"]: pr for pr in record.get("pullRequests", []) if isinstance(pr, dict)}
    links = record.get("evidencePullRequests", {})
    changelog = {"schemaVersion": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                 "highlights": [], "breaking": [], "requiredActions": [], "evidence": {}, "testflightNotes": []}
    used = set()
    for channel, items in entries.items():
        for item in items:
            text = item["text"].strip()
            references = list(dict.fromkeys(item["evidenceIds"]))
            numbers = sorted({n for reference in references for n in links.get(reference, [])})
            # PR numbers are stable across releases; text hashes stay a reviewer-visible fallback.
            base = (f"pr-{numbers[0]}{CHANNEL_SUFFIX[channel]}" if numbers
                    else "oryn-" + hashlib.sha256((channel + "\0" + text).encode()).hexdigest()[:16])
            stable_id = _stable_id(base, used)
            changelog["evidence"][stable_id] = references
            if channel == "ios-testflight":
                changelog["testflightNotes"].append({"id": stable_id, "text": text, "evidenceIds": references})
                continue
            types = [_pr_type(prs[n]) for n in numbers if n in prs]
            kind = item.get("kind") if item.get("kind") in KINDS else next(
                (k for k in KINDS if k in {t for t, _ in types}), "improvement")
            title, summary = _split_text(text)
            if isinstance(item.get("title"), str) and 0 < len(item["title"].strip()) <= 100:
                title = item["title"].strip()
                if isinstance(item.get("summary"), str) and 0 < len(item["summary"].strip()) <= 400:
                    summary = item["summary"].strip()
            group = "breaking" if item.get("required") is True or any(b for _, b in types) else "highlights"
            changelog[group].append({"id": stable_id, "title": title, "summary": summary,
                                     "platforms": [channel], "kind": kind})
    # Reviewed manifest disclosures must appear verbatim; they become required entries.
    for item in record.get("evidence", []):
        disclosure = next((d for d in manifest["requiredDisclosures"] if d["id"] == item.get("disclosureId")), None)
        if disclosure is None:
            continue
        slug = re.sub(r"[^a-z0-9]+", "-", disclosure["id"].lower()).strip("-")[:60] or item["id"]
        stable_id = _stable_id(slug if re.fullmatch(r"[a-z0-9][a-z0-9-]*", slug) else item["id"], used)
        changelog["evidence"][stable_id] = [item["id"]]
        # TestFlight also gets the verbatim testing note; the required entry keeps the disclosure
        # ahead of ordinary notes in the app, which drops the duplicate testing note.
        if "ios-testflight" in disclosure["channels"]:
            changelog["testflightNotes"].append({"id": stable_id, "text": disclosure["text"], "evidenceIds": [item["id"]]})
        title, _ = _split_text(disclosure["text"])
        changelog["requiredActions"].append({"id": stable_id, "title": title, "summary": disclosure["text"],
                                             "platforms": list(disclosure["channels"]), "kind": "improvement"})
    record["draft"] = {"model": response["model"], "promptVersion": 1, "inputSha256": response["inputSha256"],
                        "entries": result["entries"], "uncertainties": result["uncertainties"]}
    evidence_text = json.dumps(record, ensure_ascii=False, indent=2) + "\n"
    changelog_text = json.dumps(changelog, ensure_ascii=False, indent=2) + "\n"
    rendered = {filename: (render_changelog(changelog, channel)
                or "[DRAFT: add reviewed entries in changelog.json]") + "\n"
                for channel, filename in manifest["notes"].items()}
    _commit_structured(manifest, folder, changelog_text, evidence_text, rendered)


def _is_initial_changelog_scaffold(path, folder, manifest):
    try:
        scaffold = json.loads(path.read_text(encoding="utf-8"))
        evidence = json.loads((folder / "evidence.json").read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return False
    return (isinstance(evidence, dict) and "draft" not in evidence
            and scaffold == {"schemaVersion": 1, "version": manifest["version"],
                "buildNumber": manifest["buildNumber"], "highlights": [], "breaking": [],
                "requiredActions": [], "evidence": {}, "testflightNotes": []})


def render_structured(manifest, folder, replace=False):
    folder = Path(folder)
    changelog = json.loads((folder / CHANGELOG).read_text(encoding="utf-8"))
    rendered = {}
    for channel, filename in manifest["notes"].items():
        path = folder / filename
        require(not path.is_symlink(), f"Cannot render through a symlink: {filename}")
        if path.exists() and "[DRAFT:" not in path.read_text(encoding="utf-8") and not replace:
            raise ReleaseError(f"Preserving human edits in {filename}; explicit regeneration requires new review")
        text = render_changelog(changelog, channel)
        rendered[filename] = (text or "[DRAFT: add reviewed entries in changelog.json]") + "\n"
    _commit_structured(manifest, folder, (folder / CHANGELOG).read_text(encoding="utf-8"),
                       (folder / "evidence.json").read_text(encoding="utf-8"), rendered)


def _commit_structured(manifest, folder, changelog_text, evidence_text, rendered):
    with tempfile.TemporaryDirectory(dir=folder) as temporary:
        staging = Path(temporary)
        (staging / CHANGELOG).write_text(changelog_text, encoding="utf-8")
        (staging / "evidence.json").write_text(evidence_text, encoding="utf-8")
        for filename, text in rendered.items():
            (staging / filename).write_text(text, encoding="utf-8")
        validate_candidate(manifest, staging, draft=True)
        for filename in (CHANGELOG, "evidence.json", *rendered):
            os.replace(staging / filename, folder / filename)


def _render_legacy(manifest, folder, request, result, response, replace):
    evidence = {item["id"]: item for item in request["evidence"]}
    channels = manifest["channels"] + (["operators"] if manifest["product"] == "web" else [])
    entries = {channel: [] for channel in channels}
    for item in result["entries"]:
        allowed = ORYN_FIELDS | ORYN_OPTIONAL if manifest["product"] == "mobile" else ORYN_FIELDS
        require(isinstance(item, dict) and ORYN_FIELDS <= set(item) <= allowed, "Unknown note fields")
        _validate_oryn_metadata(item)
        channel = item["channel"]
        require(channel in entries and item["text"].strip() and item["evidenceIds"], "Invalid note entry")
        require(all(reference in evidence and channel in evidence[reference]["channels"]
                    for reference in item["evidenceIds"]), "Unknown or cross-platform note evidence")
        entries[channel].append(item["text"].strip())
    rendered = {}
    for channel in channels:
        filename = FILES.get(channel, "operators.zh-CN.md")
        path = folder / filename
        require(not path.is_symlink(), "Cannot render through a symlink")
        if path.exists() and "[DRAFT:" not in path.read_text(encoding="utf-8") and not replace:
            raise ReleaseError(f"Preserving human edits in {filename}; explicit regeneration requires new review")
        lines = entries[channel]
        for disclosure in manifest["requiredDisclosures"]:
            if channel in disclosure["channels"] and not any(disclosure["text"] in line for line in lines):
                lines.append(disclosure["text"])
        text = "\n\n".join(lines) if lines else "[DRAFT: human review required — no supported user-facing change identified]"
        rendered[filename] = text + "\n"
    with tempfile.TemporaryDirectory() as temporary:
        for filename, text in rendered.items():
            (Path(temporary) / filename).write_text(text, encoding="utf-8")
        validate_candidate(manifest, temporary, draft=True)
    for filename, text in rendered.items():
        (folder / filename).write_text(text, encoding="utf-8")
    evidence_path = folder / "evidence.json"
    record = json.loads(evidence_path.read_text(encoding="utf-8"))
    record["draft"] = {"model": response["model"], "promptVersion": 1,
                        "inputSha256": response["inputSha256"], "entries": result["entries"],
                        "uncertainties": result["uncertainties"]}
    evidence_path.write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    validate_candidate(manifest, folder, draft=True)
