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
        store = {'data': [{'attributes': {'versionString': '1.0.0', 'appStoreState': 'READY_FOR_SALE'},
                           'relationships': {'build': {'data': {'id': 'original'}}}}],
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
