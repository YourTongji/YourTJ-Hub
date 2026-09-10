#!/usr/bin/env python3
"""Read one of the forum's public AI-readable exports.

This helper intentionally supports only the three documented public routes. It
does not follow links found in forum content or access authenticated routes.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path
from typing import TextIO
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen


DEFAULT_TIMEOUT_SECONDS = 15
EXIT_USAGE = 2
EXIT_REQUEST_FAILED = 3
EXIT_INVALID_RESPONSE = 4

EXPORT_PATHS = {
    "index": ("/llms.txt", "text/plain"),
    "full": ("/llms-full.txt", "text/plain"),
}


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Read a documented public YourTJ Hub AI-readable export."
    )
    parser.add_argument(
        "--base-url",
        required=True,
        help="Verified forum root URL, for example http://localhost:5234",
    )
    parser.add_argument(
        "--source",
        choices=("index", "full", "topic"),
        required=True,
        help="Public projection to read",
    )
    parser.add_argument(
        "--topic-id",
        help="Positive topic ID; required when --source topic",
    )
    parser.add_argument(
        "--output",
        default="-",
        help="Output file, or - for stdout (default: -)",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=DEFAULT_TIMEOUT_SECONDS,
        help=f"HTTP timeout in seconds (default: {DEFAULT_TIMEOUT_SECONDS})",
    )
    arguments = parser.parse_args()

    if arguments.timeout <= 0:
        parser.error("--timeout must be greater than zero")
    if arguments.source == "topic":
        if not arguments.topic_id or not arguments.topic_id.isdigit() or int(arguments.topic_id) < 1:
            parser.error("--topic-id must be a positive integer for --source topic")
    elif arguments.topic_id:
        parser.error("--topic-id is only valid with --source topic")
    return arguments


def build_export_url(base_url: str, source: str, topic_id: str | None) -> tuple[str, str]:
    parsed_url = urlsplit(base_url)
    if parsed_url.scheme not in {"http", "https"} or not parsed_url.netloc:
        raise ValueError("--base-url must be an HTTP(S) URL with a host")
    if parsed_url.username or parsed_url.password or parsed_url.query or parsed_url.fragment:
        raise ValueError("--base-url must not contain credentials, query parameters, or a fragment")

    if source == "topic":
        assert topic_id is not None
        path = f"/p/posts/{topic_id}.md"
        expected_content_type = "text/markdown"
    else:
        path, expected_content_type = EXPORT_PATHS[source]

    root = base_url.rstrip("/")
    return root + path, expected_content_type


def fetch_export(url: str, timeout_seconds: float) -> tuple[int, str, str]:
    request = Request(
        url,
        headers={
            "Accept": "text/plain, text/markdown;q=0.9",
            "User-Agent": "forum-ai-readable-content-example/1",
        },
        method="GET",
    )
    try:
        with urlopen(request, timeout=timeout_seconds) as response:
            status_code = response.status
            content_type = response.headers.get_content_type()
            body = response.read().decode("utf-8", errors="replace")
            return status_code, content_type, body
    except HTTPError as error:
        return error.code, error.headers.get_content_type(), ""
    except (URLError, TimeoutError, OSError) as error:
        raise ConnectionError(str(error)) from error


def write_body(body: str, output_path: str) -> None:
    if output_path == "-":
        sys.stdout.write(body)
        if body and not body.endswith("\n"):
            sys.stdout.write("\n")
        return

    Path(output_path).write_text(body, encoding="utf-8")


def report_truncation(body: str, error_stream: TextIO) -> None:
    lowered_body = body.lower()
    if "truncated" in lowered_body:
        print(
            "warning: response contains a truncation marker; coverage is partial",
            file=error_stream,
        )


def main() -> int:
    arguments = parse_arguments()
    try:
        url, expected_content_type = build_export_url(
            arguments.base_url,
            arguments.source,
            arguments.topic_id,
        )
        status_code, content_type, body = fetch_export(url, arguments.timeout)
    except (ValueError, ConnectionError) as error:
        print(f"request failed: {error}", file=sys.stderr)
        return EXIT_REQUEST_FAILED

    if status_code < 200 or status_code >= 300:
        print(f"request failed: HTTP {status_code} ({url})", file=sys.stderr)
        return EXIT_REQUEST_FAILED
    if content_type != expected_content_type:
        print(
            f"invalid response: expected {expected_content_type}, received {content_type}",
            file=sys.stderr,
        )
        return EXIT_INVALID_RESPONSE
    if not body.strip():
        print("invalid response: body is empty", file=sys.stderr)
        return EXIT_INVALID_RESPONSE

    report_truncation(body, sys.stderr)
    try:
        write_body(body, arguments.output)
    except OSError as error:
        print(f"output failed: {error}", file=sys.stderr)
        return EXIT_INVALID_RESPONSE
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
