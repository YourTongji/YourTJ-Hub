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
                         SOURCE_SHA='a' * 40, MOBILE_VERSION='1.0.17', MOBILE_BUILD_NUMBER='21',
                         PRERELEASE_TESTFLIGHT_GROUP_ID='qa', IOS_IPA_PATH='/tmp/app.ipa',
                         RELEASE_RESULT_PATH=str(self.result),
                         GITHUB_STEP_SUMMARY=str(Path(folder.name) / 'summary'))
        env.start()
        self.addCleanup(env.stop)
        self.calls = []
        self.find = {'id': 'build'}
        self.uploads = []
        self.notes = []
        self.stage = 'IN_BETA_TESTING'
        Path(os.environ['GITHUB_STEP_SUMMARY']).write_text('')

    def asc(self, *args):
        self.calls.append(args)
        if args[:3] == ('testflight', 'groups', 'list'):
            return {'data': [{'id': 'qa', 'attributes': {'isInternalGroup': True,
                                                       'hasAccessToAllBuilds': False}}]}
        if args[:3] == ('builds', 'uploads', 'list'):
            return {'data': list(self.uploads)}
        if args[:2] == ('builds', 'upload'):
            return {}
        if args[:2] == ('builds', 'update') or args[:2] == ('builds', 'add-groups'):
            return {}
        if args[:3] == ('builds', 'test-notes', 'list'):
            return {'data': list(self.notes)}
        if args[:3] == ('builds', 'test-notes', 'create') or args[:3] == ('builds', 'test-notes', 'update'):
            return {}
        if args[:3] == ('builds', 'build-beta-detail', 'view'):
            return {'data': {'attributes': {'internalBuildState': self.stage}}}
        self.fail(f'Unexpected command: {args[:3]}')

    def run_publisher(self, existing=True):
        find = self.find if existing else None
        with patch.object(publisher, 'asc', side_effect=self.asc), \
             patch.object(publisher, 'find_build', return_value=find), \
             patch.object(publisher, 'wait_for_build', return_value='build'), \
             patch.object(publisher, 'resource', side_effect=lambda data: data['data']), \
             patch.object(publisher.subprocess, 'run'):
            publisher.main()

    def test_existing_build_is_assigned_without_upload(self):
        self.run_publisher(existing=True)
        self.assertEqual(json.loads(self.result.read_text())['stage'], 'available_for_testing')
        self.assertFalse(any(c[:2] == ('builds', 'upload') for c in self.calls))
        self.assertEqual(sum(c[:2] == ('builds', 'add-groups') for c in self.calls), 1)

    def test_missing_build_uploads_when_no_in_flight_upload(self):
        self.run_publisher(existing=False)
        self.assertEqual(sum(c[:2] == ('builds', 'upload') for c in self.calls), 1)
        self.assertEqual(json.loads(self.result.read_text())['stage'], 'available_for_testing')

    def test_in_flight_upload_waits_without_second_upload(self):
        self.uploads = [{'attributes': {'uploadedDate': '2026-10-20T00:00:00Z'}}]
        self.run_publisher(existing=False)
        self.assertFalse(any(c[:2] == ('builds', 'upload') for c in self.calls))
        self.assertTrue(any(c[:3] == ('builds', 'uploads', 'list') for c in self.calls))

    def test_uncertain_upload_without_date_is_rejected(self):
        self.uploads = [{'attributes': {}}]
        with self.assertRaises(ValueError):
            self.run_publisher(existing=False)
        self.assertFalse(any(c[:2] == ('builds', 'upload') for c in self.calls))

    def test_other_automatic_group_blocks_preview(self):
        def asc(*args):
            self.calls.append(args)
            if args[:3] == ('testflight', 'groups', 'list'):
                return {'data': [
                    {'id': 'qa', 'attributes': {'isInternalGroup': True, 'hasAccessToAllBuilds': False}},
                    {'id': 'other', 'attributes': {'isInternalGroup': True, 'hasAccessToAllBuilds': True}},
                ]}
            self.fail(f'Unexpected command: {args[:3]}')
        with patch.object(publisher, 'asc', side_effect=asc), \
             patch.object(publisher, 'find_build', return_value={'id': 'build'}), \
             patch.object(publisher.subprocess, 'run'), \
             self.assertRaises(ValueError):
            publisher.main()

    def test_not_installable_never_counts_as_success(self):
        self.stage = 'PROCESSING'
        with self.assertRaises(ValueError):
            self.run_publisher()
        self.assertNotEqual(json.loads(self.result.read_text())['stage'], 'available_for_testing')


if __name__ == '__main__':
    unittest.main()
