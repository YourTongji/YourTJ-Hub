#!/usr/bin/env python3
"""Distribute a preview only to an explicitly selected internal QA group.

Internal distribution avoids changing the production beta review account/queue.
Auto-access groups are checked before upload, not merely after distribution.
"""
import json
import os
import subprocess
import sys
from pathlib import Path
from build_environment import dart_defines
from publish_ios import asc, resource, find_build, wait_for_build, APP_ID, GROUP_ID


def check_groups(groups, selected):
    matches = [g for g in groups if g['id'] == selected]
    if (not selected or selected == GROUP_ID or len(matches) != 1
            or matches[0]['attributes'].get('isInternalGroup') is not True):
        raise ValueError('Preview needs a dedicated internal TestFlight group, distinct from production')
    for group in groups:
        if group['id'] == selected:
            continue
        attributes = group['attributes']
        automatic = attributes.get('hasAccessToAllBuilds')
        # External groups require explicit build assignment/review, and omit this internal flag.
        # https://developer.apple.com/help/app-store-connect/test-a-beta-version/invite-external-testers/
        if automatic is True or (attributes.get('isInternalGroup') is not False and automatic is not False):
            raise ValueError('Another TestFlight group may automatically receive previews; reconcile before uploading')


def preview_group():
    group = os.environ.get('PRERELEASE_TESTFLIGHT_GROUP_ID', '')
    check_groups(asc('testflight', 'groups', 'list', '--app', APP_ID, '--paginate')['data'], group)
    return group


def assign_internal_group(build, group):
    def membership():
        result = asc('builds', 'groups', 'list', '--build-id', build)
        if (result.get('appId') != APP_ID or result.get('buildId') != build
                or result.get('complete') is not True or result.get('failures')):
            raise ValueError('Cannot verify the complete preview build audience')
        groups = result.get('groups', [])
        if any(g.get('id') != group or g.get('type') != 'internal' for g in groups):
            raise ValueError('Preview build has an unexpected testing audience')
        return any(g.get('id') == group for g in groups)

    if membership():
        return
    groups = asc('testflight', 'groups', 'list', '--app', APP_ID, '--paginate')['data']
    check_groups(groups, group)
    selected = next(g for g in groups if g['id'] == group)
    # Apple rejects explicit assignment to internal groups with automatic all-build access.
    if selected['attributes'].get('hasAccessToAllBuilds') is not True:
        asc('builds', 'add-groups', '--build-id', build, '--group', group)
    if not membership():
        raise ValueError('Internal group has not received the preview; reconcile and retry')


def main():
    dart_defines()
    if os.environ.get('MOBILE_PREVIEW') != 'true':
        raise ValueError('This publisher accepts previews only')
    group = preview_group()
    version, number = os.environ['MOBILE_VERSION'], os.environ['MOBILE_BUILD_NUMBER']
    result_path = Path(os.environ.get('RELEASE_RESULT_PATH', '.preview-ios.json'))
    def record(stage, **data):
        result_path.write_text(json.dumps({'stage': stage, 'version': version, 'buildNumber': number,
            'source': os.environ['SOURCE_SHA'], 'environment': os.environ['MOBILE_ENVIRONMENT'], **data}) + '\n')
    record('checking_upload')
    existing = find_build(version, number)
    expected = os.environ.get('IOS_EXPECTED_BUILD_ID')
    if os.environ.get('IOS_EXISTING_BUILD_ONLY') == 'true' and (
            not existing or not expected or existing['id'] != expected):
        raise ValueError('Preview recovery requires the exact existing Apple build; never re-upload')
    if existing is None:
        uploads = asc('builds', 'uploads', 'list', '--app', APP_ID, '--cf-bundle-short-version', version,
                      '--cf-bundle-version', number, '--paginate')['data']
        if uploads:
            if not any(u['attributes'].get('uploadedDate') for u in uploads):
                raise ValueError('Upload outcome uncertain; reconcile ASC before retrying')
        else:
            asc('builds', 'upload', '--app', APP_ID, '--ipa', os.environ['IOS_IPA_PATH'],
                '--version', version, '--build-number', number)
    record('processing')
    build = wait_for_build(version, number)
    if expected and build != expected:
        raise ValueError('Apple build differs from the original preview receipt')
    record('processed', buildId=build)
    subprocess.run([sys.executable, str(Path(__file__).resolve().parents[1] / 'release/cycle_control.py'),
                    'check-build'], check=True)
    # Recheck scope after processing; a group may have changed while waiting.
    preview_group()
    asc('builds', 'update', '--build-id', build, '--uses-non-exempt-encryption=false')
    notes = (f"{os.environ['MOBILE_ENVIRONMENT'].upper()} testing only. Source {os.environ['SOURCE_SHA']}. "
             "Complete the release issue's device checklist. Current candidate and return-to-production instructions: "
             "https://github.com/YourTongji/YourTJ-Hub/releases/tag/preview-status")
    existing = [n for n in asc('builds', 'test-notes', 'list', '--build-id', build)['data']
                if n.get('attributes', {}).get('locale') == 'en-US']
    asc('builds', 'test-notes', 'update' if existing else 'create', '--build-id', build,
        '--locale', 'en-US', '--whats-new', notes)
    assign_internal_group(build, group)
    details = resource(asc('builds', 'build-beta-detail', 'view', '--build-id', build))
    stage = details.get('attributes', {}).get('internalBuildState')
    record('assigned', buildId=build, appleState=stage, group=group)
    if stage != 'IN_BETA_TESTING':
        raise ValueError(f'TestFlight assigned but not yet installable ({stage}); reconcile and retry')
    record('available_for_testing', buildId=build, appleState=stage, group=group)
    with open(os.environ['GITHUB_STEP_SUMMARY'], 'a') as out:
        out.write(f"TestFlight internal QA: {version} ({number}), build `{build}`, {stage}.\n\n"
                  "Availability is not a device QA pass. No App Store or external beta submission was made.\n")


if __name__ == '__main__':
    if sys.argv[1:] == ['--check-groups']:
        preview_group()
        print('Internal TestFlight audience checks passed.')
    elif sys.argv[1:]:
        raise SystemExit('Usage: publish_preview_ios.py [--check-groups]')
    else:
        main()
