import unittest
from select_inputs import select, verify_results


class SelectTests(unittest.TestCase):
    def test_docs_still_get_governance_and_current_vulnerability_scan(self):
        plan = select(["docs/operations/releases.md"])
        self.assertEqual({k for k, v in plan.items() if v["run"]}, {"docs", "govulncheck"})

    def test_mobile_platforms_and_unknown_executables(self):
        for path in ["apps/mobile/packages/forum_app/android/app/build.gradle.kts", "apps/mobile/packages/forum_app/ios/Runner/A.swift"]:
            plan = select([path])
            self.assertTrue(plan["mobile"]["run"])
            self.assertTrue(plan["android" if "/android/" in path else "ios"]["run"])
            self.assertFalse(plan["ios" if "/android/" in path else "android"]["run"])
            self.assertFalse(plan["deploy"]["run"])
        self.assertTrue(all(v["run"] for v in select(["unknown/execute.sh"]).values()))

    def test_dev_deploy_workflow_change_rehearses_the_deployment(self):
        plan = select([".github/workflows/deploy-dev.yml"])
        for domain in ("automation", "backend", "frontend", "deploy"):
            self.assertTrue(plan[domain]["run"], domain)
        self.assertFalse(plan["ios"]["run"] or plan["android"]["run"])

    def test_android_entrypoint_checker_runs_against_a_real_native_build(self):
        for path in ('scripts/ci/check_android_entrypoint.py', 'scripts/ci/test_android_entrypoint.py'):
            with self.subTest(path=path):
                plan = select([path])
                self.assertTrue(plan['android']['run'] and plan['automation']['run'])
                self.assertFalse(plan['ios']['run'] or plan['deploy']['run'])

    def test_missing_skipped_cancelled_fail_closed(self):
        plan = select(["apps/mobile/packages/core/lib/client.dart"])
        good = {k: {"result": "success" if v["run"] else "skipped"} for k, v in plan.items() if k != "deploy"}
        verify_results(plan, good)
        for result in ["skipped", "cancelled", "failure", ""]:
            with self.assertRaises(ValueError):
                verify_results(plan, good | {"mobile": {"result": result}})
        del good["mobile"]
        with self.assertRaises(ValueError):
            verify_results(plan, good)

    def test_apk_budget_changes_select_android_without_ios_or_deploy(self):
        for name in ('check_android_apk_size.py', 'test_android_apk_size.py', 'android_apk_size_baseline.json'):
            with self.subTest(name=name):
                plan = select(['scripts/ci/' + name])
                self.assertTrue(plan['android']['run'] and plan['automation']['run'])
                self.assertFalse(plan['ios']['run'] or plan['deploy']['run'])

    def test_service_changes_include_postgres_and_race(self):
        plan = select(["apps/gooseforum/app/service/test.go"])
        self.assertTrue(plan["backend"]["run"] and plan["postgres"]["run"] and plan["deploy"]["run"])

    def test_metadata_does_not_deploy_or_build_native(self):
        plan = select(["releases/requests/mobile-1.0.15-15/ios.zh-Hans.txt"])
        self.assertTrue(plan["release"]["run"])
        self.assertFalse(plan["deploy"]["run"] or plan["mobile"]["run"])


class ComparisonFallbackTests(unittest.TestCase):
    def test_manual_dispatch_explicitly_validates_all_domains_without_a_diff(self):
        import contextlib
        import io
        import json
        import os
        from pathlib import Path
        import tempfile
        from unittest.mock import patch
        import select_inputs
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'plan.json'
            with patch.dict(os.environ, GITHUB_EVENT_NAME='workflow_dispatch', GITHUB_EVENT_PATH='', GITHUB_OUTPUT=''), \
                 patch('sys.argv', ['select_inputs.py', '--base', 'a' * 40, '--output', str(output)]), \
                 patch('select_inputs.changed_paths', side_effect=AssertionError('manual runs must not narrow scope')), \
                 contextlib.redirect_stdout(io.StringIO()):
                select_inputs.main()
            plan = json.loads(output.read_text())
            self.assertEqual(set(plan), set(select_inputs.DOMAINS))
            self.assertTrue(all(v['run'] and v['reason'] == 'Explicit manual full validation' for v in plan.values()))

    def test_missing_zero_and_unresolvable_bases_run_all_domains(self):
        import contextlib
        import io
        import json
        import os
        from pathlib import Path
        import subprocess
        import tempfile
        from unittest.mock import patch
        import select_inputs
        with tempfile.TemporaryDirectory() as temporary:
            event_file = Path(temporary) / 'event.json'; output = Path(temporary) / 'plan.json'
            for event, merge_error, diff_error in (({}, False, False), ({'before': '0' * 40}, False, False),
                    ({'pull_request': {'base': {'sha': 'a' * 40}}}, True, False), ({'before': 'a' * 40}, False, True)):
                event_file.write_text(json.dumps(event))
                with self.subTest(event=event), patch.dict(os.environ, GITHUB_EVENT_PATH=str(event_file), GITHUB_OUTPUT=''), \
                     patch('sys.argv', ['select_inputs.py', '--output', str(output)]), contextlib.redirect_stdout(io.StringIO()), \
                     patch('select_inputs.subprocess.check_output', side_effect=subprocess.CalledProcessError(1, 'git') if merge_error else None), \
                     patch('select_inputs.changed_paths', side_effect=ValueError('unavailable') if diff_error else AssertionError('unexpected diff')):
                    select_inputs.main()
                plan = json.loads(output.read_text())
                self.assertEqual(set(plan), set(select_inputs.DOMAINS))
                self.assertTrue(all(v['run'] and 'unavailable' in v['reason'] for v in plan.values()))


if __name__ == "__main__":
    unittest.main()
