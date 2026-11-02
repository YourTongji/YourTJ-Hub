"""Automatic build-number reservations, shared by preview and production.

Call reserve_number only under the release-mobile-identities concurrency lock.
These annotated tags reserve platform numbers even if a build/upload later fails.
"""
import json
import re
from datetime import datetime, timedelta, timezone
from model import require

BUILD_TIMEZONE = timezone(timedelta(hours=8))


def next_display_id(github, now=None):
    day = (now or datetime.now(timezone.utc)).astimezone(BUILD_TIMEZONE).strftime('%Y%m%d')
    prefix = 'refs/tags/build-id-' + day
    rows = github.pages('git/matching-refs/tags/build-id-' + day)
    require(all(re.fullmatch(re.escape(prefix) + r'[0-9]{4}', r['ref']) for r in rows),
            'Malformed daily build reservation; reconcile before allocating')
    number = max((int(r['ref'][-4:]) for r in rows), default=0) + 1
    require(number <= 9999, 'Daily build ID space exhausted; do not wrap or reuse IDs')
    return day + f'{number:04d}'


def reserve_display_id(github, source, environment, now=None):
    """Called under release-mobile-identities, shared by dev and production RC."""
    value = next_display_id(github, now)
    tag = 'build-id-' + value
    annotation = github.api('git/tags', method='POST', data={
        'tag': tag, 'object': source, 'type': 'commit',
        'message': json.dumps({'displayBuild': value, 'environment': environment, 'sourceSha': source})})
    github.api('git/refs', method='POST', data={'ref': 'refs/tags/' + tag, 'sha': annotation['sha']})
    return value


def reserve_number(github, source, version, number, environment, *, display_build=None, ios_build_number=None):
    tag = f'mobile-build-{number}'
    require(github.api(f'git/ref/tags/{tag}', missing=True) is None, 'Build number already consumed; re-plan')
    metadata = {'version': version, 'buildNumber': number, 'environment': environment}
    if display_build:
        metadata.update(displayBuild=display_build, iosBuildNumber=ios_build_number)
    annotation = github.api('git/tags', method='POST', data={
        'tag': tag, 'object': source, 'type': 'commit', 'message': json.dumps(metadata)})
    github.api('git/refs', method='POST', data={'ref': 'refs/tags/' + tag, 'sha': annotation['sha']})


def verify_production_number(github, manifest):
    if manifest['product'] != 'mobile':
        return
    marker = github.api(f"git/ref/tags/mobile-build-{manifest['buildNumber']}", missing=True)
    if marker is None:
        # Existing immutable formal releases predate number reservations. This
        # exception cannot allocate a fresh formal version from a preview number.
        formal = github.api(f"git/ref/tags/{manifest['tag']}", missing=True)
        require(formal and formal['object']['type'] == 'tag', 'New mobile releases require a production build reservation')
        return
    require(marker['object']['type'] == 'tag', 'Build reservation must be annotated')
    annotation = github.api(f"git/tags/{marker['object']['sha']}")
    metadata = json.loads(annotation['message'])
    require(annotation['object']['type'] == 'commit' and annotation['object']['sha'] == manifest['sourceSha']
            and metadata.get('environment') == 'production' and metadata.get('version') == manifest['version']
            and metadata.get('buildNumber') == manifest['buildNumber'],
            'Preview/other-source build numbers cannot be used for formal publication or App Store promotion')
    if manifest['schemaVersion'] >= 4:
        require(all(metadata.get(k) == manifest.get(k) for k in ['displayBuild', 'iosBuildNumber']),
                'Production date/native build identity differs from its reservation')
