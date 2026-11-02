"""Automatic dev distribution admits merged dev sources, never arbitrary callers."""
import unittest
from datetime import datetime, timezone
from unittest.mock import Mock

from dev_delivery import admitted_request, project_status
from build_identity import next_display_id, reserve_display_id
from model import ReleaseError, REPOSITORY


class DevDeliveryTests(unittest.TestCase):
    def test_request_must_be_a_same_repo_dev_push_at_the_selected_sha(self):
        source = 'a' * 40
        row = {'path': '.github/workflows/dev-request.yml', 'event': 'push', 'head_branch': 'dev',
               'head_sha': source, 'head_repository': {'full_name': REPOSITORY}}
        self.assertTrue(admitted_request(row, REPOSITORY, source))
        for change in [{'event': 'pull_request'}, {'head_branch': 'main'}, {'head_sha': 'b' * 40},
                       {'path': '.github/workflows/ci.yml'}, {'head_repository': {'full_name': 'fork/repo'}}]:
            self.assertFalse(admitted_request(row | change, REPOSITORY, source))

    def test_status_keeps_platform_sources_separate_and_has_no_freeze_expiry(self):
        current = 'b' * 40
        status = project_status(current, {'android': {'source': 'a' * 40, 'run': '1'},
                                          'ios': {'source': current, 'run': '2'}}, 100)
        self.assertEqual(status['source'], current)
        self.assertEqual(status['platforms']['android']['source'], 'a' * 40)
        self.assertNotIn('until', status)
        self.assertNotIn('owner', status)

    def test_display_ids_reset_at_utc_plus_eight_midnight(self):
        remote = Mock()
        remote.pages.return_value = [{'ref': 'refs/tags/build-id-202611020002'}]
        before = datetime(2026, 11, 2, 15, 59, tzinfo=timezone.utc)
        self.assertEqual(next_display_id(remote, before), '202611020003')
        after = datetime(2026, 11, 2, 16, 0, tzinfo=timezone.utc)
        remote.pages.return_value = []
        self.assertEqual(next_display_id(remote, after), '202611030001')
        self.assertIn('20261103', remote.pages.call_args.args[0])
        remote.pages.return_value = [{'ref': 'refs/tags/build-id-202611039999'}]
        with self.assertRaises(ReleaseError):
            next_display_id(remote, after)

    def test_reservation_binds_the_display_id_to_source_and_never_uses_it_as_native_code(self):
        remote = Mock()
        remote.pages.return_value = []
        remote.api.return_value = {'sha': 'tag-object'}
        value = reserve_display_id(remote, 'a' * 40, 'dev', datetime(2026, 11, 3, tzinfo=timezone.utc))
        self.assertEqual(value, '202611030001')
        annotation = remote.api.call_args_list[0].kwargs['data']
        self.assertEqual(annotation['object'], 'a' * 40)
        self.assertEqual(remote.api.call_args.kwargs['data']['ref'], 'refs/tags/build-id-' + value)


if __name__ == '__main__': unittest.main()
