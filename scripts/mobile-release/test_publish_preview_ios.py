import json
import os
from pathlib import Path
from tempfile import TemporaryDirectory
import unittest
from unittest.mock import patch

import publish_preview_ios as publisher


class PreviewPublisherTests(unittest.TestCase):
    def setUp(self):
        folder = TemporaryDirectory()
        self.addCleanup(folder.cleanup)
        self.result = Path(folder.name) / 'result.json'
        env = patch.dict(os.environ, MOBILE_ENVIRONMENT='dev', MOBILE_PREVIEW='true',
                         SOURCE_SHA='a'*40, MOBILE_VERSION='1.0.17', MOBILE_BUILD_NUMBER='21',
                         PRERELEASE_TESTFLIGHT_GROUP_ID='qa', IOS_EXISTING_BUILD_ONLY='true',
                         IOS_EXPECTED_BUILD_ID='build', RELEASE_RESULT_PATH=str(self.result),
                         GITHUB_STEP_SUMMARY=str(Path(folder.name) / 'summary'))
        env.start()
        self.addCleanup(env.stop)
        self.calls = []
        self.automatic = True
        self.members = [{'id': 'qa', 'type': 'internal'}]
        self.complete = True

    def asc(self, *args):
        self.calls.append(args)
        if args[:3] == ('testflight', 'groups', 'list'):
            return {'data': [{'id': 'qa', 'attributes': {'isInternalGroup': True,
                                                       'hasAccessToAllBuilds': self.automatic}}]}
        if args[:3] == ('builds', 'groups', 'list'):
            return {'appId': publisher.APP_ID, 'buildId': 'build', 'complete': self.complete,
                    'groups': list(self.members)}
        if args[:2] == ('builds', 'add-groups'):
            if self.automatic:
                raise RuntimeError('Apple 422: automatic internal groups cannot be assigned manually')
            self.members = [{'id': 'qa', 'type': 'internal'}]
            return {}
        if args[:3] == ('builds', 'test-notes', 'list'):
            return {'data': []}
        if args[:3] == ('builds', 'build-beta-detail', 'view'):
            return {'data': {'attributes': {'internalBuildState': 'IN_BETA_TESTING'}}}
        if args[:2] == ('builds', 'update') or args[:3] == ('builds', 'test-notes', 'create'):
            return {}
        self.fail(f'Unexpected command: {args[:3]}')

    def run_publisher(self, build='build'):
        with patch.object(publisher, 'asc', side_effect=self.asc), \
             patch.object(publisher, 'find_build', return_value={'id': build} if build else None), \
             patch.object(publisher, 'wait_for_build', return_value=build), \
             patch.object(publisher.subprocess, 'run'):
            publisher.main()

    def test_auto_access_group_does_not_repeat_manual_assignment(self):
        self.run_publisher()
        self.assertEqual(json.loads(self.result.read_text())['stage'], 'available_for_testing')
        self.assertFalse(any(c[:2] == ('builds', 'add-groups') for c in self.calls))

    def test_existing_manual_membership_is_idempotent(self):
        self.automatic = False
        self.run_publisher()
        self.assertFalse(any(c[:2] == ('builds', 'add-groups') for c in self.calls))

    def test_unassigned_manual_group_is_added_and_verified(self):
        self.automatic = False
        self.members = []
        self.run_publisher()
        self.assertEqual(sum(c[:2] == ('builds', 'add-groups') for c in self.calls), 1)

    def test_recovery_never_uploads_missing_or_different_build(self):
        for build in [None, 'other']:
            with self.subTest(build=build), self.assertRaises(ValueError):
                self.run_publisher(build)
            self.assertFalse(any(c[:2] in [('builds', 'upload'), ('builds', 'update')] for c in self.calls))

    def test_incomplete_or_extra_audience_never_counts_as_success(self):
        for complete, members in [(False, self.members),
                                  (True, self.members + [{'id': 'public', 'type': 'external'}]),
                                  (True, [])]:
            self.complete, self.members = complete, members
            with self.subTest(complete=complete, members=members), self.assertRaises(ValueError):
                self.run_publisher()
            self.assertNotEqual(json.loads(self.result.read_text())['stage'], 'available_for_testing')
