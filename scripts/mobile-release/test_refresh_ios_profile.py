import base64
import datetime
import unittest
from unittest.mock import patch

import refresh_ios_profile as maintenance
import test_build_ios


class ProfileRefreshTest(unittest.TestCase):
    def setUp(self):
        fixture = test_build_ios.DistributionProfileTest()
        fixture.setUp()
        self.old = fixture.profile
        self.old['ExpirationDate'] = datetime.datetime.now() + datetime.timedelta(days=30)
        self.old['DeveloperCertificates'] = [b'existing distribution certificate']
        self.old['Entitlements'].pop('com.apple.developer.associated-domains')
        self.profile = {'id': 'old-profile', 'attributes': {'uuid': self.old['UUID']}}
        self.profiles = [self.profile]
        self.capabilities = []
        self.bundle = {'id': 'runner', 'attributes': {'identifier': maintenance.BUNDLE_ID, 'seedId': maintenance.TEAM}}
        self.calls = []

    def asc(self, *args):
        self.calls.append(args)
        if args[:2] == ('bundle-ids', 'list'): return {'data': [self.bundle]}
        if args[:2] == ('profiles', 'list'): return {'data': self.profiles}
        if args[:4] == ('profiles', 'view', '--id', 'old-profile'):
            return {'data': {'relationships': {'bundleId': {'data': {'id': 'runner'}},
                    'certificates': {'data': [{'id': 'existing-cert'}]}}},
                    'included': [{'id': 'existing-cert', 'type': 'certificates', 'attributes': {
                        'certificateContent': base64.b64encode(b'existing distribution certificate').decode()}}]}
        if args[:3] == ('bundle-ids', 'capabilities', 'list'): return {'data': self.capabilities}
        if args[:3] == ('bundle-ids', 'capabilities', 'add'): return {}
        if args[:2] == ('profiles', 'create') or args[:4] == ('profiles', 'view', '--id', 'new-profile'):
            return {'data': {'attributes': {'profileContent': base64.b64encode(b'new profile').decode()}}}
        self.fail('Unexpected Apple operation: ' + str(args))

    def test_only_enables_associated_domains_and_reuses_original_certificate(self):
        with patch.object(maintenance, 'asc', side_effect=self.asc):
            self.assertEqual(maintenance.refresh(self.old, 'new profile'), b'new profile')
        mutations = [c for c in self.calls if c[:2] == ('profiles', 'create') or c[:3] == ('bundle-ids', 'capabilities', 'add')]
        self.assertEqual(mutations, [
            ('bundle-ids', 'capabilities', 'add', '--bundle', 'runner', '--capability', 'ASSOCIATED_DOMAINS'),
            ('profiles', 'create', '--name', 'new profile', '--profile-type', 'IOS_APP_STORE',
             '--bundle', 'runner', '--certificate', 'existing-cert')])

    def test_bundle_selection_handles_canonical_case_and_unrelated_results(self):
        original = self.asc
        def listing(*args):
            if args[:2] == ('bundle-ids', 'list'):
                return {'data': [{'id': 'widget', 'attributes': {
                    'identifier': maintenance.BUNDLE_ID + '.ScheduleWidgets', 'seedId': maintenance.TEAM}},
                    {'id': 'runner', 'attributes': {
                    'identifier': maintenance.BUNDLE_ID.lower(), 'seedId': maintenance.TEAM}}]}
            return original(*args)
        with patch.object(maintenance, 'asc', side_effect=listing):
            self.assertEqual(maintenance.refresh(self.old, 'new profile'), b'new profile')
        self.assertTrue(any(c[:2] == ('profiles', 'create') for c in self.calls))

    def test_rerun_reuses_existing_capability_and_named_profile(self):
        self.capabilities = [{'attributes': {'capabilityType': 'ASSOCIATED_DOMAINS'}}]
        self.profiles.append({'id': 'new-profile', 'attributes': {'name': 'new profile'}})
        with patch.object(maintenance, 'asc', side_effect=self.asc):
            maintenance.refresh(self.old, 'new profile')
        self.assertFalse(any('add' in c or 'create' in c for c in self.calls))

    def test_unknown_certificate_or_bundle_stops_before_mutation(self):
        for field in ('certificate', 'bundle', 'team'):
            with self.subTest(field=field):
                self.setUp()
                if field == 'certificate': self.old['DeveloperCertificates'] = [b'other certificate']
                if field == 'bundle': self.bundle['attributes']['identifier'] = 'other.app'
                if field == 'team': self.bundle['attributes']['seedId'] = 'OTHERTEAM1'
                with patch.object(maintenance, 'asc', side_effect=self.asc), self.assertRaises(ValueError):
                    maintenance.refresh(self.old, 'new profile')
                self.assertFalse(any('add' in c or 'create' in c for c in self.calls))
