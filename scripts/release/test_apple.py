import copy
import unittest
from unittest.mock import patch
import apple
from model import ReleaseError


class AppleBaselineTests(unittest.TestCase):
    def fixtures(self):
        builds = {'data': [], 'included': []}
        for index, groups in [(1, [apple.GROUP_ID]), (2, [])]:
            identity = str(index)
            builds['data'].append({'id': identity, 'attributes': {'version': identity, 'uploadedDate': identity},
                'relationships': {'preReleaseVersion': {'data': {'id': 'v' + identity}},
                                  'buildBetaDetail': {'data': {'id': 'b' + identity}},
                                  'betaGroups': {'data': [{'id': group} for group in groups]}}})
            builds['included'] += [{'id': 'v' + identity, 'attributes': {'version': '1.0.' + identity}},
                                   {'id': 'b' + identity, 'attributes': {'externalBuildState': 'BETA_APPROVED'}}]
        # Pinned ASC 5 computed-output contract (not the raw API's "data" envelope):
        # https://github.com/rorkai/App-Store-Connect-CLI/blob/5.0.0/internal/cli/cmdtest/versions_list_latest_test.go
        store = {'items': [{'attributes': {'versionString': '1.0.0', 'appStoreState': 'READY_FOR_SALE'},
                           'relationships': {'build': {'data': {'id': 'original'}}}}],
                 'totalCount': 1, 'hasMore': False,
                 'included': [{'id': 'original', 'type': 'builds', 'attributes': {'version': '0'}}]}
        return builds, store

    def test_store_and_testflight_have_independent_actual_builds(self):
        with patch('apple.asc', side_effect=self.fixtures()):
            state = apple.discover()
        self.assertEqual(state['ios-testflight']['buildId'], '1')
        self.assertEqual(state['ios-app-store']['buildId'], 'original')
        # A newer valid build without the external group cannot advance the baseline;
        # both valid identities remain available for exact-build promotion.
        self.assertEqual(set(state['validBuilds']), {'1', '2'})

    def test_missing_distribution_relationship_is_unknown_not_first_release(self):
        builds, store = self.fixtures()
        del builds['data'][0]['relationships']['betaGroups']
        with patch('apple.asc', side_effect=[builds, store]), self.assertRaises(ReleaseError):
            apple.discover()

    def test_live_store_discovery_respects_asc_state_filter_families(self):
        builds, store = self.fixtures()
        def asc(*args):
            if args[:2] == ('builds', 'list'):
                return builds
            # ASC 5 rejects mixing appStoreState and appVersionState-only values
            # before sending a request. READY_FOR_SALE + --latest is its documented
            # live-version lookup, including historical versions still marked live.
            states = args[args.index('--state') + 1].split(',')
            if 'READY_FOR_SALE' in states and 'READY_FOR_DISTRIBUTION' in states:
                raise RuntimeError('ASC versions list: cannot mix state filter families')
            self.assertIn('--latest', args)
            return store
        with patch('apple.asc', side_effect=asc):
            discovered = apple.discover()
        self.assertEqual(discovered['ios-app-store']['buildId'], 'original')

    def test_confirmed_empty_latest_result_is_no_live_store_version(self):
        builds, _ = self.fixtures()
        with patch('apple.asc', side_effect=[builds, {'items': [], 'totalCount': 0, 'hasMore': False}]):
            self.assertIsNone(apple.discover()['ios-app-store'])

    def test_unknown_or_incomplete_latest_response_cannot_be_first_release(self):
        builds, store = self.fixtures()
        for result in ({}, {'items': None, 'totalCount': 0, 'hasMore': False},
                       {'items': [], 'totalCount': 1, 'hasMore': False}, store | {'hasMore': True}):
            with self.subTest(result=result), patch('apple.asc', side_effect=[builds, result]), \
                 self.assertRaisesRegex(ReleaseError, 'latest-version response'):
                apple.discover()
