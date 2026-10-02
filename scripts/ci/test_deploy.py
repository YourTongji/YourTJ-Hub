import unittest
from deploy_gate import admitted


class DeployGateTest(unittest.TestCase):
    def test_only_successful_current_own_dev_push_is_admitted(self):
        run = {'event': 'push', 'conclusion': 'success', 'head_branch': 'dev', 'head_sha': 'a' * 40,
               'head_repository': {'full_name': 'YourTongji/YourTJ-Hub'}, 'name': 'CI / Verify', 'path': '.github/workflows/ci.yml'}
        self.assertTrue(admitted(run, 'YourTongji/YourTJ-Hub', 'a' * 40))
        for mutation in [{'event': 'pull_request'}, {'conclusion': 'failure'}, {'head_branch': 'main'},
                         {'head_sha': 'b' * 40}, {'head_repository': {'full_name': 'attacker/fork'}}, {'path': 'another.yml'}]:
            self.assertFalse(admitted(run | mutation, 'YourTongji/YourTJ-Hub', 'a' * 40))
