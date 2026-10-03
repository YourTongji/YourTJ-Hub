"""Render platform notes only from validated evidence and reviewed structured facts."""
import hashlib
import json
import os
import re
from pathlib import Path
import tempfile

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
        require(isinstance(item, dict) and set(item) == {"channel", "text", "evidenceIds"}, "Invalid Oryn changelog draft")
        channel = item["channel"]
        require(channel in entries and isinstance(item["text"], str) and item["text"].strip()
                and len(item["text"].strip()) <= (500 if channel == "ios-testflight" else 400)
                and isinstance(item["evidenceIds"], list) and item["evidenceIds"], "Invalid Oryn changelog draft")
        require(all(reference in evidence and channel in evidence[reference]["channels"]
                    for reference in item["evidenceIds"]), "Unknown or cross-platform changelog evidence")
        entries[channel].append(item)

    changelog = {"schemaVersion": 1, "version": manifest["version"], "buildNumber": manifest["buildNumber"],
                 "highlights": [], "breaking": [], "requiredActions": [], "evidence": {}, "testflightNotes": []}
    for channel, items in entries.items():
        for item in items:
            text = item["text"].strip()
            stable_id = "oryn-" + hashlib.sha256((channel + "\0" + text).encode()).hexdigest()[:16]
            references = list(dict.fromkeys(item["evidenceIds"]))
            changelog["evidence"][stable_id] = references
            if channel == "ios-testflight":
                changelog["testflightNotes"].append({"id": stable_id, "text": text, "evidenceIds": references})
            else:
                title = re.split(r"[。！？!?；;\n]", text, maxsplit=1)[0].strip()[:100] or "本次改进"
                changelog["highlights"].append({"id": stable_id, "title": title, "summary": text,
                                                 "platforms": [channel], "kind": "improvement"})
    evidence_path = folder / "evidence.json"
    record = json.loads(evidence_path.read_text(encoding="utf-8"))
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
        require(set(item) == {"channel", "text", "evidenceIds"}, "Unknown note fields")
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
