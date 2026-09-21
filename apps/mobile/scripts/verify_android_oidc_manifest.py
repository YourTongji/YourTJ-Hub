#!/usr/bin/env python3
"""Verify the Android debug merged manifest's MainActivity OIDC contract."""

from __future__ import annotations

import sys
import xml.etree.ElementTree as ET
from pathlib import Path


ANDROID_NS = "http://schemas.android.com/apk/res/android"
ANDROID = "{" + ANDROID_NS + "}"
ACTION = ANDROID + "name"
CATEGORY = ANDROID + "name"
SCHEME = ANDROID + "scheme"
HOST = ANDROID + "host"
EXPORTED = ANDROID + "exported"
THEME = ANDROID + "theme"
AFFINITY = ANDROID + "taskAffinity"

VIEW = "android.intent.action.VIEW"
DEFAULT = "android.intent.category.DEFAULT"
BROWSABLE = "android.intent.category.BROWSABLE"
CALLBACK_ACTIVITY = "net.openid.appauth.RedirectUriReceiverActivity"
MAIN_ACTIVITY = "tj.yourtj.forum_app.MainActivity"
CALLBACK_SCHEME = "yourtj"
CALLBACK_HOST = "callback"
DEEP_LINKING_META = "flutter_deeplinking_enabled"


def default_manifest() -> Path:
    return (
        Path(__file__).resolve().parents[3]
        / "apps/mobile/packages/forum_app/build/app/intermediates/merged_manifest"
        / "debug/processDebugMainManifest/AndroidManifest.xml"
    )


def activity_name(activity: ET.Element) -> str:
    return activity.get(ACTION, "")


def main(argv: list[str]) -> int:
    manifest_path = Path(argv[1]).resolve() if len(argv) == 2 else default_manifest()
    if len(argv) > 2:
        print("usage: verify_android_oidc_manifest.py [MERGED_MANIFEST]", file=sys.stderr)
        return 2
    if not manifest_path.is_file():
        print(f"manifest not found: {manifest_path}", file=sys.stderr)
        print(
            "run the Android debug manifest merge first or pass its path explicitly",
            file=sys.stderr,
        )
        return 2

    try:
        root = ET.parse(manifest_path).getroot()
    except ET.ParseError as error:
        print(f"invalid XML in {manifest_path}: {error}", file=sys.stderr)
        return 2

    application = root.find("application")
    if application is None:
        print("manifest verification failed: missing application", file=sys.stderr)
        return 1

    activities = list(application.findall("activity"))
    callback_filters: list[tuple[ET.Element, ET.Element]] = []
    broad_scheme_filters: list[tuple[str, list[tuple[str | None, str | None]]]] = []
    for activity in activities:
        for intent_filter in activity.findall("intent-filter"):
            data_specs = [
                (data.get(SCHEME), data.get(HOST))
                for data in intent_filter.findall("data")
            ]
            if any(scheme == CALLBACK_SCHEME for scheme, _ in data_specs):
                broad_scheme_filters.append((activity_name(activity), data_specs))
            if data_specs == [(CALLBACK_SCHEME, CALLBACK_HOST)]:
                callback_filters.append((activity, intent_filter))

    errors: list[str] = []
    if len(callback_filters) != 1:
        owners = ", ".join(activity_name(activity) for activity, _ in callback_filters)
        errors.append(
            "expected exactly one intent-filter for scheme=yourtj host=callback"
            + (f"; found {len(callback_filters)} ({owners})" if owners else "")
        )
    else:
        callback_activity, callback_filter = callback_filters[0]
        owner = activity_name(callback_activity)
        if owner != MAIN_ACTIVITY:
            errors.append(
                "yourtj callback owner must be "
                f"{MAIN_ACTIVITY}; found {owner or '<unnamed>'}"
            )
        if callback_activity.get(EXPORTED) != "true":
            errors.append("MainActivity must be exported=true")

        actions = {node.get(ACTION) for node in callback_filter.findall("action")}
        categories = {node.get(CATEGORY) for node in callback_filter.findall("category")}
        if actions != {VIEW}:
            errors.append(f"yourtj callback actions must be exactly VIEW; found {sorted(actions)}")
        if categories != {DEFAULT, BROWSABLE}:
            errors.append(
                "yourtj callback categories must be exactly DEFAULT+BROWSABLE; "
                f"found {sorted(categories)}"
            )

        data_specs = [
            (data.get(SCHEME), data.get(HOST))
            for data in callback_filter.findall("data")
        ]
        if data_specs != [(CALLBACK_SCHEME, CALLBACK_HOST)]:
            errors.append(
                "yourtj callback data must be exactly scheme=yourtj host=callback; "
                f"found {data_specs}"
            )

    for owner, data_specs in broad_scheme_filters:
        if data_specs != [(CALLBACK_SCHEME, CALLBACK_HOST)]:
            errors.append(
                "no broad or duplicate yourtj scheme handler is allowed; "
                f"found {owner}: {data_specs}"
            )

    receiver_claims = [
        activity_name(activity)
        for activity, _ in callback_filters
        if activity_name(activity) == CALLBACK_ACTIVITY
    ]
    if receiver_claims:
        errors.append("RedirectUriReceiverActivity must not claim yourtj://callback")

    main_activities = [activity for activity in activities if activity_name(activity) == MAIN_ACTIVITY]
    for activity in main_activities:
        if AFFINITY in activity.attrib:
            errors.append("MainActivity must omit android:taskAffinity")
        metadata = {
            node.get(ACTION): node.get(ANDROID + "value")
            for node in activity.findall("meta-data")
        }
        if metadata.get(DEEP_LINKING_META) != "false":
            errors.append("MainActivity must set flutter_deeplinking_enabled=false")

    if len(main_activities) != 1:
        errors.append(f"expected exactly one MainActivity; found {len(main_activities)}")

    if errors:
        print(f"Android OIDC manifest verification failed: {manifest_path}", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Android OIDC manifest verification passed: {manifest_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
