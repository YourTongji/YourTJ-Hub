#!/usr/bin/env python3
"""One conservative domain map for PR/push CI and dev deployment eligibility."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

DOMAINS = ("docs", "govulncheck", "backend", "postgres", "frontend", "contract", "mobile", "android", "ios", "status", "automation", "release", "deploy")


def select(paths):
    selected = {"docs": "Repository links/governance always run", "govulncheck": "Current vulnerability database is checked on every change"}
    def add(*domains, reason):
        for domain in domains:
            selected.setdefault(domain, reason)
    for path in paths:
        if path.startswith("releases/requests/"):
            add("release", reason=path)
            continue
        if path.endswith(".md") or path.startswith(("docs/", ".agents/", "research/")) or path in {"LICENSE", ".gitignore"}:
            continue
        if path.startswith("apps/status/"):
            add("status", reason=path)
        elif path.startswith("apps/gooseforum/"):
            add("deploy", reason=path)
            if path.startswith("apps/gooseforum/resource/"):
                add("frontend", reason=path)
                if path.endswith(".go") or "/templates/" in path:
                    add("backend", reason=path)
                if "/src/gen/" in path or path.endswith("packages/client/package.json"):
                    add("contract", reason=path)
                if path.endswith(("tokens.css", "logo.svg")) or "/src/locales/" in path:
                    add("mobile", reason=path)
            else:
                add("backend", reason=path)
                if any(part in path for part in ("/service/", "/models/", "/migration/", "/connect/sqlconnect/")) or path.endswith(("go.mod", "go.sum")):
                    add("postgres", reason=path)
                if "/http/" in path:
                    add("contract", reason=path)
                if path.endswith("page_component.go") or "/testdata/" in path:
                    add("frontend", reason=path)
        elif path.startswith("packages/api-contract/"):
            add("backend", "frontend", "contract", "mobile", "deploy", reason=path)
        elif path.startswith("apps/mobile/"):
            if path.startswith("apps/mobile/store/"):
                add("automation", reason=path)
                continue
            add("mobile", reason=path)
            if "server_message" in path:
                add("frontend", reason=path)
            shared_native = path.endswith(("pubspec.yaml", "pubspec.lock", "pubspec_overrides.yaml", "build.yaml", "release-config.json")) or any(x in path for x in ("/hooks/", "/hook/", "/scripts/"))
            if shared_native or "/android/" in path:
                add("android", "automation", reason=path)
            if shared_native or "/ios/" in path:
                add("ios", "automation", reason=path)
        elif path.startswith((".github/", "scripts/", "deploy/")):
            if path.startswith("scripts/") and not path.startswith(("scripts/release/", "scripts/mobile-release/", "scripts/ci/", "scripts/test-", "scripts/verify-", "scripts/run-gates")):
                add(*DOMAINS, reason=f"Unknown executable input: {path}")
            add("automation", reason=path)
            if path.startswith("deploy/") or "/build-binary/" in path or "/push-image/" in path:
                add("backend", "frontend", "deploy", reason=path)
            if path in {".github/workflows/ci.yml", "scripts/ci/select_inputs.py"}:
                add(*DOMAINS, reason=path)
            elif path.startswith(".github/workflows/ci-"):
                name = path.split("ci-", 1)[1].split(".")[0]
                mapping = {"backend": ["backend", "postgres"], "mobile-native": ["android", "ios"], "mobile": ["mobile"], "frontend": ["frontend"], "contract": ["contract"], "status": ["status"]}
                add(*mapping.get(name, []), reason=path)
            if path in {".github/workflows/release-android.yml", "scripts/mobile-release/prepare_push.py"}:
                add("android", reason=path)
            if path in {".github/workflows/release-ios.yml", "scripts/mobile-release/build_ios.py"}:
                add("ios", reason=path)
        else:
            add(*DOMAINS, reason=f"Unknown executable input: {path}")
    return {d: {"run": d in selected, "reason": selected.get(d, "No affected inputs")} for d in DOMAINS}


def verify_results(plan, needs):
    for domain, selection in plan.items():
        if domain in {"deploy", "postgres"}:
            # PG is included in backend's reusable-workflow result; deploy is an output policy.
            continue
        result = needs.get(domain, {}).get("result")
        expected = "success" if selection["run"] else "skipped"
        if result != expected or not selection["reason"]:
            raise ValueError(f"{domain}: expected {expected}, got {result}; {selection['reason']}")


def changed_paths(base, head):
    # Both old and new paths, including deletions, without GitHub's 300/3000-file API limits.
    result = subprocess.run(["git", "diff", "--name-only", "--no-renames", "-z", base, head, "--"], capture_output=True, text=True)
    if result.returncode:
        raise ValueError("Cannot compare exact CI inputs")
    return [p for p in result.stdout.split("\0") if p]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--paths-json", action="store_true", help="classify a JSON path array from stdin")
    parser.add_argument("--base")
    parser.add_argument("--head", default="HEAD")
    parser.add_argument("--verify", action="store_true")
    parser.add_argument("--output", default="ci-plan.json")
    args = parser.parse_args()
    if args.paths_json:
        print(json.dumps(select(json.load(sys.stdin))))
        return
    if args.verify:
        verify_results(json.loads(os.environ["CI_PLAN"]), json.loads(os.environ["CI_NEEDS"]))
        return
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text()) if os.environ.get("GITHUB_EVENT_PATH") else {}
    base = args.base or event.get("pull_request", {}).get("base", {}).get("sha") or event.get("before")
    try:
        if not base or set(base) == {"0"}:
            raise ValueError("No comparison base")
        if event.get("pull_request"):
            base = subprocess.check_output(["git", "merge-base", base, args.head], text=True).strip()
        plan = select(changed_paths(base, args.head))
    except (ValueError, subprocess.CalledProcessError):
        plan = {d: {"run": True, "reason": "Comparison unavailable; conservative full validation"} for d in DOMAINS}
    Path(args.output).write_text(json.dumps(plan, indent=2) + "\n")
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as out:
            out.write("plan=" + json.dumps(plan, separators=(",", ":")) + "\n")
            for key, value in plan.items():
                out.write(f"{key}={str(value['run']).lower()}\n")
    print(json.dumps(plan, indent=2))


if __name__ == "__main__":
    main()
