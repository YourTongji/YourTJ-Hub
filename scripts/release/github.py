"""Small gh adapter: explicit 404, pagination, no silent fallbacks on API errors."""
import json
import os
import subprocess
from model import REPOSITORY, ReleaseError


def run(*args, cwd=None, input=None):
    result = subprocess.run(args, cwd=cwd, input=input, text=True, capture_output=True)
    if result.returncode:
        raise ReleaseError(f"{args[0]} {args[1]} failed: {result.stderr.strip()[:1500]}")
    return result.stdout.strip()


def git(*args):
    return run("git", *args)


class GitHub:
    def __init__(self, repository=REPOSITORY):
        self.repository = repository

    def api(self, endpoint, *, method="GET", data=None, missing=False):
        endpoint = f"repos/{self.repository}/{endpoint}" if not endpoint.startswith(("repos/", "user", "orgs/")) else endpoint
        args = ["gh", "api", endpoint, "--method", method]
        if data is not None:
            args += ["--input", "-"]
        result = subprocess.run(args, input=json.dumps(data) if data is not None else None, capture_output=True, text=True)
        if result.returncode:
            if missing and "(HTTP 404)" in result.stderr:
                return None
            # Endpoints contain no credentials. Don't echo subprocess stdout or arbitrary request bodies.
            raise ReleaseError(f"GitHub {method} {endpoint} failed (exit {result.returncode}); check authentication/API availability")
        return json.loads(result.stdout) if result.stdout.strip() else None

    def pages(self, endpoint, key=None):
        rows = []
        for page in range(1, 1001):
            result = self.api(endpoint + ("&" if "?" in endpoint else "?") + f"per_page=100&page={page}")
            batch = result[key] if key else result
            if not isinstance(batch, list):
                raise ReleaseError("Expected a paginated GitHub collection")
            rows.extend(batch)
            if len(batch) < 100:
                return rows
        raise ReleaseError("GitHub pagination exceeded limit; refuse incomplete release evidence")

    def files(self, number, expected_count=None):
        paths = []
        rows = self.pages(f"pulls/{number}/files")
        if expected_count is not None and len(rows) != expected_count:
            raise ReleaseError("Incomplete PR file inventory; refuse authorization")
        for item in rows:
            paths.append(item["filename"])
            if item.get("previous_filename"):
                paths.append(item["previous_filename"])
        return paths

    def maintainers(self, reviews):
        eligible = set()
        for login in {r["user"]["login"] for r in reviews if r["user"]["type"] == "User"}:
            permission = self.api(f"collaborators/{login}/permission")["permission"]
            if permission in {"admin", "maintain", "write"}:
                eligible.add(login)
        return eligible

    def dispatch(self, workflow, inputs):
        self.api(f"actions/workflows/{workflow}/dispatches", method="POST", data={"ref": "main", "inputs": inputs})
        return {"workflow": workflow, "ref": "main", "inputs": inputs,
                "url": f"https://github.com/{self.repository}/actions/workflows/{workflow}"}
