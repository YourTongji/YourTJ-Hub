"""Remote identities and receipts. API errors stay unknown, never successful."""
import json
import re
from model import require, ReleaseError, VERSION, SHA
from github import git


def reservations():
    rows = []
    for tag in git("tag", "--list").splitlines():
        if not re.fullmatch(r"(?:mobile-)?v\d+\.\d+\.\d+", tag):
            continue
        item = {"tag": tag, "sourceSha": git("rev-parse", f"refs/tags/{tag}^{{commit}}"), "buildNumber": None}
        if tag.startswith("mobile-"):
            raw = git("for-each-ref", "--format=%(contents)", f"refs/tags/{tag}")
            if git("cat-file", "-t", f"refs/tags/{tag}") == "tag":
                metadata = json.loads(raw)
                require(type(metadata.get("buildNumber")) is int, "Invalid annotated build number")
                item["buildNumber"] = metadata["buildNumber"]
            else:
                source = git("show", f"refs/tags/{tag}:apps/mobile/packages/forum_app/pubspec.yaml")
                found = re.search(r"(?m)^version:\s*(\d+\.\d+\.\d+)\+([1-9]\d*)\s*$", source)
                require(found and tag == "mobile-v" + found[1], f"Cannot establish build identity for {tag}")
                item["buildNumber"] = int(found[2])
        rows.append(item)
    return rows


def latest_receipt(github, candidate_id, channel):
    # GitHub returns deployments and their statuses newest first; pagination preserves order.
    deployments = github.pages(f"deployments?environment=release-{channel}")
    for deployment in deployments:
        payload = deployment.get("payload") or {}
        require(isinstance(payload, dict), 'Malformed release receipt payload; reconcile deployments')
        if payload.get("candidateId") != candidate_id:
            continue
        statuses = github.pages(f"deployments/{deployment['id']}/statuses")
        return {"deployment": deployment, "status": statuses[0] if statuses else None}
    return None


def successful_baseline(github, channel):
    # First successful receipt in GitHub's reverse chronological deployment order.
    for deployment in github.pages(f"deployments?environment=release-{channel}"):
        statuses = github.pages(f"deployments/{deployment['id']}/statuses")
        if statuses and statuses[0]["state"] == "success":
            payload = deployment.get('payload')
            require(isinstance(payload, dict) and isinstance(payload.get('tag'), str)
                    and re.fullmatch(r'(?:mobile-)?v\d+\.\d+\.\d+', payload['tag'])
                    and isinstance(deployment.get('sha'), str) and SHA.fullmatch(deployment['sha']),
                    'Malformed successful release receipt; reconcile deployments before preparing')
            return {"tag": payload["tag"], "sourceSha": deployment["sha"], "deploymentId": deployment["id"]}
    return None


def baselines(github, channels, tags, apple=None):
    by_tag = {r["tag"]: r for r in tags}
    result = {}
    for channel in channels:
        if channel.startswith("ios-"):
            require(apple is not None and channel in apple, "Apple baseline requires authenticated ASC discovery in Prepare; status is unknown locally")
            current = apple[channel]
            if current is None:
                result[channel] = {"tag": None, "sourceSha": None}
                continue
            tag = "mobile-v" + current["version"]
            identity = by_tag.get(tag)
            require(identity and str(identity["buildNumber"]) == str(current["buildNumber"]),
                    "Apple build has no matching immutable mobile tag; reconcile before preparing release")
            result[channel] = {"tag": tag, "sourceSha": identity["sourceSha"], "buildId": current["buildId"], "state": current["state"]}
        elif channel == "android":
            published = [r for r in github.pages("releases") if not r["draft"] and not r["prerelease"]
                         and re.fullmatch(r"mobile-v\d+\.\d+\.\d+", r["tag_name"])
                         and len([a for a in r["assets"] if a["name"].endswith(".apk") and (a.get("digest") or "").startswith("sha256:")]) == 3]
            current = max(published, key=lambda r: tuple(map(int, r["tag_name"][8:].split("."))), default=None)
            require(current is None or current["tag_name"] in by_tag, "Missing Android tag identity")
            result[channel] = {"tag": current["tag_name"], "sourceSha": by_tag[current["tag_name"]]["sourceSha"]} if current else {"tag": None, "sourceSha": None}
        else:
            current = successful_baseline(github, "web")
            if current is None:
                # Bootstrap only a successful production deployment, never merely the newest tag.
                runs = github.pages("actions/workflows/deploy-main.yml/runs?status=success", "workflow_runs")
                runs = [r for r in runs if r["event"] == "workflow_dispatch" and re.fullmatch(r"v\d+\.\d+\.\d+", r["head_branch"] or "")]
                run = max(runs, key=lambda r: r["id"], default=None)
                if run:
                    require(run["head_branch"] in by_tag and by_tag[run["head_branch"]]["sourceSha"] == run["head_sha"], "Production run/tag mismatch")
                    current = {"tag": run["head_branch"], "sourceSha": run["head_sha"]}
            require(current is not None, "Unknown deployed server baseline; reconcile production before release")
            result[channel] = current
    return result


def record(github, manifest, binding, channel, state, details=None):
    require(channel in manifest["channels"], "Receipt channel is not approved")
    payload = {"schemaVersion": 1, "candidateId": manifest["candidateId"], "tag": manifest["tag"],
               "binding": binding, "channel": channel, "details": details or {}}
    deployment = github.api("deployments", method="POST", data={"ref": manifest["sourceSha"],
        "environment": f"release-{channel}", "auto_merge": False, "required_contexts": [], "payload": payload,
        "description": f"{manifest['candidateId']} / {channel}", "production_environment": channel != "ios-testflight"})
    github.api(f"deployments/{deployment['id']}/statuses", method="POST", data={"state": state,
        "auto_inactive": False, "description": str((details or {}).get("availability", state))[:140]})
    return deployment["id"]
