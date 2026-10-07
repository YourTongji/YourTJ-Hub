#!/usr/bin/env python3
"""Read actual Apple baselines; no uploads, metadata edits or review submissions."""
import json
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'mobile-release'))
from publish_ios import asc, APP_ID, GROUP_ID
from model import require


def discover():
    builds_response = asc('builds', 'list', '--app', APP_ID, '--platform', 'IOS', '--processing-state', 'VALID',
                          '--include', 'preReleaseVersion,buildBetaDetail,betaGroups', '--paginate')
    included = {i['id']: i for i in builds_response.get('included', [])}
    def identity(build):
        relation = build.get('relationships', {}).get('preReleaseVersion', {}).get('data')
        require(relation and relation['id'] in included, 'Apple build is missing marketing-version relationship')
        return {'buildId': build['id'], 'buildNumber': build['attributes']['version'],
                'version': included[relation['id']]['attributes']['version']}
    distributed = []
    identities = {}
    for build in builds_response['data']:
        identities[build['id']] = identity(build)
        relation = build.get('relationships', {}).get('buildBetaDetail', {}).get('data')
        require(relation and relation['id'] in included, 'Apple build is missing beta distribution state')
        attrs = included[relation['id']]['attributes']
        groups = build.get('relationships', {}).get('betaGroups', {}).get('data')
        require(isinstance(groups, list), 'Apple build is missing beta-group membership')
        if (attrs.get('externalBuildState') in {'IN_BETA_TESTING', 'BETA_APPROVED'}
                and any(group['id'] == GROUP_ID for group in groups)):
            distributed.append((build['attributes']['uploadedDate'], identity(build) | {'state': attrs['externalBuildState']}))
    testflight = max(distributed, key=lambda row: row[0])[1] if distributed else None
    # ASC 5's documented live-version query uses the appStoreState family.
    # READY_FOR_DISTRIBUTION belongs to appVersionState and cannot be mixed here.
    versions = asc('versions', 'list', '--app', APP_ID, '--platform', 'IOS', '--state',
                   'READY_FOR_SALE', '--latest', '--include', 'build')
    # --latest emits computed CLI output rather than a JSON:API data envelope:
    # https://github.com/rorkai/App-Store-Connect-CLI/blob/5.0.0/internal/asc/output_versions.go#L79
    require(isinstance(versions, dict) and isinstance(versions.get('items'), list)
            and versions.get('hasMore') is False and versions.get('totalCount') == len(versions['items']),
            'Incomplete or invalid ASC latest-version response')
    items = versions['items']
    require(len(items) <= 1, 'Ambiguous live App Store baseline')
    store = None
    if items:
        version = items[0]
        build_id = version['relationships']['build']['data']['id']
        builds = {i['id']: i for i in versions.get('included', []) if i['type'] == 'builds'}
        require(build_id in builds, 'Live App Store version is missing its exact build')
        store = {'buildId': build_id, 'buildNumber': builds[build_id]['attributes']['version'],
                 'version': version['attributes']['versionString'], 'state': version['attributes']['appStoreState']}
    return {'ios-testflight': testflight, 'ios-app-store': store, 'validBuilds': identities}


if __name__ == '__main__':
    Path(sys.argv[1]).write_text(json.dumps(discover(), indent=2) + '\n', encoding='utf-8')
