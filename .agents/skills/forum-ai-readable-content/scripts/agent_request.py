#!/usr/bin/env python3
"""Call the fixed, documented YourTJ Hub Agent Bot API operations.

The helper is read-only by default. It accepts an Agent token only from the
YOURTJ_AGENT_TOKEN environment variable and never prints that value.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener


DEFAULT_TIMEOUT_SECONDS = 15
TOKEN_ENVIRONMENT_VARIABLE = "YOURTJ_AGENT_TOKEN"
EXIT_USAGE = 2
EXIT_REQUEST_FAILED = 3
EXIT_HTTP_FAILURE = 4
EXIT_BUSINESS_FAILURE = 5
EXIT_PROTOCOL_FAILURE = 6

READ_OPERATIONS = {"me", "topics", "topic-posts", "search"}
WRITE_OPERATIONS = {"create-topic", "create-post"}
ALL_OPERATIONS = READ_OPERATIONS | WRITE_OPERATIONS


class RedirectRejected(RuntimeError):
    """Raised when urllib tries to redirect an authenticated request."""


class NoRedirectHandler(HTTPRedirectHandler):
    def redirect_request(self, request: Request, *args: Any, **kwargs: Any) -> None:
        raise RedirectRejected("redirects are disabled for authenticated requests")


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Call one fixed YourTJ Hub Agent Bot API operation."
    )
    parser.add_argument(
        "--base-url",
        required=True,
        help="Verified forum root URL, for example http://localhost:5234",
    )
    parser.add_argument(
        "--operation",
        required=True,
        choices=sorted(ALL_OPERATIONS),
        help="Documented Agent operation to call",
    )
    parser.add_argument(
        "--topic-id",
        help="Positive topic ID for topic-posts and create-post",
    )
    parser.add_argument("--page", type=int, help="Topic/search page number")
    parser.add_argument("--page-size", type=int, help="Topic page size")
    parser.add_argument("--sort", choices=("latest", "hot", "popular", "new"))
    parser.add_argument("--category-id", type=int, help="Positive topic category ID")
    parser.add_argument("--query", dest="search_query", help="Search query")
    parser.add_argument(
        "--scope",
        choices=("all", "topics", "users", "categories"),
        help="Search scope",
    )
    parser.add_argument("--anchor-post-id", type=int)
    parser.add_argument("--anchor-post-no", type=int)
    parser.add_argument("--before-post-no", type=int)
    parser.add_argument("--after-post-no", type=int)
    parser.add_argument("--limit", type=int, help="Post window size from 1 to 50")
    parser.add_argument(
        "--data-file",
        help="JSON object file for a write operation; never pass the body as a command-line value",
    )
    parser.add_argument(
        "--allow-write",
        action="store_true",
        help="Explicitly allow the selected POST operation and its real forum side effect",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=DEFAULT_TIMEOUT_SECONDS,
        help=f"HTTP timeout in seconds (default: {DEFAULT_TIMEOUT_SECONDS})",
    )
    arguments = parser.parse_args()
    validate_arguments(parser, arguments)
    return arguments


def validate_arguments(parser: argparse.ArgumentParser, arguments: argparse.Namespace) -> None:
    if arguments.timeout <= 0:
        parser.error("--timeout must be greater than zero")

    needs_topic_id = arguments.operation in {"topic-posts", "create-post"}
    if needs_topic_id:
        if not arguments.topic_id or not arguments.topic_id.isdigit() or int(arguments.topic_id) < 1:
            parser.error("--topic-id must be a positive integer for this operation")
    elif arguments.topic_id:
        parser.error("--topic-id is only valid for topic-posts and create-post")

    if arguments.operation in WRITE_OPERATIONS:
        if not arguments.allow_write:
            parser.error("write operations require the explicit --allow-write flag")
        if not arguments.data_file:
            parser.error("write operations require --data-file with a JSON object")
    elif arguments.data_file or arguments.allow_write:
        parser.error("--data-file and --allow-write are only valid for write operations")

    if arguments.page is not None and arguments.page < 1:
        parser.error("--page must be at least 1")
    if arguments.page_size is not None and arguments.page_size < 10:
        parser.error("--page-size must be at least 10")
    if arguments.category_id is not None and arguments.category_id < 1:
        parser.error("--category-id must be positive")
    for argument_name in (
        "anchor_post_id",
        "anchor_post_no",
        "before_post_no",
        "after_post_no",
    ):
        argument_value = getattr(arguments, argument_name)
        if argument_value is not None and argument_value < 1:
            parser.error(f"--{argument_name.replace('_', '-')} must be positive")
    if arguments.limit is not None and not 1 <= arguments.limit <= 50:
        parser.error("--limit must be between 1 and 50")


def read_agent_token() -> str:
    token = os.environ.get(TOKEN_ENVIRONMENT_VARIABLE, "").strip()
    if not token.startswith("agt_") or len(token) <= len("agt_"):
        raise ValueError(
            f"{TOKEN_ENVIRONMENT_VARIABLE} is missing or does not contain an Agent token"
        )
    return token


def read_json_object(path_value: str) -> dict[str, Any]:
    try:
        payload = json.loads(Path(path_value).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(f"cannot read JSON object from --data-file: {error}") from error
    if not isinstance(payload, dict):
        raise ValueError("--data-file must contain a JSON object")
    return payload


def build_base_url(base_url: str) -> str:
    parsed_url = urlsplit(base_url)
    if parsed_url.scheme not in {"http", "https"} or not parsed_url.netloc:
        raise ValueError("--base-url must be an HTTP(S) URL with a host")
    if parsed_url.username or parsed_url.password or parsed_url.query or parsed_url.fragment:
        raise ValueError("--base-url must not contain credentials, query parameters, or a fragment")
    return base_url.rstrip("/")


def build_request_target(arguments: argparse.Namespace, base_url: str) -> tuple[str, str, bytes | None]:
    query_parameters: dict[str, str | int] = {}
    path = ""
    method = "GET"
    request_body: bytes | None = None

    if arguments.operation == "me":
        path = "/api/v1/agent/me"
    elif arguments.operation == "topics":
        path = "/api/v1/agent/topics"
        add_optional_parameter(query_parameters, "page", arguments.page)
        add_optional_parameter(query_parameters, "pageSize", arguments.page_size)
        add_optional_parameter(query_parameters, "sort", arguments.sort)
        add_optional_parameter(query_parameters, "categoryId", arguments.category_id)
    elif arguments.operation == "topic-posts":
        path = f"/api/v1/agent/topics/{arguments.topic_id}/posts"
        add_optional_parameter(query_parameters, "anchorPostId", arguments.anchor_post_id)
        add_optional_parameter(query_parameters, "anchorPostNo", arguments.anchor_post_no)
        add_optional_parameter(query_parameters, "beforePostNo", arguments.before_post_no)
        add_optional_parameter(query_parameters, "afterPostNo", arguments.after_post_no)
        add_optional_parameter(query_parameters, "limit", arguments.limit)
    elif arguments.operation == "search":
        path = "/api/v1/agent/search"
        add_optional_parameter(query_parameters, "q", arguments.search_query)
        add_optional_parameter(query_parameters, "scope", arguments.scope)
        add_optional_parameter(query_parameters, "page", arguments.page)
    elif arguments.operation == "create-topic":
        path = "/api/v1/agent/topics"
        method = "POST"
        request_body = json.dumps(
            read_json_object(arguments.data_file),
            ensure_ascii=False,
        ).encode("utf-8")
    elif arguments.operation == "create-post":
        path = f"/api/v1/agent/topics/{arguments.topic_id}/posts"
        method = "POST"
        request_body = json.dumps(
            read_json_object(arguments.data_file),
            ensure_ascii=False,
        ).encode("utf-8")
    else:
        raise ValueError(f"unsupported operation: {arguments.operation}")

    query_suffix = f"?{urlencode(query_parameters)}" if query_parameters else ""
    return base_url + path + query_suffix, method, request_body


def add_optional_parameter(
    parameters: dict[str, str | int], name: str, value: str | int | None
) -> None:
    if value is not None:
        parameters[name] = value


def perform_request(
    url: str,
    method: str,
    request_body: bytes | None,
    token: str,
    timeout_seconds: float,
) -> tuple[int, str, str, str | None]:
    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {token}",
        "User-Agent": "forum-ai-readable-content-agent-example/1",
    }
    if request_body is not None:
        headers["Content-Type"] = "application/json"
    request = Request(url, data=request_body, headers=headers, method=method)
    opener = build_opener(NoRedirectHandler)
    try:
        with opener.open(request, timeout=timeout_seconds) as response:
            response_body = response.read().decode("utf-8", errors="replace")
            return (
                response.status,
                response.headers.get_content_type(),
                response_body,
                response.headers.get("Retry-After"),
            )
    except HTTPError as error:
        response_body = error.read().decode("utf-8", errors="replace")
        return (
            error.code,
            error.headers.get_content_type(),
            response_body,
            error.headers.get("Retry-After"),
        )
    except (RedirectRejected, URLError, TimeoutError, OSError) as error:
        raise ConnectionError(str(error)) from error


def replace_token(value: str, token: str) -> str:
    return value.replace(token, "[REDACTED_AGENT_TOKEN]")


def print_response_body(body: str, token: str) -> bool:
    if not body:
        return False
    safe_body = replace_token(body, token)
    try:
        parsed_body = json.loads(safe_body)
    except json.JSONDecodeError:
        sys.stdout.write(safe_body)
        if not safe_body.endswith("\n"):
            sys.stdout.write("\n")
        return False
    sys.stdout.write(json.dumps(parsed_body, ensure_ascii=False, indent=2) + "\n")
    return isinstance(parsed_body, dict)


def main() -> int:
    arguments = parse_arguments()
    try:
        token = read_agent_token()
        base_url = build_base_url(arguments.base_url)
        url, method, request_body = build_request_target(arguments, base_url)
        status_code, content_type, body, retry_after = perform_request(
            url,
            method,
            request_body,
            token,
            arguments.timeout,
        )
    except (ValueError, ConnectionError) as error:
        print(f"request failed: {error}", file=sys.stderr)
        return EXIT_REQUEST_FAILED

    if status_code == 429:
        retry_text = retry_after or "not provided"
        print(f"rate limited: HTTP 429; Retry-After: {retry_text}", file=sys.stderr)
    elif status_code == 401:
        print("authentication failed: HTTP 401; inspect auth.required envelope", file=sys.stderr)
    elif status_code < 200 or status_code >= 300:
        print(f"request failed: HTTP {status_code}", file=sys.stderr)

    if content_type != "application/json":
        print(
            f"invalid response: expected application/json, received {content_type}",
            file=sys.stderr,
        )
        print_response_body(body, token)
        return EXIT_PROTOCOL_FAILURE

    try:
        response_data = json.loads(body)
    except json.JSONDecodeError:
        print("invalid response: body is not valid JSON", file=sys.stderr)
        print_response_body(body, token)
        return EXIT_PROTOCOL_FAILURE

    print_response_body(body, token)
    if not isinstance(response_data, dict) or not isinstance(response_data.get("code"), int):
        print("invalid response: missing integer envelope code", file=sys.stderr)
        return EXIT_PROTOCOL_FAILURE
    if status_code < 200 or status_code >= 300:
        return EXIT_HTTP_FAILURE
    if response_data["code"] != 0:
        print(
            "business failure: response envelope code is non-zero; inspect messageCode",
            file=sys.stderr,
        )
        return EXIT_BUSINESS_FAILURE
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
