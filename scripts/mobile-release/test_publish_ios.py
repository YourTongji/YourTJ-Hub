from pathlib import Path
import os
import unittest
from unittest.mock import patch
import publish_ios as publisher


class IosResumeTest(unittest.TestCase):
    def setUp(self):
        self.env = patch.dict(os.environ, MOBILE_VERSION='1.0.1', MOBILE_BUILD_NUMBER='2', IOS_IPA_PATH='/unused.ipa', IOS_PUBLISH_TARGET='both', IOS_EXISTING_BUILD_ONLY='false')
        self.env.start()
        os.environ.pop('IOS_STORE_PATH', None)
        os.environ.pop('GITHUB_STEP_SUMMARY', None)
        self.addCleanup(self.env.stop)

    def test_already_submitted_build_does_not_upload_or_resubmit(self):
        calls = []
        def asc(*args, **kwargs):
            calls.append(args)
            if args[:2] == ('builds', 'list'):
                return {'data': [{'id': 'build', 'attributes': {'processingState': 'VALID'}}]}
            if args[:3] == ('builds', 'beta-app-review-submission', 'view'):
                return {'data': {'attributes': {'betaReviewState': 'WAITING_FOR_REVIEW'}}}
            if args[:2] == ('versions', 'list'):
                return {'data': [{'id': 'version', 'attributes': {'versionString': '1.0.1', 'appStoreState': 'WAITING_FOR_REVIEW'}, 'relationships': {'build': {'data': {'id': 'build'}}}}]}
            if args[:2] == ('builds', 'update'): return {}
            self.fail(f'Unexpected mutation: {args[:2]}')
        with patch.object(publisher, 'asc', side_effect=asc):
            publisher.main()
        self.assertFalse(any('upload' in call or 'submit' in call for call in calls))

    def test_testflight_recovery_preserves_app_store_review_queue(self):
        calls = []
        def asc(*args, **kwargs):
            calls.append(args)
            if args[:2] == ('builds', 'list'):
                return {'data': [{'id': 'build', 'attributes': {'processingState': 'VALID'}}]}
            if args[:3] == ('builds', 'beta-app-review-submission', 'view'):
                return {'data': {'attributes': {'betaReviewState': 'APPROVED'}}}
            if args[:2] == ('builds', 'update'): return {}
            self.fail(f'Unexpected App Store or upload mutation: {args[:2]}')
        with patch.dict(os.environ, IOS_PUBLISH_TARGET='testflight', IOS_EXISTING_BUILD_ONLY='true'), patch.object(publisher, 'asc', side_effect=asc):
            publisher.main()
        self.assertFalse(any(call[0] == 'versions' for call in calls))

    def test_recovery_missing_build_does_not_upload(self):
        with patch.dict(os.environ, IOS_EXISTING_BUILD_ONLY='true'), patch.object(publisher, 'find_build', return_value=None), patch.object(publisher, 'wait_for_build', side_effect=AssertionError('Unexpected upload polling')), patch.object(publisher, 'asc') as asc:
            with self.assertRaisesRegex(ValueError, 'existing.*build'):
                publisher.main()
            asc.assert_not_called()

    def test_pending_other_app_store_version_reports_actionable_blocker(self):
        calls = []
        def asc(*args, **kwargs):
            calls.append(args)
            if args[:2] == ('builds', 'list'):
                return {'data': [{'id': 'build', 'attributes': {'processingState': 'VALID'}}]}
            if args[:3] == ('builds', 'beta-app-review-submission', 'view'):
                return {'data': {'attributes': {'betaReviewState': 'APPROVED'}}}
            if args[:2] == ('versions', 'list'):
                self.assertIn('--paginate', args)
                return {'data': [{'id': 'old-version', 'attributes': {'versionString': '1.0.0', 'appStoreState': 'WAITING_FOR_REVIEW'}}]}
            if args[:2] == ('builds', 'update'): return {}
            self.fail(f'Unexpected mutation while old version is pending: {args[:2]}')
        with patch.object(publisher, 'asc', side_effect=asc), self.assertRaisesRegex(ValueError, '1.0.0.*WAITING_FOR_REVIEW'):
            publisher.main()

    def test_invalid_publish_target_fails_before_network_calls(self):
        with patch.dict(os.environ, IOS_PUBLISH_TARGET='unknown'), patch.object(publisher, 'wait_for_build', side_effect=AssertionError('Unexpected upload polling')), patch.object(publisher, 'asc') as asc:
            with self.assertRaisesRegex(ValueError, 'publish target'):
                publisher.main()
            asc.assert_not_called()

    def test_incomplete_upload_reservation_requires_recovery_not_duplicate_upload(self):
        def asc(*args, **kwargs):
            if args[:2] == ('builds', 'list'): return {'data': []}
            if args[:3] == ('builds', 'uploads', 'list'):
                return {'data': [{'attributes': {'cfBundleShortVersionString': '1.0.1', 'cfBundleVersion': '2'}}]}
            self.fail('Unexpected upload')
        with patch.object(publisher, 'asc', side_effect=asc), self.assertRaisesRegex(ValueError, 'incomplete upload'):
            publisher.main()

    def test_invalid_processing_fails_without_waiting(self):
        with patch.object(publisher, 'find_build', return_value={'attributes': {'processingState': 'INVALID'}}), self.assertRaisesRegex(ValueError, 'rejected'):
            publisher.wait_for_build('1.0.1', '2')

    def test_new_version_creates_missing_review_details_and_submits(self):
        import json
        from tempfile import TemporaryDirectory
        calls = []
        def asc(*args, **kwargs):
            calls.append(args)
            prefix = args[:2]
            if prefix == ('builds', 'list'):
                return {'data': [{'id': 'build', 'attributes': {'processingState': 'VALID'}}]}
            if args[:3] == ('builds', 'beta-app-review-submission', 'view'):
                return {'data': None}
            if args[:3] == ('testflight', 'review', 'view'): return {'data': [{'id': 'beta-details'}]}
            if prefix == ('versions', 'list'): return {'data': []}
            if prefix == ('versions', 'create'): return {'data': {'id': 'version'}}
            if prefix == ('localizations', 'list'): return {'data': []}
            if prefix == ('localizations', 'create'): return {'data': {'id': 'locale'}}
            if prefix == ('review', 'details-for-version'):
                self.assertTrue(kwargs.get('allow_missing'))
                return {'data': None}
            if prefix == ('review', 'submit'): return {'submissionId': 'submission'}
            return {}
        contact = {key: 'example' for key in publisher.REVIEW_FIELDS}
        with TemporaryDirectory() as temp:
            store = Path(temp) / 'apps/mobile/store/zh-Hans'
            store.mkdir(parents=True)
            (store / 'metadata.json').write_text(json.dumps({'description': 'Community', 'whatsNew': 'Update'}))
            with patch.object(publisher, 'ROOT', Path(temp)), patch.object(publisher, 'asc', side_effect=asc), patch.dict(os.environ, IOS_REVIEW_JSON=json.dumps(contact)):
                publisher.main()
        self.assertIn(('review', 'details-create'), [call[:2] for call in calls])
        self.assertIn(('publish', 'testflight'), [call[:2] for call in calls])
        self.assertIn(('review', 'submit'), [call[:2] for call in calls])

    def test_cli_failures_do_not_echo_private_review_data(self):
        from subprocess import CompletedProcess
        with patch.object(publisher.subprocess, 'run', return_value=CompletedProcess([], 1, '', 'secret-demo-password')):
            with self.assertRaises(RuntimeError) as error:
                publisher.asc('review', 'details-update', '--demo-account-password', 'secret-demo-password')
        self.assertNotIn('secret-demo-password', str(error.exception))

    def test_absent_beta_review_cli_message_is_missing_not_failure(self):
        from subprocess import CompletedProcess
        result = CompletedProcess([], 4, '', 'Error: builds beta-app-review-submission view: no beta app review submission found for build "new-build"')
        with patch.object(publisher.subprocess, 'run', return_value=result):
            self.assertEqual(publisher.asc('builds', 'beta-app-review-submission', 'view', '--build-id', 'new-build', allow_missing=True), {'data': None})

    def test_beta_lookup_other_failures_are_not_treated_as_missing(self):
        from subprocess import CompletedProcess
        for code, message in [(4, 'Permission denied'), (1, 'no beta app review submission found for build "x"')]:
            with patch.object(publisher.subprocess, 'run', return_value=CompletedProcess([], code, '', message)), self.assertRaises(RuntimeError):
                publisher.asc('builds', 'beta-app-review-submission', 'view', '--build-id', 'x', allow_missing=True)

    def test_single_resource_handles_cli_collections_and_rejects_ambiguity(self):
        for value in ({'data': []}, {'data': None}):
            self.assertEqual(publisher.resource(value), {})
        self.assertEqual(publisher.resource({'data': [{'id': 'one'}]}), {'id': 'one'})
        with self.assertRaisesRegex(ValueError, 'Ambiguous'):
            publisher.resource({'data': [{'id': 'one'}, {'id': 'two'}]})
