"""Host-owned platform rendering. Oryn output cannot select files or targets."""
import hashlib
import json
from pathlib import Path
import tempfile
from model import FILES, ReleaseError, digest, require, validate_candidate


def render(manifest, folder, response, model_input, replace=False):
    require(response.get("schemaVersion") == 1 and response.get("promptVersion") == 1, "Unsupported Oryn result")
    require(response.get("inputSha256") == hashlib.sha256(model_input).hexdigest(), "Oryn input digest mismatch")
    request = json.loads(model_input)
    result = response["output"]
    require(set(result) == {"schemaVersion", "entries", "uncertainties"} and result["schemaVersion"] == 1, "Invalid notes output")
    evidence = {e["id"]: e for e in request["evidence"]}
    channels = manifest["channels"] + (["operators"] if manifest["product"] == "web" else [])
    entries = {c: [] for c in channels}
    for item in result["entries"]:
        require(set(item) == {"channel", "text", "evidenceIds"}, "Unknown note fields")
        channel = item["channel"]
        require(channel in entries and item["text"].strip() and item["evidenceIds"], "Invalid note entry")
        require(all(i in evidence and channel in evidence[i]["channels"] for i in item["evidenceIds"]),
                "Unknown or cross-platform note evidence")
        entries[channel].append(item["text"].strip())
    rendered = {}
    for channel in channels:
        filename = FILES.get(channel, "operators.zh-CN.md")
        path = folder / filename
        require(not path.is_symlink(), 'Cannot render through a symlink')
        if path.exists() and "[DRAFT:" not in path.read_text(encoding='utf-8') and not replace:
            raise ReleaseError(f"Preserving human edits in {filename}; explicit regeneration requires new review")
        lines = entries[channel]
        for disclosure in manifest["requiredDisclosures"]:
            if channel in disclosure["channels"] and not any(disclosure["text"] in line for line in lines):
                lines.append(disclosure["text"])
        text = "\n\n".join(lines) if lines else "[DRAFT: human review required — no supported user-facing change identified]"
        rendered[filename] = text + "\n"
    # Validate the complete proposed set before changing any existing human/draft file.
    with tempfile.TemporaryDirectory() as temporary:
        for filename, text in rendered.items():
            (Path(temporary) / filename).write_text(text, encoding='utf-8')
        validate_candidate(manifest, temporary, draft=True)
    for filename, text in rendered.items():
        (folder / filename).write_text(text, encoding='utf-8')
    evidence_path = folder / "evidence.json"
    record = json.loads(evidence_path.read_text(encoding='utf-8'))
    record["draft"] = {"model": response["model"], "promptVersion": 1,
                       "inputSha256": response["inputSha256"], "entries": result["entries"],
                       "uncertainties": result["uncertainties"]}
    evidence_path.write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding='utf-8')
    validate_candidate(manifest, folder, draft=True)
